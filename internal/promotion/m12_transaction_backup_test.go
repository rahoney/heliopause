package promotion

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectTransactionPreservesUnexpectedBackupContent(t *testing.T) {
	for _, ecosystem := range []string{"go", "cargo"} {
		for _, outcome := range []string{"commit", "rollback"} {
			t.Run(ecosystem+"/"+outcome, func(t *testing.T) {
				root := ""
				var backup string
				failure := errors.New("injected approval failure")
				inject := func() error {
					if err := os.WriteFile(filepath.Join(backup, "foreign.txt"), []byte("keep user data"), 0600); err != nil {
						t.Fatal(err)
					}
					if outcome == "rollback" {
						return failure
					}
					return nil
				}
				var err error
				if ecosystem == "go" {
					root = writeManagedGoProject(t)
					plan, e := freezeGoProject(root)
					if e != nil {
						t.Fatal(e)
					}
					plan.authorized = true
					workspace, e := plan.privateWorkspace()
					if e != nil {
						t.Fatal(e)
					}
					defer os.RemoveAll(workspace)
					tx, e := beginGoProjectTransaction(plan, workspace)
					if e != nil {
						t.Fatal(e)
					}
					backup = tx.backup
					tx.publishApproval = inject
					err = tx.commit()
				} else {
					root = writeManagedCargoProject(t)
					plan, e := freezeCargoProject(root)
					if e != nil {
						t.Fatal(e)
					}
					plan.authorized = true
					workspace, e := plan.privateWorkspace()
					if e != nil {
						t.Fatal(e)
					}
					defer os.RemoveAll(workspace)
					tx, e := beginCargoProjectTransaction(plan, workspace)
					if e != nil {
						t.Fatal(e)
					}
					backup = tx.backup
					tx.publishApproval = inject
					err = tx.commit()
				}
				if err == nil {
					t.Error("transaction accepted unexpected backup content")
				}
				if outcome == "rollback" && !errors.Is(err, failure) {
					t.Error("primary approval failure was lost")
				}
				body, e := os.ReadFile(filepath.Join(backup, "foreign.txt"))
				if e != nil || !bytes.Equal(body, []byte("keep user data")) {
					t.Fatalf("foreign backup content was deleted: %v", e)
				}
			})
		}
	}
}

func TestProjectTransactionRejectsBackupDrift(t *testing.T) {
	kinds := []string{"foreign-directory", "foreign-selected", "backup-inode", "selected-inode", "original-content", "original-mode", "original-inode", "original-hardlink", "original-missing"}
	for _, ecosystem := range []string{"go", "cargo"} {
		for _, outcome := range []string{"commit", "rollback"} {
			for _, kind := range kinds {
				t.Run(ecosystem+"/"+outcome+"/"+kind, func(t *testing.T) {
					var backup, root string
					var preserved string
					primary := errors.New("injected approval failure")
					inject := func() error {
						name := "go.mod"
						if ecosystem == "cargo" {
							name = "Cargo.toml"
						}
						control := filepath.Join(backup, name)
						switch kind {
						case "foreign-directory":
							preserved = filepath.Join(backup, "user", "nested.txt")
							if err := os.Mkdir(filepath.Dir(preserved), 0700); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(preserved, []byte("keep"), 0600); err != nil {
								t.Fatal(err)
							}
						case "foreign-selected":
							preserved = filepath.Join(backup, "selected", "user.txt")
							if err := os.WriteFile(preserved, []byte("keep"), 0600); err != nil {
								t.Fatal(err)
							}
						case "backup-inode":
							if err := os.Rename(backup, backup+".original"); err != nil {
								t.Fatal(err)
							}
							if err := os.Mkdir(backup, 0700); err != nil {
								t.Fatal(err)
							}
							preserved = filepath.Join(backup, "user.txt")
							if err := os.WriteFile(preserved, []byte("keep"), 0600); err != nil {
								t.Fatal(err)
							}
						case "selected-inode":
							selected := filepath.Join(backup, "selected")
							if err := os.Rename(selected, selected+".original"); err != nil {
								t.Fatal(err)
							}
							if err := os.Mkdir(selected, 0700); err != nil {
								t.Fatal(err)
							}
							preserved = filepath.Join(selected, "user.txt")
							if err := os.WriteFile(preserved, []byte("keep"), 0600); err != nil {
								t.Fatal(err)
							}
						case "original-content":
							preserved = control
							if err := os.WriteFile(control, []byte("keep"), 0600); err != nil {
								t.Fatal(err)
							}
						case "original-mode":
							if err := os.Chmod(control, 0644); err != nil {
								t.Fatal(err)
							}
						case "original-inode":
							body, err := os.ReadFile(control)
							if err != nil {
								t.Fatal(err)
							}
							if err := os.Rename(control, control+".original"); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(control, body, 0600); err != nil {
								t.Fatal(err)
							}
						case "original-hardlink":
							if err := os.Link(control, filepath.Join(root, "linked-original")); err != nil {
								t.Fatal(err)
							}
						case "original-missing":
							if err := os.Rename(control, filepath.Join(root, "saved-original")); err != nil {
								t.Fatal(err)
							}
						}
						if outcome == "rollback" {
							return primary
						}
						return nil
					}
					var err error
					if ecosystem == "go" {
						root = writeManagedGoProject(t)
						plan, e := freezeGoProject(root)
						if e != nil {
							t.Fatal(e)
						}
						plan.authorized = true
						workspace, e := plan.privateWorkspace()
						if e != nil {
							t.Fatal(e)
						}
						defer os.RemoveAll(workspace)
						tx, e := beginGoProjectTransaction(plan, workspace)
						if e != nil {
							t.Fatal(e)
						}
						backup = tx.backup
						tx.publishApproval = inject
						err = tx.commit()
					} else {
						root = writeManagedCargoProject(t)
						plan, e := freezeCargoProject(root)
						if e != nil {
							t.Fatal(e)
						}
						plan.authorized = true
						workspace, e := plan.privateWorkspace()
						if e != nil {
							t.Fatal(e)
						}
						defer os.RemoveAll(workspace)
						tx, e := beginCargoProjectTransaction(plan, workspace)
						if e != nil {
							t.Fatal(e)
						}
						backup = tx.backup
						tx.publishApproval = inject
						err = tx.commit()
					}
					if err == nil {
						t.Error("backup drift was accepted")
					}
					if outcome == "rollback" && !errors.Is(err, primary) {
						t.Error("primary failure was lost")
					}
					if _, e := os.Stat(backup); e != nil {
						t.Fatalf("uncertain recovery journal removed: %v", e)
					}
					if preserved != "" {
						body, e := os.ReadFile(preserved)
						if e != nil || !bytes.Equal(body, []byte("keep")) {
							t.Fatalf("foreign or changed backup bytes lost: %v", e)
						}
					}
				})
			}
		}
	}
}
