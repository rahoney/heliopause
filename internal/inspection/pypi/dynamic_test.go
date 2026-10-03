package pypi

import (
	"context"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestDynamicInspectorUsesOnlyStaticImportNames(t *testing.T) {
	artifact := pypiWheelArtifact(t)
	runner := &wheelRunner{result: completedResult(t)}
	inspector, err := NewDynamicInspector(runner)
	if err != nil {
		t.Fatal(err)
	}
	static := artifactpypi.WheelInspection{Project: "example", Version: "1.0", ImportNames: []string{"example"}}
	report, err := inspector.InspectWheel(context.Background(), artifact, static)
	if err != nil || report.Execution().Status() != domain.ExecutionCompleted || strings.Join(runner.imports, ",") != "example" {
		t.Fatalf("InspectWheel() = %#v, %v; imports=%q", report, err, runner.imports)
	}
}

func TestDynamicInspectorDistinguishesProvenNoImportFromUnknown(t *testing.T) {
	artifact := pypiWheelArtifact(t)
	runner := &noImportWheelRunner{result: completedResult(t)}
	inspector, err := NewDynamicInspector(runner)
	if err != nil {
		t.Fatal(err)
	}
	static := artifactpypi.WheelInspection{Project: "example", Version: "1.0", NoImportSurface: true,
		Surface: artifactpypi.RuntimeSurface{InstalledFiles: []artifactpypi.InstalledFile{{ArchivePath: "example-1.0.dist-info/METADATA", Scheme: artifactpypi.SchemeDistInfo, Destination: "example-1.0.dist-info/METADATA", Role: artifactpypi.RuntimeRoleMetadata}}}}
	report, err := inspector.InspectWheel(context.Background(), artifact, static)
	if err != nil || runner.noImportCalls != 1 || runner.importCalls != 0 || report.Executions()[0].ID().String() != "pypi-dynamic-import-not-applicable" || len(report.Evidence()) != 1 {
		t.Fatalf("proven no-import report = %#v, %v; runner = %#v", report, err, runner)
	}
	static.NoImportSurface = false
	if _, err := inspector.InspectWheel(context.Background(), artifact, static); err == nil {
		t.Fatal("unknown empty import surface accepted")
	}
	static.NoImportSurface = true
	static.ImportNames = []string{"example"}
	if _, err := inspector.InspectWheel(context.Background(), artifact, static); err == nil {
		t.Fatal("ambiguous no-import surface accepted")
	}
}

func TestDynamicInspectorScriptOnlyNeverQualifiesAsNoImport(t *testing.T) {
	artifact := pypiWheelArtifact(t)
	runner := &noImportWheelRunner{result: completedResult(t)}
	inspector, err := NewDynamicInspector(runner)
	if err != nil {
		t.Fatal(err)
	}
	static := artifactpypi.WheelInspection{Project: "example", Version: "1.0", Surface: artifactpypi.RuntimeSurface{Scripts: []string{"bin/tool"}}}
	report, err := inspector.InspectWheel(context.Background(), artifact, static)
	if err != nil || report.Execution().Status() != domain.ExecutionIncomplete || runner.noImportCalls != 0 || runner.importCalls != 0 {
		t.Fatalf("script-only inspection = %#v, %v, runner %#v", report, err, runner)
	}
}

type noImportWheelRunner struct {
	result        domain.SandboxResult
	noImportCalls int
	importCalls   int
}

func (r *noImportWheelRunner) InspectWheel(context.Context, domain.AcquiredArtifact, []string) (domain.SandboxResult, error) {
	r.importCalls++
	return r.result, nil
}

func (r *noImportWheelRunner) InspectWheelWithoutImportSurface(context.Context, domain.AcquiredArtifact, []domain.AcquiredArtifact) (domain.SandboxResult, error) {
	r.noImportCalls++
	return r.result, nil
}

func TestDynamicInspectorNormalizesFindingsAndBoundedSummary(t *testing.T) {
	artifact := pypiWheelArtifact(t)
	observations := []domain.SandboxObservation{
		observation(t, domain.ObservationHoneytoken, "honeytoken-access"),
		observation(t, domain.ObservationNetwork, "network-attempt"),
		observation(t, domain.ObservationProcess, "process-exec-unexpected"),
		observation(t, domain.ObservationFilesystem, "filesystem-outside-workspace"),
	}
	inspector, err := NewDynamicInspector(&wheelRunner{result: completedResultWithObservations(t, observations)})
	if err != nil {
		t.Fatal(err)
	}
	report, err := inspector.InspectWheel(context.Background(), artifact, artifactpypi.WheelInspection{Project: "example", Version: "1.0", ImportNames: []string{"example"}})
	if err != nil || report.Execution().Status() != domain.ExecutionCompleted || len(report.Evidence()) != 1 {
		t.Fatalf("report = %#v, %v", report, err)
	}
	if summary := report.Evidence()[0].Summary(); !strings.Contains(summary, `"schema":"m11-004"`) || !strings.Contains(summary, `"total":4`) {
		t.Fatalf("Evidence summary = %q", summary)
	}
	got := make([]string, 0, len(report.Findings()))
	for _, finding := range report.Findings() {
		got = append(got, finding.Code())
	}
	if strings.Join(got, ",") != "M3_HONEYTOKEN_ACCESS,M3_NETWORK_ATTEMPT,M3_UNEXPECTED_PROCESS,M3_FILESYSTEM_VIOLATION" {
		t.Fatalf("findings = %q", got)
	}
}

// Expected runtime operations are retained telemetry, not a grant that hides
// later artifact behavior or completes another required check.
func TestDynamicInspectorExpectedQueryDoesNotSuppressLaterFindings(t *testing.T) {
	for _, test := range []struct {
		name, subject, code string
		category            domain.ObservationCategory
	}{
		{"query only", "", "", domain.ObservationProcess},
		{"later exec", "process-exec-unexpected", "M3_UNEXPECTED_PROCESS", domain.ObservationProcess},
		{"later network", "network-attempt", "M3_NETWORK_ATTEMPT", domain.ObservationNetwork},
		{"later write", "filesystem-outside-workspace", "M3_FILESYSTEM_VIOLATION", domain.ObservationFilesystem},
	} {
		t.Run(test.name, func(t *testing.T) {
			observations := []domain.SandboxObservation{observation(t, domain.ObservationProcess, "process-exec-expected")}
			if test.subject != "" {
				observations = append(observations, observation(t, test.category, test.subject))
			}
			inspector, _ := NewDynamicInspector(&wheelRunner{result: completedResultWithObservations(t, observations)})
			report, err := inspector.InspectWheel(context.Background(), pypiWheelArtifact(t), artifactpypi.WheelInspection{Project: "example", Version: "1.0", ImportNames: []string{"example"}})
			if err != nil || report.Execution().Status() != domain.ExecutionCompleted {
				t.Fatalf("report: %v", err)
			}
			if test.code == "" {
				if len(report.Findings()) != 0 {
					t.Fatal(report.Findings())
				}
				return
			}
			if len(report.Findings()) != 1 || report.Findings()[0].Code() != test.code {
				t.Fatal(report.Findings())
			}
		})
	}
}

func TestDynamicInspectorFailsClosedOnIncompleteSandbox(t *testing.T) {
	artifact := pypiWheelArtifact(t)
	session, _ := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	result, _ := domain.NewSandboxResult(session, domain.SandboxIncomplete, "M5_PYPI_DYNAMIC_OBSERVATION_INCOMPLETE", nil)
	inspector, _ := NewDynamicInspector(&wheelRunner{result: result})
	report, err := inspector.InspectWheel(context.Background(), artifact, artifactpypi.WheelInspection{Project: "example", Version: "1.0", ImportNames: []string{"example"}})
	if err != nil || report.Execution().Status() != domain.ExecutionIncomplete {
		t.Fatalf("report = %#v, %v", report, err)
	}
}

func TestDynamicInspectorAcceptsDerivedWheelForReinspection(t *testing.T) {
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "derived-wheel")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, _ := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:derived-wheel", 1)
	inspector, err := NewDynamicInspector(&wheelRunner{result: completedResult(t)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inspector.InspectWheel(context.Background(), artifact, artifactpypi.WheelInspection{Project: "example", Version: "1.0", ImportNames: []string{"example"}}); err != nil {
		t.Fatalf("derived wheel reinspection rejected: %v", err)
	}
}

type wheelRunner struct {
	result  domain.SandboxResult
	err     error
	imports []string
}

func (r *wheelRunner) InspectWheel(_ context.Context, _ domain.AcquiredArtifact, imports []string) (domain.SandboxResult, error) {
	r.imports = append([]string(nil), imports...)
	return r.result, r.err
}
func completedResult(t *testing.T) domain.SandboxResult {
	return completedResultWithObservations(t, nil)
}

func completedResultWithObservations(t *testing.T, observations []domain.SandboxObservation) domain.SandboxResult {
	t.Helper()
	session, err := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	result, err := domain.NewSandboxResult(session, domain.SandboxCompleted, "", observations)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func pypiWheelArtifact(t *testing.T) domain.AcquiredArtifact {
	t.Helper()
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "wheel")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:wheel", 1)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func observation(t *testing.T, category domain.ObservationCategory, subject string) domain.SandboxObservation {
	t.Helper()
	value, err := domain.NewSandboxObservation(category, subject)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
