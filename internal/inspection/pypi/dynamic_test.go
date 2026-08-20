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
	t.Helper()
	session, err := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	result, err := domain.NewSandboxResult(session, domain.SandboxCompleted, "", nil)
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
