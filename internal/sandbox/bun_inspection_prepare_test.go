package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// Reuse the controlled-intake fixture; these are inert transport bytes, not a
// static/crypto or native runtime qualification.
func bunPreparationInputFixture(t *testing.T, runtime string) (string, artifactnpm.BunInspectionInput, []artifactnpm.ArchiveInventory) {
	t.Helper()
	root, pnpm, inventories := pnpmPreparationInputFixture(t, runtime)
	subject := pnpm.Artifacts()[0]
	identity := subject.Identity()
	sri, _ := subject.DeclaredIntegrity()
	manifest := pnpm.Context().Controls()[0]
	if manifest.Name() != "package.json" {
		t.Fatal("fixture control order")
	}
	lockBody, err := json.Marshal(map[string]any{"lockfileVersion": 2, "configVersion": 1, "workspaces": map[string]any{"": map[string]any{"dependencies": map[string]string{identity.Name(): identity.Version()}}}, "packages": map[string]any{identity.Name(): []any{identity.Name() + "@" + identity.Version(), "", map[string]any{}, sri}}})
	if err != nil {
		t.Fatal(err)
	}
	lockControl := pnpmTestControl(t, "bun.lock", string(lockBody), true)
	lock, err := artifactnpm.ParseBunLock(manifest.Body(), lockControl, runtimeidentity.BunVersion)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := json.Marshal(map[string]any{"name": identity.Name(), "version": identity.Version(), "dist": map[string]string{"tarball": "https://registry.npmjs.org/" + identity.Name() + "/-/" + identity.Name() + "-" + identity.Version() + ".tgz", "integrity": sri}})
	if err != nil {
		t.Fatal(err)
	}
	source, err := artifactnpm.NewResolver("https://registry.npmjs.org/", &http.Client{Transport: pnpmPreparationRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/"+identity.Name()+"/"+identity.Version() {
			t.Fatal("unexpected fixture source")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(metadata))}, nil
	})}, root)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := source.BindBunLock(context.Background(), lock)
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := selection.InspectionContext(pnpm.Context().Snapshot().Context(), []domain.ProjectControlFile{manifest, lockControl}, runtime)
	if err != nil {
		t.Fatal(err)
	}
	node, err := domain.ProjectDependencyNodeID(identity)
	if err != nil {
		t.Fatal(err)
	}
	input, err := frozen.BindAcquired(frozen.Snapshot(), map[domain.DependencyNodeID]domain.AcquiredArtifact{node: subject})
	if err != nil {
		t.Fatal(err)
	}
	return root, input, inventories
}

func TestBunPreparationPreflightPreservesOrderingAndCause(t *testing.T) {
	for _, change := range []string{"zero", "coverage", "runtime", "canceled", "changed-intake", "supply-failure"} {
		t.Run(change, func(t *testing.T) {
			runtime := BunResolverRuntimeIdentity()
			if change == "runtime" {
				runtime = "fixture:foreign-runtime"
			}
			root, input, inventories := bunPreparationInputFixture(t, runtime)
			runner, endpoints := &pnpmControlTransport{}, &pnpmCountingEndpoints{}
			probes, supplies := 0, 0
			cause := errors.New("fixture bun supply failed")
			native, err := NewBunResolver(runner, endpoints, &recordingObserver{}, func(context.Context) (Capability, error) { probes++; return Capability{Available: true}, nil }, &recordingResolverPolicyService{})
			if err != nil {
				t.Fatal(err)
			}
			native.scope.prepare = func(context.Context) (nativeProjectTool, error) { supplies++; return nil, cause }
			preparer, err := NewBunInspectionPreparer(native, root)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			switch change {
			case "zero":
				input = artifactnpm.BunInspectionInput{}
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
				t.Fatal("preflight did runtime work or exposed output", err)
			}
			want := 0
			if change == "supply-failure" {
				want = 1
			}
			if probes != want || supplies != want {
				t.Fatal("preflight ordering differs", probes, supplies)
			}
			if (change == "canceled" || change == "supply-failure") && !errors.Is(err, cause) {
				t.Fatal("original cause lost", err)
			}
		})
	}
}

