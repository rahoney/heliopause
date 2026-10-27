package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipxCIConcreteSupplyAndDispatch(t *testing.T) {
	path := filepath.Join("..", "ci-pipx-runtime.sh")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(body)
	if findings := validatePipxRuntimeScript(baseline); len(findings) != 0 {
		t.Fatal(findings)
	}
	for name, change := range map[string]func(string) string{
		"missing-fetch": func(s string) string {
			return strings.Replace(s, "go run ./scripts/update-runtime fetch-pipx-supply", "echo skipped", 1)
		},
		"candidate-materializer": func(s string) string {
			return strings.Replace(s, "materialize-pipx --root", "materialize-pipx-candidate --root", 1)
		},
		"missing-inspect-env": func(s string) string { return strings.Replace(s, "HELOX_PIPX_INSPECTION_INTEGRATION=1 ", "", 1) },
		"commented-inspect": func(s string) string {
			return strings.Replace(s, pipxRequiredInspectionCommand, "# "+pipxRequiredInspectionCommand, 1)
		},
		"echoed-graph": func(s string) string { return strings.Replace(s, pipxCurrentGraphCommand, "echo skipped", 1) },
		"ignored-graph": func(s string) string {
			return strings.Replace(s, pipxCurrentGraphCommand, pipxCurrentGraphCommand+" || true", 1)
		},
		"duplicate-inspect": func(s string) string {
			return strings.Replace(s, pipxRequiredInspectionCommand, pipxRequiredInspectionCommand+"\n"+pipxRequiredInspectionCommand, 1)
		},
		"wrong-cwd": func(s string) string {
			return strings.Replace(s, "pipx_ci_run internal/artifact/githubrelease", "pipx_ci_run internal/bootstrap", 1)
		},
		"wrong-filter": func(s string) string {
			return strings.Replace(s, "-test.run '^TestPipxAcquiredRuntimeRequiredInspectionIntegration$'", "-test.run '^Missing$'", 1)
		},
		"missing-root-namespace": func(s string) string {
			return strings.Replace(s, "unshare --mount --net --fork", "unshare --mount --fork", 1)
		},
		"missing-pending":       func(s string) string { return strings.Replace(s, pipxPendingNamespaceCommand, "echo skipped", 1) },
		"missing-skip-guard":    func(s string) string { return strings.Replace(s, "/--- SKIP:/ { bad=1 }", "", 1) },
		"substring-parent-pass": func(s string) string { return strings.Replace(s, "if(seen[name]!=1)", "if(seen[name]>1)", 1) },
		"lost-primary": func(s string) string {
			return strings.Replace(s, "if ((primary == 0)); then primary=$cleanup_status; fi", "primary=$cleanup_status", 1)
		},
		"no-cleanup": func(s string) string { return strings.Replace(s, "trap pipx_cleanup EXIT", "", 1) },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := change(baseline)
			if candidate == baseline {
				t.Fatal("fixture unchanged")
			}
			if len(validatePipxRuntimeScript(candidate)) == 0 {
				t.Fatal("invalid concrete pipx binding accepted")
			}
		})
	}
	if out, err := exec.Command("bash", "-n", path).CombinedOutput(); err != nil {
		t.Fatal(string(out), err)
	}
	for _, args := range [][]string{nil, {"unexpected"}, {"inspect", "extra"}} {
		command := exec.Command("bash", append([]string{path}, args...)...)
		if err := command.Run(); err == nil {
			t.Fatal("invalid mode reached supply", args)
		}
	}
}

func TestPipxCIClientLifetimeOrder(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(body)
	if findings := validatePipxInspectionClientOrder(baseline); len(findings) != 0 {
		t.Fatal(findings)
	}
	graph := "          bash scripts/ci-pipx-runtime.sh graph\n"
	for name, change := range map[string]func(string) string{
		"omitted-inspect": func(s string) string {
			return strings.Replace(s, "          bash scripts/ci-pipx-runtime.sh inspect\n", "", 1)
		},
		"commented-graph": func(s string) string {
			return strings.Replace(s, graph, "          # bash scripts/ci-pipx-runtime.sh graph\n", 1)
		},
		"echoed-namespace": func(s string) string {
			return strings.Replace(s, "bash scripts/ci-pipx-runtime.sh namespace", "echo 'bash scripts/ci-pipx-runtime.sh namespace'", 1)
		},
		"duplicate-graph": func(s string) string { return strings.Replace(s, graph, graph+graph, 1) },
		"wrong-package": func(s string) string {
			return strings.Replace(s, "helox-pipx-graph-test\" ./internal/artifact/githubrelease", "helox-pipx-graph-test\" ./internal/bootstrap", 1)
		},
		"wrong-client": func(s string) string {
			return strings.Replace(s, "haa_install_ci_client \"$RUNNER_TEMP/helox-pipx-graph-test\"", "haa_install_ci_client \"$RUNNER_TEMP/helox-sandbox-test\"", 1)
		},
		"missing-owning-helper": func(s string) string {
			return strings.Replace(s, "          haa_start_policy_helper\n"+graph, graph, 1)
		},
		"conditional-step": func(s string) string {
			return strings.Replace(s, "- name: Install pinned runtime and run lifecycle integration", "- name: Install pinned runtime and run lifecycle integration\n        if: false", 1)
		},
		"missing-production-restore": func(s string) string {
			return strings.Replace(s, "          haa_install_ci_client \"$RUNNER_TEMP/helox\"\n", "", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := change(baseline)
			if candidate == baseline {
				t.Fatal("fixture unchanged")
			}
			if len(validatePipxInspectionClientOrder(candidate)) == 0 {
				t.Fatal("wrong client/lifecycle accepted")
			}
		})
	}
}
