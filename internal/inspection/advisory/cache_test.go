package advisory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

type advisoryCacheFixture struct {
	data              map[string][]byte
	reads, writes     int
	readErr, writeErr error
	capacityMiss      bool
}

func (*advisoryCacheFixture) Policy() domain.StoragePolicy { return domain.DefaultStoragePolicy() }
func (*advisoryCacheFixture) Reserve(context.Context, ports.StorageRequest) (ports.StorageLease, error) {
	return nil, errors.New("cache fixture has no payload reservation")
}
func (s *advisoryCacheFixture) ReadAdvisory(_ context.Context, key domain.ContentDigest, _ uint64) ([]byte, bool, error) {
	s.reads++
	body, ok := s.data[key.String()]
	return bytes.Clone(body), ok, s.readErr
}
func (s *advisoryCacheFixture) WriteAdvisory(_ context.Context, key domain.ContentDigest, body []byte) (bool, error) {
	s.writes++
	if s.writeErr != nil {
		return false, s.writeErr
	}
	if s.capacityMiss {
		return false, nil
	}
	if s.data == nil {
		s.data = map[string][]byte{}
	}
	s.data[key.String()] = bytes.Clone(body)
	return true, nil
}

func TestAdvisoryCacheReusesOnlyQueryFactsAndRebindsCurrentSubject(t *testing.T) {
	store := &advisoryCacheFixture{}
	ctx := ports.WithStorage(context.Background(), store)
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return jsonResponse(`{"results":[{}]}`), nil })}
	inner := &retainedInspector{}
	inspector, err := newInspector(inner, client, calculatorBuildInfo())
	if err != nil {
		t.Fatal(err)
	}
	artifact := advisoryArtifact(t, "npm", "fixture", "1.0.0")
	first, err := inspector.Inspect(ctx, artifact)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	current, err := domain.NewAcquiredArtifact(artifact.Identity(), digest, "intake:current:test", 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := inspector.Inspect(ctx, current)
	if err != nil || calls != 1 || len(inner.calls) != 2 || store.writes != 1 || len(second.Executions()) != 2 || second.Executions()[1].Status() != domain.ExecutionCompleted {
		t.Fatal("cache bypassed fresh inspection or queried again", err, calls, store.writes)
	}
	if first.Evidence()[1].Summary() != second.Evidence()[1].Summary() {
		t.Fatal("cache hit changed the original fetch proof/time")
	}
	for _, fact := range second.Evidence() {
		if fact.Digest() != digest {
			t.Fatal("cached facts retained the earlier content subject")
		}
	}
	if _, err := inspector.Inspect(ports.WithAdvisoryRefresh(ctx, true), current); err != nil || calls != 2 || store.reads != 2 {
		t.Fatal("explicit refresh reused the cache or changed query checks", err, calls, store.reads)
	}
	updated := calculatorBuildInfo()
	updated.Deps[0].Version = "v0.6.5"
	other, err := newInspector(inner, client, updated)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Inspect(ctx, current); err != nil || calls != 3 || len(store.data) != 2 {
		t.Fatal("different calculator profile reused old results", err, calls, len(store.data))
	}
}

func TestAdvisoryCacheExpiryRefreshesButMalformedBundleNeverBecomesCleanZero(t *testing.T) {
	for _, mode := range []string{"expired", "future", "wrong-query", "wrong-profile", "missing-rows", "missing-records", "duplicate-json"} {
		t.Run(mode, func(t *testing.T) {
			store := &advisoryCacheFixture{}
			ctx := ports.WithStorage(context.Background(), store)
			calls := 0
			inspector, err := newInspector(&retainedInspector{}, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return jsonResponse(`{"results":[{}]}`), nil })}, calculatorBuildInfo())
			if err != nil {
				t.Fatal(err)
			}
			artifact := advisoryArtifact(t, "npm", "fixture", "1.0.0")
			if _, err := inspector.Inspect(ctx, artifact); err != nil {
				t.Fatal(err)
			}
			var key string
			var bundle cachedLookup
			for k, body := range store.data {
				key = k
				if err := json.Unmarshal(body, &bundle); err != nil {
					t.Fatal(err)
				}
			}
			switch mode {
			case "expired":
				bundle.Responses[0].Fetched = time.Now().Add(-25 * time.Hour).UTC().Format(time.RFC3339Nano)
			case "future":
				bundle.Responses[0].Fetched = time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
			case "wrong-query":
				bundle.Queries[0].Version = "2.0.0"
			case "wrong-profile":
				bundle.Profile = strings.Repeat("f", 64)
			case "missing-rows":
				bundle.Rows = nil
			case "missing-records":
				bundle.Records = nil
			}
			body, err := json.Marshal(bundle)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "duplicate-json" {
				body = bytes.Replace(body, []byte(`"schema":`), []byte(`"SCHEMA":"ambiguous","schema":`), 1)
			}
			store.data[key] = body
			report, err := inspector.Inspect(ctx, artifact)
			if mode == "expired" {
				if err != nil || calls != 2 || report.Executions()[1].Status() != domain.ExecutionCompleted {
					t.Fatal("expired cache was reused or no fresh query completed", err, calls)
				}
				return
			}
			if !errors.Is(err, errCachedResponse) || calls != 1 || report.Executions()[1].Status() != domain.ExecutionIncomplete || len(report.Findings()) != 0 {
				t.Fatal("malformed cache was silently refreshed or treated as zero", err, calls)
			}
			if limitation, _ := report.Executions()[1].LimitationCode(); limitation != "M13_ADVISORY_CACHE_INCOMPLETE" {
				t.Fatal("cache failure was attributed to the provider", limitation)
			}
		})
	}
}

