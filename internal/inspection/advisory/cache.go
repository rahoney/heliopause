package advisory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

const cacheSchema = "helox.advisory-cache/v1"

var (
	errCachedResponse = errors.New("advisory cache is incomplete or changed")
	errCachedExpired  = errors.New("advisory cache response expired")
)

// The same proof shape is used by the normalized report and the cache. Stored
// times are the original authenticated fetch times, never the time of a hit.
type responseProof struct {
	Request, Response, Fetched string
	Bytes                      int64
}

type cachedLookup struct {
	Schema    string                     `json:"schema"`
	Profile   string                     `json:"profile"`
	Queries   []batchQuery               `json:"queries"`
	Rows      [][]string                 `json:"rows"`
	Records   map[string]json.RawMessage `json:"records"`
	Responses []responseProof            `json:"responses"`
}

// Query failures and optional cache persistence errors are separate: a complete
// live response remains a completed inspection fact even if its cache write fails.
func (i *Inspector) lookup(ctx context.Context, packages []osvPackage) ([][]osvRecord, []responseFact, error, error) {
	if ctx == nil || len(packages) == 0 || len(packages) > maxLookupPackages {
		return nil, nil, errResponse, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err, nil
	}
	storage := ports.StorageFromContext(ctx)
	cache, enabled := storage.(ports.AdvisoryStorage)
	if !enabled {
		records, facts, err := lookupAdvisories(ctx, i.client, packages)
		return records, facts, err, nil
	}
	queries := make([]batchQuery, len(packages))
	for n, query := range packages {
		if !validPackage(query) {
			return nil, nil, errResponse, nil
		}
		queries[n] = batchQuery{Package: query, Version: query.Version}
	}
	keyBody, err := json.Marshal(struct {
		Schema, Provider, Profile string
		Queries                   []batchQuery
	}{cacheSchema, osvEndpoint, i.profile.String(), queries})
	if err != nil {
		return nil, nil, err, nil
	}
	sum := sha256.Sum256(keyBody)
	key, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if !ports.AdvisoryRefreshFromContext(ctx) {
		body, hit, err := cache.ReadAdvisory(ctx, key, maxLookupBytes)
		if err != nil {
			return nil, nil, errors.Join(errCachedResponse, err), nil
		}
		if hit {
			records, facts, err := i.decodeCached(body, packages, time.Now().UTC(), time.Duration(storage.Policy().Spec().AdvisoryTTLSeconds)*time.Second)
			if cancelled := ctx.Err(); cancelled != nil {
				return nil, nil, errors.Join(err, cancelled), nil
			}
			if err == nil {
				return records, facts, nil, nil
			}
			if !errors.Is(err, errCachedExpired) {
				return nil, nil, errors.Join(errCachedResponse, err), nil
			}
		}
	}
	records, facts, err := lookupAdvisories(ctx, i.client, packages)
	if err != nil {
		return records, facts, err, nil
	}
	if len(records) != len(packages) {
		return records, facts, errResponse, nil
	}
	bundle := cachedLookup{Schema: cacheSchema, Profile: i.profile.String(), Queries: queries, Rows: make([][]string, len(packages)), Records: map[string]json.RawMessage{}}
	for n := 0; n < len(packages) && n < len(records); n++ {
		query := packages[n]
		bundle.Rows[n] = []string{}
		for _, record := range records[n] {
			if _, err := assessRecord(query, record); err != nil {
				return records, facts, nil, nil
			} // assessment will retain its own incomplete report
			bundle.Rows[n] = append(bundle.Rows[n], record.ID)
			if _, exists := bundle.Records[record.ID]; exists {
				continue
			}
			if record.SchemaVersion == "" {
				record.SchemaVersion = "1.0.0"
			} // OSV's defined absent-field default
			body, err := json.Marshal(record)
			if err != nil {
				return records, facts, nil, err
			}
			bundle.Records[record.ID] = body
		}
	}
	for _, fact := range facts {
		bundle.Responses = append(bundle.Responses, responseProof{fact.requestSHA256, fact.responseSHA256, fact.fetchedAt.UTC().Format(time.RFC3339Nano), fact.responseBytes})
	}
	body, err := json.Marshal(bundle)
	if err != nil {
		return records, facts, nil, err
	}
	// Normalization overhead is not permission to exceed the existing finite
	// lookup bound. A valid live result can simply remain uncached.
	if len(body) > maxLookupBytes || validateJSON(body) != nil {
		return records, facts, nil, nil
	}
	_, writeErr := cache.WriteAdvisory(ctx, key, body)
	return records, facts, nil, writeErr
}

