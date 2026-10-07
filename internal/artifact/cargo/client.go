package cargo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const MaxCrateBytes = 64 << 20

type Client struct {
	intakeRoot string
	http       *http.Client
}

func PublicHTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{Proxy: nil, ForceAttemptHTTP2: true, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second, IdleConnTimeout: 30 * time.Second, MaxIdleConns: 4, MaxIdleConnsPerHost: 2}}
}

func NewPublicClient(intakeRoot string) (*Client, error) {
	if !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || intakeRoot == "/" {
		return nil, errors.New("cargo intake root must be absolute and canonical")
	}
	return &Client{intakeRoot: intakeRoot, http: PublicHTTPClient()}, nil
}

func (c *Client) Resolve(ctx context.Context, reference domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	if c == nil || ctx == nil || reference.Source() != Source() {
		return domain.ResolvedArtifact{}, errors.New("cargo exact resolution request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return domain.ResolvedArtifact{}, err
	}
	parsed, err := ParseReference(reference.Locator())
	if err != nil || parsed != reference {
		return domain.ResolvedArtifact{}, errors.New("cargo exact reference is invalid")
	}
	name, version, _ := splitReference(reference)
	checksum, err := c.LookupChecksum(ctx, name, version)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	identity, err := domain.NewResolvedArtifactIdentity(Source(), name, version, "crate")
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	locator, err := DownloadURL(name, version)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	return domain.NewResolvedArtifact(identity, locator, "sha256="+checksum)
}

// LookupChecksum always consults the fixed HTTPS public index. It does not use
// Cargo's cache, project lock, source replacement, token, or ambient proxy.
func (c *Client) LookupChecksum(ctx context.Context, name, version string) (string, error) {
	if c == nil || c.http == nil || ctx == nil {
		return "", errors.New("cargo registry request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	endpoint, err := IndexURL(name)
	if err != nil {
		return "", err
	}
	if _, err := ParseReference(name + "@" + version); err != nil {
		return "", err
	}
	config, err := c.read(ctx, "https://index.crates.io/config.json", 64<<10)
	if err != nil {
		return "", err
	}
	if err := ValidateRegistryConfig(config); err != nil {
		return "", err
	}
	body, err := c.read(ctx, endpoint, MaxIndexBytes)
	if err != nil {
		return "", err
	}
	return ParseIndexChecksum(body, name, version)
}

func splitReference(reference domain.ArtifactReference) (string, string, error) {
	for index, character := range reference.Locator() {
		if character == '@' {
			return reference.Locator()[:index], reference.Locator()[index+1:], nil
		}
	}
	return "", "", errors.New("cargo exact reference is invalid")
}

func (c *Client) Acquire(ctx context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (result domain.AcquiredArtifact, resultErr error) {
	if c == nil || ctx == nil || c.http == nil || run.String() == "" || resolved.Identity().Source() != Source() || resolved.Identity().Variant() != "crate" {
		return result, errors.New("cargo acquisition request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	endpoint, err := DownloadURL(resolved.Identity().Name(), resolved.Identity().Version())
	if err != nil || endpoint != resolved.AcquisitionLocator() {
		return result, errors.New("cargo acquisition locator is not canonical")
	}
	if _, err := ParseIntegrity(resolved.DeclaredIntegrity()); err != nil {
		return result, err
	}
	body, err := c.read(ctx, endpoint, MaxCrateBytes)
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(c.intakeRoot, 0o700); err != nil {
		return result, err
	}
	root, err := openPrivateIntake(c.intakeRoot)
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
	file, err := root.OpenFile(run.String()+"/package.crate", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return result, err
	}
	_, writeErr := file.Write(body)
	syncErr, closeErr := file.Sync(), file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return result, err
	}
	digest := sha256.Sum256(body)
	content, err := domain.NewSHA256Digest(hex.EncodeToString(digest[:]))
	if err != nil {
		return result, err
	}
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), content, "intake:"+run.String()+":crate", uint64(len(body)), resolved.DeclaredIntegrity())
}

func (c *Client) read(ctx context.Context, endpoint string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New("cargo public request is invalid")
	}
	request.Header.Set("User-Agent", "helox/0")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, errors.Join(errors.New("cargo public request failed"), ctx.Err())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength > limit {
		return nil, errors.New("cargo public response is unexpected")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || len(body) == 0 || int64(len(body)) > limit || (response.ContentLength >= 0 && int64(len(body)) != response.ContentLength) {
		return nil, errors.Join(errors.New("cargo public response is incomplete or excessive"), ctx.Err())
	}
	return body, nil
}
