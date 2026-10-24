//go:build linux || darwin

package githubrelease_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectiongithub "github.com/rahoney/heliopause/internal/inspection/githubrelease"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	verificationgithub "github.com/rahoney/heliopause/internal/verification/githubrelease"
	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

func privatePipelineRoot(t *testing.T) string {
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

type releasePipelineSandbox struct {
	incomplete string
	calls      int
}

func (s *releasePipelineSandbox) Execute(_ context.Context, request domain.SandboxRequest) (domain.SandboxResult, error) {
	s.calls++
	id, err := domain.NewSandboxSessionID()
	if err != nil {
		return domain.SandboxResult{}, err
	}
	if request.Artifact().Identity().Variant() == s.incomplete {
		return domain.NewSandboxResult(id, domain.SandboxIncomplete, "M3_DYNAMIC_OBSERVER_FAILED", nil)
	}
	return domain.NewSandboxResult(id, domain.SandboxCompleted, "", nil)
}

type releasePromotionSpy struct{ calls int }

func (s *releasePromotionSpy) Promote(_ context.Context, _ domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (domain.PromotedInstall, error) {
	s.calls++
	return domain.NewPromotedInstall(bundle.ManifestID(), install.Target())
}

func TestHeliopauseSetUsesExistingCompleteWorkflowAndStopsBeforePromotion(t *testing.T) {
	for _, name := range []string{"source-and-staging-only", "native-incomplete", "production-authenticity-rejects-unsigned-set"} {
		t.Run(name, func(t *testing.T) {
			f := githubrelease.NewReleaseSetPipelineFixture(t)
			storeRoot := privatePipelineRoot(t)
			store, err := local.NewStore(storeRoot)
			if err != nil {
				t.Fatal(err)
			}
			static, err := inspectiongithub.NewStaticInspector(f.IntakeRoot)
			if err != nil {
				t.Fatal(err)
			}
			sandbox := &releasePipelineSandbox{}
			if name == "native-incomplete" {
				sandbox.incomplete = "haa-release-installer-linux-amd64"
			}
			dynamic, err := inspectiongithub.NewDynamicInspector(sandbox)
			if err != nil {
				t.Fatal(err)
			}
			native, err := inspectiongithub.NewCompositeInspector(static, dynamic)
			if err != nil {
				t.Fatal(err)
			}
			inspector, err := inspectiongithub.NewReleaseSetInspector(f.IntakeRoot, f.Source.Manifest, native)
			if err != nil {
				t.Fatal(err)
			}
			// The positive is explicitly source/staging-only: there is no signed new
			// native release fixture. The real provenance branch below must BLOCK it.
			var verifier interface {
				Verify(context.Context, domain.AcquiredArtifact) (domain.VerificationReport, error)
			} = verificationgithub.IntegrityVerifier{}
			if name == "production-authenticity-rejects-unsigned-set" {
				verifier, err = releaseprovenance.NewResolvedArtifactVerifier(func(artifact domain.AcquiredArtifact) (releaseprovenance.Identity, [][]byte, error) {
					m, proof, err := f.Source.ReleaseProof(artifact)
					return releaseprovenance.Identity{Version: m.Version(), SourceCommit: m.SourceCommit(), WorkflowRun: m.WorkflowRun()}, proof, err
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			inspection, err := application.NewInstallInspectService(f.Source, f.Source, verifier, inspector, store, policy.M6{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			stageRoot := privatePipelineRoot(t)
			stager, err := promotion.NewLocalStaging(f.IntakeRoot, storeRoot, stageRoot)
			if err != nil {
				t.Fatal(err)
			}
			promoter := &releasePromotionSpy{}
			service, err := application.NewInstallService(inspection, evidence.Generator{}, stager, promoter)
			if err != nil {
				t.Fatal(err)
			}
			request, err := application.NewInstallRequest(f.Source.Reference(), f.Install)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := service.Install(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if len(outcome.Inspection().Set().Inspections()) != 9 || sandbox.calls != 5 {
				t.Fatal("incomplete per-member inspection coverage")
			}
			checks, evidenceCount := 0, 0
			for _, entry := range outcome.Inspection().Set().Inspections() {
				checks += len(entry.Checks())
				evidenceCount += len(entry.Evidence())
				if name == "source-and-staging-only" {
					for _, check := range entry.Checks() {
						if !check.Required() || check.Status() != domain.ExecutionCompleted {
							t.Fatal("required check was incomplete")
						}
					}
				}
			}
			// Five native members require three checks each; manifest and
			// three passive metadata members require two checks each.
			if checks != 23 {
				t.Fatalf("required checks=%d, want 23", checks)
			}
			if name == "source-and-staging-only" {
				if outcome.Inspection().Decision().Decision() != domain.DecisionAllow || promoter.calls != 1 || outcome.Staged().Handle() == "" || evidenceCount != 23 {
					t.Fatal("existing complete-set staging was not reached")
				}
			} else {
				if outcome.Inspection().Decision().Decision() == domain.DecisionAllow || promoter.calls != 0 || outcome.Staged().Handle() != "" {
					t.Fatal("non-ALLOW set was staged or promoted")
				}
				entries, err := os.ReadDir(stageRoot)
				if err != nil || len(entries) != 0 {
					t.Fatal("non-ALLOW wrote staging data")
				}
			}
		})
	}
}
