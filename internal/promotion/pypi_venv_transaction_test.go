package promotion

import (
	"os"
	"path/filepath"
	"testing"
)

func makePypiVenvFixture(t *testing.T) (string, pypiVenvPlan) {
	t.Helper()
	root := realPromotionRoot(t)
	if err := os.WriteFile(filepath.Join(root, "pyvenv.cfg"), []byte("version = 3.14.7\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	site := filepath.Join(root, "lib", "python3.14", "site-packages")
	if err := os.MkdirAll(site, 0o700); err != nil {
		t.Fatal(err)
	}
	plan, err := discoverPythonVenv(root)
	if err != nil {
		t.Fatal(err)
	}
	return root, plan
}

func TestDiscoverPythonVenvRequiresPinnedMinorAndLayout(t *testing.T) {
	_, plan := makePypiVenvFixture(t)
	if plan.site == "" {
		t.Fatal("site-packages was not discovered")
	}
}

func TestPypiVenvCommitTracksOnlyHAAOwnedFiles(t *testing.T) {
	root, plan := makePypiVenvFixture(t)
	output := filepath.Join(root, "output")
	if err := os.MkdirAll(filepath.Join(output, "demo"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "demo", "__init__.py"), []byte("v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	desired, err := plan.outputState(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.commit(output, desired); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(filepath.Join(plan.site, "demo", "__init__.py")); err != nil || string(body) != "v1\n" {
		t.Fatalf("committed file=%q error=%v", body, err)
	}
	state, managed, err := plan.readState()
	if err != nil || !managed || len(state.Files) != 1 {
		t.Fatalf("state=%#v managed=%v error=%v", state, managed, err)
	}
	if err := plan.verifyState(state); err != nil {
		t.Fatal(err)
	}
}

func TestPypiVenvRejectsBaselineCollisionAndRollsBackPartialPublish(t *testing.T) {
	_, plan := makePypiVenvFixture(t)
	if err := os.MkdirAll(filepath.Join(plan.site, "baseline"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plan.site, "baseline", "__init__.py"), []byte("host\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(plan.root, "output")
	if err := os.MkdirAll(filepath.Join(output, "baseline"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "baseline", "__init__.py"), []byte("replace\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	desired, err := plan.outputState(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.commit(output, desired); err == nil {
		t.Fatal("baseline collision was accepted")
	}
	body, err := os.ReadFile(filepath.Join(plan.site, "baseline", "__init__.py"))
	if err != nil || string(body) != "host\n" {
		t.Fatalf("baseline changed: %q error=%v", body, err)
	}
}

func TestPypiVenvRejectsInterruptedTransaction(t *testing.T) {
	root, plan := makePypiVenvFixture(t)
	if err := os.Mkdir(filepath.Join(root, ".heliopause-pypi-commit-stale"), 0o700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "output")
	if err := os.Mkdir(output, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "new.py"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	desired, err := plan.outputState(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.commit(output, desired); err == nil {
		t.Fatal("interrupted transaction was reused")
	}
}
