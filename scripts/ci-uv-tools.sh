#!/bin/bash
# The parent owns the protected tagged client and its policy-helper lifecycle.
# Interpreter fixture bytes are exported from a pinned, never-started image.
set -euo pipefail
case "$#:${1-}" in
  0:|1:bootstrap|1:pipx-bootstrap|1:python-execution-preparation|1:python-execution-inputs) ;;
  *) echo 'unsupported UV tools CI profile' >&2; exit 2 ;;
esac
test "$(id -u)" != 0
if [[ ${1-} != bootstrap && ${1-} != pipx-bootstrap && ${1-} != python-execution-preparation && ${1-} != python-execution-inputs ]]; then
  (
    cd internal/sandbox
    HELOX_UV_TOOL_RESOLVER_INTEGRATION=1 HELOX_UV_TOOL_UPGRADE_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestUVPublicTool(Install|Upgrade)ResolverIntegration$'
  )
  exit 0
fi
test -n "${RUNNER_TEMP:-}"
test "${RUNNER_TEMP:0:1}" = /
uv_tools_work=$(mktemp -d "$RUNNER_TEMP/haa-uv-tools.XXXXXXXX")
uv_tools_container=''
uv_tools_owner=''
uv_tools_image=''
uv_tools_image_id=''
uv_tools_attempted=0
uv_tools_owned() {
  docker inspect "$uv_tools_container" | jq -e --arg id "$uv_tools_container" --arg image "$uv_tools_image_id" --arg owner "$uv_tools_owner" '
    length == 1 and .[0].Id == $id and .[0].Image == $image and
    .[0].Config.Labels["io.heliopause.ci.uv-tools-data"] == $owner and
    .[0].State.Status == "created" and .[0].State.Running == false' >/dev/null
}
uv_tools_remove() {
  local found
  found=$(docker ps -aq --no-trunc --filter "label=io.heliopause.ci.uv-tools-data=$uv_tools_owner") || return $?
  if [[ -n "$found" ]]; then
    [[ "$found" =~ ^[0-9a-f]{64}$ ]] || return 1
    [[ -z "$uv_tools_container" || "$found" == "$uv_tools_container" ]] || return 1
    uv_tools_container=$found
    uv_tools_owned || return $?
    docker rm "$uv_tools_container" >/dev/null || return $?
  fi
  uv_tools_container=''
  uv_tools_attempted=0
}
uv_tools_cleanup() {
  local primary=$? cleanup_status=0
  trap - EXIT
  if ((uv_tools_attempted)); then uv_tools_remove || cleanup_status=1; fi
  if ((cleanup_status == 0)); then rm -rf -- "$uv_tools_work" || cleanup_status=1; fi
  if ((cleanup_status)); then echo 'UV tool image data cleanup unproven' >&2; fi
  if ((primary == 0)); then primary=$cleanup_status; fi
  exit "$primary"
}
trap uv_tools_cleanup EXIT
uv_tools_owner=$(cat /proc/sys/kernel/random/uuid)
uv_tools_image=$(jq -er '.python_image.reference' scripts/runtimes.lock.json)
go run ./scripts/generate-runtime-lock.go -check
[[ "$uv_tools_image" == python:*@sha256:* ]]
docker image inspect "$uv_tools_image" | jq -e --arg reference "python@${uv_tools_image##*@}" '
  length == 1 and .[0].Os == "linux" and .[0].Architecture == "amd64" and
  (.[0].RepoDigests | index($reference) != null)' >/dev/null
uv_tools_image_id=$(docker image inspect --format '{{.Id}}' "$uv_tools_image")
uv_tools_attempted=1
uv_tools_container=$(docker create --pull never --network none --read-only --cap-drop ALL --label "io.heliopause.ci.uv-tools-data=$uv_tools_owner" --entrypoint /bin/false "$uv_tools_image")
[[ "$uv_tools_container" =~ ^[0-9a-f]{64}$ ]]
uv_tools_owned
docker cp "$uv_tools_container:/usr/local/bin/python3.14" "$uv_tools_work/python3.14"
test -f "$uv_tools_work/python3.14"
test ! -L "$uv_tools_work/python3.14"
test "$(stat -c '%h' "$uv_tools_work/python3.14")" = 1
test "$(stat -c '%s' "$uv_tools_work/python3.14")" -gt 0
test "$(stat -c '%s' "$uv_tools_work/python3.14")" -le 16777216
chmod 0400 "$uv_tools_work/python3.14"
uv_tools_sha=$(sha256sum "$uv_tools_work/python3.14" | cut -d ' ' -f 1)
uv_tools_owned
uv_tools_remove
if [[ ${1-} == python-execution-preparation ]]; then
  (
    cd internal/bootstrap
    HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_PYTHON_EXECUTION_PREPARATION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestLinuxPythonExecutionPreparationIntegration$'
  )
  exit 0
fi
if [[ ${1-} == python-execution-inputs ]]; then
  (
    cd internal/bootstrap
    HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_PYTHON_EXECUTION_INPUT_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestLinuxPythonExecutionInputsIntegration$'
  )
  exit 0
fi
if [[ ${1-} == pipx-bootstrap ]]; then
  (
    cd internal/bootstrap
    HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_PIPX_TOOL_INSTALL_CLI_INTEGRATION=1 HELOX_PIPX_TOOL_UPDATE_CLI_INTEGRATION=1 HELOX_PIPX_TOOL_NATIVE_ADOPTION_INTEGRATION=1 HELOX_PIPX_TOOL_UPDATE_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=12m -test.run '^(TestLinuxPipxTool(Install|Upgrade|NativeAdoption)CLIIntegration|TestPipxPublicToolApprovedUpgradeIntegration)$'
  )
else
(
  cd internal/bootstrap
  HELOX_TOOL_TARGET_PYTHON_FIXTURE="$uv_tools_work/python3.14" HELOX_TOOL_TARGET_PYTHON_SHA256="$uv_tools_sha" HELOX_UV_TOOL_INSTALL_CLI_INTEGRATION=1 HELOX_UV_TOOL_UPDATE_CLI_INTEGRATION=1 HELOX_UV_TOOL_UPDATE_INTEGRATION=1 HELOX_UV_TOOL_UPDATE_EVIDENCE_FAILURE=1 HELOX_UV_TOOL_NATIVE_ADOPTION_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=12m -test.run '^(TestLinuxUVTool(Install|Upgrade|NativeAdoption)CLIIntegration|TestUVPublicTool(ApprovedUpgrade|UpgradeEvidenceFailure)Integration)$'
)
fi
