package promotion

import (
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestStagingAcceptsOnlyCanonicalPythonSourceProfiles(t *testing.T) {
	profile, ok := artifactpypi.PyTorchProfile("cpu")
	if !ok {
		t.Fatal("CPU profile is missing")
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	identity, _ := domain.NewResolvedArtifactIdentity(profile.Source(), "torch", "2.9.1+cpu", "wheel")
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:wheel", 1)
	if err != nil {
		t.Fatal(err)
	}
	intakeName, stagedName, err := stagedArtifactNames(artifact)
	if err != nil || intakeName != "wheel.whl" || stagedName != digest.String()+".whl" {
		t.Fatalf("PyTorch staging names = %q/%q, %v", intakeName, stagedName, err)
	}
	unknown, _ := domain.NewSourceID("untrusted-python")
	identity, _ = domain.NewResolvedArtifactIdentity(unknown, "torch", "2.9.1", "wheel")
	artifact, _ = domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:wheel", 1)
	if _, _, err := stagedArtifactNames(artifact); err == nil {
		t.Fatal("staging accepted an unknown Python source")
	}
}
