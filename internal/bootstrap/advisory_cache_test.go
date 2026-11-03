package bootstrap

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/inspection/advisory"
	"github.com/rahoney/heliopause/internal/storage"
)

type advisoryCacheOriginal struct{ calls int }

func (p *advisoryCacheOriginal) ExpectedChecks(context.Context, domain.AcquiredArtifact, domain.CheckKind) ([]domain.CheckRequirement, error) {
	id, _ := domain.NewCheckID("cache-original")
	profile, _ := domain.CheckProfileDigest("cache-original-v1", "test-only original inspection")
	check, err := domain.NewCheckRequirement(id, domain.CheckInspection, true, profile)
	return []domain.CheckRequirement{check}, err
}
func (p *advisoryCacheOriginal) Inspect(_ context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	p.calls++
	id, _ := domain.NewCheckID("cache-original")
	profile, _ := domain.CheckProfileDigest("cache-original-v1", "test-only original inspection")
	check, err := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	eid, _ := domain.NewEvidenceID("cache-original-result")
	fact, err := domain.NewEvidence(eid, id, artifact.Identity(), artifact.Digest(), "test-original", "synthetic original; no Artifact executed", profile)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	return domain.NewInspectionReport(check, nil, []domain.Evidence{fact})
}

type advisoryCacheHTTP func(*http.Request) (*http.Response, error)

func (f advisoryCacheHTTP) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Actual common storage and public Inspector construction, with controlled HTTP
// and an explicit synthetic original inspector. This is not native qualification.
func TestAdvisoryCacheCompositionPersistsFactsAcrossStorageReopen(t *testing.T) {
	spec := domain.DefaultStoragePolicy().Spec()
	spec.TotalBytes = 16 << 20
	spec.CacheBytes = 4 << 20
	spec.EvidenceBytes = 4 << 20
	spec.MinimumFreeBytes = 1
	policy, err := domain.NewStoragePolicy(spec)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "haa")
	backend, err := storage.NewLocal(context.Background(), root, policy)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := domain.NewOperationID()
	if err != nil {
		t.Fatal(err)
	}
	ctx := ports.WithStorage(ports.WithStorageOperation(context.Background(), operation), backend)
	original := &advisoryCacheOriginal{}
	requests := 0
	client := &http.Client{Transport: advisoryCacheHTTP(func(request *http.Request) (*http.Response, error) {
		requests++
		if request.URL.String() != "https://api.osv.dev/v1/querybatch" || request.Method != http.MethodPost {
			t.Fatal("composition changed the typed request")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"results":[{}]}`))}, nil
	})}
	inspector, err := advisory.NewInspector(original, client)
	if err != nil {
		t.Fatal("actual linked calculator identity unavailable", err)
	}
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "fixture", "1.0.0", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:composition:tarball", 1)
	if err != nil {
		t.Fatal(err)
	}
	first, err := inspector.Inspect(ctx, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.Close(); err != nil {
		t.Fatal(err)
	}
	backend, err = storage.NewLocal(context.Background(), root, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	operation, err = domain.NewOperationID()
	if err != nil {
		t.Fatal(err)
	}
	ctx = ports.WithStorage(ports.WithStorageOperation(context.Background(), operation), backend)
	digest, _ = domain.NewSHA256Digest(strings.Repeat("b", 64))
	artifact, err = domain.NewAcquiredArtifact(identity, digest, "intake:current:tarball", 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := inspector.Inspect(ctx, artifact)
	if err != nil || requests != 1 || original.calls != 2 || len(second.Executions()) != 2 || second.Executions()[1].Status() != domain.ExecutionCompleted {
		t.Fatal("storage/frontend cache composition lost a complete result", err, requests, original.calls)
	}
	if first.Evidence()[1].Summary() != second.Evidence()[1].Summary() {
		t.Fatal("reopened cache replaced the original authenticated fetch facts")
	}
	for _, fact := range second.Evidence() {
		if fact.Digest() != digest {
			t.Fatal("old subject approval escaped through response caching")
		}
	}
	if _, err := inspector.Inspect(ports.WithAdvisoryRefresh(ctx, true), artifact); err != nil || requests != 2 || original.calls != 3 {
		t.Fatal("force refresh skipped the actual source query", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "advisory-cache"))
	if err != nil || len(entries) != 1 {
		t.Fatal("refresh accumulated duplicate response copies", err, len(entries))
	}
}

func TestEmbeddedAdvisoryNoticesDoNotRequireRuntimeOrExternalNoticeFile(t *testing.T) {
	var out bytes.Buffer
	if err := Run(context.Background(), []string{"licenses", "--output", "human", "--lang", "en"}, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	if out.String() != advisory.ThirdPartyNotices || !strings.Contains(out.String(), "Copyright (c) 2022 Lucas TESSON") || !strings.Contains(out.String(), "Permission is hereby granted") {
		t.Fatal("compiled dependency copyright/permission text was not supplied")
	}
}
