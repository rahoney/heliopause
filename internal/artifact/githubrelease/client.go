package githubrelease

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	assetTimeout       = 90 * time.Second
	maximumRedirects   = 3
	assetLocatorScheme = "github-release"
)

// Client is the public, unauthenticated GitHub Releases adapter. It resolves
// an exact tag once, then acquires only the resulting numeric asset ID.
type Client struct {
	api        *url.URL
	httpClient *http.Client
	intakeRoot string
	production bool
}

// NewPublicClient constructs the production adapter. It deliberately disables
// proxy discovery and does not accept credentials or arbitrary API origins.
func NewPublicClient(intakeRoot string) (*Client, error) {
	api, err := url.Parse(APIBaseURL)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 0, Transport: &http.Transport{
		Proxy: nil, ForceAttemptHTTP2: true, MaxIdleConns: 4, MaxIdleConnsPerHost: 2,
		IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}}
	return newClient(api, client, intakeRoot, true)
}

func newClient(api *url.URL, client *http.Client, intakeRoot string, production bool) (*Client, error) {
	if api == nil || client == nil || !filepath.IsAbs(intakeRoot) || api.User != nil || api.RawQuery != "" || api.Fragment != "" || (api.Path != "" && api.Path != "/") {
		return nil, errors.New("GitHub Release client configuration is invalid")
	}
	if production && (api.Scheme != "https" || !strings.EqualFold(api.Hostname(), "api.github.com") || api.Port() != "") {
		return nil, errors.New("GitHub Release production API must be https://api.github.com")
	}
	copyAPI := *api
	return &Client{api: &copyAPI, httpClient: client, intakeRoot: filepath.Clean(intakeRoot), production: production}, nil
}

// Resolve obtains one published exact-tag release and one unique uploaded
// asset. No release URL or source archive is accepted at this boundary.
func (c *Client) Resolve(ctx context.Context, reference domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	if err := validRequest(ctx, c); err != nil {
		return domain.ResolvedArtifact{}, err
	}
	if reference.Source().String() != SourceName {
		return domain.ResolvedArtifact{}, errors.New("GitHub Release resolver requires GitHub Release reference")
	}
	selector, err := ParseSelector(reference.Locator())
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	body, err := c.release(ctx, selector)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	asset, err := ParseReleaseForSelector(selector, body)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	identity, err := domain.NewResolvedArtifactIdentity(reference.Source(), selector.Owner()+"-"+selector.Repo(), selector.Tag(), selector.Asset())
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	return domain.NewResolvedArtifact(identity, assetLocator(asset), "sha256:"+asset.Digest().String())
}

// ResolveDependencies adapts one standalone Release asset to the existing
// complete-set workflow without adding a package dependency resolver.
func (c *Client) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, _ domain.InstallContext) (domain.DependencyResolution, error) {
	resolved, err := c.Resolve(ctx, reference)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	nodeID, _ := domain.NewDependencyNodeID("github-release-primary")
	node, err := domain.NewLockedDependencyWithRecordPath(nodeID, domain.DependencyPrimary, resolved, resolved.Identity().Variant(), false)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	declared, _ := domain.NewSHA256Digest(strings.TrimPrefix(resolved.DeclaredIntegrity(), "sha256:"))
	return domain.NewDependencyResolution(graph, "github-release-standalone", declared)
}

