package domain_test

import (
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
