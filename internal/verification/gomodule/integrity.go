// Package gomodule authenticates frozen public Go module bytes against the
// pinned SumDB signing key and transparency proofs, independently of go.sum.
package gomodule

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
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
	required, err := artifactgo.HashRequiredMods(bundle, artifact.Identity(), declared)
	if err != nil {
		return verificationReport(artifact, false, "M12_GO_DECLARED_SUM_MISMATCH")
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	started := time.Now()
	ops := &databaseOps{ctx: lookupCtx, http: v.http, key: v.key}
	client := sumdb.NewClient(ops)
	name, version := artifact.Identity().Name(), artifact.Identity().Version()
	zipLines, err := client.Lookup(name, version)
	if err != nil {
		return domain.VerificationReport{}, ops.lookupError("archive", started)
	}
	modLines, err := client.Lookup(name, version+"/go.mod")
	if err != nil {
		return domain.VerificationReport{}, ops.lookupError("control", started)
	}
	if !exactLine(zipLines, name+" "+version+" "+archiveSum) || !exactLine(modLines, name+" "+version+"/go.mod "+modSum) {
		return verificationReport(artifact, false, "M12_GO_SUMDB_MISMATCH")
	}
	for _, ref := range required {
		lines, err := client.Lookup(name, ref.Version+"/go.mod")
		if err != nil {
			return domain.VerificationReport{}, ops.lookupError("required-control", started)
		}
		if !exactLine(lines, name+" "+ref.Version+"/go.mod "+ref.Sum) {
			return verificationReport(artifact, false, "M12_GO_SUMDB_MISMATCH")
		}
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
	// The upstream client formats errors with %v, losing their types. Keep only
	// bounded local diagnostics, never the upstream error text or request path.
	lastRemoteFailure string
	lastRemoteStage   string
	lastHTTPStatus    int
	securityFailure   bool
}

func (o *databaseOps) ReadRemote(relative string) ([]byte, error) {
	if len(relative) > 2048 || (!strings.HasPrefix(relative, "/lookup/") && !strings.HasPrefix(relative, "/tile/")) || relative != path.Clean(relative) || strings.ContainsAny(relative, "?#\\\r\n") {
		return nil, o.remoteFailure("path", "invalid_path", 0)
	}
	stage := "lookup"
	if strings.HasPrefix(relative, "/tile/") {
		stage = "tile"
	}
	o.mu.Lock()
	o.requests++
	allowed := o.requests <= 64
	o.mu.Unlock()
	if !allowed {
		return nil, o.remoteFailure(stage+"-request", "request_bound", 0)
	}
	req, err := http.NewRequestWithContext(o.ctx, http.MethodGet, "https://sum.golang.org"+relative, nil)
	if err != nil {
		return nil, o.remoteFailure(stage+"-request", "invalid_request", 0)
	}
	response, err := o.http.Do(req)
	if err != nil {
		return nil, o.remoteFailure(stage+"-request", remoteErrorCategory(err), 0)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, o.remoteFailure(stage+"-headers", "http_status", response.StatusCode)
	}
	if response.ContentLength > 1<<20 {
		return nil, o.remoteFailure(stage+"-headers", "response_bound", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		return nil, o.remoteFailure(stage+"-body", remoteErrorCategory(err), response.StatusCode)
	}
	if len(body) == 0 || len(body) > 1<<20 {
		return nil, o.remoteFailure(stage+"-body", "response_bound", response.StatusCode)
	}
	o.mu.Lock()
	o.bytes += int64(len(body))
	allowed = o.bytes <= 4<<20
	o.mu.Unlock()
	if !allowed {
		return nil, o.remoteFailure(stage+"-body", "total_response_bound", response.StatusCode)
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
func (o *databaseOps) SecurityError(string) {
	o.mu.Lock()
	o.securityFailure = true
	o.mu.Unlock()
}

func remoteErrorCategory(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return "incomplete_response"
	}
	var timeout net.Error
	if errors.As(err, &timeout) && timeout.Timeout() {
		return "timeout"
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return "dns"
	}
	return "transport"
}

func (o *databaseOps) remoteFailure(stage, category string, status int) error {
	o.mu.Lock()
	o.lastRemoteStage, o.lastRemoteFailure, o.lastHTTPStatus = stage, category, status
	o.mu.Unlock()
	return errors.New("go SumDB " + category)
}

func (o *databaseOps) lookupError(stage string, started time.Time) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	cause := errors.New("record or proof rejected")
	if o.lastRemoteFailure != "" {
		cause = errors.New("remote request failed")
	}
	if o.ctx.Err() != nil {
		cause = o.ctx.Err()
	} else if o.lastRemoteFailure == "deadline" {
		cause = context.DeadlineExceeded
	} else if o.lastRemoteFailure == "cancelled" {
		cause = context.Canceled
	}
	if o.securityFailure {
		cause = errors.New("signed tree consistency rejected")
	}
	// A failed partial tile may be recovered by the upstream full-tile fallback.
	// These fields describe observations, not proof that the last remote failure
	// caused the final rejection. No peer error is wrapped or logged.
	return fmt.Errorf("go SumDB authenticated %s lookup failed: %w (last_remote_failure=%s last_remote_stage=%s last_http_status=%d requests=%d bytes=%d elapsed_ms=%d)",
		stage, cause, o.lastRemoteFailure, o.lastRemoteStage, o.lastHTTPStatus, o.requests, o.bytes, time.Since(started).Milliseconds())
}
