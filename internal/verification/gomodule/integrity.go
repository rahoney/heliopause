// Package gomodule authenticates frozen public Go module bytes against the
// pinned SumDB signing key and transparency proofs, independently of go.sum.
package gomodule

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/sumdb"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const sumDBKey = "sum.golang.org+033de0ae+Ac4zctda0e5eza+HJyk9SxEdh+s3Ux18htTTAD8OuAn8"

type IntegrityVerifier struct {
	intakeRoot string
	http       *http.Client
	key        string
}

// NewIntegrityVerifier binds a canonical controlled intake and the fixed public signing key.
func NewIntegrityVerifier(intakeRoot string) (*IntegrityVerifier, error) {
	if !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || intakeRoot == "/" {
		return nil, errors.New("go verifier intake root is invalid")
	}
	return &IntegrityVerifier{intakeRoot: intakeRoot, http: artifactgo.PublicHTTPClient(), key: sumDBKey}, nil
}

func (v *IntegrityVerifier) Verify(ctx context.Context, artifact domain.AcquiredArtifact) (domain.VerificationReport, error) {
	if ctx == nil || ctx.Err() != nil || v == nil || v.http == nil || artifact.Identity().Source() != artifactgo.Source() {
		return domain.VerificationReport{}, errors.New("go module verification request is invalid")
	}
	bundle, err := artifactgo.ReadIntake(v.intakeRoot, artifact)
	if err != nil {
		return domain.VerificationReport{}, err
	}
	archiveSum, modSum, err := artifactgo.HashBundle(bundle, artifact.Identity())
	if err != nil {
		return verificationReport(artifact, false, "M12_GO_MODULE_CONTENT_INVALID")
	}
	declared, present := artifact.DeclaredIntegrity()
	if present {
		expectedZip, expectedMod, err := artifactgo.ParseIntegrity(declared)
		if err != nil || archiveSum != expectedZip || modSum != expectedMod {
			return verificationReport(artifact, false, "M12_GO_DECLARED_SUM_MISMATCH")
		}
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	ops := &databaseOps{ctx: lookupCtx, http: v.http, key: v.key}
	client := sumdb.NewClient(ops)
	name, version := artifact.Identity().Name(), artifact.Identity().Version()
	zipLines, err := client.Lookup(name, version)
	if err != nil {
		return domain.VerificationReport{}, errors.New("go SumDB authenticated archive lookup failed")
	}
	modLines, err := client.Lookup(name, version+"/go.mod")
	if err != nil {
		return domain.VerificationReport{}, errors.New("go SumDB authenticated control lookup failed")
	}
	if !exactLine(zipLines, name+" "+version+" "+archiveSum) || !exactLine(modLines, name+" "+version+"/go.mod "+modSum) {
		return verificationReport(artifact, false, "M12_GO_SUMDB_MISMATCH")
	}
	return verificationReport(artifact, true, "")
}

func exactLine(lines []string, expected string) bool {
	count := 0
	for _, line := range lines {
		if line == expected {
			count++
		}
	}
	return count == 1
}

func verificationReport(artifact domain.AcquiredArtifact, verified bool, code string) (domain.VerificationReport, error) {
	checkID, _ := domain.NewCheckID("go-module-sumdb")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("go-module-sumdb-result")
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "go-module-sumdb", "Exact proxy module envelope h1 identities checked against authenticated SumDB records.")
	if err != nil {
		return domain.VerificationReport{}, err
	}
	if verified {
		return domain.NewVerificationReport(check, domain.VerificationVerified, []domain.Evidence{evidence})
	}
	finding, err := domain.NewFinding(code, []domain.EvidenceID{evidenceID})
	if err != nil {
		return domain.VerificationReport{}, err
	}
	return domain.NewVerificationReportWithFindings(check, domain.VerificationMismatch, []domain.Finding{finding}, []domain.Evidence{evidence})
}

type databaseOps struct {
	ctx      context.Context
	http     *http.Client
	key      string
	mu       sync.Mutex
	latest   []byte
	requests int
	bytes    int64
}

func (o *databaseOps) ReadRemote(relative string) ([]byte, error) {
	if len(relative) > 2048 || (!strings.HasPrefix(relative, "/lookup/") && !strings.HasPrefix(relative, "/tile/")) || relative != path.Clean(relative) || strings.ContainsAny(relative, "?#\\\r\n") {
		return nil, errors.New("go SumDB path is invalid")
	}
	o.mu.Lock()
	o.requests++
	allowed := o.requests <= 64
	o.mu.Unlock()
	if !allowed {
		return nil, errors.New("go SumDB request bound exceeded")
	}
	req, err := http.NewRequestWithContext(o.ctx, http.MethodGet, "https://sum.golang.org"+relative, nil)
	if err != nil {
		return nil, errors.New("go SumDB request is invalid")
	}
	response, err := o.http.Do(req)
	if err != nil {
		return nil, errors.New("go SumDB request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength > 1<<20 {
		return nil, errors.New("go SumDB response is invalid")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(body) == 0 || len(body) > 1<<20 {
		return nil, errors.New("go SumDB response exceeds bounds")
	}
	o.mu.Lock()
	o.bytes += int64(len(body))
	allowed = o.bytes <= 4<<20
	o.mu.Unlock()
	if !allowed {
		return nil, errors.New("go SumDB total response bound exceeded")
	}
	return body, nil
}
func (o *databaseOps) ReadConfig(file string) ([]byte, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	switch file {
	case "key":
		return []byte(o.key), nil
	case "sum.golang.org/latest":
		return bytes.Clone(o.latest), nil
	}
	return nil, errors.New("go SumDB configuration name is invalid")
}
func (o *databaseOps) WriteConfig(file string, old, next []byte) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if file != "sum.golang.org/latest" || len(next) > 8192 {
		return errors.New("go SumDB configuration update is invalid")
	}
	if !bytes.Equal(old, o.latest) {
		return sumdb.ErrWriteConflict
	}
	o.latest = bytes.Clone(next)
	return nil
}
func (*databaseOps) ReadCache(string) ([]byte, error) {
	return nil, errors.New("operation-private SumDB cache miss")
}
func (*databaseOps) WriteCache(string, []byte) {}
func (*databaseOps) Log(string)                {}
func (*databaseOps) SecurityError(string)      {} // Client returns ErrSecurity; raw peer text is not published.