func TestBunInspectionCommandsAndPrivateSeedHaveOnlyFixedInputs(t *testing.T) {
	for _, operation := range []string{"seed", "install"} {
		args := bunInspectionCommand(operation)
		if len(args) != 14 || args[0] != bunToolGuestPath || args[2] != "install" || args[4] != "--ignore-scripts" || args[5] != "--no-env-file" || args[6] != "--registry=https://registry.npmjs.org/" || args[8] != "--backend=copyfile" || args[9] != "--linker=hoisted" {
			t.Fatal("command widened")
		}
		if operation == "seed" && (args[1] != "--cwd="+bunProjectRoot+"/seed" || args[3] != "--lockfile-only" || args[13] != "--offline") {
			t.Fatal("seed shape")
		}
		if operation == "install" && (args[1] != "--cwd="+bunProjectDirectory || args[3] != "--offline" || args[13] != "--frozen-lockfile") {
			t.Fatal("frozen shape")
		}
	}
	if bunInspectionCommand("add") != nil || bunInspectionCommand("run") != nil {
		t.Fatal("generic native action admitted")
	}
	root, input, _ := bunPreparationInputFixture(t, BunResolverRuntimeIdentity())
	runner := &pnpmControlTransport{}
	intake, err := NewDockerArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	archives, err := intake.freezeBunArchives(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := bunInspectionSeedManifest(input, archives)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Dependencies, Overrides map[string]string
		Scripts                 json.RawMessage
	}
	if json.Unmarshal(manifest, &document) != nil || len(document.Dependencies) != 1 || len(document.Overrides) != 1 || document.Scripts != nil {
		t.Fatal("seed private metadata")
	}
	archive := archives[0]
	alias := "haa-cache-" + archive.subject.Digest().String()
	file := "file:" + bunProjectRoot + "/" + archive.name
	if document.Dependencies[alias] != file || document.Overrides[archive.subject.Identity().Name()+"@*"] != file {
		t.Fatal("seed changed source locator")
	}
	changed := append([]pnpmInspectionArchive(nil), archives...)
	changed[0].name = "artifacts/../../escape.tgz"
	if _, err := bunInspectionSeedManifest(input, changed); err == nil {
		t.Fatal("foreign archive path")
	}
	if _, err := bunInspectionSeedManifest(input, nil); err == nil {
		t.Fatal("partial source coverage")
	}
}

func TestBunInstalledProjectRejectsUnknownControlsBeforeBinding(t *testing.T) {
	_, input, inventories := bunPreparationInputFixture(t, BunResolverRuntimeIdentity())
	for _, fault := range []string{"complete", "unknown", "changed", "missing"} {
		t.Run(fault, func(t *testing.T) {
			var output bytes.Buffer
			writer := tar.NewWriter(&output)
			if err := writer.WriteHeader(&tar.Header{Name: "project", Typeflag: tar.TypeDir, Mode: 0o700, Uid: 1000, Gid: 1000}); err != nil {
				t.Fatal(err)
			}
			for _, control := range input.Context().Controls() {
				if fault == "missing" && control.Name() == "bun.lock" {
					continue
				}
				body := control.Body()
				if fault == "changed" && control.Name() == "package.json" {
					body = append(body, '\n')
				}
				if err := writer.WriteHeader(&tar.Header{Name: "project/" + control.Name(), Typeflag: tar.TypeReg, Mode: 0o600, Uid: 1000, Gid: 1000, Size: int64(len(body))}); err != nil {
					t.Fatal(err)
				}
				if _, err := writer.Write(body); err != nil {
					t.Fatal(err)
				}
			}
			name := input.Artifacts()[0].Identity().Name()
			for _, directory := range []string{"project/node_modules", "project/node_modules/" + name} {
				if err := writer.WriteHeader(&tar.Header{Name: directory, Typeflag: tar.TypeDir, Mode: 0o755, Uid: 1000, Gid: 1000}); err != nil {
					t.Fatal(err)
				}
			}
			if len(inventories) != 1 || len(inventories[0].Entries()) != 1 || inventories[0].Entries()[0].Path != "package.json" {
				t.Fatal("project positive fixture inventory changed")
			}
			manifest := inventories[0].Manifest()
			if err := writer.WriteHeader(&tar.Header{Name: "project/node_modules/" + name + "/package.json", Typeflag: tar.TypeReg, Mode: 0o644, Uid: 1000, Gid: 1000, Size: int64(len(manifest))}); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write(manifest); err != nil {
				t.Fatal(err)
			}
			if fault == "unknown" {
				body := []byte("unknown")
				if err := writer.WriteHeader(&tar.Header{Name: "project/unknown", Typeflag: tar.TypeReg, Mode: 0o600, Uid: 1000, Gid: 1000, Size: int64(len(body))}); err != nil {
					t.Fatal(err)
				}
				if _, err := writer.Write(body); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			layout, err := bindBunInspectionProject(context.Background(), input, inventories, output.Bytes())
			if fault == "complete" {
				if err != nil || !layout.MatchesInput(input) {
					t.Fatal("positive complete project failed", err)
				}
				return
			}
			if err == nil || layout.Digest().String() != "" {
				t.Fatal("unknown/changed/missing project controls admitted")
			}
		})
	}
}
