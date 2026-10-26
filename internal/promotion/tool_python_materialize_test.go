package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type toolMaterializerRunner struct {
	call  func(string, []string) error
	calls int
}

func (r *toolMaterializerRunner) Run(_ context.Context, project string, arguments []string) error {
	r.calls++
	return r.call(project, arguments)
}

func toolMaterializerWorkspace(t *testing.T) string {
	t.Helper()
	parent := realPromotionRoot(t)
	project := filepath.Join(parent, "project")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	return project
}

func toolMaterializerView(t *testing.T, project string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(project), toolPythonViewName+"*"))
	if err != nil || len(matches) != 1 {
		t.Fatal("one controlled sibling interpreter view required")
	}
	return matches[0]
}

func toolMaterializerContext(t *testing.T, target string) domain.InstallContext {
	t.Helper()
	value, err := domain.NewInstallTarget(target)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewPythonVenvInstallContext(value)
	if err != nil {
		t.Fatal(err)
	}
	return install
}

func TestToolMaterializerRetainsOfflineRecipeAndBindsFinalInterpreter(t *testing.T) {
	project := toolMaterializerWorkspace(t)
	policy := artifactpypi.ResourcePolicyFromContext(context.Background())
	for _, target := range []string{"/tools/cowsay", "/tools/env with spaces"} {
		got, err := toolPythonPromotionArguments(project, filepath.Join(filepath.Dir(project), toolPythonViewName+"fixture"), target, policy)
		if err != nil {
			t.Fatal(err)
		}
		base := pypiPromotionArguments(project, policy)
		image := sandbox.PinnedPythonRuntime().ImageReference
		index := -1
		for i, value := range base {
			if value == image {
				index = i
			}
		}
		if index < 0 {
			t.Fatal("fixed image missing")
		}
		want := append([]string(nil), base[:index]...)
		want = append(want, "--env", "PYTHONDONTWRITEBYTECODE=1", "--mount", "type=bind,src="+filepath.Join(filepath.Dir(project), toolPythonViewName+"fixture")+",dst="+target)
		want = append(want, image, "-I", "-B", "-m", "pip", "--python", target+"/bin/python")
		want = append(want, base[index+4:]...)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("offline recipe changed: %q", got)
		}
	}
}

func TestToolMaterializerRejectsUnsafeGuestMountBeforeExecution(t *testing.T) {
	for _, target := range []string{"/usr", "/usr/local/bin/env", "/workspace/tool", "/proc", "/sys/env", "/dev", "/tmp", "/etc/tool", "/haa-runtime/tool", "/tools/a,b", "/tools/a:b", "/tools/a\\b"} {
		t.Run(strings.ReplaceAll(target, "/", "_"), func(t *testing.T) {
			project := toolMaterializerWorkspace(t)
			runner := &toolMaterializerRunner{call: func(string, []string) error { t.Fatal("unsafe target reached runner"); return nil }}
			p := &PyPIPromotion{runner: runner, goos: "linux", goarch: "amd64"}
			install := toolMaterializerContext(t, target)
			result, err := p.materializeToolPythonProject(context.Background(), project, nil, nil, install)
			if err == nil || result != nil || runner.calls != 0 {
				t.Fatal("unsafe mount accepted")
			}
			if matches, err := filepath.Glob(filepath.Join(filepath.Dir(project), toolPythonViewName+"*")); err != nil || len(matches) != 0 {
				t.Fatal("unsafe input created interpreter view")
			}
		})
	}
}

