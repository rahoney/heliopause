package npm

import (
	"os"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func bunActualControls(t *testing.T, name string) ([]byte, []byte, bool) {
	t.Helper()
	manifest, err := os.ReadFile("testdata/bun-lock/" + name + ".package.json")
	if err != nil {
		t.Fatal(err)
	}
	if name == "empty" {
		return manifest, nil, false
	}
	return manifest, bunNativeFixture(t, name), true
}

func TestBunProjectSnapshotFreezesCompleteNativeControlsAndUniqueArchives(t *testing.T) {
	for _, test := range []struct {
		name      string
		artifacts int
	}{{"empty", 0}, {"single", 1}, {"transitive", 2}, {"peer", 4}} {
		t.Run(test.name, func(t *testing.T) {
			manifest, body, present := bunActualControls(t, test.name)
			selected, controls := bunTestSelection(t, manifest, body, present)
			install := pnpmSnapshotContext(t, "/tmp/haa-bun-snapshot", true)
			snapshot, err := selected.ProjectSnapshot(install, controls, "fixture-runtime:bun")
			if err != nil || !snapshot.Valid() || snapshot.Source().String() != "npm" || snapshot.Context() != install || len(snapshot.Dependencies()) != test.artifacts || len(snapshot.ControlDigests()) != 2 || snapshot.DependencyFree() != (test.artifacts == 0) {
				t.Fatal("snapshot coverage differs", err)
			}
			reordered, err := selected.ProjectSnapshot(install, []domain.ProjectControlFile{controls[1], controls[0]}, "fixture-runtime:bun")
			if err != nil || reordered.GraphDigest() != snapshot.GraphDigest() {
				t.Fatal("control argument order changed selection")
			}
			changedRuntime, err := selected.ProjectSnapshot(install, controls, "fixture-runtime:bun-other")
			if err != nil || changedRuntime.GraphDigest() == snapshot.GraphDigest() {
				t.Fatal("runtime was omitted from the fingerprint")
			}
			changed := append([]domain.ProjectControlFile(nil), controls...)
			changed[0] = bunTestControl(t, "package.json", append(manifest, '\n'), true)
			if got, err := selected.ProjectSnapshot(install, changed, "fixture-runtime:bun"); err == nil || got.Valid() {
				t.Fatal("changed manifest bytes reused earlier source selection")
			}
			changed = append([]domain.ProjectControlFile(nil), controls...)
			changed[1] = bunTestControl(t, "bun.lock", append(body, '\n'), true)
			if got, err := selected.ProjectSnapshot(install, changed, "fixture-runtime:bun"); err == nil || got.Valid() {
				t.Fatal("changed lock bytes/presence reused earlier source selection")
			}
			for _, runtime := range []string{"", "bad runtime", "bad\x00runtime"} {
				if got, err := selected.ProjectSnapshot(install, controls, runtime); err == nil || got.Valid() {
					t.Fatal("unbound runtime accepted")
				}
			}
			if got, err := (BunSelection{}).ProjectSnapshot(install, controls, "fixture-runtime:bun"); err == nil || got.Valid() {
				t.Fatal("unbound selection produced a snapshot")
			}
		})
	}
}

func TestBunProjectSnapshotRejectsMissingDuplicateAndForeignArchiveCoverage(t *testing.T) {
	manifest, body, present := bunActualControls(t, "peer")
	selected, controls := bunTestSelection(t, manifest, body, present)
	install := pnpmSnapshotContext(t, "/tmp/haa-bun-snapshot", true)
	for _, fault := range []string{"missing", "duplicate", "foreign", "extra"} {
		t.Run(fault, func(t *testing.T) {
			changed := selected
			changed.artifacts = selected.Artifacts()
			switch fault {
			case "missing":
				changed.artifacts = changed.artifacts[1:]
			case "duplicate":
				changed.artifacts[0] = changed.artifacts[1]
			case "foreign":
				changed.artifacts[0] = domain.ResolvedArtifact{}
			case "extra":
				changed.artifacts = append(changed.artifacts, changed.artifacts[0])
			}
			if got, err := changed.ProjectSnapshot(install, controls, "fixture-runtime:bun"); err == nil || got.Valid() {
				t.Fatal("incomplete source coverage produced a snapshot")
			}
		})
	}
}
