package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type pnpmPreparationRoundTripper func(*http.Request) (*http.Response, error)

func (f pnpmPreparationRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Explicit structural test input, never public archive authenticity. The fake
// transport answers only the existing exact-version metadata request. Archive
// bytes are inert data for intake/transport tests, not executable package data.
func pnpmPreparationInputFixture(t *testing.T, runtime string) (string, artifactnpm.PNPMInspectionInput, []artifactnpm.ArchiveInventory) {
	t.Helper()
	root := t.TempDir()
	manifest := []byte(`{"dependencies":{"is-number":"6.0.0"}}`)
	lockfile, err := os.ReadFile(filepath.Join("..", "artifact", "npm", "testdata", "pnpm-v9-single.synthetic.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	lock, err := artifactnpm.ParsePNPMLock(manifest, lockfile, runtimeidentity.PNPMVersion)
	if err != nil {
		t.Fatal(err)
	}
	pkg := lock.Packages()[0]
	metadata, err := json.Marshal(map[string]any{"name": pkg.Name, "version": pkg.Version, "dist": map[string]string{"tarball": "https://registry.npmjs.org/is-number/-/is-number-6.0.0.tgz", "integrity": pkg.Integrity}})
	if err != nil {
		t.Fatal(err)
	}
	source, err := artifactnpm.NewResolver("https://registry.npmjs.org/", &http.Client{Transport: pnpmPreparationRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/is-number/6.0.0" {
			t.Fatal("unexpected source request", r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(metadata))}, nil
	})}, root)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := source.BindPNPMLock(context.Background(), lock)
	if err != nil {
		t.Fatal(err)
	}
	controls := []domain.ProjectControlFile{pnpmTestControl(t, "package.json", string(manifest), true), pnpmTestControl(t, "pnpm-lock.yaml", string(lockfile), true)}
	target, err := domain.NewInstallTarget(filepath.Join(root, "project"))
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewNPMProjectInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := selection.InspectionContext(install, controls, runtime)
	if err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("inert transport fixture archive bytes\n")
	hash := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
	if err != nil {
		t.Fatal(err)
	}
	subject, err := domain.NewAcquiredArtifactWithIntegrity(selection.Artifacts()[0].Identity(), digest, "intake:"+run.String()+":tarball", uint64(len(body)), pkg.Integrity, pkg.Integrity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, run.String()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, run.String(), "tarball.tgz"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	packageJSON := []byte(`{"name":"is-number","version":"6.0.0"}`)
	manifestHash := sha256.Sum256(packageJSON)
	manifestDigest, err := domain.NewSHA256Digest(hex.EncodeToString(manifestHash[:]))
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := artifactnpm.NewArchiveInventory(subject, []artifactnpm.ArchiveEntry{{Path: "package.json", Size: int64(len(packageJSON)), Mode: 0o644, Digest: manifestDigest}}, packageJSON)
	if err != nil {
		t.Fatal(err)
	}
	node, err := domain.ProjectDependencyNodeID(subject.Identity())
	if err != nil {
		t.Fatal(err)
	}
	input, err := frozen.BindAcquired(frozen.Snapshot(), map[domain.DependencyNodeID]domain.AcquiredArtifact{node: subject})
	if err != nil {
		t.Fatal(err)
	}
	return root, input, []artifactnpm.ArchiveInventory{inventory}
}

func TestPNPMIntakeFreezeRejectsChangedBytesAndSymlinks(t *testing.T) {
	for _, change := range []string{"unchanged", "bytes", "missing", "file-symlink", "parent-symlink"} {
		t.Run(change, func(t *testing.T) {
			root, input, _ := pnpmPreparationInputFixture(t, PNPMResolverRuntimeIdentity())
			introducer, err := NewDockerArtifactIntroducer(root, &pnpmControlTransport{})
			if err != nil {
				t.Fatal(err)
			}
			file, err := introducer.tarballPath(input.Artifacts()[0].ContentHandle())
			if err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "bytes":
				changed := bytes.Clone(original)
				changed[0] ^= 1
				if err := os.WriteFile(file, changed, 0o600); err != nil {
					t.Fatal(err)
				}
			case "missing":
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
			case "file-symlink":
				other := filepath.Join(root, "other.tgz")
				if err := os.WriteFile(other, original, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, file); err != nil {
					t.Fatal(err)
				}
			case "parent-symlink":
				parent := filepath.Dir(file)
				other := parent + "-moved"
				if err := os.Rename(parent, other); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, parent); err != nil {
					t.Fatal(err)
				}
			}
			archives, err := introducer.freezePNPMArchives(context.Background(), input)
			if change != "unchanged" {
				if err == nil || len(archives) != 0 {
					t.Fatal("changed intake accepted")
				}
				return
			}
			if err != nil || len(archives) != 1 || !bytes.Equal(archives[0].body, original) {
				t.Fatal("exact input unavailable", err)
			}
			if err := os.WriteFile(file, []byte("changed after freeze"), 0o600); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(archives[0].body, original) {
				t.Fatal("frozen bytes changed with intake")
			}
		})
	}
}

