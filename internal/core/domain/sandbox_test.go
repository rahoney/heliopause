package domain_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestSandboxResultSeparatesRawObservationFromVerdict(t *testing.T) {
	t.Parallel()

	sessionID, err := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	observation, err := domain.NewSandboxObservation(domain.ObservationHoneytoken, "synthetic-ssh-key")
	if err != nil {
		t.Fatal(err)
	}
	result, err := domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "", []domain.SandboxObservation{observation})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status() != domain.SandboxCompleted || result.SessionID() != sessionID || len(result.Observations()) != 1 {
		t.Fatalf("Sandbox result = %#v", result)
	}
	if _, err := domain.NewSandboxResult(sessionID, domain.SandboxIncomplete, "", nil); err == nil {
		t.Fatal("incomplete Sandbox result accepted without limitation")
	}
	if _, err := domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "M3_TIMEOUT", nil); err == nil {
		t.Fatal("completed Sandbox result accepted with limitation")
	}
}

func TestSandboxObservationSummaryIsBoundedAndDoesNotRetainPayloads(t *testing.T) {
	t.Parallel()
	sessionID, err := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	observations := []domain.SandboxObservation{
		mustObservation(t, domain.ObservationProcess, "process-exec-expected"),
		mustObservation(t, domain.ObservationProcess, "process-exec-expected"),
		mustObservation(t, domain.ObservationHoneytoken, "honeytoken-access"),
	}
	result, err := domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "", observations)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := result.ObservationSummary()
	if err != nil || !strings.Contains(summary, `"schema":"m11-004"`) || !strings.Contains(summary, `"total":3`) {
		t.Fatalf("summary = %q, err = %v", summary, err)
	}
	if strings.Contains(summary, "/") || strings.Contains(summary, "argv") || strings.Contains(summary, "pathname") {
		t.Fatalf("summary retained an unnormalized payload: %q", summary)
	}

	tooMany := make([]domain.SandboxObservation, 0, 33)
	for index := 0; index < 33; index++ {
		tooMany = append(tooMany, mustObservation(t, domain.ObservationProcess, fmt.Sprintf("subject-%02d", index)))
	}
	result, err = domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "", tooMany)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := result.ObservationSummary(); err == nil {
		t.Fatal("summary accepted more than the unique-subject bound")
	}

	repeated := make([]domain.SandboxObservation, 10001)
	for index := range repeated {
		repeated[index] = mustObservation(t, domain.ObservationNetwork, "network-attempt")
	}
	result, err = domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "", repeated)
	if err != nil {
		t.Fatal(err)
	}
	summary, err = result.ObservationSummary()
	if err != nil || !strings.Contains(summary, `"total":10000`) || !strings.Contains(summary, `"NETWORK:network-attempt":10000`) {
		t.Fatalf("summary count did not saturate: %q, err = %v", summary, err)
	}
}

func TestCountedSandboxObservationIsBoundedAndSummaryPreservesCount(t *testing.T) {
	t.Parallel()
	sessionID, err := domain.ParseSandboxSessionID("sbx_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := domain.NewCountedSandboxObservation(domain.ObservationFilesystem, "filesystem-workspace-access", 1153)
	if err != nil || workspace.Count() != 1153 {
		t.Fatalf("workspace = %#v, %v", workspace, err)
	}
	duplicate, err := domain.NewCountedSandboxObservation(domain.ObservationFilesystem, "filesystem-workspace-access", 9000)
	if err != nil {
		t.Fatal(err)
	}
	result, err := domain.NewSandboxResult(sessionID, domain.SandboxCompleted, "", []domain.SandboxObservation{workspace, duplicate})
	if err != nil {
		t.Fatal(err)
	}
	if observations := result.Observations(); len(observations) != 1 || observations[0].Count() != domain.MaximumObservationSummaryCount {
		t.Fatalf("observations = %#v", observations)
	}
	summary, err := result.ObservationSummary()
	if err != nil || !strings.Contains(summary, `"FILESYSTEM:filesystem-workspace-access":10000`) {
		t.Fatalf("summary = %q, %v", summary, err)
	}
	for _, count := range []uint64{0, domain.MaximumObservationSummaryCount + 1} {
		if _, err := domain.NewCountedSandboxObservation(domain.ObservationFilesystem, "filesystem-workspace-access", count); err == nil {
			t.Fatalf("count %d was accepted", count)
		}
	}
}

func mustObservation(t *testing.T, category domain.ObservationCategory, subject string) domain.SandboxObservation {
	t.Helper()
	observation, err := domain.NewSandboxObservation(category, subject)
	if err != nil {
		t.Fatal(err)
	}
	return observation
}

func TestSandboxRequestRequiresAcquiredContent(t *testing.T) {
	t.Parallel()
	if _, err := domain.NewSandboxRequest(domain.AcquiredArtifact{}); err == nil {
		t.Fatal("zero Artifact accepted")
	}
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.NewSandboxRequest(artifact); err != nil {
		t.Fatal(err)
	}
}
