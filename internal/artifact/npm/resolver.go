// Package npm implements the public npm registry boundary.
package npm

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	publicRegistryURL = "https://registry.npmjs.org/"
	metadataLimit     = 2 << 20
	metadataTimeout   = 10 * time.Second
	tarballLimit      = 50 << 20
	tarballTimeout    = 60 * time.Second
	metadataAccept    = "application/vnd.npm.install-v1+json"
)

var (
	packageSegment = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	tagPattern     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
	exactVersion   = regexp.MustCompile(`^(?:v)?(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
)

// Resolver resolves only unauthenticated package references against the public npm registry.
type Resolver struct {
	registry   *url.URL
	client     *http.Client
	intakeRoot string
}

// NewPublicResolver constructs the production public-registry resolver.
func NewPublicResolver(intakeRoot string) (*Resolver, error) {
	registry, err := url.Parse(publicRegistryURL)
	if err != nil {
		return nil, fmt.Errorf("parse public npm registry URL: %w", err)
	}
	client := &http.Client{
		Timeout: 0,
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
	return newResolver(registry, client, intakeRoot, true)
}

func newResolver(registry *url.URL, client *http.Client, intakeRoot string, requirePublicHTTPS bool) (*Resolver, error) {
	if registry == nil || client == nil || registry.User != nil || registry.RawQuery != "" || registry.Fragment != "" || (registry.Path != "" && registry.Path != "/") {
		return nil, errors.New("npm registry configuration is invalid")
	}
	if requirePublicHTTPS && (registry.Scheme != "https" || !strings.EqualFold(registry.Hostname(), "registry.npmjs.org") || registry.Port() != "") {
		return nil, errors.New("npm production registry must be https://registry.npmjs.org/")
	}
	configuredRegistry := *registry
	configuredRegistry.Path = "/"
	return &Resolver{registry: &configuredRegistry, client: client, intakeRoot: intakeRoot}, nil
}

// Acquire streams a resolved npm tarball into the configured Run-local controlled intake root.
func (r *Resolver) Acquire(ctx context.Context, runID domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if ctx == nil {
		return domain.AcquiredArtifact{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if r == nil || r.client == nil || r.intakeRoot == "" {
		return domain.AcquiredArtifact{}, errors.New("npm intake is not configured")
	}
	if resolved.Identity().Source().String() != "npm" {
		return domain.AcquiredArtifact{}, errors.New("npm intake requires npm resolved Artifact")
	}
	if err := r.validateTarballURL(resolved.AcquisitionLocator()); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if runID.String() == "" {
		return domain.AcquiredArtifact{}, errors.New("npm intake requires Run ID")
	}
	runDirectory, err := createRunDirectory(r.intakeRoot, runID.String())
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	path, digest, observedIntegrity, size, err := r.downloadTarball(ctx, resolved.AcquisitionLocator(), runDirectory)
	if err != nil {
		if cleanupErr := os.RemoveAll(runDirectory); cleanupErr != nil {
			return domain.AcquiredArtifact{}, errors.Join(err, fmt.Errorf("remove incomplete npm intake: %w", cleanupErr))
		}
		return domain.AcquiredArtifact{}, err
	}
	_ = path
	return domain.NewAcquiredArtifactWithIntegrity(resolved.Identity(), digest, "intake:"+runID.String()+":tarball", size, resolved.DeclaredIntegrity(), observedIntegrity)
}

func createRunDirectory(root, runID string) (string, error) {
	cleanRoot := filepath.Clean(root)
	if !filepath.IsAbs(cleanRoot) {
		return "", errors.New("npm intake root must be absolute")
	}
	if err := rejectSymlinkComponents(cleanRoot, true); err != nil {
		return "", err
	}
	if err := os.MkdirAll(cleanRoot, 0o700); err != nil {
		return "", fmt.Errorf("create npm intake root: %w", err)
	}
	if err := rejectSymlinkComponents(cleanRoot, false); err != nil {
		return "", err
	}
	info, err := os.Lstat(cleanRoot)
	if err != nil || !info.IsDir() {
		return "", errors.New("npm intake root is not a trusted directory")
	}
	runDirectory := filepath.Join(cleanRoot, runID)
	if filepath.Dir(runDirectory) != cleanRoot {
		return "", errors.New("npm intake Run directory escapes root")
	}
	if err := os.Mkdir(runDirectory, 0o700); err != nil {
		return "", fmt.Errorf("create npm intake Run directory: %w", err)
	}
	return runDirectory, nil
}

func rejectSymlinkComponents(path string, allowMissing bool) error {
	volume := filepath.VolumeName(path)
	filesystemRoot := volume + string(filepath.Separator)
	relative, err := filepath.Rel(filesystemRoot, path)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("npm intake root is not a trusted directory")
	}
	current := filesystemRoot
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) && allowMissing {
			return nil
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("npm intake root contains a symbolic link or unavailable component")
		}
	}
	return nil
}

func (r *Resolver) downloadTarball(ctx context.Context, rawURL, directory string) (string, domain.ContentDigest, string, uint64, error) {
	timeoutContext, cancel := context.WithTimeout(ctx, tarballTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(timeoutContext, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", domain.ContentDigest{}, "", 0, fmt.Errorf("create npm tarball request: %w", err)
	}
	response, err := r.client.Do(request)
	if err != nil {
		return "", domain.ContentDigest{}, "", 0, fmt.Errorf("request npm tarball: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", domain.ContentDigest{}, "", 0, fmt.Errorf("npm tarball returned unexpected status %d", response.StatusCode)
	}
	if !isTarballContentType(response.Header.Get("Content-Type")) {
		return "", domain.ContentDigest{}, "", 0, errors.New("npm tarball response has an unsupported content type")
	}
	temporary, err := os.OpenFile(filepath.Join(directory, ".tarball.tmp"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", domain.ContentDigest{}, "", 0, fmt.Errorf("create npm intake temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	hash := sha256.New()
	integrityHash := sha512.New()
	limited := io.LimitReader(response.Body, tarballLimit+1)
	written, copyErr := io.Copy(io.MultiWriter(temporary, hash, integrityHash), limited)
	syncErr := temporary.Sync()
	closeErr := temporary.Close()
	if copyErr != nil {
		return "", domain.ContentDigest{}, "", 0, fmt.Errorf("stream npm tarball: %w", copyErr)
	}
	if written > tarballLimit {
		return "", domain.ContentDigest{}, "", 0, errors.New("npm tarball exceeds configured limit")
	}
	if syncErr != nil {
		return "", domain.ContentDigest{}, "", 0, fmt.Errorf("sync npm intake temporary file: %w", syncErr)
	}
	if closeErr != nil {
		return "", domain.ContentDigest{}, "", 0, fmt.Errorf("close npm intake temporary file: %w", closeErr)
	}
	if err := os.Rename(temporaryPath, filepath.Join(directory, "tarball.tgz")); err != nil {
		return "", domain.ContentDigest{}, "", 0, fmt.Errorf("finalize npm intake tarball: %w", err)
	}
	digest, err := domain.NewSHA256Digest(fmt.Sprintf("%x", hash.Sum(nil)))
	if err != nil {
		return "", domain.ContentDigest{}, "", 0, err
	}
	return filepath.Join(directory, "tarball.tgz"), digest, "sha512-" + base64.StdEncoding.EncodeToString(integrityHash.Sum(nil)), uint64(written), nil
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
	timeoutContext, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(timeoutContext, http.MethodGet, endpoint.String(), nil)
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
	if err != nil || tarball.User != nil || tarball.RawQuery != "" || tarball.Fragment != "" || tarball.Path == "" || tarball.Scheme != r.registry.Scheme || !strings.EqualFold(tarball.Host, r.registry.Host) {
		return errors.New("npm tarball URL is outside the configured registry")
	}
	return nil
}

func isTarballContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}
	switch strings.ToLower(mediaType) {
	case "application/octet-stream", "application/gzip", "application/x-gzip":
		return true
	default:
		return false
	}
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
