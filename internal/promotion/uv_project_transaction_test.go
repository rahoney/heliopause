package promotion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type uvProjectTransactionFixture struct {
	tx        *pypiVenvTransaction
	root      string
	controls  []domain.ProjectControlFile
	originals map[string]*pypiFileSnapshot
}

// Synthetic filesystem transaction inputs, not approval/native installation.
func newUVProjectTransactionFixture(t *testing.T, existingLock, empty bool) uvProjectTransactionFixture {
	t.Helper()
	old, _ := makePypiVenvFixture(t)
	root := realPromotionRoot(t)
	venv := filepath.Join(root, ".venv")
	if err := os.Rename(old, venv); err != nil {
		t.Fatal(err)
	}
	plan, err := discoverPythonVenv(venv)
	if err != nil {
		t.Fatal(err)
	}
	runVenvFixture(t, plan, map[string]string{"site/old.py": "old owned bytes", "scripts/old-tool": "old script"}, "old")
	if err := os.WriteFile(filepath.Join(plan.site, "user.txt"), []byte("preserve user bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}
	// Project the historical format onto the current runtime; keep captured testdata unchanged.
	manifest = bytes.ReplaceAll(manifest, []byte("3.14.7"), []byte(runtimeidentity.PythonVersion))
	lock, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	// Project the historical format onto the current runtime; keep captured testdata unchanged.
	lock = bytes.ReplaceAll(lock, []byte("3.14.7"), []byte(runtimeidentity.PythonVersion))
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	if existingLock {
		if err := os.WriteFile(filepath.Join(root, "uv.lock"), lock, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	originals := map[string]*pypiFileSnapshot{}
	for _, p := range []string{filepath.Join(root, "pyproject.toml"), filepath.Join(root, "uv.lock"), filepath.Join(plan.site, "old.py"), filepath.Join(venv, "bin/old-tool"), filepath.Join(venv, pypiVenvMetadata), filepath.Join(plan.site, "user.txt")} {
		if _, err := os.Lstat(p); errors.Is(err, os.ErrNotExist) {
			originals[p] = nil
			continue
		}
		s, err := snapshotPyPIFile(p)
		if err != nil {
			t.Fatal(err)
		}
		originals[p] = &s
	}
	tx, err := beginPyPIVenvTransaction(plan)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tx.close(); err != nil {
			t.Error(err)
		}
	})
	intent, selector := artifactpypi.UVProjectSync, ""
	if !empty {
		intent, selector = artifactpypi.UVProjectAdd, "packaging==25.0"
	}
	if _, err := freezeUVProjectFiles(context.Background(), tx, root, intent, selector, []string{realPromotionRoot(t)}); err != nil {
		t.Fatal(err)
	}
	if !empty {
		manifest = bytes.ReplaceAll(manifest, []byte("dependencies = []"), []byte(`dependencies = ["packaging==25.0"]`))
		lock = []byte("version=1\nrevision=5\nrequires-python='==" + runtimeidentity.PythonVersion + "'\n[[package]]\nname='packaging'\nversion='25.0'\nsource={registry='https://pypi.org/simple/'}\nwheels=[{url='https://files.pythonhosted.org/packages/packaging-25.0-py3-none-any.whl',hash='sha256:" + strings.Repeat("a", 64) + "',size=1}]\n[[package]]\nname='haa-empty-project'\nversion='0.1.0'\nsource={virtual='.'}\ndependencies=[{name='packaging'}]\n[package.metadata]\nrequires-dist=[{name='packaging',specifier='==25.0'}]\n")
	}
	controls := []domain.ProjectControlFile{}
	for name, body := range map[string][]byte{"pyproject.toml": manifest, "uv.lock": lock} {
		c, err := domain.NewProjectControlFile(name, body, true)
		if err != nil {
			t.Fatal(err)
		}
		controls = append(controls, c)
	}
	return uvProjectTransactionFixture{tx, root, controls, originals}
}

func (f uvProjectTransactionFixture) assertOriginals(t *testing.T) {
	t.Helper()
	for path, s := range f.originals {
		if s == nil {
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("absent original changed", path, err)
			}
		} else if !s.matches(path) {
			t.Fatal("original bytes/inode/mode changed", path)
		}
	}
}

func TestUVProjectControlsShareCompleteVenvPublication(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(map[bool]string{false: "new selection", true: "empty retirement"}[empty], func(t *testing.T) {
			f := newUVProjectTransactionFixture(t, false, empty)
			var destinations []pypiDestination
			if !empty {
				destinations = venvDestinations(t, map[string]string{"site/packaging.py": "synthetic new code", "scripts/packaging-tool": "synthetic script"}, "packaging")
				for i := range destinations {
					destinations[i].Version = "25.0"
				}
			}
			if err := f.tx.commitCompleteProject(destinations, empty, f.controls); err != nil {
				t.Fatal(err)
			}
			for _, c := range f.controls {
				body, err := os.ReadFile(filepath.Join(f.root, c.Name()))
				if err != nil || !bytes.Equal(body, c.Body()) {
					t.Fatal("project selection changed", err)
				}
				if before := f.originals[filepath.Join(f.root, c.Name())]; before != nil {
					info, err := os.Stat(filepath.Join(f.root, c.Name()))
					if err != nil || info.Mode().Perm() != before.Mode.Perm() {
						t.Fatal("project control permissions changed", err)
					}
				}
			}
			for _, path := range []string{filepath.Join(f.tx.plan.site, "old.py"), filepath.Join(f.tx.plan.root, "bin/old-tool")} {
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("retired owner remains", path, err)
				}
			}
			if !f.originals[filepath.Join(f.tx.plan.site, "user.txt")].matches(filepath.Join(f.tx.plan.site, "user.txt")) {
				t.Fatal("user file changed")
			}
			state, present, err := f.tx.plan.readState()
			if err != nil || present == empty {
				t.Fatal("installed ownership presence", err)
			}
			if len(state.Files) != len(destinations) {
				t.Fatal("project controls were recorded as wheel ownership")
			}
			if err := f.tx.commitCompleteProject(destinations, empty, f.controls); err == nil {
				t.Fatal("second publication accepted")
			}
		})
	}
}

