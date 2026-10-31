//go:build linux && realcorpus

package githubrelease_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectiongithub "github.com/rahoney/heliopause/internal/inspection/githubrelease"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationgithub "github.com/rahoney/heliopause/internal/verification/githubrelease"
	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

// Test-only type conversions join public ports to the existing real backend.
// Production bootstrap composition is separately qualified by its own actual.
type pipelinePipxRunner struct{ backend *sandbox.PythonDynamicBackend }

func (r pipelinePipxRunner) InspectPipxRuntime(ctx context.Context, a domain.AcquiredArtifact, p pypi.PipxRuntimePlan) (inspectiongithub.PipxRuntimeObservation, error) {
	observed, err := r.backend.InspectPipxRuntime(ctx, a, p)
	if err != nil {
		return inspectiongithub.PipxRuntimeObservation{}, err
	}
	result := inspectiongithub.PipxRuntimeObservation{SandboxResult: observed.SandboxResult}
	for _, c := range observed.Commands {
		result.Commands = append(result.Commands, inspectiongithub.PipxCommandObservation{Module: c.Module, UnitID: c.UnitID, OwnerSHA256: c.OwnerSHA256, ZeroExit: c.ZeroExit})
	}
	return result, nil
}

type pipelineResources struct {
	client *hosttool.ObservationResourceClient
}

func pipelineLease(l hosttool.ObservationResourceLease) sandbox.ObservationResourceLease {
	return sandbox.ObservationResourceLease{CgroupParent: l.CgroupParent, CPU: l.CPU, UsageUsec: l.UsageUsec, ReadTiming: sandbox.ObservationResourceReadTiming(l.ReadTiming)}
}
func (r pipelineResources) Create(c context.Context, t, p string) (sandbox.ObservationResourceLease, error) {
	l, e := r.client.Create(c, t, p)
	return pipelineLease(l), e
}
func (r pipelineResources) Register(c context.Context, t, n, p string) (sandbox.ObservationResourceLease, error) {
	l, e := r.client.Register(c, t, n, p)
	return pipelineLease(l), e
}
func (r pipelineResources) Read(c context.Context, t string) (sandbox.ObservationResourceLease, error) {
	l, e := r.client.Read(c, t)
	return pipelineLease(l), e
}
func (r pipelineResources) Terminate(c context.Context, t, n string) (sandbox.ObservationResourceLease, error) {
	l, e := r.client.Terminate(c, t, n)
	return pipelineLease(l), e
}
func (r pipelineResources) Close(c context.Context, t string) (sandbox.ObservationResourceLease, error) {
	l, e := r.client.Close(c, t)
	return pipelineLease(l), e
}

