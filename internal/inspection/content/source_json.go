// Package content interprets bounded source-analysis DATA for an exact artifact.
package content

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const sourceOutputLimit = 1 << 20
const sourceSignalLimit = 512

var errSourceOutput = errors.New("source analysis output is incomplete, ambiguous or inconsistent with its fixed inputs")

type sourcePosition struct {
	Line   int64 `json:"line"`
	Column int64 `json:"column"`
	Offset int64 `json:"offset"`
}
type sourceSignal struct {
	Rule, Severity, Path string
	RuleVersion          int
	Start, End           sourcePosition
}
type sourceScan struct {
	signals                                     []sourceSignal
	scannedFiles                                map[string]bool
	complete                                    bool
	scanned, errors, skipped, timeouts, matches int
}
type sourceRule struct {
	ID        string   `json:"id"`
	Languages []string `json:"languages"`
	Severity  string   `json:"severity"`
	Message   string   `json:"message"`
	Metadata  struct {
		Version     int    `json:"rule_version"`
		Disposition string `json:"disposition"`
	} `json:"metadata"`
}

// Read the document once, rather than importing an unrelated advisory parser
// or introducing a shared parser framework. Objects are case-sensitive maps;
// duplicate and Unicode case-fold aliases are rejected before interpretation.
func sourceValue(d *json.Decoder, depth int) (any, error) {
	if depth > 32 {
		return nil, errSourceOutput
	}
	token, err := d.Token()
	if err != nil {
		return nil, errSourceOutput
	}
	delimiter, aggregate := token.(json.Delim)
	if !aggregate {
		return token, nil
	}
	switch delimiter {
	case '{':
		value := map[string]any{}
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			name, ok := key.(string)
			if err != nil || !ok {
				return nil, errSourceOutput
			}
			var folded strings.Builder
			for _, ch := range name {
				canonical := ch
				for next := unicode.SimpleFold(ch); next != ch; next = unicode.SimpleFold(next) {
					if next < canonical {
						canonical = next
					}
				}
				folded.WriteRune(canonical)
			}
			if seen[folded.String()] {
				return nil, errSourceOutput
			}
			seen[folded.String()] = true
			member, err := sourceValue(d, depth+1)
			if err != nil {
				return nil, err
			}
			value[name] = member
		}
		end, err := d.Token()
		if err != nil || end != json.Delim('}') {
			return nil, errSourceOutput
		}
		return value, nil
	case '[':
		value := []any{}
		for d.More() {
			member, err := sourceValue(d, depth+1)
			if err != nil {
				return nil, err
			}
			value = append(value, member)
		}
		end, err := d.Token()
		if err != nil || end != json.Delim(']') {
			return nil, errSourceOutput
		}
		return value, nil
	default:
		return nil, errSourceOutput
	}
}

func sourceObject(value any, required []string, optional ...string) (map[string]any, bool) {
	object, ok := value.(map[string]any)
	if !ok || object == nil {
		return nil, false
	}
	allowed := make(map[string]bool, len(required)+len(optional))
	for _, key := range required {
		allowed[key] = true
		if _, ok := object[key]; !ok {
			return nil, false
		}
	}
	for _, key := range optional {
		allowed[key] = true
	}
	for key := range object {
		if !allowed[key] {
			return nil, false
		}
	}
	return object, true
}

func sourceString(value any) (string, bool) { text, ok := value.(string); return text, ok }
func sourceInteger(value any) (int64, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	n, err := number.Int64()
	return n, err == nil
}
func sourceLocation(value any, size int) (sourcePosition, bool) {
	row, ok := sourceObject(value, []string{"line", "col", "offset"})
	if !ok {
		return sourcePosition{}, false
	}
	line, l := sourceInteger(row["line"])
	col, c := sourceInteger(row["col"])
	offset, o := sourceInteger(row["offset"])
	return sourcePosition{line, col, offset}, l && c && o && line >= 1 && col >= 1 && offset >= 0 && offset <= int64(size) && line <= int64(size)+1 && col <= int64(size)+1
}

func sourceRuleCatalogue(body []byte) (map[string]sourceRule, error) {
	pin := runtimeidentity.SelectedSemgrepSupply()
	sum := sha256.Sum256(body)
	if int64(len(body)) != pin.Rules.Size || hex.EncodeToString(sum[:]) != pin.Rules.SHA256 {
		return nil, errSourceOutput
	}
	var pack struct {
		Rules []sourceRule `json:"rules"`
	}
	if json.Unmarshal(body, &pack) != nil || len(pack.Rules) == 0 {
		return nil, errSourceOutput
	}
	rules := make(map[string]sourceRule, len(pack.Rules))
	for _, rule := range pack.Rules {
		if rule.ID == "" || rules[rule.ID].ID != "" || rule.Metadata.Version <= 0 || rule.Metadata.Disposition != "manual-review" || rule.Severity != "WARNING" || len(rule.Languages) == 0 {
			return nil, errSourceOutput
		}
		// The pinned rule format calls this language terraform; the fixed
		// target protocol uses its hcl alias. Keep that translation here.
		for index, language := range rule.Languages {
			if language == "terraform" {
				rule.Languages[index] = "hcl"
			}
		}
		rules[rule.ID] = rule
	}
	return rules, nil
}

