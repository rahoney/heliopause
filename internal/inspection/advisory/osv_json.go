// Package advisory owns bounded advisory responses for exact selected packages.
package advisory

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxResponseBytes = 2 << 20
	maxBatchPackages = 1000
)

var (
	advisoryID  = regexp.MustCompile(`^[A-Za-z0-9:_.-]{1,192}$`)
	errResponse = errors.New("advisory response is malformed, ambiguous or exceeds bounds")
)

type advisoryReference struct {
	id       string
	modified time.Time
}

type batchResult struct {
	vulnerabilities []advisoryReference
	nextPage        string
}

// decodeBatch retains exact request ordering. A missing/null result is never an
// empty result. Only the provider's complete empty object means no matching IDs.
func decodeBatch(body []byte, expected int) ([]batchResult, error) {
	if expected <= 0 || expected > maxBatchPackages || validateJSON(body) != nil {
		return nil, errResponse
	}
	var outer map[string]json.RawMessage
	if json.Unmarshal(body, &outer) != nil || len(outer) != 1 || outer["results"] == nil {
		return nil, errResponse
	}
	var rows []json.RawMessage
	if json.Unmarshal(outer["results"], &rows) != nil || len(rows) != expected {
		return nil, errResponse
	}
	results := make([]batchResult, len(rows))
	for index, raw := range rows {
		var row map[string]json.RawMessage
		if json.Unmarshal(raw, &row) != nil || row == nil || len(row) > 2 {
			return nil, errResponse
		}
		for key := range row {
			if key != "vulns" && key != "next_page_token" {
				return nil, errResponse
			}
		}
		if token, found := row["next_page_token"]; found {
			if json.Unmarshal(token, &results[index].nextPage) != nil || !validPageToken(results[index].nextPage) {
				return nil, errResponse
			}
		}
		if value, found := row["vulns"]; found {
			var references []map[string]json.RawMessage
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) || json.Unmarshal(value, &references) != nil || len(references) > 1000 {
				return nil, errResponse
			}
			seen := make(map[string]bool, len(references))
			for _, reference := range references {
				var id, modified string
				if len(reference) != 2 || json.Unmarshal(reference["id"], &id) != nil || !advisoryID.MatchString(id) || seen[id] || json.Unmarshal(reference["modified"], &modified) != nil || len(modified) > 64 {
					return nil, errResponse
				}
				stamp, err := time.Parse(time.RFC3339Nano, modified)
				if err != nil || stamp.IsZero() {
					return nil, errResponse
				}
				seen[id] = true
				results[index].vulnerabilities = append(results[index].vulnerabilities, advisoryReference{id, stamp})
			}
		}
	}
	return results, nil
}

func validPageToken(value string) bool {
	if value == "" || len(value) > 2048 {
		return false
	}
	for _, ch := range value {
		if ch < 33 || ch > 126 {
			return false
		}
	}
	return true
}

// Walk before typed decoding: encoding/json accepts duplicate and case-folded
// field names, invalid UTF-8 and trailing values that must not define coverage.
func validateJSON(body []byte) error {
	if len(body) == 0 || len(body) > maxResponseBytes || !utf8.Valid(body) {
		return errResponse
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	if jsonValue(d, 0) != nil {
		return errResponse
	}
	if _, err := d.Token(); err != io.EOF {
		return errResponse
	}
	return nil
}

func jsonValue(d *json.Decoder, depth int) error {
	if depth > 32 {
		return errResponse
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	delimiter, aggregate := token.(json.Delim)
	if !aggregate {
		return nil
	}
	if delimiter != '{' && delimiter != '[' {
		return errResponse
	}
	seen := map[string]bool{}
	for d.More() {
		if delimiter == '{' {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			folded := strings.ToLower(name)
			if !ok || seen[folded] {
				return errResponse
			}
			seen[folded] = true
		}
		if err := jsonValue(d, depth+1); err != nil {
			return err
		}
	}
	closing, err := d.Token()
	if err != nil || delimiter == '{' && closing != json.Delim('}') || delimiter == '[' && closing != json.Delim(']') {
		return errResponse
	}
	return nil
}
