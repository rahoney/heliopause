package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	artifactgithub "github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	evidencelocal "github.com/rahoney/heliopause/internal/evidence/local"
)

type githubOwnershipFixture struct {
	root, target string
	p            *GitHubReleasePromotion
	reader       *evidencelocal.Store
}

func newGitHubOwnershipFixture(t *testing.T) githubOwnershipFixture {
	t.Helper()
	root := realPromotionRoot(t)
	reader, err := evidencelocal.NewStore(filepath.Join(root, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewRecordedGitHubReleasePromotion(filepath.Join(root, "staging"), filepath.Join(root, "evidence"), filepath.Join(root, "state"), reader)
	if err != nil {
		t.Fatal(err)
	}
	return githubOwnershipFixture{root, filepath.Join(root, "target"), p, reader}
}

// Controlled normalized proof fixtures exercise ownership/transactions only.
// They are not public source authentication or actual archive/ELF execution.
func (f githubOwnershipFixture) bundle(t *testing.T, tag, asset string, payload []byte, managed bool) (domain.VerifiedBundle, domain.StagedSet, domain.InstallContext) {
	t.Helper()
	source, _ := domain.NewSourceID(artifactgithub.SourceName)
	identity, _ := domain.NewResolvedArtifactIdentity(source, "owner-repo", tag, asset)
	sum := sha256.Sum256(payload)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	run, _ := domain.NewRunID()
	intake := filepath.Join(f.root, "intake", run.String())
	if err := os.MkdirAll(intake, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(intake, "asset"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	q := url.Values{"tag": {tag}, "asset": {asset}, "release": {"11"}, "asset_id": {"7"}, "sha256": {digest.String()}, "size": {strconv.Itoa(len(payload))}}
	resolved, err := domain.NewResolvedArtifact(identity, "github-release://owner/repo?"+q.Encode(), "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	nodeID, _ := domain.NewDependencyNodeID("github-release-primary")
	node, _ := domain.NewLockedDependencyWithRecordPath(nodeID, domain.DependencyPrimary, resolved, asset, false)
	graph, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
	acquired, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":github-release", uint64(len(payload)), "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	checks := []domain.CheckExecution{}
	facts := []domain.Evidence{}
	for _, v := range []struct {
		id, kind string
		check    domain.CheckKind
	}{{"github-release-declared-sha256", "github-release-declared-sha256", domain.CheckVerification}, {"github-release-static", "github-release-zip-static", domain.CheckInspection}} {
		id, _ := domain.NewCheckID(v.id)
		check, _ := domain.NewCheckExecution(id, v.check, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		checks = append(checks, check)
		eid, _ := domain.NewEvidenceID(v.id + "-result")
		fact, err := domain.NewEvidence(eid, id, identity, digest, v.kind, "Controlled transaction fixture; not public artifact qualification.")
		if err != nil {
			t.Fatal(err)
		}
		facts = append(facts, fact)
	}
	refs, err := f.reader.Record(context.Background(), run, facts)
	if err != nil {
		t.Fatal(err)
	}
	decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "m6-github-release", 1, []string{"M6_REQUIRED_CHECKS_COMPLETED"})
	inspection, err := domain.NewDependencyInspection(nodeID, run, acquired, checks, refs, decision)
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedDependencySet(graph, []domain.DependencyInspection{inspection})
	if err != nil {
		t.Fatal(err)
	}
	verified, err := domain.NewVerifiedSet(inspected, decision)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(f.target)
	install, _ := domain.NewInstallContext(target)
	if managed {
		install, _ = domain.NewGitHubAssetInstallContext(target)
	}
	op, _ := domain.NewOperationID()
	bundle, err := evidence.BuildVerifiedBundle(evidence.ManifestContext{OperationID: op, InstallContext: install, ResolverRuntime: "github-release-standalone", LockfileDigest: digest}, verified)
	if err != nil {
		t.Fatal(err)
	}
	stager, err := NewLocalStaging(filepath.Join(f.root, "intake"), filepath.Join(f.root, "evidence"), filepath.Join(f.root, "staging"))
	if err != nil {
		t.Fatal(err)
	}
	staged, err := stager.Stage(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	return bundle, staged, install
}

func (f githubOwnershipFixture) initial(t *testing.T) (githubAssetApproval, []byte, *githubDirectorySnapshot) {
	t.Helper()
	bundle, staged, install := f.bundle(t, "v1", "tool.zip", []byte("old controlled archive"), false)
	if _, err := f.p.Promote(context.Background(), staged, bundle, install); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(f.target, githubOwnershipMarker))
	if err != nil {
		t.Fatal(err)
	}
	approval, err := decodeGitHubApproval(body)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := snapshotGitHubDirectory(f.target, approval, body)
	if err != nil {
		t.Fatal(err)
	}
	return approval, body, tree
}

type githubSelectionFixture struct {
	resolution domain.DependencyResolution
	calls      int
	mutate     func()
}

func (r *githubSelectionFixture) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	r.calls++
	if r.mutate != nil {
		r.mutate()
	}
	return r.resolution, nil
}

func (f githubOwnershipFixture) begin(t *testing.T, bundle domain.VerifiedBundle, install domain.InstallContext) (*githubReplacementSession, *githubSelectionFixture, error) {
	t.Helper()
	digest := bundle.Set().Inspected().Inspections()[0].Artifact().Digest()
	resolution, err := domain.NewDependencyResolution(bundle.Set().Inspected().Graph(), "github-release-standalone", digest)
	if err != nil {
		t.Fatal(err)
	}
	r := &githubSelectionFixture{resolution: resolution}
	preparation, err := NewGitHubReleaseUpdatePreparation(f.p, r)
	if err != nil {
		t.Fatal(err)
	}
	selector, _, err := artifactgithub.SelectionForResolved(resolution.Graph().Nodes()[0].Artifact())
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID(artifactgithub.SourceName)
	s, err := preparation.BeginInstallUpdate(context.Background(), source, selector.Locator(), install, false)
	if err != nil {
		return nil, r, err
	}
	session := s.(*githubReplacementSession)
	t.Cleanup(func() { _ = session.Close() })
	return session, r, nil
}

func TestGitHubRecordedInstallAndManagedReplacementPreserveProofAndFreshRuns(t *testing.T) {
	f := newGitHubOwnershipFixture(t)
	old, body, _ := f.initial(t)
	for _, v := range []struct{ tag, asset, payload string }{{"v2", "next.zip", "new controlled archive"}, {"v2", "next.zip", "new controlled archive"}} {
		bundle, staged, install := f.bundle(t, v.tag, v.asset, []byte(v.payload), true)
		s, r, err := f.begin(t, bundle, install)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.ResolveDependencies(context.Background(), s.Reference(), install); err != nil {
			t.Fatal(err)
		}
		result, err := s.Promote(context.Background(), staged, bundle, install)
		if err != nil || result.Target().String() != f.target || r.calls != 1 {
			t.Fatal(result, err, r.calls)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		next, err := os.ReadFile(filepath.Join(f.target, githubOwnershipMarker))
		if err != nil {
			t.Fatal(err)
		}
		current, err := decodeGitHubApproval(next)
		if err != nil || current.Run == old.Run || bytes.Equal(next, body) || current.Version != v.tag || current.Asset != v.asset {
			t.Fatal(current, err)
		}
		state, err := os.ReadFile(filepath.Join(f.p.stateRoot, s.stateName))
		if err != nil || !bytes.Equal(state, next) {
			t.Fatal("independent approval not bound", err)
		}
		if _, err := os.Stat(filepath.Join(f.target, "tool.zip")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("retired asset remains", err)
		}
		if err := f.p.verifyApproval(context.Background(), current); err != nil {
			t.Fatal(err)
		}
		old, body = current, next
	}
}

func TestGitHubReplacementRefusesMissingCorruptAliasedAndForeignOriginals(t *testing.T) {
	for _, name := range []string{"missing-state", "corrupt-state", "marker-only", "foreign-file", "changed-payload", "changed-mode", "state-hardlink", "asset-hardlink", "missing-evidence", "changed-evidence", "recovery"} {
		t.Run(name, func(t *testing.T) {
			f := newGitHubOwnershipFixture(t)
			old, _, _ := f.initial(t)
			sum := sha256.Sum256([]byte(f.target))
			state := filepath.Join(f.p.stateRoot, hex.EncodeToString(sum[:])+".json")
			evidencePath := filepath.Join(f.root, "evidence", old.Run, old.Evidence[0].ID+".json")
			switch name {
			case "missing-state", "marker-only":
				if err := os.Remove(state); err != nil {
					t.Fatal(err)
				}
			case "corrupt-state":
				if err := os.WriteFile(state, []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "foreign-file":
				if err := os.WriteFile(filepath.Join(f.target, "user-note"), []byte("preserve"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "changed-payload":
				if err := os.Chmod(filepath.Join(f.target, old.Asset), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(f.target, old.Asset), []byte("foreign payload"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "changed-mode":
				if err := os.Chmod(filepath.Join(f.target, old.Asset), 0o600); err != nil {
					t.Fatal(err)
				}
			case "state-hardlink":
				if err := os.Link(state, filepath.Join(f.root, "state-alias")); err != nil {
					t.Fatal(err)
				}
			case "asset-hardlink":
				if err := os.Link(filepath.Join(f.target, old.Asset), filepath.Join(f.root, "asset-alias")); err != nil {
					t.Fatal(err)
				}
			case "missing-evidence":
				if err := os.Remove(evidencePath); err != nil {
					t.Fatal(err)
				}
			case "changed-evidence":
				if err := os.WriteFile(evidencePath, []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "recovery":
				if err := os.Mkdir(filepath.Join(f.root, ".target.haa-github-commit-incomplete"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.ReadFile(filepath.Join(f.target, old.Asset))
			if err != nil {
				t.Fatal(err)
			}
			bundle, _, install := f.bundle(t, "v2", "next.zip", []byte("next controlled archive"), true)
			s, r, err := f.begin(t, bundle, install)
			if err == nil || s != nil || r.calls != 0 {
				t.Fatal("unproven original admitted", s, err, r.calls)
			}
			after, err := os.ReadFile(filepath.Join(f.target, old.Asset))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("refusal changed original", err)
			}
			if name == "foreign-file" {
				if b, err := os.ReadFile(filepath.Join(f.target, "user-note")); err != nil || string(b) != "preserve" {
					t.Fatal("foreign file lost")
				}
			}
		})
	}
}

func TestGitHubReplacementRollsBackEveryPrecommitFault(t *testing.T) {
	for _, phase := range []string{"before-original-move", "after-original-move", "after-target-publish", "before-approval-publish", "after-approval-publish", "before-backup-cleanup"} {
		t.Run(phase, func(t *testing.T) {
			f := newGitHubOwnershipFixture(t)
			_, body, tree := f.initial(t)
			bundle, staged, install := f.bundle(t, "v2", "next.zip", []byte("next controlled archive"), true)
			s, _, err := f.begin(t, bundle, install)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.ResolveDependencies(context.Background(), s.Reference(), install); err != nil {
				t.Fatal(err)
			}
			primary := errors.New("controlled publication fault")
			f.p.checkpoint = func(p string) error {
				if p == phase {
					return primary
				}
				return nil
			}
			if _, err := s.Promote(context.Background(), staged, bundle, install); !errors.Is(err, primary) {
				t.Fatal("primary fault lost", err)
			}
			if err := tree.verify(f.target); err != nil {
				t.Fatal("old target not exactly restored", err)
			}
			state, err := os.ReadFile(filepath.Join(f.p.stateRoot, s.stateName))
			if err != nil || !bytes.Equal(state, body) {
				t.Fatal("old independent approval not restored", err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			if paths, _ := filepath.Glob(filepath.Join(f.root, ".target.haa-github-commit-*")); len(paths) != 0 {
				t.Fatal("owned recovery payload leaked", paths)
			}
		})
	}
}

func TestGitHubReplacementPreservesForeignPublishedAndBackupMembers(t *testing.T) {
	for _, where := range []string{"selected", "published", "backup"} {
		t.Run(where, func(t *testing.T) {
			f := newGitHubOwnershipFixture(t)
			f.initial(t)
			bundle, staged, install := f.bundle(t, "v2", "next.zip", []byte("next controlled archive"), true)
			s, _, err := f.begin(t, bundle, install)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.ResolveDependencies(context.Background(), s.Reference(), install); err != nil {
				t.Fatal(err)
			}
			var foreign string
			f.p.checkpoint = func(phase string) error {
				if where == "selected" && phase == "before-original-move" {
					foreign = filepath.Join(s.selectedPath, "user-note")
				}
				if where == "published" && phase == "after-target-publish" {
					foreign = filepath.Join(f.target, "user-note")
				}
				if where == "backup" && phase == "before-backup-cleanup" {
					foreign = filepath.Join(s.backupPath, "user-note")
				}
				if foreign != "" {
					return os.WriteFile(foreign, []byte("foreign member must survive"), 0o600)
				}
				return nil
			}
			if _, err := s.Promote(context.Background(), staged, bundle, install); err == nil {
				t.Fatal("foreign transaction mutation accepted")
			}
			if b, err := os.ReadFile(foreign); err != nil || string(b) != "foreign member must survive" {
				t.Fatal("foreign member removed", err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			f.p.checkpoint = nil
			if next, _, err := f.begin(t, bundle, install); err == nil || next != nil {
				t.Fatal("incomplete recovery was silently adopted")
			}
		})
	}
}

func TestGitHubReplacementRejectsConcurrentIdentityAndTimestampDrift(t *testing.T) {
	for _, name := range []string{"bytes", "timestamp", "lock", "lock-empty", "lock-mode", "lock-content"} {
		t.Run(name, func(t *testing.T) {
			f := newGitHubOwnershipFixture(t)
			old, _, _ := f.initial(t)
			bundle, _, install := f.bundle(t, "v2", "next.zip", []byte("next controlled archive"), true)
			s, r, err := f.begin(t, bundle, install)
			if err != nil {
				t.Fatal(err)
			}
			r.mutate = func() {
				switch name {
				case "bytes":
					if err := os.Chmod(filepath.Join(f.target, old.Asset), 0o600); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(f.target, old.Asset), []byte("concurrent foreign content"), 0o600); err != nil {
						t.Fatal(err)
					}
				case "timestamp":
					p := filepath.Join(f.target, old.Asset)
					i, err := os.Stat(p)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.Chtimes(p, i.ModTime(), i.ModTime().Add(time.Second)); err != nil {
						t.Fatal(err)
					}
				case "lock", "lock-empty":
					if err := os.Remove(s.guard.path); err != nil {
						t.Fatal(err)
					}
					body := []byte("foreign lock")
					if name == "lock-empty" {
						body = nil
					}
					if err := os.WriteFile(s.guard.path, body, 0o600); err != nil {
						t.Fatal(err)
					}
				case "lock-mode":
					if err := os.Chmod(s.guard.path, 0o400); err != nil {
						t.Fatal(err)
					}
				case "lock-content":
					if err := os.WriteFile(s.guard.path, []byte("foreign lock"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			if _, err := s.ResolveDependencies(context.Background(), s.Reference(), install); err == nil {
				t.Fatal("concurrent original drift admitted")
			}
			if name == "lock" || name == "lock-empty" || name == "lock-mode" || name == "lock-content" {
				if err := s.Close(); err == nil {
					t.Fatal("foreign lock cleanup accepted")
				}
				want := "foreign lock"
				if name == "lock-empty" || name == "lock-mode" {
					want = ""
				}
				if b, _ := os.ReadFile(s.guard.path); string(b) != want {
					t.Fatal("foreign lock removed")
				}
			}
		})
	}
}

func TestGitHubInitialPublicationFaultRestoresAbsenceAndWrongTargetRefuses(t *testing.T) {
	for _, phase := range []string{"after-target-publish", "after-approval-publish", "wrong-target"} {
		t.Run(phase, func(t *testing.T) {
			f := newGitHubOwnershipFixture(t)
			bundle, staged, install := f.bundle(t, "v1", "tool.zip", []byte("controlled archive"), false)
			primary := errors.New("initial receipt fault")
			if phase == "wrong-target" {
				target, _ := domain.NewInstallTarget(filepath.Join(f.root, "other"))
				install, _ = domain.NewInstallContext(target)
			} else {
				f.p.checkpoint = func(p string) error {
					if p == phase {
						return primary
					}
					return nil
				}
			}
			_, err := f.p.Promote(context.Background(), staged, bundle, install)
			if err == nil || phase != "wrong-target" && !errors.Is(err, primary) {
				t.Fatal("initial fault lost", err)
			}
			if _, err := os.Stat(install.Target().String()); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed initial target left published", err)
			}
			if records, _ := filepath.Glob(filepath.Join(f.p.stateRoot, "*.json")); len(records) != 0 {
				t.Fatal("failed initial install approved", records)
			}
		})
	}
}

func TestGitHubOwnershipSnapshotKeepsM6LimitAndRejectsConcurrentGuard(t *testing.T) {
	f := newGitHubOwnershipFixture(t)
	f.initial(t)
	bundle, _, install := f.bundle(t, "v2", "next.zip", []byte("next"), true)
	s, _, err := f.begin(t, bundle, install)
	if err != nil {
		t.Fatal(err)
	}
	if second, _, err := f.begin(t, bundle, install); err == nil || second != nil {
		t.Fatal("concurrent guard admitted")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(f.root, "large-owned-asset")
	fd, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := fd.Truncate((64 << 20) + 1); err != nil {
		t.Fatal(err)
	}
	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := snapshotBoundedPromotionFile(file, githubAssetLimit)
	if err != nil || !snapshot.matches(file) {
		t.Fatal("existing M6 support narrowed to npm limit", err)
	}
}

func TestGitHubReplacementRejectsRepositoryChangeBeforeSelection(t *testing.T) {
	f := newGitHubOwnershipFixture(t)
	f.initial(t)
	r := &githubSelectionFixture{}
	preparation, err := NewGitHubReleaseUpdatePreparation(f.p, r)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(f.target)
	install, _ := domain.NewGitHubAssetInstallContext(target)
	source, _ := domain.NewSourceID(artifactgithub.SourceName)
	s, err := preparation.BeginInstallUpdate(context.Background(), source, "other/repo@v2#tool.zip", install, false)
	if err == nil || s != nil || ports.InstallUpdateFailureCode(err) != "UPDATE_SOURCE_CHANGED" || r.calls != 0 {
		t.Fatal("repository switch admitted", s, err, r.calls)
	}
}
