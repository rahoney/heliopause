package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Only versioned public metadata is needed here; real wheel fetch/materialization
// is an explicit actual check, never a hidden dependency of the default suite.
func TestPipxFetchPlanUsesOnlyFrozenSelectedInputs(t *testing.T) {
	root, _ := fixture(t)
	plan, err := planPipxFetch(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Documents) != 31 || len(plan.Wheels) != 11 {
		t.Fatal("input census differs")
	}
	for _, wheel := range plan.Wheels {
		if !strings.HasPrefix(wheel.Archive.URL, "https://files.pythonhosted.org/packages/") || wheel.Archive.Size <= 0 || wheel.Archive.Size > 8<<20 {
			t.Fatal("unbounded/nonofficial wheel")
		}
	}
}

func TestPipxFetchRejectsMetadataBeforeCreatingOutput(t *testing.T) {
	root, _ := fixture(t)
	document := filepath.Join(root, "tools/pipx/supply/pipx/metadata.json")
	body, err := os.ReadFile(document)
	if err != nil {
		t.Fatal(err)
	}
	writeTest(t, document, append(body, ' '))
	output := filepath.Join(t.TempDir(), "inputs")
	var stdout bytes.Buffer
	if err := run([]string{"fetch-pipx-supply", "--root", root, "--output", output}, &stdout); err == nil || stdout.Len() != 0 {
		t.Fatal("changed metadata accepted")
	}
	if _, err := os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("metadata failure created output")
	}
}

func TestPipxFetchRejectsChangedFrozenWheelSelection(t *testing.T) {
	root, _ := fixture(t)
	document := filepath.Join(root, "tools/pipx/supply/pipx/metadata.json")
	original, err := os.ReadFile(document)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(root, runtimeFile)
	originalLock, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"host", "scheme", "userinfo", "port", "query", "fragment", "encoded", "traversal", "filename", "hash", "empty-size", "oversize", "sdist", "yanked", "duplicate", "missing"} {
		t.Run(name, func(t *testing.T) {
			var metadata map[string]any
			if err := json.Unmarshal(original, &metadata); err != nil {
				t.Fatal(err)
			}
			rows := metadata["urls"].([]any)
			var selected map[string]any
			for _, raw := range rows {
				row := raw.(map[string]any)
				if row["filename"] == "pipx-1.18.4-py3-none-any.whl" {
					selected = row
				}
			}
			if selected == nil {
				t.Fatal("selected frozen metadata missing")
			}
			address := selected["url"].(string)
			switch name {
			case "host":
				selected["url"] = strings.Replace(address, "files.pythonhosted.org", "example.invalid", 1)
			case "scheme":
				selected["url"] = strings.Replace(address, "https:", "http:", 1)
			case "userinfo":
				selected["url"] = strings.Replace(address, "https://", "https://user@", 1)
			case "port":
				selected["url"] = strings.Replace(address, ".org/", ".org:443/", 1)
			case "query":
				selected["url"] = address + "?token=fixture"
			case "fragment":
				selected["url"] = address + "#fragment"
			case "encoded":
				selected["url"] = strings.Replace(address, "/packages/", "/packages/%2e/", 1)
			case "traversal":
				selected["url"] = strings.Replace(address, "/packages/", "/packages/../", 1)
			case "filename":
				selected["url"] = address + ".other"
			case "hash":
				selected["digests"].(map[string]any)["sha256"] = strings.Repeat("0", 64)
			case "empty-size":
				selected["size"] = 0
			case "oversize":
				selected["size"] = (8 << 20) + 1
			case "sdist":
				selected["packagetype"] = "sdist"
			case "yanked":
				selected["yanked"] = true
			case "duplicate":
				metadata["urls"] = append(rows, selected)
			case "missing":
				selected["filename"] = "absent.whl"
			}
			changed, err := json.Marshal(metadata)
			if err != nil {
				t.Fatal(err)
			}
			// Bind this deliberate synthetic metadata mutation to the test owner so the
			// semantic URL/selection guard, rather than the preceding hash guard, runs.
			var lock map[string]any
			if err := json.Unmarshal(originalLock, &lock); err != nil {
				t.Fatal(err)
			}
			for _, raw := range lock["pipx_tool"].(map[string]any)["inputs"].([]any) {
				row := raw.(map[string]any)
				if row["path"] == "pipx/metadata.json" {
					row["sha256"] = hash(changed)
				}
			}
			updated, err := json.Marshal(lock)
			if err != nil {
				t.Fatal(err)
			}
			writeTest(t, document, changed)
			writeTest(t, lockPath, updated)
			if _, err := planPipxFetch(root); err == nil {
				t.Fatal("invalid frozen selection accepted")
			}
		})
	}
}

func TestPipxFetchFailureNeverClaimsCompleteInputs(t *testing.T) {
	root, _ := fixture(t)
	plan, err := planPipxFetch(root)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "inputs")
	calls := 0
	injected := errors.New("injected selected download failure")
	err = executePipxFetch(plan, output, func(string, archive) error { calls++; return injected })
	if !errors.Is(err, injected) || calls != 1 {
		t.Fatal("download failure not preserved", err, calls)
	}
	if err := executePipxFetch(plan, output, func(string, archive) error { return nil }); err == nil {
		t.Fatal("partial directory adopted")
	}
	if err := executePipxFetch(plan, filepath.Join(t.TempDir(), "inputs"), func(string, archive) error { return nil }); err == nil {
		t.Fatal("absent downloads passed final readback")
	}
}
