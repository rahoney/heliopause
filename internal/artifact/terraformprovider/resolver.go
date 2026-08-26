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
type Resolver struct{ client *http.Client }

func NewPublicResolver() (*Resolver, error) {
	endpoint, err := url.Parse(registryEndpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() != "registry.terraform.io" {
		return nil, errors.New("terraform Registry endpoint is invalid")
	}
	return &Resolver{client: &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{
		Proxy: nil, ForceAttemptHTTP2: true, MaxIdleConns: 2, MaxIdleConnsPerHost: 2,
		IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second,
	}}}, nil
}

func (r *Resolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, _ domain.InstallContext) (domain.DependencyResolution, error) {
	if r == nil || r.client == nil || ctx == nil || reference.Source() != providerSource || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return domain.DependencyResolution{}, errors.New("valid Terraform Provider resolver request is required")
	}
	parts := strings.SplitN(reference.Locator(), "@", 2)
	if len(parts) != 2 {
		return domain.DependencyResolution{}, errors.New("terraform Provider reference is invalid")
	}
	endpoint := registryEndpoint + "/v1/providers/" + parts[0] + "/" + parts[1] + "/download/linux/amd64"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return domain.DependencyResolution{}, errors.New("create terraform Registry request")
	}
	response, err := r.client.Do(request)
	if err != nil {
		return domain.DependencyResolution{}, errors.New("request terraform Registry package")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "application/json") {
		return domain.DependencyResolution{}, errors.New("terraform Registry package response is invalid")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maximumPackageResponseBytes+1))
	if err != nil || len(body) == 0 || len(body) > maximumPackageResponseBytes {
		return domain.DependencyResolution{}, errors.New("read terraform Registry package response")
	}
	artifact, err := ParsePackageResponse(reference, body, Platform{OS: "linux", Arch: "amd64"})
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	graph, err := BuildLockedGraph(artifact)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	digestBytes := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(digestBytes[:]))
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return domain.NewDependencyResolution(graph, "terraform-registry:registry.terraform.io;platform:linux/amd64", digest)
}
