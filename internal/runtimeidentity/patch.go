package runtimeidentity

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PatchIntegrityError describes a failure to validate patch authenticity or identity.
type PatchIntegrityError struct {
	Reason string
	Cause  error
}

func (e *PatchIntegrityError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("patch integrity error: %s: %v", e.Reason, e.Cause)
	}
	return fmt.Sprintf("patch integrity error: %s", e.Reason)
}

// ValidatePatchFile verifies that the patch file exists within the expected
// repository boundary and matches the expected SHA-256 checksum.
func ValidatePatchFile(repoRoot, patchRelPath, expectedSHA256 string) error {
	if patchRelPath == "" {
		return &PatchIntegrityError{Reason: "missing patch path"}
	}
	if strings.Contains(patchRelPath, "..") || !strings.HasPrefix(patchRelPath, "tools/gvisor/") || !strings.HasSuffix(patchRelPath, ".patch") {
		return &PatchIntegrityError{Reason: fmt.Sprintf("patch path %q escapes tools/gvisor/ boundary", patchRelPath)}
	}
	if len(expectedSHA256) != 64 {
		return &PatchIntegrityError{Reason: "invalid expected patch sha256"}
	}
	fullPath := filepath.Join(repoRoot, filepath.FromSlash(patchRelPath))
	info, err := os.Stat(fullPath)
	if err != nil {
		return &PatchIntegrityError{Reason: "patch file missing or inaccessible", Cause: err}
	}
	if !info.Mode().IsRegular() {
		return &PatchIntegrityError{Reason: "patch path is not a regular file"}
	}
	file, err := os.Open(fullPath)
	if err != nil {
		return &PatchIntegrityError{Reason: "cannot open patch file", Cause: err}
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return &PatchIntegrityError{Reason: "cannot read patch file", Cause: err}
	}
	actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
	if actualSHA256 != strings.ToLower(expectedSHA256) {
		return &PatchIntegrityError{Reason: fmt.Sprintf("patch digest mismatch: expected %s, got %s", expectedSHA256, actualSHA256)}
	}
	return nil
}

// SourceTreeState represents the clean, applied, or dirty state of a gVisor tree.
type SourceTreeState int

const (
	SourceTreeStateCleanUnpatched SourceTreeState = iota
	SourceTreeStateAlreadyPatched
	SourceTreeStateUnexpectedDirty
	SourceTreeStateWrongCommit
)

// InspectSourceTree inspects a gVisor working tree against the pinned commit and patch.
func InspectSourceTree(gvisorDir, patchAbsPath, expectedCommit string) (SourceTreeState, error) {
	cmd := exec.Command("git", "-C", gvisorDir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return SourceTreeStateUnexpectedDirty, fmt.Errorf("git rev-parse HEAD failed: %w", err)
	}
	actualCommit := strings.TrimSpace(string(out))
	if actualCommit != expectedCommit {
		return SourceTreeStateWrongCommit, fmt.Errorf("source commit mismatch: expected %s, got %s", expectedCommit, actualCommit)
	}

	cmd = exec.Command("git", "-C", gvisorDir, "status", "--porcelain")
	out, err = cmd.Output()
	if err != nil {
		return SourceTreeStateUnexpectedDirty, fmt.Errorf("git status failed: %w", err)
	}
	status := strings.TrimSpace(string(out))
	if status == "" {
		return SourceTreeStateCleanUnpatched, nil
	}

	// Test if already patched: reversing patch must succeed and leave tree matching clean state
	cmd = exec.Command("git", "-C", gvisorDir, "apply", "--check", "--reverse", patchAbsPath)
	if err := cmd.Run(); err == nil {
		return SourceTreeStateAlreadyPatched, nil
	}

	return SourceTreeStateUnexpectedDirty, errors.New("gVisor source tree is dirty with unexpected modifications")
}

// ApplyPatchDeterministic applies the verified patch to the gVisor working tree,
// failing closed if the tree is not cleanly at expectedCommit or if apply fails.
func ApplyPatchDeterministic(gvisorDir, patchAbsPath, expectedCommit string) error {
	state, err := InspectSourceTree(gvisorDir, patchAbsPath, expectedCommit)
	if err != nil {
		return fmt.Errorf("source tree inspection failed: %w", err)
	}
	switch state {
	case SourceTreeStateAlreadyPatched:
		return nil
	case SourceTreeStateCleanUnpatched:
		var stderr bytes.Buffer
		cmd := exec.Command("git", "-C", gvisorDir, "apply", "--check", patchAbsPath)
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git apply --check failed: %w (stderr: %s)", err, stderr.String())
		}
		cmd = exec.Command("git", "-C", gvisorDir, "apply", patchAbsPath)
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git apply failed: %w (stderr: %s)", err, stderr.String())
		}
		return nil
	case SourceTreeStateWrongCommit:
		return fmt.Errorf("refusing to patch: source tree is at wrong commit")
	case SourceTreeStateUnexpectedDirty:
		fallthrough
	default:
		return fmt.Errorf("refusing to patch: source tree has unexpected dirty state")
	}
}
