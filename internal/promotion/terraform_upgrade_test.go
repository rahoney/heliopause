package promotion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func seedTerraformUpgrade(t *testing.T, p *TerraformProjectPromotion, install domain.InstallContext) {
	t.Helper()
	ctx := context.Background()
	g, err := p.Begin(ctx, install)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	update, set, _ := terraformApprovedFixture(t, p, g)
	staged, err := p.cache.StageProject(ctx, set)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Commit(ctx, update, staged); err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
}

type terraformOriginalMember struct {
	info os.FileInfo
	body []byte
}

func terraformProjectSnapshot(t *testing.T, root string) map[string]terraformOriginalMember {
	t.Helper()
	result := map[string]terraformOriginalMember{}
	err := filepath.WalkDir(root, func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		var body []byte
		if info.Mode().IsRegular() {
			body, err = os.ReadFile(path)
		} else if info.Mode()&os.ModeSymlink != 0 {
			var link string
			link, err = os.Readlink(path)
			body = []byte(link)
		}
		if err != nil {
			return err
		}
		result[relative] = terraformOriginalMember{info, body}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func terraformProjectUnchanged(t *testing.T, root string, before map[string]terraformOriginalMember) {
	t.Helper()
	after := terraformProjectSnapshot(t, root)
	if len(before) != len(after) {
		t.Fatal("project namespace changed")
	}
	for name, old := range before {
		current, ok := after[name]
		if !ok || !os.SameFile(old.info, current.info) || old.info.Mode() != current.info.Mode() || !bytes.Equal(old.body, current.body) || !old.info.IsDir() && !old.info.ModTime().Equal(current.info.ModTime()) {
			t.Fatal("original member changed", name)
		}
	}
}

func TestTerraformUpgradeEnrollmentPreservesIndependentOldAuthority(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		managed, adopt, allowed bool
	}{
		{"native-implicit", false, false, false}, {"native-explicit", false, true, true},
		{"managed", true, false, true}, {"changed-implicit", true, false, false}, {"changed-explicit", true, true, true},
		{"missing-evidence", true, true, false}, {"corrupt-state", true, true, false}, {"marker-tamper", true, true, false},
		{"state-hardlink", true, true, false}, {"control-hardlink", true, true, false}, {"recovery", true, true, false}, {"unmanaged-installed", true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, install := newTerraformPromotionFixture(t)
			root := install.Target().String()
			if tc.managed {
				seedTerraformUpgrade(t, p, install)
			}
			state := filepath.Join(p.stateRoot, terraformHash([]byte(root))+".json")
			if tc.name == "changed-implicit" || tc.name == "changed-explicit" || tc.name == "missing-evidence" {
				file := filepath.Join(root, "main.tf")
				body, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, append(body, []byte("\n// deliberate edited control\n")...), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			switch tc.name {
			case "missing-evidence":
				files, err := filepath.Glob(filepath.Join(p.cache.evidenceRoot, "*", "*.json"))
				if err != nil || len(files) == 0 {
					t.Fatal(err)
				}
				if err := os.Remove(files[0]); err != nil {
					t.Fatal(err)
				}
			case "corrupt-state":
				if err := os.WriteFile(state, []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "marker-tamper":
				if err := os.WriteFile(filepath.Join(root, terraformTransactionMetadata), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "state-hardlink":
				if err := os.Link(state, state+".alias"); err != nil {
					t.Fatal(err)
				}
			case "control-hardlink":
				if err := os.Link(filepath.Join(root, artifactterraform.LockControl), filepath.Join(root, "foreign-lock")); err != nil {
					t.Fatal(err)
				}
			case "recovery":
				if err := os.Mkdir(filepath.Join(root, ".heliopause-terraform-commit-foreign"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "unmanaged-installed":
				if err := os.Remove(state); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(filepath.Join(root, terraformTransactionMetadata)); err != nil {
					t.Fatal(err)
				}
			}
			before := terraformProjectSnapshot(t, root)
			stateBefore, _ := os.ReadFile(state)
			g, err := p.BeginUpgrade(context.Background(), install, tc.adopt)
			if (err == nil) != tc.allowed {
				t.Fatal("unexpected enrollment result", err)
			}
			if g != nil {
				if err := g.Close(); err != nil {
					t.Fatal(err)
				}
			}
			terraformProjectUnchanged(t, root, before)
			stateAfter, _ := os.ReadFile(state)
			if !bytes.Equal(stateBefore, stateAfter) {
				t.Fatal("Begin rewrote independent authority")
			}
		})
	}
}

func TestTerraformUpgradeRollbackAndChangedLockPermissions(t *testing.T) {
	for _, changed := range []bool{false, true} {
		for _, phase := range []string{"normal", "before-backup", "after-lock-publication", "after-provider-publication", "before-approval", "after-approval"} {
			t.Run(phase+map[bool]string{false: "-retained", true: "-changed"}[changed], func(t *testing.T) {
				p, install := newTerraformPromotionFixture(t)
				seedTerraformUpgrade(t, p, install)
				root := install.Target().String()
				lock := filepath.Join(root, artifactterraform.LockControl)
				if changed {
					body, err := os.ReadFile(lock)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(lock, append([]byte("# deliberate user lock comment\n"), body...), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Chmod(lock, 0o640); err != nil {
					t.Fatal(err)
				}
				before := terraformProjectSnapshot(t, root)
				g, err := p.BeginUpgrade(context.Background(), install, changed)
				if err != nil {
					t.Fatal(err)
				}
				defer g.Close()
				update, set, _ := terraformApprovedFixture(t, p, g)
				update, err = domain.NewProjectDependencyUpdateFromSnapshot(update.OriginalControls(), update.SelectedControls(), update.Snapshot())
				if err != nil {
					t.Fatal(err)
				}
				staged, err := p.cache.StageProject(context.Background(), set)
				if err != nil {
					t.Fatal(err)
				}
				fault := errors.New("upgrade exact phase failure")
				p.checkpoint = func(current string) error {
					if current == phase {
						return fault
					}
					return nil
				}
				err = g.Commit(context.Background(), update, staged)
				if phase == "normal" {
					if err != nil {
						t.Fatal(err)
					}
				} else if !errors.Is(err, fault) {
					t.Fatal("primary phase cause lost", err)
				}
				if err := g.Close(); err != nil {
					t.Fatal(err)
				}
				if phase != "normal" {
					terraformProjectUnchanged(t, root, before)
				} else {
					info, err := os.Stat(lock)
					if err != nil || info.Mode().Perm() != 0o640 {
						t.Fatal("changed lock permission lost", err)
					}
					if !changed && !os.SameFile(info, before[artifactterraform.LockControl].info) {
						t.Fatal("no-op lock replaced")
					}
				}
			})
		}
	}
}

func TestTerraformUpgradePreservesForeignRetainedLockAndBackup(t *testing.T) {
	for _, kind := range []string{"content", "mode", "timestamp", "backup-member"} {
		t.Run(kind, func(t *testing.T) {
			p, install := newTerraformPromotionFixture(t)
			seedTerraformUpgrade(t, p, install)
			root := install.Target().String()
			lock := filepath.Join(root, artifactterraform.LockControl)
			g, err := p.BeginUpgrade(context.Background(), install, false)
			if err != nil {
				t.Fatal(err)
			}
			defer g.Close()
			update, set, _ := terraformApprovedFixture(t, p, g)
			update, err = domain.NewProjectDependencyUpdateFromSnapshot(update.OriginalControls(), update.SelectedControls(), update.Snapshot())
			if err != nil {
				t.Fatal(err)
			}
			staged, err := p.cache.StageProject(context.Background(), set)
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(lock)
			if err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(lock)
			if err != nil {
				t.Fatal(err)
			}
			foreign := append(append([]byte(nil), original...), []byte("# external edit must survive\n")...)
			var sentinel string
			wantTime := before.ModTime().Add(time.Second)
			p.checkpoint = func(phase string) error {
				if phase != "before-approval" {
					return nil
				}
				switch kind {
				case "content":
					return os.WriteFile(lock, foreign, 0o600)
				case "mode":
					return os.Chmod(lock, 0o640)
				case "timestamp":
					return os.Chtimes(lock, wantTime, wantTime)
				case "backup-member":
					dirs, err := filepath.Glob(filepath.Join(root, ".heliopause-terraform-commit-*"))
					if err != nil || len(dirs) != 1 {
						t.Fatal("expected one owned backup", dirs, err)
					}
					sentinel = filepath.Join(dirs[0], "external-user-file")
					return os.WriteFile(sentinel, []byte("preserve external file"), 0o600)
				}
				return nil
			}
			if err := g.Commit(context.Background(), update, staged); err == nil {
				t.Fatal("foreign mutation was accepted")
			}
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(lock)
			if err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(lock)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "content":
				if !bytes.Equal(body, foreign) {
					t.Fatal("foreign lock content overwritten")
				}
			case "mode":
				if info.Mode().Perm() != 0o640 {
					t.Fatal("foreign lock mode overwritten")
				}
			case "timestamp":
				if !info.ModTime().Equal(wantTime) {
					t.Fatal("foreign lock timestamp overwritten")
				}
			case "backup-member":
				b, err := os.ReadFile(sentinel)
				if err != nil || string(b) != "preserve external file" {
					t.Fatal("foreign backup member deleted", err)
				}
			}
			dirs, err := filepath.Glob(filepath.Join(root, ".heliopause-terraform-commit-*"))
			if err != nil || len(dirs) != 1 {
				t.Fatal("uncertain recovery namespace removed", dirs, err)
			}
			if next, err := p.BeginUpgrade(context.Background(), install, false); err == nil {
				next.Close()
				t.Fatal("incomplete recovery accepted")
			}
		})
	}
}
