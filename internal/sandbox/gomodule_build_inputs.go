package sandbox

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errGoBuildInput = errors.New("go build input is invalid or differs from its frozen identity")

const (
	goBuildCacheFiles       = 20000
	goBuildCacheBytes int64 = 512 << 20
)

type goBuildInputFile struct {
	file      *os.File
	root, run *os.Root
	before    os.FileInfo
	artifact  domain.AcquiredArtifact
	prefix    string
	limits    projectBuildInputLimits
}

type goBuildInputMember struct {
	kind   byte
	size   int64
	sha256 string
}

// goBuildInputManifest describes consumed data only. Retained project approval,
// observed completion, Evidence and Policy remain independent requirements.
type goBuildInputManifest struct {
	members  map[string]goBuildInputMember
	controls map[string][]byte
	storage  int64
	identity string
}

func goBuildSingleLink(info os.FileInfo) bool {
	v := reflect.ValueOf(info.Sys())
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return false
	}
	n := v.Elem().FieldByName("Nlink")
	return n.IsValid() && n.CanUint() && n.Uint() == 1
}

type projectBuildInputLimits struct {
	variant                string
	files                  int
	maxPath                int
	payload, file, control int64
	controls               [2]string
}

func goBuildInputLimits(prefix string) projectBuildInputLimits {
	limits := projectBuildInputLimits{maxPath: 1024, variant: map[string]string{"project": "go-source", "cache": "go-cache"}[prefix], files: artifactgo.MaxModuleFiles, payload: artifactgo.MaxModuleExpandedBytes, file: artifactgo.MaxModuleFileBytes, control: artifactgo.MaxProjectControlBytes, controls: [2]string{"go.mod", "go.sum"}}
	if prefix == "cache" {
		limits.files = goBuildCacheFiles
		limits.payload = goBuildCacheBytes
	}
	return limits
}

func openGoBuildInput(intake string, artifact domain.AcquiredArtifact, prefix string) (*goBuildInputFile, error) {
	return openProjectBuildInput(intake, artifact, prefix, goBuildInputLimits(prefix))
}

func openProjectBuildInput(intake string, artifact domain.AcquiredArtifact, prefix string, limits projectBuildInputLimits) (_ *goBuildInputFile, resultErr error) {
	variant := limits.variant
	parts := strings.Split(artifact.ContentHandle(), ":")
	limit := limits.payload + int64(2*limits.files+2)*512
	if prefix == "cache" {
		limit = limits.payload + int64(4*limits.files+2)*512
	}
	declared, ok := artifact.DeclaredIntegrity()
	if !filepath.IsAbs(intake) || filepath.Clean(intake) != intake || intake == "/" || variant == "" || limits.files <= 0 || limits.payload <= 0 || limits.file <= 0 || limits.control <= 0 || (prefix != "project" && prefix != "cache") ||
		artifact.Identity().Source().String() != "project-local" || artifact.Identity().Variant() != variant ||
		artifact.Digest().String() == "" || !ok || declared != "sha256:"+artifact.Digest().String() ||
		artifact.SizeBytes() == 0 || artifact.SizeBytes() > uint64(limit) || len(parts) != 3 || parts[0] != "intake" || parts[2] != variant {
		return nil, errGoBuildInput
	}
	run, err := domain.ParseRunID(parts[1])
	if err != nil {
		return nil, errGoBuildInput
	}
	input := &goBuildInputFile{artifact: artifact, prefix: prefix, limits: limits}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, input.close())
		}
	}()
	beforeRoot, err := os.Lstat(intake)
	if err != nil || !beforeRoot.IsDir() || beforeRoot.Mode().Perm() != 0o700 || beforeRoot.Mode()&os.ModeSymlink != 0 {
		return nil, errGoBuildInput
	}
	input.root, err = os.OpenRoot(intake)
	if err != nil {
		return nil, errGoBuildInput
	}
	openedRoot, err := input.root.Stat(".")
	if err != nil || !os.SameFile(beforeRoot, openedRoot) {
		return nil, errGoBuildInput
	}
	beforeRun, err := input.root.Lstat(run.String())
	if err != nil || !beforeRun.IsDir() || beforeRun.Mode().Perm() != 0o700 || beforeRun.Mode()&os.ModeSymlink != 0 {
		return nil, errGoBuildInput
	}
	input.run, err = input.root.OpenRoot(run.String())
	if err != nil {
		return nil, errGoBuildInput
	}
	openedRun, err := input.run.Stat(".")
	if err != nil || !os.SameFile(beforeRun, openedRun) {
		return nil, errGoBuildInput
	}
	input.before, err = input.run.Lstat(variant + ".tar")
	if err != nil || !input.before.Mode().IsRegular() || input.before.Mode().Perm() != 0o400 || !goBuildSingleLink(input.before) ||
		input.before.Size() <= 0 || uint64(input.before.Size()) != artifact.SizeBytes() {
		return nil, errGoBuildInput
	}
	input.file, err = input.run.OpenFile(variant+".tar", os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errGoBuildInput
	}
	opened, err := input.file.Stat()
	if err != nil || !os.SameFile(input.before, opened) || opened.Mode() != input.before.Mode() || !goBuildSingleLink(opened) {
		return nil, errGoBuildInput
	}
	return input, nil
}

