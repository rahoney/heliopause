package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type semgrepAsset = runtimeidentity.SemgrepSupplyAsset
type semgrepMaterializationOwner = runtimeidentity.SemgrepSupplyOwner

func readSemgrepMaterializationOwner(root string) (semgrepMaterializationOwner, error) {
	body, _, err := readRegular(root, runtimeFile)
	if err != nil {
		return semgrepMaterializationOwner{}, err
	}
	return parseSemgrepMaterializationOwner(body)
}

func parseSemgrepMaterializationOwner(body []byte) (semgrepMaterializationOwner, error) {
	var pin semgrepMaterializationOwner
	fields, err := jsonSpans(body)
	if err != nil {
		return pin, err
	}
	schema, ok := fields["/schema_version"]
	var version int
	if !ok || json.Unmarshal(body[schema.start:schema.end], &version) != nil || version != 6 {
		return pin, errors.New("semgrep runtime owner requires the explicit current schema")
	}
	selected, ok := fields["/semgrep_tool"]
	if !ok {
		return pin, errors.New("semgrep selected runtime owner is missing")
	}
	expected := map[string]bool{"/semgrep_tool": true}
	for _, key := range []string{"semgrep_version", "architecture", "runtime_image", "source_commit", "source_tag_object", "source_manifest_sha256", "source_date_epoch"} {
		expected["/semgrep_tool/"+key] = true
	}
	for _, key := range []string{"tool", "sources", "notices", "rules", "patch", "worker"} {
		expected["/semgrep_tool/"+key] = true
		for _, suffix := range []string{"path", "sha256", "size"} {
			expected["/semgrep_tool/"+key+"/"+suffix] = true
		}
	}
	count := 0
	for field := range fields {
		if field == "/semgrep_tool" || strings.HasPrefix(field, "/semgrep_tool/") {
			if !expected[field] {
				return pin, errors.New("semgrep runtime owner has unknown or noncanonical fields")
			}
			count++
		}
	}
	if count != len(expected) {
		return pin, errors.New("semgrep runtime owner has missing fields")
	}
	decoder := json.NewDecoder(bytes.NewReader(body[selected.start:selected.end]))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&pin) != nil || decoder.Decode(new(any)) != io.EOF || !runtimeidentity.ValidSemgrepSupplyOwner(pin) {
		return semgrepMaterializationOwner{}, errors.New("semgrep selected runtime owner is invalid")
	}
	return pin, nil
}

// Same upstream identity is required for an explicitly requested HAA-only refresh.
func sameSemgrepUpstream(a, b semgrepMaterializationOwner) bool {
	return a.Version == b.Version && a.SourceCommit == b.SourceCommit && a.SourceTagObject == b.SourceTagObject && a.SourceDateEpoch == b.SourceDateEpoch
}
