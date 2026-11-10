package advisory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestBatchLookupDeduplicatesAndFollowsOnlyPendingPackages(t *testing.T) {
	first := osvPackage{"npm", "first", "1.0.0"}
	second := osvPackage{"PyPI", "second", "2.0.0"}
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.String() != osvEndpoint || r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Fatal("query escaped its fixed advisory operation")
		}
		var request struct {
			Queries []batchQuery `json:"queries"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if calls == 1 {
			if len(request.Queries) != 2 || request.Queries[0].Package.Name != first.Name || request.Queries[0].Version != first.Version || request.Queries[1].Package.Name != second.Name || request.Queries[0].PageToken != "" {
				t.Fatalf("wrong first batch: %#v", request)
			}
			return jsonResponse(`{"results":[{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}],"next_page_token":"next-A"},{}]}`), nil
		}
		if calls != 2 || len(request.Queries) != 1 || request.Queries[0].Package.Name != first.Name || request.Queries[0].PageToken != "next-A" {
			t.Fatalf("completed or wrong package queried: %#v", request)
		}
		return jsonResponse(`{"results":[{"vulns":[{"id":"OSV-2","modified":"2026-10-21T00:00:00Z"}]}]}`), nil
	})}
	got, facts, err := queryBatch(context.Background(), client, []osvPackage{first, second, first})
	if err != nil || calls != 2 || len(got) != 3 || len(got[0]) != 2 || len(got[1]) != 0 || len(got[2]) != 2 || len(facts) != 2 {
		t.Fatalf("lost exact batch coverage: %#v, %#v, %v", got, facts, err)
	}
	for _, fact := range facts {
		if len(fact.requestSHA256) != 64 || len(fact.responseSHA256) != 64 || fact.fetchedAt.IsZero() {
			t.Fatal("missing response binding")
		}
	}
	got[0][0].id = "changed"
	if got[2][0].id != "OSV-1" {
		t.Fatal("duplicate nodes share mutable result storage")
	}
}

func TestBatchLookupRejectsPartialPaginationAndChangedRecords(t *testing.T) {
	for _, failure := range []string{"cycle", "duplicate-id", "missing-page", "changed-record"} {
		t.Run(failure, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return jsonResponse(`{"results":[{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}],"next_page_token":"next"}]}`), nil
				}
				switch failure {
				case "cycle":
					return jsonResponse(`{"results":[{"next_page_token":"next"}]}`), nil
				case "duplicate-id":
					return jsonResponse(`{"results":[{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}]}]}`), nil
				case "changed-record":
					return jsonResponse(`{"results":[{"vulns":[{"id":"OSV-1","modified":"2026-10-21T00:00:00Z"}]}]}`), nil
				default:
					return jsonResponse(`{"results":[]}`), nil
				}
			})}
			got, facts, err := queryBatch(context.Background(), client, []osvPackage{{"npm", "first", "1.0.0"}})
			if err == nil || got != nil || calls != 2 || len(facts) != 2 {
				t.Fatalf("partial lookup accepted: %#v %#v %v", got, facts, err)
			}
		})
	}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(`{"results":[{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}]},{"vulns":[{"id":"OSV-1","modified":"2026-10-21T00:00:00Z"}]}]}`), nil
	})}
	if got, _, err := queryBatch(context.Background(), client, []osvPackage{{"npm", "first", "1.0.0"}, {"npm", "second", "1.0.0"}}); err == nil || got != nil {
		t.Fatal("one lookup accepted conflicting modified timestamps")
	}
}

func TestBatchLookupStopsAtFiniteRequestBudget(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return jsonResponse(fmt.Sprintf(`{"results":[{"next_page_token":"page-%d"}]}`, calls)), nil
	})}
	got, facts, err := queryBatch(context.Background(), client, []osvPackage{{"npm", "first", "1.0.0"}})
	if err == nil || got != nil || calls != maxLookupRequests || len(facts) != maxLookupRequests {
		t.Fatalf("unbounded pages: %d, %v", calls, err)
	}
}

func TestBatchLookupSplitsClientBatchBoundWithoutLosingNodes(t *testing.T) {
	packages := make([]osvPackage, 1001)
	for i := range packages {
		packages[i] = osvPackage{"Go", fmt.Sprintf("example.invalid/module%d", i), "v1.0.0"}
	}
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var request struct {
			Queries []batchQuery `json:"queries"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Fatal("invalid query")
		}
		want := 1000
		if calls == 2 {
			want = 1
		}
		if calls > 2 || len(request.Queries) != want {
			t.Fatal("client batch bound violated")
		}
		return jsonResponse(`{"results":[` + strings.TrimSuffix(strings.Repeat("{},", want), ",") + `]}`), nil
	})}
	got, facts, err := queryBatch(context.Background(), client, packages)
	if err != nil || len(got) != 1001 || calls != 2 || len(facts) != 2 {
		t.Fatalf("chunked coverage failed: %d, %v", len(got), err)
	}
}

