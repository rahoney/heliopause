package releaseprovenance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"golang.org/x/mod/semver"
)

const (
	uvRepository = "https://github.com/astral-sh/uv"
	uvImageName  = "ghcr.io/astral-sh/uv"
	uvRef        = "refs/heads/main"
	uvBuilder    = uvRepository + "/.github/workflows/publish-docker-image.yml@" + uvRef
	uvCaller     = uvRepository + "/.github/workflows/release.yml@" + uvRef
	ociIndexType = "application/vnd.oci.image.index.v1+json"
	ociImageType = "application/vnd.oci.image.manifest.v1+json"
)

// UVImageIdentity is a frozen discovery request. It grants no authority until
// VerifyUVImage authenticates the publisher and the entire index/config chain.
type UVImageIdentity struct {
	Version        string `json:"version"`
	SourceCommit   string `json:"source_commit"`
	WorkflowRun    string `json:"workflow_run"`
	IndexSHA256    string `json:"index_sha256"`
	ManifestSHA256 string `json:"manifest_sha256"`
}

// UVImageReceipt authenticates upstream supply metadata, not executable bytes,
// sandbox compatibility or permission to activate a new runtime.
type UVImageReceipt struct {
	identity     UVImageIdentity
	bundleSHA256 string
	configSHA256 string
	layers       []UVImageLayer
	invocation   string
}

type UVImageLayer struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func (r UVImageReceipt) Identity() UVImageIdentity { return r.identity }
func (r UVImageReceipt) BundleSHA256() string      { return r.bundleSHA256 }
func (r UVImageReceipt) ConfigSHA256() string      { return r.configSHA256 }
func (r UVImageReceipt) Invocation() string        { return r.invocation }
func (r UVImageReceipt) Layers() []UVImageLayer    { return append([]UVImageLayer(nil), r.layers...) }

