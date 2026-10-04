package gomodule

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

// Client acquires only exact canonical public proxy URLs, without ambient
// credentials, proxies, redirects, VCS fallback or a user's module cache.
type Client struct {
	intakeRoot string
	http       *http.Client
}

func NewPublicClient(intakeRoot string) (*Client, error) {
	if !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || intakeRoot == "/" {
		return nil, errors.New("go intake root must be absolute and canonical")
	}
	return &Client{intakeRoot: intakeRoot, http: PublicHTTPClient()}, nil
}

// PublicHTTPClient deliberately has no proxy or redirect discovery.
func PublicHTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{Proxy: nil, ForceAttemptHTTP2: true, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second, IdleConnTimeout: 30 * time.Second, MaxIdleConns: 4, MaxIdleConnsPerHost: 2}}
}

// Resolve is exact-only. Integrity declarations come from the frozen graph;
// calling this method never selects a newer version or authenticates content.
func (c *Client) Resolve(ctx context.Context, reference domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	if ctx == nil || ctx.Err() != nil || c == nil || reference.Source() != Source() {
		return domain.ResolvedArtifact{}, errors.New("go exact resolution request is invalid")
	}
	ref, err := ParseReference(reference.Locator())
	if err != nil || ref != reference {
		return domain.ResolvedArtifact{}, errors.New("go exact reference is invalid")
	}
	name, version, err := splitReference(ref)
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	identity, err := domain.NewResolvedArtifactIdentity(Source(), name, version, "module")
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	locator, err := ProxyURL(name, version, ".zip")
	if err != nil {
		return domain.ResolvedArtifact{}, err
	}
	return domain.NewResolvedArtifact(identity, locator, "")
}

func splitReference(ref domain.ArtifactReference) (string, string, error) {
	for i, ch := range ref.Locator() {
		if ch == '@' {
			return ref.Locator()[:i], ref.Locator()[i+1:], nil
		}
	}
	return "", "", errors.New("go exact reference is invalid")
}

func (c *Client) Acquire(ctx context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (result domain.AcquiredArtifact, err error) {
	if ctx == nil || ctx.Err() != nil || c == nil || c.http == nil || run.String() == "" || resolved.Identity().Source() != Source() || resolved.Identity().Variant() != "module" {
		return result, errors.New("go module acquisition request is invalid")
	}
	zipURL, err := ProxyURL(resolved.Identity().Name(), resolved.Identity().Version(), ".zip")
	if err != nil || zipURL != resolved.AcquisitionLocator() {
		return result, errors.New("go module acquisition locator is not canonical")
	}
	if resolved.DeclaredIntegrity() != "" {
		if _, _, err := ParseIntegrity(resolved.DeclaredIntegrity()); err != nil {
			return result, err
		}
	}
	modURL, err := ProxyURL(resolved.Identity().Name(), resolved.Identity().Version(), ".mod")
	if err != nil {
		return result, err
	}
	mod, err := c.read(ctx, modURL, MaxModBytes)
	if err != nil {
		return result, err
	}
	archive, err := c.read(ctx, zipURL, MaxZipBytes)
	if err != nil {
		return result, err
	}
	body, err := encodeBundle(mod, archive)
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(c.intakeRoot, 0o700); err != nil {
		return result, err
	}
	info, err := os.Lstat(c.intakeRoot)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return result, errors.New("go module intake root is untrusted")
	}
	root, err := os.OpenRoot(c.intakeRoot)
	if err != nil {
		return result, err
	}
	defer root.Close()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return result, errors.New("go module intake root changed")
	}
	if err := root.Mkdir(run.String(), 0o700); err != nil {
		return result, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, root.RemoveAll(run.String()))
			result = domain.AcquiredArtifact{}
		}
	}()
	f, err := root.OpenFile(run.String()+"/module.bundle", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return result, err
	}
	_, writeErr := f.Write(body)
	syncErr := f.Sync()
	closeErr := f.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return result, err
	}
	digest := sha256.Sum256(body)
	content, err := domain.NewSHA256Digest(hex.EncodeToString(digest[:]))
	if err != nil {
		return result, err
	}
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), content, "intake:"+run.String()+":module", uint64(len(body)), resolved.DeclaredIntegrity())
}

func (c *Client) read(ctx context.Context, endpoint string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New("go proxy request is invalid")
	}
	req.Header.Set("User-Agent", "helox/0")
	response, err := c.http.Do(req)
	if err != nil {
		return nil, errors.New("go proxy request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength > limit {
		return nil, errors.New("go proxy returned unexpected response")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || len(body) == 0 || int64(len(body)) > limit {
		return nil, errors.New("go proxy content exceeds bounds or is incomplete")
	}
	return body, nil
}
