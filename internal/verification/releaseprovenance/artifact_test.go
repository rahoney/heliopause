package releaseprovenance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectiongithub "github.com/rahoney/heliopause/internal/inspection/githubrelease"
	"github.com/rahoney/heliopause/internal/policy"
)

type manifestFixture struct {
	root     string
	body     []byte
	resolved domain.ResolvedArtifact
}

func (f manifestFixture) Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	return f.resolved, nil
}
func (f manifestFixture) Acquire(_ context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if resolved != f.resolved {
		return domain.AcquiredArtifact{}, errors.New("fixture selection changed")
	}
	dir := filepath.Join(f.root, run.String())
	if err := os.Mkdir(dir, 0700); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, "asset"), f.body, 0600); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	sum := sha256.Sum256(f.body)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), digest, "intake:"+run.String()+":github-release", uint64(len(f.body)), resolved.DeclaredIntegrity())
}

func publishedArtifact(t *testing.T) (manifestFixture, domain.ArtifactReference, domain.AcquiredArtifact) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0", "helox-release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("github-release")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "rahoney-heliopause", "v0.1.0", "helox-release-manifest.json")
	sum := sha256.Sum256(body)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	resolved, err := domain.NewResolvedArtifact(identity, "fixed-committed-fixture", "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	reference, err := domain.NewArtifactReference(source, "rahoney/heliopause@v0.1.0#helox-release-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	fixture := manifestFixture{t.TempDir(), body, resolved}
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := fixture.Acquire(context.Background(), run, resolved)
	if err != nil {
		t.Fatal(err)
	}
	return fixture, reference, artifact
}

func TestAuthenticityPortAndSchemaFeedExistingEvidenceAndPolicy(t *testing.T) {
	bundle, identity, _ := actualSignedRelease(t)
	for _, valid := range []bool{true, false} {
		name := "valid"
		if !valid {
			name = "invalid-proof"
		}
		t.Run(name, func(t *testing.T) {
			fixture, reference, _ := publishedArtifact(t)
			proof := append([]byte(nil), bundle...)
			if !valid {
				proof[0] = '!'
			}
			verifier, err := NewArtifactVerifier(identity, [][]byte{proof})
			if err != nil {
				t.Fatal(err)
			}
			// The caller cannot mutate a verifier's held proof after construction.
			proof[0] = '?'
			inspector, err := inspectiongithub.NewReleaseManifestInspector(fixture.root)
			if err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			service, err := application.NewInspectService(fixture, verifier, inspector, store, policy.M6{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			request, err := application.NewInspectRequest(reference)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Inspect(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			decision, ok := result.PolicyDecision()
			if !ok {
				t.Fatal("no Policy decision")
			}
			want := domain.DecisionAllow
			if !valid {
				want = domain.DecisionBlock
			}
			if decision.Decision() != want || len(result.Checks()) != 2 || len(result.Evidence()) != 2 {
				t.Fatalf("decision=%v checks=%d evidence=%d", decision.Decision(), len(result.Checks()), len(result.Evidence()))
			}
			if result.Checks()[0].ID().String() != "release-authenticity" || !result.Checks()[0].Required() {
				t.Fatal("authenticity check missing")
			}
		})
	}
}

func TestAuthenticityPortRejectsSubstitutionsAndPreservesCancellation(t *testing.T) {
	bundle, expected, _ := actualSignedRelease(t)
	_, _, artifact := publishedArtifact(t)
	cases := []struct {
		name     string
		identity Identity
		subject  domain.AcquiredArtifact
	}{
		{"wrong-source", Identity{expected.Version, strings.Repeat("a", 40), expected.WorkflowRun}, artifact},
		{"wrong-run", Identity{expected.Version, expected.SourceCommit, "1"}, artifact},
		{"wrong-tag", Identity{"v0.1.1", expected.SourceCommit, expected.WorkflowRun}, artifact},
	}
	source, _ := domain.NewSourceID("github-release")
	for _, change := range []string{"wrong-repository", "wrong-name", "wrong-digest", "wrong-declared"} {
		name, variant := "rahoney-heliopause", "helox-release-manifest.json"
		if change == "wrong-repository" {
			name = "other-heliopause"
		}
		if change == "wrong-name" {
			variant = "helox-linux-amd64"
		}
		id, err := domain.NewResolvedArtifactIdentity(source, name, expected.Version, variant)
		if err != nil {
			t.Fatal(err)
		}
		digest := artifact.Digest()
		declared, _ := artifact.DeclaredIntegrity()
		if change == "wrong-digest" {
			digest, _ = domain.NewSHA256Digest(strings.Repeat("b", 64))
			declared = "sha256:" + digest.String()
		}
		if change == "wrong-declared" {
			declared = "sha256:" + strings.Repeat("a", 64)
		}
		changed, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(id, digest, artifact.ContentHandle(), artifact.SizeBytes(), declared)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, struct {
			name     string
			identity Identity
			subject  domain.AcquiredArtifact
		}{change, expected, changed})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			verifier, err := NewArtifactVerifier(test.identity, [][]byte{bundle})
			if err != nil {
				t.Fatal(err)
			}
			report, err := verifier.Verify(context.Background(), test.subject)
			if err != nil || report.Outcome() != domain.VerificationMismatch || len(report.Findings()) != 1 || len(report.Evidence()) != 1 {
				t.Fatalf("report=%v error=%v", report, err)
			}
		})
	}
	verifier, err := NewArtifactVerifier(expected, [][]byte{[]byte("invalid"), bundle})
	if err != nil {
		t.Fatal(err)
	}
	report, err := verifier.Verify(context.Background(), artifact)
	if err != nil || report.Outcome() != domain.VerificationVerified {
		t.Fatalf("valid proof after invalid=%v %v", report, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := verifier.Verify(ctx, artifact); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation cause lost")
	}
	if _, err := NewArtifactVerifier(Identity{}, [][]byte{bundle}); err == nil {
		t.Fatal("invalid identity accepted")
	}
	for _, proofs := range [][][]byte{nil, {nil}, {make([]byte, maxBundle+1)}, make([][]byte, 17)} {
		if _, err := NewArtifactVerifier(expected, proofs); err == nil {
			t.Fatal("invalid proofs accepted")
		}
	}
}