// VerifyUVImage uses the same pinned offline Sigstore root as HAA releases,
// with a separate, fixed upstream certificate/predicate policy. HAA Verify's
// tag, repository and workflow policy is unchanged. No network is consulted.
func VerifyUVImage(ctx context.Context, proof, index, manifest, config []byte, expected UVImageIdentity) (UVImageReceipt, error) {
	if ctx == nil {
		return UVImageReceipt{}, errors.New("uv image provenance context is required")
	}
	if err := ctx.Err(); err != nil {
		return UVImageReceipt{}, err
	}
	version := "v" + expected.Version
	if len(proof) == 0 || len(proof) > maxBundle || len(expected.Version) > 64 || !releaseVersion.MatchString(version) || !semver.IsValid(version) || semver.Prerelease(version) != "" || semver.Build(version) != "" || !hexCommit.MatchString(expected.SourceCommit) || !runNumber.MatchString(expected.WorkflowRun) || !hexSHA256.MatchString(expected.IndexSHA256) || !hexSHA256.MatchString(expected.ManifestSHA256) || len(index) == 0 || len(index) > 1<<20 || len(manifest) == 0 || len(manifest) > 64<<10 || len(config) == 0 || len(config) > 64<<10 {
		return UVImageReceipt{}, errors.New("uv image provenance input is invalid")
	}
	if uvDigest(index) != expected.IndexSHA256 || uvDigest(manifest) != expected.ManifestSHA256 {
		return UVImageReceipt{}, errors.New("uv image discovery digest mismatch")
	}
	trusted, err := root.NewTrustedRootFromJSON(publicRoot)
	if err != nil {
		return UVImageReceipt{}, errors.New("uv image trust root is unavailable")
	}
	verifier, err := verify.NewVerifier(trusted, verify.WithSignedCertificateTimestamps(1), verify.WithTransparencyLog(1), verify.WithObserverTimestamps(1))
	if err != nil {
		return UVImageReceipt{}, errors.New("uv image verifier is unavailable")
	}
	var signed bundle.Bundle
	if err := signed.UnmarshalJSON(proof); err != nil {
		return UVImageReceipt{}, errors.New("uv image attestation bundle is invalid")
	}
	san, err := verify.NewSANMatcher(uvBuilder, "")
	if err != nil {
		return UVImageReceipt{}, err
	}
	issuer, err := verify.NewIssuerMatcher("https://token.actions.githubusercontent.com", "")
	if err != nil {
		return UVImageReceipt{}, err
	}
	identity, err := verify.NewCertificateIdentity(san, issuer, certificate.Extensions{
		BuildSignerURI: uvBuilder, BuildSignerDigest: expected.SourceCommit,
		RunnerEnvironment: "github-hosted", SourceRepositoryURI: uvRepository,
		SourceRepositoryDigest: expected.SourceCommit, SourceRepositoryRef: uvRef,
		SourceRepositoryIdentifier: "699532645", SourceRepositoryOwnerURI: "https://github.com/astral-sh",
		SourceRepositoryOwnerIdentifier: "115962839", BuildConfigURI: uvCaller,
		BuildConfigDigest: expected.SourceCommit, BuildTrigger: "workflow_dispatch",
		SourceRepositoryVisibilityAtSigning: "public",
	})
	if err != nil {
		return UVImageReceipt{}, errors.New("uv image certificate policy is invalid")
	}
	digest, _ := hex.DecodeString(expected.IndexSHA256)
	verified, err := verifier.Verify(&signed, verify.NewPolicy(verify.WithArtifactDigest("sha256", digest), verify.WithCertificateIdentity(identity)))
	if err != nil || verified == nil || verified.Signature == nil || verified.Signature.Certificate == nil || verified.Statement == nil {
		return UVImageReceipt{}, errors.New("uv image attestation verification failed")
	}
	if err := ctx.Err(); err != nil {
		return UVImageReceipt{}, err
	}
	invocation := verified.Signature.Certificate.RunInvocationURI
	attempt, found := strings.CutPrefix(invocation, uvRepository+"/actions/runs/"+expected.WorkflowRun+"/attempts/")
	statement := verified.Statement
	if !found || !runNumber.MatchString(attempt) || statement.GetType() != "https://in-toto.io/Statement/v1" || statement.GetPredicateType() != "https://slsa.dev/provenance/v1" || len(statement.GetSubject()) != 1 {
		return UVImageReceipt{}, errors.New("uv image statement or invocation mismatch")
	}
	subject := statement.GetSubject()[0]
	if subject == nil || subject.GetName() != uvImageName || len(subject.GetDigest()) != 1 || subject.GetDigest()["sha256"] != expected.IndexSHA256 {
		return UVImageReceipt{}, errors.New("uv image signed subject mismatch")
	}
	predicate, err := json.Marshal(statement.GetPredicate())
	if err != nil || !validUVPredicate(predicate, expected, invocation) {
		return UVImageReceipt{}, errors.New("uv image build provenance mismatch")
	}
	layers, err := bindUVImage(index, manifest, config, expected)
	if err != nil {
		return UVImageReceipt{}, err
	}
	return UVImageReceipt{expected, uvDigest(proof), uvDigest(config), layers, invocation}, nil
}

