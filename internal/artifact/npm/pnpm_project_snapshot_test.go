package npm

import (
	"context"
	"go.yaml.in/yaml/v3"
	"net/http"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func pnpmSnapshotContext(t *testing.T, target string, project bool) domain.InstallContext {
	t.Helper()
	path, err := domain.NewInstallTarget(target)
	if err != nil {
		t.Fatal(err)
	}
	var install domain.InstallContext
	if project {
		install, err = domain.NewNPMProjectInstallContext(path)
	} else {
		install, err = domain.NewInstallContext(path)
	}
	if err != nil {
		t.Fatal(err)
	}
	return install
}

func pnpmSnapshotFixture(t *testing.T, manifest, body []byte) (PNPMSelection, []domain.ProjectControlFile) {
	t.Helper()
	lock, err := parsePNPMTestLock(manifest, body)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]PNPMLockedPackage{}
	for _, pkg := range lock.Packages() {
		byPath["/"+pkg.Name+"/"+pkg.Version] = pkg
	}
	resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		pkg, ok := byPath[request.URL.Path]
		if !ok {
			t.Fatal("unexpected exact-version registry binding")
		}
		return response(http.StatusOK, "application/json", pnpmMetadataBody(t, pkg, publicRegistryURL+"snapshot/"+pkg.Name+"-"+pkg.Version+".tgz")), nil
	}))
	bound, err := resolver.BindPNPMLock(context.Background(), lock)
	if err != nil {
		t.Fatal(err)
	}
	manifestControl, err := domain.NewProjectControlFile("package.json", manifest, true)
	if err != nil {
		t.Fatal(err)
	}
	lockControl, err := domain.NewProjectControlFile("pnpm-lock.yaml", body, true)
	if err != nil {
		t.Fatal(err)
	}
	return bound, []domain.ProjectControlFile{manifestControl, lockControl}
}

func TestPNPMProjectSnapshotKeepsUniqueArchivesAndCompletePeerState(t *testing.T) {
	for _, test := range []struct {
		name, manifest      string
		body                []byte
		archives, instances int
	}{
		{"empty", `{}`, []byte(pnpmEmptyLock), 0, 0},
		{"single", pnpmSingleManifest, pnpmFixture(t, "single"), 1, 1},
		{"peer", pnpmPeerManifest, pnpmFixture(t, "peer"), 4, 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			bound, controls := pnpmSnapshotFixture(t, []byte(test.manifest), test.body)
			install := pnpmSnapshotContext(t, "/tmp/haa-pnpm-snapshot", true)
			snapshot, err := bound.ProjectSnapshot(install, controls, "fixture-runtime:pnpm")
			if err != nil || !snapshot.Valid() || snapshot.Source().String() != "npm" || snapshot.Context() != install || snapshot.DependencyFree() != (test.archives == 0) || len(snapshot.Dependencies()) != test.archives || len(snapshot.ControlDigests()) != 2 || len(bound.Lock().Snapshots()) != test.instances {
				t.Fatalf("complete snapshot lost: %v %v", snapshot, err)
			}
			reordered, err := bound.ProjectSnapshot(install, []domain.ProjectControlFile{controls[1], controls[0]}, "fixture-runtime:pnpm")
			if err != nil || reordered.GraphDigest() != snapshot.GraphDigest() {
				t.Fatal("control argument order changed identity")
			}
			if test.archives != 0 {
				copy := snapshot.Dependencies()
				copy[0] = domain.ResolvedArtifact{}
				if snapshot.Dependencies()[0].Identity().Name() == "" {
					t.Fatal("snapshot exposed mutable archives")
				}
			}
		})
	}
}

