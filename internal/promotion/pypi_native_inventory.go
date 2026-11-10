package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

const nativePyPIRecordLimit int64 = 1 << 20
const nativePyPIRecordsLimit = 4 << 20
const nativePyPIFileLimit int64 = 256 << 20
const nativePyPIBytecodeLimit int64 = 4 << 20
const nativePyPITreeLimit int64 = 1 << 30
const nativePyPIEntryLimit = 10000

// A claim is frozen input only. It cannot be inserted into current ownership
// until the independent old wheel/installed-role proof has succeeded.
type pypiNativeFile struct {
	destination pypiDestination
	original    pypiFileSnapshot
	unhashed    bool
}

type pypiNativeDistribution struct {
	project, version, directory string
	record                      []byte
	files                       map[string]pypiNativeFile
}

type pypiNativeInventory struct {
	projects map[string]*pypiNativeDistribution
}

// A distribution is managed only when its exact METADATA has one matching
// ledger owner. A payload file carrying the same name is not sufficient.
func managedPyPIInventory(transaction *pypiVenvTransaction, input artifactpypi.UpgradeInput) (map[string]string, error) {
	return managedPyPIInventoryFromMetadata(transaction, input.MetadataFiles())
}

func managedPyPIInventoryFromMetadata(transaction *pypiVenvTransaction, metadataFiles []artifactpypi.InstalledMetadataFile) (map[string]string, error) {
	if _, err := pypiOriginalObjects(transaction); err != nil {
		return nil, err
	}
	owned := map[string]string{}
	for _, destination := range transaction.current.Files {
		for _, owner := range destination.owners() {
			if version := owned[owner.Distribution]; version != "" && version != owner.Version {
				return nil, errors.New("installed ownership has conflicting distribution versions")
			}
			owned[owner.Distribution] = owner.Version
		}
	}
	managed := map[string]string{}
	seenMetadata := map[string]bool{}
	for _, metadata := range metadataFiles {
		name, version, ok := installedDistInfo(metadata.Directory)
		if !ok {
			return nil, errors.New("installed ownership metadata identity is invalid")
		}
		// These claims must be unambiguous independently of UpgradeInput.
		bodyName, bodyVersion, err := parseInstalledMetadata(metadata.Body)
		if err != nil || bodyName != name || bodyVersion != version || seenMetadata[name] {
			return nil, errors.New("installed METADATA identity is ambiguous or differs from its directory")
		}
		seenMetadata[name] = true
		if destination, present := transaction.current.Files["site/"+metadata.Directory+"/METADATA"]; present {
			owners := destination.owners()
			if len(owners) != 1 || owners[0].Distribution != name || owners[0].Version != version || owned[name] != version {
				return nil, errors.New("installed METADATA has ambiguous HAA ownership")
			}
			managed[name] = version
		} else if owned[name] != "" {
			return nil, errors.New("installed distribution has partial HAA ownership without exact METADATA")
		}
	}
	if len(managed) != len(owned) {
		return nil, errors.New("HAA ownership lacks complete installed distribution metadata")
	}
	return managed, nil
}

func pypiOriginalObjects(transaction *pypiVenvTransaction) (map[[2]uint64]string, error) {
	objects := map[[2]uint64]string{}
	for key, destination := range transaction.current.Files {
		before := transaction.before[destination.Final]
		if before == nil {
			return nil, errors.New("owned file has no original identity snapshot")
		}
		identity := [2]uint64{before.Device, before.Inode}
		if _, present := objects[identity]; present {
			return nil, errors.New("owned installation has aliased original file paths")
		}
		objects[identity] = key
	}
	return objects, nil
}

func capturePyPINativeInventory(ctx context.Context, transaction *pypiVenvTransaction, input artifactpypi.UpgradeInput) (pypiNativeInventory, error) {
	return capturePyPINativeInventoryFromMetadata(ctx, transaction, input.MetadataFiles())
}

func capturePyPINativeInventoryFromMetadata(ctx context.Context, transaction *pypiVenvTransaction, metadataFiles []artifactpypi.InstalledMetadataFile) (pypiNativeInventory, error) {
	return capturePyPINativeInventoryWithToolScripts(ctx, transaction, metadataFiles, nil)
}