func decodeSourceScan(input ports.SourceAnalysisInput, output ports.SourceAnalysisOutput) (scan sourceScan, decodeErr error) {
	stage := "runtime-and-subject"
	defer func() {
		if decodeErr != nil {
			decodeErr = fmt.Errorf("source report %s: %w", stage, decodeErr)
		}
	}()
	if output.Runtime.Status() != domain.SandboxCompleted || output.Output == nil || output.Output.StdoutOverflow() || output.Output.StderrOverflow() || input.Artifact.Digest().String() == "" || len(input.Files) == 0 || len(output.Targets) != len(input.Files) {
		return scan, errSourceOutput
	}
	stage = "fixed-rule-catalogue"
	rules, err := sourceRuleCatalogue(output.Rules)
	if err != nil {
		return scan, err
	}
	stage = "source-input-and-target-map"
	originals := make(map[string]ports.SourceAnalysisFile, len(input.Files))
	languages := map[string]bool{}
	for _, rule := range rules {
		for _, language := range rule.Languages {
			languages[language] = true
		}
	}
	for _, file := range input.Files {
		if !languages[file.Language] || !utf8.Valid(file.Body) {
			return scan, errSourceOutput
		}
		if _, found := originals[file.Path]; found {
			return scan, errSourceOutput
		}
		originals[file.Path] = file
	}
	files := make(map[string]ports.SourceAnalysisFile, len(output.Targets))
	seenOriginal := map[string]bool{}
	for opaque, original := range output.Targets {
		file, ok := originals[original]
		if !ok || opaque == "" || seenOriginal[original] {
			return scan, errSourceOutput
		}
		seenOriginal[original] = true
		files[opaque] = file
	}
	stage = "json-document"
	body := output.Output.Stdout()
	if len(body) == 0 || len(body) > sourceOutputLimit || !utf8.Valid(body) {
		return scan, errSourceOutput
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	value, err := sourceValue(decoder, 0)
	if err != nil {
		return scan, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return scan, errSourceOutput
	}
	stage = "root-version-and-engine"
	root, ok := sourceObject(value, []string{"version", "results", "errors", "paths", "time", "rules_by_engine", "engine_requested", "interfile_languages_used", "skipped_rules", "profiling_results"})
	if !ok || root["version"] != runtimeidentity.SelectedSemgrepSupply().Version || root["engine_requested"] != "OSS" {
		return scan, errSourceOutput
	}
	for _, key := range []string{"interfile_languages_used", "profiling_results"} {
		rows, ok := root[key].([]any)
		if !ok || len(rows) != 0 {
			return scan, errSourceOutput
		}
	}
	stage = "rule-engine-census"
	engines, ok := root["rules_by_engine"].([]any)
	if !ok || len(engines) != len(rules) {
		return scan, errSourceOutput
	}
	knownRules := map[string]bool{}
	for _, value := range engines {
		row, ok := value.([]any)
		if !ok || len(row) != 2 || row[1] != "OSS" {
			return scan, errSourceOutput
		}
		id, ok := sourceString(row[0])
		if !ok || rules[id].ID == "" || knownRules[id] {
			return scan, errSourceOutput
		}
		knownRules[id] = true
	}
	stage = "scanned-target-inventory"
	paths, ok := sourceObject(root["paths"], []string{"scanned"}, "skipped")
	if !ok {
		return scan, errSourceOutput
	}
	scanned, ok := paths["scanned"].([]any)
	if !ok {
		return scan, errSourceOutput
	}
	knownPaths := map[string]bool{}
	for _, value := range scanned {
		name, ok := sourceString(value)
		if !ok || files[name].Path == "" || knownPaths[name] {
			return scan, errSourceOutput
		}
		knownPaths[name] = true
	}
	scan.scanned = len(scanned)
	scan.scannedFiles = make(map[string]bool, len(knownPaths))
	for opaque := range knownPaths {
		scan.scannedFiles[files[opaque].Path] = true
	}
	if value, found := paths["skipped"]; found {
		rows, ok := value.([]any)
		if !ok {
			return scan, errSourceOutput
		}
		scan.skipped += len(rows)
	}
	stage = "error-and-skip-coverage"
	errors, ok := root["errors"].([]any)
	if !ok {
		return scan, errSourceOutput
	}
	scan.errors = len(errors)
	skipped, ok := root["skipped_rules"].([]any)
	if !ok {
		return scan, errSourceOutput
	}
	scan.skipped += len(skipped)
	stage = "timeout-coverage"
	timing, ok := sourceObject(root["time"], []string{"rules", "rules_parse_time", "profiling_times", "parsing_time", "scanning_time", "matching_time", "tainting_time", "fixpoint_timeouts", "prefiltering", "targets", "total_bytes", "max_memory_bytes"})
	if !ok {
		return scan, errSourceOutput
	}
	for _, key := range []string{"rules", "targets"} {
		if _, ok := timing[key].([]any); !ok {
			return scan, errSourceOutput
		}
	}
	for _, key := range []string{"profiling_times", "parsing_time", "scanning_time", "matching_time", "tainting_time", "prefiltering"} {
		if _, ok := timing[key].(map[string]any); !ok {
			return scan, errSourceOutput
		}
	}
	for _, key := range []string{"rules_parse_time", "total_bytes", "max_memory_bytes"} {
		number, ok := timing[key].(json.Number)
		if !ok {
			return scan, errSourceOutput
		}
		value, err := number.Float64()
		if err != nil || value < 0 {
			return scan, errSourceOutput
		}
	}
	timeouts, ok := timing["fixpoint_timeouts"].([]any)
	if !ok {
		return scan, errSourceOutput
	}
	scan.timeouts = len(timeouts)
	stage = "source-signals"
	results, ok := root["results"].([]any)
	if !ok {
		return scan, errSourceOutput
	}
	positions := make(map[string][]sourcePosition)
	unique := map[string]bool{}
	for _, value := range results {
		result, ok := sourceObject(value, []string{"check_id", "path", "start", "end", "extra"})
		if !ok {
			return sourceScan{}, errSourceOutput
		}
		id, ok := sourceString(result["check_id"])
		rule, known := rules[id]
		if !ok || !known {
			return sourceScan{}, errSourceOutput
		}
		target, ok := sourceString(result["path"])
		file, known := files[target]
		if !ok || !known || !knownPaths[target] {
			return sourceScan{}, errSourceOutput
		}
		language := false
		for _, name := range rule.Languages {
			language = language || name == file.Language
		}
		if !language {
			return sourceScan{}, errSourceOutput
		}
		start, ok := sourceLocation(result["start"], len(file.Body))
		end, valid := sourceLocation(result["end"], len(file.Body))
		if !ok || !valid || start.Offset >= end.Offset {
			return sourceScan{}, errSourceOutput
		}
		extra, ok := sourceObject(result["extra"], []string{"metavars", "engine_kind", "is_ignored", "message", "metadata", "severity", "validation_state"}, "dataflow_trace")
		if !ok || extra["engine_kind"] != "OSS" || extra["severity"] != rule.Severity || extra["message"] != rule.Message || extra["validation_state"] != "NO_VALIDATOR" {
			return sourceScan{}, errSourceOutput
		}
		if _, ok := extra["metavars"].(map[string]any); !ok {
			return sourceScan{}, errSourceOutput
		}
		// This optional explanation contains source snippets. Validate its
		// document shape and discard it; it supplies no completion authority.
		if trace, present := extra["dataflow_trace"]; present {
			if _, ok := trace.(map[string]any); !ok {
				return sourceScan{}, errSourceOutput
			}
		}
		if _, ok := extra["is_ignored"].(bool); !ok {
			return sourceScan{}, errSourceOutput
		}
		metadata, ok := sourceObject(extra["metadata"], []string{"rule_version", "disposition"})
		if !ok || metadata["disposition"] != rule.Metadata.Disposition {
			return sourceScan{}, errSourceOutput
		}
		version, ok := sourceInteger(metadata["rule_version"])
		if !ok || version != int64(rule.Metadata.Version) {
			return sourceScan{}, errSourceOutput
		}
		keyBytes, _ := json.Marshal([]any{id, target, start.Offset, end.Offset})
		key := string(keyBytes)
		if unique[key] {
			return sourceScan{}, errSourceOutput
		}
		unique[key] = true
		scan.signals = append(scan.signals, sourceSignal{id, rule.Severity, file.Path, rule.Metadata.Version, start, end})
		positions[target] = append(positions[target], start, end)
	}
	// Verify byte offsets and line/column pairs with one forward pass per source.
	// This does not rescan a large file for every match or retain every newline.
	stage = "source-position-binding"
	for target, points := range positions {
		sort.Slice(points, func(a, b int) bool { return points[a].Offset < points[b].Offset })
		file := files[target]
		line, col, offset := int64(1), int64(1), int64(0)
		for _, point := range points {
			for offset < point.Offset {
				if file.Body[offset] == '\n' {
					line++
					col = 1
				} else {
					col++
				}
				offset++
			}
			if point.Line != line || point.Column != col {
				return sourceScan{}, errSourceOutput
			}
		}
	}
	scan.matches = len(scan.signals)
	scan.complete = scan.scanned == len(files) && scan.errors == 0 && scan.skipped == 0 && scan.timeouts == 0 && scan.matches <= sourceSignalLimit
	if len(scan.signals) > sourceSignalLimit {
		scan.signals = scan.signals[:sourceSignalLimit]
	}
	return scan, nil
}