func TestPNPMPreparationPreflightDoesNoRuntimeWork(t *testing.T) {
	for _, change := range []string{"zero", "coverage", "runtime", "canceled", "changed-intake", "supply-failure"} {
		t.Run(change, func(t *testing.T) {
			runtime := PNPMResolverRuntimeIdentity()
			if change == "runtime" {
				runtime = "fixture:foreign-runtime"
			}
			root, input, inventories := pnpmPreparationInputFixture(t, runtime)
			runner, endpoints := &pnpmControlTransport{}, &pnpmCountingEndpoints{}
			probes, supplies := 0, 0
			cause := errors.New("fixture source supply failed")
			native, err := NewPNPMResolver(runner, endpoints, &recordingObserver{}, func(context.Context) (Capability, error) { probes++; return Capability{Available: true}, nil }, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			native.prepare = func(context.Context) (*pnpmPreparedTool, error) { supplies++; return nil, cause }
			preparer, err := NewPNPMInspectionPreparer(native, root)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			switch change {
			case "zero":
				input = artifactnpm.PNPMInspectionInput{}
			case "coverage":
				inventories = nil
			case "canceled":
				cancel(cause)
			case "changed-intake":
				file, err := preparer.intake.tarballPath(input.Artifacts()[0].ContentHandle())
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte("drift"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			layout, err := preparer.PrepareInspection(ctx, input, inventories)
			if err == nil || layout.Digest().String() != "" || runner.calls != 0 || endpoints.calls != 0 {
				t.Fatal("preflight exposed output or performed runtime work", err)
			}
			want := 0
			if change == "supply-failure" {
				want = 1
			}
			if probes != want || supplies != want {
				t.Fatal("preflight ordering changed", probes, supplies)
			}
			if (change == "canceled" || change == "supply-failure") && !errors.Is(err, cause) {
				t.Fatal("first cause lost", err)
			}
		})
	}
}

func TestPNPMInspectionCommandsHaveOnlyExactGuestInputs(t *testing.T) {
	file := pnpmProjectRoot + "/artifacts/" + strings.Repeat("a", 64) + ".tgz"
	for _, operation := range []string{"store-add", "fetch", "install"} {
		var files []string
		if operation == "store-add" {
			files = []string{file}
		}
		arguments := pnpmInspectionCommand(operation, files)
		for _, required := range []string{pnpmToolGuestPath, "--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--registry=https://registry.npmjs.org/", "--store-dir", pnpmProjectRoot + "/store", "--state-dir", pnpmProjectRoot + "/state"} {
			if !slices.Contains(arguments, required) {
				t.Fatal("missing fixed option", operation, required)
			}
		}
		if operation != "fetch" && !slices.Contains(arguments, "--offline") {
			t.Fatal("offline phase uses network")
		}
		if operation == "install" && (!slices.Contains(arguments, "--frozen-lockfile") || !slices.Contains(arguments, "--node-linker=isolated")) {
			t.Fatal("frozen isolated install changed")
		}
	}
	for _, files := range [][]string{nil, {"/tmp/foreign.tgz"}, {pnpmProjectRoot + "/artifacts/../artifacts/" + strings.Repeat("a", 64) + ".tgz"}, {file, file}, {pnpmProjectRoot + "/artifacts/not-a-digest.tgz"}} {
		if pnpmInspectionCommand("store-add", files) != nil {
			t.Fatal("unsafe file input accepted")
		}
	}
	if pnpmInspectionCommand("fetch", []string{file}) != nil || pnpmInspectionCommand("unknown", nil) != nil {
		t.Fatal("unexpected native operation or positional input accepted")
	}
}

type pnpmNetworkQueryRunner struct {
	body  []byte
	calls int
}

func (r *pnpmNetworkQueryRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls++
	if name != "docker" {
		return nil, errors.New("unexpected tool")
	}
	if r.calls == 1 {
		if len(args) != 4 || args[0] != "network" || args[1] != "disconnect" {
			return nil, errors.New("disconnect absent")
		}
		return nil, nil
	}
	return r.body, nil
}

func TestPNPMInspectionNetworkUnknownIsNotAbsence(t *testing.T) {
	for _, body := range []string{"{}", "null", "{invalid}", `{"retained":{}}`, strings.Repeat(" ", 65<<10) + "{}"} {
		r := &pnpmNetworkQueryRunner{body: []byte(body)}
		err := disconnectPNPMInspectionNetwork(context.Background(), r, strings.Repeat("a", 64), "owned-network")
		if (err == nil) != (body == "{}") || r.calls != 2 {
			t.Fatal("network query was treated as proven absence", body[:min(30, len(body))], err)
		}
	}
}

func TestPNPMGuestArchiveReadbackRejectsDriftAndAliases(t *testing.T) {
	root, input, _ := pnpmPreparationInputFixture(t, PNPMResolverRuntimeIdentity())
	introducer, err := NewDockerArtifactIntroducer(root, &pnpmControlTransport{})
	if err != nil {
		t.Fatal(err)
	}
	archives, err := introducer.freezePNPMArchives(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	archive := archives[0]
	for _, change := range []string{"exact", "bytes", "owner", "alias", "tail"} {
		header := tar.Header{Name: filepath.Base(archive.name), Typeflag: tar.TypeReg, Mode: 0o600, Uid: 1000, Gid: 1000}
		body := bytes.Clone(archive.body)
		switch change {
		case "bytes":
			body[0] ^= 1
		case "owner":
			header.Uid = 0
		case "alias":
			header.Typeflag = tar.TypeLink
			header.Linkname = "foreign"
			body = nil
		}
		output := pnpmControlArchive(t, header, body)
		if change == "tail" {
			output = append(output, '!')
		}
		runner := &pnpmControlTransport{body: output}
		err := verifyPNPMGuestArchive(context.Background(), runner, strings.Repeat("a", 64), archive)
		if (err == nil) != (change == "exact") {
			t.Fatal("readback drift admitted", change, err)
		}
	}
}
