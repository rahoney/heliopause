package promotion

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
	evidencelocal "github.com/rahoney/heliopause/internal/evidence/local"
)

func newTerraformPromotionFixture(t *testing.T) (*TerraformProjectPromotion, domain.InstallContext) {
	t.Helper()
	root := canonicalGoTestRoot(t)
	project := filepath.Join(root, "project")
	if e := os.Mkdir(project, 0o700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(project, "main.tf"), []byte(`terraform {
 required_providers {
 random = { source = "hashicorp/random", version = "3.7.2" }
 }
}
`), 0o600); e != nil {
		t.Fatal(e)
	}
	store, e := evidencelocal.NewStore(filepath.Join(root, "evidence"))
	if e != nil {
		t.Fatal(e)
	}
	cache, e := NewTerraformVerifiedCache(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"), store)
	if e != nil {
		t.Fatal(e)
	}
	promoter, e := NewTerraformProjectPromotion(cache, filepath.Join(root, "state"))
	if e != nil {
		t.Fatal(e)
	}
	target, e := domain.NewInstallTarget(project)
	if e != nil {
		t.Fatal(e)
	}
	install, e := domain.NewInstallContext(target)
	if e != nil {
		t.Fatal(e)
	}
	return promoter, install
}

// Synthetic ALLOW/Evidence exercises the approval consumer, never public signer
// qualification. Public source, real observation and CLI tests remain separate.
func terraformApprovedFixture(t *testing.T, p *TerraformProjectPromotion, g *approvedTerraformProjectGuard) (domain.ProjectDependencyUpdate, domain.ProjectVerifiedSet, string) {
	t.Helper()
	elf := make([]byte, 128)
	copy(elf, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint16(elf[16:], 2)
	binary.LittleEndian.PutUint16(elf[18:], 62)
	binary.LittleEndian.PutUint32(elf[20:], 1)
	binary.LittleEndian.PutUint64(elf[32:], 64)
	binary.LittleEndian.PutUint16(elf[52:], 64)
	binary.LittleEndian.PutUint16(elf[54:], 56)
	binary.LittleEndian.PutUint16(elf[56:], 1)
	binary.LittleEndian.PutUint16(elf[58:], 64)
	binary.LittleEndian.PutUint32(elf[64:], 1)
	binary.LittleEndian.PutUint64(elf[96:], 128)
	binary.LittleEndian.PutUint64(elf[104:], 128)
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	h := &zip.FileHeader{Name: "terraform-provider-random_v3.7.2_x5", Method: zip.Store}
	h.SetMode(0o755)
	f, e := w.CreateHeader(h)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.Write(elf); e != nil {
		t.Fatal(e)
	}
	if e = w.Close(); e != nil {
		t.Fatal(e)
	}
	parts := [][]byte{[]byte("synthetic discovery"), []byte("synthetic versions"), []byte("synthetic package"), []byte("synthetic checksums"), []byte("synthetic signature"), archive.Bytes()}
	frame := func(header string, parts [][]byte) []byte {
		b := bytes.NewBufferString(header)
		for _, part := range parts {
			if e := binary.Write(b, binary.BigEndian, uint64(len(part))); e != nil {
				t.Fatal(e)
			}
			b.Write(part)
		}
		return b.Bytes()
	}
	raw := frame("HAA-TERRAFORM-PROVIDER-1\n", parts)
	integrity := "registry=" + terraformHash(frame("HAA-TERRAFORM-REGISTRY-1\n", parts[:3])) + ";archive=" + terraformHash(archive.Bytes())
	run, e := domain.NewRunID()
	if e != nil {
		t.Fatal(e)
	}
	directory := filepath.Join(p.cache.intakeRoot, run.String())
	if e := os.MkdirAll(directory, 0o700); e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(directory, "provider.bundle")
	if e := os.WriteFile(file, raw, 0o600); e != nil {
		t.Fatal(e)
	}
	identity, e := domain.NewResolvedArtifactIdentity(artifactterraform.Source(), "hashicorp_random", "3.7.2", "linux/amd64")
	if e != nil {
		t.Fatal(e)
	}
	digest, e := domain.NewSHA256Digest(terraformHash(raw))
	if e != nil {
		t.Fatal(e)
	}
	acquired, e := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":linux/amd64", uint64(len(raw)), integrity)
	if e != nil {
		t.Fatal(e)
	}
	bundle, e := artifactterraform.ReadIntake(p.cache.intakeRoot, acquired)
	if e != nil {
		t.Fatal(e)
	}
	contents, e := artifactterraform.InspectPackage(context.Background(), bundle, "hashicorp/random@3.7.2")
	if e != nil {
		t.Fatal(e)
	}
	lockBody, e := artifactterraform.EncodeLock([]artifactterraform.LockedProvider{{Address: "registry.terraform.io/hashicorp/random", Version: "3.7.2", Constraints: "3.7.2", Hashes: []string{contents.H1, contents.ZH}}})
	if e != nil {
		t.Fatal(e)
	}
	controls := g.Controls()
	for i, c := range controls {
		if c.Name() == artifactterraform.LockControl {
			controls[i], e = domain.NewProjectControlFile(c.Name(), lockBody, true)
			if e != nil {
				t.Fatal(e)
			}
		}
	}
	var digests []domain.ProjectControlDigest
	for _, c := range controls {
		d, e := domain.NewProjectControlDigest(c.Name(), c.Digest())
		if e != nil {
			t.Fatal(e)
		}
		digests = append(digests, d)
	}
	resolved, e := domain.NewResolvedArtifact(identity, "terraform-registry:"+bundle.RegistryDigest(), integrity)
	if e != nil {
		t.Fatal(e)
	}
	graphDigest, e := domain.NewSHA256Digest(strings.Repeat("a", 64))
	if e != nil {
		t.Fatal(e)
	}
	snapshot, e := domain.NewProjectDependencySnapshot(g.install, artifactterraform.Source(), digests, []domain.ResolvedArtifact{resolved}, graphDigest)
	if e != nil {
		t.Fatal(e)
	}
	nodeID, e := domain.ProjectDependencyNodeID(identity)
	if e != nil {
		t.Fatal(e)
	}
	node, e := domain.NewLockedDependency(nodeID, domain.DependencyPrimary, resolved)
	if e != nil {
		t.Fatal(e)
	}
	graph, e := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
	if e != nil {
		t.Fatal(e)
	}
	resolution, e := domain.NewDependencyResolution(graph, "synthetic terraform approval consumer", graphDigest)
	if e != nil {
		t.Fatal(e)
	}
	update, e := domain.NewProjectDependencyUpdate(g.Controls(), controls, snapshot, resolution)
	if e != nil {
		t.Fatal(e)
	}
	allow, e := domain.NewPolicyDecision(domain.DecisionAllow, "fixture", 1, []string{"FIXTURE_ALLOW"})
	if e != nil {
		t.Fatal(e)
	}
	var checks []domain.CheckExecution
	var evidence []domain.Evidence
	for i, name := range []string{"terraform-signed-package", "terraform-provider-static", "terraform-provider-dynamic"} {
		id, e := domain.NewCheckID(name)
		if e != nil {
			t.Fatal(e)
		}
		kind := domain.CheckInspection
		if i == 0 {
			kind = domain.CheckVerification
		}
		check, e := domain.NewCheckExecution(id, kind, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		if e != nil {
			t.Fatal(e)
		}
		checks = append(checks, check)
		eid, e := domain.NewEvidenceID(name)
		if e != nil {
			t.Fatal(e)
		}
		ev, e := domain.NewEvidence(eid, id, identity, digest, "fixture", "Synthetic approval consumer evidence.")
		if e != nil {
			t.Fatal(e)
		}
		evidence = append(evidence, ev)
	}
	store, e := evidencelocal.NewStore(p.cache.evidenceRoot)
	if e != nil {
		t.Fatal(e)
	}
	refs, e := store.Record(context.Background(), run, evidence)
	if e != nil {
		t.Fatal(e)
	}
	entry, e := domain.NewDependencyInspection(nodeID, run, acquired, checks, refs, allow)
	if e != nil {
		t.Fatal(e)
	}
	inspected, e := domain.NewInspectedProjectSet(snapshot, []domain.DependencyInspection{entry})
	if e != nil {
		t.Fatal(e)
	}
	set, e := domain.NewProjectVerifiedSet(inspected, allow)
	if e != nil {
		t.Fatal(e)
	}
	return update, set, file
}

func TestTerraformVerifiedCacheRequiresRecordedEvidenceAndExactTree(t *testing.T) {
	for _, scenario := range []string{"normal", "intake-tamper", "missing-evidence-stage", "missing-evidence-reuse", "file-tamper", "extra-file", "extra-directory", "symlink", "hardlink", "writable-executable", "receipt-tamper"} {
		t.Run(scenario, func(t *testing.T) {
			p, install := newTerraformPromotionFixture(t)
			g, e := p.Begin(context.Background(), install)
			if e != nil {
				t.Fatal(e)
			}
			defer g.Close()
			_, set, intake := terraformApprovedFixture(t, p, g)
			evidenceRun := filepath.Join(p.cache.evidenceRoot, set.Inspected().Inspections()[0].RunID().String())
			if scenario == "intake-tamper" {
				if e := os.WriteFile(intake, []byte("tamper"), 0o600); e != nil {
					t.Fatal(e)
				}
			}
			if scenario == "missing-evidence-stage" {
				if e := os.RemoveAll(evidenceRun); e != nil {
					t.Fatal(e)
				}
			}
			staged, e := p.cache.StageProject(context.Background(), set)
			if scenario == "intake-tamper" || scenario == "missing-evidence-stage" {
				if e == nil {
					t.Fatal("invalid approval staged")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			tree, e := p.cache.OpenProjectCache(context.Background(), staged)
			if e != nil {
				t.Fatal(e)
			}
			exe := filepath.Join(tree, "registry.terraform.io/hashicorp/random/3.7.2/linux_amd64/terraform-provider-random_v3.7.2_x5")
			switch scenario {
			case "missing-evidence-reuse":
				e = os.RemoveAll(evidenceRun)
			case "file-tamper":
				e = os.Chmod(exe, 0o755)
				if e == nil {
					e = os.WriteFile(exe, []byte("tamper"), 0o555)
				}
			case "extra-file":
				e = os.WriteFile(filepath.Join(tree, "extra"), []byte("extra"), 0o444)
			case "extra-directory":
				e = os.Mkdir(filepath.Join(tree, "extra"), 0o755)
			case "symlink":
				e = os.Symlink(exe, filepath.Join(tree, "alias"))
			case "hardlink":
				e = os.Link(exe, filepath.Join(tree, "alias"))
			case "writable-executable":
				e = os.Chmod(exe, 0o755)
			case "receipt-tamper":
				e = os.WriteFile(filepath.Join(filepath.Dir(tree), "receipt.json"), []byte("{}"), 0o600)
			}
			if e != nil {
				t.Fatal(e)
			}
			_, e = p.cache.OpenProjectCache(context.Background(), staged)
			if scenario == "normal" {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil {
				t.Fatal("drifted approval reused")
			}
		})
	}
}

func TestTerraformGuardPublishesRetainedStateAndRollsBackEveryPhase(t *testing.T) {
	for _, retained := range []bool{false, true} {
		for _, phase := range []string{"normal", "before-backup", "after-lock-publication", "after-provider-publication", "before-approval", "after-approval"} {
			t.Run(phase+map[bool]string{false: "-initial", true: "-retained"}[retained], func(t *testing.T) {
				p, install := newTerraformPromotionFixture(t)
				ctx := context.Background()
				publish := func() {
					g, e := p.Begin(ctx, install)
					if e != nil {
						t.Fatal(e)
					}
					update, set, _ := terraformApprovedFixture(t, p, g)
					staged, e := p.cache.StageProject(ctx, set)
					if e != nil {
						t.Fatal(e)
					}
					if e = g.Commit(ctx, update, staged); e != nil {
						t.Fatal(e)
					}
					if e = g.Close(); e != nil {
						t.Fatal(e)
					}
				}
				if retained {
					publish()
				}
				var originalLock, originalMarker, originalState []byte
				originalLock, _ = os.ReadFile(filepath.Join(install.Target().String(), artifactterraform.LockControl))
				originalMarker, _ = os.ReadFile(filepath.Join(install.Target().String(), terraformTransactionMetadata))
				originalState, _ = os.ReadFile(filepath.Join(p.stateRoot, terraformHash([]byte(install.Target().String()))+".json"))
				g, e := p.Begin(ctx, install)
				if e != nil {
					t.Fatal(e)
				}
				if _, e := p.Begin(ctx, install); e == nil {
					t.Fatal("concurrent init obtained same guard")
				}
				update, set, _ := terraformApprovedFixture(t, p, g)
				staged, e := p.cache.StageProject(ctx, set)
				if e != nil {
					t.Fatal(e)
				}
				fault := errors.New("exact phase failure")
				p.checkpoint = func(at string) error {
					if at == phase {
						return fault
					}
					return nil
				}
				e = g.Commit(ctx, update, staged)
				if phase == "normal" {
					if e != nil {
						t.Fatal(e)
					}
				} else {
					if !errors.Is(e, fault) {
						t.Fatalf("primary failure lost: %v", e)
					}
					for path, want := range map[string][]byte{artifactterraform.LockControl: originalLock, terraformTransactionMetadata: originalMarker} {
						got, re := os.ReadFile(filepath.Join(install.Target().String(), path))
						if len(want) == 0 {
							if !errors.Is(re, os.ErrNotExist) {
								t.Fatalf("unexpected published %s: %v", path, re)
							}
						} else if re != nil || !bytes.Equal(got, want) {
							t.Fatalf("original %s not restored: %v", path, re)
						}
					}
					got, se := os.ReadFile(filepath.Join(p.stateRoot, terraformHash([]byte(install.Target().String()))+".json"))
					if len(originalState) == 0 {
						if !errors.Is(se, os.ErrNotExist) {
							t.Fatal("independent approval survived rollback")
						}
					} else if se != nil || !bytes.Equal(got, originalState) {
						t.Fatal("retained state not restored")
					}
					if !retained {
						if _, e := os.Lstat(filepath.Join(install.Target().String(), ".terraform")); !errors.Is(e, os.ErrNotExist) {
							t.Fatal("partial providers survived rollback")
						}
						if _, e := os.Lstat(filepath.Join(install.Target().String(), ".heliopause")); !errors.Is(e, os.ErrNotExist) {
							t.Fatal("new metadata directory survived rollback")
						}
					}
				}
				if e = g.Close(); e != nil {
					t.Fatal(e)
				}
				p.checkpoint = nil
				next, e := p.Begin(ctx, install)
				if e != nil {
					t.Fatalf("clean retry/retained validation failed: %v", e)
				}
				if e = next.Close(); e != nil {
					t.Fatal(e)
				}
			})
		}
	}
}

func TestTerraformGuardRejectsControlAndApprovalAuthorityDrift(t *testing.T) {
	for _, scenario := range []string{"configuration", "new-local-module", "lock", "foreign-marker", "foreign-provider", "metadata-directory-alias", "state-root-alias"} {
		t.Run(scenario, func(t *testing.T) {
			p, install := newTerraformPromotionFixture(t)
			ctx := context.Background()
			g, e := p.Begin(ctx, install)
			if e != nil {
				t.Fatal(e)
			}
			defer g.Close()
			project := install.Target().String()
			switch scenario {
			case "configuration":
				e = os.WriteFile(filepath.Join(project, "main.tf"), []byte("changed"), 0o600)
			case "new-local-module":
				e = os.WriteFile(filepath.Join(project, "extra.tf"), []byte("module \"external\" { source = \"https://example.invalid/module\" }"), 0o600)
			case "lock":
				e = os.WriteFile(filepath.Join(project, artifactterraform.LockControl), []byte("changed"), 0o600)
			case "foreign-marker":
				e = os.Mkdir(filepath.Join(project, ".heliopause"), 0o700)
				if e == nil {
					e = os.WriteFile(filepath.Join(project, terraformTransactionMetadata), []byte("forged"), 0o600)
				}
			case "foreign-provider":
				e = os.Mkdir(filepath.Join(project, ".terraform"), 0o755)
			case "metadata-directory-alias":
				e = os.Symlink(p.stateRoot, filepath.Join(project, ".heliopause"))
			case "state-root-alias":
				e = os.Rename(p.stateRoot, p.stateRoot+".old")
				if e == nil {
					e = os.Mkdir(p.stateRoot, 0o700)
				}
			}
			if e != nil {
				t.Fatal(e)
			}
			if e := g.VerifyUnchanged(ctx); e == nil {
				t.Fatal("changed original boundary accepted")
			}
		})
	}
}

func TestTerraformRetainedApprovalAndMarkerPermissions(t *testing.T) {
	for _, scenario := range []string{"approval-before-begin", "marker-before-begin", "approval-during-guard", "marker-during-guard", "approval-after-publication", "approval-after-publication-clean-return", "approval-body-after-publication", "lock-after-publication", "marker-after-publication"} {
		t.Run(scenario, func(t *testing.T) {
			p, install := newTerraformPromotionFixture(t)
			ctx := context.Background()
			g, err := p.Begin(ctx, install)
			if err != nil {
				t.Fatal(err)
			}
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
			changed := filepath.Join(p.stateRoot, g.stateName)
			if strings.HasPrefix(scenario, "marker-") {
				changed = filepath.Join(install.Target().String(), terraformTransactionMetadata)
			}
			if scenario == "lock-after-publication" {
				changed = filepath.Join(install.Target().String(), artifactterraform.LockControl)
			}
			if strings.HasSuffix(scenario, "before-begin") {
				if err := os.Chmod(changed, 0o644); err != nil {
					t.Fatal(err)
				}
				g, err = p.Begin(ctx, install)
				if err == nil {
					_ = g.Close()
					t.Fatal("changed retained control permissions accepted")
				}
				return
			}
			g, err = p.Begin(ctx, install)
			if err != nil {
				t.Fatal(err)
			}
			defer g.Close()
			if strings.HasSuffix(scenario, "during-guard") {
				if err := os.Chmod(changed, 0o644); err != nil {
					t.Fatal(err)
				}
				if err := g.VerifyUnchanged(ctx); err == nil {
					t.Fatal("permissions drift during guard accepted")
				}
				return
			}
			update, set, _ = terraformApprovedFixture(t, p, g)
			staged, err = p.cache.StageProject(ctx, set)
			if err != nil {
				t.Fatal(err)
			}
			p.checkpoint = func(phase string) error {
				if phase == "after-approval" {
					if scenario == "approval-body-after-publication" {
						if err := os.WriteFile(changed, []byte("foreign approval bytes"), 0o600); err != nil {
							t.Fatal(err)
						}
						return nil
					}
					if err := os.Chmod(changed, 0o644); err != nil {
						t.Fatal(err)
					}
					if scenario == "approval-after-publication" {
						return errors.New("foreign approval permissions after publication")
					}
				}
				return nil
			}
			if err := g.Commit(ctx, update, staged); err == nil {
				t.Fatal("foreign approval mode accepted")
			}
			info, err := os.Lstat(changed)
			if scenario == "approval-body-after-publication" {
				body, err := os.ReadFile(changed)
				if err != nil || string(body) != "foreign approval bytes" {
					t.Fatal("foreign approval content overwritten by rollback")
				}
				return
			}
			if err != nil || info.Mode().Perm() != 0o644 {
				t.Fatal("foreign approval mode was overwritten by rollback")
			}
		})
	}
}

func TestTerraformRollbackPreservesForeignMutationAndRecoveryBackup(t *testing.T) {
	p, install := newTerraformPromotionFixture(t)
	ctx := context.Background()
	g, e := p.Begin(ctx, install)
	if e != nil {
		t.Fatal(e)
	}
	update, set, _ := terraformApprovedFixture(t, p, g)
	staged, e := p.cache.StageProject(ctx, set)
	if e != nil {
		t.Fatal(e)
	}
	foreign := filepath.Join(install.Target().String(), ".terraform/providers/foreign")
	fault := errors.New("foreign modification after publication")
	p.checkpoint = func(phase string) error {
		if phase != "after-provider-publication" {
			return nil
		}
		if e := os.WriteFile(foreign, []byte("foreign bytes must survive"), 0o444); e != nil {
			t.Fatal(e)
		}
		return fault
	}
	e = g.Commit(ctx, update, staged)
	if !errors.Is(e, fault) || !strings.Contains(e.Error(), "rollback is incomplete") {
		t.Fatalf("uncertain rollback accepted: %v", e)
	}
	if body, e := os.ReadFile(foreign); e != nil || string(body) != "foreign bytes must survive" {
		t.Fatal("foreign modified tree removed")
	}
	backups, e := filepath.Glob(filepath.Join(install.Target().String(), ".heliopause-terraform-commit-*"))
	if e != nil || len(backups) != 1 {
		t.Fatal("uncertain recovery backup lost")
	}
	if e := g.Close(); e != nil {
		t.Fatal(e)
	}
	p.checkpoint = nil
	if next, e := p.Begin(ctx, install); e == nil {
		next.Close()
		t.Fatal("incomplete transaction automatically adopted")
	}
}

func TestTerraformCommitAndRollbackPreserveForeignBackupMembers(t *testing.T) {
	for _, where := range []string{"root", "selected"} {
		t.Run(where, func(t *testing.T) {
			p, install := newTerraformPromotionFixture(t)
			ctx := context.Background()
			g, e := p.Begin(ctx, install)
			if e != nil {
				t.Fatal(e)
			}
			update, set, _ := terraformApprovedFixture(t, p, g)
			staged, e := p.cache.StageProject(ctx, set)
			if e != nil {
				t.Fatal(e)
			}
			var foreign string
			p.checkpoint = func(phase string) error {
				if phase != "after-approval" {
					return nil
				}
				backups, e := filepath.Glob(filepath.Join(install.Target().String(), ".heliopause-terraform-commit-*"))
				if e != nil || len(backups) != 1 {
					t.Fatal("exact private backup missing")
				}
				dir := backups[0]
				if where == "selected" {
					dir = filepath.Join(dir, "selected")
				}
				foreign = filepath.Join(dir, "foreign")
				return os.WriteFile(foreign, []byte("foreign backup bytes"), 0o444)
			}
			if e := g.Commit(ctx, update, staged); e == nil {
				t.Fatal("foreign backup was deleted on commit")
			}
			if body, e := os.ReadFile(foreign); e != nil || string(body) != "foreign backup bytes" {
				t.Fatal("foreign backup removed by rollback")
			}
			if e := g.Close(); e != nil {
				t.Fatal(e)
			}
			if next, e := p.Begin(ctx, install); e == nil {
				next.Close()
				t.Fatal("uncertain backup automatically adopted")
			}
		})
	}
}
