package promotion

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func legacyNPMTransactionFixture(t *testing.T) *npmProjectTransaction {
	t.Helper()
	root := realPromotionRoot(t)
	for name, body := range map[string]string{"package.json": `{"dependencies":{}}`, "package-lock.json": `{"packages":{"":{}}}`, "node_modules/old": "old", ".heliopause/old": "old"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := freezeNPMProject(root)
	if err != nil {
		t.Fatal(err)
	}
	w, err := p.privateWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(w) })
	for _, name := range []string{"node_modules/new", ".heliopause/artifacts/verified.tgz"} {
		path := filepath.Join(w, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("new"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := beginNPMProjectTransaction(p, w)
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

func TestNPMTransactionPreservesForeignBackup(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		name := "commit"
		if rollback {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			tx := legacyNPMTransactionFixture(t)
			foreign := filepath.Join(tx.backup, "foreign.txt")
			if err := os.WriteFile(foreign, []byte("foreign"), 0o600); err != nil {
				t.Fatal(err)
			}
			if rollback {
				if err := os.Remove(filepath.Join(tx.workspace, "package-lock.json")); err != nil {
					t.Fatal(err)
				}
			}
			err := tx.commit()
			body, readErr := os.ReadFile(foreign)
			if readErr != nil || string(body) != "foreign" {
				t.Errorf("foreign backup deleted: %q %v", body, readErr)
			}
			if err == nil {
				t.Error("uncertain backup cleanup reported success")
			}
			if err := rejectInterruptedNPMTransaction(tx.plan.root); err == nil {
				t.Error("recovery boundary did not block next transaction")
			}
		})
	}
}

func TestNPMTransactionPreservesForeignPublishedMember(t *testing.T) {
	for _, name := range []string{"node_modules", "haa"} {
		t.Run(name, func(t *testing.T) {
			tx := legacyNPMTransactionFixture(t)
			if err := tx.backupCurrent(); err != nil {
				t.Fatal(err)
			}
			if err := tx.publishWorkspace(); err != nil {
				t.Fatal(err)
			}
			foreign := filepath.Join(tx.target(name), "foreign.txt")
			if err := os.WriteFile(foreign, []byte("foreign"), 0o600); err != nil {
				t.Fatal(err)
			}
			primary := errors.New("test primary publication failure")
			if err := tx.fail(primary); !errors.Is(err, primary) {
				t.Errorf("primary failure lost: %v", err)
			}
			body, err := os.ReadFile(foreign)
			if err != nil || string(body) != "foreign" {
				t.Errorf("foreign publication deleted: %q %v", body, err)
			}
		})
	}
}

func TestVenvTransactionPreservesForeignBackup(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		name := "commit"
		if rollback {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			_, p := makePypiVenvFixture(t)
			runVenvFixture(t, p, map[string]string{"site/old.py": "old"}, "fixture")
			tx, err := beginPyPIVenvTransaction(p)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.close()
			foreign := ""
			primary := errors.New("test primary sync failure")
			tx.fault = func(phase string) error {
				if phase != "sync" {
					return nil
				}
				foreign = filepath.Join(tx.backup, "foreign.txt")
				if err := os.WriteFile(foreign, []byte("foreign"), 0o600); err != nil {
					t.Fatal(err)
				}
				if rollback {
					return primary
				}
				return nil
			}
			err = tx.commit(venvDestinations(t, map[string]string{"site/old.py": "new"}, "fixture"))
			if rollback && !errors.Is(err, primary) {
				t.Errorf("primary failure lost: %v", err)
			}
			body, readErr := os.ReadFile(foreign)
			if readErr != nil || string(body) != "foreign" {
				t.Errorf("foreign backup deleted: %q %v", body, readErr)
			}
			if err == nil || !tx.recovery {
				t.Error("uncertain backup cleanup reported success or released recovery boundary")
			}
		})
	}
}

func driftLegacyMember(t *testing.T, path, kind string) (string, os.FileInfo, []byte) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	switch kind {
	case "content":
		body = []byte("foreign replacement")
		err = os.WriteFile(path, body, 0o600)
	case "mode":
		err = os.Chmod(path, 0o644)
	case "inode":
		err = os.Rename(path, path+".saved")
		if err == nil {
			err = os.WriteFile(path, body, 0o600)
		}
	case "hardlink":
		err = os.Link(path, path+".foreign-link")
	case "missing":
		err = os.Rename(path, path+".foreign")
		path += ".foreign"
	default:
		t.Fatal(kind)
	}
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, info, body
}