func validUVPredicate(body []byte, expected UVImageIdentity, invocation string) bool {
	var value struct {
		BuildDefinition struct {
			BuildType          string
			ExternalParameters struct {
				Workflow struct{ Ref, Repository, Path string }
			}
			InternalParameters struct {
				GitHub struct {
					Event      string `json:"event_name"`
					Repository string `json:"repository_id"`
					Owner      string `json:"repository_owner_id"`
					Runner     string `json:"runner_environment"`
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
	if decodeUVJSON(body, &value) != nil {
		return false
	}
	build := value.BuildDefinition
	github := build.InternalParameters.GitHub
	if build.BuildType != "https://actions.github.io/buildtypes/workflow/v1" || build.ExternalParameters.Workflow.Ref != uvRef || build.ExternalParameters.Workflow.Repository != uvRepository || build.ExternalParameters.Workflow.Path != ".github/workflows/release.yml" || github.Event != "workflow_dispatch" || github.Repository != "699532645" || github.Owner != "115962839" || github.Runner != "github-hosted" || len(build.ResolvedDependencies) != 1 {
		return false
	}
	dependency := build.ResolvedDependencies[0]
	return dependency.URI == "git+"+uvRepository+"@"+uvRef && len(dependency.Digest) == 1 && dependency.Digest["gitCommit"] == expected.SourceCommit && value.RunDetails.Builder.ID == uvBuilder && value.RunDetails.Metadata.InvocationID == invocation
}

type uvOCIDescriptor struct {
	MediaType string                                     `json:"mediaType"`
	Digest    string                                     `json:"digest"`
	Size      int64                                      `json:"size"`
	Platform  struct{ OS, Architecture, Variant string } `json:"platform"`
}

func bindUVImage(index, manifest, config []byte, expected UVImageIdentity) ([]UVImageLayer, error) {
	var list struct {
		Schema    int               `json:"schemaVersion"`
		MediaType string            `json:"mediaType"`
		Manifests []uvOCIDescriptor `json:"manifests"`
	}
	if decodeUVJSON(index, &list) != nil || list.Schema != 2 || list.MediaType != ociIndexType || len(list.Manifests) == 0 || len(list.Manifests) > 64 {
		return nil, errors.New("uv image index is unsupported")
	}
	matches := 0
	seen := map[string]bool{}
	for _, item := range list.Manifests {
		if item.MediaType != ociImageType || !uvOCIDigest(item.Digest) || item.Size <= 0 || item.Size > 64<<10 || seen[item.Digest] {
			return nil, errors.New("uv image index descriptor is invalid")
		}
		seen[item.Digest] = true
		if item.Platform.OS == "linux" && item.Platform.Architecture == "amd64" {
			matches++
			if item.Platform.Variant != "" || item.Digest != "sha256:"+expected.ManifestSHA256 || item.Size != int64(len(manifest)) {
				return nil, errors.New("uv image platform selection mismatch")
			}
		}
	}
	if matches != 1 {
		return nil, errors.New("uv image amd64 selection is absent or ambiguous")
	}
	var image struct {
		Schema    int               `json:"schemaVersion"`
		MediaType string            `json:"mediaType"`
		Config    uvOCIDescriptor   `json:"config"`
		Layers    []uvOCIDescriptor `json:"layers"`
	}
	if decodeUVJSON(manifest, &image) != nil || image.Schema != 2 || image.MediaType != ociImageType || image.Config.MediaType != "application/vnd.oci.image.config.v1+json" || image.Config.Digest != "sha256:"+uvDigest(config) || image.Config.Size != int64(len(config)) || len(image.Layers) == 0 || len(image.Layers) > 4 {
		return nil, errors.New("uv image config binding is invalid")
	}
	var configuration struct {
		OS, Architecture string
		Config           struct {
			Labels     map[string]string
			Entrypoint []string
		}
		RootFS struct {
			Type    string
			DiffIDs []string `json:"diff_ids"`
		} `json:"rootfs"`
	}
	if decodeUVJSON(config, &configuration) != nil || configuration.OS != "linux" || configuration.Architecture != "amd64" || len(configuration.Config.Entrypoint) != 1 || configuration.Config.Entrypoint[0] != "/uv" || configuration.RootFS.Type != "layers" || len(configuration.RootFS.DiffIDs) != len(image.Layers) {
		return nil, errors.New("uv image runtime configuration is unsupported")
	}
	labels := configuration.Config.Labels
	if labels["org.opencontainers.image.source"] != uvRepository || labels["org.opencontainers.image.version"] != expected.Version || labels["org.opencontainers.image.revision"] != expected.SourceCommit {
		return nil, errors.New("uv image release identity mismatch")
	}
	result := make([]UVImageLayer, 0, len(image.Layers))
	seen = map[string]bool{}
	var total int64
	for i, item := range image.Layers {
		if item.MediaType != "application/vnd.oci.image.layer.v1.tar+gzip" || !uvOCIDigest(item.Digest) || !uvOCIDigest(configuration.RootFS.DiffIDs[i]) || item.Size <= 0 || item.Size > 64<<20 || total > (64<<20)-item.Size || seen[item.Digest] {
			return nil, errors.New("uv image layer descriptor is unsupported")
		}
		seen[item.Digest] = true
		total += item.Size
		result = append(result, UVImageLayer{strings.TrimPrefix(item.Digest, "sha256:"), item.Size})
	}
	return result, nil
}

func uvOCIDigest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && hexSHA256.MatchString(strings.TrimPrefix(value, "sha256:"))
}
func uvDigest(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }
func decodeUVJSON(body []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("uv image metadata has trailing JSON")
	}
	return nil
}
