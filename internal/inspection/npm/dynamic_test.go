package npm

import (
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestDynamicInspectorNormalizesOnlyBoundedM3Findings(t *testing.T) {
	result := sandboxResult(t, domain.SandboxCompleted, "", []domain.SandboxObservation{
		observation(t, domain.ObservationHoneytoken, "honeytoken-access"),
		observation(t, domain.ObservationNetwork, "network-attempt"),
		observation(t, domain.ObservationProcess, "process-unexpected"),
	})
	inspector, err := NewDynamicInspector(fakeSandbox{result: result})
	if err != nil {
		t.Fatal(err)
	}
	report, err := inspector.Inspect(context.Background(), dynamicArtifact(t))
	if err != nil {
		t.Fatal(err)
	}
	if report.Execution().Status() != domain.ExecutionCompleted || len(report.Evidence()) != 1 {
		t.Fatalf("report = %#v", report)
	}
	if summary := report.Evidence()[0].Summary(); !strings.Contains(summary, `"schema":"m11-004"`) || !strings.Contains(summary, `"total":3`) {
		t.Fatalf("Evidence summary = %q", summary)
	}
	got := []string{}
	for _, finding := range report.Findings() {
		got = append(got, finding.Code())
	}
	if strings.Join(got, ",") != "M3_HONEYTOKEN_ACCESS,M3_NETWORK_ATTEMPT,M3_UNEXPECTED_PROCESS" {
		t.Fatalf("findings = %q", got)
	}
}

func TestDynamicInspectorFailsClosedOnSummaryOverflow(t *testing.T) {
	observations := make([]domain.SandboxObservation, 0, 33)
	for index := 0; index < 33; index++ {
		observations = append(observations, observation(t, domain.ObservationProcess, "subject-"+strings.Repeat("a", index+1)))
	}
	inspector, err := NewDynamicInspector(fakeSandbox{result: sandboxResult(t, domain.SandboxCompleted, "", observations)})
	if err != nil {
		t.Fatal(err)
	}
	report, err := inspector.Inspect(context.Background(), dynamicArtifact(t))
	if err != nil {
		t.Fatal(err)
	}
	if report.Execution().Status() != domain.ExecutionIncomplete || len(report.Evidence()) != 0 {
		t.Fatalf("summary overflow was not fail-closed: %#v", report)
	}
	if code, _ := report.Execution().LimitationCode(); code != "M11_DYNAMIC_SUMMARY_INVALID" {
		t.Fatalf("limitation = %q", code)
	}
}

func TestDynamicInspectorPreservesUnavailableAndResourceLimits(t *testing.T) {
	tests := []struct {
		name       string
		result     domain.SandboxResult
		capability domain.Capability
		status     domain.ExecutionStatus
		code       string
	}{
		{"linux unsupported", sandboxResult(t, domain.SandboxIncomplete, "M3_LINUX_ONLY", nil), domain.CapabilityUnsupported, domain.ExecutionNotExecuted, "M3_LINUX_ONLY"},
		{"observer failure", sandboxResult(t, domain.SandboxIncomplete, "M3_DYNAMIC_OBSERVER_FAILED", nil), domain.CapabilitySupported, domain.ExecutionIncomplete, "M3_DYNAMIC_OBSERVER_FAILED"},
		{"resource limit", sandboxResult(t, domain.SandboxCompleted, "", []domain.SandboxObservation{observation(t, domain.ObservationResource, "resource-limit")}), domain.CapabilitySupported, domain.ExecutionIncomplete, "M3_DYNAMIC_RESOURCE_LIMIT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inspector, _ := NewDynamicInspector(fakeSandbox{result: test.result})
			report, err := inspector.Inspect(context.Background(), dynamicArtifact(t))
			if err != nil {
				t.Fatal(err)
			}
			if report.Execution().Capability() != test.capability || report.Execution().Status() != test.status {
				t.Fatalf("execution = %#v", report.Execution())
			}
			if code, _ := report.Execution().LimitationCode(); code != test.code {
				t.Fatalf("limitation = %q", code)
			}
			if len(report.Findings()) != 0 || len(report.Evidence()) != 0 {
				t.Fatal("incomplete report contains findings or evidence")
			}
		})
	}
}

type fakeSandbox struct{ result domain.SandboxResult }

func (f fakeSandbox) Execute(context.Context, domain.SandboxRequest) (domain.SandboxResult, error) {
	return f.result, nil
}

func dynamicArtifact(t *testing.T) domain.AcquiredArtifact {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", 1)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func sandboxResult(t *testing.T, status domain.SandboxStatus, code string, observations []domain.SandboxObservation) domain.SandboxResult {
	t.Helper()
	id, err := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	result, err := domain.NewSandboxResult(id, status, code, observations)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func observation(t *testing.T, category domain.ObservationCategory, subject string) domain.SandboxObservation {
	t.Helper()
	item, err := domain.NewSandboxObservation(category, subject)
	if err != nil {
		t.Fatal(err)
	}
	return item
}
