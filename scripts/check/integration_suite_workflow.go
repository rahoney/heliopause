package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// integrationSuites is owned by the checker, independently of the workflow.
var integrationSuites = []string{"npm-core", "pnpm", "bun", "uv", "requirements", "pipx", "go-cargo-tf", "exec-py"}

const (
	integrationSuiteJobMinutes = 120
	integrationSuiteClose      = "fi # suite"
	integrationSuiteCUDAExit   = `if [ "$HELOX_CI_CUDA_ONLY" = true ] && [ "$HELOX_CI_SUITE" != exec-py ]; then echo 'qualification_scope=exec-py-leg'; exit 0; fi`
	integrationSuiteMatrixEnv  = "${{ matrix.suite }}"
	integrationSuiteLifecycle  = "Install pinned runtime and run lifecycle integration"
)

var (
	integrationSuiteOpen = regexp.MustCompile(`^if haa_ci_suite ([a-z][a-z-]*); then$`)
	integrationSuiteLog  = regexp.MustCompile(`\$[a-z_]+_log\b|\$RUNNER_TEMP/[A-Za-z0-9._-]+\.log\b`)
)

// Suite selection is not a command: removing exactly these lines lets every
// other validator keep checking the same commands in the same order.
func isIntegrationSuiteSelection(line string) bool {
	return integrationSuiteOpen.MatchString(line) || line == integrationSuiteClose || line == integrationSuiteCUDAExit
}

func withoutIntegrationSuiteGuards(lines []string) []string {
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if !isIntegrationSuiteSelection(line) {
			kept = append(kept, line)
		}
	}
	return kept
}

// Client binding and rebuilt inputs stay shared so that every leg keeps the
// ordering the removed-guard validators assert.
func isIntegrationSuiteCommonLine(line string) bool {
	for _, prefix := range []string{"source ", "haa_ci_initialize", "haa_install_ci_client", "haa_start_policy_helper", "go test -c", "cmp "} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func isIntegrationSuiteTestLine(line string) bool {
	if strings.HasPrefix(line, "bash scripts/ci-") || strings.HasPrefix(line, "/usr/libexec/heliopause/helox ") {
		return true
	}
	return strings.Contains(line, "-test.run") || strings.Contains(line, "go test -v") || strings.Contains(line, "go test -run")
}

func validateIntegrationSuites(contents string) []string {
	doc, err := parseWorkflow(contents)
	if err != nil {
		return nil
	}
	job := workflowMap(workflowMap(doc["jobs"])["gvisor-integration"])
	var findings []string
	strategy, matrix := workflowMap(job["strategy"]), workflowMap(workflowMap(job["strategy"])["matrix"])
	if strategy["fail-fast"] != false || len(strategy) != 2 || len(matrix) != 1 {
		findings = append(findings, "integration matrix requires fail-fast false and no other strategy controls")
	}
	suites, _ := matrix["suite"].([]any)
	names := make([]string, 0, len(suites))
	for _, suite := range suites {
		name, _ := suite.(string)
		names = append(names, name)
	}
	if !reflect.DeepEqual(names, integrationSuites) {
		findings = append(findings, "integration matrix must list exactly the reviewed suites")
	}
	if job["timeout-minutes"] != integrationSuiteJobMinutes {
		findings = append(findings, fmt.Sprintf("integration leg limit must stay %d minutes", integrationSuiteJobMinutes))
	}
	if workflowMap(job["env"])["HELOX_CI_SUITE"] != integrationSuiteMatrixEnv {
		findings = append(findings, "integration job must bind HELOX_CI_SUITE to the matrix suite")
	}
	used := map[string]int{}
	steps, _ := job["steps"].([]any)
	for _, raw := range steps {
		step := workflowMap(raw)
		name, _ := step["name"].(string)
		if workflowMap(step["env"])["HELOX_CI_SUITE"] != nil {
			findings = append(findings, "integration suite must not be overridden by a step: "+name)
		}
		if run, _ := step["run"].(string); run != "" {
			findings = append(findings, validateIntegrationSuiteRun(name, run, used)...)
		}
	}
	for _, suite := range integrationSuites {
		if used[suite] == 0 {
			findings = append(findings, "integration suite runs no tests: "+suite)
		}
		delete(used, suite)
	}
	for suite := range used {
		findings = append(findings, "integration guard names an unreviewed suite: "+suite)
	}
	return findings
}

func validateIntegrationSuiteRun(step, run string, used map[string]int) []string {
	var findings []string
	active, logs := "", map[string]string{}
	exits := 0
	for index, raw := range strings.Split(run, "\n") {
		line := strings.TrimSpace(raw)
		if open := integrationSuiteOpen.FindStringSubmatch(line); open != nil {
			if active != "" {
				findings = append(findings, "integration suite guards must not nest: "+step)
			}
			active = open[1]
			used[active]++
			continue
		}
		if line == integrationSuiteClose {
			if active == "" {
				findings = append(findings, "integration suite guard closes without opening: "+step)
			}
			active = ""
			continue
		}
		if line == integrationSuiteCUDAExit {
			exits++
			if step != integrationSuiteLifecycle || index != 1 {
				findings = append(findings, "CUDA leg selection must directly follow the first lifecycle shell option")
			}
			continue
		}
		if strings.Contains(line, "haa_ci_suite") || strings.Contains(line, "HELOX_CI_SUITE") || strings.HasPrefix(line, "fi #") {
			findings = append(findings, "integration suite selection must use the exact guard: "+step)
		}
		if active != "" && isIntegrationSuiteCommonLine(line) {
			findings = append(findings, "shared client and build lines must stay outside suite guards: "+line)
		}
		if active == "" && isIntegrationSuiteTestLine(line) {
			findings = append(findings, "test command must belong to one suite: "+line)
		}
		for _, name := range integrationSuiteLog.FindAllString(line, -1) {
			if owner, seen := logs[name]; seen && owner != active {
				findings = append(findings, "log producer and consumer must share one suite: "+name)
			}
			logs[name] = active
		}
	}
	if active != "" {
		findings = append(findings, "integration suite guard is not closed: "+step)
	}
	if step == integrationSuiteLifecycle && exits != 1 {
		findings = append(findings, "manual CUDA runs must leave only the exec-py leg active")
	}
	return findings
}
