package cargo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type fixtureRegistry struct {
	checksum string
	err      error
	calls    int
}

func (r *fixtureRegistry) LookupChecksum(_ context.Context, name, version string) (string, error) {
	r.calls++
	if name != "fixture" || version != "1.2.3" {
		return "", errors.New("wrong exact lookup")
	}
	return r.checksum, r.err
}

func TestCargoIntegrityRequiresThreeMatchingIdentities(t *testing.T) {
	body := []byte("exact crate content")
	digest := sha256.Sum256(body)
	checksum := hex.EncodeToString(digest[:])
	for _, fault := range []string{"normal", "lock mismatch", "index mismatch", "missing declaration", "lookup failure", "intake tamper", "cancelled"} {
		t.Run(fault, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "intake")
			run, _ := domain.NewRunID()
			directory := filepath.Join(root, run.String())
			if err := os.MkdirAll(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "package.crate"), body, 0o600); err != nil {
				t.Fatal(err)
			}
			identity, _ := domain.NewResolvedArtifactIdentity(artifactcargo.Source(), "fixture", "1.2.3", "crate")
			observed, _ := domain.NewSHA256Digest(checksum)
			declared := "sha256=" + checksum
			registry := &fixtureRegistry{checksum: checksum}
			expected := domain.VerificationVerified
			operational := false
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch fault {
			case "lock mismatch":
				declared = "sha256=" + strings.Repeat("a", 64)
				expected = domain.VerificationMismatch
			case "index mismatch":
				registry.checksum = strings.Repeat("a", 64)
				expected = domain.VerificationMismatch
			case "missing declaration":
				declared = ""
				expected = domain.VerificationMismatch
			case "lookup failure":
				registry.err = errors.New("unavailable")
				operational = true
			case "intake tamper":
				if err := os.WriteFile(filepath.Join(directory, "package.crate"), []byte("changed content now"), 0o600); err != nil {
					t.Fatal(err)
				}
				operational = true
			case "cancelled":
				cancel()
				operational = true
			}
			artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, observed, "intake:"+run.String()+":crate", uint64(len(body)), declared)
			if err != nil {
				t.Fatal(err)
			}
			verifier := &IntegrityVerifier{intakeRoot: root, registry: registry}
			report, err := verifier.Verify(ctx, artifact)
			if operational {
				if err == nil || report.Outcome() != domain.VerificationOutcome("") {
					t.Fatal("operational failure became verification success/result")
				}
				if fault == "cancelled" && !errors.Is(err, context.Canceled) {
					t.Fatal("cancellation cause lost")
				}
				return
			}
			if err != nil || report.Outcome() != expected || !report.Execution().Required() || len(report.Evidence()) != 1 {
				t.Fatalf("report=%s err=%v", report.Outcome(), err)
			}
			if fault == "lock mismatch" || fault == "missing declaration" {
				if registry.calls != 0 {
					t.Fatal("invalid declared binding reached network")
				}
			} else if registry.calls != 1 {
				t.Fatal("official checksum was not independently checked")
			}
			if expected == domain.VerificationMismatch && len(report.Findings()) != 1 {
				t.Fatal("mismatch finding lost")
			}
		})
	}
}