func TestBatchLookupRejectsInvalidInputsBeforeNetwork(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return jsonResponse(`{"results":[{}]}`), nil
	})}
	for _, packages := range [][]osvPackage{
		nil, {}, {{"npm", "", "1.0.0"}}, {{"npm", "first", ""}},
		{{"unknown", "first", "1.0.0"}}, {{"github-release", "first", "1.0.0"}},
		{{"npm", "first\nsecond", "1.0.0"}}, {{"npm", "first", "1.0.0\x1b"}},
		{{"npm", "first", strings.Repeat("1", 257)}}, make([]osvPackage, maxLookupPackages+1),
	} {
		if got, _, err := queryBatch(context.Background(), client, packages); err == nil || got != nil {
			t.Fatal("invalid advisory input reached lookup")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, _, err := queryBatch(ctx, client, []osvPackage{{"npm", "first", "1.0.0"}}); !errors.Is(err, context.Canceled) || got != nil {
		t.Fatal("cancelled lookup accepted")
	}
	if calls != 0 {
		t.Fatal("invalid input reached network")
	}
}

func TestBatchLookupChargesWholeResponseByteBudget(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		body := fmt.Sprintf(`{"results":[{"next_page_token":"page-%d"}]}`, calls)
		return jsonResponse(body + strings.Repeat(" ", maxResponseBytes-len(body))), nil
	})}
	got, facts, err := queryBatch(context.Background(), client, []osvPackage{{"npm", "first", "1.0.0"}})
	if err == nil || got != nil || calls != maxLookupBytes/maxResponseBytes || len(facts) != calls {
		t.Fatalf("whole byte budget lost: calls=%d, err=%v", calls, err)
	}
}

type failedRead struct{ cause error }

func (r failedRead) Read([]byte) (int, error) { return 0, r.cause }

func TestBatchLookupPreservesIncompleteResponseReadError(t *testing.T) {
	cause := errors.New("partial network response")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		response := jsonResponse("")
		response.Body = io.NopCloser(io.MultiReader(strings.NewReader(`{"results":[`), failedRead{cause}))
		return response, nil
	})}
	if got, _, err := queryBatch(context.Background(), client, []osvPackage{{"npm", "first", "1.0.0"}}); !errors.Is(err, cause) || got != nil {
		t.Fatal("partial response became success or lost primary error")
	}
}

func TestBatchLookupRejectsHTTPFailuresAndCancellation(t *testing.T) {
	for _, failure := range []string{"redirect", "status", "media", "encoding", "oversize", "cancel", "transport"} {
		t.Run(failure, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			original := errors.New("original transport failure")
			client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { t.Fatal("injected redirect policy used"); return nil }, Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				response := jsonResponse(`{"results":[{}]}`)
				switch failure {
				case "redirect":
					response.StatusCode = 302
					response.Header.Set("Location", "https://other.invalid/")
				case "status":
					response.StatusCode = 503
				case "media":
					response.Header.Set("Content-Type", "text/html")
				case "encoding":
					response.Header.Set("Content-Encoding", "gzip")
				case "oversize":
					response.Body = io.NopCloser(strings.NewReader(strings.Repeat(" ", maxResponseBytes+1)))
				case "cancel":
					cancel()
				case "transport":
					return nil, original
				}
				return response, nil
			})}
			got, _, err := queryBatch(ctx, client, []osvPackage{{"npm", "first", "1.0.0"}})
			if err == nil || got != nil || calls != 1 {
				t.Fatalf("failed query accepted/retried: %#v %v calls=%d", got, err, calls)
			}
			if failure == "transport" && !errors.Is(err, original) {
				t.Fatal("primary transport error lost")
			}
			if failure == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("primary cancellation lost")
			}
		})
	}
}

func TestLocalBuildQueriesKeepExactIdentityAndUpstreamBoundaryCandidates(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var request struct {
			Queries []batchQuery `json:"queries"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Queries) != 3 || request.Queries[0].Version != "2.6.0+cpu" || request.Queries[1].Version != "2.6.0" || request.Queries[2].Version != "2.6.0+cu126" {
			t.Fatalf("missing exact or upstream candidate: %+v", request.Queries)
		}
		return jsonResponse(`{"results":[{"vulns":[{"id":"GHSA-aaaa-bbbb-cccc","modified":"2026-01-01T00:00:00Z"}]},{"vulns":[{"id":"GHSA-aaaa-bbbb-cccc","modified":"2026-01-01T00:00:00Z"},{"id":"GHSA-dddd-eeee-ffff","modified":"2026-01-01T00:00:00Z"}]},{}]}`), nil
	})}
	queries := []osvPackage{{"PyPI", "torch", "2.6.0+cpu"}, {"PyPI", "torch", "2.6.0+cu126"}, {"PyPI", "torch", "2.6.0"}}
	rows, facts, err := queryBatch(context.Background(), client, queries)
	if err != nil || calls != 1 || len(facts) != 1 || len(rows) != 3 {
		t.Fatal(rows, facts, err)
	}
	for _, row := range rows {
		if len(row) != 2 || row[0].id != "GHSA-aaaa-bbbb-cccc" || row[1].id != "GHSA-dddd-eeee-ffff" {
			t.Fatalf("lost or duplicated upstream evidence: %+v", row)
		}
	}
	if queries[0].Version != "2.6.0+cpu" {
		t.Fatal("artifact version projected onto public release")
	}
}

func TestLocalBuildCompanionQueriesRetainOriginalKeyBudget(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("must not request") })}
	packages := make([]osvPackage, maxLookupPackages)
	for n := range packages {
		packages[n] = osvPackage{"PyPI", fmt.Sprintf("fixture-%d", n), "1.0+cpu"}
	}
	if _, _, err := queryBatch(context.Background(), client, packages); err == nil || calls != 0 {
		t.Fatal("companion query expanded original key budget", calls, err)
	}
}
