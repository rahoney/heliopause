package promotion

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testedMutationLock interface {
	verify() error
	release() error
}

func TestMutationLockIdentityAcrossPhysicalOwners(t *testing.T) {
	owners := []struct {
		name    string
		acquire func(string) (testedMutationLock, string, error)
	}{
		{"go-cargo-terraform", func(root string) (testedMutationLock, string, error) {
			g, err := acquireProjectTransactionGuard(root, ".heliopause-go-transaction.lock")
			return g, g.path, err
		}},
		{"npm", func(root string) (testedMutationLock, string, error) {
			g, err := acquireNPMProjectGuard(root)
			return g, g.path, err
		}},
		{"pypi", func(root string) (testedMutationLock, string, error) {
			g, err := acquirePyPIVenvGuard(root)
			return g, g.path, err
		}},
	}
	for _, owner := range owners {
		t.Run(owner.name, func(t *testing.T) {
			for _, mutation := range []string{"ordinary", "rewrite", "unlink-empty", "unlink-nonempty", "mode", "hardlink", "mtime"} {
				t.Run(mutation, func(t *testing.T) {
					root := realPromotionRoot(t)
					g, path, err := owner.acquire(root)
					if err != nil {
						t.Fatal(err)
					}
					before, err := os.Stat(path)
					if err != nil {
						t.Fatal(err)
					}
					want := ""
					switch mutation {
					case "ordinary":
					case "rewrite":
						want = "foreign lock bytes"
						if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
							t.Fatal(err)
						}
					case "unlink-empty", "unlink-nonempty":
						if err := os.Remove(path); err != nil {
							t.Fatal(err)
						}
						if mutation == "unlink-nonempty" {
							want = "foreign replacement"
						}
						if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
							t.Fatal(err)
						}
					case "mode":
						if err := os.Chmod(path, 0o400); err != nil {
							t.Fatal(err)
						}
					case "hardlink":
						if err := os.Link(path, filepath.Join(root, "foreign-alias")); err != nil {
							t.Fatal(err)
						}
					case "mtime":
						if err := os.Chtimes(path, before.ModTime(), before.ModTime().Add(time.Second)); err != nil {
							t.Fatal(err)
						}
					}
					after, err := os.Stat(path)
					if err != nil {
						t.Fatal(err)
					}
					t.Logf("mutation=%s pathname retained same inode=%v", mutation, os.SameFile(before, after))
					if mutation == "ordinary" {
						if err := g.verify(); err != nil {
							t.Fatal(err)
						}
						if err := g.release(); err != nil {
							t.Fatal(err)
						}
						if _, err := os.Stat(path); !os.IsNotExist(err) {
							t.Fatal("owned lock was not released", err)
						}
						return
					}
					if err := g.verify(); err == nil {
						t.Error("foreign mutation admitted")
					}
					if err := g.release(); err == nil {
						t.Error("foreign mutation released as owned")
					}
					body, err := os.ReadFile(path)
					if err != nil || string(body) != want {
						t.Error("foreign lock pathname or bytes removed", err)
					}
				})
			}
		})
	}
}

func TestMutationLockRecoveryPreservesNameAndClosesDescriptor(t *testing.T) {
	root := realPromotionRoot(t)
	guard, err := acquirePyPIVenvGuard(root)
	if err != nil {
		t.Fatal(err)
	}
	transaction := pypiVenvTransaction{guard: guard, recovery: true}
	if err := transaction.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(guard.path); err != nil {
		t.Fatal("recovery pathname removed", err)
	}
	if _, err := guard.file.Stat(); !os.IsNotExist(err) && !errors.Is(err, os.ErrClosed) {
		t.Fatal("recovery descriptor leaked", err)
	}
	if _, err := acquirePyPIVenvGuard(root); err == nil {
		t.Fatal("recovery lock silently replaced")
	}
}
