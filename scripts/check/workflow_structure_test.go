package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCIWorkflowStructureMutations(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	good := string(body)
	if findings := validateCIWorkflow(good); len(findings) != 0 {
		t.Fatalf("corrected workflow: %v", findings)
	}
	start := strings.Index(good, "  wheel-corpus:\n")
	end := strings.Index(good[start:], "  quick:\n") + start
	corpus := good[start:end]
	without := good[:start] + good[end:]
	// Reconstruct the exact demonstrated malformed insertion, including broken cu126 fields.
	malformed := strings.Replace(without, "        required: false\n", strings.Replace(corpus, "  wheel-corpus:", "        wheel-corpus:", 1)+"  required: false\n", 1)
	misplaced := strings.Replace(without, "    inputs:\n", "    inputs:\n"+indentWorkflow(corpus, 4), 1)
	fixtures := map[string]string{
		"actual malformed YAML":           malformed,
		"valid job shaped dispatch input": misplaced,
		"commented corpus":                strings.Replace(good, corpus, commentWorkflow(corpus), 1),
		"missing corpus":                  without,
		"duplicate corpus":                strings.Replace(good, corpus, corpus+corpus, 1),
		"wrong location":                  strings.Replace(without, "permissions:\n", "wheel-corpus:\n"+indentWorkflow(corpus, 2)+"permissions:\n", 1),
		"missing corpus needs":            strings.Replace(good, "      - wheel-corpus\n", "", 1),
		"unknown needs":                   strings.Replace(good, "      - wheel-corpus\n", "      - nonexistent\n", 1),
		"required ignored error":          strings.Replace(good, "  required:\n", "  required:\n    continue-on-error: true\n", 1),
		"required skipped step":           strings.Replace(good, "      - name: Aggregate required job results\n", "      - name: Aggregate required job results\n        if: false\n", 1),
		"missing binding":                 strings.Replace(good, "      CORPUS_RESULT: ${{ needs.wheel-corpus.result }}\n", "", 1),
		"substitute result":               strings.Replace(good, "CORPUS_RESULT: ${{ needs.wheel-corpus.result }}", "CORPUS_RESULT: ${{ needs.quick.result }}", 1),
		"substitute argument":             strings.Replace(good, `"$CORPUS_RESULT"`, `"$QUICK_RESULT"`, 1),
		"skippable corpus":                strings.Replace(good, "  wheel-corpus:\n", "  wheel-corpus:\n    if: false\n", 1),
		"wrong input default":             strings.Replace(good, "        default: false", "        default: true", 1),
	}
	// This fixture must be valid YAML to exercise Actions structure, not parser rejection.
	if _, err := parseWorkflow(misplaced); err != nil {
		t.Fatalf("misplaced fixture is not valid YAML: %v", err)
	}
	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			if len(validateCIWorkflow(fixture)) == 0 {
				t.Fatal("real policy accepted broken workflow")
			}
			// Exercise canonical on-disk entry point, not a separate test validator.
			dir := t.TempDir()
			target := filepath.Join(dir, workflowRelativePath)
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0755); err != nil {
				t.Fatal(err)
			}
			lock, err := os.ReadFile(filepath.Join(root, "scripts/runtimes.lock.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "scripts/runtimes.lock.json"), lock, 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, []byte(fixture), 0644); err != nil {
				t.Fatal(err)
			}
			if err := checkWorkflow(dir, workflowRelativePath, validateCIWorkflow); err == nil {
				t.Fatal("canonical checker accepted corrupted disposable workflow")
			}
		})
	}
}
func indentWorkflow(s string, n int) string {
	return strings.Repeat(" ", n) + strings.ReplaceAll(strings.TrimSuffix(s, "\n"), "\n", "\n"+strings.Repeat(" ", n)) + "\n"
}
func commentWorkflow(s string) string {
	return "# " + strings.ReplaceAll(strings.TrimSuffix(s, "\n"), "\n", "\n# ") + "\n"
}

func TestCorpusTargetRejectsEmptySelection(t *testing.T) {
	good := ""
	for _, name := range []string{"TestModelC_RealCorpusQualification", "TestModelC_MissingCorpusDirectoryFails", "TestModelC_CorpusHashMismatchFails"} {
		good += "=== RUN   " + name + "\n--- PASS: " + name + " (0.01s)\n"
	}
	if err := validateCorpusTestExecution(good); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{"ok package [no tests to run]", "PASS", strings.Replace(good, "--- PASS: TestModelC_RealCorpusQualification", "--- SKIP: TestModelC_RealCorpusQualification", 1)} {
		if validateCorpusTestExecution(output) == nil {
			t.Fatal("accepted absent/skipped required tests")
		}
	}
}

func TestWorkflowParserIsToolOnlyDependency(t *testing.T) {
	const module = "example.test/project"
	if findings := validateCurrentImports(module, []packageMetadata{{ImportPath: module + "/scripts/check", Imports: []string{"go.yaml.in/yaml/v3"}}}); len(findings) != 0 {
		t.Fatal(findings)
	}
	if findings := validateCurrentImports(module, []packageMetadata{{ImportPath: module + "/internal/artifact/pypi", Imports: []string{"go.yaml.in/yaml/v3"}}}); len(findings) == 0 {
		t.Fatal("parser permitted in product path")
	}
}
