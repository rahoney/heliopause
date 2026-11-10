package advisory

import (
	"strings"
	"testing"
)

func TestBatchResponseRetainsMembershipAndPages(t *testing.T) {
	body := []byte(`{"results":[{}, {"vulns":[]}, {"vulns":[{"id":"GHSA-1234-abcd-5678","modified":"2026-10-20T20:34:23.349055Z"}],"next_page_token":"opaque+next/=token"}]}`)
	got, err := decodeBatch(body, 3)
	if err != nil || len(got) != 3 || len(got[0].vulnerabilities) != 0 || len(got[1].vulnerabilities) != 0 || len(got[2].vulnerabilities) != 1 || got[2].vulnerabilities[0].id != "GHSA-1234-abcd-5678" || got[2].vulnerabilities[0].modified.IsZero() || got[2].nextPage != "opaque+next/=token" {
		t.Fatalf("exact ordered result/page lost: %#v, %v", got, err)
	}
}

func TestBatchResponseRejectsIncompleteOrAmbiguousCoverage(t *testing.T) {
	for name, body := range map[string]string{
		"missing results":    `{}`,
		"null results":       `{"results":null}`,
		"empty results":      `{"results":[]}`,
		"extra result":       `{"results":[{},{}]}`,
		"null row":           `{"results":[null]}`,
		"null vulns":         `{"results":[{"vulns":null}]}`,
		"wrong vulns type":   `{"results":[{"vulns":{}}]}`,
		"wrong case":         `{"Results":[{}]}`,
		"duplicate root":     `{"results":[{}],"results":[{}]}`,
		"casefold duplicate": `{"results":[{}],"Results":[{}]}`,
		"escaped duplicate":  `{"results":[{}],"\u0072esults":[{}]}`,
		"duplicate nested":   `{"results":[{"vulns":[],"VULNS":[]}]}`,
		"unknown row field":  `{"results":[{"ignored":true}]}`,
		"empty token":        `{"results":[{"next_page_token":""}]}`,
		"null token":         `{"results":[{"next_page_token":null}]}`,
		"token control":      `{"results":[{"next_page_token":"next\npage"}]}`,
		"missing modified":   `{"results":[{"vulns":[{"id":"OSV-1"}]}]}`,
		"invalid modified":   `{"results":[{"vulns":[{"id":"OSV-1","modified":"yesterday"}]}]}`,
		"zero modified":      `{"results":[{"vulns":[{"id":"OSV-1","modified":"0001-01-01T00:00:00Z"}]}]}`,
		"path injection":     `{"results":[{"vulns":[{"id":"../private","modified":"2026-10-20T00:00:00Z"}]}]}`,
		"id control":         `{"results":[{"vulns":[{"id":"OSV-\u001b","modified":"2026-10-20T00:00:00Z"}]}]}`,
		"duplicate id":       `{"results":[{"vulns":[{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"},{"id":"OSV-1","modified":"2026-10-20T00:00:00Z"}]}]}`,
		"trailing value":     `{"results":[{}]} {}`,
		"trailing garbage":   `{"results":[{}]}broken`,
		"truncated":          `{"results":[{`,
		"invalid utf8":       "{\"results\":[{}],\"\xff\":0}",
		"too deep":           `{"results":[{"x":` + strings.Repeat("[", 33) + "0" + strings.Repeat("]", 33) + `}]}`,
		"too large":          `{"results":[{}]}` + strings.Repeat(" ", maxResponseBytes),
	} {
		t.Run(name, func(t *testing.T) {
			if result, err := decodeBatch([]byte(body), 1); err == nil || result != nil {
				t.Fatalf("incomplete/ambiguous coverage accepted: %#v, %v", result, err)
			}
		})
	}
	for _, expected := range []int{0, -1, 1001} {
		if result, err := decodeBatch([]byte(`{"results":[{}]}`), expected); err == nil || result != nil {
			t.Fatalf("invalid expected membership %d accepted", expected)
		}
	}
}
