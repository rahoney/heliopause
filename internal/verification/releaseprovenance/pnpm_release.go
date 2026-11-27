package releaseprovenance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"golang.org/x/mod/semver"
)

const (
	pnpmRepository = "https://github.com/pnpm/pnpm"
	pnpmWorkflow   = ".github/workflows/release.yml"
	pnpmArchive    = "pnpm-linux-x64.tar.gz"
)

// PNPMReleaseIdentity is an untrusted discovery request for the Linux amd64
// release archive. No runtime, installation or content approval is implied.
type PNPMReleaseIdentity struct {
	Version       string `json:"version"`
	SourceCommit  string `json:"source_commit"`
	WorkflowRun   string `json:"workflow_run"`
	ArchiveSHA256 string `json:"archive_sha256"`
}

// PNPMReleaseReceipt binds an archive digest to the exact upstream build.
// Its caller must separately compare acquired bytes and qualify the runtime.
type PNPMReleaseReceipt struct {
	identity     PNPMReleaseIdentity
	bundleSHA256 string
	invocation   string
}

func (r PNPMReleaseReceipt) Identity() PNPMReleaseIdentity { return r.identity }
func (r PNPMReleaseReceipt) BundleSHA256() string          { return r.bundleSHA256 }
func (r PNPMReleaseReceipt) Invocation() string            { return r.invocation }

// VerifyPNPMRelease authenticates only pnpm's fixed release publisher policy.
// The upstream self-hosted build identity does not alter HAA or UV policies.
// Verification uses the existing pinned public root, entirely offline.
func VerifyPNPMRelease(ctx context.Context, body []byte, expected PNPMReleaseIdentity) (PNPMReleaseReceipt, error) {
	if ctx == nil {
		return PNPMReleaseReceipt{}, errors.New("pnpm release context is required")
	}
	if err := ctx.Err(); err != nil {
		return PNPMReleaseReceipt{}, err
	}
	version := "v" + expected.Version
	if len(body) == 0 || len(body) > maxBundle || len(expected.Version) > 64 || !releaseVersion.MatchString(version) || !semver.IsValid(version) || semver.Prerelease(version) != "" || semver.Build(version) != "" || !hexCommit.MatchString(expected.SourceCommit) || !runNumber.MatchString(expected.WorkflowRun) || !hexSHA256.MatchString(expected.ArchiveSHA256) {
		return PNPMReleaseReceipt{}, errors.New("pnpm release identity or proof is invalid")
	}
	trusted, err := root.NewTrustedRootFromJSON(publicRoot)
	if err != nil {
		return PNPMReleaseReceipt{}, errors.New("pnpm release trust root is unavailable")
	}
	verifier, err := verify.NewVerifier(trusted, verify.WithSignedCertificateTimestamps(1), verify.WithTransparencyLog(1), verify.WithObserverTimestamps(1))
	if err != nil {
		return PNPMReleaseReceipt{}, errors.New("pnpm release verifier is unavailable")
	}
	var signed bundle.Bundle
	if err := signed.UnmarshalJSON(body); err != nil {
		return PNPMReleaseReceipt{}, errors.New("pnpm release bundle is invalid")
	}
	ref := "refs/tags/" + version
	signer := pnpmRepository + "/" + pnpmWorkflow + "@" + ref
	san, err := verify.NewSANMatcher(signer, "")
	if err != nil {
		return PNPMReleaseReceipt{}, err
	}
	issuer, err := verify.NewIssuerMatcher("https://token.actions.githubusercontent.com", "")
	if err != nil {
		return PNPMReleaseReceipt{}, err
	}
	identity, err := verify.NewCertificateIdentity(san, issuer, certificate.Extensions{
		BuildSignerURI: signer, BuildSignerDigest: expected.SourceCommit,
		RunnerEnvironment: "self-hosted", SourceRepositoryURI: pnpmRepository,
		SourceRepositoryDigest: expected.SourceCommit, SourceRepositoryRef: ref,
		SourceRepositoryIdentifier: "50565430", SourceRepositoryOwnerURI: "https://github.com/pnpm",
		SourceRepositoryOwnerIdentifier: "21320719", BuildConfigURI: signer,
		BuildConfigDigest: expected.SourceCommit, BuildTrigger: "push",
		SourceRepositoryVisibilityAtSigning: "public",
	})
	if err != nil {
		return PNPMReleaseReceipt{}, errors.New("pnpm release certificate policy is invalid")
	}
	digest, _ := hex.DecodeString(expected.ArchiveSHA256)
	verified, err := verifier.Verify(&signed, verify.NewPolicy(verify.WithArtifactDigest("sha256", digest), verify.WithCertificateIdentity(identity)))
	if err != nil || verified == nil || verified.Signature == nil || verified.Signature.Certificate == nil || verified.Statement == nil {
		return PNPMReleaseReceipt{}, errors.New("pnpm release attestation verification failed")
	}
	if err := ctx.Err(); err != nil {
		return PNPMReleaseReceipt{}, err
	}
	invocation := verified.Signature.Certificate.RunInvocationURI
	attempt, found := strings.CutPrefix(invocation, pnpmRepository+"/actions/runs/"+expected.WorkflowRun+"/attempts/")
	statement := verified.Statement
	if !found || !runNumber.MatchString(attempt) || statement.GetType() != "https://in-toto.io/Statement/v1" || statement.GetPredicateType() != "https://slsa.dev/provenance/v1" || len(statement.GetSubject()) != 1 {
		return PNPMReleaseReceipt{}, errors.New("pnpm release statement or invocation mismatch")
	}
	subject := statement.GetSubject()[0]
	if subject == nil || subject.GetName() != pnpmArchive || len(subject.GetDigest()) != 1 || subject.GetDigest()["sha256"] != expected.ArchiveSHA256 {
		return PNPMReleaseReceipt{}, errors.New("pnpm release signed archive mismatch")
	}
	predicate, err := json.Marshal(statement.GetPredicate())
	if err != nil || !validPNPMPredicate(predicate, expected, invocation, signer) {
		return PNPMReleaseReceipt{}, errors.New("pnpm release build provenance mismatch")
	}
	sum := sha256.Sum256(body)
	return PNPMReleaseReceipt{expected, hex.EncodeToString(sum[:]), invocation}, nil
}

