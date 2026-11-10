package releaseprovenance

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"golang.org/x/mod/semver"
)

const pipxRepository = "https://github.com/pypa/pipx"
const pipxPublishPredicate = "https://docs.pypi.org/attestations/publish/v1"

// PipxPublishIdentity is an untrusted request for a single published wheel.
// It carries no dependency, runtime or installation authority.
type PipxPublishIdentity struct {
	Version      string `json:"version"`
	SourceCommit string `json:"source_commit"`
	WorkflowRun  string `json:"workflow_run"`
	WheelSHA256  string `json:"wheel_sha256"`
}

func (i PipxPublishIdentity) valid() bool {
	v := "v" + i.Version
	return len(i.Version) <= 64 && releaseVersion.MatchString(v) && semver.IsValid(v) && semver.Prerelease(v) == "" && semver.Build(v) == "" && hexCommit.MatchString(i.SourceCommit) && runNumber.MatchString(i.WorkflowRun) && hexSHA256.MatchString(i.WheelSHA256)
}

// WheelFilename derives the only supported subject, or returns empty for an
// invalid identity. Callers never use a proof-supplied filename as a path.
func (i PipxPublishIdentity) WheelFilename() string {
	if !i.valid() {
		return ""
	}
	return "pipx-" + i.Version + "-py3-none-any.whl"
}

func (i *PipxPublishIdentity) UnmarshalJSON(body []byte) error {
	if i == nil {
		return errPipxJSON
	}
	*i = PipxPublishIdentity{}
	if err := boundPipxJSON(body, 4096); err != nil {
		return err
	}
	object, err := pipxObject(body, "version", "source_commit", "workflow_run", "wheel_sha256")
	if err != nil {
		return err
	}
	value := PipxPublishIdentity{pipxString(object["version"]), pipxString(object["source_commit"]), pipxString(object["workflow_run"]), pipxString(object["wheel_sha256"])}
	if !value.valid() {
		return errPipxJSON
	}
	*i = value
	return nil
}

// PipxPublishReceipt authenticates publish provenance only. The caller must
// compare acquired wheel bytes and separately qualify its closure and runtime.
type PipxPublishReceipt struct {
	identity         PipxPublishIdentity
	provenanceSHA256 string
	invocation       string
}

func (r PipxPublishReceipt) Identity() PipxPublishIdentity { return r.identity }
func (r PipxPublishReceipt) ProvenanceSHA256() string      { return r.provenanceSHA256 }
func (r PipxPublishReceipt) Invocation() string            { return r.invocation }

// VerifyPipxPublish verifies the fixed upstream pipx GitHub publisher, entirely
// offline, without extending the existing HAA/UV/PNPM build provenance policies.
func VerifyPipxPublish(ctx context.Context, body []byte, expected PipxPublishIdentity) (PipxPublishReceipt, error) {
	if !expected.valid() {
		return PipxPublishReceipt{}, errors.New("pipx publish identity is invalid")
	}
	proof, err := verifyPyPIPublish(ctx, body, pypiPublishRequest{expected.SourceCommit, expected.WorkflowRun, expected.WheelSHA256, expected.WheelFilename()}, pipxPublisherPolicy(expected.Version))
	if err != nil {
		return PipxPublishReceipt{}, err
	}
	return PipxPublishReceipt{expected, proof.provenanceSHA256, proof.invocation}, nil
}

// Policy is private: only the fixed upstream entry points select a publisher.
type pypiPublisherPolicy struct {
	repository, workflow, ref, repositoryID, owner, ownerID, trigger, environment string
}

type pypiPublishRequest struct{ SourceCommit, WorkflowRun, WheelSHA256, Filename string }
type pypiPublishProof struct{ provenanceSHA256, invocation string }

func pipxPublisherPolicy(version string) pypiPublisherPolicy {
	return pypiPublisherPolicy{pipxRepository, "release.yml", "refs/tags/" + version, "151871286", "pypa", "647025", "push", "release"}
}