func TestUVProjectControlsRollBackWithWheelAndMetadataFailures(t *testing.T) {
	for _, locked := range []bool{false, true} {
		for _, phase := range []string{"before-mutation", "after-publish:project-control", "after-publish:site", "after-publish:scripts", "metadata", "sync", "reconcile"} {
			t.Run(phase+map[bool]string{false: "/absent lock", true: "/existing lock"}[locked], func(t *testing.T) {
				f := newUVProjectTransactionFixture(t, locked, false)
				cause := errors.New("synthetic commit failure")
				f.tx.fault = func(step string) error {
					if step == phase {
						return cause
					}
					return nil
				}
				destinations := venvDestinations(t, map[string]string{"site/packaging.py": "new code", "scripts/packaging-tool": "new script"}, "packaging")
				if err := f.tx.commitCompleteProject(destinations, false, f.controls); !errors.Is(err, cause) {
					t.Fatal("primary failure lost", err)
				}
				if f.tx.recovery {
					t.Fatal("ordinary fault failed to restore originals")
				}
				f.assertOriginals(t)
				for _, d := range destinations {
					bound, err := f.tx.plan.bind(d)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := os.Lstat(bound.Final); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("new wheel output remains", err)
					}
				}
				for _, pattern := range []string{".haa-uv-project-controls-*", ".heliopause-pypi-commit-*"} {
					paths, err := filepath.Glob(filepath.Join(f.tx.plan.root, pattern))
					if err != nil || len(paths) != 0 {
						t.Fatal("ordinary failure retained temporary state", paths, err)
					}
				}
			})
		}
	}
}

