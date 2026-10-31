package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPublicMetadataDispositionRequiresExactOriginalAndBothFindings(t *testing.T) {
	original, err := os.ReadFile(filepath.Join("../..", publicPygmentsMetadata))
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"valid", "changed", "missing", "symlink", "path", "rule", "column", "end", "line", "commit", "duplicate", "incomplete", "empty"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			name := filepath.Join(root, publicPygmentsMetadata)
			writeTestFile(t, name, string(original))
			c := &checker{root: root, stdout: io.Discard}
			findings := []publicMetadataFinding{{RuleID: "square-access-token", File: publicPygmentsMetadata, StartLine: 1, EndLine: 1, StartColumn: 3155, EndColumn: 3187}, {RuleID: "square-access-token", File: publicPygmentsMetadata, StartLine: 1, EndLine: 1, StartColumn: 3357, EndColumn: 3389}}
			switch mode {
			case "changed":
				writeTestFile(t, name, string(original)+" ")
			case "missing":
				if err := os.Remove(name); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				findings[0].SymlinkFile = "elsewhere"
			case "path":
				findings[0].File = "other/metadata.json"
			case "rule":
				findings[0].RuleID = "generic-api-key"
			case "column":
				findings[0].StartColumn++
			case "end":
				findings[0].EndColumn++
			case "line":
				findings[0].EndLine++
			case "commit":
				findings[0].Commit = "--all"
			case "duplicate":
				findings = append(findings, findings[0])
			case "incomplete":
				findings = findings[:1]
			case "empty":
				findings = nil
			}
			if err := c.verifyPublicMetadataFindings(findings); (err == nil) != (mode == "valid") {
				t.Fatal("unexpected public metadata disposition", err)
			}
		})
	}
}

// The real pinned scanner must still find changed/same-path credentials and
// copies at other paths. Run only inside the canonical managed-tool contract.
func testGitleaksPublicMetadataContract(t *testing.T, c *checker, executable string) {
	t.Helper()
	original, err := os.ReadFile(filepath.Join(c.root, publicPygmentsMetadata))
	if err != nil {
		t.Fatal(err)
	}
	token := "EA" + "AA" + strings.Repeat("Z", 28)
	for _, mode := range []string{"original", "changed-value", "same-path-extra-token", "other-path", "history-original", "history-extra-token"} {
		t.Run("public metadata "+mode, func(t *testing.T) {
			scan := *c
			scan.root = t.TempDir()
			scan.stdout = io.Discard
			name := filepath.Join(scan.root, publicPygmentsMetadata)
			body := string(original)
			if mode == "changed-value" {
				body = strings.ReplaceAll(body, "eaaa"+"46b6eef9ba4d6abd73d4f45be371", token)
			}
			if mode == "same-path-extra-token" {
				body += "\naccess_token = \"" + token + "\"\n"
			}
			if mode == "other-path" {
				name = filepath.Join(scan.root, "other.json")
			}
			writeTestFile(t, name, body)
			args := []string{"detect", "--no-git", "--no-banner", "--redact=100", "--source", "."}
			if strings.HasPrefix(mode, "history-") {
				git, err := exec.LookPath(executableName("git"))
				if err != nil {
					t.Fatal(err)
				}
				run := func(args ...string) {
					t.Helper()
					if _, err := scan.runCommandWithTimeout("public metadata history fixture", time.Minute, scan.offlineEnvironment(), git, args...); err != nil {
						t.Fatal(err)
					}
				}
				run("init", "--quiet")
				run("add", ".")
				run("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "commit", "--quiet", "-m", "public metadata fixture")
				if mode == "history-extra-token" {
					writeTestFile(t, filepath.Join(scan.root, "other.env"), "token = \""+token+"\"\n")
					run("add", ".")
					run("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "commit", "--quiet", "-m", "synthetic detection fixture")
				}
				args = []string{"detect", "--no-banner", "--redact=100", "--log-opts=--all", "--source", "."}
			}
			err := scan.runGitleaksWithPublicMetadata("public metadata scanner contract", executable, args...)
			wantSuccess := mode == "original" || mode == "history-original"
			if (err == nil) != wantSuccess {
				t.Fatal("scanner disposition differs", mode, err)
			}
		})
	}
}