func TestAdvisoryCacheKeepsOneRecordBodyAcrossQueryVersions(t *testing.T) {
	store := &advisoryCacheFixture{}
	ctx := ports.WithStorage(context.Background(), store)
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Method == http.MethodGet {
			return jsonResponse(recordFixture), nil
		}
		return jsonResponse(`{"results":[{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}]},{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}]}]}`), nil
	})}
	inspector, err := newInspector(&retainedInspector{}, client, calculatorBuildInfo())
	if err != nil {
		t.Fatal(err)
	}
	queries := []osvPackage{{"npm", "fixture", "1.0.0"}, {"npm", "fixture", "1.1.0"}}
	for n := 0; n < 2; n++ {
		records, facts, err, cacheErr := inspector.lookup(ctx, queries)
		if err != nil || cacheErr != nil || len(records) != 2 || len(records[0]) != 1 || len(records[1]) != 1 || len(facts) != 2 {
			t.Fatal("record bindings lost", err, cacheErr)
		}
	}
	if calls != 2 {
		t.Fatal("cache repeated a record/query read", calls)
	}
	for _, body := range store.data {
		var bundle cachedLookup
		if err := json.Unmarshal(body, &bundle); err != nil || len(bundle.Records) != 1 || len(bundle.Rows) != 2 {
			t.Fatal("cache expanded duplicate record bodies", err)
		}
	}
}

func TestAdvisoryCachePreservesQueryAndStorageFailureAxes(t *testing.T) {
	for _, mode := range []string{"query-failed", "write-failed", "capacity"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("controlled first cause")
			store := &advisoryCacheFixture{}
			if mode == "write-failed" {
				store.writeErr = cause
			}
			store.capacityMiss = mode == "capacity"
			inspector, err := newInspector(&retainedInspector{}, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				if mode == "query-failed" {
					return nil, cause
				}
				return jsonResponse(`{"results":[{}]}`), nil
			})}, calculatorBuildInfo())
			if err != nil {
				t.Fatal(err)
			}
			report, err := inspector.Inspect(ports.WithStorage(context.Background(), store), advisoryArtifact(t, "npm", "fixture", "1.0.0"))
			if mode == "capacity" {
				if err != nil || report.Executions()[1].Status() != domain.ExecutionCompleted {
					t.Fatal("optional cache capacity changed a completed query", err)
				}
				return
			}
			if !errors.Is(err, cause) {
				t.Fatal("primary query/storage failure lost", err)
			}
			if mode == "query-failed" {
				if store.writes != 0 || report.Executions()[1].Status() != domain.ExecutionIncomplete {
					t.Fatal("failed query entered the cache")
				}
			} else if report.Executions()[1].Status() != domain.ExecutionCompleted || len(report.Evidence()) != 2 {
				t.Fatal("persistence error erased completed query facts")
			}
		})
	}
}

func TestOfficialLocalProfileRetainsUpstreamReviewAcrossCacheAndRefreshFailure(t *testing.T) {
	store := &advisoryCacheFixture{}
	ctx := ports.WithStorage(context.Background(), store)
	calls := 0
	cause := errors.New("upstream candidate unavailable")
	fail := false
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Method == "POST" {
			var body struct {
				Queries []batchQuery `json:"queries"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body.Queries) != 2 || body.Queries[0].Version != "2.6.0+cpu" || body.Queries[1].Version != "2.6.0" {
				t.Fatal("missing exact/upstream binding", body.Queries)
			}
			if fail {
				return nil, cause
			}
			return jsonResponse(`{"results":[{},{"vulns":[{"id":"GHSA-aaaa-bbbb-cccc","modified":"2026-01-01T00:00:00Z"}]}]}`), nil
		}
		return jsonResponse(`{"schema_version":"1.7.3","id":"GHSA-aaaa-bbbb-cccc","modified":"2026-01-01T00:00:00Z","affected":[{"package":{"ecosystem":"PyPI","name":"torch"},"ranges":[{"type":"ECOSYSTEM","events":[{"introduced":"0"},{"last_affected":"2.6.0"}]}]}],"severity":[{"type":"CVSS_V3","score":"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"}]}`), nil
	})}
	inner := &retainedInspector{}
	inspector, err := newInspector(inner, client, calculatorBuildInfo())
	if err != nil {
		t.Fatal(err)
	}
	artifact := advisoryArtifact(t, "pytorch-cpu", "torch", "2.6.0+cpu")
	for n := 0; n < 2; n++ {
		report, err := inspector.Inspect(ctx, artifact)
		if err != nil || len(report.Findings()) != 1 || report.Findings()[0].Code() != "M13_ADVISORY_REVIEW" || calls != 2 || len(inner.calls) != n+1 {
			t.Fatal("upstream boundary lost or original inspection reused", report.Findings(), calls, inner.calls, err)
		}
		seen := false
		for _, fact := range report.Evidence() {
			if fact.Identity() != artifact.Identity() {
				t.Fatal("evidence identity projected")
			}
			if fact.Kind() == "advisory-applicability" && strings.Contains(fact.Summary(), "2.6.0+cpu") {
				seen = true
			}
		}
		if !seen {
			t.Fatal("exact/upstream applicability omitted")
		}
	}
	fail = true
	report, err := inspector.Inspect(ports.WithAdvisoryRefresh(ctx, true), artifact)
	if !errors.Is(err, cause) || len(report.Executions()) != 2 || report.Executions()[1].Status() != domain.ExecutionIncomplete || len(inner.calls) != 3 || calls != 3 {
		t.Fatal("failed companion refresh reused old clean/approval", report, err, calls)
	}
}
