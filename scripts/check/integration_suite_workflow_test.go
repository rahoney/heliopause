package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// These tests mutate the reviewed command sequence, which guards only select from.
func workflowWithoutSuiteGuards(contents string) string {
	lines := strings.Split(contents, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if !isIntegrationSuiteSelection(strings.TrimSpace(line)) {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func integrationSuiteWorkflow(t *testing.T) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestIntegrationSuitesCoverEveryCommandOnce(t *testing.T) {
	original := integrationSuiteWorkflow(t)
	if findings := validateIntegrationSuites(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	const pnpmOpen = "          if haa_ci_suite pnpm; then\n"
	const client = `          haa_install_ci_client "$RUNNER_TEMP/helox-sandbox-test"` + "\n"
	const npmNative = "          bash scripts/ci-npm-project.sh native\n"
	const refusalRemove = "          rm -- \"$node_refusal_log\"\n          fi # suite\n"
	first := func(old, updated string) func(string) string {
		return func(s string) string { return strings.Replace(s, old, updated, 1) }
	}
	for _, tc := range []struct {
		name, want string
		mutate     func(string) string
	}{
		{"fail fast", "fail-fast false", first("      fail-fast: false\n", "      fail-fast: true\n")},
		{"parallel limit", "no other strategy", first("      fail-fast: false\n", "      fail-fast: false\n      max-parallel: 1\n")},
		{"include", "no other strategy", first("        suite:\n", "        include:\n          - suite: bun\n        suite:\n")},
		{"dropped suite", "exactly the reviewed", first("          - bun\n", "")},
		{"raised limit", "must stay 120", first("    timeout-minutes: 120\n", "    timeout-minutes: 240\n")},
		{"unbound suite", "must bind", first("      HELOX_CI_SUITE: ${{ matrix.suite }}\n", "")},
		{"step override", "must not be overridden", first("      - name: Qualify production advisory quarantine\n", "      - name: Qualify production advisory quarantine\n        env:\n          HELOX_CI_SUITE: bun\n")},
		{"unknown suite", "unreviewed suite", first(pnpmOpen, "          if haa_ci_suite other; then\n")},
		{"two suites", "exact guard", first(pnpmOpen, "          if haa_ci_suite pnpm bun; then\n")},
		{"nested", "must not nest", first(npmNative, "          if haa_ci_suite uv; then\n"+npmNative)},
		{"stray close", "closes without opening", first(npmNative, npmNative+"          fi # suite\n          fi # suite\n")},
		{"unguarded test", "must belong to one suite", first("          if haa_ci_suite requirements; then\n          bash scripts/ci-uv-tools.sh python-requirements\n          fi # suite\n", "          bash scripts/ci-uv-tools.sh python-requirements\n")},
		{"guarded client", "outside suite guards", first(npmNative, npmNative+client)},
		{"split log consumer", "share one suite", first(refusalRemove, "          fi # suite\n          if haa_ci_suite bun; then\n          rm -- \"$node_refusal_log\"\n          fi # suite\n")},
		{"missing cuda exit", "leave only the exec-py leg", first("          "+integrationSuiteCUDAExit+"\n", "")},
		{"moved cuda exit", "directly follow", first("          "+integrationSuiteCUDAExit+"\n", "          true\n          "+integrationSuiteCUDAExit+"\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := tc.mutate(original)
			if candidate == original {
				t.Fatal("fixture unchanged")
			}
			findings := validateIntegrationSuites(candidate)
			if !strings.Contains(strings.Join(findings, "\n"), tc.want) {
				t.Fatalf("want %q, got %q", tc.want, findings)
			}
		})
	}
}

func TestIntegrationSuiteRunBodiesParse(t *testing.T) {
	doc, err := parseWorkflow(integrationSuiteWorkflow(t))
	if err != nil {
		t.Fatal(err)
	}
	expression := regexp.MustCompile(`\$\{\{.*?\}\}`)
	steps, _ := workflowMap(workflowMap(doc["jobs"])["gvisor-integration"])["steps"].([]any)
	checked := 0
	for _, raw := range steps {
		run, _ := workflowMap(raw)["run"].(string)
		if run == "" {
			continue
		}
		checked++
		command := exec.Command("bash", "-n")
		command.Stdin = strings.NewReader(expression.ReplaceAllString(run, "true"))
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v: %s", workflowMap(raw)["name"], err, out)
		}
	}
	if checked != 5 {
		t.Fatal("unexpected integration step count", checked)
	}
}

func TestIntegrationSuiteSelectionMatchesOnlyTheLeg(t *testing.T) {
	const program = `eval "$(grep '^haa_ci_suite()' ../ci-integration-cleanup.sh)"
HELOX_CI_SUITE=uv
haa_ci_suite uv
! haa_ci_suite bun
unset HELOX_CI_SUITE
! haa_ci_suite uv
`
	if out, err := exec.Command("bash", "-euc", program).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
}