// Acquire streams the asset selected during Resolve into a Run-private intake
// directory. API-declared size and SHA-256 must both match observed bytes.
func (c *Client) Acquire(ctx context.Context, runID domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if err := validRequest(ctx, c); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if runID.String() == "" || resolved.Identity().Source().String() != SourceName {
		return domain.AcquiredArtifact{}, errors.New("GitHub Release intake request is invalid")
	}
	asset, err := parseAssetLocator(resolved.AcquisitionLocator())
	if err != nil || resolved.Identity().Name() != asset.selector.Owner()+"-"+asset.selector.Repo() || resolved.Identity().Version() != asset.selector.Tag() || resolved.Identity().Variant() != asset.selector.Asset() || resolved.DeclaredIntegrity() != "sha256:"+asset.digest.String() {
		return domain.AcquiredArtifact{}, errors.New("GitHub Release resolved artifact is invalid")
	}
	directory, err := createGithubRunDirectory(c.intakeRoot, runID.String())
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	cleanup := func(cause error) (domain.AcquiredArtifact, error) {
		if removeErr := os.RemoveAll(directory); removeErr != nil {
			return domain.AcquiredArtifact{}, errors.Join(cause, fmt.Errorf("remove incomplete GitHub Release intake: %w", removeErr))
		}
		return domain.AcquiredArtifact{}, cause
	}
	digest, size, err := c.downloadAsset(ctx, asset, directory)
	if err != nil {
		return cleanup(err)
	}
	if size != asset.size || digest != asset.digest.String() {
		return cleanup(errors.New("GitHub Release acquired bytes differ from resolved integrity"))
	}
	contentDigest, err := domain.NewSHA256Digest(digest)
	if err != nil {
		return cleanup(err)
	}
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), contentDigest, "intake:"+runID.String()+":github-release", size, resolved.DeclaredIntegrity())
}

func validRequest(ctx context.Context, c *Client) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c == nil || c.api == nil || c.httpClient == nil || c.intakeRoot == "" {
		return errors.New("GitHub Release client is not configured")
	}
	return nil
}

func (c *Client) release(ctx context.Context, selector Selector) ([]byte, error) {
	endpoint := *c.api
	endpoint.Path = "/repos/" + selector.Owner() + "/" + selector.Repo() + "/releases/tags/" + selector.Tag()
	endpoint.RawPath = "/repos/" + selector.Owner() + "/" + selector.Repo() + "/releases/tags/" + url.PathEscape(selector.Tag())
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	response, err := c.request(requestCtx, endpoint.String(), "application/vnd.github+json", false)
	if err != nil {
		return nil, errors.New("request GitHub Release")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "application/json") {
		return nil, errors.New("GitHub Release returned unexpected response")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maximumReleaseResponseBytes+1))
	if err != nil || len(body) == 0 || len(body) > maximumReleaseResponseBytes {
		return nil, errors.New("read GitHub Release response")
	}
	return body, nil
}

