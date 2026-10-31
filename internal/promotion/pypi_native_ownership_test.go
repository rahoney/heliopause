package promotion

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func nativeWheelOwnershipFixture(t *testing.T, mode string) (pypiVenvPlan, *pypiVenvTransaction, *pypiNativeDistribution, artifactpypi.WheelInspection, string, string) {
	t.Helper()
	_, plan := makePypiVenvFixture(t)
	archivePath := filepath.Join(realPromotionRoot(t), "old.whl")
	writeFixtureWheel(t, archivePath)
	archiveBody, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archiveBody)
	digest := hex.EncodeToString(sum[:])
	filename := "haa_promotion_fixture-1.0.0-py3-none-any.whl"
	wheel, err := artifactpypi.InspectWheel(bytes.NewReader(archiveBody), int64(len(archiveBody)), filename, digest, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"}, artifactpypi.DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(archiveBody), int64(len(archiveBody)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, file := range archive.File {
		if strings.HasSuffix(file.Name, "/RECORD") {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
		files[file.Name] = body
	}
	dist := "haa_promotion_fixture-1.0.0.dist-info"
	files[dist+"/INSTALLER"] = []byte("pip\n")
	files[dist+"/REQUESTED"] = []byte{}
	if mode == "user-file" {
		files["user-notes.txt"] = []byte("preserve unrelated user data")
	}
	if mode == "installer" {
		files[dist+"/INSTALLER"] = []byte("evil")
	}
	if mode == "requested" {
		files[dist+"/REQUESTED"] = []byte("unproven")
	}
	for name, body := range files {
		path := filepath.Join(plan.site, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cachePath := ""
	if mode != "no-compile" && mode != "user-file" && mode != "installer" && mode != "requested" && mode != "missing-wheel" {
		cachePath = "haa_promotion_fixture/__pycache__/__init__.cpython-314.pyc"
		if mode == "cache-tag" {
			cachePath = "haa_promotion_fixture/__pycache__/__init__.cpython-313.pyc"
		}
		if mode == "cache-no-source" {
			cachePath = "unrelated/__pycache__/private.cpython-314.pyc"
		}
		source, err := os.Stat(filepath.Join(plan.site, "haa_promotion_fixture/__init__.py"))
		if err != nil {
			t.Fatal(err)
		}
		body := make([]byte, 16)
		magic, _ := hex.DecodeString("2b0e0d0a")
		copy(body, magic)
		binary.LittleEndian.PutUint32(body[8:12], uint32(source.ModTime().Unix()))
		binary.LittleEndian.PutUint32(body[12:16], uint32(source.Size()))
		if mode == "cache-magic" {
			body[0] ^= 1
		}
		if mode == "cache-flags" {
			binary.LittleEndian.PutUint32(body[4:8], 3)
		}
		if mode == "cache-mtime" {
			body[8] ^= 1
		}
		if mode == "cache-size" {
			body[12] ^= 1
		}
		// Deliberately not executable/marshal data. No code loader may inspect
		// or approve these bytes; only the exact cache role can be removed.
		body = append(body, []byte("never load or execute this cache body")...)
		files[cachePath] = body
		path := filepath.Join(plan.site, filepath.FromSlash(cachePath))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var record bytes.Buffer
	writer := csv.NewWriter(&record)
	for _, name := range names {
		if mode == "missing-wheel" && name == dist+"/WHEEL" {
			continue
		}
		body := files[name]
		sum := sha256.Sum256(body)
		row := []string{name, "sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]), strconv.Itoa(len(body))}
		if name == cachePath {
			row = []string{name, "", ""}
		}
		if err := writer.Write(row); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Write([]string{dist + "/RECORD", "", ""}); err != nil {
		t.Fatal(err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plan.site, dist, "RECORD"), record.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	requirement, _ := artifactpypi.ParseUpgradeRequirement("haa-promotion-fixture")
	input, err := artifactpypi.NewUpgradeInput(requirement, []artifactpypi.InstalledMetadataFile{{Directory: dist, Body: files[dist+"/METADATA"]}}, runtimeidentity.PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := beginPyPIVenvTransaction(plan)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := capturePyPINativeInventory(context.Background(), transaction, input)
	if err != nil {
		_ = transaction.close()
		t.Fatal(err)
	}
	return plan, transaction, inventory.projects["haa-promotion-fixture"], wheel, digest, archivePath
}

func TestNativePyPIProofRequiresCanonicalWheelFootprintBeforeOwnership(t *testing.T) {
	for _, mode := range []string{"no-compile", "cache", "user-file", "installer", "requested", "missing-wheel", "cache-tag", "cache-no-source", "cache-magic", "cache-flags", "cache-mtime", "cache-size", "wheel-digest", "wheel-identity", "generated-script"} {
		t.Run(mode, func(t *testing.T) {
			plan, transaction, project, wheel, digest, _ := nativeWheelOwnershipFixture(t, mode)
			defer func() {
				if err := transaction.close(); err != nil {
					t.Fatal(err)
				}
			}()
			if mode == "wheel-digest" {
				wheel.ObservedSHA256 = strings.Repeat("f", 64)
			}
			if mode == "wheel-identity" {
				wheel.Version = "2.0.0"
			}
			if mode == "generated-script" {
				wheel.Surface.EntryPointDetails = append(wheel.Surface.EntryPointDetails, artifactpypi.EntryPoint{Group: "console_scripts", Name: "tool", Module: "haa_promotion_fixture"})
			}
			runtimeCalls := 0
			destinations, proof, err := proveNativePyPIWheelFootprint(project, wheel, digest, func() (pypiNativeRuntime, error) {
				runtimeCalls++
				return pypiNativeRuntime{magic: "2b0e0d0a", cacheTag: "cpython-314"}, nil
			})
			accepted := mode == "no-compile" || mode == "cache"
			if accepted != (err == nil) {
				t.Fatal("independent native ownership proof acceptance changed", mode, err)
			}
			if len(transaction.current.Files) != 0 || transaction.metadata != nil {
				t.Fatal("proof lookup modified ownership before new inspection")
			}
			if _, err := os.Lstat(filepath.Join(plan.root, pypiVenvMetadata)); !os.IsNotExist(err) {
				t.Fatal("proof lookup wrote a ledger", err)
			}
			if !accepted {
				return
			}
			if len(destinations) != len(project.files) || len(proof) != len(destinations) {
				t.Fatal("incomplete old footprint proof")
			}
			cacheProof := false
			for i, destination := range destinations {
				if destination.ArtifactDigest != digest || destination.Digest != project.files[destination.key()].original.Digest || proof[i].Role == "" {
					t.Fatal("authentic old wheel and actual file fingerprints lost")
				}
				if strings.HasSuffix(destination.Relative, ".pyc") {
					cacheProof = proof[i].Role == "pinned-timestamp-cache-removal-only"
				}
			}
			if (mode == "cache") != cacheProof || (mode == "cache") != (runtimeCalls == 1) {
				t.Fatal("untrusted cache became approved code or ordinary bytes required a runtime probe")
			}
		})
	}
}
