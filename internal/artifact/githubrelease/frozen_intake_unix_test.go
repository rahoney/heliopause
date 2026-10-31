//go:build linux || darwin

package githubrelease

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectiongithub "github.com/rahoney/heliopause/internal/inspection/githubrelease"
	"github.com/rahoney/heliopause/internal/policy"
	verificationgithub "github.com/rahoney/heliopause/internal/verification/githubrelease"
)

func privateFrozenRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func acquiredFrozenFixture(t *testing.T, root, name string, body []byte) (FrozenEntry, string) {
	t.Helper()
	selector, err := ParseSelector("owner/repo@v1.2.3#" + name)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.NewSHA256Digest(fmt.Sprintf("%x", sha256.Sum256(body)))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := resolvedArtifactForAsset(mustSource(t), ResolvedAsset{selector: selector, releaseID: 41, assetID: 7, digest: digest, size: uint64(len(body))})
	if err != nil {
		t.Fatal(err)
	}
	reference, err := ParseReference(selector.Locator())
	if err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, run.String())
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(directory, "asset")
	if err := os.WriteFile(filename, body, 0600); err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), digest, "intake:"+run.String()+":"+SourceName, uint64(len(body)), resolved.DeclaredIntegrity())
	if err != nil {
		t.Fatal(err)
	}
	return FrozenEntry{Reference: reference, Resolved: resolved, Acquired: artifact}, filename
}

