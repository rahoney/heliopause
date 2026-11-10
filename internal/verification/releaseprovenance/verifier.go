// Package releaseprovenance verifies official Heliopause release attestations.
// Provenance is not a content inspection or a Policy decision.
package releaseprovenance

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"golang.org/x/mod/semver"
)

const (
	repository  = "https://github.com/rahoney/heliopause"
	workflow    = ".github/workflows/heliopause-release-build.yml"
	maxBundle   = 1 << 20
	maxSubjects = 64
)

//go:embed trusted-root.json
var publicRoot []byte

var (
	hexSHA256      = regexp.MustCompile(`^[0-9a-f]{64}$`)
	hexCommit      = regexp.MustCompile(`^[0-9a-f]{40}$`)
	runNumber      = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
	assetName      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	releaseVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
)

// Identity is the exact release selected before verification. Its values must
// also match the signed manifest; unsigned discovery metadata is not authority.
type Identity struct {
	Version      string
	SourceCommit string
	WorkflowRun  string
}

type Subject struct {
	Name   string
	SHA256 string
}

// Receipt contains authenticated provenance facts, never permission to install.
// It can only be constructed here after cryptographic verification.
type Receipt struct {
	identity     Identity
	subjects     []Subject
	bundleSHA256 string
	invocation   string
}

func (r Receipt) Identity() Identity   { return r.identity }
func (r Receipt) Subjects() []Subject  { return append([]Subject(nil), r.subjects...) }
func (r Receipt) BundleSHA256() string { return r.bundleSHA256 }
func (r Receipt) Invocation() string   { return r.invocation }

// Verify uses fixed offline trust material and exact certificate/subject policy.
// It neither acquires content nor runs a verifier from the release or PATH.
func Verify(ctx context.Context, body []byte, expected Identity, subjects []Subject) (Receipt, error) {
	if ctx == nil {
		return Receipt{}, errors.New("release provenance context is required")
	}
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	if len(body) == 0 || len(body) > maxBundle || !validIdentity(expected) || !validSubjects(subjects) {
		return Receipt{}, errors.New("release provenance input is invalid")
	}
	trusted, err := root.NewTrustedRootFromJSON(publicRoot)
	if err != nil {
		return Receipt{}, errors.New("release provenance trust root is unavailable")
	}
	verifier, err := verify.NewVerifier(trusted, verify.WithSignedCertificateTimestamps(1), verify.WithTransparencyLog(1), verify.WithObserverTimestamps(1))
	if err != nil {
		return Receipt{}, errors.New("release provenance verifier is unavailable")
	}
	var signed bundle.Bundle
	if err := signed.UnmarshalJSON(body); err != nil {
		return Receipt{}, errors.New("release attestation bundle is invalid")
	}
	ref := "refs/tags/" + expected.Version
	signer := repository + "/" + workflow + "@" + ref
	san, err := verify.NewSANMatcher(signer, "")
	if err != nil {
		return Receipt{}, errors.New("release signer identity is invalid")
	}
	issuer, err := verify.NewIssuerMatcher("https://token.actions.githubusercontent.com", "")
	if err != nil {
		return Receipt{}, errors.New("release issuer identity is invalid")
	}
	identity, err := verify.NewCertificateIdentity(san, issuer, certificate.Extensions{
		BuildSignerURI: signer, BuildSignerDigest: expected.SourceCommit,
		RunnerEnvironment: "github-hosted", SourceRepositoryURI: repository,
		SourceRepositoryDigest: expected.SourceCommit, SourceRepositoryRef: ref,
		SourceRepositoryIdentifier: "1331904030", SourceRepositoryOwnerURI: "https://github.com/rahoney",
		SourceRepositoryOwnerIdentifier: "168336203", BuildConfigURI: signer,
		BuildConfigDigest: expected.SourceCommit, BuildTrigger: "push",
		SourceRepositoryVisibilityAtSigning: "public",
	})
	if err != nil {
		return Receipt{}, errors.New("release certificate policy is invalid")
	}
	digest, err := hex.DecodeString(subjects[0].SHA256)
	if err != nil {
		return Receipt{}, errors.New("release subject digest is invalid")
	}
	verified, err := verifier.Verify(&signed, verify.NewPolicy(verify.WithArtifactDigest("sha256", digest), verify.WithCertificateIdentity(identity)))
	if err != nil || verified == nil || verified.Signature == nil || verified.Signature.Certificate == nil || verified.Statement == nil {
		return Receipt{}, errors.New("release attestation verification failed")
	}
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	invocation := verified.Signature.Certificate.RunInvocationURI
	prefix := repository + "/actions/runs/" + expected.WorkflowRun + "/attempts/"
	attempt, found := strings.CutPrefix(invocation, prefix)
	if !found || !runNumber.MatchString(attempt) {
		return Receipt{}, errors.New("release workflow invocation does not match")
	}
	statement := verified.Statement
	if statement.GetType() != "https://in-toto.io/Statement/v1" || statement.GetPredicateType() != "https://slsa.dev/provenance/v1" || len(statement.GetSubject()) == 0 || len(statement.GetSubject()) > maxSubjects {
		return Receipt{}, errors.New("release attestation statement is invalid")
	}
	actual := make(map[string]string, len(statement.GetSubject()))
	for _, item := range statement.GetSubject() {
		if item == nil || !assetName.MatchString(item.GetName()) || len(item.GetDigest()) != 1 || !hexSHA256.MatchString(item.GetDigest()["sha256"]) || actual[item.GetName()] != "" {
			return Receipt{}, errors.New("release attestation subject set is invalid")
		}
		actual[item.GetName()] = item.GetDigest()["sha256"]
	}
	for _, wanted := range subjects {
		if actual[wanted.Name] != wanted.SHA256 {
			return Receipt{}, errors.New("release attestation subject does not match")
		}
	}
	predicate, err := json.Marshal(statement.GetPredicate())
	if err != nil || !validPredicate(predicate, expected, invocation, signer) {
		return Receipt{}, errors.New("release build provenance does not match")
	}
	sum := sha256.Sum256(body)
	return Receipt{identity: expected, subjects: append([]Subject(nil), subjects...), bundleSHA256: hex.EncodeToString(sum[:]), invocation: invocation}, nil
}

