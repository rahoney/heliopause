package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func toolWorkspaceFixture(t *testing.T) (*toolFreshTarget, *toolPythonWorkspace) {
	t.Helper()
	install, requirement, bin, storage := toolTargetFixture(t)
	g, err := beginToolFreshTarget(context.Background(), install, requirement, bin, []string{storage})
	if err != nil {
		t.Fatal(err)
	}
	w, err := newToolPythonWorkspace(g.tx)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("known inert bytes")
	if err := os.WriteFile(filepath.Join(w.path, "known"), body, 0600); err != nil {
		t.Fatal(err)
	}
	if err := w.takeFile("known", toolBytesDigest(body), int64(len(body))); err != nil {
		t.Fatal(err)
	}
	return g, w
}

func TestToolWorkspaceRemovesOnlyKnownFilesAndClosesIdempotently(t *testing.T) {
	g, w := toolWorkspaceFixture(t)
	if err := w.Verify(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(w.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("known workspace retained")
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestToolWorkspacePreservesUnknownChangedAndUnprovenMissingFiles(t *testing.T) {
	for _, mutation := range []string{"foreign", "changed", "missing", "late-foreign", "late-replacement"} {
		t.Run(mutation, func(t *testing.T) {
			g, w := toolWorkspaceFixture(t)
			var err error
			switch mutation {
			case "foreign":
				err = os.WriteFile(filepath.Join(w.path, "foreign"), []byte("keep"), 0600)
			case "changed":
				err = os.WriteFile(filepath.Join(w.path, "known"), []byte("changed inert bytes"), 0600)
			case "missing":
				err = os.Remove(filepath.Join(w.path, "known"))
			case "late-foreign":
				w.beforeRemove = func() error { return os.WriteFile(filepath.Join(w.path, "foreign"), []byte("keep"), 0600) }
			case "late-replacement":
				w.beforeRemove = func() error {
					if err := os.Rename(filepath.Join(w.path, "known"), filepath.Join(w.path, "known.saved")); err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(w.path, "known"), []byte("known inert bytes"), 0600)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err == nil || !g.tx.recovery {
				t.Fatal("uncertain workspace cleanup accepted")
			}
			if _, err := os.Lstat(w.path); err != nil {
				t.Fatal("uncertain workspace root removed")
			}
			if mutation == "foreign" || mutation == "late-foreign" {
				body, err := os.ReadFile(filepath.Join(w.path, "foreign"))
				if err != nil || string(body) != "keep" {
					t.Fatal("foreign bytes lost")
				}
			}
			if mutation == "changed" {
				body, err := os.ReadFile(filepath.Join(w.path, "known"))
				if err != nil || string(body) != "changed inert bytes" {
					t.Fatal("changed bytes lost")
				}
			}
			if mutation == "late-replacement" {
				for _, name := range []string{"known", "known.saved"} {
					if _, err := os.Lstat(filepath.Join(w.path, name)); err != nil {
						t.Fatal("replacement or original lost")
					}
				}
			}
			_ = g.Close()
		})
	}
}

func TestToolWorkspaceRetiresOnlySourceMovedByHeldTransaction(t *testing.T) {
	g, destinations, receipt, work := toolPublicationFixture(t)
	w, err := newToolPythonWorkspace(g.tx)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("published inert bytes")
	path := filepath.Join(w.path, "source")
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	if err := w.takeFile("source", toolBytesDigest(body), int64(len(body))); err != nil {
		t.Fatal(err)
	}
	destinations = append(destinations, pypiDestination{Scheme: "site", Relative: "extra.py", Distribution: "haa-tool-fixture", Version: "1.0.0", ArtifactDigest: strings.Repeat("a", 64), Source: path, Digest: toolBytesDigest(body), Size: int64(len(body))})
	if err := g.commitTool(context.Background(), destinations, receipt, work); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if actual, err := os.ReadFile(filepath.Join(g.tx.plan.site, "extra.py")); err != nil || string(actual) != string(body) {
		t.Fatal("published output removed with workspace")
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
}
