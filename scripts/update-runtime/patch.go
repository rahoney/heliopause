package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

func selectedVersion(root, component string) (string, error) {
	if component == "terraform" || component == "github" {
		return "inspection-subject", nil
	}
	var lock map[string]any
	if err := loadJSON(filepath.Join(root, runtimeFile), &lock); err != nil {
		return "", err
	}
	key, versionKey := "", ""
	switch component {
	case "go":
		key, versionKey = "go_image", "go_version"
	case "node":
		key = "node_image"
	case "python":
		key, versionKey = "python_image", "python_version"
	case "rust":
		key, versionKey = "rust_image", "rust_version"
	default:
		return "", errors.New("unknown component")
	}
	image, ok := lock[key].(map[string]any)
	if !ok {
		return "", errors.New("runtime owner missing")
	}
	if versionKey != "" {
		value, ok := image[versionKey].(string)
		if !ok || !versionPattern.MatchString(value) {
			return "", errors.New("runtime version missing")
		}
		return value, nil
	}
	ref, ok := image["reference"].(string)
	if !ok {
		return "", errors.New("node image missing")
	}
	value := strings.TrimPrefix(strings.Split(ref, "-slim@")[0], "node:")
	if !versionPattern.MatchString(value) {
		return "", errors.New("node version tag invalid")
	}
	return value, nil
}

// Spans come from the canonical JSON decoder, not textual value replacement.
// Keeping the surrounding bytes avoids unrelated formatting and fixture changes.
type span struct{ start, end int }

