#!/bin/bash
# Hosted-CI-only native supply; the payload runs only inside gVisor as UID1000.
set -euo pipefail
test "$#" = 0
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
