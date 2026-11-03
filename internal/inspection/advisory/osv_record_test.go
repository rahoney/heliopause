package advisory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"
)

const recordFixture = `{"schema_version":"1.9.2","id":"OSV-1","modified":"2026-10-20T00:00:00Z","aliases":["CVE-2026-1000"],"affected":[{"package":{"ecosystem":"npm","name":"fixture"},"ranges":[{"type":"SEMVER","events":[{"introduced":"0"},{"fixed":"1.2.3"}]}]}],"database_specific":{"severity":"HIGH"},"details":"Untrusted text; never an execution input."}`

func recordReference(t *testing.T) advisoryReference {
	t.Helper()
	stamp, err := time.Parse(time.RFC3339, "2026-10-20T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	return advisoryReference{"OSV-1", stamp}
}

func TestRecordKeepsIdentityWithdrawalAndRangeFacts(t *testing.T) {
	record, err := decodeRecord([]byte(recordFixture), recordReference(t))
	if err != nil || record.ID != "OSV-1" || record.Withdrawn != nil || len(record.Affected) != 1 || record.Affected[0].Ranges[0].Events[1].Fixed != "1.2.3" || record.Database.Severity != "HIGH" || len(record.Aliases) != 1 {
		t.Fatalf("lost normalized record: %#v %v", record, err)
	}
	withdrawn := `{"id":"OSV-1","modified":"2026-10-20T00:00:00Z","withdrawn":"2026-10-19T00:00:00Z"}`
	record, err = decodeRecord([]byte(withdrawn), recordReference(t))
	if err != nil || record.Withdrawn == nil || len(record.Affected) != 0 {
		t.Fatal("withdrawn record without affected entries was lost", err)
	}
	if _, err := decodeRecord([]byte(strings.Replace(recordFixture, `"schema_version":"1.9.2",`, "", 1)), recordReference(t)); err != nil {
		t.Fatal("default schema 1.0.0 rejected", err)
	}
}

func TestRecordRejectsWrongSubjectOrMalformedFacts(t *testing.T) {
	for name, body := range map[string]string{
		"wrong id":                          strings.Replace(recordFixture, `"OSV-1"`, `"OSV-2"`, 1),
		"wrong modified":                    strings.Replace(recordFixture, `2026-10-20`, `2026-10-21`, 1),
		"null modified":                     strings.Replace(recordFixture, `"2026-10-20T00:00:00Z"`, `null`, 1),
		"new major":                         strings.Replace(recordFixture, `1.9.2`, `2.0.0`, 1),
		"noncanonical schema field":         strings.Replace(recordFixture, `schema_version`, `SCHEMA_VERSION`, 1),
		"empty schema":                      strings.Replace(recordFixture, `"1.9.2"`, `""`, 1),
		"null schema":                       strings.Replace(recordFixture, `"1.9.2"`, `null`, 1),
		"duplicate id":                      strings.Replace(recordFixture, `"id":"OSV-1"`, `"id":"OSV-1","ID":"OSV-2"`, 1),
		"unicode folded severity duplicate": strings.Replace(recordFixture, `"severity":"HIGH"`, `"severity":"HIGH","ſeverity":"LOW"`, 1),
		"duplicate alias":                   strings.Replace(recordFixture, `"CVE-2026-1000"`, `"CVE-2026-1000","CVE-2026-1000"`, 1),
		"alias path":                        strings.Replace(recordFixture, `CVE-2026-1000`, `../private`, 1),
		"no active affected":                `{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}`,
		"ambiguous event":                   strings.Replace(recordFixture, `"introduced":"0"`, `"introduced":"0","fixed":"1.0.0"`, 1),
		"empty event":                       strings.Replace(recordFixture, `{"introduced":"0"}`, `{}`, 1),
		"empty ranges type":                 strings.Replace(recordFixture, `"SEMVER"`, `""`, 1),
		"git without repo":                  strings.Replace(recordFixture, `"SEMVER"`, `"GIT"`, 1),
		"oversized version":                 strings.Replace(recordFixture, `"1.2.3"`, `"`+strings.Repeat("1", 257)+`"`, 1),
		"truncated":                         recordFixture[:len(recordFixture)-1],
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeRecord([]byte(body), recordReference(t)); err == nil {
				t.Fatal("malformed/wrong-subject record accepted")
			}
		})
	}
}

func TestRecordGETUsesFixedOriginAndSharedCredentialRedirectBoundary(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	origin, _ := url.Parse("https://api.osv.dev/")
	jar.SetCookies(origin, []*http.Cookie{{Name: "private", Value: "must-not-send"}})
	calls := 0
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { t.Fatal("injected redirect policy used"); return nil }, Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Method != "GET" || request.URL.String() != "https://api.osv.dev/v1/vulns/OSV-1" || request.Header.Get("Cookie") != "" || request.Header.Get("Authorization") != "" || request.Header.Get("Accept") != "application/json" {
			t.Fatal("record read escaped its typed operation")
		}
		if calls == 1 {
			return jsonResponse(recordFixture), nil
		}
		response := jsonResponse("")
		response.StatusCode = 302
		response.Header.Set("Location", "https://other.invalid/")
		return response, nil
	})}
	record, fact, err := getRecord(context.Background(), client, recordReference(t), maxResponseBytes)
	if err != nil || record.ID != "OSV-1" || len(fact.responseSHA256) != 64 || len(fact.requestSHA256) != 64 || fact.fetchedAt.IsZero() {
		t.Fatal("record read missing binding", err)
	}
	if _, _, err := getRecord(context.Background(), client, recordReference(t), maxResponseBytes); err == nil || calls != 2 {
		t.Fatal("record redirect followed")
	}
	for _, id := range []string{".", "..", "../private", "OSV-1?redirect=other", "OSV-\n"} {
		expected := recordReference(t)
		expected.id = id
		if _, _, err := getRecord(context.Background(), client, expected, maxResponseBytes); err == nil {
			t.Fatal("invalid record ID reached network")
		}
	}
	if calls != 2 || len(client.Jar.Cookies(origin)) != 1 {
		t.Fatal("invalid ID was fetched or original cookie jar was mutated")
	}
}