func validIdentity(value Identity) bool {
	return len(value.Version) <= 128 && releaseVersion.MatchString(value.Version) && semver.IsValid(value.Version) && hexCommit.MatchString(value.SourceCommit) && runNumber.MatchString(value.WorkflowRun)
}

func validSubjects(subjects []Subject) bool {
	if len(subjects) == 0 || len(subjects) > maxSubjects {
		return false
	}
	seen := make(map[string]bool, len(subjects))
	for _, item := range subjects {
		if !assetName.MatchString(item.Name) || !hexSHA256.MatchString(item.SHA256) || seen[item.Name] {
			return false
		}
		seen[item.Name] = true
	}
	return true
}

func validPredicate(body []byte, expected Identity, invocation, signer string) bool {
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
	ref := "refs/tags/" + expected.Version
	github := build.InternalParameters.GitHub
	if build.BuildType != "https://actions.github.io/buildtypes/workflow/v1" || build.ExternalParameters.Workflow.Ref != ref || build.ExternalParameters.Workflow.Repository != repository || build.ExternalParameters.Workflow.Path != workflow || github.EventName != "push" || github.RepositoryID != "1331904030" || github.RepositoryOwnerID != "168336203" || github.RunnerEnvironment != "github-hosted" || len(build.ResolvedDependencies) != 1 {
		return false
	}
	dependency := build.ResolvedDependencies[0]
	return dependency.URI == "git+"+repository+"@"+ref && len(dependency.Digest) == 1 && dependency.Digest["gitCommit"] == expected.SourceCommit && predicate.RunDetails.Builder.ID == signer && predicate.RunDetails.Metadata.InvocationID == invocation
}
