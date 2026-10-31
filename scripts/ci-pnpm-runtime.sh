#!/bin/bash
# Hosted-CI-only native supply; the payload runs only inside gVisor as UID1000.
set -euo pipefail
test "$#" -le 1
pnpm_profile=${1:-sandbox}
case "$pnpm_profile" in
  sandbox|project|run) ;;
  *) echo 'unsupported PNPM CI profile' >&2; exit 2 ;;
esac
test "$(id -u)" != 0
test -n "${RUNNER_TEMP:-}"
test "${RUNNER_TEMP:0:1}" = /
test ! -e /usr/lib/heliopause
test ! -L /usr/lib/heliopause
pnpm_work=$(mktemp -d "$RUNNER_TEMP/haa-pnpm.XXXXXXXX")
pnpm_root_created=0
pnpm_cleanup() {
  local primary=$? cleanup_status=0
  trap - EXIT
  if ((pnpm_root_created)); then
    sudo -n rm -f -- /usr/lib/heliopause/current /usr/lib/heliopause/versions/v1.0.0-pnpm-ci/pnpm-linux-amd64 || cleanup_status=1
    sudo -n rmdir -- /usr/lib/heliopause/versions/v1.0.0-pnpm-ci /usr/lib/heliopause/versions /usr/lib/heliopause || cleanup_status=1
  fi
  rm -rf -- "$pnpm_work" || cleanup_status=1
  if ((cleanup_status)); then echo 'PNPM CI supply cleanup unproven' >&2; fi
  if ((primary == 0)); then primary=$cleanup_status; fi
  exit "$primary"
}
trap pnpm_cleanup EXIT
go run ./scripts/generate-runtime-lock.go -check
pnpm_url=$(jq -er '.pnpm_tool.archive.url' scripts/runtimes.lock.json)
pnpm_size=$(jq -er '.pnpm_tool.archive.size' scripts/runtimes.lock.json)
curl --fail --show-error --silent --location --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 120 --max-filesize "$pnpm_size" --output "$pnpm_work/archive.tar.gz" "$pnpm_url"
go run ./scripts/update-runtime materialize-pnpm --root "$PWD" --archive "$pnpm_work/archive.tar.gz" --output "$pnpm_work/pnpm-linux-amd64" --notices-output "$pnpm_work/pnpm-NOTICE.txt"
rm -- "$pnpm_work/archive.tar.gz"
# mkdir refuses an existing installation; cleanup removes only this CI layout.
sudo -n mkdir -m 0755 -- /usr/lib/heliopause
pnpm_root_created=1
sudo -n install -d -m 0755 /usr/lib/heliopause/versions/v1.0.0-pnpm-ci
sudo -n install -m 0555 "$pnpm_work/pnpm-linux-amd64" /usr/lib/heliopause/versions/v1.0.0-pnpm-ci/pnpm-linux-amd64
sudo -n ln -s versions/v1.0.0-pnpm-ci /usr/lib/heliopause/current
# The owning parent installed the matching tagged bootstrap client and started
# its authenticated helper. This child never replaces a client or owns rebind.
if [[ "$pnpm_profile" == run ]]; then
  pnpm_run_log="$pnpm_work/run.log"
  pnpm_run_status=0
  (
    cd internal/bootstrap
    HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_RUN_EXECUTION_WORKFLOW_INTEGRATION=1 HELOX_PNPM_RUN_EXECUTION_CLI_INTEGRATION=1 HELOX_PNPM_RUN_EXECUTION_CLI_REFUSAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=10m -test.run '^TestLinuxPNPMRunExecution(Workflow|CLI(Refusal)?)Integration$'
  ) >"$pnpm_run_log" 2>&1 || pnpm_run_status=$?
  pnpm_run_log_status=0
  cat -- "$pnpm_run_log" || pnpm_run_log_status=$?
  if ((pnpm_run_status)); then exit "$pnpm_run_status"; fi
  if ((pnpm_run_log_status)); then exit "$pnpm_run_log_status"; fi
  awk '
