//go:build linux && amd64

package promotion

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestVenvMkdirJournalPreservesInterruptedPublication(t *testing.T) {
	const rootEnv = "HAA_TEST_VENV_MKDIR_JOURNAL_ROOT"
	const stepEnv = "HAA_TEST_VENV_MKDIR_JOURNAL_STEP"

	if root := os.Getenv(rootEnv); root != "" {
		plan, err := discoverPythonVenv(root)
		if err != nil {
			t.Fatal(err)
		}
		tx, err := beginPyPIVenvTransaction(plan)
		if err != nil {
			t.Fatal(err)
		}
		tx.fault = func(step string) error {
			if step == os.Getenv(stepEnv) {
				return syscall.Kill(os.Getpid(), syscall.SIGKILL)
			}
			return nil
		}
		t.Fatal("expected SIGKILL", tx.commit(venvDestinations(t, map[string]string{
			"site/a.py":        "first",
			"site/nested/b.py": "second",
		}, "new")))
	}

	for _, tc := range []struct {
		name, step string
		mkdir      bool
	}{
		{"after-file-before-mkdir", "after-publish:site", false},
		{"after-mkdir-before-next-file", "before-publish:site/nested/b.py", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, plan := makePypiVenvFixture(t)
			runVenvFixture(t, plan, map[string]string{"site/keep.py": "original"}, "retained")
			metadata := filepath.Join(root, pypiVenvMetadata)
			original, err := os.ReadFile(metadata)
			if err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(os.Args[0], "-test.run=^TestVenvMkdirJournalPreservesInterruptedPublication$")
			cmd.Env = append(os.Environ(), rootEnv+"="+root, stepEnv+"="+tc.step)
			out, err := cmd.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("expected interrupted child: %v, %s", err, out)
			}
			status, ok := exit.Sys().(syscall.WaitStatus)
			if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
				t.Fatalf("expected SIGKILL: %v, %s", err, out)
			}

			paths, err := filepath.Glob(filepath.Join(root, ".heliopause-pypi-commit-*", "ledger.json"))
			if err != nil || len(paths) != 1 {
				t.Fatalf("missing interrupted journal: %v %v", paths, err)
			}
			raw, err := os.ReadFile(paths[0])
			if err != nil {
				t.Fatal(err)
			}
			lines := bytes.Split(bytes.TrimSuffix(raw, []byte("\n")), []byte("\n"))
			var last []struct {
				Action, Path, Created string
				Published             *pypiDestination
			}
			for _, line := range lines[1:] {
				var record struct {
					Action, Path, Created string
					Published             *pypiDestination
				}
				if err := json.Unmarshal(line, &record); err != nil {
					t.Fatal(err)
				}
				last = append(last, record)
			}
			if len(last) == 0 {
				t.Fatal("missing first publication intent")
			}
			dir := filepath.Join(plan.site, "nested")
			first := filepath.Join(plan.site, "a.py")
			if tc.mkdir {
				if len(last) < 4 {
					t.Fatalf("incomplete mkdir journal: %+v", last)
				}
				tail := last[len(last)-4:]
				if tail[0].Action != "publish" || tail[0].Path != first ||
					tail[1].Published == nil || tail[1].Published.key() != "site/a.py" ||
					tail[2].Action != "mkdir" || tail[2].Path != dir || tail[3].Created != dir {
					t.Fatalf("incorrect completion and next intent order: %+v", tail)
				}
				if info, err := os.Stat(dir); err != nil || !info.IsDir() {
					t.Fatalf("missing created directory: %v", err)
				}
			} else {
				tail := last[len(last)-1]
				if tail.Action != "publish" || tail.Path != first {
					t.Fatalf("unexpected terminal journal record: %+v", tail)
				}
				if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("next directory existed before its intent: %v", err)
				}
			}

			assertRestartRejected := func(reason string) {
				t.Helper()
				stopped, err := beginPyPIVenvTransaction(plan)
				if err == nil {
					_ = stopped.close()
					t.Fatal("interrupted transaction was reopened")
				}
				if !strings.Contains(err.Error(), reason) {
					t.Fatalf("unexpected restart error: %v", err)
				}
			}
			assertRestartRejected("virtual environment is already being mutated")
			// The child is dead and this fixture owns the lock. Move it aside
			// in the fixture to exercise the second, independent restart guard.
			guard := filepath.Join(root, ".heliopause-pypi-transaction.lock")
			if err := os.Rename(guard, guard+".fixture"); err != nil {
				t.Fatal(err)
			}
			assertRestartRejected("interrupted transaction requires explicit recovery")
			for path, want := range map[string][]byte{
				paths[0]: raw, metadata: original, first: []byte("first"),
				filepath.Join(plan.site, "keep.py"): []byte("original"),
			} {
				body, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(body, want) {
					t.Fatalf("restart modified %s: %v", path, err)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "b.py")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("later file published across interruption: %v", err)
			}
		})
	}
}
