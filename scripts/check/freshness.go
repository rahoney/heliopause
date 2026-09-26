package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type supportRecordFile struct {
	SchemaVersion int             `json:"schema_version"`
	Records       []supportRecord `json:"records"`
}
type supportRecord struct {
	Component      string  `json:"component"`
	PinnedVersion  string  `json:"pinned_version"`
	SupportRole    string  `json:"support_role"`
	OfficialSource string  `json:"official_source"`
	CheckedAt      string  `json:"checked_at"`
	Latest         string  `json:"latest_stable_at_check"`
	Status         string  `json:"support_status"`
	EOL            *string `json:"eol_date_if_known"`
	Decision       string  `json:"decision"`
	Reason         string  `json:"reason"`
}

func checkVersionSupport(root string, strict bool, output io.Writer) error {
	body, err := os.ReadFile(filepath.Join(root, "scripts", "version-support.lock.json"))
	if err != nil {
		return err
	}
	var file supportRecordFile
	decoder := json.NewDecoder(&limitedReader{reader: body})
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return errors.New("version support record is malformed")
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return errors.New("version support record is malformed")
	}
	return validateVersionSupport(file, time.Now().UTC(), strict, output)
}

func validateVersionSupport(file supportRecordFile, now time.Time, strict bool, output io.Writer) error {
	if file.SchemaVersion != 1 || len(file.Records) < 17 {
		return errors.New("version support record schema is invalid")
	}
	stale := false
	seen := map[string]bool{}
	for _, record := range file.Records {
		if record.Component == "" || record.PinnedVersion == "" || record.SupportRole == "" || record.OfficialSource == "" || record.Latest == "" || record.Status == "" || record.Decision == "" || record.Reason == "" || seen[record.Component] {
			return errors.New("version support record has missing or duplicate field")
		}
		seen[record.Component] = true
		checked, err := time.Parse("2006-01-02", record.CheckedAt)
		if err != nil || (record.EOL != nil && !validSupportDate(*record.EOL)) {
			return errors.New("version support record date is invalid")
		}
		if now.Sub(checked) > 90*24*time.Hour {
			stale = true
			fmt.Fprintf(output, "WARNING: version support review is older than 90 days: %s\n", record.Component)
		}
	}
	if stale && strict {
		return errors.New("version support review is stale")
	}
	return nil
}
func validSupportDate(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

type limitedReader struct{ reader []byte }

func (r *limitedReader) Read(p []byte) (int, error) {
	if len(r.reader) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.reader)
	r.reader = r.reader[n:]
	return n, nil
}