func TestUVProjectControlsRefuseConcurrentGuardAndSourceDrift(t *testing.T) {
	for _, change := range []string{"content", "inode", "mode", "lock appeared", "hardlink"} {
		t.Run(change, func(t *testing.T) {
			f := newUVProjectTransactionFixture(t, false, false)
			if _, err := acquireProjectTransactionGuard(f.root, uvProjectTransactionLock); err == nil {
				t.Fatal("concurrent project guard accepted")
			}
			path := filepath.Join(f.root, "pyproject.toml")
			var err error
			switch change {
			case "content":
				err = os.WriteFile(path, []byte("changed"), 0o600)
			case "inode":
				err = os.Rename(path, path+".old")
				if err == nil {
					err = os.WriteFile(path, f.tx.project.input.Body(), 0o600)
				}
			case "mode":
				err = os.Chmod(path, 0o400)
			case "lock appeared":
				err = os.WriteFile(filepath.Join(f.root, "uv.lock"), []byte("foreign"), 0o600)
			case "hardlink":
				err = os.Link(path, path+".alias")
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := f.tx.commitCompleteProject(venvDestinations(t, map[string]string{"site/packaging.py": "new"}, "packaging"), false, f.controls); err == nil {
				t.Fatal("source drift accepted")
			}
			if !f.originals[filepath.Join(f.tx.plan.site, "old.py")].matches(filepath.Join(f.tx.plan.site, "old.py")) {
				t.Fatal("source drift changed installed output")
			}
		})
	}
}

func TestUVProjectControlsRetainUncertainPrivateFilesAndRecoveryJournal(t *testing.T) {
	for _, change := range []string{"foreign private file", "changed private file", "rollback failure"} {
		t.Run(change, func(t *testing.T) {
			f := newUVProjectTransactionFixture(t, false, false)
			cause := errors.New("synthetic primary failure")
			rollback := errors.New("synthetic rollback failure")
			f.tx.fault = func(step string) error {
				if change == "rollback failure" {
					if step == "after-publish:project-control" {
						return cause
					}
					if step == "rollback" {
						return rollback
					}
					return nil
				}
				if step == "before-mutation" {
					path := f.tx.project.selected[0].Source
					if change == "foreign private file" {
						path = filepath.Join(filepath.Dir(path), "foreign")
					}
					if err := os.WriteFile(path, []byte("preserve uncertain bytes"), 0o600); err != nil {
						t.Fatal(err)
					}
					return cause
				}
				return nil
			}
			err := f.tx.commitCompleteProject(venvDestinations(t, map[string]string{"site/packaging.py": "new"}, "packaging"), false, f.controls)
			if !errors.Is(err, cause) || !f.tx.recovery {
				t.Fatal("uncertain failure lost primary/recovery", err)
			}
			if change == "rollback failure" {
				if !errors.Is(err, rollback) {
					t.Fatal("rollback cause lost", err)
				}
				body, err := os.ReadFile(filepath.Join(f.tx.backup, "ledger.json"))
				if err != nil || !bytes.Contains(body, []byte(`"Project":{"Root":`)) || !bytes.Contains(body, []byte(`"project-control"`)) {
					t.Fatal("project recovery boundary missing", err)
				}
			} else {
				f.assertOriginals(t)
				path := f.tx.project.selected[0].Source
				if change == "foreign private file" {
					path = filepath.Join(filepath.Dir(path), "foreign")
				}
				if body, err := os.ReadFile(path); err != nil || string(body) != "preserve uncertain bytes" {
					t.Fatal("uncertain private file deleted", err)
				}
			}
			for _, path := range []string{f.tx.guard.path, f.tx.project.guard.path} {
				if _, err := os.Lstat(path); err != nil {
					t.Fatal("recovery guard missing", err)
				}
			}
		})
	}
}

func TestUVProjectControlsFreezeRefusesAmbiguousSources(t *testing.T) {
	for _, name := range []string{"nil context", "canceled", "noncanonical", "missing manifest", "symlink manifest", "hardlink manifest", "lock directory", "trusted overlap", "venv overlap", "repeated freeze"} {
		t.Run(name, func(t *testing.T) {
			_, plan := makePypiVenvFixture(t)
			tx, _ := heldCompleteTestTransaction(t, plan)
			root := realPromotionRoot(t)
			manifest, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.pyproject.toml")
			if err != nil {
				t.Fatal(err)
			}
			// Project the historical format onto the current runtime; keep captured testdata unchanged.
			manifest = bytes.ReplaceAll(manifest, []byte("3.14.7"), []byte(runtimeidentity.PythonVersion))
			path := filepath.Join(root, "pyproject.toml")
			if err := os.WriteFile(path, manifest, 0o600); err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			trusted := []string{realPromotionRoot(t)}
			switch name {
			case "nil context":
				ctx = nil
			case "canceled":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			case "noncanonical":
				root += "/."
			case "missing manifest":
				err = os.Remove(path)
			case "symlink manifest":
				err = os.Rename(path, path+".original")
				if err == nil {
					err = os.Symlink(path+".original", path)
				}
			case "hardlink manifest":
				err = os.Link(path, path+".alias")
			case "lock directory":
				err = os.Mkdir(filepath.Join(root, "uv.lock"), 0o700)
			case "trusted overlap":
				trusted = []string{root}
			case "venv overlap":
				root = plan.root
				err = os.WriteFile(filepath.Join(root, "pyproject.toml"), manifest, 0o600)
			case "repeated freeze":
				_, err = freezeUVProjectFiles(ctx, tx, root, artifactpypi.UVProjectSync, "", trusted)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := freezeUVProjectFiles(ctx, tx, root, artifactpypi.UVProjectSync, "", trusted); err == nil {
				t.Fatal("ambiguous project source accepted")
			}
			if name != "repeated freeze" {
				if _, err := os.Lstat(filepath.Join(root, uvProjectTransactionLock)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("rejected source retained project lock", err)
				}
			}
		})
	}
}

func TestUVProjectControlsRequireExplicitProjectCommit(t *testing.T) {
	f := newUVProjectTransactionFixture(t, false, true)
	if err := f.tx.commitComplete(nil, true); err == nil {
		t.Fatal("held project used venv-only publication")
	}
	f.assertOriginals(t)
}

func TestUVProjectControlsCleanupFailureStillRestoresOriginals(t *testing.T) {
	f := newUVProjectTransactionFixture(t, false, false)
	foreign := ""
	f.tx.fault = func(step string) error {
		if step == "reconcile" {
			foreign = filepath.Join(filepath.Dir(f.tx.project.selected[0].Source), "foreign-after-publication")
			if err := os.WriteFile(foreign, []byte("preserve late foreign bytes"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}
	if err := f.tx.commitCompleteProject(venvDestinations(t, map[string]string{"site/packaging.py": "new code"}, "packaging"), false, f.controls); err == nil || !f.tx.recovery {
		t.Fatal("late cleanup uncertainty accepted", err)
	}
	f.assertOriginals(t)
	if body, err := os.ReadFile(foreign); err != nil || string(body) != "preserve late foreign bytes" {
		t.Fatal("late foreign file deleted", err)
	}
}