func TestPipxActualCompleteGraphCurrentEvidencePolicyAndStaging(t *testing.T) {
	if os.Getenv("HELOX_PIPX_GRAPH_INTEGRATION") != "1" {
		t.Skip("explicit selected DATA and isolated pinned observer required")
	}
	if os.Geteuid() == 0 {
		t.Fatal("ordinary-user protected client required")
	}
	root := os.Getenv("HELOX_PIPX_RELEASE_ACTUAL_DIR")
	zip, err := os.ReadFile(filepath.Join(root, "pipx-runtime-linux-amd64.zip"))
	if err != nil {
		t.Fatal(err)
	}
	sbom, err := os.ReadFile(filepath.Join(root, "helox-release-sbom.cdx.json"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	launcher, err := hosttool.NewSystemObserverLauncher()
	if err != nil {
		t.Fatal(err)
	}
	supervisor, err := sandbox.NewObserverSupervisor(ctx, func(c context.Context, remote, output string) (sandbox.ObserverProcess, error) {
		return launcher.StartObserver(c, remote, output)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	resources, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	backend, err := sandbox.NewLinuxPipxRuntimeBackendWithExecutor(executor, supervisor.Observer(), pipelineResources{resources})
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"current-source-and-staging-only", "missing-required-runner", "production-authenticity-rejects-unsigned-set"} {
		t.Run(mode, func(t *testing.T) {
			f := githubrelease.NewPipxReleaseSetPipelineFixture(t, zip, sbom)
			storeRoot := privatePipelineRoot(t)
			store, err := local.NewStore(storeRoot)
			if err != nil {
				t.Fatal(err)
			}
			static, err := inspectiongithub.NewStaticInspector(f.IntakeRoot)
			if err != nil {
				t.Fatal(err)
			}
			nativeSandbox := &releasePipelineSandbox{}
			dynamic, err := inspectiongithub.NewDynamicInspector(nativeSandbox)
			if err != nil {
				t.Fatal(err)
			}
			native, err := inspectiongithub.NewCompositeInspector(static, dynamic)
			if err != nil {
				t.Fatal(err)
			}
			var pipx ports.Inspection
			if mode != "missing-required-runner" {
				pipx, err = inspectiongithub.NewPipxRuntimeInspector(f.IntakeRoot, pipelinePipxRunner{backend})
				if err != nil {
					t.Fatal(err)
				}
			}
			inspector, err := inspectiongithub.NewReleaseSetInspector(f.IntakeRoot, f.Source.Manifest, native, pipx)
			if err != nil {
				t.Fatal(err)
			}
			var verifier ports.Verification = verificationgithub.IntegrityVerifier{}
			if mode == "production-authenticity-rejects-unsigned-set" {
				verifier, err = releaseprovenance.NewResolvedArtifactVerifier(func(a domain.AcquiredArtifact) (releaseprovenance.Identity, [][]byte, error) {
					m, p, e := f.Source.ReleaseProof(a)
					return releaseprovenance.Identity{Version: m.Version(), SourceCommit: m.SourceCommit(), WorkflowRun: m.WorkflowRun()}, p, e
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
			outcome, err := service.Install(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			entries := outcome.Inspection().Set().Inspections()
			if len(entries) != 13 || nativeSandbox.calls != 6 {
				t.Fatal("complete graph coverage differs", len(entries), nativeSandbox.calls)
			}
			checks, evidenceCount, zipChecks, zipEvidence := 0, 0, 0, 0
			for _, entry := range entries {
				checks += len(entry.Checks())
				evidenceCount += len(entry.Evidence())
				for _, check := range entry.Checks() {
					if mode == "current-source-and-staging-only" && (!check.Required() || check.Status() != domain.ExecutionCompleted) {
						t.Fatal("required check not completed", check.ID())
					}
					if check.ID().String() == "github-release-pipx-runtime" {
						zipChecks++
					}
				}
				for _, reference := range entry.Evidence() {
					item, _, err := store.ReadReference(ctx, entry.RunID(), reference, entry.Artifact().Identity(), entry.Artifact().Digest())
					if err != nil {
						t.Fatal("current stored Evidence failed readback", err)
					}
					if item.Identity().Variant() != "pipx-runtime-linux-amd64.zip" {
						continue
					}
					zipEvidence++
					if item.Identity().Source().String() != "github-release" || item.Identity().Name() != "rahoney-heliopause" || item.Digest().String() != runtimeidentity.PipxToolSHA256 {
						t.Fatal("original HAA ZIP subject substituted")
					}
					if item.Kind() == "github-release-pipx-command-observation" {
						var value struct {
							Disposition   string
							Owner         string `json:"owner_sha256"`
							Unit          string `json:"unit"`
							Functionality bool   `json:"functionality_attested"`
							Enforced      bool   `json:"later_execution_enforced"`
						}
						if err := json.Unmarshal([]byte(item.Summary()), &value); err != nil {
							t.Fatal(err)
						}
						if value.Disposition != "ATTESTED" || value.Owner != runtimeidentity.PipxToolSHA256 || value.Unit == "" || value.Functionality || value.Enforced {
							t.Fatal("bounded command scope changed", value)
						}
					}
				}
			}
			if mode == "current-source-and-staging-only" {
				if checks != 32 || evidenceCount != 37 || zipChecks != 1 || zipEvidence != 7 || outcome.Inspection().Decision().Decision() != domain.DecisionAllow || promoter.calls != 1 || outcome.Staged().Handle() == "" {
					t.Fatal("current complete set not staged", checks, evidenceCount, zipChecks, zipEvidence)
				}
			} else {
				if outcome.Inspection().Decision().Decision() == domain.DecisionAllow || promoter.calls != 0 || outcome.Staged().Handle() != "" {
					t.Fatal("non-ALLOW staged/promoted")
				}
				files, err := os.ReadDir(stageRoot)
				if err != nil || len(files) != 0 {
					t.Fatal("non-ALLOW wrote staging", err)
				}
			}
			t.Logf("scope=%s subjects=%d checks=%d Evidence=%d ZIPchecks=%d ZIPEvidence=%d native_fixture_calls=%d promoter_spy=%d; native execution/authenticity not claimed", mode, len(entries), checks, evidenceCount, zipChecks, zipEvidence, nativeSandbox.calls, promoter.calls)
		})
	}
}
