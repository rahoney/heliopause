package advisory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	osvEndpoint       = "https://api.osv.dev/v1/querybatch"
	maxLookupPackages = 4096
	maxLookupRequests = 64
	maxLookupBytes    = 16 << 20
	maxLookupIDs      = 8192
)

type osvPackage struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Version   string `json:"-"`
}

type responseFact struct {
	requestSHA256, responseSHA256 string
	fetchedAt                     time.Time
}

type batchQuery struct {
	Package   osvPackage `json:"package"`
	Version   string     `json:"version"`
	PageToken string     `json:"page_token,omitempty"`
}

// queryBatch queries each distinct package once, follows only that package's
// pages, then maps results to the original positions. It grants no guest egress
// or installation authority. The owning adapter supplies a trusted transport.
func queryBatch(ctx context.Context, client *http.Client, packages []osvPackage) ([][]advisoryReference, []responseFact, error) {
	if ctx == nil || client == nil || client.Transport == nil || len(packages) == 0 || len(packages) > maxLookupPackages {
		return nil, nil, errResponse
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	httpClient := *client
	httpClient.Jar = nil
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if httpClient.Timeout <= 0 || httpClient.Timeout > 10*time.Second {
		httpClient.Timeout = 10 * time.Second
	}
	unique := make([]osvPackage, 0, len(packages))
	byPackage := make(map[osvPackage]int, len(packages))
	positions := make([]int, len(packages))
	for index, p := range packages {
		if !validPackage(p) {
			return nil, nil, errResponse
		}
		position, exists := byPackage[p]
		if !exists {
			position = len(unique)
			byPackage[p] = position
			unique = append(unique, p)
		}
		positions[index] = position
	}
	results := make([][]advisoryReference, len(unique))
	ids := make([]map[string]bool, len(unique))
	tokens := make([]map[string]bool, len(unique))
	modified := map[string]time.Time{}
	remainingBytes, countIDs := int64(maxLookupBytes), 0
	facts := make([]responseFact, 0)
	for start := 0; start < len(unique); start += maxBatchPackages {
		end := min(start+maxBatchPackages, len(unique))
		pending := make([]batchQuery, end-start)
		indices := make([]int, end-start)
		for n := range pending {
			p := unique[start+n]
			pending[n] = batchQuery{Package: p, Version: p.Version}
			indices[n] = start + n
			ids[start+n], tokens[start+n] = map[string]bool{}, map[string]bool{}
		}
		for len(pending) != 0 {
			if err := ctx.Err(); err != nil {
				return nil, facts, err
			}
			if len(facts) >= maxLookupRequests || remainingBytes <= 0 {
				return nil, facts, errResponse
			}
			payload, err := json.Marshal(struct {
				Queries []batchQuery `json:"queries"`
			}{pending})
			if err != nil {
				return nil, facts, err
			}
			body, err := postBatch(ctx, &httpClient, payload, min(int64(maxResponseBytes), remainingBytes))
			if err != nil {
				return nil, facts, err
			}
			remainingBytes -= int64(len(body))
			requestHash, responseHash := sha256.Sum256(payload), sha256.Sum256(body)
			facts = append(facts, responseFact{hex.EncodeToString(requestHash[:]), hex.EncodeToString(responseHash[:]), time.Now().UTC()})
			page, err := decodeBatch(body, len(pending))
			if err != nil {
				return nil, facts, err
			}
			next := make([]batchQuery, 0, len(pending))
			nextIndices := make([]int, 0, len(pending))
			for n, result := range page {
				index := indices[n]
				for _, ref := range result.vulnerabilities {
					previous, exists := modified[ref.id]
					if ids[index][ref.id] || exists && !previous.Equal(ref.modified) || countIDs >= maxLookupIDs {
						return nil, facts, errResponse
					}
					modified[ref.id], ids[index][ref.id] = ref.modified, true
					results[index] = append(results[index], ref)
					countIDs++
				}
				if result.nextPage != "" {
					if tokens[index][result.nextPage] {
						return nil, facts, errResponse
					}
					tokens[index][result.nextPage] = true
					query := pending[n]
					query.PageToken = result.nextPage
					next, nextIndices = append(next, query), append(nextIndices, index)
				}
			}
			pending, indices = next, nextIndices
		}
	}
	ordered := make([][]advisoryReference, len(packages))
	for index, position := range positions {
		ordered[index] = append([]advisoryReference(nil), results[position]...)
	}
	return ordered, facts, nil
}

func validPackage(p osvPackage) bool {
	switch p.Ecosystem {
	case "npm", "PyPI", "Go", "crates.io":
	default:
		return false
	}
	for _, value := range []string{p.Name, p.Version} {
		if value == "" || len(value) > 256 || !utf8.ValidString(value) {
			return false
		}
		for _, ch := range value {
			if unicode.IsControl(ch) || unicode.IsSpace(ch) {
				return false
			}
		}
	}
	return true
}

func postBatch(ctx context.Context, client *http.Client, payload []byte, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, osvEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	contentType, _, typeErr := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if response.StatusCode != http.StatusOK || typeErr != nil || contentType != "application/json" || response.Header.Get("Content-Encoding") != "" {
		return nil, errResponse
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || int64(len(body)) > limit {
		return nil, errors.Join(errResponse, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return body, nil
}
