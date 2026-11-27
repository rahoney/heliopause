package promotion

import (
	"context"
	"github.com/rahoney/heliopause/internal/core/domain"
	"os"
	"path/filepath"
	"testing"
)

func TestGoCacheRequiresRecordedEvidenceAtBothBoundaries(t *testing.T) {
	for _, stage := range []string{"before-stage", "before-open"} {
		t.Run(stage, func(t *testing.T) {
			root := canonicalGoTestRoot(t)
			cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"))
			if err != nil {
				t.Fatal(err)
			}
			set, _ := goCacheFixture(t, cache.intakeRoot, filepath.Join(root, "project"), "example.com/module", map[string]string{"go.mod": "module example.com/module\n", "module.go": "package module\n"})
			if stage == "before-stage" {
				if err := os.RemoveAll(cache.evidenceRoot); err != nil {
					t.Fatal(err)
				}
				staged, err := cache.StageProject(context.Background(), set)
				if err == nil || staged.Valid() {
					t.Fatal("cache accepted missing recorded Evidence")
				}
				return
			}
			staged, err := cache.StageProject(context.Background(), set)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(cache.evidenceRoot); err != nil {
				t.Fatal(err)
			}
			if _, err := cache.OpenProjectCache(context.Background(), staged); err == nil {
				t.Fatal("cache reopened after required Evidence disappeared")
			}
		})
	}
}

// A reader can return legacy content without changing its opaque reference.
// Profile validation must remain independent of the returned record hash.
type profileChangedEvidenceReader struct {
	ProjectEvidenceReader
	profile domain.ContentDigest
}

func (r profileChangedEvidenceReader) ReadReference(ctx context.Context, run domain.RunID, ref domain.EvidenceReference, identity domain.ResolvedArtifactIdentity, digest domain.ContentDigest) (domain.Evidence, domain.ContentDigest, error) {
	item, hash, err := r.ProjectEvidenceReader.ReadReference(ctx, run, ref, identity, digest)
	if err != nil {
		return item, hash, err
	}
	var profiles []domain.ContentDigest
	if r.profile.String() != "" {
		profiles = append(profiles, r.profile)
	}
	changed, err := domain.NewEvidence(item.ID(), item.CheckID(), item.Identity(), item.Digest(), item.Kind(), item.Summary(), profiles...)
	return changed, hash, err
}

func TestProjectCacheRejectsLegacyDeclarationAndEvidenceProfile(t *testing.T) {
	for _, mode := range []string{"missing-plan", "legacy-evidence", "foreign-profile"} {
		t.Run(mode, func(t *testing.T) {
			root := canonicalGoTestRoot(t)
			cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"))
			if err != nil {
				t.Fatal(err)
			}
			set, _ := goCacheFixture(t, cache.intakeRoot, filepath.Join(root, "project"), "example.com/module", map[string]string{"go.mod": "module example.com/module\n", "module.go": "package module\n"})
			inspection := set.Inspected().Inspections()[0]
			reader := cache.evidence
			if mode == "missing-plan" {
				inspection, err = domain.NewDependencyInspection(inspection.Node(), inspection.RunID(), inspection.Artifact(), inspection.Checks(), inspection.Evidence(), inspection.PolicyDecision())
				if err != nil {
					t.Fatal(err)
				}
			} else {
				var profile domain.ContentDigest
				if mode == "foreign-profile" {
					profile, _ = domain.CheckProfileDigest("foreign-profile-v1", "")
				}
				reader = profileChangedEvidenceReader{reader, profile}
			}
			if _, _, err := projectCacheEvidenceFacts(context.Background(), reader, inspection); err == nil {
				t.Fatal("legacy or foreign-profile approval reused")
			}
		})
	}
}

func TestGoRetainedLegacyEvidenceIsOwnershipOnly(t *testing.T) {
	root := canonicalGoTestRoot(t)
	cache, err := newGoCacheForTest(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	set, _ := goCacheFixture(t, cache.intakeRoot, filepath.Join(root, "project"), "example.com/module", map[string]string{"go.mod": "module example.com/module\n", "module.go": "package module\n"})
	approval, err := cache.goCacheApproval(context.Background(), set)
	if err != nil {
		t.Fatal(err)
	}
	cache.evidence = profileChangedEvidenceReader{ProjectEvidenceReader: cache.evidence}
	if err := cache.verifyRecordedApproval(context.Background(), approval, false); err != nil {
		t.Fatal("legacy ownership was discarded", err)
	}
	if err := cache.verifyRecordedApproval(context.Background(), approval, true); err == nil {
		t.Fatal("legacy ownership became current build approval")
	}
}
