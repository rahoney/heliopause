package githubrelease

import (
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestDynamicInspectorNormalizesAndFailsClosed(t *testing.T) {
	artifact := dynamicArtifact(t)
	completed := result(t, domain.SandboxCompleted, "", []domain.SandboxObservation{obs(t, domain.ObservationHoneytoken, "honeytoken-access"), obs(t, domain.ObservationNetwork, "network-attempt"), obs(t, domain.ObservationProcess, "process-unexpected")})
	inspector, _ := NewDynamicInspector(githubFakeSandbox{result: completed})
	report, err := inspector.Inspect(context.Background(), artifact)
	if err != nil || report.Execution().Status() != domain.ExecutionCompleted {
		t.Fatalf("Inspect() = %#v, %v", report, err)
	}
	got := []string{}
	for _, finding := range report.Findings() {
		got = append(got, finding.Code())
	}
	if strings.Join(got, ",") != "M3_HONEYTOKEN_ACCESS,M3_NETWORK_ATTEMPT,M3_UNEXPECTED_PROCESS" {
		t.Fatalf("findings = %v", got)
	}
	for _, fixture := range []domain.SandboxResult{result(t, domain.SandboxIncomplete, "M3_DYNAMIC_OBSERVER_FAILED", nil), result(t, domain.SandboxCompleted, "", []domain.SandboxObservation{obs(t, domain.ObservationResource, "resource-limit")})} {
		inspector, _ := NewDynamicInspector(githubFakeSandbox{result: fixture})
		report, err := inspector.Inspect(context.Background(), artifact)
		if err != nil || report.Execution().Status() != domain.ExecutionIncomplete || len(report.Evidence()) != 0 {
			t.Fatalf("incomplete = %#v, %v", report, err)
		}
	}
}

type githubFakeSandbox struct{ result domain.SandboxResult }

func (f githubFakeSandbox) Execute(context.Context, domain.SandboxRequest) (domain.SandboxResult, error) {
	return f.result, nil
}
func dynamicArtifact(t *testing.T) domain.AcquiredArtifact {
	source, _ := domain.NewSourceID("github-release")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "owner-repo", "v1", "tool")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:github-release", 1)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
func result(t *testing.T, status domain.SandboxStatus, code string, observations []domain.SandboxObservation) domain.SandboxResult {
	id, _ := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	value, err := domain.NewSandboxResult(id, status, code, observations)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
func obs(t *testing.T, category domain.ObservationCategory, subject string) domain.SandboxObservation {
	value, err := domain.NewSandboxObservation(category, subject)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
