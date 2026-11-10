package advisory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type osvSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

type osvEvent struct {
	Introduced   string `json:"introduced"`
	Fixed        string `json:"fixed"`
	LastAffected string `json:"last_affected"`
	Limit        string `json:"limit"`
}

type osvRange struct {
	Type   string     `json:"type"`
	Repo   string     `json:"repo"`
	Events []osvEvent `json:"events"`
}

type osvAffected struct {
	Package  osvPackage    `json:"package"`
	Versions []string      `json:"versions"`
	Ranges   []osvRange    `json:"ranges"`
	Severity []osvSeverity `json:"severity"`
}

// Raw provider severity/ranges are facts for later evaluation, not a verdict.
type osvRecord struct {
	SchemaVersion string        `json:"schema_version"`
	ID            string        `json:"id"`
	Modified      time.Time     `json:"modified"`
	Withdrawn     *time.Time    `json:"withdrawn"`
	Aliases       []string      `json:"aliases"`
	Affected      []osvAffected `json:"affected"`
	Severity      []osvSeverity `json:"severity"`
	Database      struct {
		Severity string `json:"severity"`
	} `json:"database_specific"`
}

func decodeRecord(body []byte, expected advisoryReference) (osvRecord, error) {
	if !advisoryID.MatchString(expected.id) || expected.modified.IsZero() || validateJSON(body) != nil {
		return osvRecord{}, errResponse
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields["id"] == nil || fields["modified"] == nil {
		return osvRecord{}, errResponse
	}
	var result osvRecord
	if json.Unmarshal(body, &result) != nil || result.ID != expected.id || result.Withdrawn != nil && result.Withdrawn.IsZero() {
		return osvRecord{}, errResponse
	}
	// OSV querybatch reads its index timestamp from Datastore (microseconds,
	// rounded down); GetFull retains the record's original nanoseconds. Compare
	// at that index precision only when the index is microsecond-aligned. A
	// higher precision index must still match exactly. Keep Modified unchanged.
	if !result.Modified.Equal(expected.modified) && !(expected.modified.Nanosecond()%1000 == 0 && result.Modified.Truncate(time.Microsecond).Equal(expected.modified)) {
		return osvRecord{}, errResponse
	}
	// OSV 1.x changes are additive. Missing schema_version is defined as 1.0.0.
	if value, found := fields["schema_version"]; found && (json.Unmarshal(value, &result.SchemaVersion) != nil || !validRecordSchema(result.SchemaVersion)) {
		return osvRecord{}, errResponse
	}
	if fields["schema_version"] == nil && result.SchemaVersion != "" {
		return osvRecord{}, errResponse
	}
	if len(result.Aliases) > 128 || len(result.Affected) > 1024 || !validSeverities(result.Severity) || len(result.Database.Severity) > 32 || len(result.Affected) == 0 && result.Withdrawn == nil {
		return osvRecord{}, errResponse
	}
	seen := map[string]bool{result.ID: true}
	for _, alias := range result.Aliases {
		if !advisoryID.MatchString(alias) || seen[alias] {
			return osvRecord{}, errResponse
		}
		seen[alias] = true
	}
	for _, affected := range result.Affected {
		// Other ecosystems can coexist in one advisory; only mapped packages
		// are evaluated by its owning inspection. Do not guess their identity.
		if affected.Package.Ecosystem == "" || len(affected.Package.Ecosystem) > 64 || affected.Package.Name == "" || len(affected.Package.Name) > 256 || len(affected.Versions) > 8192 || len(affected.Ranges) > 256 || !validSeverities(affected.Severity) {
			return osvRecord{}, errResponse
		}
		for _, version := range affected.Versions {
			if version == "" || len(version) > 256 {
				return osvRecord{}, errResponse
			}
		}
		for _, interval := range affected.Ranges {
			if interval.Type == "" || len(interval.Type) > 32 || len(interval.Repo) > 2048 || interval.Type == "GIT" && interval.Repo == "" || len(interval.Events) == 0 || len(interval.Events) > 1024 {
				return osvRecord{}, errResponse
			}
			for _, event := range interval.Events {
				count := 0
				for _, value := range []string{event.Introduced, event.Fixed, event.LastAffected, event.Limit} {
					if len(value) > 256 {
						return osvRecord{}, errResponse
					}
					if value != "" {
						count++
					}
				}
				if count != 1 {
					return osvRecord{}, errResponse
				}
			}
		}
	}
	return result, nil
}

func validSeverities(values []osvSeverity) bool {
	if len(values) > 16 {
		return false
	}
	for _, value := range values {
		if value.Type == "" || len(value.Type) > 32 || value.Score == "" || len(value.Score) > 512 {
			return false
		}
	}
	return true
}

func validRecordSchema(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 3 || parts[0] != "1" || len(value) > 32 {
		return false
	}
	for _, part := range parts {
		if part == "" || len(part) > 1 && part[0] == '0' {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

func getRecord(ctx context.Context, client *http.Client, expected advisoryReference, limit int64) (osvRecord, responseFact, error) {
	if ctx == nil || client == nil || client.Transport == nil || !advisoryID.MatchString(expected.id) || expected.id == "." || expected.id == ".." || expected.modified.IsZero() || limit <= 0 || limit > maxResponseBytes {
		return osvRecord{}, responseFact{}, errResponse
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.osv.dev/v1/vulns/"+expected.id, nil)
	if err != nil {
		return osvRecord{}, responseFact{}, err
	}
	body, err := readResponse(ctx, client, request, limit)
	if err != nil {
		return osvRecord{}, responseFact{}, err
	}
	// The method and exact fixed URL bind this read; no query/body is present.
	requestHash := sha256.Sum256([]byte(http.MethodGet + " " + request.URL.String()))
	responseHash := sha256.Sum256(body)
	fact := responseFact{hex.EncodeToString(requestHash[:]), hex.EncodeToString(responseHash[:]), time.Now().UTC(), int64(len(body))}
	record, err := decodeRecord(body, expected)
	return record, fact, err
}

// lookupAdvisories shares the batch budget with record reads; detailed data
// never receives a fresh request/byte/wall allowance per package or record.
func lookupAdvisories(ctx context.Context, client *http.Client, packages []osvPackage) ([][]osvRecord, []responseFact, error) {
	if ctx == nil {
		return nil, nil, errResponse
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	ids, facts, err := queryBatch(ctx, client, packages)
	if err != nil {
		return nil, facts, err
	}
	remaining := int64(maxLookupBytes)
	for _, fact := range facts {
		remaining -= fact.responseBytes
	}
	records := make(map[string]osvRecord)
	for _, references := range ids {
		for _, reference := range references {
			if _, found := records[reference.id]; found {
				continue
			}
			if len(facts) >= maxLookupRequests || remaining <= 0 {
				return nil, facts, errResponse
			}
			record, fact, err := getRecord(ctx, client, reference, min(int64(maxResponseBytes), remaining))
			if fact.responseSHA256 != "" {
				facts = append(facts, fact)
				remaining -= fact.responseBytes
			}
			if err != nil {
				return nil, facts, err
			}
			records[reference.id] = record
		}
	}
	ordered := make([][]osvRecord, len(packages))
	for index, references := range ids {
		for _, reference := range references {
			ordered[index] = append(ordered[index], records[reference.id])
		}
	}
	return ordered, facts, nil
}