func capturePyPINativeInventoryWithToolScripts(ctx context.Context, transaction *pypiVenvTransaction, metadataFiles []artifactpypi.InstalledMetadataFile, scripts *uvNativeToolScriptProof) (pypiNativeInventory, error) {
	inventory := pypiNativeInventory{projects: map[string]*pypiNativeDistribution{}}
	managed, err := managedPyPIInventoryFromMetadata(transaction, metadataFiles)
	if err != nil {
		return pypiNativeInventory{}, err
	}
	objects, err := pypiOriginalObjects(transaction)
	if err != nil {
		return pypiNativeInventory{}, err
	}
	claims := map[string]bool{}
	var totalBytes int64
	recordBytes, entries := 0, 0
	for _, metadata := range metadataFiles {
		if err := ctx.Err(); err != nil {
			return pypiNativeInventory{}, err
		}
		name, version, ok := installedDistInfo(metadata.Directory)
		if !ok {
			return pypiNativeInventory{}, errors.New("native PyPI metadata identity is invalid")
		}
		key := "site/" + metadata.Directory + "/METADATA"
		if managed[name] == version {
			continue // HAA's private --target RECORD must not be reinterpreted.
		}
		directory := filepath.Join(transaction.plan.site, metadata.Directory)
		if err := transaction.freezeParents(directory); err != nil {
			return pypiNativeInventory{}, err
		}
		recordPath := filepath.Join(directory, "RECORD")
		recordSnapshot, err := snapshotBoundedPromotionFile(recordPath, nativePyPIRecordLimit)
		if err != nil {
			return pypiNativeInventory{}, err
		}
		record, err := readBoundedPromotionFile(recordPath, nativePyPIRecordLimit)
		if err != nil {
			return pypiNativeInventory{}, err
		}
		sum := sha256.Sum256(record)
		if recordSnapshot.Digest != hex.EncodeToString(sum[:]) {
			return pypiNativeInventory{}, errors.New("native RECORD changed while reading")
		}
		recordBytes += len(record)
		if recordBytes > nativePyPIRecordsLimit {
			return pypiNativeInventory{}, errors.New("native RECORD inventory exceeds control bound")
		}
		project := &pypiNativeDistribution{project: name, version: version, directory: metadata.Directory, record: record, files: map[string]pypiNativeFile{}}
		reader := csv.NewReader(bytes.NewReader(record))
		reader.FieldsPerRecord = 3
		selfKey := "site/" + metadata.Directory + "/RECORD"
		for {
			if err := ctx.Err(); err != nil {
				return pypiNativeInventory{}, err
			}
			row, err := reader.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return pypiNativeInventory{}, errors.New("native RECORD is malformed")
			}
			destination, err := nativePyPIRecordDestination(transaction.plan, row[0])
			if err != nil {
				return pypiNativeInventory{}, err
			}
			if claims[destination.key()] {
				return pypiNativeInventory{}, errors.New("native RECORD has duplicate or shared claims")
			}
			if _, owned := transaction.current.Files[destination.key()]; owned {
				return pypiNativeInventory{}, errors.New("native RECORD claims another HAA owner's path")
			}
			entries++
			if entries > nativePyPIEntryLimit {
				return pypiNativeInventory{}, errors.New("native RECORD file count exceeds bound")
			}
			limit := nativePyPIFileLimit
			if destination.key() == selfKey || destination.key() == key {
				limit = nativePyPIRecordLimit
			} else if destination.Scheme == "site" && strings.HasSuffix(destination.Relative, ".pyc") {
				limit = nativePyPIBytecodeLimit
			}
			snapshot, err := snapshotBoundedPromotionFile(destination.Final, limit)
			if err != nil {
				return pypiNativeInventory{}, err
			}
			identity := [2]uint64{snapshot.Device, snapshot.Inode}
			if _, present := objects[identity]; present {
				return pypiNativeInventory{}, errors.New("native RECORD has aliased or already-owned original files")
			}
			objects[identity] = destination.key()
			if snapshot.Size > nativePyPITreeLimit-totalBytes {
				return pypiNativeInventory{}, errors.New("native original payload exceeds bound")
			}
			totalBytes += snapshot.Size
			unhashed := row[1] == "" && row[2] == ""
			if unhashed {
				if destination.key() != selfKey && !(destination.Scheme == "site" && strings.HasSuffix(destination.Relative, ".pyc")) {
					return pypiNativeInventory{}, errors.New("native RECORD has an unsupported unhashed claim")
				}
			} else {
				if destination.key() == selfKey || !strings.HasPrefix(row[1], "sha256=") {
					return pypiNativeInventory{}, errors.New("native RECORD hash is unsupported")
				}
				digest, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(row[1], "sha256="))
				size, sizeErr := strconv.ParseInt(row[2], 10, 64)
				if err != nil || len(digest) != sha256.Size || sizeErr != nil || strconv.FormatInt(size, 10) != row[2] {
					return pypiNativeInventory{}, errors.New("native RECORD claim differs from the frozen file")
				}
				if scripts != nil && scripts.client == toolClientPipx && destination.Scheme == "scripts" && scripts.exposed[destination.Relative] {
					if err := scripts.capturePipxRecord(ctx, destination, snapshot, hex.EncodeToString(digest), size); err != nil {
						return pypiNativeInventory{}, err
					}
				} else if size != snapshot.Size || hex.EncodeToString(digest) != snapshot.Digest {
					return pypiNativeInventory{}, errors.New("native RECORD claim differs from the frozen file")
				}
			}
			if destination.key() == selfKey && snapshot.Digest != recordSnapshot.Digest {
				return pypiNativeInventory{}, errors.New("native RECORD changed during inventory")
			}
			if err := transaction.freezeParents(filepath.Dir(destination.Final)); err != nil {
				return pypiNativeInventory{}, err
			}
			destination.Distribution, destination.Version = name, version
			destination.Digest, destination.Size = snapshot.Digest, snapshot.Size
			project.files[destination.key()] = pypiNativeFile{destination, snapshot, unhashed}
			claims[destination.key()] = true
			frozen := snapshot
			transaction.before[destination.Final] = &frozen // snapshot only; not current ownership
		}
		metadataClaim, hasMetadata := project.files[key]
		metadataSum := sha256.Sum256(metadata.Body)
		if _, self := project.files[selfKey]; !self || !hasMetadata || metadataClaim.original.Digest != hex.EncodeToString(metadataSum[:]) {
			return pypiNativeInventory{}, errors.New("native RECORD does not bind its own metadata and record")
		}
		inventory.projects[name] = project
	}
	if err := ctx.Err(); err != nil {
		return pypiNativeInventory{}, err
	}
	if err := transaction.checkBefore(); err != nil {
		return pypiNativeInventory{}, err
	}
	return inventory, nil
}