func TestLookupSharesDetailedRecordAcrossExactQueries(t *testing.T) {
	posts, gets := 0, 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method == http.MethodPost {
			posts++
			return jsonResponse(`{"results":[{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}]},{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}]}]}`), nil
		}
		gets++
		return jsonResponse(recordFixture), nil
	})}
	packages := []osvPackage{{"npm", "fixture", "1.0.0"}, {"npm", "fixture", "1.1.0"}, {"npm", "fixture", "1.0.0"}}
	got, facts, err := lookupAdvisories(context.Background(), client, packages)
	if err != nil || posts != 1 || gets != 1 || len(got) != 3 || len(facts) != 2 {
		t.Fatalf("record deduplication or coverage lost: %#v %v", got, err)
	}
	for _, record := range got {
		if len(record) != 1 || record[0].ID != "OSV-1" {
			t.Fatal("wrong query record mapping")
		}
	}
	for _, fact := range facts {
		if fact.responseBytes <= 0 {
			t.Fatal("response omitted from shared byte ledger")
		}
	}
}

func TestLookupRejectsDetailChangeAfterBatch(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method == http.MethodPost {
			return jsonResponse(`{"results":[{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}]}]}`), nil
		}
		return jsonResponse(strings.Replace(recordFixture, "2026-10-20", "2026-10-21", 1)), nil
	})}
	got, facts, err := lookupAdvisories(context.Background(), client, []osvPackage{{"npm", "fixture", "1.0.0"}})
	if err == nil || got != nil || len(facts) != 2 {
		t.Fatal("changed record became a complete lookup or lost failed response binding")
	}
}

func TestLookupRecordReadsShareBatchRequestBudget(t *testing.T) {
	for _, pages := range []int{maxLookupRequests - 1, maxLookupRequests} {
		t.Run(fmt.Sprint(pages), func(t *testing.T) {
			posts, gets := 0, 0
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.Method == http.MethodGet {
					gets++
					return jsonResponse(recordFixture), nil
				}
				posts++
				if posts == pages {
					return jsonResponse(`{"results":[{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}]}]}`), nil
				}
				return jsonResponse(fmt.Sprintf(`{"results":[{"next_page_token":"page-%d"}]}`, posts)), nil
			})}
			got, facts, err := lookupAdvisories(context.Background(), client, []osvPackage{{"npm", "fixture", "1.0.0"}})
			if pages == maxLookupRequests {
				if err == nil || got != nil || gets != 0 {
					t.Fatal("detail lookup received a new request budget")
				}
			} else if err != nil || len(got) != 1 || gets != 1 {
				t.Fatal("last permitted record read lost", err)
			}
			if len(facts) != maxLookupRequests || posts+gets != maxLookupRequests {
				t.Fatal("shared request accounting drifted")
			}
		})
	}
}

func TestLookupRecordReadsShareWholeByteBudget(t *testing.T) {
	gets := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body string
		if request.Method == http.MethodPost {
			refs := make([]map[string]string, 8)
			for i := range refs {
				refs[i] = map[string]string{"id": fmt.Sprintf("OSV-%d", i+1), "modified": "2026-10-20T00:00:00Z"}
			}
			encoded, _ := json.Marshal(map[string]any{"results": []any{map[string]any{"vulns": refs}}})
			body = string(encoded)
		} else {
			gets++
			id := strings.TrimPrefix(request.URL.Path, "/v1/vulns/")
			body = strings.Replace(recordFixture, `"OSV-1"`, `"`+id+`"`, 1)
		}
		return jsonResponse(body + strings.Repeat(" ", maxResponseBytes-len(body))), nil
	})}
	got, facts, err := lookupAdvisories(context.Background(), client, []osvPackage{{"npm", "fixture", "1.0.0"}})
	if err == nil || got != nil || gets != 7 || len(facts) != 8 {
		t.Fatalf("record bytes escaped shared budget: gets=%d facts=%d err=%v", gets, len(facts), err)
	}
}

func TestBatchLookupBoundsDuplicateNodeResultExpansion(t *testing.T) {
	packages := make([]osvPackage, maxLookupPackages)
	for i := range packages {
		packages[i] = osvPackage{"npm", "fixture", "1.0.0"}
	}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(`{"results":[{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"},{"id":"OSV-2","modified":"2026-10-20T00:00:00Z"},{"id":"OSV-3","modified":"2026-10-20T00:00:00Z"}]}]}`), nil
	})}
	if got, facts, err := queryBatch(context.Background(), client, packages); err == nil || got != nil || len(facts) != 1 {
		t.Fatal("deduplicated lookup amplified unbounded per-node result copies")
	}
}
