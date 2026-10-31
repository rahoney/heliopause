//go:build linux

package releasebootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/testutil/releasefixture"
)

type sourceFixture struct {
	manifest                    releaseinstall.ReleaseManifest
	files                       map[string][]byte
	proof                       []byte
	closeErr                    error
	closeCalls, opens           int
	badSize, badHash, readClose bool
	onClose                     func()
}

func (f *sourceFixture) BootstrapMaterials() (releaseinstall.ReleaseManifest, [][]byte, error) {
	return f.manifest, [][]byte{f.proof}, nil
}
func (f *sourceFixture) OpenBootstrapMember(_ context.Context, name string) (io.ReadCloser, uint64, error) {
	f.opens++
	body, found := f.files[name]
	if !found {
		return nil, 0, errors.New("unknown member")
	}
	size := uint64(len(body))
	if f.badSize {
		size++
	}
	if f.badHash {
		body = []byte("wrong bytes")
	}
	closeErr := error(nil)
	if f.readClose {
		closeErr = errors.New("fixture reader close failed")
	}
	return &fixtureReader{Reader: bytes.NewReader(body), closeErr: closeErr}, size, nil
}
func (f *sourceFixture) Close() error {
	f.closeCalls++
	if f.onClose != nil {
		f.onClose()
	}
	return f.closeErr
}

type fixtureReader struct {
	*bytes.Reader
	closeErr error
}

func (r *fixtureReader) Close() error { return r.closeErr }

func preparationFixture(t *testing.T) *sourceFixture {
	t.Helper()
	base := filepath.Join("..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0")
	read := func(name string) []byte {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(base, name))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	var value map[string]any
	if err := json.Unmarshal(read("helox-release-manifest.json"), &value); err != nil {
		t.Fatal(err)
	}
	f := &sourceFixture{files: map[string][]byte{}, proof: read("actual-signed-bundle.json")}
	roles := map[string]string{"helox-linux-amd64": "helox", "haa_gvisor_observer-linux-amd64": "gvisor-observer", "haa-network-policy-helper-linux-amd64": "network-policy-helper", "haa-release-installer-linux-amd64": "release-installer", "pnpm-linux-amd64": "pnpm", "pnpm-NOTICE.txt": "release-metadata", "bun-linux-amd64": "bun", "bun-NOTICE.txt": "release-metadata"}
	hashes := func(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }
	assets := value["assets"].([]any)
	seen := map[string]bool{}
	for _, raw := range assets {
		asset := raw.(map[string]any)
		name := asset["name"].(string)
		if _, found := roles[name]; found {
			body := []byte("bootstrap-fixture-" + name)
			if name == "pnpm-NOTICE.txt" || name == "bun-NOTICE.txt" {
				var err error
				body, err = os.ReadFile(filepath.Join("..", "..", "tools", strings.TrimSuffix(name, "-NOTICE.txt"), "NOTICE"))
				if err != nil {
					t.Fatal(err)
				}
			}
			f.files[name] = body
			asset["sha256"] = hashes(body)
			asset["size"] = len(body)
			seen[name] = true
		}
	}
	for name, role := range roles {
		if !seen[name] {
			body := []byte("bootstrap-fixture-" + name)
			if name == "pnpm-NOTICE.txt" || name == "bun-NOTICE.txt" {
				var err error
				body, err = os.ReadFile(filepath.Join("..", "..", "tools", strings.TrimSuffix(name, "-NOTICE.txt"), "NOTICE"))
				if err != nil {
					t.Fatal(err)
				}
			}
			f.files[name] = body
			assets = append(assets, map[string]any{"name": name, "role": role, "platform": "linux/amd64", "sha256": hashes(body), "size": len(body)})
		}
	}
	value["assets"] = assets
	for _, name := range []string{"helox-release-sbom.cdx.json", "helox-runtime-images.json"} {
		f.files[name] = read(name)
	}
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	materials, err := releasefixture.BindCurrentRuntime(body, f.files["helox-release-sbom.cdx.json"], f.files["helox-runtime-images.json"])
	if err != nil {
		t.Fatal(err)
	}
	body = materials.Manifest
	f.files["helox-release-sbom.cdx.json"], f.files["helox-runtime-images.json"] = materials.SBOM, materials.Images
	f.files["helox-release-manifest.json"] = body
	f.manifest, err = releaseinstall.ParseReleaseManifest(body)
	if err != nil {
		t.Fatal(err)
	}
	if f.manifest.RuntimeLockSHA256() != runtimeidentity.RuntimeLockSHA256 {
		t.Fatal("fixture runtime differs from compiled lock")
	}
	return f
}

