package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// Explicitly wired by canonical security. Default package tests do not acquire
// an external tool or depend on a developer's private fixture directory.
func TestGitleaksDetectionContract(t *testing.T) {
	if os.Getenv("HELOX_GITLEAKS_CONTRACT") != "1" {
		t.Skip("canonical security supplies the managed tool cache")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	c, err := newChecker(root, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	tool, err := c.tool("gitleaks")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.verifyTool(tool); err != nil {
		t.Fatal(err)
	}
	current := c.toolExecutable(tool)
	baseline := os.Getenv("HELOX_GITLEAKS_BASELINE")
	if baseline != "" {
		if !filepath.IsAbs(baseline) {
			t.Fatal("baseline must be an explicit absolute control executable")
		}
		output, err := c.runCommandWithTimeout("Gitleaks baseline version", 10*time.Second, c.offlineEnvironment(), baseline, "version")
		if err != nil || strings.TrimSpace(output) != tool.ExpectedVersion {
			t.Fatal("baseline control version mismatch")
		}
	}
	// Reconstructed from the documented M7 case classes, not recovered original
	// raw tokens: canonical GitHub PAT, synthetic generic credential and benign data.
	// No fixture is an issued credential or used for authentication.
	alphabet := "aB7qR2vX9mN4zK8pD5sW1cF6yH3jL0uTgE5Q"
	pat := "gh" + "p_" + alphabet
	api := alphabet + "2J9d4V"
	for _, test := range []struct {
		name, body string
		rule       string
	}{
		{"github PAT", "GITHUB_TOKEN=" + pat + "\n", "github-pat"},
		{"generic credential", "api_key = \"" + api + "\"\n", "generic-api-key"},
		{"benign release metadata", "version = \"1.26.9\"\nrepository = \"https://github.com/rahoney/heliopause\"\n", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			inputs := filepath.Join(dir, "inputs")
			if err := os.Mkdir(inputs, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(inputs, "credentials.env"), []byte(test.body), 0o600); err != nil {
				t.Fatal(err)
			}
			actual := scanGitleaksContract(t, c, current, inputs, filepath.Join(dir, "current.json"))
			found := false
			for _, f := range actual {
				if f.Rule == test.rule {
					found = true
				}
			}
			if test.rule == "" {
				if len(actual) != 0 {
					t.Fatal("benign input reported a finding")
				}
			} else if !found {
				t.Fatal("required synthetic detection missing:", test.rule)
			}
			if baseline != "" {
				control := scanGitleaksContract(t, c, baseline, inputs, filepath.Join(dir, "baseline.json"))
				if !reflect.DeepEqual(actual, control) {
					t.Fatal("source build changed normalized findings relative to official archive")
				}
			}
		})
	}
}

type contractFinding struct {
	Rule                                       string
	File                                       string
	StartLine, EndLine, StartColumn, EndColumn int
}

func scanGitleaksContract(t *testing.T, c *checker, executable, inputs, report string) []contractFinding {
	t.Helper()
	_, err := c.runCommandWithTimeout("Gitleaks synthetic contract", 30*time.Second, c.offlineEnvironment(), executable, "detect", "--no-git", "--no-banner", "--redact=100", "--source", inputs, "--report-format=json", "--report-path", report)
	var exitError *exec.ExitError
	if err != nil && (!errors.As(err, &exitError) || exitError.ExitCode() != 1) {
		t.Fatal("scanner execution failed; not a clean result")
	}
	body, readErr := os.ReadFile(report)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var findings []struct {
		RuleID, File                               string
		StartLine, EndLine, StartColumn, EndColumn int
	}
	if err := json.Unmarshal(body, &findings); err != nil {
		t.Fatal("scanner report malformed")
	}
	if (err == nil) != (len(findings) == 0) {
		t.Fatal("scanner exit/report disagreement")
	}
	result := []contractFinding{}
	for _, f := range findings {
		result = append(result, contractFinding{f.RuleID, filepath.Base(f.File), f.StartLine, f.EndLine, f.StartColumn, f.EndColumn})
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		if a.StartLine != b.StartLine {
			return a.StartLine < b.StartLine
		}
		return a.StartColumn < b.StartColumn
	})
	return result
}