BEGIN {
  expected["TestLinuxPNPMRunExecutionWorkflowIntegration"]=1
  expected["TestLinuxPNPMRunExecutionCLIIntegration"]=1
  expected["TestLinuxPNPMRunExecutionCLIRefusalIntegration"]=1
  cli[1]="alias=dlx format=json language=en show_output=false exit_code=0"
  cli[2]="alias=pnx format=json language=ko show_output=true exit_code=1"
  cli[3]="alias=pnpx format=human language=ko show_output=true exit_code=0"
}
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if (!($3 in expected) || parents[$3]++) bad=1; parent_count++ }
/production_pnpm_run_application attempt=/ {
  if ($0 !~ /^[[:space:]]+node_execution_run_integration_test\.go:[0-9]+: production_pnpm_run_application attempt=[12] artifact_role=true target_loaded=true exit_code=[01] stdout_bytes=[1-9][0-9]* stderr_bytes=0 user_program_run=true$/) bad=1
  id=$3; sub(/^attempt=/,"",id)
  if (apps[id]++ || $6 != "exit_code=" (id-1)) bad=1
  app_count++
}
/production_pnpm_run_cli case=/ {
  id=$3; sub(/^case=/,"",id)
  if (!(id in cli) || cases[id]++ || $0 !~ ("^[[:space:]]+npm_execution_integration_test\\.go:[0-9]+: production_pnpm_run_cli case=" id " " cli[id] " current_policy=ALLOW guest_cleanup=CONFIRMED$")) bad=1
  case_count++
}
/production_pnpm_run_cli_graph / {
  if ($0 !~ /^[[:space:]]+npm_execution_integration_test\.go:[0-9]+: production_pnpm_run_cli_graph nodes=[1-9][0-9]* edges=[1-9][0-9]* primary=rimraf current_policy=ALLOW$/ || graph++) bad=1
  nodes=$3; sub(/^nodes=/,"",nodes); if (nodes<=1) bad=1
}
/production_pnpm_run_refusal case=/ {
  id=$3; sub(/^case=/,"",id)
  reason=(id==1 ? "ambiguous" : "missing")
  if (id !~ /^[12]$/ || refused[id]++ || $0 !~ ("^[[:space:]]+npm_execution_refusal_integration_test\\.go:[0-9]+: production_pnpm_run_refusal case=" id " reason=" reason " current_policy=ALLOW preparation=INCOMPLETE runtime_attempted=false exit_absent=true output_bytes=0 stored_manifest=0$")) bad=1
  refusal_count++
}
END { exit (bad || parent_count!=3 || app_count!=2 || case_count!=3 || refusal_count!=2 || graph!=1) }
' "$pnpm_run_log"
  exit 0
fi
if [[ "$pnpm_profile" == project ]]; then
  HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 HELOX_PNPM_CACHE_INTEGRATION=1 HELOX_PNPM_PROJECT_TRANSACTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMPublic(ApprovedCache|ProjectTransaction)Integration$'
  HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 HELOX_PNPM_PROJECT_CLI_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMProjectCLIIntegration$'
  HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 HELOX_PNPM_NATIVE_ORIGINAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMNativeOriginalIntegration$'
  exit 0
fi
HELOX_PNPM_NATIVE_RUNTIME=1 go test -tags=realcorpus -v -timeout=3m ./internal/sandbox -run '^TestPNPMActual(Isolated(NativeStartup|EmptyLock|NodeQuery)|ReleaseArtifactProbe)$'
# The parent lifecycle installed this realcorpus binary and owns the running
# authenticated policy helper. Execute that exact client as the ordinary user.
HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMActualProductionProjectInspection$'
# Direct compiled tests keep the caller cwd; this consumer reads tracked package fixtures.
(
  cd internal/sandbox
  HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_PUBLIC_METADATA_FIXTURE=1 HELOX_PNPM_INSTANCE_EXECUTION_FIXTURE=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMActualProductionInstanceExecution$/^controlled-(success|exit37|long-name)$'
)
HELOX_PNPM_PUBLIC_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run '^TestPNPMPublicNativeResolverIntegration$'
pnpm_cold_log="$pnpm_work/cold.log"
pnpm_cold_status=0
(
  cd internal/sandbox
  HELOX_PNPM_NATIVE_RUNTIME=1 HELOX_PNPM_TOOL_RUN_SELECTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=3m -test.run '^TestPNPMPublicColdToolRunSelectionIntegration$'
) >"$pnpm_cold_log" 2>&1 || pnpm_cold_status=$?
pnpm_cold_log_status=0
cat -- "$pnpm_cold_log" || pnpm_cold_log_status=$?
if ((pnpm_cold_status)); then exit "$pnpm_cold_status"; fi
if ((pnpm_cold_log_status)); then exit "$pnpm_cold_log_status"; fi
awk '
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if ($3 != "TestPNPMPublicColdToolRunSelectionIntegration" || parents++) bad=1 }
/pnpm_cold_run / {
  if ($0 !~ /^[[:space:]]+pnpm_run_resolver_real_test\.go:[0-9]+: pnpm_cold_run roots=1 packages=1 instances=1 cache_entries=[1-9][0-9]* source_artifacts=1 selected=typescript@5\.9\.3 command=SEALED_MISSING_PATH native_DATA=UNAPPROVED$/ || cases++) bad=1
}
END { exit (bad || parents!=1 || cases!=1) }
' "$pnpm_cold_log"
