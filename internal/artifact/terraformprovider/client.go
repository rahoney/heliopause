package terraformprovider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const MaxProviderArchiveBytes = 64 << 20

// Client freezes public discovery, version and package bytes once per exact
// selection. Acquire consumes that selection, never performs a new resolve,
// and gives the independent verifier the signed checksum and archive bytes.
type Client struct {
	intakeRoot string
	resolver   *Resolver
	releases   *githubrelease.Client
	mu         sync.Mutex
	frozen     map[string]RegistrySnapshot
}

func NewPublicClient(intakeRoot string) (*Client, error) {
	if !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || intakeRoot == "/" {
		return nil, errors.New("terraform intake root must be absolute and canonical")
	}
	resolver, err := NewPublicResolver()
	if err != nil {
		return nil, err
	}
	releases, err := githubrelease.NewPublicClient(intakeRoot)
	if err != nil {
		return nil, err
	}
	return &Client{intakeRoot: intakeRoot, resolver: resolver, releases: releases, frozen: map[string]RegistrySnapshot{}}, nil
}

func (c *Client) Resolve(ctx context.Context, reference domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	if c == nil || c.resolver == nil || ctx == nil {
		return domain.ResolvedArtifact{}, errors.New("terraform selection request is invalid")
	}
	snapshot, artifact, err := c.resolver.ResolveSnapshot(ctx, reference)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	encoded, err := encodeParts(registryHeader, registryParts(snapshot), registryLimits)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	key := sha256Hex(encoded)
	graph, err := BuildLockedGraph(artifact)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	identity := graph.Nodes()[0].Artifact().Identity()
	resolved, err := domain.NewResolvedArtifact(identity, "terraform-registry:"+key, "registry="+key+";archive="+artifact.SHA256)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.frozen) >= 64 && c.frozen[key].Package == nil {
		return domain.ResolvedArtifact{}, errors.New("terraform frozen selection count exceeds bound")
	}
	c.frozen[key] = snapshot
	return resolved, nil
}

func (c *Client) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, _ domain.InstallContext) (domain.DependencyResolution, error) {
	resolved, err := c.Resolve(ctx, reference)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	key := strings.TrimPrefix(resolved.AcquisitionLocator(), "terraform-registry:")
	nodeID, _ := domain.NewDependencyNodeID("t" + key[:24])
	node, err := domain.NewLockedDependency(nodeID, domain.DependencyPrimary, resolved)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	digest, _ := domain.NewSHA256Digest(key)
	return domain.NewDependencyResolution(graph, "terraform-registry:registry.terraform.io;platform:linux/amd64", digest)
}

func (c *Client) Acquire(ctx context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (result domain.AcquiredArtifact, resultErr error) {
	if c == nil || c.resolver == nil || c.releases == nil || ctx == nil || run.String() == "" {
		return result, errors.New("terraform acquisition request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	reference, err := IdentityReference(resolved.Identity())
	if err != nil {
		return result, err
	}
	key, archiveDigest, err := ParseIntegrity(resolved.DeclaredIntegrity())
	if err != nil || resolved.AcquisitionLocator() != "terraform-registry:"+key {
		return result, errors.New("terraform acquisition selection binding is invalid")
	}
	c.mu.Lock()
	snapshot, ok := c.frozen[key]
	c.mu.Unlock()
	if !ok {
		return result, errors.New("terraform acquisition has no frozen selection")
	}
	encoded, err := encodeParts(registryHeader, registryParts(snapshot), registryLimits)
	if err != nil || sha256Hex(encoded) != key {
		return result, errors.New("terraform frozen registry selection changed")
	}
	artifact, err := snapshot.Parse(reference, Platform{OS: "linux", Arch: "amd64"})
	if err != nil || artifact.SHA256 != archiveDigest {
		return result, errors.New("terraform acquisition differs from selected package")
	}
	acquireCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	sums, err := c.download(acquireCtx, artifact.ShasumsURL, 1<<20)
	if err != nil {
		return result, err
	}
	signature, err := c.download(acquireCtx, artifact.SignatureURL, 16<<10)
	if err != nil {
		return result, err
	}
	archive, err := c.download(acquireCtx, artifact.DownloadURL, MaxProviderArchiveBytes)
	if err != nil {
		return result, err
	}
	body, err := encodeParts(bundleHeader, [][]byte{snapshot.Discovery, snapshot.Versions, snapshot.Package, sums, signature, archive}, bundleLimits)
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(c.intakeRoot, 0o700); err != nil {
		return result, err
	}
	root, err := openPrivateRoot(c.intakeRoot)
	if err != nil {
		return result, err
	}
	defer root.Close()
	if err := root.Mkdir(run.String(), 0o700); err != nil {
		return result, err
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, root.RemoveAll(run.String()))
			result = domain.AcquiredArtifact{}
		}
	}()
	directory, err := root.OpenRoot(run.String())
	if err != nil {
		return result, err
	}
	defer directory.Close()
	file, err := directory.OpenFile("provider.bundle", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return result, err
	}
	_, writeErr := file.Write(body)
	syncErr, closeErr := file.Sync(), file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return result, err
	}
	digest, _ := domain.NewSHA256Digest(sha256Hex(body))
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), digest, "intake:"+run.String()+":"+resolved.Identity().Variant(), uint64(len(body)), resolved.DeclaredIntegrity())
}

func (c *Client) download(ctx context.Context, endpoint string, limit int64) ([]byte, error) {
	if err := trustedURL(endpoint); err != nil {
		return nil, err
	}
	if strings.HasPrefix(endpoint, "https://github.com/") {
		return c.releases.ReadPublicReleaseFile(ctx, endpoint, limit)
	}
	if !strings.HasPrefix(endpoint, "https://releases.hashicorp.com/") {
		return nil, errors.New("terraform download endpoint is unsupported")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New("terraform download request is invalid")
	}
	request.Header.Set("User-Agent", "helox/0")
	client := *c.resolver.client
	client.Timeout = 90 * time.Second
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Join(errors.New("terraform exact download request failed"), ctx.Err())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength > limit {
		return nil, errors.New("terraform exact download response is unexpected")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || len(body) == 0 || int64(len(body)) > limit || (response.ContentLength >= 0 && response.ContentLength != int64(len(body))) {
		return nil, errors.Join(errors.New("terraform download is incomplete or exceeds bounds"), ctx.Err())
	}
	return body, nil
}

func sha256Hex(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func IdentityReference(identity domain.ResolvedArtifactIdentity) (domain.ArtifactReference, error) {
	if identity.Source() != Source() || identity.Variant() != "linux/amd64" || strings.Count(identity.Name(), "_") != 1 {
		return domain.ArtifactReference{}, errors.New("terraform provider identity is invalid")
	}
	return ParseReference(strings.Replace(identity.Name(), "_", "/", 1) + "@" + identity.Version())
}

func ParseIntegrity(value string) (registry, archive string, err error) {
	left, right, ok := strings.Cut(value, ";archive=")
	registry = strings.TrimPrefix(left, "registry=")
	archive = right
	if !ok || left != "registry="+registry || !isSHA256(registry) || !isSHA256(archive) || strings.ToLower(value) != value {
		return "", "", errors.New("terraform frozen integrity declaration is invalid")
	}
	return registry, archive, nil
}
