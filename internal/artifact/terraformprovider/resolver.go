package terraformprovider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const maximumPackageResponseBytes = 1 << 20

// Resolver reads only the fixed public Terraform Registry package endpoint.
// It neither follows user-supplied registry URLs nor inherits proxy settings.
type Resolver struct {
	client   *http.Client
	endpoint *url.URL
	platform Platform
}

func NewPublicResolver() (*Resolver, error) {
	endpoint, err := url.Parse(registryEndpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() != "registry.terraform.io" {
		return nil, errors.New("terraform Registry endpoint is invalid")
	}
	return newResolver(endpoint, &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{
		Proxy: nil, ForceAttemptHTTP2: true, MaxIdleConns: 2, MaxIdleConnsPerHost: 2,
		IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second,
	}})
}

func newResolver(endpoint *url.URL, client *http.Client) (*Resolver, error) {
	if endpoint == nil || client == nil || endpoint.Scheme != "https" || endpoint.Host != "registry.terraform.io" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Opaque != "" || endpoint.RawPath != "" || endpoint.ForceQuery || (endpoint.Path != "" && endpoint.Path != "/") {
		return nil, errors.New("terraform Registry resolver configuration is invalid")
	}
	copyEndpoint := *endpoint
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Resolver{client: &copyClient, endpoint: &copyEndpoint, platform: Platform{OS: "linux", Arch: "amd64"}}, nil
}

type RegistrySnapshot struct {
	Discovery []byte
	Versions  []byte
	Package   []byte
}

func (s RegistrySnapshot) Parse(reference domain.ArtifactReference, platform Platform) (ProviderArtifact, error) {
	parsed, err := ParseReference(reference.Locator())
	if err != nil || parsed != reference {
		return ProviderArtifact{}, errors.New("terraform snapshot reference is invalid")
	}
	if err := ValidateDiscovery(s.Discovery); err != nil {
		return ProviderArtifact{}, err
	}
	parts := strings.SplitN(reference.Locator(), "@", 2)
	if err := ParseVersionResponse(s.Versions, parts[1], platform); err != nil {
		return ProviderArtifact{}, err
	}
	artifact, err := ParsePackageResponse(reference, s.Package, platform)
	if err != nil {
		return ProviderArtifact{}, err
	}
	var versions versionDocument
	if err := decodeRegistryJSON(s.Versions, &versions, 2<<20); err != nil {
		return ProviderArtifact{}, err
	}
	for _, version := range versions.Versions {
		if version.Version == parts[1] && strings.Join(version.Protocols, ",") != strings.Join(artifact.Protocols, ",") {
			return ProviderArtifact{}, errors.New("terraform registry protocol declarations differ")
		}
	}
	return artifact, nil
}

func (r *Resolver) ResolveSnapshot(ctx context.Context, reference domain.ArtifactReference) (RegistrySnapshot, ProviderArtifact, error) {
	if r == nil || r.client == nil || ctx == nil || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return RegistrySnapshot{}, ProviderArtifact{}, errors.New("terraform public resolver request is invalid")
	}
	parsed, err := ParseReference(reference.Locator())
	if err != nil || parsed != reference {
		return RegistrySnapshot{}, ProviderArtifact{}, errors.New("terraform public reference is invalid")
	}
	if err := ctx.Err(); err != nil {
		return RegistrySnapshot{}, ProviderArtifact{}, err
	}
	var snapshot RegistrySnapshot
	snapshot.Discovery, err = r.readJSON(ctx, "/.well-known/terraform.json", 64<<10)
	if err != nil {
		return snapshot, ProviderArtifact{}, err
	}
	if err := ValidateDiscovery(snapshot.Discovery); err != nil {
		return snapshot, ProviderArtifact{}, err
	}
	parts := strings.SplitN(reference.Locator(), "@", 2)
	snapshot.Versions, err = r.readJSON(ctx, "/v1/providers/"+parts[0]+"/versions", 2<<20)
	if err != nil {
		return snapshot, ProviderArtifact{}, err
	}
	if err := ParseVersionResponse(snapshot.Versions, parts[1], r.platform); err != nil {
		return snapshot, ProviderArtifact{}, err
	}
	snapshot.Package, err = r.readJSON(ctx, "/v1/providers/"+parts[0]+"/"+parts[1]+"/download/"+r.platform.OS+"/"+r.platform.Arch, maximumPackageResponseBytes)
	if err != nil {
		return snapshot, ProviderArtifact{}, err
	}
	artifact, err := snapshot.Parse(reference, r.platform)
	return snapshot, artifact, err
}

func (r *Resolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, _ domain.InstallContext) (domain.DependencyResolution, error) {
	if r == nil || r.client == nil || r.endpoint == nil || ctx == nil || reference.Source() != providerSource || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return domain.DependencyResolution{}, errors.New("valid Terraform Provider resolver request is required")
	}
	snapshot, artifact, err := r.ResolveSnapshot(ctx, reference)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	graph, err := BuildLockedGraph(artifact)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	digestBytes := sha256.Sum256(snapshot.Package)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(digestBytes[:]))
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return domain.NewDependencyResolution(graph, "terraform-registry:registry.terraform.io;platform:linux/amd64", digest)
}

func (r *Resolver) readJSON(ctx context.Context, relative string, limit int64) ([]byte, error) {
	endpoint := *r.endpoint
	endpoint.Path = relative
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, errors.New("create terraform Registry request")
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "helox/0")
	response, err := r.client.Do(request)
	if err != nil {
		return nil, errors.Join(errors.New("request terraform Registry metadata"), ctx.Err())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength > limit || (response.Header.Get("Content-Type") != "application/json" && !strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "application/json;")) {
		return nil, errors.New("terraform Registry metadata response is invalid")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || len(body) == 0 || int64(len(body)) > limit || (response.ContentLength >= 0 && int64(len(body)) != response.ContentLength) {
		return nil, errors.Join(errors.New("read bounded terraform Registry metadata"), ctx.Err())
	}
	return body, nil
}
