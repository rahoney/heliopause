// Package terraformprovider normalizes Terraform Registry Provider discovery
// and download metadata. Registry responses are untrusted until the exact
// version/platform/checksum/signer binding is established.
package terraformprovider

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"path"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const registryEndpoint = "https://registry.terraform.io"

var providerSegment = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)
var providerVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?$`)
var signingKeyID = regexp.MustCompile(`^[0-9A-F]{16}$`)
var signatureSuffix = regexp.MustCompile(`^(?:\.[0-9A-F]{8,16})?\.sig$`)
var providerSource = mustSource("terraform-registry")

var defaultDownloadHosts = map[string]bool{"releases.hashicorp.com": true, "github.com": true}

func Source() domain.SourceID  { return providerSource }
func RegistryEndpoint() string { return registryEndpoint }

func ParseReference(value string) (domain.ArtifactReference, error) {
	parts := strings.Split(value, "@")
	if len(parts) != 2 {
		return domain.ArtifactReference{}, errors.New("terraform Provider reference requires namespace/type@version")
	}
	path := strings.Split(parts[0], "/")
	if len(path) != 2 || !providerSegment.MatchString(path[0]) || !providerSegment.MatchString(path[1]) || !validProviderVersion(parts[1]) {
		return domain.ArtifactReference{}, errors.New("terraform Provider reference is invalid")
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
	Protocols           []string `json:"protocols"`
	OS                  string   `json:"os"`
	Arch                string   `json:"arch"`
	Filename            string   `json:"filename"`
	DownloadURL         string   `json:"download_url"`
	ShasumsURL          string   `json:"shasums_url"`
	ShasumsSignatureURL string   `json:"shasums_signature_url"`
	Shasum              string   `json:"shasum"`
	SigningKeys         struct {
		Keys []SigningKey `json:"gpg_public_keys"`
	} `json:"signing_keys"`
}

// SigningKey is an untrusted registry declaration. KeyID and source labels
// never confer trust; verification uses the certificate's full fingerprint.
type SigningKey struct {
	KeyID          string `json:"key_id"`
	ASCIIArmor     string `json:"ascii_armor"`
	TrustSignature string `json:"trust_signature"`
}

func ParseVersionResponse(body []byte, requestedVersion string, platform Platform) error {
	if len(body) == 0 || len(body) > 2<<20 || !validProviderVersion(requestedVersion) || !providerSegment.MatchString(platform.OS) || !providerSegment.MatchString(platform.Arch) {
		return errors.New("terraform Provider version request is invalid")
	}
	var document versionDocument
	if err := decodeRegistryJSON(body, &document, 2<<20); err != nil || len(document.Versions) == 0 || len(document.Versions) > 8192 {
		return errors.New("terraform Registry version response is invalid")
	}
	seen := map[string]bool{}
	available := false
	for _, entry := range document.Versions {
		if !validProviderVersion(entry.Version) || seen[entry.Version] || len(entry.Platforms) > 64 || len(entry.Protocols) > 8 {
			return errors.New("terraform Registry version identity is invalid or ambiguous")
		}
		seen[entry.Version] = true
		if entry.Version != requestedVersion {
			continue
		}
		if !validProtocols(entry.Protocols) {
			return errors.New("terraform requested provider protocols are invalid or unavailable")
		}
		seenPlatforms := map[Platform]bool{}
		for _, candidate := range entry.Platforms {
			if !providerSegment.MatchString(candidate.OS) || !providerSegment.MatchString(candidate.Arch) || seenPlatforms[candidate] {
				return errors.New("terraform requested provider platforms are invalid or ambiguous")
			}
			seenPlatforms[candidate] = true
			if candidate == platform {
				available = true
			}
		}
	}
	if available {
		return nil
	}
	return errors.New("terraform Provider version is unavailable")
}

// ProviderArtifact binds the Registry response to an exact source artifact.
type ProviderArtifact struct {
	Reference    domain.ArtifactReference
	Platform     Platform
	DownloadURL  string
	SHA256       string
	SignerKeyIDs []string
	Filename     string
	ShasumsURL   string
	SignatureURL string
	Protocols    []string
	SigningKeys  []SigningKey
}

// BuildLockedGraph binds one exact Provider installation to the generic
// dependency graph used by verification and Promotion.
func BuildLockedGraph(artifact ProviderArtifact) (domain.LockedDependencyGraph, error) {
	if artifact.Reference.Source() != providerSource || artifact.DownloadURL == "" || !isSHA256(artifact.SHA256) || len(artifact.SignerKeyIDs) == 0 {
		return domain.LockedDependencyGraph{}, errors.New("terraform Provider artifact is incomplete")
	}
	parts := strings.SplitN(artifact.Reference.Locator(), "@", 2)
	if len(parts) != 2 {
		return domain.LockedDependencyGraph{}, errors.New("terraform Provider artifact reference is invalid")
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
		return ProviderArtifact{}, errors.New("terraform Provider package request is invalid")
	}
	parts := strings.SplitN(reference.Locator(), "@", 2)
	parsed, parseErr := ParseReference(reference.Locator())
	if len(parts) != 2 || parseErr != nil || parsed != reference || platform != (Platform{OS: "linux", Arch: "amd64"}) {
		return ProviderArtifact{}, errors.New("terraform Provider reference is invalid")
	}
	var document packageDocument
	if err := decodeRegistryJSON(body, &document, 1<<20); err != nil {
		return ProviderArtifact{}, errors.New("terraform Registry package response is invalid")
	}
	segments := strings.Split(parts[0], "/")
	base := "terraform-provider-" + segments[1] + "_" + parts[1]
	filename := base + "_" + platform.OS + "_" + platform.Arch + ".zip"
	if document.OS != platform.OS || document.Arch != platform.Arch || document.Filename != filename || !validProtocols(document.Protocols) || !isSHA256(document.Shasum) || document.Shasum != strings.ToLower(document.Shasum) || document.DownloadURL == "" || document.ShasumsURL == "" || document.ShasumsSignatureURL == "" {
		return ProviderArtifact{}, errors.New("terraform Provider package binding is incomplete")
	}
	if err := trustedURL(document.DownloadURL); err != nil {
		return ProviderArtifact{}, errors.New("terraform Provider package endpoint is untrusted")
	}
	if err := trustedURL(document.ShasumsURL); err != nil {
		return ProviderArtifact{}, errors.New("terraform Provider package endpoint is untrusted")
	}
	if err := trustedURL(document.ShasumsSignatureURL); err != nil {
		return ProviderArtifact{}, errors.New("terraform Provider package endpoint is untrusted")
	}
	download, _ := url.Parse(document.DownloadURL)
	sums, _ := url.Parse(document.ShasumsURL)
	signature, _ := url.Parse(document.ShasumsSignatureURL)
	if download.Hostname() == "registry.terraform.io" || sums.Hostname() != signature.Hostname() || download.Hostname() != sums.Hostname() || !allowedHosts[strings.ToLower(download.Hostname())] {
		return ProviderArtifact{}, errors.New("terraform Provider download endpoint identity is invalid")
	}
	parent := path.Dir(download.Path)
	if path.Base(download.Path) != filename || path.Dir(sums.Path) != parent || path.Dir(signature.Path) != parent || path.Base(sums.Path) != base+"_SHA256SUMS" || !strings.HasPrefix(path.Base(signature.Path), path.Base(sums.Path)) || !signatureSuffix.MatchString(strings.TrimPrefix(path.Base(signature.Path), path.Base(sums.Path))) {
		return ProviderArtifact{}, errors.New("terraform Provider download paths do not bind exact package")
	}
	switch download.Hostname() {
	case "releases.hashicorp.com":
		if segments[0] != "hashicorp" || parent != "/terraform-provider-"+segments[1]+"/"+parts[1] {
			return ProviderArtifact{}, errors.New("terraform Provider official endpoint binding is invalid")
		}
	case "github.com":
		expected := "/" + segments[0] + "/terraform-provider-" + segments[1] + "/releases/download/"
		if parent != expected+parts[1] && parent != expected+"v"+parts[1] {
			return ProviderArtifact{}, errors.New("terraform Provider release endpoint binding is invalid")
		}
	default:
		return ProviderArtifact{}, errors.New("terraform Provider download endpoint is unsupported")
	}
	keys := make([]string, 0, len(document.SigningKeys.Keys))
	seenKeys := map[string]bool{}
	if len(document.SigningKeys.Keys) > 8 {
		return ProviderArtifact{}, errors.New("terraform Provider signing key count exceeds bound")
	}
	for _, key := range document.SigningKeys.Keys {
		if !signingKeyID.MatchString(key.KeyID) || seenKeys[key.KeyID] || len(key.ASCIIArmor) > 128<<10 || !strings.HasPrefix(key.ASCIIArmor, "-----BEGIN PGP PUBLIC KEY BLOCK-----") || len(key.TrustSignature) > 16<<10 {
			return ProviderArtifact{}, errors.New("terraform Provider signing declaration is invalid or ambiguous")
		}
		seenKeys[key.KeyID] = true
		keys = append(keys, key.KeyID)
	}
	if len(keys) == 0 {
		return ProviderArtifact{}, errors.New("terraform Provider signer identity is missing")
	}
	return ProviderArtifact{Reference: reference, Platform: platform, DownloadURL: document.DownloadURL, SHA256: document.Shasum, SignerKeyIDs: keys, Filename: filename, ShasumsURL: document.ShasumsURL, SignatureURL: document.ShasumsSignatureURL, Protocols: document.Protocols, SigningKeys: document.SigningKeys.Keys}, nil
}

func VerifyLockHash(lockHashes []string, sha256Hex string) error {
	if !isSHA256(sha256Hex) {
		return errors.New("terraform Provider checksum is invalid")
	}
	for _, value := range lockHashes {
		// zh is the archive SHA-256 form. h1 is Terraform's directory hash and
		// cannot be equated to the Registry shasum without inspecting the
		// extracted archive, so h1-only locks fail closed at this boundary.
		if strings.HasPrefix(value, "zh:") && strings.EqualFold(strings.TrimPrefix(value, "zh:"), sha256Hex) {
			return nil
		}
	}
	return errors.New("terraform Provider checksum is absent from lock file")
}

func trustedURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || len(value) > 2048 || strings.ContainsAny(value, "\x00\r\n\\") || parsed.Scheme != "https" || parsed.Opaque != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.Hostname() == "" || parsed.Host != parsed.Hostname() || parsed.RawPath != "" || parsed.Path == "" || path.Clean(parsed.Path) != parsed.Path || parsed.String() != value {
		return errors.New("uRL is not canonical HTTPS")
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

func validProviderVersion(value string) bool {
	return providerVersion.MatchString(value) && semver.IsValid("v"+value)
}