func TestPNPMProjectSnapshotBindsContextRuntimeAndRawControls(t *testing.T) {
	bound, controls := pnpmSnapshotFixture(t, []byte(pnpmPeerManifest), pnpmFixture(t, "peer"))
	install := pnpmSnapshotContext(t, "/tmp/haa-pnpm-snapshot", true)
	before, err := bound.ProjectSnapshot(install, controls, "fixture-runtime:pnpm")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, runtime string
		context       domain.InstallContext
		controls      []domain.ProjectControlFile
	}{
		{"target", "fixture-runtime:pnpm", pnpmSnapshotContext(t, "/tmp/haa-pnpm-other", true), controls},
		{"mode", "fixture-runtime:pnpm", pnpmSnapshotContext(t, "/tmp/haa-pnpm-snapshot", false), controls},
		{"runtime", "fixture-runtime:other", install, controls},
		{"manifest-bytes", "fixture-runtime:pnpm", install, []domain.ProjectControlFile{pnpmInputControl(t, "package.json", pnpmPeerManifest+"\n", true), controls[1]}},
		{"lock-bytes", "fixture-runtime:pnpm", install, []domain.ProjectControlFile{controls[0], pnpmInputControl(t, "pnpm-lock.yaml", string(controls[1].Body())+"\n", true)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			after, err := bound.ProjectSnapshot(test.context, test.controls, test.runtime)
			if err != nil || after.GraphDigest() == before.GraphDigest() {
				t.Fatalf("selection input not bound: %v", err)
			}
		})
	}
}

func TestPNPMProjectSnapshotRejectsUnboundAndMismatchedControls(t *testing.T) {
	bound, controls := pnpmSnapshotFixture(t, []byte(pnpmPeerManifest), pnpmFixture(t, "peer"))
	install := pnpmSnapshotContext(t, "/tmp/haa-pnpm-snapshot", true)
	path, _ := domain.NewInstallTarget("/tmp/haa-pnpm-python")
	python, err := domain.NewPythonVenvInstallContext(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, runtime string
		context       domain.InstallContext
		controls      []domain.ProjectControlFile
	}{
		{"missing", "fixture", install, controls[:1]},
		{"duplicate", "fixture", install, []domain.ProjectControlFile{controls[0], controls[0]}},
		{"extra", "fixture", install, append(append([]domain.ProjectControlFile{}, controls...), pnpmInputControl(t, ".npmrc", "registry=other", true))},
		{"absent", "fixture", install, []domain.ProjectControlFile{controls[0], pnpmInputControl(t, "pnpm-lock.yaml", "", false)}},
		{"empty", "fixture", install, []domain.ProjectControlFile{controls[0], pnpmInputControl(t, "pnpm-lock.yaml", "", true)}},
		{"manifest-drift", "fixture", install, []domain.ProjectControlFile{pnpmInputControl(t, "package.json", `{}`, true), controls[1]}},
		{"other-lock", "fixture", install, []domain.ProjectControlFile{controls[0], pnpmInputControl(t, "pnpm-lock.yaml", string(pnpmFixture(t, "single")), true)}},
		{"zero-context", "fixture", domain.InstallContext{}, controls},
		{"other-mode", "fixture", python, controls},
		{"no-runtime", "", install, controls},
		{"large-runtime", strings.Repeat("x", 1025), install, controls},
		{"runtime-newline", "fixture\nother", install, controls},
		{"runtime-space", "fixture other", install, controls},
		{"runtime-invalid-utf8", "fixture\xff", install, controls},
	} {
		t.Run(test.name, func(t *testing.T) {
			if result, err := bound.ProjectSnapshot(test.context, test.controls, test.runtime); err == nil || result.Valid() {
				t.Fatal("mismatched snapshot approved")
			}
		})
	}
	for _, selection := range []PNPMSelection{{}, {lock: bound.lock, artifacts: bound.Artifacts()}} {
		if result, err := selection.ProjectSnapshot(install, controls, "fixture"); err == nil || result.Valid() {
			t.Fatal("unbound selection accepted")
		}
	}
	broken := bound
	broken.artifacts = nil
	if _, err := broken.ProjectSnapshot(install, controls, "fixture"); err == nil {
		t.Fatal("missing archive coverage accepted")
	}
	var wire pnpmLockWire
	if err := yaml.Unmarshal(controls[1].Body(), &wire); err != nil {
		t.Fatal(err)
	}
	metadata := wire.Packages["react@18.3.1"]
	metadata.Deprecated = "explicit inert metadata drift"
	wire.Packages["react@18.3.1"] = metadata
	body, err := yaml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parsePNPMTestLock(controls[0].Body(), body); err != nil {
		t.Fatal("drift fixture must still parse", err)
	}
	changed := []domain.ProjectControlFile{controls[0], pnpmInputControl(t, "pnpm-lock.yaml", string(body), true)}
	if _, err := bound.ProjectSnapshot(install, changed, "fixture"); err == nil {
		t.Fatal("a different valid lock was paired with an earlier source binding")
	}
}
