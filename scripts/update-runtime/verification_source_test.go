package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestVerificationProjectsFrozenWorktreeWithStandaloneGit(t *testing.T) {
	root, state := fixture(t)
	if err := os.Chmod(filepath.Join(root, "tools/gvisor-observer/BUILD"), 0o640); err != nil {
		t.Fatal(err)
	}
	p := futurePlan(t, root, state, "go")
	if err := applyPlan(p, state, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(state, "candidate")
	info, err := os.Lstat(filepath.Join(candidate, ".git"))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatal("fixture is not a linked worktree")
	}
	attempt, err := os.MkdirTemp(state, "verification-")
	if err != nil {
		t.Fatal(err)
	}
	projection, err := verificationSource(p, candidate, attempt)
	if err != nil {
		t.Fatal(err)
	}
	info, err = os.Lstat(filepath.Join(projection, ".git"))
	if err != nil || !info.IsDir() {
		t.Fatal("Go requires a standalone Git directory")
	}
	if err := checkCandidate(p, projection, true); err != nil {
		t.Fatal(err)
	}
	if err := compareSnapshot(p); err != nil {
		t.Fatal(err)
	}
	// A changed check input is never accepted as the frozen candidate.
	writeTest(t, filepath.Join(projection, "go.mod"), []byte("changed projection"))
	if err := checkCandidate(p, projection, true); err == nil {
		t.Fatal("changed checked source accepted")
	}
	if err := checkCandidate(p, candidate, true); err != nil {
		t.Fatal("projection changed authoritative candidate", err)
	}
}

func TestVerificationPreservesForeignCandidateEdit(t *testing.T) {
	root, state := fixture(t)
	p := noopPlan(t, root, state, "go")
	if err := applyPlan(p, state, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(state, "candidate")
	writeTest(t, filepath.Join(candidate, "go.mod"), []byte("foreign edit"))
	attempt, err := os.MkdirTemp(state, "verification-")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verificationSource(p, candidate, attempt); err == nil {
		t.Fatal("foreign edit accepted")
	}
	body, err := os.ReadFile(filepath.Join(candidate, "go.mod"))
	if err != nil || string(body) != "foreign edit" {
		t.Fatal("foreign edit was overwritten")
	}
	if _, err := os.Lstat(filepath.Join(attempt, "source")); !os.IsNotExist(err) {
		t.Fatal("unvalidated projection was created")
	}
}
