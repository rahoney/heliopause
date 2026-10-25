package sandbox

import (
	"bytes"
	"os"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func uvEmptyProjectUpdateFixture(t *testing.T) ([]domain.ProjectControlFile, []domain.ProjectControlFile) {
	t.Helper()
	manifest, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	input, err := artifactpypi.ParseUVProjectInput(manifest, nil, false, artifactpypi.UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	original, err := NewUVProjectOriginalControls(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := artifactpypi.ParseUVNativeProjectLock(input, body)
	if err != nil {
		t.Fatal(err)
	}
	export, err := lock.ParseTargetExport([]byte("lock-version='1.0'\ncreated-by='uv'\nrequires-python='==" + PinnedPythonRuntime().PythonVersion + "'\npackages=[]\n"))
	if err != nil {
		t.Fatal(err)
	}
	provenance, err := export.Controls()
	if err != nil {
		t.Fatal(err)
	}
	selected := append([]domain.ProjectControlFile(nil), original...)
	for i, c := range selected {
		for _, p := range provenance {
			if c.Name() == p.Name() {
				selected[i] = p
			}
		}
	}
	return original, selected
}

func uvEmptyProjectUpdate(t *testing.T, original, selected []domain.ProjectControlFile) domain.ProjectDependencyUpdate {
	t.Helper()
	digests := []domain.ProjectControlDigest{}
	for _, c := range selected {
		d, err := domain.NewProjectControlDigest(c.Name(), c.Digest())
		if err != nil {
			t.Fatal(err)
		}
		digests = append(digests, d)
	}
	snapshot, err := domain.NewDependencyFreeProjectSnapshot(uvSyncTestContext(t), artifactpypi.PublicPyPIProfile().Source(), digests, selected[0].Digest())
	if err != nil {
		t.Fatal(err)
	}
	update, err := domain.NewProjectDependencyUpdateFromSnapshot(original, selected, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return update
}

func TestUVProjectControlOnlyUpdateRequiresLockUpgradeIntent(t *testing.T) {
	for _, intent := range []artifactpypi.UVProjectIntent{artifactpypi.UVProjectSync, artifactpypi.UVProjectLockUpgrade} {
		original, selected := uvEmptyProjectUpdateFixture(t)
		input, err := artifactpypi.ParseUVProjectInput(original[0].Body(), selected[1].Body(), true, intent, "")
		if err != nil {
			t.Fatal(err)
		}
		original, err = NewUVProjectOriginalControls(input)
		if err != nil {
			t.Fatal(err)
		}
		for i, c := range selected {
			if c.Name() == artifactpypi.UVProjectIntentControl {
				for _, o := range original {
					if o.Name() == c.Name() {
						selected[i] = o
					}
				}
			}
		}
		base := uvEmptyProjectUpdate(t, original, selected)
		controls, err := domain.NewPythonProjectControlContext(base.Snapshot().Context().Target())
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := domain.NewDependencyFreeProjectSnapshot(controls, base.Snapshot().Source(), base.Snapshot().ControlDigests(), base.Snapshot().GraphDigest())
		if err != nil {
			t.Fatal(err)
		}
		update, err := domain.NewProjectDependencyUpdateFromSnapshot(original, selected, snapshot)
		if err != nil {
			t.Fatal(err)
		}
		err = ValidateUVProjectUpdate(update)
		if (err == nil) != (intent == artifactpypi.UVProjectLockUpgrade) {
			t.Fatal("control-only intent boundary changed", intent, err)
		}
	}
}

func TestUVProjectUpdateRevalidatesNativeProvenance(t *testing.T) {
	for _, name := range []string{"valid", "changed intent", "stale runtime", "changed manifest", "native lock", "raw export", "derived view", "missing controls"} {
		t.Run(name, func(t *testing.T) {
			original, selected := uvEmptyProjectUpdateFixture(t)
			change := func(controls []domain.ProjectControlFile, name string, body []byte) {
				for i, c := range controls {
					if c.Name() == name {
						var err error
						controls[i], err = domain.NewProjectControlFile(name, body, true)
						if err != nil {
							t.Fatal(err)
						}
						return
					}
				}
				t.Fatal("fixture lacks control", name)
			}
			switch name {
			case "changed intent":
				change(selected, artifactpypi.UVProjectIntentControl, []byte("{}"))
			case "stale runtime":
				change(original, uvProjectRuntimeControlName, []byte("stale"))
				change(selected, uvProjectRuntimeControlName, []byte("stale"))
			case "changed manifest":
				change(selected, artifactpypi.UVProjectInputControl, append(selected[0].Body(), []byte("\n# rewrite\n")...))
			case "native lock":
				change(selected, artifactpypi.UVNativeLockControl, []byte("version=99"))
			case "raw export":
				change(selected, artifactpypi.UVProjectExportControl, []byte("lock-version='1.0'\npackages=[]"))
			case "derived view":
				for _, c := range selected {
					if c.Name() == artifactpypi.UVSelectionLockControlName {
						change(selected, c.Name(), bytes.ReplaceAll(c.Body(), []byte("uv"), []byte("other")))
					}
				}
			case "missing controls":
				original = original[:5]
				selected = selected[:5]
			}
			err := ValidateUVProjectUpdate(uvEmptyProjectUpdate(t, original, selected))
			if (err == nil) != (name == "valid") {
				t.Fatal("native provenance boundary", name, err)
			}
		})
	}
	if ValidateUVProjectUpdate(domain.ProjectDependencyUpdate{}) == nil {
		t.Fatal("zero update accepted")
	}
}

func TestUVProjectOriginalLockMayPrecedeEditedManifest(t *testing.T) {
	manifest, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	manifest = bytes.ReplaceAll(manifest, []byte("dependencies = []"), []byte(`dependencies = ["packaging==25.0"]`))
	input, err := artifactpypi.ParseUVProjectInput(manifest, lock, true, artifactpypi.UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	controls, err := NewUVProjectOriginalControls(input)
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseUVProjectOriginalControls(controls)
	if err != nil || !bytes.Equal(got.Body(), manifest) {
		t.Fatal("historical original lock prevented edited manifest selection", err)
	}
	if _, err := artifactpypi.ParseUVNativeProjectLock(input, lock); err == nil {
		t.Fatal("stale lock accepted as selected current result")
	}
}