func assertLegacyMemberPreserved(t *testing.T, path string, info os.FileInfo, body []byte) {
	t.Helper()
	now, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, now) || info.Mode() != now.Mode() {
		t.Errorf("changed or foreign member identity lost: %v", err)
		return
	}
	actual, err := os.ReadFile(path)
	if err != nil || string(actual) != string(body) {
		t.Errorf("changed or foreign member content lost: %v", err)
	}
}

func TestNPMRollbackPreservesChangedOriginal(t *testing.T) {
	for _, member := range []string{"package.json", "node_modules/old"} {
		for _, kind := range []string{"content", "mode", "inode", "hardlink", "missing"} {
			t.Run(member+"/"+kind, func(t *testing.T) {
				tx := legacyNPMTransactionFixture(t)
				if err := tx.backupCurrent(); err != nil {
					t.Fatal(err)
				}
				if err := tx.publishWorkspace(); err != nil {
					t.Fatal(err)
				}
				path, info, body := driftLegacyMember(t, filepath.Join(tx.backup, member), kind)
				primary := errors.New("test primary publication failure")
				if err := tx.fail(primary); !errors.Is(err, primary) {
					t.Errorf("primary failure lost: %v", err)
				}
				assertLegacyMemberPreserved(t, path, info, body)
				if err := rejectInterruptedNPMTransaction(tx.plan.root); err == nil {
					t.Error("backup uncertainty did not block next transaction")
				}
			})
		}
	}
}

func TestNPMTransactionPreservesCompetingDestination(t *testing.T) {
	tx := legacyNPMTransactionFixture(t)
	if err := tx.backupCurrent(); err != nil {
		t.Fatal(err)
	}
	path := tx.target("package.json")
	if err := os.WriteFile(path, []byte("foreign"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Lstat(path)
	err := tx.publishWorkspace()
	if err == nil {
		t.Error("competing destination was overwritten")
	}
	_ = tx.fail(errors.New("test primary failure"))
	assertLegacyMemberPreserved(t, path, info, []byte("foreign"))
}

func TestNPMGuardPreservesReplacedLock(t *testing.T) {
	root := realPromotionRoot(t)
	g, err := acquireNPMProjectGuard(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(g.path, g.path+".saved"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(g.path, []byte("foreign"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Lstat(g.path)
	if err := g.release(); err == nil {
		t.Error("foreign lock release accepted")
	}
	assertLegacyMemberPreserved(t, g.path, info, []byte("foreign"))
}

func TestVenvTransactionPreservesChangedBackup(t *testing.T) {
	for _, member := range []string{"file-0", "metadata", "ledger.json"} {
		for _, kind := range []string{"content", "mode", "inode", "hardlink", "missing"} {
			for _, rollback := range []bool{false, true} {
				name := "commit"
				if rollback {
					name = "rollback"
				}
				t.Run(member+"/"+kind+"/"+name, func(t *testing.T) {
					_, p := makePypiVenvFixture(t)
					runVenvFixture(t, p, map[string]string{"site/old.py": "old"}, "fixture")
					tx, err := beginPyPIVenvTransaction(p)
					if err != nil {
						t.Fatal(err)
					}
					defer tx.close()
					var path string
					var info os.FileInfo
					var body []byte
					primary := errors.New("test primary sync failure")
					tx.fault = func(phase string) error {
						if phase != "sync" {
							return nil
						}
						path, info, body = driftLegacyMember(t, filepath.Join(tx.backup, member), kind)
						if rollback {
							return primary
						}
						return nil
					}
					err = tx.commit(venvDestinations(t, map[string]string{"site/old.py": "new"}, "fixture"))
					if err == nil || !tx.recovery {
						t.Error("uncertain cleanup reported success or released recovery")
					}
					if rollback && !errors.Is(err, primary) {
						t.Errorf("primary failure lost: %v", err)
					}
					assertLegacyMemberPreserved(t, path, info, body)
				})
			}
		}
	}
}
