#!/bin/bash
# Hosted-CI-only native supply; Bun executes only inside gVisor as UID1000.
set -euo pipefail
case "$#:${1-}" in
  0:|1:project) ;;
  *) echo 'unsupported Bun CI profile' >&2; exit 2 ;;
esac
bun_profile=${1:-sandbox}
test "$(id -u)" != 0
test -n "${RUNNER_TEMP:-}"
test "${RUNNER_TEMP:0:1}" = /
test ! -e /usr/lib/heliopause
test ! -L /usr/lib/heliopause
bun_work=$(mktemp -d "$RUNNER_TEMP/haa-bun.XXXXXXXX")
bun_root_created=0
bun_cleanup() {
  local primary=$? cleanup_status=0
  trap - EXIT
  if ((bun_root_created)); then
    sudo -n rm -f -- /usr/lib/heliopause/current /usr/lib/heliopause/versions/v1.0.0-bun-ci/bun-linux-amd64 || cleanup_status=1
    sudo -n rmdir -- /usr/lib/heliopause/versions/v1.0.0-bun-ci /usr/lib/heliopause/versions /usr/lib/heliopause || cleanup_status=1
  fi
  rm -rf -- "$bun_work" || cleanup_status=1
  if ((cleanup_status)); then echo 'Bun CI supply cleanup unproven' >&2; fi
  if ((primary == 0)); then primary=$cleanup_status; fi
  exit "$primary"
}
trap bun_cleanup EXIT
go run ./scripts/generate-runtime-lock.go -check
bun_url=$(jq -er '.bun_tool.archive.url' scripts/runtimes.lock.json)
bun_size=$(jq -er '.bun_tool.archive.size' scripts/runtimes.lock.json)
curl --fail --show-error --silent --location --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 120 --max-filesize "$bun_size" --output "$bun_work/archive.zip" "$bun_url"
go run ./scripts/update-runtime materialize-bun --root "$PWD" --archive "$bun_work/archive.zip" --output "$bun_work/bun-linux-amd64" --notices-output "$bun_work/bun-NOTICE.txt"
rm -- "$bun_work/archive.zip"
sudo -n mkdir -m 0755 -- /usr/lib/heliopause
bun_root_created=1
sudo -n install -d -m 0755 /usr/lib/heliopause/versions/v1.0.0-bun-ci
sudo -n install -m 0555 "$bun_work/bun-linux-amd64" /usr/lib/heliopause/versions/v1.0.0-bun-ci/bun-linux-amd64
sudo -n ln -s versions/v1.0.0-bun-ci /usr/lib/heliopause/current
# The owning parent installed this tagged client and started its policy helper.
if [[ "$bun_profile" == project ]]; then
  (
    cd internal/bootstrap
    HELOX_BUN_CACHE_INTEGRATION=1 HELOX_BUN_PROJECT_TRANSACTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestBunPublic(ApprovedCache|ProjectTransaction)Integration$'
    HELOX_BUN_PROJECT_CLI_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestBunProjectCLIIntegration$'
    HELOX_BUN_NATIVE_ORIGINAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestBunNativeOriginalIntegration$'
  )
  bun_run_log="$bun_work/run.log"
  bun_run_status=0
  (
    cd internal/bootstrap
    HELOX_BUN_NATIVE_RUNTIME=1 HELOX_BUN_RUN_EXECUTION_WORKFLOW_INTEGRATION=1 HELOX_BUN_RUN_EXECUTION_CLI_INTEGRATION=1 HELOX_BUN_RUN_EXECUTION_CLI_REFUSAL_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=10m -test.run '^TestLinuxBunRunExecution(Workflow|CLI(Refusal)?)Integration$'
  ) >"$bun_run_log" 2>&1 || bun_run_status=$?
  bun_run_log_status=0
  cat -- "$bun_run_log" || bun_run_log_status=$?
  if ((bun_run_status)); then exit "$bun_run_status"; fi
  if ((bun_run_log_status)); then exit "$bun_run_log_status"; fi
  awk '
  BEGIN {
  expected["TestLinuxBunRunExecutionWorkflowIntegration"]=1
  expected["TestLinuxBunRunExecutionCLIIntegration"]=1
  expected["TestLinuxBunRunExecutionCLIRefusalIntegration"]=1
  cli[1]="alias=bunx format=json language=en show_output=false exit_code=0"
  cli[2]="alias=x format=json language=ko show_output=true exit_code=1"
  cli[3]="alias=bunx format=human language=ko show_output=true exit_code=0"
  }
  /--- SKIP:/ { bad=1 }
  /^--- PASS:/ { if (!($3 in expected) || parents[$3]++) bad=1; parent_count++ }
  /production_bun_run_application attempt=/ {
  if ($0 !~ /^[[:space:]]+node_execution_run_integration_test\.go:[0-9]+: production_bun_run_application attempt=[12] artifact_role=true target_loaded=true exit_code=[01] stdout_bytes=[1-9][0-9]* stderr_bytes=0 user_program_run=true$/) bad=1
  id=$3; sub(/^attempt=/,"",id)
  if (apps[id]++ || $6 != "exit_code=" (id-1)) bad=1
  app_count++
  }
  /production_bun_run_cli case=/ {
  id=$3; sub(/^case=/,"",id)
  if (!(id in cli) || cases[id]++ || $0 !~ ("^[[:space:]]+npm_execution_integration_test\\.go:[0-9]+: production_bun_run_cli case=" id " " cli[id] " current_policy=ALLOW guest_cleanup=CONFIRMED$")) bad=1
  case_count++
  }
  /production_bun_run_cli_graph / {
  if ($0 !~ /^[[:space:]]+npm_execution_integration_test\.go:[0-9]+: production_bun_run_cli_graph nodes=[1-9][0-9]* edges=[1-9][0-9]* primary=rimraf current_policy=ALLOW$/ || graph++) bad=1
  nodes=$3; sub(/^nodes=/,"",nodes); if (nodes<=1) bad=1
  }
  /production_bun_run_refusal case=/ {
  id=$3; sub(/^case=/,"",id)
  reason=(id==1 ? "ambiguous" : "missing")
  if (id !~ /^[12]$/ || refused[id]++ || $0 !~ ("^[[:space:]]+npm_execution_refusal_integration_test\\.go:[0-9]+: production_bun_run_refusal case=" id " reason=" reason " current_policy=ALLOW preparation=INCOMPLETE runtime_attempted=false exit_absent=true output_bytes=0 stored_manifest=0$")) bad=1
  refusal_count++
  }
  END { exit (bad || parent_count!=3 || app_count!=2 || case_count!=3 || refusal_count!=2 || graph!=1) }
  ' "$bun_run_log"
  exit 0
