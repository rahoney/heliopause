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
	"github.com/rahoney/heliopause/internal/core/ports"
)

const pypiDistributionTimeout = 90 * time.Second

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

func (i *Intake) Acquire(ctx context.Context, runID domain.RunID, resolved domain.ResolvedArtifact) (artifact domain.AcquiredArtifact, resultErr error) {
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
	root, err := os.OpenRoot(i.intakeRoot)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	identity, err := root.Lstat(runID.String())
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	held, err := root.OpenRoot(runID.String())
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, held.Close()) }()
	opened, err := held.Stat(".")
	if err != nil || !os.SameFile(identity, opened) {
		return domain.AcquiredArtifact{}, errors.New("PyPI intake directory changed")
	}
	var lease ports.StorageLease
	successful := false
	defer func() {
		if !successful {
			current, cleanupErr := root.Lstat(runID.String())
			if cleanupErr == nil && os.SameFile(identity, current) {
				// Remove is deliberately non-recursive: unproven payloads stay
				// accounted until their lifetime/ownership can be recovered.
				cleanupErr = root.Remove(runID.String())
				if cleanupErr == nil {
					parent, err := root.Open(".")
					cleanupErr = err
					if err == nil {
						cleanupErr = errors.Join(parent.Sync(), parent.Close())
					}
				}
			} else {
				cleanupErr = errors.Join(errors.New("PyPI intake cleanup ownership is unknown"), cleanupErr)
			}
			resultErr = errors.Join(resultErr, cleanupErr)
			if lease != nil && cleanupErr == nil {
				cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
				resultErr = errors.Join(resultErr, lease.Abort(cleanupCtx))
				cancel()
			}
		}
		if lease != nil {
			resultErr = errors.Join(resultErr, lease.Close())
		}
	}()
	if storage := ports.StorageFromContext(ctx); storage != nil {
		policy, _ := resourcePolicyFromContext(ctx)
		lease, err = storage.Reserve(ctx, ports.StorageRequest{Operation: ports.StorageOperationFromContext(ctx), Run: runID, Kind: "intake", Temporary: runID.String(), Target: runID.String(), Bytes: uint64(policy.MaxArtifactCompressed()) + 1 + uint64(len(filename)), Files: 2})
		if err != nil {
			return domain.AcquiredArtifact{}, err
		}
	}
	_, digest, size, err := i.download(ctx, resolved.AcquisitionLocator(), held, filename, profile)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if digest != strings.TrimPrefix(declared, "sha256:") {
		return domain.AcquiredArtifact{}, errors.New("PyPI acquired SHA-256 differs from resolved integrity")
	}
	file, err := held.OpenFile("filename", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return domain.AcquiredArtifact{}, fmt.Errorf("persist PyPI distribution filename: %w", err)
	}
	_, writeErr := file.WriteString(filename)
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	artifactIdentity := resolved.Identity()
	handle := "intake:" + runID.String() + ":" + artifactIdentity.Variant()
	artifact, err = domain.NewAcquiredArtifactWithDeclaredIntegrity(artifactIdentity, mustSHA256(digest), handle, size, declared)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if lease != nil {
		if err := lease.Commit(ctx); err != nil {
			return domain.AcquiredArtifact{}, err
		}
	}
	successful = true
	return artifact, nil
}

func (i *Intake) download(ctx context.Context, rawURL string, directory *os.Root, filename string, profile SourceProfile) (string, string, uint64, error) {
	if err := validateDistributionURLForSource(rawURL, filename, profile, false); err != nil {
		return "", "", 0, errors.New("PyPI distribution URL is invalid")
	}
	policy, session := resourcePolicyFromContext(ctx)
	timeout := pypiDistributionTimeout
	if policy.Duration() > defaultResourcePolicy().Duration() {
		timeout = policy.Duration()
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
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
	if CheckTemporaryDisk(directory.Name(), policy) != nil || response.ContentLength > policy.MaxArtifactCompressed() || session.beginArtifact(response.ContentLength) != nil {
		return "", "", 0, errors.New("PyPI distribution exceeds resource budget")
	}
	variantFile := map[string]string{"wheel": "wheel.whl", "sdist": "sdist.tar.gz"}[distributionVariantName(filename)]
	if variantFile == "" {
		return "", "", 0, errors.New("PyPI distribution type is unsupported")
	}
	temporary, err := directory.OpenFile(".download", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", "", 0, err
	}
	temporaryPath := ".download"
	defer directory.Remove(temporaryPath)
	hash := sha256.New()
	writer := io.MultiWriter(temporary, hash, resourceBudgetWriter{session: session})
	written, copyErr := io.Copy(writer, ports.DownloadReader(ctx, io.LimitReader(response.Body, policy.MaxArtifactCompressed()+1), response.ContentLength))
	syncErr, closeErr := temporary.Sync(), temporary.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil || written == 0 || written > policy.MaxArtifactCompressed() {
		return "", "", 0, errors.New("stream PyPI distribution failed or exceeded bounds")
	}
	if err := directory.Rename(temporaryPath, variantFile); err != nil {
		return "", "", 0, err
	}
	return variantFile, hex.EncodeToString(hash.Sum(nil)), uint64(written), nil
}

type resourceBudgetWriter struct{ session *resourceSession }

func (w resourceBudgetWriter) Write(body []byte) (int, error) {
	if err := w.session.charge(int64(len(body))); err != nil {
		return 0, err
	}
	return len(body), nil
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
