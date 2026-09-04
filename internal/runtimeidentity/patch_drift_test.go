package runtimeidentity_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func getRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Walk up until go.mod is found
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root containing go.mod")
		}
		dir = parent
	}
}

// A. exact patch accepted
func TestPatchDrift_A_ExactPatchAccepted(t *testing.T) {
	t.Parallel()
	repoRoot := getRepoRoot(t)
	err := runtimeidentity.ValidatePatchFile(repoRoot, runtimeidentity.GVisorPatchPath, runtimeidentity.GVisorPatchSHA256)
	if err != nil {
		t.Fatalf("expected exact patch to be accepted, got: %v", err)
	}
}

// B. one-byte patch mutation rejected
func TestPatchDrift_B_OneBytePatchMutationRejected(t *testing.T) {
	t.Parallel()
	repoRoot := getRepoRoot(t)
	origBytes, err := os.ReadFile(filepath.Join(repoRoot, runtimeidentity.GVisorPatchPath))
	if err != nil {
		t.Fatal(err)
	}

	tempRoot := t.TempDir()
	targetDir := filepath.Join(tempRoot, "tools", "gvisor")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	mutated := make([]byte, len(origBytes))
	copy(mutated, origBytes)
	// Mutate one byte
	if len(mutated) > 10 {
		mutated[10] ^= 0xff
	}

	mutatedPath := filepath.Join(targetDir, filepath.Base(runtimeidentity.GVisorPatchPath))
	if err := os.WriteFile(mutatedPath, mutated, 0o644); err != nil {
		t.Fatal(err)
	}

	err = runtimeidentity.ValidatePatchFile(tempRoot, runtimeidentity.GVisorPatchPath, runtimeidentity.GVisorPatchSHA256)
	if err == nil {
		t.Fatal("expected one-byte patch mutation to be rejected, but it was accepted")
	}
	if !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch error, got: %v", err)
	}
}

// C. missing patch rejected
func TestPatchDrift_C_MissingPatchRejected(t *testing.T) {
	t.Parallel()
	repoRoot := getRepoRoot(t)
	err := runtimeidentity.ValidatePatchFile(repoRoot, "tools/gvisor/nonexistent.patch", runtimeidentity.GVisorPatchSHA256)
	if err == nil {
		t.Fatal("expected missing patch to be rejected, but it was accepted")
	}
}

// D. wrong digest rejected
func TestPatchDrift_D_WrongDigestRejected(t *testing.T) {
	t.Parallel()
	repoRoot := getRepoRoot(t)
	wrongDigest := strings.Repeat("0", 64)
	err := runtimeidentity.ValidatePatchFile(repoRoot, runtimeidentity.GVisorPatchPath, wrongDigest)
	if err == nil {
		t.Fatal("expected wrong digest to be rejected, but it was accepted")
	}
	if !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch error, got: %v", err)
	}
}

// E. wrong upstream commit rejected
func TestPatchDrift_E_WrongUpstreamCommitRejected(t *testing.T) {
	t.Parallel()
	repoRoot := getRepoRoot(t)
	patchAbs := filepath.Join(repoRoot, runtimeidentity.GVisorPatchPath)

	tempGit := t.TempDir()
	initGitRepo(t, tempGit)

	// HEAD is an initial commit that does not match GVisorCommit
	err := runtimeidentity.ApplyPatchDeterministic(tempGit, patchAbs, runtimeidentity.GVisorCommit)
	if err == nil {
		t.Fatal("expected wrong upstream commit to be rejected, but it was accepted")
	}
	if !strings.Contains(err.Error(), "source commit mismatch") && !strings.Contains(err.Error(), "wrong commit") {
		t.Fatalf("expected commit mismatch error, got: %v", err)
	}
}

