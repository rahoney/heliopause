package main

import "strings"

const pipxRequiredInspectionCommand = "pipx_ci_run internal/bootstrap TestPipxAcquiredRuntimeRequiredInspectionIntegration inspect env HELOX_PIPX_INSPECTION_INTEGRATION=1 HELOX_PIPX_RUNTIME_ZIP=\"$pipx_work/pipx-runtime-linux-amd64.zip\" /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=6m -test.run '^TestPipxAcquiredRuntimeRequiredInspectionIntegration$'"
const pipxCurrentGraphCommand = "pipx_ci_run internal/artifact/githubrelease TestPipxActualCompleteGraphCurrentEvidencePolicyAndStaging graph env HELOX_PIPX_GRAPH_INTEGRATION=1 HELOX_PIPX_RELEASE_ACTUAL_DIR=\"$pipx_work\" /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=8m -test.run '^TestPipxActualCompleteGraphCurrentEvidencePolicyAndStaging$'"
const pipxBootstrapNamespaceCommand = "pipx_ci_run internal/releaseactivation TestPipxBootstrapCopyRecoveryAndCleanupInDisposableNamespace,TestInitialNativeFilesInDisposableNamespace,TestProtectedStageInDisposableNamespace bootstrap sudo -n env HELOX_PIPX_ROOT_DATA_NAMESPACE=1 HELOX_BOOTSTRAP_NAMESPACE=1 HELOX_STAGE_NAMESPACE=1 HELOX_PIPX_RELEASE_ACTUAL_DIR=\"$pipx_work\" unshare --mount --net --fork \"$RUNNER_TEMP/helox-releaseactivation-test\" -test.v -test.failfast -test.timeout=3m -test.run '^(TestPipxBootstrapCopyRecoveryAndCleanupInDisposableNamespace|TestInitialNativeFilesInDisposableNamespace|TestProtectedStageInDisposableNamespace)$'"
const pipxPendingNamespaceCommand = "pipx_ci_run internal/releaseinstall TestInterruptedSystemCopyInDisposableNamespace pending sudo -n env HELOX_COPY_INTERRUPTION_NAMESPACE=1 unshare --mount --net --fork \"$RUNNER_TEMP/helox-releaseinstall-test\" -test.v -test.failfast -test.timeout=3m -test.run '^TestInterruptedSystemCopyInDisposableNamespace$'"
const pipxGraphSBOMFixtureCommand = "cp -- \"$pipx_root/docs/planning/evidence/m13-003-release-build/v0.1.0/helox-release-sbom.cdx.json\" \"$pipx_work/helox-release-sbom.cdx.json\""

