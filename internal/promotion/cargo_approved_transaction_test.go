package promotion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// This fixture has explicit synthetic ALLOW records. Actual public acquisition
// and the installed CLI have separate integration gates.
func approvedCargoFixture(t *testing.T) (*CargoProjectPromotion, *approvedCargoProjectGuard, domain.ProjectDependencyUpdate, domain.StagedProjectSet) {
	t.Helper()
	c, root := newCargoCacheForTest(t)
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(project, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("[package]\nname='haa_schema_fixture'\nversion='0.1.0'\nedition='2021'\n")
	if err := os.WriteFile(filepath.Join(project, "Cargo.toml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	src := []byte("fn main() {}\n")
	if err := os.WriteFile(filepath.Join(project, "src/main.rs"), src, 0o600); err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(project)
	install, _ := domain.NewInstallContext(target)
	p, err := NewCargoProjectPromotion(c, filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}
	guard, err := p.Begin(context.Background(), install)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !guard.closed {
			if err := guard.Close(); err != nil {
				t.Error(err)
			}
		}
	})
	set, _ := cargoCacheFixture(t, c, project, nil)
	entry := set.Inspected().Inspections()[0]
	metadata, err := os.ReadFile("../artifact/cargo/testdata/metadata-v1-rust-1.99.0.json")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.ReadFile("../artifact/cargo/testdata/metadata-v1-rust-1.99.0.lock")
	if err != nil {
		t.Fatal(err)
	}
	lock = bytes.ReplaceAll(lock, []byte("92ecc6618181def0457392ccd0ee51198e065e016d1d527a7ac1b6dc7c1f09d2"), []byte(entry.Artifact().Digest().String()))
	selectedManifest := append(append([]byte(nil), manifest...), []byte("[dependencies]\nitoa='=1.0.17'\n")...)
	files := map[string][]byte{"Cargo.toml": selectedManifest, "Cargo.lock": lock, "src/main.rs": src}
	snapshot, err := artifactcargo.BuildProjectSnapshot(install, metadata, selectedManifest, lock, "/fixture", sandbox.PinnedCargoRuntime().ImageReference, files)
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedProjectSet(snapshot, set.Inspected().Inspections())
	if err != nil {
		t.Fatal(err)
	}
	set, err = domain.NewProjectVerifiedSet(inspected, set.Decision())
	if err != nil {
		t.Fatal(err)
	}
	staged, err := c.StageProject(context.Background(), set)
	if err != nil {
		t.Fatal(err)
	}
	var selected []domain.ProjectControlFile
	for _, name := range []string{"Cargo.toml", "Cargo.lock"} {
		control, err := domain.NewProjectControlFile(name, files[name], true)
		if err != nil {
			t.Fatal(err)
		}
		selected = append(selected, control)
	}
	records, edges, _, err := artifactcargo.ParseLockedMetadata(metadata, lock, "/fixture")
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := artifactcargo.ParseReference("itoa@1.0.17")
	graph, err := artifactcargo.BuildLockedGraph(reference, records, edges)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := domain.NewDependencyResolution(graph, "synthetic:single-selection", snapshot.GraphDigest())
	if err != nil {
		t.Fatal(err)
	}
	update, err := domain.NewProjectDependencyUpdate(guard.Controls(), selected, snapshot, resolution)
	if err != nil {
		t.Fatal(err)
	}
	return p, guard, update, staged
}

func TestCargoApprovedGuardCommitsFrozenControlsAndRetainsApproval(t *testing.T) {
	p, g, update, staged := approvedCargoFixture(t)
	if concurrent, err := p.Begin(context.Background(), g.context); err == nil || concurrent != nil {
		t.Fatal("concurrent guard entered")
	}
	if err := g.Commit(context.Background(), update, staged); err != nil {
		t.Fatal(err)
	}
	for _, control := range update.SelectedControls() {
		body, err := os.ReadFile(filepath.Join(g.plan.root, control.Name()))
		if err != nil || !bytes.Equal(body, control.Body()) {
			t.Fatal("frozen controls not committed")
		}
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := p.Begin(context.Background(), g.context)
	if err != nil {
		t.Fatal(err)
	}
	if err := next.CommitSnapshot(context.Background(), update.Snapshot(), staged); err != nil {
		t.Fatal(err)
	}
	if err := next.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(p.cache.evidenceRoot); err != nil {
		t.Fatal(err)
	}
	if next, err := p.Begin(context.Background(), g.context); err == nil || next != nil {
		t.Fatal("retained approval outlived Evidence")
	}
}

func TestCargoApprovedGuardRejectsDrift(t *testing.T) {
	for _, kind := range []string{"control-content", "control-inode", "missing-lock-appeared", "source-content", "source-inode", "source-extra", "cache", "evidence", "state-appeared"} {
		t.Run(kind, func(t *testing.T) {
			p, g, update, staged := approvedCargoFixture(t)
			root := g.plan.root
			switch kind {
			case "control-content":
				if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname='changed'\nversion='0.1.0'\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "control-inode":
				if err := os.Rename(filepath.Join(root, "Cargo.toml"), filepath.Join(root, "old")); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), g.Controls()[0].Body(), 0o600); err != nil {
					t.Fatal(err)
				}
			case "missing-lock-appeared":
				if err := os.WriteFile(filepath.Join(root, "Cargo.lock"), cargoFixtureControl(t, update.SelectedControls(), "Cargo.lock"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "source-content":
				if err := os.WriteFile(filepath.Join(root, "src/main.rs"), []byte("fn main() { panic!(\"changed\"); }\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "source-inode":
				if err := os.Rename(filepath.Join(root, "src/main.rs"), filepath.Join(root, "src/old.rs")); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "src/main.rs"), []byte("fn main() {}\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "source-extra":
				if err := os.WriteFile(filepath.Join(root, "src/extra.rs"), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			case "cache":
				vendor, err := p.cache.OpenProjectCache(context.Background(), staged)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(vendor, "extra"), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			case "evidence":
				if err := os.RemoveAll(p.cache.evidenceRoot); err != nil {
					t.Fatal(err)
				}
			case "state-appeared":
				if err := os.WriteFile(filepath.Join(p.stateRoot, g.stateName), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := g.Commit(context.Background(), update, staged); err == nil {
				t.Fatal("drift authorized publication")
			}
			if _, err := os.Lstat(filepath.Join(root, cargoTransactionMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed selection published marker")
			}
		})
	}
}

func TestCargoApprovedGuardRejectsForeignCache(t *testing.T) {
	_, g, update, _ := approvedCargoFixture(t)
	_, _, _, foreign := approvedCargoFixture(t)
	if err := g.Commit(context.Background(), update, foreign); err == nil {
		t.Fatal("foreign cache authorized publication")
	}
}

func TestCargoApprovedGuardDoesNotTrustRawMarker(t *testing.T) {
	p, g, _, _ := approvedCargoFixture(t)
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(g.plan.root, ".heliopause"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(g.plan.root, cargoTransactionMetadata), []byte("{\"approval\":\"ALLOW\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if next, err := p.Begin(context.Background(), g.context); err == nil || next != nil {
		t.Fatal("raw marker restored missing independent approval")
	}
}

func TestCargoApprovedTransactionRollbackAndUncertainty(t *testing.T) {
	for _, kind := range []string{"clean-rollback", "foreign-control", "changed-source"} {
		t.Run(kind, func(t *testing.T) {
			p, g, update, staged := approvedCargoFixture(t)
			primary := errors.New("synthetic approval publication failure")
			p.checkpoint = func(phase string) error {
				if phase != "before-approval" {
					return nil
				}
				switch kind {
				case "foreign-control":
					replacement := filepath.Join(g.plan.root, ".foreign")
					if err := os.WriteFile(replacement, []byte("foreign"), 0o600); err != nil {
						return err
					}
					if err := os.Rename(replacement, filepath.Join(g.plan.root, "Cargo.toml")); err != nil {
						return err
					}
				case "changed-source":
					if err := os.WriteFile(filepath.Join(g.plan.root, "src/main.rs"), []byte("changed"), 0o600); err != nil {
						return err
					}
					return nil
				}
				return primary
			}
			err := g.Commit(context.Background(), update, staged)
			if err == nil || (kind != "changed-source" && !errors.Is(err, primary)) {
				t.Fatalf("primary failure missing: %v", err)
			}
			if kind == "foreign-control" {
				body, e := os.ReadFile(filepath.Join(g.plan.root, "Cargo.toml"))
				if e != nil || string(body) != "foreign" || !strings.Contains(err.Error(), "rollback is incomplete") {
					t.Fatalf("uncertain rollback removed foreign member: %v", err)
				}
			} else {
				body, e := os.ReadFile(filepath.Join(g.plan.root, "Cargo.toml"))
				if e != nil || !bytes.Equal(body, cargoFixtureControl(t, update.OriginalControls(), "Cargo.toml")) {
					t.Fatal("rollback failed to restore original controls")
				}
				if _, e := os.Lstat(filepath.Join(g.plan.root, "Cargo.lock")); !errors.Is(e, os.ErrNotExist) {
					t.Fatal("rollback invented original lock")
				}
			}
			if _, e := os.Lstat(filepath.Join(p.stateRoot, g.stateName)); !errors.Is(e, os.ErrNotExist) {
				t.Fatal("failed commit left approval")
			}
		})
	}
}

func cargoFixtureControl(t *testing.T, controls []domain.ProjectControlFile, name string) []byte {
	t.Helper()
	for _, control := range controls {
		if control.Name() == name {
			return control.Body()
		}
	}
	t.Fatalf("fixture control %s missing", name)
	return nil
}

func TestCargoApprovedGuardRequiresRetainedStateAndJournalReconciliation(t *testing.T) {
	for _, kind := range []string{"missing-state", "corrupt-state", "stale-journal"} {
		t.Run(kind, func(t *testing.T) {
			p, g, update, staged := approvedCargoFixture(t)
			if err := g.Commit(context.Background(), update, staged); err != nil {
				t.Fatal(err)
			}
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "missing-state":
				if err := os.Remove(filepath.Join(p.stateRoot, g.stateName)); err != nil {
					t.Fatal(err)
				}
			case "corrupt-state":
				if err := os.WriteFile(filepath.Join(p.stateRoot, g.stateName), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "stale-journal":
				if err := os.Mkdir(filepath.Join(g.plan.root, ".heliopause-cargo-commit-interrupted"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if next, err := p.Begin(context.Background(), g.context); err == nil || next != nil {
				t.Fatal("incomplete retained state allowed selection")
			}
			if _, err := os.Lstat(g.guard.path); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed guard leaked its owned lock")
			}
		})
	}
}

func TestCargoApprovedGuardDoesNotRemoveForeignLock(t *testing.T) {
	_, g, _, _ := approvedCargoFixture(t)
	replacement := g.guard.path + ".foreign"
	if err := os.WriteFile(replacement, []byte("foreign owner"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, g.guard.path); err != nil {
		t.Fatal(err)
	}
	if err := g.VerifyUnchanged(context.Background()); err == nil {
		t.Fatal("foreign lock accepted")
	}
	if err := g.Close(); err == nil {
		t.Fatal("foreign lock treated as successful cleanup")
	}
	body, err := os.ReadFile(g.guard.path)
	if err != nil || string(body) != "foreign owner" {
		t.Fatal("guard removed foreign lock")
	}
}

func TestCargoApprovedTransactionRefusesPublicationRace(t *testing.T) {
	for _, kind := range []string{"selected-content", "selected-inode", "foreign-destination"} {
		t.Run(kind, func(t *testing.T) {
			p, g, update, staged := approvedCargoFixture(t)
			p.checkpoint = func(phase string) error {
				if phase != "before-publication" {
					return nil
				}
				entries, err := filepath.Glob(filepath.Join(g.plan.root, ".heliopause-cargo-commit-*"))
				if err != nil || len(entries) != 1 {
					return errors.New("fixture publication journal missing")
				}
				selected := filepath.Join(entries[0], "selected", "Cargo.toml")
				switch kind {
				case "selected-content":
					return os.WriteFile(selected, []byte("[package]\nname='foreign'\nversion='0.1.0'\n"), 0o600)
				case "selected-inode":
					body, err := os.ReadFile(selected)
					if err != nil {
						return err
					}
					if err := os.WriteFile(selected+".foreign", body, 0o600); err != nil {
						return err
					}
					return os.Rename(selected+".foreign", selected)
				case "foreign-destination":
					return os.WriteFile(filepath.Join(g.plan.root, "Cargo.toml"), []byte("foreign destination"), 0o600)
				}
				return nil
			}
			err := g.Commit(context.Background(), update, staged)
			if err == nil {
				t.Fatal("publication race committed")
			}
			if kind == "foreign-destination" {
				body, e := os.ReadFile(filepath.Join(g.plan.root, "Cargo.toml"))
				if e != nil || string(body) != "foreign destination" || !strings.Contains(err.Error(), "rollback is incomplete") {
					t.Fatalf("foreign destination overwritten: %v", err)
				}
			} else {
				body, e := os.ReadFile(filepath.Join(g.plan.root, "Cargo.toml"))
				if e != nil || !bytes.Equal(body, cargoFixtureControl(t, update.OriginalControls(), "Cargo.toml")) {
					t.Fatalf("rejected race lost original controls: %v", err)
				}
			}
			if _, e := os.Lstat(filepath.Join(p.stateRoot, g.stateName)); !errors.Is(e, os.ErrNotExist) {
				t.Fatal("failed publication left independent approval")
			}
		})
	}
}
