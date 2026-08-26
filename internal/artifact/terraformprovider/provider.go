// Package terraformprovider normalizes Terraform Registry Provider discovery
// and download metadata. Registry responses are untrusted until the exact
// version/platform/checksum/signer binding is established.
package terraformprovider

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const registryEndpoint = "https://registry.terraform.io"

var providerSegment = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,127}$`)
var providerVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)
var providerSource = mustSource("terraform-registry")

var defaultDownloadHosts = map[string]bool{"releases.hashicorp.com": true}

func Source() domain.SourceID  { return providerSource }
func RegistryEndpoint() string { return registryEndpoint }

func ParseReference(value string) (domain.ArtifactReference, error) {
	parts := strings.Split(value, "@")
	if len(parts) != 2 {
		return domain.ArtifactReference{}, errors.New("Terraform Provider reference requires namespace/type@version")
	}
	path := strings.Split(parts[0], "/")
	if len(path) != 2 || !providerSegment.MatchString(path[0]) || !providerSegment.MatchString(path[1]) || !providerVersion.MatchString(parts[1]) {
		return domain.ArtifactReference{}, errors.New("Terraform Provider reference is invalid")
	}
	return domain.NewArtifactReference(providerSource, parts[0]+"@"+parts[1])
}

type Platform struct {
	OS   string
	Arch string
}
type versionDocument struct {
	Versions []versionEntry `json:"versions"`
}
type versionEntry struct {
	Version   string     `json:"version"`
	Protocols []string   `json:"protocols"`
	Platforms []Platform `json:"platforms"`
}
type packageDocument struct {
	Protocol            string `json:"protocol"`
	OS                  string `json:"os"`
	Arch                string `json:"arch"`
	Filename            string `json:"filename"`
	DownloadURL         string `json:"download_url"`
	ShasumsURL          string `json:"shasums_url"`
	ShasumsSignatureURL string `json:"shasums_signature_url"`
	Shasum              string `json:"shasum"`
	SigningKeys         []struct {
		KeyID string `json:"key_id"`
	} `json:"signing_keys"`
}

func ParseVersionResponse(body []byte, requestedVersion string, platform Platform) error {
	if len(body) == 0 || len(body) > 2<<20 || !providerVersion.MatchString(requestedVersion) || !providerSegment.MatchString(platform.OS) || !providerSegment.MatchString(platform.Arch) {
		return errors.New("Terraform Provider version request is invalid")
	}
	var document versionDocument
	if err := json.Unmarshal(body, &document); err != nil {
		return errors.New("Terraform Registry version response is invalid")
	}
	for _, entry := range document.Versions {
		if entry.Version != requestedVersion {
			continue
		}
		for _, candidate := range entry.Platforms {
			if candidate == platform {
				return nil
			}
		}
		return errors.New("Terraform Provider platform is unavailable")
	}
	return errors.New("Terraform Provider version is unavailable")
}

// ProviderArtifact binds the Registry response to an exact source artifact.
type ProviderArtifact struct {
	Reference    domain.ArtifactReference
	Platform     Platform
	DownloadURL  string
	SHA256       string
	SignerKeyIDs []string
}

// BuildLockedGraph binds one exact Provider installation to the generic
// dependency graph used by verification and Promotion.
func BuildLockedGraph(artifact ProviderArtifact) (domain.LockedDependencyGraph, error) {
	if artifact.Reference.Source() != providerSource || artifact.DownloadURL == "" || !isSHA256(artifact.SHA256) || len(artifact.SignerKeyIDs) == 0 {
		return domain.LockedDependencyGraph{}, errors.New("Terraform Provider artifact is incomplete")
	}
	parts := strings.SplitN(artifact.Reference.Locator(), "@", 2)
	if len(parts) != 2 {
		return domain.LockedDependencyGraph{}, errors.New("Terraform Provider artifact reference is invalid")
	}
	name := strings.ReplaceAll(parts[0], "/", "_")
	identity, err := domain.NewResolvedArtifactIdentity(providerSource, name, parts[1], artifact.Platform.OS+"/"+artifact.Platform.Arch)
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	resolved, err := domain.NewResolvedArtifact(identity, artifact.DownloadURL, "sha256="+artifact.SHA256+";signer="+strings.Join(artifact.SignerKeyIDs, ","))
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	digest := sha256.Sum256([]byte(artifact.Reference.Locator()))
	node, err := domain.NewDependencyNodeID("t" + hex.EncodeToString(digest[:])[:24])
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	locked, err := domain.NewLockedDependency(node, domain.DependencyPrimary, resolved)
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	return domain.NewLockedDependencyGraph([]domain.LockedDependency{locked}, nil)
}

func ParsePackageResponse(reference domain.ArtifactReference, body []byte, platform Platform) (ProviderArtifact, error) {
	return ParsePackageResponseWithAllowedHosts(reference, body, platform, defaultDownloadHosts)
}

// ParsePackageResponseWithAllowedHosts applies the caller's canonical vendor
// endpoint policy to the exact host returned by the Registry response.
func ParsePackageResponseWithAllowedHosts(reference domain.ArtifactReference, body []byte, platform Platform, allowedHosts map[string]bool) (ProviderArtifact, error) {
	if reference.Source() != providerSource || len(body) == 0 || len(body) > 1<<20 {
		return ProviderArtifact{}, errors.New("Terraform Provider package request is invalid")
	}
	parts := strings.SplitN(reference.Locator(), "@", 2)
	if len(parts) != 2 {
		return ProviderArtifact{}, errors.New("Terraform Provider reference is invalid")
	}
	var document packageDocument
	if err := json.Unmarshal(body, &document); err != nil {
		return ProviderArtifact{}, errors.New("Terraform Registry package response is invalid")
	}
	if document.OS != platform.OS || document.Arch != platform.Arch || !providerVersion.MatchString(parts[1]) || document.Filename == "" || !isSHA256(document.Shasum) || document.DownloadURL == "" || document.ShasumsURL == "" || document.ShasumsSignatureURL == "" {
		return ProviderArtifact{}, errors.New("Terraform Provider package binding is incomplete")
	}
	if err := trustedURL(document.DownloadURL); err != nil {
		return ProviderArtifact{}, errors.New("Terraform Provider package endpoint is untrusted")
	}
	if err := trustedURL(document.ShasumsURL); err != nil {
		return ProviderArtifact{}, errors.New("Terraform Provider package endpoint is untrusted")
	}
	if err := trustedURL(document.ShasumsSignatureURL); err != nil {
		return ProviderArtifact{}, errors.New("Terraform Provider package endpoint is untrusted")
	}
	download, _ := url.Parse(document.DownloadURL)
	sums, _ := url.Parse(document.ShasumsURL)
	signature, _ := url.Parse(document.ShasumsSignatureURL)
	if download.Hostname() == "registry.terraform.io" || sums.Hostname() != signature.Hostname() || download.Hostname() != sums.Hostname() || !allowedHosts[strings.ToLower(download.Hostname())] {
		return ProviderArtifact{}, errors.New("Terraform Provider download endpoint identity is invalid")
	}
	keys := make([]string, 0, len(document.SigningKeys))
	for _, key := range document.SigningKeys {
		if key.KeyID != "" {
			keys = append(keys, key.KeyID)
		}
	}
	if len(keys) == 0 {
		return ProviderArtifact{}, errors.New("Terraform Provider signer identity is missing")
	}
	return ProviderArtifact{Reference: reference, Platform: platform, DownloadURL: document.DownloadURL, SHA256: strings.ToLower(document.Shasum), SignerKeyIDs: keys}, nil
}

func VerifyLockHash(lockHashes []string, sha256Hex string) error {
	if !isSHA256(sha256Hex) {
		return errors.New("Terraform Provider checksum is invalid")
	}
	for _, value := range lockHashes {
		// zh is the archive SHA-256 form. h1 is Terraform's directory hash and
		// cannot be equated to the Registry shasum without inspecting the
		// extracted archive, so h1-only locks fail closed at this boundary.
		if strings.HasPrefix(value, "zh:") && strings.EqualFold(strings.TrimPrefix(value, "zh:"), sha256Hex) {
			return nil
		}
	}
	return errors.New("Terraform Provider checksum is absent from lock file")
}

func trustedURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Hostname() == "" || parsed.Path == "" {
		return errors.New("URL is not canonical HTTPS")
	}
	return nil
}
func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
func mustSource(value string) domain.SourceID {
	source, err := domain.NewSourceID(value)
	if err != nil {
		panic(err)
	}
	return source
}