func jsonSpans(body []byte) (map[string]span, error) {
	d := json.NewDecoder(bytes.NewReader(body))
	spans := map[string]span{}
	var walk func(string) error
	walk = func(path string) error {
		start := int(d.InputOffset())
		for start < len(body) && strings.ContainsRune(" \t\r\n:,", rune(body[start])) {
			start++
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					k, err := d.Token()
					if err != nil {
						return err
					}
					key, ok := k.(string)
					if !ok || seen[key] {
						return errors.New("duplicate/non-string JSON key")
					}
					seen[key] = true
					if err := walk(path + "/" + key); err != nil {
						return err
					}
				}
			case '[':
				for i := 0; d.More(); i++ {
					if err := walk(path + "/" + strconv.Itoa(i)); err != nil {
						return err
					}
				}
			default:
				return errors.New("unexpected JSON delimiter")
			}
			if _, err := d.Token(); err != nil {
				return err
			}
		}
		spans[path] = span{start, int(d.InputOffset())}
		return nil
	}
	if err := walk(""); err != nil {
		return nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, errors.New("trailing JSON")
	}
	return spans, nil
}
func editJSON(body []byte, replacements map[string]any) ([]byte, error) {
	spans, err := jsonSpans(body)
	if err != nil {
		return nil, err
	}
	type edit struct {
		span
		body []byte
	}
	edits := []edit{}
	for key, value := range replacements {
		s, ok := spans[key]
		if !ok {
			return nil, fmt.Errorf("unknown JSON consumer %s", key)
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		edits = append(edits, edit{s, encoded})
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	last := len(body) + 1
	for _, e := range edits {
		if e.end > last {
			return nil, errors.New("overlapping JSON edits")
		}
		body = append(append(append([]byte{}, body[:e.start]...), e.body...), body[e.end:]...)
		last = e.start
	}
	return body, nil
}

func renderChanges(p plan, state string) ([]change, error) {
	bin, err := os.ReadFile(filepath.Join(state, "generator"))
	if err != nil {
		return nil, err
	}
	if hash(bin) != p.GeneratorSHA256 {
		return nil, errors.New("frozen generator identity changed")
	}
	shadow, err := os.MkdirTemp(state, "render-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(shadow)
	files := map[string][]byte{}
	for _, path := range []string{runtimeFile, supportFile, "tools/gvisor-observer/BUILD"} {
		body, _, err := readRegular(p.Root, path)
		if err != nil {
			return nil, err
		}
		files[path] = body
	}
	if p.Version != p.Current && p.Component != "terraform" && p.Component != "github" {
		if err := validateMetadata(p.Component, p.Version, p.Metadata); err != nil {
			return nil, err
		}
		key := map[string]string{"go": "go_image", "node": "node_image", "python": "python_image", "rust": "rust_image"}[p.Component]
		replacements := map[string]any{"/" + key + "/reference": p.Metadata.Image}
		owners := map[string]string{}
		switch p.Component {
		case "go":
			replacements["/go_image/go_version"] = p.Version
			owners["Go"] = p.Version
		case "node":
			replacements["/node_image/npm_version"] = p.Metadata.Bundled
			owners["Node"] = p.Version
			owners["npm"] = p.Metadata.Bundled
		case "python":
			replacements["/python_image/python_version"] = p.Version
			replacements["/python_image/pip_version"] = p.Metadata.Bundled
			parts := strings.Split(p.Version, ".")
			abi := "cp" + parts[0] + parts[1]
			replacements["/python_image/target/interpreter"] = abi
			replacements["/python_image/target/abi"] = abi
			owners["Python"] = p.Version
			owners["pip"] = p.Metadata.Bundled
		case "rust":
			replacements["/rust_image/rust_version"] = p.Version
			owners["Rust/Cargo"] = p.Version
		default:
			return nil, errors.New("unknown component")
		}
		files[runtimeFile], err = editJSON(files[runtimeFile], replacements)
		if err != nil {
			return nil, err
		}
		files[supportFile], err = editReviews(files[supportFile], owners, p.Review)
		if err != nil {
			return nil, err
		}
		if p.Component == "go" {
			body, _, err := readRegular(p.Root, "go.mod")
			if err != nil {
				return nil, err
			}
			old := []byte("\ngo " + p.Current + "\n")
			if bytes.Count(body, old) != 1 || bytes.Contains(body, []byte("\ntoolchain ")) {
				return nil, errors.New("unknown go.mod version consumer")
			}
			files["go.mod"] = bytes.Replace(body, old, []byte("\ngo "+p.Version+"\n"), 1)
			body, _, err = readRegular(p.Root, "scripts/tools.lock.json")
			if err != nil {
				return nil, err
			}
			var tools struct {
				Tools []struct {
					Setup string `json:"setupGo"`
				} `json:"tools"`
			}
			if err := json.Unmarshal(body, &tools); err != nil {
				return nil, err
			}
			if len(tools.Tools) != 5 {
				return nil, errors.New("unknown quality tool inventory")
			}
			replacements = map[string]any{}
			for i, t := range tools.Tools {
				if t.Setup != p.Current {
					return nil, errors.New("partial tool setup baseline")
				}
				replacements["/tools/"+strconv.Itoa(i)+"/setupGo"] = p.Version
			}
			files["scripts/tools.lock.json"], err = editJSON(body, replacements)
			if err != nil {
				return nil, err
			}
			known := map[string]bool{"heliopause-ci.yml": true, "heliopause-security.yml": true, "heliopause-release-build.yml": true, "heliopause-release-publish.yml": true}
			entries, err := os.ReadDir(filepath.Join(p.Root, ".github/workflows"))
			if err != nil {
				return nil, err
			}
			for _, entry := range entries {
				if !strings.HasSuffix(entry.Name(), ".yml") && !strings.HasSuffix(entry.Name(), ".yaml") {
					continue
				}
				if !known[entry.Name()] {
					return nil, errors.New("unknown workflow consumer")
				}
				delete(known, entry.Name())
				path := ".github/workflows/" + entry.Name()
				body, _, err := readRegular(p.Root, path)
				if err != nil {
					return nil, err
				}
				files[path], err = editWorkflow(body, p.Current, p.Version)
				if err != nil {
					return nil, err
				}
			}
			if len(known) != 0 {
				return nil, errors.New("missing workflow consumer")
			}
		}
	} else if len(p.Review) != 0 || !reflect.DeepEqual(p.Metadata, metadata{}) {
		return nil, errors.New("same-version plan must not refresh metadata or reviews")
	}
	if p.GvisorCapture != nil {
		files[runtimeFile], err = gvisorCaptureChanges(p, state, files[runtimeFile])
		if err != nil {
			return nil, err
		}
	}
	for path, body := range files {
		full := filepath.Join(shadow, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(full, body, 0o644); err != nil {
			return nil, err
		}
	}
	if _, err := command(shadow, nil, filepath.Join(state, "generator")); err != nil {
		return nil, err
	}
	files[generatedFile], err = os.ReadFile(filepath.Join(shadow, generatedFile))
	if err != nil {
		return nil, err
	}
	changes := []change{}
	for path, after := range files {
		before, _, err := readRegular(p.Root, path)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(before, after) {
			changes = append(changes, change{path, before, after})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	if p.Current == p.Version && p.GvisorCapture == nil && len(changes) != 0 {
		return nil, errors.New("same-version baseline has generated drift; reconcile it explicitly first")
	}
	return changes, nil
}

func editReviews(body []byte, owners map[string]string, review []map[string]any) ([]byte, error) {
	var file struct {
		Records []map[string]any `json:"records"`
	}
	if err := json.Unmarshal(body, &file); err != nil {
		return nil, err
	}
	if len(review) != len(owners) {
		return nil, errors.New("review records must match changed version owners exactly")
	}
	replacements := map[string]any{}
	seen := map[string]bool{}
	for _, record := range review {
		component, _ := record["component"].(string)
		version, ok := owners[component]
		if !ok || seen[component] || record["pinned_version"] != version {
			return nil, errors.New("missing/duplicate/mismatched support review")
		}
		seen[component] = true
		var index int = -1
		for i, old := range file.Records {
			if old["component"] == component {
				if index != -1 {
					return nil, errors.New("duplicate existing review")
				}
				index = i
			}
		}
		if index < 0 {
			return nil, errors.New("unknown support consumer")
		}
		old := file.Records[index]
		if len(record) != len(old) {
			return nil, errors.New("support review schema changed")
		}
		for k := range old {
			if _, ok := record[k]; !ok {
				return nil, errors.New("support review field missing")
			}
		}
		if record["support_role"] != old["support_role"] || record["official_source"] != old["official_source"] {
			return nil, errors.New("review must preserve canonical role and official source")
		}
		for _, k := range []string{"checked_at", "latest_stable_at_check", "support_status", "decision", "reason"} {
			v, ok := record[k].(string)
			if !ok || strings.TrimSpace(v) == "" {
				return nil, errors.New("review field missing")
			}
		}
		checked, err := time.Parse("2006-01-02", record["checked_at"].(string))
		if err != nil || checked.After(time.Now().UTC().Add(24*time.Hour)) || time.Since(checked) > 90*24*time.Hour {
			return nil, errors.New("support review must be an actual current review")
		}
		replacements["/records/"+strconv.Itoa(index)] = record
	}
	return editJSON(body, replacements)
}

func editWorkflow(body []byte, current, next string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	lines := strings.Split(string(body), "\n")
	count := 0
	var walk func(*yaml.Node) error
	field := func(n *yaml.Node, key string) *yaml.Node {
		if n.Kind != yaml.MappingNode {
			return nil
		}
		for i := 0; i < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				return n.Content[i+1]
			}
		}
		return nil
	}
	walk = func(n *yaml.Node) error {
		if uses := field(n, "uses"); uses != nil && strings.HasPrefix(uses.Value, "actions/setup-go@") {
			if uses.Value != "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e" {
				return errors.New("unreviewed setup-go action")
			}
			with := field(n, "with")
			if with == nil || len(with.Content) != 6 {
				return errors.New("unknown setup-go selector")
			}
			v := field(with, "go-version")
			cache := field(with, "cache")
			latest := field(with, "check-latest")
			if v == nil || v.Value != current || v.Style != yaml.SingleQuotedStyle || cache == nil || cache.Value != "false" || latest == nil || latest.Value != "false" {
				return errors.New("partial/floating setup-go baseline")
			}
			line := lines[v.Line-1]
			start := v.Column - 1
			old := "'" + current + "'"
			if start+len(old) > len(line) || line[start:start+len(old)] != old {
				return errors.New("setup-go source span mismatch")
			}
			lines[v.Line-1] = line[:start] + "'" + next + "'" + line[start+len(old):]
			count++
		}
		for _, child := range n.Content {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(&doc); err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("missing setup-go consumer")
	}
	return []byte(strings.Join(lines, "\n")), nil
}
