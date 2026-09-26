package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/policy"
)

func TestExecuteInstallPresentsCompletedAndFailedPromotionWithoutLosingPolicy(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		promotion error
		exit      int
		status    string
	}{
		{name: "completed", exit: 0, status: "COMPLETED"},
		{name: "promotion failure", promotion: errors.New("promotion failed"), exit: 1, status: "FAILED"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, request := installCLIFixture(t, policy.M4{}, test.promotion)
			var output bytes.Buffer
			exitCode, err := cli.ExecuteInstall(context.Background(), service, request, true, &output)
			if exitCode != test.exit || !errors.Is(err, test.promotion) {
				t.Fatalf("ExecuteInstall() exit=%d error=%v", exitCode, err)
			}
			var document map[string]any
			if err := json.Unmarshal(output.Bytes(), &document); err != nil {
				t.Fatal(err)
			}
			if document["schema_version"] != "helox.operation-result/v1" || document["operation"] != "INSTALL" || document["operation_status"] != test.status || document["promotion_status"] != test.status {
				t.Fatalf("Install result = %#v", document)
			}
			policyDocument, _ := document["policy"].(map[string]any)
			verified, _ := document["verified_set"].(map[string]any)
			if policyDocument["decision"] != "ALLOW" || verified["entry_count"] != float64(1) || verified["manifest_id"] == "" {
				t.Fatalf("Policy/Verified Set = %#v / %#v", policyDocument, verified)
			}
			attribution, _ := document["dependency_policy_attribution"].([]any)
			if len(attribution) != 1 {
				t.Fatalf("dependency policy attribution = %#v", attribution)
			}
			entry, _ := attribution[0].(map[string]any)
			if entry["node"] != "safe" || entry["package"] != "safe" || entry["entry_policy_decision"] != "ALLOW" {
				t.Fatalf("dependency policy attribution entry = %#v", entry)
			}
			if test.promotion != nil {
				failure := document["error"].(map[string]any)
				if failure["code"] != "PROMOTION_FAILED" || strings.Contains(output.String(), test.promotion.Error()) {
					t.Fatalf("failure disclosure = %s", output.String())
				}
			}
		})
	}
}

func TestExecuteInstallReportsNonAllowWithoutVerifiedSet(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		decision domain.Decision
		exit     int
	}{
		{decision: domain.DecisionManualReview, exit: 10},
		{decision: domain.DecisionBlock, exit: 20},
	} {
		t.Run(string(test.decision), func(t *testing.T) {
			service, request := installCLIFixture(t, fixedCLISetPolicy{decision: test.decision}, nil)
			var output bytes.Buffer
			exitCode, err := cli.ExecuteInstall(context.Background(), service, request, true, &output)
			if err != nil || exitCode != test.exit {
				t.Fatalf("ExecuteInstall() exit=%d error=%v", exitCode, err)
			}
			var document map[string]any
			_ = json.Unmarshal(output.Bytes(), &document)
			policyDocument, _ := document["policy"].(map[string]any)
			if document["operation_status"] != "NOT_PERFORMED" || document["promotion_status"] != "NOT_PERFORMED" || document["verified_set"] != nil || policyDocument["decision"] != string(test.decision) {
				t.Fatalf("non-ALLOW result = %#v", document)
			}
		})
	}
}

func installCLIFixture(t *testing.T, setPolicy application.DependencySetPolicy, promotionError error) (*application.InstallService, application.InstallRequest) {
	t.Helper()
	ports := newInstallCLIPorts(t)
	inspection, err := application.NewInstallInspectService(ports, ports, ports, ports, ports, policy.M1{}, setPolicy, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewInstallService(inspection, evidence.Generator{}, installCLIStaging{}, installCLIPromotion{err: promotionError})
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/heliopause-cli-target")
	installContext, _ := domain.NewInstallContext(target)
	request, _ := application.NewInstallRequest(ports.reference, installContext)
	return service, request
}

type installCLIPorts struct {
	reference domain.ArtifactReference
	resolved  domain.ResolvedArtifact
	graph     domain.LockedDependencyGraph
}

func newInstallCLIPorts(t *testing.T) *installCLIPorts {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	reference, _ := domain.NewArtifactReference(source, "safe@1.0.0")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "safe", "1.0.0", "tarball")
	resolved, _ := domain.NewResolvedArtifact(identity, "https://registry.npmjs.org/safe/-/safe-1.0.0.tgz", "sha512-declared")
	nodeID, _ := domain.NewDependencyNodeID("safe")
	node, _ := domain.NewLockedDependencyWithRecordPath(nodeID, domain.DependencyPrimary, resolved, "node_modules/safe", false)
	graph, _ := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
	return &installCLIPorts{reference: reference, resolved: resolved, graph: graph}
}

func (p *installCLIPorts) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	digest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	return domain.NewDependencyResolution(p.graph, "node:22.23.1;npm:10.9.8", digest)
}

func (p *installCLIPorts) Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	return domain.ResolvedArtifact{}, errors.New("individual resolve is forbidden")
}

func (p *installCLIPorts) Acquire(_ context.Context, runID domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), digest, "intake:"+runID.String()+":tarball", 1, resolved.DeclaredIntegrity())
}

func (p *installCLIPorts) Verify(_ context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	checkID, _ := domain.NewCheckID("cli-integrity")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	item, _ := domain.NewEvidence(mustCLIInstallEvidenceID("cli-integrity"), checkID, artifact.Identity(), artifact.Digest(), "fixture", "integrity completed")
	return domain.NewVerificationReport(check, domain.VerificationVerified, []domain.Evidence{item})
}

func (p *installCLIPorts) Inspect(_ context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	checkID, _ := domain.NewCheckID("cli-inspection")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	item, _ := domain.NewEvidence(mustCLIInstallEvidenceID("cli-inspection"), checkID, artifact.Identity(), artifact.Digest(), "fixture", "inspection completed")
	return domain.NewInspectionReport(check, nil, []domain.Evidence{item})
}

func (p *installCLIPorts) Record(_ context.Context, runID domain.RunID, items []domain.Evidence) ([]domain.EvidenceReference, error) {
	references := make([]domain.EvidenceReference, 0, len(items))
	for _, item := range items {
		reference, _ := domain.NewEvidenceReference(item.ID(), "evidence:"+runID.String()+":"+item.ID().String())
		references = append(references, reference)
	}
	return references, nil
}

type installCLIStaging struct{}

func (installCLIStaging) Stage(_ context.Context, bundle domain.VerifiedBundle) (domain.StagedSet, error) {
	return domain.NewStagedSet(bundle.ManifestID(), "staging:"+bundle.ManifestID().String())
}

type installCLIPromotion struct{ err error }

func (p installCLIPromotion) Promote(_ context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, installContext domain.InstallContext) (domain.PromotedInstall, error) {
	if p.err != nil {
		return domain.PromotedInstall{}, p.err
	}
	return domain.NewPromotedInstall(bundle.ManifestID(), installContext.Target())
}

type fixedCLISetPolicy struct{ decision domain.Decision }

func (p fixedCLISetPolicy) EvaluateSet(domain.InspectedDependencySet) (domain.PolicyDecision, error) {
	return domain.NewPolicyDecision(p.decision, "non-allow-set", 1, []string{"NON_ALLOW"})
}

func mustCLIInstallEvidenceID(value string) domain.EvidenceID {
	id, err := domain.NewEvidenceID(value)
	if err != nil {
		panic(err)
	}
	return id
}
