#!/bin/bash
# Small installed-bundle smoke; no runtime rebuild, config change or auth bypass.
set -euo pipefail
if [[ $# != 1 || ! -f $1 || ! -x $1 ]]; then echo 'usage: ci-helper-smoke.sh /absolute/canonical-built/helox' >&2; exit 2; fi
candidate=$1
if [[ $candidate != /* ]]; then echo 'candidate path must be absolute' >&2; exit 2; fi
if [[ -e /run/heliopause-network-policy/helper.sock || -e /run/heliopause-network-policy/observation.sock ]]; then
  echo 'refusing to replace an existing helper lifecycle' >&2; exit 1
fi
RUNNER_TEMP=$(mktemp -d /tmp/haa-ci-smoke.XXXXXXXX)
export RUNNER_TEMP
printf 'smoke evidence: %s\n' "$RUNNER_TEMP"
cp /usr/libexec/heliopause/helox "$RUNNER_TEMP/helox"
sha256sum /usr/libexec/heliopause/helox /usr/libexec/heliopause/haa-network-policy-helper /usr/libexec/heliopause/haa_gvisor_observer /etc/heliopause/network-policy.json > "$RUNNER_TEMP/installed-before.sha256"
stat -c '%n %u:%g %a' /usr/libexec/heliopause/helox /usr/libexec/heliopause/haa-network-policy-helper /etc/heliopause/network-policy.json > "$RUNNER_TEMP/installed-before.stat"
systemctl list-units 'haaobs*.slice' --all --no-legend > "$RUNNER_TEMP/slices-before.txt"
source "$(dirname "${BASH_SOURCE[0]}")/ci-integration-cleanup.sh"
haa_ci_initialize
haa_start_policy_helper
ps -o pid,ppid,pgid,stat,args -p "$haa_recorded_pid" >> "$RUNNER_TEMP/lifecycle.txt"
/usr/libexec/heliopause/helox npm install is-number@7.0.0 --target "$RUNNER_TEMP/before-target" > "$RUNNER_TEMP/production-before.json"
jq -e '.operation_status == "COMPLETED" and .policy.decision == "ALLOW" and .promotion_status == "COMPLETED"' "$RUNNER_TEMP/production-before.json" >/dev/null
haa_install_ci_client "$candidate"
haa_start_policy_helper
ps -o pid,ppid,pgid,stat,args -p "$haa_recorded_pid" >> "$RUNNER_TEMP/lifecycle.txt"
/usr/libexec/heliopause/helox npm install is-number@7.0.0 --target "$RUNNER_TEMP/rebound-target" > "$RUNNER_TEMP/rebound.json"
jq -e '.operation_status == "COMPLETED" and .policy.decision == "ALLOW" and .promotion_status == "COMPLETED"' "$RUNNER_TEMP/rebound.json" >/dev/null
haa_install_ci_client "$RUNNER_TEMP/helox"
haa_start_policy_helper
/usr/libexec/heliopause/helox npm install is-number@7.0.0 --target "$RUNNER_TEMP/restored-target" > "$RUNNER_TEMP/production-restored.json"
jq -e '.operation_status == "COMPLETED" and .policy.decision == "ALLOW" and .promotion_status == "COMPLETED"' "$RUNNER_TEMP/production-restored.json" >/dev/null
haa_stop_policy_helper
if [[ -e /run/heliopause-network-policy/helper.sock || -e /run/heliopause-network-policy/observation.sock ]]; then echo 'helper sockets remain after confirmed stop' >&2; exit 1; fi
systemctl list-units 'haaobs*.slice' --all --no-legend > "$RUNNER_TEMP/slices-after.txt"
cmp "$RUNNER_TEMP/slices-before.txt" "$RUNNER_TEMP/slices-after.txt"
sha256sum --check "$RUNNER_TEMP/installed-before.sha256"
stat -c '%n %u:%g %a' /usr/libexec/heliopause/helox /usr/libexec/heliopause/haa-network-policy-helper /etc/heliopause/network-policy.json > "$RUNNER_TEMP/installed-after.stat"
cmp "$RUNNER_TEMP/installed-before.stat" "$RUNNER_TEMP/installed-after.stat"
printf 'authenticated restart/rebind/restoration complete: %s\n' "$RUNNER_TEMP"
# The shared EXIT handler verifies cleanup again and preserves any primary error.