fi
(
  cd internal/sandbox
  HELOX_BUN_NATIVE_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=3m -test.run '^(TestBunPublicNativeResolverIntegration|TestBunActualNativeProjectSelectionBoundary)$'
  HELOX_BUN_PROJECT_QUARANTINE=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestBunPublicProjectQuarantineIntegration$'
  HELOX_BUN_ARTIFACT_EXECUTION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=3m -test.run '^TestBunActualArtifactExecutionBoundary$'
)

bun_cold_log="$bun_work/cold.log"
bun_cold_status=0
(
  cd internal/sandbox
  HELOX_BUN_TOOL_RUN_SELECTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=3m -test.run '^TestBunPublicColdToolRunSelectionIntegration$'
) >"$bun_cold_log" 2>&1 || bun_cold_status=$?
bun_cold_log_status=0
cat -- "$bun_cold_log" || bun_cold_log_status=$?
if ((bun_cold_status)); then exit "$bun_cold_status"; fi
if ((bun_cold_log_status)); then exit "$bun_cold_log_status"; fi
awk '
/--- SKIP:/ { bad=1 }
/^--- PASS:/ { if ($3 != "TestBunPublicColdToolRunSelectionIntegration" || parents++) bad=1 }
/bun_cold_run / {
  if ($0 !~ /^[[:space:]]+bun_run_resolver_real_test\.go:[0-9]+: bun_cold_run request=typescript@5\.9\.3 roots=1 instances=1 cache_entries=[1-9][0-9]* source_artifacts=1 command=SEALED_MISSING_PATH native_DATA=UNAPPROVED Policy\/program=NOT_RUN$/ || cases++) bad=1
}
END { exit (bad || parents!=1 || cases!=1) }
' "$bun_cold_log"