// Native venv paths start at site-packages. Only exactly three leading parents
// reach the venv's bin/share/include roots. The private output mapper is separate.
func nativePyPIRecordDestination(plan pypiVenvPlan, value string) (pypiDestination, error) {
	if value == "" || len(value) > 4096 || !utf8.ValidString(value) || value != strings.TrimSpace(value) || strings.ContainsAny(value, "\\:\x00\r\n\t") || strings.HasPrefix(value, "/") {
		return pypiDestination{}, errors.New("native RECORD path is unsupported")
	}
	parts := strings.Split(value, "/")
	parents := 0
	for parents < len(parts) && parts[parents] == ".." {
		parents++
	}
	for _, part := range parts[parents:] {
		if part == "" || part == "." || part == ".." {
			return pypiDestination{}, errors.New("native RECORD path is noncanonical")
		}
	}
	destination := pypiDestination{Scheme: "site", Relative: value}
	if parents != 0 {
		if parents != 3 || len(parts) < 5 {
			return pypiDestination{}, errors.New("native RECORD parent depth is unsupported")
		}
		switch parts[3] {
		case "bin":
			destination.Scheme = "scripts"
		case "share":
			destination.Scheme = "data"
		case "include":
			destination.Scheme = "headers"
		default:
			return pypiDestination{}, errors.New("native RECORD escapes the supported venv schemes")
		}
		destination.Relative = strings.Join(parts[4:], "/")
	}
	return plan.bind(destination)
}
