#!/bin/bash
# CI fixture only: expose compiled fixed DATA inside a disposable mount
# namespace, then run the existing protected test client as its ordinary owner.
set -euo pipefail
case "$#" in 5|6) ;; *) exit 2 ;; esac
umask 077
case "$1" in tools-native|tools-bootstrap|run) ;; *) exit 2 ;; esac
profile=$1
input=$2
user_id=$3
group_id=$4
user_home=$5
test "$(id -u)" = 0
[[ "$user_id" =~ ^[0-9]+$ && "$group_id" =~ ^[0-9]+$ ]]
test "$user_id" != 0
test "$(readlink /proc/self/ns/net)" = "$(readlink /proc/1/ns/net)"
test -n "${RUNNER_TEMP:-}"
test "${RUNNER_TEMP:0:1}" = /
test "${input:0:1}" = /
test "${user_home:0:1}" = /
test -f "$input"
test ! -L "$input"
test "$(stat -c '%h' "$input")" = 1
test "$(stat -c '%u' "$input")" = "$user_id"
test ! -e /usr/lib/heliopause
test ! -L /usr/lib/heliopause
repo=$(cd -- "$(dirname -- "$0")/.." && pwd -P)
expected_sha=$(jq -er '.pipx_tool.tool.sha256' "$repo/scripts/runtimes.lock.json")
expected_size=$(jq -er '.pipx_tool.tool.size' "$repo/scripts/runtimes.lock.json")
test "$(stat -c '%s' "$input")" = "$expected_size"
test "$(sha256sum "$input" | cut -d ' ' -f 1)" = "$expected_sha"
if [[ "$#" == 5 ]]; then
  work=$(mktemp -d "$RUNNER_TEMP/haa-pipx-installed.XXXXXXXX")
  work_identity=$(stat -c '%d:%i:%u:%a' "$work")
  cleanup() {
    local primary=$? cleanup_status=0 namespace current path
    trap - EXIT
    if [[ -f "$work/mount-namespace" && ! -L "$work/mount-namespace" && "$(stat -c '%s' "$work/mount-namespace")" -le 64 ]]; then
      namespace=$(cat "$work/mount-namespace") || cleanup_status=1
      if [[ ! "$namespace" =~ ^mnt:\[[0-9]+\]$ ]]; then cleanup_status=1; fi
      for path in /proc/[0-9]*/ns/mnt; do
        if current=$(readlink "$path"); then
          if [[ "$current" == "$namespace" ]]; then cleanup_status=1; fi
        elif [[ -d "${path%/ns/mnt}" ]]; then
          cleanup_status=1
        fi
      done
    else
      cleanup_status=1
    fi
    if [[ "$(stat -c '%d:%i:%u:%a' "$work")" != "$work_identity" ]]; then cleanup_status=1; fi
    if ((cleanup_status == 0)); then rm -rf -- "$work" || cleanup_status=1; fi
    if ((cleanup_status)); then echo 'pipx installed fixture cleanup unproven' >&2; else echo 'pipx installed fixture namespace reaped; owned workspace removed'; fi
    if ((primary == 0)); then primary=$cleanup_status; fi
    exit "$primary"
  }
  trap cleanup EXIT
  unshare --mount --fork --kill-child /bin/bash "$0" "$@" "$work"
  exit 0
fi
work=$6
test "$(readlink /proc/self/ns/mnt)" != "$(readlink /proc/1/ns/mnt)"
test -d "$work"
test ! -L "$work"
test "$(stat -c '%u:%a' "$work")" = 0:700
readlink /proc/self/ns/mnt > "$work/mount-namespace"
mount --make-rprivate /
mkdir -m 0755 "$work/upper"
mkdir -m 0700 "$work/mount-work"
mount -t overlay overlay -o "lowerdir=/usr/lib,upperdir=$work/upper,workdir=$work/mount-work" /usr/lib
test "$(stat -c '%a' /usr/lib)" = 755
mkdir -m 0755 /usr/lib/heliopause /usr/lib/heliopause/versions /usr/lib/heliopause/versions/v0.0.0-ci-fixture
ln -s versions/v0.0.0-ci-fixture /usr/lib/heliopause/current
installed=/usr/lib/heliopause/versions/v0.0.0-ci-fixture/pipx-runtime-linux-amd64.zip
install -m 0444 -o 0 -g 0 -- "$input" "$installed"
test "$(sha256sum "$installed" | cut -d ' ' -f 1)" = "$expected_sha"
test "$(sha256sum "$input" | cut -d ' ' -f 1)" = "$expected_sha"
mount --bind /usr/lib/heliopause /usr/lib/heliopause
mount -o remount,bind,ro,noexec,nosuid,nodev /usr/lib/heliopause
cd -- "$repo"
ordinary=(/usr/bin/setpriv --reuid="$user_id" --regid="$group_id" --init-groups /usr/bin/env -i "PATH=$PATH" "HOME=$user_home" LANG=C.UTF-8 "RUNNER_TEMP=$RUNNER_TEMP" "GOCACHE=${GOCACHE:-}" "GOMODCACHE=${GOMODCACHE:-}" "HELOX_TOOL_CACHE=${HELOX_TOOL_CACHE:-}" GOTOOLCHAIN=local GOENV=off GOWORK=off GOFLAGS= "GOMAXPROCS=${GOMAXPROCS:-2}" "GOPROXY=${GOPROXY:-https://proxy.golang.org,direct}")
case "$profile" in
  run)
    "${ordinary[@]}" /bin/bash scripts/ci-uv-tools.sh pipx-run-workflow
    "${ordinary[@]}" /bin/bash scripts/ci-uv-tools.sh pipx-run-cli
    ;;
  tools-native)
    cd internal/sandbox
    "${ordinary[@]}" HELOX_PIPX_NATIVE_SESSION_INTEGRATION=1 HELOX_PIPX_NATIVE_INSTALLED_SUPPLY=1 HELOX_PIPX_QUERY_SUPPLY_ROOT="$(dirname -- "$input")/supply" HELOX_PIPX_RUNTIME_ZIP=/usr/lib/heliopause/current/pipx-runtime-linux-amd64.zip HELOX_PIPX_PUBLIC_RESOLVER_INTEGRATION=1 HELOX_PIPX_PUBLIC_UPGRADE_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^(TestLinuxPipxObservedNativeInstallIntegration|TestLinuxPipxPublicInstallResolverIntegration|TestLinuxPipxPublicOriginalUpgradeIntegration)$'
    ;;
  tools-bootstrap)
    "${ordinary[@]}" /bin/bash scripts/ci-uv-tools.sh pipx-bootstrap
    ;;
esac