func TestFrozenIntakeFeedsExistingInspectWorkflow(t *testing.T) {
	root := privateFrozenRoot(t)
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	file, err := writer.Create("README.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("controlled archive fixture")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	entry, original := acquiredFrozenFixture(t, root, "package.zip", archive.Bytes())
	frozen, err := NewFrozenIntake(context.Background(), root, []FrozenEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	defer frozen.Close()
	inspector, err := inspectiongithub.NewStaticInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := local.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewInspectService(frozen, verificationgithub.IntegrityVerifier{}, inspector, evidence, policy.M6{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewInspectRequest(entry.Reference)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Inspect(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	decision, ok := result.PolicyDecision()
	if result.Status() != domain.OperationCompleted || !ok || decision.Decision() != domain.DecisionAllow || len(result.Checks()) != 2 || len(result.Evidence()) != 2 {
		t.Fatalf("existing archive inspection did not complete: %+v", result)
	}
	run, ok := result.RunID()
	if !ok {
		t.Fatal("new inspection Run missing")
	}
	originalRun := strings.Split(entry.Acquired.ContentHandle(), ":")[1]
	if run.String() == originalRun {
		t.Fatal("reused acquisition Run as a new inspection Run")
	}
	copied, err := os.ReadFile(filepath.Join(root, run.String(), "asset"))
	if err != nil || !bytes.Equal(copied, archive.Bytes()) {
		t.Fatal("inspection consumed different bytes")
	}
	if err := frozen.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(original); err != nil {
		t.Fatal("Close removed caller-owned intake")
	}
	if _, err := frozen.Resolve(context.Background(), entry.Reference); err == nil {
		t.Fatal("closed intake resolved")
	}
}

func TestFrozenIntakePinsOriginalFDAndRejectsContentDrift(t *testing.T) {
	for _, mutateContents := range []bool{false, true} {
		t.Run(fmt.Sprint(mutateContents), func(t *testing.T) {
			root := privateFrozenRoot(t)
			entry, original := acquiredFrozenFixture(t, root, "tool-linux-amd64", []byte("one"))
			frozen, err := NewFrozenIntake(context.Background(), root, []FrozenEntry{entry})
			if err != nil {
				t.Fatal(err)
			}
			defer frozen.Close()
			if !mutateContents {
				if err := os.Rename(original, original+".previous"); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(original, []byte("bad"), 0600); err != nil {
				t.Fatal(err)
			}
			run, _ := domain.NewRunID()
			acquired, err := frozen.Acquire(context.Background(), run, entry.Resolved)
			if mutateContents {
				if err == nil || acquired.ContentHandle() != "" {
					t.Fatal("changed original FD content was accepted")
				}
				if _, err := os.Lstat(filepath.Join(root, run.String())); !os.IsNotExist(err) {
					t.Fatal("failed copy left partial intake")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				copied, err := os.ReadFile(filepath.Join(root, run.String(), "asset"))
				if err != nil || string(copied) != "one" {
					t.Fatal("path replacement changed held bytes")
				}
			}
		})
	}
}

func TestFrozenIntakeRejectsLinksPermissionsAndExistingDestination(t *testing.T) {
	for _, kind := range []string{"file-symlink", "run-symlink", "hardlink", "public-file", "public-run", "public-root", "wrong-digest"} {
		t.Run(kind, func(t *testing.T) {
			root := privateFrozenRoot(t)
			entry, original := acquiredFrozenFixture(t, root, "tool-linux-amd64", []byte("one"))
			switch kind {
			case "file-symlink":
				if err := os.Rename(original, original+".old"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(original+".old", original); err != nil {
					t.Fatal(err)
				}
			case "run-symlink":
				dir := filepath.Dir(original)
				if err := os.Rename(dir, dir+".old"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(dir+".old", dir); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(original, original+".alias"); err != nil {
					t.Fatal(err)
				}
			case "public-file":
				if err := os.Chmod(original, 0644); err != nil {
					t.Fatal(err)
				}
			case "public-run":
				if err := os.Chmod(filepath.Dir(original), 0755); err != nil {
					t.Fatal(err)
				}
			case "public-root":
				if err := os.Chmod(root, 0755); err != nil {
					t.Fatal(err)
				}
			case "wrong-digest":
				if err := os.WriteFile(original, []byte("bad"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			frozen, err := NewFrozenIntake(context.Background(), root, []FrozenEntry{entry})
			if err == nil {
				_ = frozen.Close()
				t.Fatal("untrusted source accepted")
			}
		})
	}
	root := privateFrozenRoot(t)
	entry, original := acquiredFrozenFixture(t, root, "tool-linux-amd64", []byte("one"))
	frozen, err := NewFrozenIntake(context.Background(), root, []FrozenEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	defer frozen.Close()
	run, _ := domain.ParseRunID(strings.Split(entry.Acquired.ContentHandle(), ":")[1])
	if _, err := frozen.Acquire(context.Background(), run, entry.Resolved); err == nil {
		t.Fatal("existing Run overwritten")
	}
	body, err := os.ReadFile(original)
	if err != nil || string(body) != "one" {
		t.Fatal("existing Run was removed or changed")
	}
}

func TestFrozenIntakeConcurrentAcquireAndClose(t *testing.T) {
	root := privateFrozenRoot(t)
	entry, _ := acquiredFrozenFixture(t, root, "tool-linux-amd64", []byte("one"))
	frozen, err := NewFrozenIntake(context.Background(), root, []FrozenEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	var copied atomic.Int32
	var first sync.Once
	firstCopy := make(chan struct{})
	finished := make(chan struct{})
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for attempt := 0; attempt < 16; attempt++ {
				run, _ := domain.NewRunID()
				acquired, err := frozen.Acquire(context.Background(), run, entry.Resolved)
				if err != nil {
					return
				}
				if acquired.Digest() != entry.Acquired.Digest() {
					t.Error("concurrent acquire returned a different digest")
				}
				copied.Add(1)
				first.Do(func() { close(firstCopy) })
			}
		}()
	}
	go func() { group.Wait(); close(finished) }()
	select {
	case <-firstCopy:
	case <-finished:
		if copied.Load() == 0 {
			t.Fatal("no acquisition reached copying before Close")
		}
	}
	if err := frozen.Close(); err != nil {
		t.Fatal(err)
	}
	<-finished
	if err := frozen.Close(); err != nil {
		t.Fatal("Close is not idempotent")
	}
	var zero FrozenIntake
	if err := zero.Close(); err != nil {
		t.Fatal(err)
	}
}
