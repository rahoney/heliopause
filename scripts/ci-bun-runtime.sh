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
  exit 0
fi
(
  cd internal/sandbox
  HELOX_BUN_NATIVE_RESOLVER_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=3m -test.run '^(TestBunPublicNativeResolverIntegration|TestBunActualNativeProjectSelectionBoundary)$'
  HELOX_BUN_PROJECT_QUARANTINE=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^TestBunPublicProjectQuarantineIntegration$'
)