func (i *Inspector) decodeCached(body []byte, packages []osvPackage, now time.Time, ttl time.Duration) ([][]osvRecord, []responseFact, error) {
	if len(body) == 0 || len(body) > maxLookupBytes || len(packages) == 0 || len(packages) > maxLookupPackages || ttl <= 0 || validateJSON(body) != nil {
		return nil, nil, errResponse
	}
	var bundle cachedLookup
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&bundle) != nil || decoder.Decode(&struct{}{}) != io.EOF || bundle.Schema != cacheSchema || bundle.Profile != i.profile.String() || len(bundle.Queries) != len(packages) || len(bundle.Rows) != len(packages) || len(bundle.Responses) == 0 || len(bundle.Responses) > maxLookupRequests || bundle.Records == nil || len(bundle.Records) >= maxLookupRequests {
		return nil, nil, errResponse
	}
	var facts []responseFact
	var total int64
	expired := false
	for _, proof := range bundle.Responses {
		if _, err := domain.NewSHA256Digest(proof.Request); err != nil {
			return nil, nil, errResponse
		}
		if _, err := domain.NewSHA256Digest(proof.Response); err != nil {
			return nil, nil, errResponse
		}
		fetched, err := time.Parse(time.RFC3339Nano, proof.Fetched)
		if err != nil || fetched.IsZero() || fetched.After(now) || proof.Bytes <= 0 || proof.Bytes > maxResponseBytes || total > maxLookupBytes-proof.Bytes {
			return nil, nil, errResponse
		}
		total += proof.Bytes
		expired = expired || now.Sub(fetched) >= ttl
		facts = append(facts, responseFact{proof.Request, proof.Response, fetched, proof.Bytes})
	}
	if len(bundle.Records) >= len(facts) {
		return nil, nil, errResponse
	}
	decoded := make(map[string]osvRecord, len(bundle.Records))
	for id, body := range bundle.Records {
		var identity struct {
			ID       string    `json:"id"`
			Modified time.Time `json:"modified"`
		}
		if json.Unmarshal(body, &identity) != nil || identity.ID != id {
			return nil, nil, errResponse
		}
		record, err := decodeRecord(body, advisoryReference{id, identity.Modified})
		if err != nil {
			return nil, nil, err
		}
		decoded[id] = record
	}
	ordered := make([][]osvRecord, len(packages))
	used := map[string]bool{}
	references := 0
	for n, query := range packages {
		stored := bundle.Queries[n]
		if stored.PageToken != "" || (osvPackage{stored.Package.Ecosystem, stored.Package.Name, stored.Version}) != query || bundle.Rows[n] == nil {
			return nil, nil, errResponse
		}
		seen := map[string]bool{}
		for _, id := range bundle.Rows[n] {
			references++
			record, ok := decoded[id]
			if !ok || seen[id] || references > maxLookupIDs {
				return nil, nil, errResponse
			}
			if _, err := assessRecord(query, record); err != nil {
				return nil, nil, err
			}
			seen[id], used[id] = true, true
			ordered[n] = append(ordered[n], record)
		}
	}
	if len(used) != len(decoded) {
		return nil, nil, errResponse
	}
	if expired {
		return nil, nil, errCachedExpired
	}
	return ordered, facts, nil
}
