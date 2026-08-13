// Package npm implements the public npm registry boundary.
package npm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	publicRegistryURL = "https://registry.npmjs.org/"
	metadataLimit     = 2 << 20
	metadataTimeout   = 10 * time.Second
	metadataAccept    = "application/vnd.npm.install-v1+json"
)

var (
	packageSegment = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	tagPattern     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
	exactVersion   = regexp.MustCompile(`^(?:v)?(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
)

// Resolver resolves only unauthenticated package references against the public npm registry.
type Resolver struct {
	registry *url.URL
	client   *http.Client
}

// NewPublicResolver constructs the production public-registry resolver.
func NewPublicResolver() (*Resolver, error) {
	registry, err := url.Parse(publicRegistryURL)
	if err != nil {
		return nil, fmt.Errorf("parse public npm registry URL: %w", err)
	}
	client := &http.Client{
		Timeout: metadataTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("npm registry redirects are not permitted")
		},
		Transport: &http.Transport{
			Proxy:                 nil,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          4,
			MaxIdleConnsPerHost:   2,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: time.Second,
		},
	}
	return newResolver(registry, client, true)
}

func newResolver(registry *url.URL, client *http.Client, requirePublicHTTPS bool) (*Resolver, error) {
	if registry == nil || client == nil || registry.User != nil || registry.RawQuery != "" || registry.Fragment != "" || (registry.Path != "" && registry.Path != "/") {
		return nil, errors.New("npm registry configuration is invalid")
	}
	if requirePublicHTTPS && (registry.Scheme != "https" || !strings.EqualFold(registry.Hostname(), "registry.npmjs.org") || registry.Port() != "") {
		return nil, errors.New("npm production registry must be https://registry.npmjs.org/")
	}
	configuredRegistry := *registry
	configuredRegistry.Path = "/"
	return &Resolver{registry: &configuredRegistry, client: client}, nil
}

// ParseReference parses one supported npm package locator without resolving it.
func ParseReference(locator string) (domain.ArtifactReference, error) {
	parsed, err := parseLocator(locator)
	if err != nil {
		return domain.ArtifactReference{}, err
	}
	source, err := domain.NewSourceID("npm")
	if err != nil {
		return domain.ArtifactReference{}, err
	}
	return domain.NewArtifactReference(source, parsed.locator)
}

// Resolve converts an npm reference into an exact target and declared integrity descriptor.
func (r *Resolver) Resolve(ctx context.Context, reference domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	if ctx == nil {
		return domain.ResolvedArtifact{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return domain.ResolvedArtifact{}, err
	}
	if r == nil || r.registry == nil || r.client == nil {
		return domain.ResolvedArtifact{}, errors.New("npm resolver is not configured")
	}
	if reference.Source().String() != "npm" {
		return domain.ResolvedArtifact{}, errors.New("npm resolver requires npm Artifact Reference")
	}
	parsed, err := parseLocator(reference.Locator())
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	metadata, err := r.fetchMetadata(ctx, parsed.name)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	if metadata.Name != parsed.name {
		return domain.ResolvedArtifact{}, errors.New("npm metadata name does not match requested package")
	}
	version, err := parsed.selectVersion(metadata)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	selected, found := metadata.Versions[version]
	if !found || selected.Name != parsed.name || selected.Version != version {
		return domain.ResolvedArtifact{}, errors.New("npm metadata selected version is inconsistent")
	}
	if err := r.validateTarballURL(selected.Dist.Tarball); err != nil {
		return domain.ResolvedArtifact{}, err
	}
	identity, err := domain.NewResolvedArtifactIdentity(reference.Source(), selected.Name, selected.Version, "tarball")
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	return domain.NewResolvedArtifact(identity, selected.Dist.Tarball, selected.Dist.Integrity)
}

type npmLocator struct {
	locator  string
	name     string
	selector string
	exact    bool
}

func parseLocator(locator string) (npmLocator, error) {
	if locator == "" || locator != strings.TrimSpace(locator) || strings.ContainsAny(locator, "\\:?#") || (!strings.HasPrefix(locator, "@") && strings.Contains(locator, "/")) {
		return npmLocator{}, errors.New("npm package reference is invalid")
	}
	name, selector := locator, ""
	if strings.HasPrefix(locator, "@") {
		slash := strings.IndexByte(locator, '/')
		if slash <= 1 {
			return npmLocator{}, errors.New("scoped npm package reference is invalid")
		}
		if at := strings.LastIndex(locator[slash+1:], "@"); at >= 0 {
			at += slash + 1
			name, selector = locator[:at], locator[at+1:]
		}
	} else if at := strings.LastIndexByte(locator, '@'); at >= 0 {
		name, selector = locator[:at], locator[at+1:]
	}
	if err := validatePackageName(name); err != nil {
		return npmLocator{}, err
	}
	if selector == "" {
		return npmLocator{locator: locator, name: name, selector: "latest"}, nil
	}
	if exactVersion.MatchString(selector) {
		return npmLocator{locator: locator, name: name, selector: selector, exact: true}, nil
	}
	if !tagPattern.MatchString(selector) {
		return npmLocator{}, errors.New("npm selector must be an exact version or dist-tag")
	}
	return npmLocator{locator: locator, name: name, selector: selector}, nil
}

func validatePackageName(name string) error {
	if strings.HasPrefix(name, "@") {
		parts := strings.Split(name[1:], "/")
		if len(parts) != 2 || !packageSegment.MatchString(parts[0]) || !packageSegment.MatchString(parts[1]) {
			return errors.New("npm scoped package name is invalid")
		}
		return nil
	}
	if !packageSegment.MatchString(name) {
		return errors.New("npm package name is invalid")
	}
	return nil
}

func (locator npmLocator) selectVersion(metadata packument) (string, error) {
	if locator.exact {
		return locator.selector, nil
	}
	version, found := metadata.DistTags[locator.selector]
	if !found || !exactVersion.MatchString(version) {
		return "", errors.New("npm dist-tag does not resolve to an exact version")
	}
	return version, nil
}

func (r *Resolver) fetchMetadata(ctx context.Context, packageName string) (packument, error) {
	endpoint := *r.registry
	endpoint.Path = "/" + packageName
	endpoint.RawPath = "/" + url.PathEscape(packageName)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return packument{}, fmt.Errorf("create npm metadata request: %w", err)
	}
	request.Header.Set("Accept", metadataAccept)
	request.Header.Set("User-Agent", "helox/0")
	response, err := r.client.Do(request)
	if err != nil {
		return packument{}, fmt.Errorf("request npm metadata: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return packument{}, fmt.Errorf("npm metadata returned unexpected status %d", response.StatusCode)
	}
	contentType := response.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "application/json") {
		return packument{}, errors.New("npm metadata response is not JSON")
	}
	body, err := readLimited(response.Body, metadataLimit)
	if err != nil {
		return packument{}, fmt.Errorf("read npm metadata: %w", err)
	}
	var metadata packument
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	if err := decoder.Decode(&metadata); err != nil {
		return packument{}, errors.New("npm metadata is invalid JSON")
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return packument{}, err
	}
	return metadata, nil
}

func (r *Resolver) validateTarballURL(rawURL string) error {
	tarball, err := url.Parse(rawURL)
	if err != nil || tarball.User != nil || tarball.RawQuery != "" || tarball.Fragment != "" || tarball.Scheme != r.registry.Scheme || !strings.EqualFold(tarball.Host, r.registry.Host) {
		return errors.New("npm tarball URL is outside the configured registry")
	}
	return nil
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(reader, limit+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errors.New("response body exceeds configured limit")
	}
	return body, nil
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	}
	return errors.New("npm metadata contains trailing JSON data")
}

type packument struct {
	Name     string                        `json:"name"`
	DistTags map[string]string             `json:"dist-tags"`
	Versions map[string]npmVersionMetadata `json:"versions"`
}

type npmVersionMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dist    struct {
		Tarball   string `json:"tarball"`
		Integrity string `json:"integrity"`
	} `json:"dist"`
}
