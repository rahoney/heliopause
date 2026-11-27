package githubrelease

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func pipxReportFixture(t *testing.T) (domain.AcquiredArtifact, []artifactpypi.PlannedObservationUnit, PipxRuntimeObservation) {
	t.Helper()
	_, artifact, _ := pipxInputFixture(t, nil)
	units := []artifactpypi.PlannedObservationUnit{{Kind: artifactpypi.DirectImportUnit, Candidate: "pipx", Coverage: artifactpypi.RequiredObservation}}
	observed := PipxRuntimeObservation{SandboxResult: result(t, domain.SandboxCompleted, "", nil)}
	for index, name := range []string{"_pytest.config", "argcomplete.scripts.activate_global_python_argcomplete", "argcomplete.scripts.register_python_argcomplete", "pygments.cmdline", "userpath.cli"} {
		units = append(units, artifactpypi.PlannedObservationUnit{Kind: artifactpypi.DirectImportUnit, Candidate: name, Coverage: artifactpypi.PostInstallCommandObservation})
		observed.Commands = append(observed.Commands, PipxCommandObservation{Module: name, OwnerSHA256: artifact.Digest().String(), UnitID: "DIRECT_IMPORT:" + strings.Repeat(string(rune('a'+index)), 64), ZeroExit: index != 0})
	}
	return artifact, units, observed
}

func TestPipxRuntimeRequiredReportPreservesSubjectFindingsAndCommandDisposition(t *testing.T) {
	for _, suspicious := range []bool{false, true} {
		artifact, units, observed := pipxReportFixture(t)
		if suspicious {
			observed.SandboxResult = result(t, domain.SandboxCompleted, "", []domain.SandboxObservation{obs(t, domain.ObservationNetwork, "network-attempt")})
		}
		report, err := pipxRuntimeReport(artifact, units, observed)
		if err != nil || !report.Execution().Required() || report.Execution().ID().String() != pipxRuntimeCheck || report.Execution().Status() != domain.ExecutionCompleted || len(report.Evidence()) != 6 {
			t.Fatal("required report did not complete", err)
		}
		if (len(report.Findings()) == 1) != suspicious {
			t.Fatal("finding was lost")
		}
		for _, evidence := range report.Evidence() {
			if evidence.Identity() != artifact.Identity() || evidence.Digest() != artifact.Digest() {
				t.Fatal("original HAA identity lost")
			}
		}

		for index, evidence := range report.Evidence()[1:] {
			var command struct {
				Module, Disposition string
				Functionality       bool `json:"functionality_attested"`
				Enforced            bool `json:"later_execution_enforced"`
			}
			if err := json.Unmarshal([]byte(evidence.Summary()), &command); err != nil {
				t.Fatal(err)
			}
			expected := "ATTESTED"
			if suspicious || index == 0 {
				expected = "NOT_ATTESTED"
			}
			if command.Disposition != expected || command.Functionality || command.Enforced {
				t.Fatal("bounded import became functionality authority", command)
			}
		}
	}
}

func TestPipxRuntimeReportRejectsIncompleteAndSubstitutedCommandEvidence(t *testing.T) {
	for _, mode := range []string{"zero-result", "incomplete", "resource", "missing", "duplicate", "extra", "module", "owner", "unit-kind", "unit-size", "unit-hex", "duplicate-unit"} {
		t.Run(mode, func(t *testing.T) {
			artifact, units, observed := pipxReportFixture(t)
			switch mode {
			case "zero-result":
				observed.SandboxResult = domain.SandboxResult{}
			case "incomplete":
				observed.SandboxResult = result(t, domain.SandboxIncomplete, "P4_PIPX_ACCOUNTING_UNAVAILABLE", nil)
			case "resource":
				observed.SandboxResult = result(t, domain.SandboxCompleted, "", []domain.SandboxObservation{obs(t, domain.ObservationResource, "resource-limit")})
			case "missing":
				observed.Commands = observed.Commands[:4]
			case "duplicate":
				observed.Commands[1] = observed.Commands[0]
			case "extra":
				observed.Commands = append(observed.Commands, observed.Commands[0])
			case "module":
				observed.Commands[0].Module = "foreign"
			case "owner":
				observed.Commands[0].OwnerSHA256 = strings.Repeat("f", 64)
			case "unit-kind":
				observed.Commands[0].UnitID = "HOOK:" + strings.Repeat("a", 64)
			case "unit-size":
				observed.Commands[0].UnitID = "DIRECT_IMPORT:a"
			case "unit-hex":
				observed.Commands[0].UnitID = "DIRECT_IMPORT:" + strings.Repeat("g", 64)
			case "duplicate-unit":
				observed.Commands[1].UnitID = observed.Commands[0].UnitID
			}
			report, err := pipxRuntimeReport(artifact, units, observed)
			if err != nil || report.Execution().Status() != domain.ExecutionIncomplete || !report.Execution().Required() || len(report.Evidence()) != 0 {
				t.Fatal("invalid outcome produced completed Evidence", err)
			}
		})
	}
}

type pipxInspectionSpy struct {
	calls    int
	observed PipxRuntimeObservation
	err      error
	after    func()
}

func (s *pipxInspectionSpy) InspectPipxRuntime(context.Context, domain.AcquiredArtifact, artifactpypi.PipxRuntimePlan) (PipxRuntimeObservation, error) {
	s.calls++
	if s.after != nil {
		s.after()
	}
	return s.observed, s.err
}
func TestPipxRuntimeInspectorRejectsIntakeBeforeRunner(t *testing.T) {
	reader, artifact, _ := pipxInputFixture(t, nil)
	spy := &pipxInspectionSpy{}
	inspector := &PipxRuntimeInspector{static: reader, runner: spy}
	if _, err := inspector.Inspect(context.Background(), artifact); err == nil || spy.calls != 0 {
		t.Fatal("invalid intake reached runtime")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := inspector.Inspect(ctx, artifact); !errors.Is(err, context.Canceled) || spy.calls != 0 {
		t.Fatal("cancellation lost", err)
	}
	if _, err := NewPipxRuntimeInspector(t.TempDir(), nil); err == nil {
		t.Fatal("missing runner admitted")
	}
}

func TestPipxUnavailableCodeDoesNotChangeNativeCapability(t *testing.T) {
	artifact := dynamicArtifact(t)
	observed := result(t, domain.SandboxIncomplete, "P4_PIPX_RUNTIME_UNAVAILABLE", nil)
	for _, test := range []struct {
		name       string
		status     domain.ExecutionStatus
		capability domain.Capability
	}{
		{"github-release-elf-dynamic", domain.ExecutionIncomplete, domain.CapabilitySupported},
		{pipxRuntimeCheck, domain.ExecutionNotExecuted, domain.CapabilityUnsupported},
	} {
		report, err := githubDynamicReport(artifact, observed, test.name)
		if err != nil || report.Execution().Status() != test.status || report.Execution().Capability() != test.capability || len(report.Evidence()) != 0 {
			t.Fatalf("%s changed capability semantics: %+v %v", test.name, report.Execution(), err)
		}
	}
}
