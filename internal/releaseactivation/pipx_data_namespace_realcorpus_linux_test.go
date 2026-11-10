//go:build linux && realcorpus

package releaseactivation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/testutil/releasefixture"
	"golang.org/x/sys/unix"
)

func requirePipxRootDataNamespace(t *testing.T) string {
	t.Helper()
	if os.Getenv("HELOX_PIPX_ROOT_DATA_NAMESPACE") != "1" {
		t.Skip("explicit isolated root DATA filesystem qualification")
	}
	if os.Geteuid() != 0 {
		t.Fatal("root required only in disposable namespace")
	}
	for _, kind := range []string{"mnt", "net"} {
		self, err := os.Readlink("/proc/self/ns/" + kind)
		if err != nil {
			t.Fatal(err)
		}
		host, err := os.Readlink("/proc/1/ns/" + kind)
		if err != nil || self == host {
			t.Fatal("refusing Host namespace")
		}
	}
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		t.Fatal(err)
	}
	return os.Getenv("HELOX_PIPX_RELEASE_ACTUAL_DIR")
}
func assertPipxRootMember(t *testing.T, name string, expected []byte, mode os.FileMode) {
	t.Helper()
	body, err := os.ReadFile(name)
	if err != nil || !bytes.Equal(body, expected) {
		t.Fatal("DATA bytes differ", name, err)
	}
	info, err := os.Lstat(name)
	if err != nil {
		t.Fatal(err)
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || owner.Gid != 0 || owner.Nlink != 1 || info.Mode().Perm() != mode {
		t.Fatal("DATA owner/link/mode differs", name, info.Mode())
	}
}

func TestPipxActualFullWirePrivateStageInDisposableNamespace(t *testing.T) {
	source := requirePipxRootDataNamespace(t)
	body, err := os.ReadFile(filepath.Join(source, "helox-release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	release, err := newWireRelease(body, [][]byte{[]byte("unsigned local producer fixture")})
	if err != nil || len(release.files) != 12 {
		t.Fatal("complete DATA wire missing", err)
	}
	if err := syscall.Mount("tmpfs", "/var/lib", "tmpfs", syscall.MS_NODEV|syscall.MS_NOSUID, "mode=0755,size=536870912"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}
	fd, err := unix.PidfdOpen(os.Getpid(), 0)
	if err != nil {
		t.Fatal(err)
	}
	// Authorization/current-chain are explicit filesystem-test doubles only.
	peer := &controllerPeer{pidfd: os.NewFile(uintptr(fd), "pipx-stage-test"), uid: 1000, gid: 1000}
	defer peer.Close()
	current := InstalledRelease{record: registration{Schema: registrationSchema, Version: release.manifest.Version(), ClientUID: 1000, ClientGID: 1000}, manifest: release.manifest, proof: []byte("test-current")}
	ctx := context.Background()
	var preamble bytes.Buffer
	if err := writeWirePreamble(ctx, &preamble, release); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareWireRelease(ctx, &preamble, peer, current); !errors.Is(err, ErrUntrusted) {
		t.Fatal("production SDK accepted unsigned candidate", err)
	}
	if _, err := os.Lstat(releaseinstall.SystemStagingRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unsigned candidate wrote root stage")
	}
	payload, err := os.CreateTemp(t.TempDir(), "payload-")
	if err != nil {
		t.Fatal(err)
	}
	defer payload.Close()
	if err := writeWirePayloads(ctx, payload, release, func(name string) (io.ReadCloser, error) { return os.Open(filepath.Join(source, name)) }); err != nil {
		t.Fatal(err)
	}
	if err := payload.Sync(); err != nil {
		t.Fatal(err)
	}
	info, err := payload.Stat()
	if err != nil {
		t.Fatal(err)
	}
	var zipStart, offset, zipSize int64
	for _, file := range release.files {
		if file.IsRuntimeData() {
			zipStart, zipSize = offset, file.Size
		}
		offset += file.Size
	}
	if zipSize != int64(runtimeidentity.PipxToolSize) {
		t.Fatal("selected ZIP missing")
	}
	stage, err := stageCheckedWire(ctx, io.NewSectionReader(payload, 0, info.Size()), peer, release, release.proofs[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range release.files {
		expected, err := os.ReadFile(filepath.Join(source, file.Name))
		if err != nil {
			t.Fatal(err)
		}
		assertPipxRootMember(t, filepath.Join(stage.directory, file.Name), expected, 0600)
	}
	if err := stage.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stage.Close(); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"corrupt-ZIP", "truncated-ZIP", "bad-terminal-marker"} {
		t.Run(mode, func(t *testing.T) {
			var input io.Reader
			switch mode {
			case "corrupt-ZIP":
				first := []byte{0}
				if _, err := payload.ReadAt(first, zipStart); err != nil {
					t.Fatal(err)
				}
				first[0] ^= 1
				input = io.MultiReader(io.NewSectionReader(payload, 0, zipStart), bytes.NewReader(first), io.NewSectionReader(payload, zipStart+1, info.Size()-zipStart-1))
			case "truncated-ZIP":
				input = io.NewSectionReader(payload, 0, zipStart+zipSize-1)
			case "bad-terminal-marker":
				input = io.MultiReader(io.NewSectionReader(payload, 0, info.Size()-1), bytes.NewReader([]byte("!")))
			}
			if _, err := stageCheckedWire(ctx, input, peer, release, release.proofs[0]); err == nil {
				t.Fatal("bad DATA wire became prepared")
			}
			remaining, err := os.ReadDir(releaseinstall.SystemStagingRoot)
			if err != nil || len(remaining) != 0 {
				t.Fatal("partial DATA stage leaked", err)
			}
		})
	}
	t.Logf("actual unsigned candidate12payload/%dB: root single-link0600 exact bytes; selected ZIP corruption/truncation/marker failures cleaned; production SDK refusal before root writes; auth mock only/no activation", offset)
}

func pipxBootstrapDataFixture(t *testing.T, source string) (nativeBootstrapSet, []byte, []byte) {
	t.Helper()
	set := bootstrapFixture(t)
	zip, err := os.ReadFile(filepath.Join(source, runtimeidentity.PipxRuntimeAssetName))
	if err != nil || bodySHA256(zip) != runtimeidentity.PipxToolSHA256 {
		t.Fatal("selected original ZIP missing", err)
	}
	notice, err := os.ReadFile(filepath.Join(source, "pipx-NOTICE.txt"))
	if err != nil || bodySHA256(notice) != runtimeidentity.PipxNoticesSHA256 {
		t.Fatal("selected original notice missing", err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(set.release.body, &metadata); err != nil {
		t.Fatal(err)
	}
	assets := metadata["assets"].([]any)
	for name, data := range map[string][]byte{runtimeidentity.PipxRuntimeAssetName: zip, "pipx-NOTICE.txt": notice} {
		role := "release-metadata"
		if name == runtimeidentity.PipxRuntimeAssetName {
			role = "pipx-runtime"
		}
		assets = append(assets, map[string]any{"name": name, "platform": "linux/amd64", "role": role, "size": len(data), "sha256": fmt.Sprintf("%x", sha256.Sum256(data))})
		enrollmentWrite(t, filepath.Join(bootstrapDirectory, name), data, 0600)
	}
	metadata["assets"] = assets
	body, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	sbom, err := os.ReadFile(filepath.Join(bootstrapDirectory, "helox-release-sbom.cdx.json"))
	if err != nil {
		t.Fatal(err)
	}
	images, err := os.ReadFile(filepath.Join(bootstrapDirectory, "helox-runtime-images.json"))
	if err != nil {
		t.Fatal(err)
	}
	materials, err := releasefixture.BindCurrentRuntime(body, sbom, images)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"helox-release-manifest.json": materials.Manifest, "helox-release-sbom.cdx.json": materials.SBOM, "helox-runtime-images.json": materials.Images} {
		enrollmentWrite(t, filepath.Join(bootstrapDirectory, name), data, 0600)
	}
	release, err := newWireRelease(materials.Manifest, set.release.proofs)
	if err != nil {
		t.Fatal(err)
	}
	set.release = release
	if err := storeCheckedProof(context.Background(), release.manifest.SHA256(), set.proof); err != nil {
		t.Fatal(err)
	}
	set.record, err = nativeRegistration(release.manifest, set.configs)
	if err != nil {
		t.Fatal(err)
	}
	return set, zip, notice
}

func TestPipxBootstrapCopyRecoveryAndCleanupInDisposableNamespace(t *testing.T) {
	source := requirePipxRootDataNamespace(t)
	for _, mode := range []string{"fresh-complete-retry", "partial-DATA-copy-recovery", "executable-DATA-copy-refused", "tampered-DATA-refusal-and-retry", "foreign-DATA-source-retained"} {
		t.Run(mode, func(t *testing.T) {
			set, zip, notice := pipxBootstrapDataFixture(t, source)
			checks := bootstrapChecksFixture(t, set)
			ctx := context.Background()
			if _, err := readBootstrapSet(ctx); err == nil {
				t.Fatal("synthetic proof passed production SDK")
			}
			if _, err := os.Lstat(bootstrapIntentPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("unsigned read wrote initial intent")
			}
			if mode == "partial-DATA-copy-recovery" || mode == "executable-DATA-copy-refused" {
				body, err := json.Marshal(set.record)
				if err != nil {
					t.Fatal(err)
				}
				enrollmentWrite(t, bootstrapIntentPath, body, 0600)
				versions := filepath.Join(releaseinstall.SystemInstallRoot, "versions")
				if err := os.MkdirAll(versions, 0755); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{releaseinstall.SystemInstallRoot, versions} {
					if err := os.Chmod(name, 0755); err != nil {
						t.Fatal(err)
					}
				}
				name, err := releaseinstall.SystemCopyStageName(set.record.ManifestSHA256)
				if err != nil {
					t.Fatal(err)
				}
				partial := filepath.Join(versions, name)
				if err := os.Mkdir(partial, 0700); err != nil {
					t.Fatal(err)
				}
				enrollmentWrite(t, filepath.Join(partial, "helox-release-manifest.json"), set.release.body[:len(set.release.body)/2], 0444)
				partialZIP := filepath.Join(partial, runtimeidentity.PipxRuntimeAssetName)
				fileMode := os.FileMode(0444)
				if mode == "executable-DATA-copy-refused" {
					fileMode = 0555
				}
				enrollmentWrite(t, partialZIP, zip[:len(zip)/2], fileMode)
				if mode == "executable-DATA-copy-refused" {
					before, err := os.Lstat(partialZIP)
					if err != nil {
						t.Fatal(err)
					}
					if err := prepareBootstrapFiles(ctx, checks); err == nil {
						t.Fatal("executable DATA partial copy was accepted for cleanup")
					}
					after, err := os.Lstat(partialZIP)
					if err != nil || !os.SameFile(before, after) {
						t.Fatal("invalid DATA partial copy was removed or replaced")
					}
					assertPipxRootMember(t, partialZIP, zip[:len(zip)/2], 0555)
					return
				}
			}
			if mode == "tampered-DATA-refusal-and-retry" {
				changed := bytes.Clone(zip)
				changed[0] ^= 1
				enrollmentWrite(t, filepath.Join(bootstrapDirectory, runtimeidentity.PipxRuntimeAssetName), changed, 0600)
				if err := prepareBootstrapFiles(ctx, checks); err == nil {
					t.Fatal("tampered source DATA installed")
				}
				if _, err := os.Lstat(filepath.Join(releaseinstall.SystemInstallRoot, "current")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("bad DATA published current")
				}
				enrollmentWrite(t, filepath.Join(bootstrapDirectory, runtimeidentity.PipxRuntimeAssetName), zip, 0600)
			}
			if err := prepareBootstrapFiles(ctx, checks); err != nil {
				t.Fatal(err)
			}
			if err := prepareBootstrapFiles(ctx, checks); err != nil {
				t.Fatal("complete retry failed", err)
			}
			installedRoot := filepath.Join(releaseinstall.SystemInstallRoot, "versions", set.release.manifest.Version())
			assertPipxRootMember(t, filepath.Join(installedRoot, runtimeidentity.PipxRuntimeAssetName), zip, 0444)
			assertPipxRootMember(t, filepath.Join(installedRoot, "pipx-NOTICE.txt"), notice, 0444)
			if _, err := os.Lstat(registrationPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("file preparation published healthy registration")
			}
			if _, err := AcquireRuntimeLease(ctx); !errors.Is(err, ErrTransition) {
				t.Fatal("unregistered files enabled runtime", err)
			}
			installed := InstalledRelease{set.record, set.release.manifest, set.proof}
			if err := cleanupBootstrapSource(ctx, installed); err == nil {
				t.Fatal("mock installed proof passed production cleanup SDK")
			}
			if mode == "foreign-DATA-source-retained" {
				name := filepath.Join(bootstrapDirectory, runtimeidentity.PipxRuntimeAssetName)
				if err := os.Rename(name, filepath.Join(stateRoot, "retained-original-DATA")); err != nil {
					t.Fatal(err)
				}
				enrollmentWrite(t, name, []byte("foreign DATA replacement"), 0600)
				before := bootstrapSourceInventory(t)
				if err := discardBootstrapSource(ctx, installed); err == nil {
					t.Fatal("foreign DATA source removed")
				}
				if after := bootstrapSourceInventory(t); !reflect.DeepEqual(before, after) {
					t.Fatal("failed cleanup changed source")
				}
			} else {
				if err := discardBootstrapSource(ctx, installed); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Lstat(bootstrapDirectory); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("owned DATA source not cleaned")
				}
				assertPipxRootMember(t, filepath.Join(installedRoot, runtimeidentity.PipxRuntimeAssetName), zip, 0444)
			}
			t.Logf("%s: actual ZIP/NOTICE root installed0444/private source0600; mock auth/crypto filesystem mechanics only; unsigned production rejection/no registration or runtime activation", mode)
		})
	}
}