func validPNPMPredicate(body []byte, expected PNPMReleaseIdentity, invocation, signer string) bool {
	var predicate struct {
		BuildDefinition struct {
			BuildType          string
			ExternalParameters struct {
				Workflow struct{ Ref, Repository, Path string }
			}
			InternalParameters struct {
				GitHub struct {
					EventName         string `json:"event_name"`
					RepositoryID      string `json:"repository_id"`
					RepositoryOwnerID string `json:"repository_owner_id"`
					RunnerEnvironment string `json:"runner_environment"`
				}
			}
			ResolvedDependencies []struct {
				URI    string
				Digest map[string]string
			}
		}
		RunDetails struct {
			Builder  struct{ ID string }
			Metadata struct{ InvocationID string }
		}
	}
	if json.Unmarshal(body, &predicate) != nil {
		return false
	}
	build := predicate.BuildDefinition
	ref := "refs/tags/v" + expected.Version
	github := build.InternalParameters.GitHub
	if build.BuildType != "https://actions.github.io/buildtypes/workflow/v1" || build.ExternalParameters.Workflow.Ref != ref || build.ExternalParameters.Workflow.Repository != pnpmRepository || build.ExternalParameters.Workflow.Path != pnpmWorkflow || github.EventName != "push" || github.RepositoryID != "50565430" || github.RepositoryOwnerID != "21320719" || github.RunnerEnvironment != "self-hosted" || len(build.ResolvedDependencies) != 1 {
		return false
	}
	dependency := build.ResolvedDependencies[0]
	return dependency.URI == "git+"+pnpmRepository+"@"+ref && len(dependency.Digest) == 1 && dependency.Digest["gitCommit"] == expected.SourceCommit && predicate.RunDetails.Builder.ID == signer && predicate.RunDetails.Metadata.InvocationID == invocation
}