func (c *Client) downloadAsset(ctx context.Context, asset resolvedLocator, directory string) (string, uint64, error) {
	endpoint := *c.api
	endpoint.Path = "/repos/" + asset.selector.Owner() + "/" + asset.selector.Repo() + "/releases/assets/" + strconv.FormatUint(asset.assetID, 10)
	response, err := c.assetRequest(ctx, endpoint.String())
	if err != nil {
		return "", 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || (response.ContentLength >= 0 && uint64(response.ContentLength) != asset.size) {
		return "", 0, errors.New("GitHub Release asset returned unexpected response")
	}
	temporary, err := os.OpenFile(filepath.Join(directory, ".asset.tmp"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", 0, err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(temporary, hash), io.LimitReader(response.Body, int64(asset.size)+1))
	syncErr, closeErr := temporary.Sync(), temporary.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil || written == 0 || uint64(written) != asset.size {
		return "", 0, errors.New("stream GitHub Release asset failed or differed from declared size")
	}
	if err := os.Rename(temporaryPath, filepath.Join(directory, "asset")); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), uint64(written), nil
}

func (c *Client) request(ctx context.Context, rawURL, accept string, permitRedirect bool) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", accept)
	request.Header.Set("User-Agent", "helox/0")
	clientCopy := *c.httpClient
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := clientCopy.Do(request)
	if err != nil || !permitRedirect || response.StatusCode < http.StatusMultipleChoices || response.StatusCode > http.StatusPermanentRedirect {
		return response, err
	}
	return response, nil
}

func (c *Client) assetRequest(ctx context.Context, rawURL string) (*http.Response, error) {
	for redirects := 0; ; redirects++ {
		response, err := c.request(ctx, rawURL, "application/octet-stream", true)
		if err != nil {
			return nil, err
		}
		if response.StatusCode < http.StatusMultipleChoices || response.StatusCode > http.StatusPermanentRedirect {
			return response, nil
		}
		location, err := response.Location()
		response.Body.Close()
		if err != nil || redirects >= maximumRedirects || !c.allowedAssetRedirect(location) {
			return nil, errors.New("GitHub Release asset redirect is invalid")
		}
		rawURL = location.String()
	}
}

func (c *Client) allowedAssetRedirect(target *url.URL) bool {
	if target == nil || target.User != nil || target.Fragment != "" {
		return false
	}
	if c.production && target.Scheme != "https" {
		return false
	}
	host := strings.ToLower(target.Hostname())
	return host == "github.com" || host == "api.github.com" || host == "objects.githubusercontent.com" || host == "github-releases.githubusercontent.com" || host == "release-assets.githubusercontent.com" || (!c.production && host == strings.ToLower(c.api.Hostname()))
}

type resolvedLocator struct {
	selector  Selector
	releaseID uint64
	assetID   uint64
	digest    domain.ContentDigest
	size      uint64
}

func assetLocator(asset ResolvedAsset) string {
	values := url.Values{"tag": {asset.selector.Tag()}, "asset": {asset.selector.Asset()}, "release": {strconv.FormatUint(asset.releaseID, 10)}, "asset_id": {strconv.FormatUint(asset.assetID, 10)}, "sha256": {asset.digest.String()}, "size": {strconv.FormatUint(asset.size, 10)}}
	return assetLocatorScheme + "://" + asset.selector.Owner() + "/" + asset.selector.Repo() + "?" + values.Encode()
}

func parseAssetLocator(value string) (resolvedLocator, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != assetLocatorScheme || parsed.User != nil || parsed.Fragment != "" || parsed.Host == "" || strings.Count(strings.TrimPrefix(parsed.Path, "/"), "/") != 0 {
		return resolvedLocator{}, errors.New("GitHub Release acquisition locator is invalid")
	}
	selector, err := ParseSelector(parsed.Host + "/" + strings.TrimPrefix(parsed.Path, "/") + "@" + parsed.Query().Get("tag") + "#" + parsed.Query().Get("asset"))
	if err != nil {
		return resolvedLocator{}, errors.New("GitHub Release acquisition locator is invalid")
	}
	query := parsed.Query()
	if len(query) != 6 || len(query["tag"]) != 1 || len(query["asset"]) != 1 || len(query["release"]) != 1 || len(query["asset_id"]) != 1 || len(query["sha256"]) != 1 || len(query["size"]) != 1 {
		return resolvedLocator{}, errors.New("GitHub Release acquisition locator is invalid")
	}
	releaseID, releaseErr := strconv.ParseUint(query.Get("release"), 10, 64)
	assetID, assetErr := strconv.ParseUint(query.Get("asset_id"), 10, 64)
	size, sizeErr := strconv.ParseUint(query.Get("size"), 10, 64)
	digest, digestErr := domain.NewSHA256Digest(query.Get("sha256"))
	if releaseErr != nil || releaseID == 0 || assetErr != nil || assetID == 0 || sizeErr != nil || size == 0 || size > maximumAssetBytes || digestErr != nil {
		return resolvedLocator{}, errors.New("GitHub Release acquisition locator is invalid")
	}
	return resolvedLocator{selector: selector, releaseID: releaseID, assetID: assetID, digest: digest, size: size}, nil
}

func createGithubRunDirectory(root, runID string) (string, error) {
	cleanRoot := filepath.Clean(root)
	if !filepath.IsAbs(cleanRoot) {
		return "", errors.New("GitHub Release intake root must be absolute")
	}
	if err := os.MkdirAll(cleanRoot, 0o700); err != nil {
		return "", fmt.Errorf("create GitHub Release intake root: %w", err)
	}
	info, err := os.Lstat(cleanRoot)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("GitHub Release intake root is untrusted")
	}
	directory := filepath.Join(cleanRoot, runID)
	if filepath.Dir(directory) != cleanRoot {
		return "", errors.New("GitHub Release intake Run directory escapes root")
	}
	if err := os.Mkdir(directory, 0o700); err != nil {
		return "", fmt.Errorf("create GitHub Release intake Run directory: %w", err)
	}
	return directory, nil
}
