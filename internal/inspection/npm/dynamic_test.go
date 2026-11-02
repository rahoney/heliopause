package npm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
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
	if report.Execution().Status() != domain.ExecutionCompleted || len(report.Evidence()) != 3 {
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

type fakeSandbox struct {
	result domain.SandboxResult
	err    error
}

func (f fakeSandbox) Execute(context.Context, domain.SandboxRequest) (domain.SandboxResult, error) {
	return f.result, f.err
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
	var states []domain.SandboxEnvironmentState
	if status == domain.SandboxCompleted {
		digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
		for _, profile := range []domain.SandboxEnvironmentProfile{domain.SandboxEnvironmentMinimal, domain.SandboxEnvironmentDevelopment} {
			states = append(states, domain.SandboxEnvironmentState{Profile: profile, FixtureDigest: digest, Before: digest, After: digest, VFSOperationsCovered: true})
		}
	}
	result, err := domain.NewSandboxResult(id, status, code, observations, states...)
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

func TestDynamicInspectorPreservesTrustedIncompleteCause(t *testing.T) {
	cause := errors.New("resource cleanup failed")
	for _, status := range []domain.SandboxStatus{domain.SandboxIncomplete, domain.SandboxCompleted} {
		code := ""
		if status == domain.SandboxIncomplete {
			code = "M3_DYNAMIC_TIMEOUT"
		}
		inspector, _ := NewDynamicInspector(fakeSandbox{result: sandboxResult(t, status, code, nil), err: cause})
		report, err := inspector.Inspect(context.Background(), dynamicArtifact(t))
		if status == domain.SandboxCompleted {
			if !errors.Is(err, cause) {
				t.Fatal("operational error accepted as completion")
			}
			continue
		}
		if got, _ := report.Execution().LimitationCode(); err != nil || report.Execution().Status() != domain.ExecutionIncomplete || got != code {
			t.Fatalf("first incomplete result lost: %s %v", got, err)
		}
	}
}

func TestDynamicInspectorRequiresFreshEnvironmentProfile(t *testing.T) {
	inspector, _ := NewDynamicInspector(fakeSandbox{result: sandboxResult(t, domain.SandboxCompleted, "", nil)})
	artifact := dynamicArtifact(t)
	expected, err := inspector.ExpectedChecks(context.Background(), artifact, domain.CheckInspection)
	if err != nil || len(expected) != 1 {
		t.Fatal("required profile declaration unavailable", err)
	}
	report, err := inspector.Inspect(context.Background(), artifact)
	if err != nil || !expected[0].Required() || report.Execution().ProfileDigest() != expected[0].ProfileDigest() {
		t.Fatal("current result/declaration disagree", err)
	}
	run, _ := domain.NewRunID()
	plan, err := domain.NewCheckPlan(run, artifact, expected)
	if err != nil {
		t.Fatal(err)
	}
	for _, previous := range []struct{ name, descriptor string }{
		{"npm-dynamic-lifecycle-v1", runtimeidentity.NodeImageReference},
		{"npm-dynamic-lifecycle-environments-v2", runtimeidentity.NodeImageReference},
		{dynamicCheckProfileName, strings.Replace(dynamicCheckRuntimeDescriptor, domain.SandboxEnvironmentStateSchema, "workspace-state-v1", 1)},
	} {
		legacy, _ := domain.CheckProfileDigest(previous.name, previous.descriptor)
		old, err := domain.NewCheckExecution(expected[0].ID(), domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", legacy)
		if err != nil || plan.Validate(run, artifact, []domain.CheckExecution{old}) == nil || plan.Validate(run, artifact, []domain.CheckExecution{report.Execution()}) != nil {
			t.Fatal("retained approval was accepted or current approval rejected", previous.name, err)
		}
	}
}

func TestDynamicInspectorRequiresBothStateEvidenceAndRetainsBoundScope(t *testing.T) {
	full := sandboxResult(t, domain.SandboxCompleted, "", nil)
	for _, count := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			result, err := domain.NewSandboxResult(full.SessionID(), domain.SandboxCompleted, "", nil, full.EnvironmentStates()[:count]...)
			if err != nil {
				t.Fatal(err)
			}
			inspector, _ := NewDynamicInspector(fakeSandbox{result: result})
			artifact := dynamicArtifact(t)
			report, err := inspector.Inspect(context.Background(), artifact)
			if err != nil {
				t.Fatal(err)
			}
			if count < 2 {
				code, _ := report.Execution().LimitationCode()
				if report.Execution().Status() != domain.ExecutionIncomplete || code != "M13_DYNAMIC_ENVIRONMENT_INCOMPLETE" || len(report.Evidence()) != 0 {
					t.Fatal("missing state accepted")
				}
				return
			}
			if report.Execution().Status() != domain.ExecutionCompleted || len(report.Evidence()) != 3 {
				t.Fatal("state facts missing")
			}
			for _, e := range report.Evidence()[1:] {
				if e.Identity() != artifact.Identity() || e.Digest() != artifact.Digest() || e.CheckID() != report.Execution().ID() || len(e.Summary()) > 1024 || !strings.Contains(e.Summary(), `"transient_changes":"SESSION_VFS_OPERATIONS"`) || !strings.Contains(e.Summary(), `"scope":"MUTABLE_WORKSPACE"`) {
					t.Fatal("state subject, bound or scope lost")
				}
			}
		})
	}
}

func TestDynamicInspectorRefusesSnapshotWithoutActiveOperationCoverage(t *testing.T) {
	full := sandboxResult(t, domain.SandboxCompleted, "", nil)
	states := full.EnvironmentStates()
	states[1].VFSOperationsCovered = false
	result, err := domain.NewSandboxResult(full.SessionID(), domain.SandboxCompleted, "", nil, states...)
	if err != nil {
		t.Fatal(err)
	}
	inspector, _ := NewDynamicInspector(fakeSandbox{result: result})
	report, err := inspector.Inspect(context.Background(), dynamicArtifact(t))
	code, _ := report.Execution().LimitationCode()
	if err != nil || report.Execution().Status() != domain.ExecutionIncomplete || code != "M13_DYNAMIC_ENVIRONMENT_INCOMPLETE" || len(report.Evidence()) != 0 {
		t.Fatal("snapshot-only evidence accepted as active operation coverage", err)
	}
}