func TestOrdinaryPreparationExportsOnlyBoundFilesAndMakesNoInstallationGrant(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("ordinary-user preparation contract")
	}
	f := preparationFixture(t)
	deps := preparationDependencies{source: func(context.Context, string, string) (releaseSource, error) { return f, nil }, verify: func(context.Context, releaseinstall.ReleaseManifest, [][]byte) ([]byte, error) {
		return []byte("mocked-provenance-for-file-mechanics"), nil
	}}
	var progress []ports.Progress
	ctx := ports.WithProgress(context.Background(), func(event ports.Progress) { progress = append(progress, event) })
	result, err := prepare(ctx, "v0.1.0", deps)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(result.Directory)
	if len(result.Files) != 11 || len(result.Proofs) != 1 || result.Version != "v0.1.0" || result.ManifestSHA256 != f.manifest.SHA256() || f.closeCalls != 1 || f.opens != 11 {
		t.Fatal("incomplete preparation", result, f.closeCalls, f.opens)
	}
	info, err := os.Lstat(result.Directory)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatal(err)
	}
	for _, name := range result.Files {
		path := filepath.Join(result.Directory, name)
		body, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(body, f.files[name]) {
			t.Fatal("prepared bytes changed", err)
		}
		info, err := os.Lstat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("prepared member is not private", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(result.Directory, "intake")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("completed original intake retained", err)
	}
	if len(progress) == 0 || progress[0].Stage != "resolve" {
		t.Fatal("progress missing")
	}
	// The real SDK rejects this fabricated ninth asset; the positive above is
	// file mechanics only, never an attestation or initial-install qualification.
	if _, err := verifyMaterials(ctx, f.manifest, [][]byte{f.proof}); err == nil {
		t.Fatal("old proof authorized synthetic native9")
	}
}

func TestPreparationRejectsForeignRuntimeBeforeVerificationAndFileOpen(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("ordinary-user preparation contract")
	}
	f := preparationFixture(t)
	var value map[string]any
	if err := json.Unmarshal(f.files["helox-release-manifest.json"], &value); err != nil {
		t.Fatal(err)
	}
	foreign := strings.Repeat("a", 64)
	if foreign == runtimeidentity.RuntimeLockSHA256 {
		foreign = strings.Repeat("b", 64)
	}
	value["runtime_lock"].(map[string]any)["sha256"] = foreign
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	f.manifest, err = releaseinstall.ParseReleaseManifest(body)
	if err != nil {
		t.Fatal(err)
	}
	verified := false
	deps := preparationDependencies{source: func(context.Context, string, string) (releaseSource, error) { return f, nil }, verify: func(context.Context, releaseinstall.ReleaseManifest, [][]byte) ([]byte, error) {
		verified = true
		return []byte("must not verify"), nil
	}}
	result, err := prepare(context.Background(), "v0.1.0", deps)
	var failure *ports.BootstrapPreparationFailure
	if !errors.As(err, &failure) || failure.Code() != ports.BootstrapProvenanceFailed || verified || f.opens != 0 || f.closeCalls != 1 || result.Directory != "" {
		t.Fatalf("foreign runtime crossed boundary: %+v %v opens=%d verify=%v", result, err, f.opens, verified)
	}
}