// F. patch apply failure rejected
func TestPatchDrift_F_PatchApplyFailureRejected(t *testing.T) {
	t.Parallel()
	repoRoot := getRepoRoot(t)
	patchAbs := filepath.Join(repoRoot, runtimeidentity.GVisorPatchPath)

	tempGit := t.TempDir()
	initGitRepo(t, tempGit)
	// Force the commit ID to match GVisorCommit in a synthetic branch or git tag,
	// but the contents are incompatible
	commit := getHeadCommit(t, tempGit)

	// Even if commit matches, incompatible files cause git apply --check to fail
	err := runtimeidentity.ApplyPatchDeterministic(tempGit, patchAbs, commit)
	if err == nil {
		t.Fatal("expected patch apply failure to be rejected, but it was accepted")
	}
	if !strings.Contains(err.Error(), "apply") {
		t.Fatalf("expected apply failure error, got: %v", err)
	}
}

// G. dirty unexpected source rejected where relevant
func TestPatchDrift_G_DirtyUnexpectedSourceRejected(t *testing.T) {
	t.Parallel()
	repoRoot := getRepoRoot(t)
	patchAbs := filepath.Join(repoRoot, runtimeidentity.GVisorPatchPath)

	tempGit := t.TempDir()
	initGitRepo(t, tempGit)
	commit := getHeadCommit(t, tempGit)

	// Create an unexpected dirty untracked / modified file
	if err := os.WriteFile(filepath.Join(tempGit, "unexpected_dirty.txt"), []byte("dirty content"), 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := runtimeidentity.InspectSourceTree(tempGit, patchAbs, commit)
	if state != runtimeidentity.SourceTreeStateUnexpectedDirty || err == nil {
		t.Fatalf("expected SourceTreeStateUnexpectedDirty, got state=%v err=%v", state, err)
	}
	applyErr := runtimeidentity.ApplyPatchDeterministic(tempGit, patchAbs, commit)
	if applyErr == nil {
		t.Fatal("expected dirty unexpected source to be rejected by ApplyPatchDeterministic, but it was accepted")
	}
}

// H. unpatched runtime cannot satisfy required patched capability
func TestPatchDrift_H_UnpatchedRuntimeCannotSatisfyCapability(t *testing.T) {
	t.Parallel()
	// Unpatched gVisor trace metadata (like the one dumped earlier from stock /usr/libexec/heliopause/runsc)
	stockTraceMetadata := `
Name: sentry/clone, optional fields: []
Name: sentry/execve, optional fields: [binary_info]
Name: sentry/exit_notify_parent, optional fields: []
Name: sentry/task_exit, optional fields: []
Name: syscall/open/enter, optional fields: []
Name: syscall/openat/enter, optional fields: []
`
	err := sandbox.VerifyPatchCapability(stockTraceMetadata)
	if err == nil {
		t.Fatal("expected stock unpatched trace metadata to fail capability verification, but it passed")
	}
}

// I. exact patched runtime satisfies capability probe
func TestPatchDrift_I_ExactPatchedRuntimeSatisfiesCapability(t *testing.T) {
	t.Parallel()
	patchedTraceMetadata := `
Name: sentry/clone, optional fields: []
Name: sentry/mount_topology_mutation, optional fields: []
Name: sentry/mount_topology_snapshot, optional fields: []
Name: syscall/open_result, optional fields: []
Name: sentry/task_exit, optional fields: []
`
	err := sandbox.VerifyPatchCapability(patchedTraceMetadata)
	if err != nil {
		t.Fatalf("expected patched trace metadata to satisfy capability verification, got: %v", err)
	}

	// Also verify against the real candidate runsc binary if available on the system
	candidateRunsc := "/tmp/m12-runtime-root.F9NzjR/gvisor/bazel-bin/runsc/runsc_/runsc"
	if _, err := os.Stat(candidateRunsc); err == nil {
		out, err := exec.CommandContext(context.Background(), candidateRunsc, "trace", "metadata").Output()
		if err != nil {
			t.Fatalf("candidate runsc trace metadata failed: %v", err)
		}
		if err := sandbox.VerifyPatchCapability(string(out)); err != nil {
			t.Fatalf("candidate runsc binary failed capability verification: %v", err)
		}
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "test")
	runGit(t, dir, "config", "user.email", "test@test.com")
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# Test Repo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial commit")
}

func getHeadCommit(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v, output: %s", strings.Join(args, " "), err, string(out))
	}
}