func TestToolMaterializerUsesWholeRecordValidatorAndClearsSuccessfulResultOnDrift(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		project := toolMaterializerWorkspace(t)
		runner := &toolMaterializerRunner{call: func(root string, _ []string) error {
			fixture, _, _ := validPromotionOutputFixture(t)
			// The production runner returns pip's unrelocated --target schemes.
			if err := os.Rename(filepath.Join(fixture, "site"), filepath.Join(root, "site")); err != nil {
				t.Fatal(err)
			}
			for _, scheme := range []string{"bin", "share"} {
				if err := os.Rename(filepath.Join(fixture, scheme), filepath.Join(root, "site", scheme)); err != nil {
					t.Fatal(err)
				}
			}
			if foreign {
				if err := os.WriteFile(filepath.Join(toolMaterializerView(t, root), "foreign"), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			return nil
		}}
		p := &PyPIPromotion{runner: runner, goos: "linux", goarch: "amd64"}
		result, err := p.materializeToolPythonProject(context.Background(), project, []byte("fixture==1.0.0\n"), fixtureExpected(), toolMaterializerContext(t, "/tools/env"))
		if foreign {
			if err == nil || result != nil {
				t.Fatal("successful output survived cleanup failure")
			}
			if body, readErr := os.ReadFile(filepath.Join(toolMaterializerView(t, project), "foreign")); readErr != nil || string(body) != "keep" {
				t.Fatal("foreign bytes not retained")
			}
		} else if err != nil || len(result) != 5 {
			t.Fatalf("whole validated output missing: destinations=%d error=%v", len(result), err)
		}
	}
}

func TestToolPythonViewCleanupPreservesChangedOrForeignData(t *testing.T) {
	for _, mutation := range []string{"none", "foreign", "config", "alias", "root"} {
		t.Run(mutation, func(t *testing.T) {
			parent := realPromotionRoot(t)
			view, err := prepareToolPythonView(parent)
			if err != nil {
				t.Fatal(err)
			}
			if err := view.verify(); err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "foreign":
				err = os.WriteFile(filepath.Join(view.path, "foreign"), []byte("keep"), 0600)
			case "config":
				err = os.WriteFile(filepath.Join(view.path, "pyvenv.cfg"), []byte("foreign config"), 0600)
			case "alias":
				err = os.Remove(filepath.Join(view.path, "bin/python"))
				if err == nil {
					err = os.Symlink("/foreign/python", filepath.Join(view.path, "bin/python"))
				}
			case "root":
				err = os.Rename(view.path, view.path+"-saved")
				if err == nil {
					err = os.Mkdir(view.path, 0700)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			err = view.close()
			if mutation == "none" {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := os.Lstat(view.path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("owned view retained")
				}
			} else {
				if err == nil {
					t.Fatal("changed view cleaned")
				}
				if _, err := os.Lstat(view.path); err != nil {
					t.Fatal("changed root removed")
				}
				if mutation == "foreign" {
					body, err := os.ReadFile(filepath.Join(view.path, "foreign"))
					if err != nil || string(body) != "keep" {
						t.Fatal("foreign bytes lost")
					}
				}
				if mutation == "config" {
					body, err := os.ReadFile(filepath.Join(view.path, "pyvenv.cfg"))
					if err != nil || string(body) != "foreign config" {
						t.Fatal("changed config lost")
					}
				}
				if mutation == "alias" {
					value, err := os.Readlink(filepath.Join(view.path, "bin/python"))
					if err != nil || value != "/foreign/python" {
						t.Fatal("foreign alias lost")
					}
				}
				if mutation == "root" {
					if _, err := os.Lstat(view.path + "-saved/pyvenv.cfg"); err != nil {
						t.Fatal("original root lost")
					}
				}
			}
		})
	}
}

func TestToolMaterializerPreservesPrimaryFailureAndRejectsCleanupDrift(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		project := toolMaterializerWorkspace(t)
		primary := errors.New("materializer stopped")
		runner := &toolMaterializerRunner{call: func(root string, _ []string) error {
			if foreign {
				if err := os.WriteFile(filepath.Join(toolMaterializerView(t, root), "foreign"), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			return primary
		}}
		p := &PyPIPromotion{runner: runner, goos: "linux", goarch: "amd64"}
		result, err := p.materializeToolPythonProject(context.Background(), project, nil, nil, toolMaterializerContext(t, "/tools/env with spaces"))
		if !errors.Is(err, primary) || result != nil || runner.calls != 1 {
			t.Fatal("primary error or result discarded incorrectly")
		}
		matches, statErr := filepath.Glob(filepath.Join(filepath.Dir(project), toolPythonViewName+"*"))
		if foreign {
			if statErr != nil || len(matches) != 1 || !strings.Contains(err.Error(), "foreign") {
				t.Fatal("cleanup drift not preserved")
			}
		} else if statErr != nil || len(matches) != 0 {
			t.Fatal("owned interpreter view leaked")
		}
	}
}

func TestToolMaterializerCancellationHasNoSideEffects(t *testing.T) {
	project := toolMaterializerWorkspace(t)
	runner := &toolMaterializerRunner{call: func(string, []string) error { t.Fatal("cancelled operation executed"); return nil }}
	p := &PyPIPromotion{runner: runner, goos: "linux", goarch: "amd64"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := p.materializeToolPythonProject(ctx, project, nil, nil, toolMaterializerContext(t, "/tools/env"))
	if !errors.Is(err, context.Canceled) || result != nil || runner.calls != 0 {
		t.Fatal("cancelled materialization accepted")
	}
	if matches, err := filepath.Glob(filepath.Join(filepath.Dir(project), toolPythonViewName+"*")); err != nil || len(matches) != 0 {
		t.Fatal("cancelled operation created interpreter view")
	}
}