func TestPreparationPreservesPrimaryFailureAndCleansOnlyOwnedFiles(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("ordinary-user preparation contract")
	}
	for _, kind := range []string{"acquire", "signature", "size", "hash", "reader-close", "source-close", "unknown-output", "post-copy-tamper", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			f := preparationFixture(t)
			sentinel := errors.New("primary fixture failure")
			var directory string
			deps := preparationDependencies{source: func(_ context.Context, intake, version string) (releaseSource, error) {
				directory = filepath.Dir(intake)
				t.Cleanup(func() {
					if err := os.RemoveAll(directory); err != nil {
						t.Error(err)
					}
				})
				if kind == "acquire" {
					return f, sentinel
				}
				if kind == "unknown-output" {
					if err := os.WriteFile(filepath.Join(directory, "foreign.txt"), []byte("retain"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return f, nil
			}, verify: func(context.Context, releaseinstall.ReleaseManifest, [][]byte) ([]byte, error) {
				if kind == "signature" || kind == "unknown-output" {
					return nil, sentinel
				}
				return []byte("mocked-proof"), nil
			}}
			switch kind {
			case "size":
				f.badSize = true
			case "hash":
				f.badHash = true
			case "reader-close":
				f.readClose = true
			case "source-close":
				f.closeErr = sentinel
			}
			if kind == "post-copy-tamper" {
				f.onClose = func() {
					if err := os.WriteFile(filepath.Join(directory, "helox-linux-amd64"), []byte("retain changed output"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "cancel" {
				cancel()
			}
			result, err := prepare(ctx, "v0.1.0", deps)
			if err == nil {
				t.Fatal("failed preparation succeeded")
			}
			if kind == "cancel" {
				if directory != "" || !errors.Is(err, context.Canceled) {
					t.Fatal("cancellation reached source", err)
				}
				return
			}
			if f.closeCalls != 1 {
				t.Fatal("source cleanup missing", f.closeCalls)
			}
			if kind == "acquire" || kind == "signature" || kind == "source-close" || kind == "unknown-output" {
				if !errors.Is(err, sentinel) {
					t.Fatal("primary cause lost", err)
				}
			}
			if kind == "post-copy-tamper" {
				defer os.RemoveAll(directory)
				if result.Directory != directory {
					t.Fatal("changed output was not retained")
				}
				body, readErr := os.ReadFile(filepath.Join(directory, "helox-linux-amd64"))
				if readErr != nil || string(body) != "retain changed output" {
					t.Fatal("changed output was removed", readErr)
				}
				var failure *ports.BootstrapPreparationFailure
				if !errors.As(err, &failure) || failure.Code() != ports.BootstrapExportFailed {
					t.Fatal("handoff verification did not fail first", err)
				}
			} else if kind == "unknown-output" {
				defer os.RemoveAll(directory)
				body, readErr := os.ReadFile(filepath.Join(directory, "foreign.txt"))
				if readErr != nil || string(body) != "retain" || result.Directory != directory {
					t.Fatal("unknown output removed", readErr)
				}
				var failure *ports.BootstrapPreparationFailure
				if !errors.As(err, &failure) || failure.Code() != ports.BootstrapProvenanceFailed {
					t.Fatal("cleanup replaced primary code", err)
				}
			} else {
				if result.Directory != "" {
					t.Fatal("completed failed output retained", result.Directory)
				}
				if _, err := os.Lstat(directory); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("failed output not removed", err)
				}
			}
		})
	}
}

func TestInvalidPreparationNeverCallsSource(t *testing.T) {
	calls := 0
	deps := preparationDependencies{source: func(context.Context, string, string) (releaseSource, error) {
		calls++
		return nil, errors.New("unreachable")
	}, verify: verifyMaterials}
	for _, version := range []string{"latest", "v1.2", "v1.2.3/escape", ""} {
		if _, err := prepare(context.Background(), version, deps); err == nil {
			t.Fatal("invalid version accepted")
		}
	}
	if calls != 0 {
		t.Fatal("invalid request reached source")
	}
}

func TestOwnedBootstrapCleanupKeepsReplacedFilesAndRemovesIndependentOwnedData(t *testing.T) {
	output, err := newOwnedOutput()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(output.path)
	for _, name := range []string{"owned", "replaced"} {
		body := []byte(name)
		sum := sha256.Sum256(body)
		if err := output.copy(context.Background(), name, bytes.NewReader(body), uint64(len(body)), hex.EncodeToString(sum[:])); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(output.path, "replaced")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := output.removeIntake(); err != nil {
		t.Fatal(err)
	}
	if err := output.discardFiles(); err == nil {
		t.Fatal("replaced output accepted as owned")
	}
	after, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("replaced output changed", err)
	}
	if _, err := os.Lstat(filepath.Join(output.path, "owned")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("independent known output was not removed", err)
	}
	if err := output.close(); err != nil {
		t.Fatal(err)
	}
}