func (i *goBuildInputFile) close() error {
	var err error
	if i.file != nil {
		err = errors.Join(err, i.file.Close())
		i.file = nil
	}
	if i.run != nil {
		err = errors.Join(err, i.run.Close())
		i.run = nil
	}
	if i.root != nil {
		err = errors.Join(err, i.root.Close())
		i.root = nil
	}
	if err != nil {
		return errors.New("go build input handle cleanup failed")
	}
	return nil
}

// scan also validates a second consumed copy when writer is nonnil. Metadata
// is reconstructed, so untrusted TAR headers never select ownership or paths.
func (i *goBuildInputFile) scan(ctx context.Context, manifest *goBuildInputManifest, writer *tar.Writer) error {
	if i == nil || i.file == nil || ctx == nil || manifest == nil {
		return errGoBuildInput
	}
	if _, err := i.file.Seek(0, io.SeekStart); err != nil {
		return errGoBuildInput
	}
	hash := sha256.New()
	bounded := &io.LimitedReader{R: i.file, N: int64(i.artifact.SizeBytes()) + 1}
	stream := io.TeeReader(bounded, hash)
	archive := tar.NewReader(stream)
	seen := map[string]bool{}
	directories := map[string]bool{".": true}
	files, entries := 0, 0
	var payload int64
	fileLimit, payloadLimit := i.limits.files, i.limits.payload
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return errGoBuildInput
		}
		entries++
		name := strings.TrimSuffix(header.Name, "/")
		if entries > 2*fileLimit || !validClosureDestination(name) || len(name) > i.limits.maxPath || strings.Contains(name, ":") ||
			seen[name] || !directories[path.Dir(name)] || header.Linkname != "" || header.Uid != 0 || header.Gid != 0 ||
			header.Uname != "" || header.Gname != "" {
			return errGoBuildInput
		}
		for key, value := range header.PAXRecords {
			if key != "path" || value != header.Name {
				return errGoBuildInput
			}
		}
		if i.prefix == "project" {
			for _, component := range strings.Split(name, "/") {
				if strings.HasPrefix(component, ".") {
					return errGoBuildInput
				}
			}
		}
		seen[name] = true
		member := goBuildInputMember{kind: header.Typeflag, size: header.Size}
		destination := i.prefix + "/" + name
		if header.Typeflag == tar.TypeDir {
			if header.Mode != 0o700 || header.Size != 0 {
				return errGoBuildInput
			}
			directories[name] = true
		} else if header.Typeflag == tar.TypeReg {
			files++
			if header.Mode != 0o400 || header.Size < 0 || header.Size > i.limits.file || files > fileLimit || header.Size > payloadLimit-payload {
				return errGoBuildInput
			}
		} else {
			return errGoBuildInput
		}
		if writer != nil {
			expected, ok := manifest.members[destination]
			if !ok || expected.kind != member.kind || expected.size != member.size {
				return errGoBuildInput
			}
			mode := int64(0o400)
			if member.kind == tar.TypeDir {
				mode = 0o500
			}
			if err := writer.WriteHeader(&tar.Header{Name: destination, Typeflag: member.kind, Mode: mode, Size: member.size, Uid: 1000, Gid: 1000}); err != nil {
				return err
			}
		}
		if member.kind == tar.TypeReg {
			fileHash := sha256.New()
			var output io.Writer = fileHash
			if writer != nil {
				output = io.MultiWriter(writer, fileHash)
			}
			control := i.prefix == "project" && (name == i.limits.controls[0] || name == i.limits.controls[1])
			var body strings.Builder
			if control {
				if member.size > i.limits.control {
					return errGoBuildInput
				}
				output = io.MultiWriter(output, &body)
			}
			if _, err := io.CopyN(output, archive, member.size); err != nil {
				return errGoBuildInput
			}
			member.sha256 = hex.EncodeToString(fileHash.Sum(nil))
			if writer != nil {
				if member.sha256 != manifest.members[destination].sha256 {
					return errGoBuildInput
				}
			} else if control {
				manifest.controls[name] = []byte(body.String())
			}
			payload += member.size
		}
		if writer == nil {
			if _, exists := manifest.members[destination]; exists {
				return errGoBuildInput
			}
			manifest.members[destination] = member
			manifest.storage += 4096 // bounded entry metadata/storage reserve
			if member.kind == tar.TypeReg {
				manifest.storage += (member.size + 4095) / 4096 * 4096
			}
			if manifest.storage > goBuildInputCapacity {
				return errGoBuildInput
			}
		}
	}
	// Hash all framing bytes as well as file content, including the bounded
	// trailer. Nothing beyond the exact acquired bytes may be consumed.
	if _, err := io.Copy(io.Discard, stream); err != nil || bounded.N != 1 || hex.EncodeToString(hash.Sum(nil)) != i.artifact.Digest().String() {
		return errGoBuildInput
	}
	after, err := i.file.Stat()
	if err != nil || !os.SameFile(i.before, after) || after.Mode() != i.before.Mode() || after.Size() != i.before.Size() || !after.ModTime().Equal(i.before.ModTime()) || !goBuildSingleLink(after) {
		return errGoBuildInput
	}
	return nil
}

func goBuildManifestIdentity(source, cache domain.AcquiredArtifact) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("HAA-GO-BUILD-INPUT-1\x00%s\x00%s", source.Digest().String(), cache.Digest().String())))
	return hex.EncodeToString(hash[:])
}

func validateGoBuildInputBinding(snapshot domain.ProjectDependencySnapshot, source, cache domain.AcquiredArtifact, manifest goBuildInputManifest) error {
	if !snapshot.Valid() || snapshot.Source() != artifactgo.Source() || len(snapshot.ControlDigests()) != 2 || len(manifest.controls) != 2 {
		return errGoBuildInput
	}
	project := sha256.Sum256([]byte(snapshot.Context().Target().String()))
	key := hex.EncodeToString(project[:])
	if source.Identity().Name() != "project-"+key || source.Identity().Version() != "snapshot" ||
		cache.Identity().Name() != "cache-"+key || cache.Identity().Version() != snapshot.GraphDigest().String() {
		return errGoBuildInput
	}
	for _, control := range snapshot.ControlDigests() {
		body, ok := manifest.controls[control.Name()]
		hash := sha256.Sum256(body)
		if !ok || hex.EncodeToString(hash[:]) != control.Digest().String() {
			return errGoBuildInput
		}
	}
	return nil
}