func verifyPyPIPublish(ctx context.Context, body []byte, expected pypiPublishRequest, policy pypiPublisherPolicy) (pypiPublishProof, error) {
	if ctx == nil {
		return pypiPublishProof{}, errors.New("PyPI publish context is required")
	}
	if err := ctx.Err(); err != nil {
		return pypiPublishProof{}, err
	}
	converted, certDER, err := decodePyPIPublish(body, expected.Filename, expected.WheelSHA256, policy)
	if err != nil {
		return pypiPublishProof{}, err
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return pypiPublishProof{}, errors.New("PyPI publish certificate is invalid")
	}
	key, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok || key.Curve != elliptic.P256() {
		return pypiPublishProof{}, errors.New("PyPI publish certificate requires ECDSA P256")
	}
	trusted, err := root.NewTrustedRootFromJSON(publicRoot)
	if err != nil {
		return pypiPublishProof{}, errors.New("PyPI publish trust root is unavailable")
	}
	verifier, err := verify.NewVerifier(trusted, verify.WithSignedCertificateTimestamps(1), verify.WithTransparencyLog(1), verify.WithObserverTimestamps(1))
	if err != nil {
		return pypiPublishProof{}, errors.New("PyPI publish verifier is unavailable")
	}
	var signed bundle.Bundle
	if signed.UnmarshalJSON(converted) != nil {
		return pypiPublishProof{}, errors.New("PyPI publish bundle is invalid")
	}
	ref := policy.ref
	signer := policy.repository + "/.github/workflows/" + policy.workflow + "@" + ref
	san, err := verify.NewSANMatcher(signer, "")
	if err != nil {
		return pypiPublishProof{}, err
	}
	issuer, err := verify.NewIssuerMatcher("https://token.actions.githubusercontent.com", "")
	if err != nil {
		return pypiPublishProof{}, err
	}
	identity, err := verify.NewCertificateIdentity(san, issuer, certificate.Extensions{
		BuildSignerURI: signer, BuildSignerDigest: expected.SourceCommit,
		RunnerEnvironment: "github-hosted", SourceRepositoryURI: policy.repository,
		SourceRepositoryDigest: expected.SourceCommit, SourceRepositoryRef: ref,
		SourceRepositoryIdentifier: policy.repositoryID, SourceRepositoryOwnerURI: "https://github.com/" + policy.owner,
		SourceRepositoryOwnerIdentifier: policy.ownerID, BuildConfigURI: signer,
		BuildConfigDigest: expected.SourceCommit, BuildTrigger: policy.trigger,
		SourceRepositoryVisibilityAtSigning: "public",
	})
	if err != nil {
		return pypiPublishProof{}, errors.New("PyPI publish certificate policy is invalid")
	}
	digest, _ := hex.DecodeString(expected.WheelSHA256)
	verified, err := verifier.Verify(&signed, verify.NewPolicy(verify.WithArtifactDigest("sha256", digest), verify.WithCertificateIdentity(identity)))
	if err != nil || verified == nil || verified.Statement == nil || verified.Signature == nil || verified.Signature.Certificate == nil {
		return pypiPublishProof{}, errors.New("PyPI publish attestation verification failed")
	}
	// This is the same leaf DER passed to sigstore above. These two signed OIDs
	// are not exposed by the pinned library's certificate.Extensions type.
	if !validPyPIPublisherClaims(cert, policy) {
		return pypiPublishProof{}, errors.New("PyPI publish environment or token subject mismatch")
	}
	invocation := verified.Signature.Certificate.RunInvocationURI
	attempt, found := strings.CutPrefix(invocation, policy.repository+"/actions/runs/"+expected.WorkflowRun+"/attempts/")
	statement := verified.Statement
	if !found || !runNumber.MatchString(attempt) || statement.GetType() != "https://in-toto.io/Statement/v1" || statement.GetPredicateType() != pipxPublishPredicate || len(statement.GetSubject()) != 1 || (statement.GetPredicate() != nil && len(statement.GetPredicate().GetFields()) != 0) {
		return pypiPublishProof{}, errors.New("PyPI publish statement or invocation mismatch")
	}
	subject := statement.GetSubject()[0]
	if subject == nil || subject.GetName() != expected.Filename || len(subject.GetDigest()) != 1 || subject.GetDigest()["sha256"] != expected.WheelSHA256 {
		return pypiPublishProof{}, errors.New("PyPI publish signed wheel mismatch")
	}
	if err := ctx.Err(); err != nil {
		return pypiPublishProof{}, err
	}
	sum := sha256.Sum256(body)
	return pypiPublishProof{hex.EncodeToString(sum[:]), invocation}, nil
}

func validPipxPublisherClaims(cert *x509.Certificate) bool {
	return validPyPIPublisherClaims(cert, pipxPublisherPolicy(""))
}

func validPyPIPublisherClaims(cert *x509.Certificate, policy pypiPublisherPolicy) bool {
	if cert == nil {
		return false
	}
	subject := "repo:" + strings.TrimPrefix(policy.repository, "https://github.com/") + ":environment:" + policy.environment
	if policy.environment == "" {
		if policy.trigger != "push" || !strings.HasPrefix(policy.ref, "refs/tags/") {
			return false
		}
		subject = "repo:" + strings.TrimPrefix(policy.repository, "https://github.com/") + ":ref:" + policy.ref
	}
	for _, claim := range []struct {
		oid    asn1.ObjectIdentifier
		value  string
		absent bool
	}{
		{asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 23}, policy.environment, policy.environment == ""},
		{asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 24}, subject, false},
	} {
		count := 0
		for _, extension := range cert.Extensions {
			if !extension.Id.Equal(claim.oid) {
				continue
			}
			if claim.absent {
				return false
			}
			count++
			var value asn1.RawValue
			rest, err := asn1.Unmarshal(extension.Value, &value)
			if err != nil || len(rest) != 0 || value.Class != asn1.ClassUniversal || value.Tag != asn1.TagUTF8String || value.IsCompound || !utf8.Valid(value.Bytes) || string(value.Bytes) != claim.value {
				return false
			}
		}
		if !claim.absent && count != 1 {
			return false
		}
	}
	return true
}
