package pypi

// This file owns the public PyPI distribution intake boundary.  The resolver
// has already cross-checked the exact URL and SHA-256; this adapter streams
// those bytes into a Run-private directory and retains only the validated
// distribution filename needed by the later static and Promotion adapters.

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
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	pypiDistributionLimit   = 64 << 20
	pypiDistributionTimeout = 90 * time.Second
)

// Intake acquires only already-resolved public PyPI distributions.
type Intake struct {
	client     *http.Client
	intakeRoot string
}

func NewPublicIntake(intakeRoot string) (*Intake, error) {
	if !filepath.IsAbs(intakeRoot) {
		return nil, errors.New("PyPI intake root must be absolute")
	}
	return &Intake{intakeRoot: filepath.Clean(intakeRoot), client: &http.Client{
		Timeout: 0,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("PyPI distribution redirects are not permitted")
		},
		Transport: &http.Transport{Proxy: nil, ForceAttemptHTTP2: true, MaxIdleConns: 4, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second},
	}}, nil
}

func (i *Intake) Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	return domain.ResolvedArtifact{}, errors.New("PyPI intake cannot resolve a reference; use the isolated resolver")
}

func (i *Intake) Acquire(ctx context.Context, runID domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if ctx == nil || ctx.Err() != nil || i == nil || i.client == nil || runID.String() == "" {
		return domain.AcquiredArtifact{}, errors.New("PyPI intake request is invalid")
	}
	profile, ok := ProfileForSource(resolved.Identity().Source())
	if !ok {
		return domain.AcquiredArtifact{}, errors.New("PyPI intake source profile is unsupported")
	}
	filename, err := pypiResolvedFilename(resolved)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	declared := resolved.DeclaredIntegrity()
	if !strings.HasPrefix(declared, "sha256:") || !validSHA256(strings.TrimPrefix(declared, "sha256:")) {
		return domain.AcquiredArtifact{}, errors.New("PyPI resolved Artifact lacks SHA-256 integrity")
	}
	if err := os.MkdirAll(i.intakeRoot, 0o700); err != nil {
		return domain.AcquiredArtifact{}, fmt.Errorf("create PyPI intake root: %w", err)
	}
	if info, err := os.Lstat(i.intakeRoot); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return domain.AcquiredArtifact{}, errors.New("PyPI intake root is untrusted")
	}
	directory := filepath.Join(i.intakeRoot, runID.String())
	if filepath.Dir(directory) != i.intakeRoot {
		return domain.AcquiredArtifact{}, errors.New("PyPI intake Run directory escapes root")
	}
	if err := os.Mkdir(directory, 0o700); err != nil {
		return domain.AcquiredArtifact{}, fmt.Errorf("create PyPI intake Run directory: %w", err)
	}
	cleanup := func(cause error) (domain.AcquiredArtifact, error) {
		if removeErr := os.RemoveAll(directory); removeErr != nil {
			return domain.AcquiredArtifact{}, errors.Join(cause, fmt.Errorf("remove incomplete PyPI intake: %w", removeErr))
		}
		return domain.AcquiredArtifact{}, cause
	}
	file, digest, size, err := i.download(ctx, resolved.AcquisitionLocator(), directory, filename, profile)
	if err != nil {
		return cleanup(err)
	}
	if digest != strings.TrimPrefix(declared, "sha256:") {
		return cleanup(errors.New("PyPI acquired SHA-256 differs from resolved integrity"))
	}
	if err := os.WriteFile(filepath.Join(directory, "filename"), []byte(filename), 0o400); err != nil {
		return cleanup(fmt.Errorf("persist PyPI distribution filename: %w", err))
	}
	identity := resolved.Identity()
	handle := "intake:" + runID.String() + ":" + identity.Variant()
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, mustSHA256(digest), handle, size, declared)
	if err != nil {
		return cleanup(err)
	}
	_ = file
	return artifact, nil
}

func (i *Intake) download(ctx context.Context, rawURL, directory, filename string, profile SourceProfile) (string, string, uint64, error) {
	if err := validateDistributionURLForSource(rawURL, filename, profile, false); err != nil {
		return "", "", 0, errors.New("PyPI distribution URL is invalid")
	}
	requestCtx, cancel := context.WithTimeout(ctx, pypiDistributionTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", 0, err
	}
	response, err := i.client.Do(request)
	if err != nil {
		return "", "", 0, errors.New("request PyPI distribution")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", "", 0, errors.New("PyPI distribution returned unexpected status")
	}
	variantFile := map[string]string{"wheel": "wheel.whl", "sdist": "sdist.tar.gz"}[distributionVariantName(filename)]
	if variantFile == "" {
		return "", "", 0, errors.New("PyPI distribution type is unsupported")
	}
	temporary, err := os.OpenFile(filepath.Join(directory, ".download"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", "", 0, err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(temporary, hash), io.LimitReader(response.Body, pypiDistributionLimit+1))
	syncErr, closeErr := temporary.Sync(), temporary.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil || written == 0 || written > pypiDistributionLimit {
		return "", "", 0, errors.New("stream PyPI distribution failed or exceeded bounds")
	}
	if err := os.Rename(temporaryPath, filepath.Join(directory, variantFile)); err != nil {
		return "", "", 0, err
	}
	return variantFile, hex.EncodeToString(hash.Sum(nil)), uint64(written), nil
}

func pypiResolvedFilename(resolved domain.ResolvedArtifact) (string, error) {
	parsed, err := url.Parse(resolved.AcquisitionLocator())
	if err != nil {
		return "", errors.New("PyPI distribution URL is invalid")
	}
	filename := path.Base(parsed.Path)
	variant, ok := distributionVariant(filename)
	if !ok || variant != resolved.Identity().Variant() {
		return "", errors.New("PyPI distribution filename does not match identity")
	}
	return filename, nil
}

func distributionVariantName(filename string) string {
	variant, _ := distributionVariant(filename)
	return variant
}
func mustSHA256(value string) domain.ContentDigest {
	digest, _ := domain.NewSHA256Digest(value)
	return digest
}