// Concrete fixed CI convention only; executed dispatch and cleanup are separate.
func validatePipxRuntimeScript(contents string) []string {
	counts := map[string]int{}
	var active []string
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		counts[line]++
		if line != "" && !strings.HasPrefix(line, "#") {
			active = append(active, line)
		}
	}
	var findings []string
	for _, command := range []string{
		"set -euo pipefail",
		"case \"$#:${1-}\" in",
		"1:inspect|1:graph|1:namespace) ;;",
		"*) echo 'unsupported pipx CI profile' >&2; exit 2 ;;",
		"pipx_profile=$1",
		"test \"$(id -u)\" != 0",
		"test -n \"${RUNNER_TEMP:-}\"",
		"test \"${RUNNER_TEMP:0:1}\" = /",
		"pipx_root=$PWD",
		"pipx_work=$(mktemp -d \"$RUNNER_TEMP/haa-pipx.XXXXXXXX\")",
		"trap pipx_cleanup EXIT",
		"rm -rf -- \"$pipx_work\" || cleanup_status=1",
		"if ((primary == 0)); then primary=$cleanup_status; fi",
		"exit \"$primary\"",
		"umask 077",
		"go run ./scripts/generate-runtime-lock.go -check",
		"go run ./scripts/update-runtime fetch-pipx-supply --root \"$pipx_root\" --output \"$pipx_work/supply\"",
		"go run ./scripts/update-runtime materialize-pipx --root \"$pipx_root\" --input \"$pipx_work/supply\" --output \"$pipx_work/pipx-runtime-linux-amd64.zip\" --notices-output \"$pipx_work/pipx-NOTICE.txt\"",
		"local package=$1 expected=$2 label=$3 status=0",
		"cd \"$package\"",
		"\"$@\"",
		") >\"$log\" 2>&1 || status=$?",
		"if ((status)); then return \"$status\"; fi",
		"/^--- PASS: / { if($3 in expected) seen[$3]++ }",
		"/--- SKIP:/ { bad=1 }",
		"END { if(bad) exit 1; for(name in expected) if(seen[name]!=1) exit 1 }",
		pipxRequiredInspectionCommand, pipxCurrentGraphCommand, pipxBootstrapNamespaceCommand, pipxPendingNamespaceCommand, pipxGraphSBOMFixtureCommand,
	} {
		if counts[command] != 1 {
			findings = append(findings, "pipx CI requires one exact active command: "+command)
		}
	}
	dispatch := strings.Join([]string{"case \"$pipx_profile\" in", "inspect)", pipxRequiredInspectionCommand, ";;", "graph)", pipxGraphSBOMFixtureCommand, pipxCurrentGraphCommand, ";;", "namespace)", pipxBootstrapNamespaceCommand, pipxPendingNamespaceCommand, ";;", "esac"}, "\n")
	if strings.Count(strings.Join(active, "\n"), dispatch) != 1 {
		findings = append(findings, "pipx CI requires exact exclusive mode/cwd/consumer dispatch")
	}
	return findings
}

func validatePipxInspectionClientOrder(contents string) []string {
	blocks, err := activeJobRunLines(contents, "gvisor-integration", "Install pinned runtime and run lifecycle integration")
	if err != nil {
		return nil
	}
	expected := []string{
		"HELOX_PYPI_DYNAMIC_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestLinuxPyPI(WheelDynamic|SdistBuild)Integration$'",
		"haa_install_ci_client \"$RUNNER_TEMP/helox-pnpm-project-test\"",
		"haa_start_policy_helper",
		"bash scripts/ci-pipx-runtime.sh inspect",
		"go test -c -tags=realcorpus -o \"$RUNNER_TEMP/helox-pipx-graph-test\" ./internal/artifact/githubrelease",
		"haa_install_ci_client \"$RUNNER_TEMP/helox-pipx-graph-test\"",
		"haa_start_policy_helper",
		"bash scripts/ci-pipx-runtime.sh graph",
		"go test -c -tags=realcorpus -o \"$RUNNER_TEMP/helox-releaseactivation-test\" ./internal/releaseactivation",
		"go test -c -tags=realcorpus -o \"$RUNNER_TEMP/helox-releaseinstall-test\" ./internal/releaseinstall",
		"bash scripts/ci-pipx-runtime.sh namespace",
		"haa_install_ci_client \"$RUNNER_TEMP/helox\"",
		"haa_start_policy_helper",
	}
	counts := map[string]int{}
	matched := 0
	for _, lines := range blocks {
		var active []string
		for _, line := range lines {
			counts[line]++
			if line != "" && !strings.HasPrefix(line, "#") {
				active = append(active, line)
			}
		}
		matched += strings.Count(strings.Join(active, "\n"), strings.Join(expected, "\n"))
	}
	var findings []string
	if matched != 1 {
		findings = append(findings, "pipx actual requires current tagged clients, owning helper, namespaces and production restore in order")
	}
	for _, command := range []string{"bash scripts/ci-pipx-runtime.sh inspect", "bash scripts/ci-pipx-runtime.sh graph", "bash scripts/ci-pipx-runtime.sh namespace", expected[4], expected[8], expected[9]} {
		if counts[command] != 1 {
			findings = append(findings, "pipx actual requires one lifecycle command: "+command)
		}
	}
	return findings
}
