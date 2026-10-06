#!/usr/bin/env bash
set -euo pipefail
cd /tmp/haa-m12-004-source
state_root=/home/winamp/.local/state/heliopause/m12-004
original_root=/home/winamp/.local/state/heliopause/m12-003
RUNNER_TEMP="$state_root/lifecycle"
export RUNNER_TEMP
if [[ -e /run/heliopause-network-policy/helper.sock || -e /run/heliopause-network-policy/observation.sock || -e /run/heliopause-observer/gvisor-remote.sock || -e /run/heliopause-observer/.heliopause-supervisor.lock ]]; then
  echo 'existing lifecycle: refuse replacement' >&2
  exit 1
fi
python3 "$state_root/verify-init-candidate.py"
python3 "$original_root/verify-runtime-files.py" before
python3 "$original_root/verify-runtime-files.py" candidate
sudo python3 "$state_root/verify-no-installed-consumers.py"
python3 "$original_root/verify-no-traced-consumers.py"
sha256sum --check "$RUNNER_TEMP/installed-before.sha256"
sudo sha256sum --check "$RUNNER_TEMP/pod-init-before.sha256"
source scripts/ci-integration-cleanup.sh
haa_ci_initialize
finish_with_observer() {
  local original=$? stop=0 restore=0
  trap - EXIT
  if haa_stop_policy_helper; then :; else stop=$?; fi
  if ((stop==0)); then
    # Runtime replacement requires both parent-owned helper stop and complete
    # container absence; Docker query failure remains uncertain.
    if python3 "$original_root/verify-no-traced-consumers.py"; then
      while IFS=' ' read -r restore_mode restore_member; do
        if sudo install -m "$restore_mode" "$RUNNER_TEMP/runtime-original/$restore_member" "/usr/libexec/heliopause/gvisor/$restore_member"; then :; else restore=$?; fi
      done < "$RUNNER_TEMP/runtime-restore-list.txt"
      if python3 "$original_root/verify-runtime-files.py" before; then :; else restore=$?; fi
    else restore=1; fi
    if sudo install -m 0600 "$RUNNER_TEMP/pod-init-original.json" /etc/heliopause/pod-init.json; then :; else restore=$?; fi
    if sudo sha256sum --check "$RUNNER_TEMP/pod-init-before.sha256"; then :; else restore=$?; fi
    sudo stat -c '%u %g %a %n' /etc/heliopause/pod-init.json > "$RUNNER_TEMP/pod-init-after.modes"
    if cmp "$RUNNER_TEMP/pod-init-before.modes" "$RUNNER_TEMP/pod-init-after.modes"; then :; else restore=$?; fi
    if sudo install -m 0755 "$RUNNER_TEMP/observer" /usr/libexec/heliopause/haa_gvisor_observer; then :; else restore=$?; fi
    if sudo install -m 0755 "$RUNNER_TEMP/haa-network-policy-helper" /usr/libexec/heliopause/haa-network-policy-helper; then :; else restore=$?; fi
    if haa_install_ci_client "$RUNNER_TEMP/helox"; then :; else restore=$?; fi
    if sha256sum --check "$RUNNER_TEMP/installed-before.sha256"; then :; else restore=$?; fi
    stat -c '%u %g %a %n' /usr/libexec/heliopause/helox /usr/libexec/heliopause/haa-network-policy-helper /usr/libexec/heliopause/haa_gvisor_observer /etc/heliopause/network-policy.json > "$RUNNER_TEMP/installed-after.modes"
    if cmp "$RUNNER_TEMP/installed-before.modes" "$RUNNER_TEMP/installed-after.modes"; then :; else restore=$?; fi
  else restore=1; fi
  if ((original==0 && restore!=0)); then original=$restore; fi
  haa_ci_finish "$original"
}
trap finish_with_observer EXIT
while IFS=' ' read -r install_mode install_member; do
  sudo install -m "$install_mode" "$original_root/kernel-runtime-candidate/verified-bundle/$install_member" "/usr/libexec/heliopause/gvisor/$install_member"
done < "$RUNNER_TEMP/runtime-restore-list.txt"
sudo install -m 0755 "$state_root/baseline/qualified-timing-policy-helper" /usr/libexec/heliopause/haa-network-policy-helper
sudo install -m 0755 "$state_root/baseline/provider-probe-observer" /usr/libexec/heliopause/haa_gvisor_observer
sudo install -m 0600 tools/gvisor-observer/pod-init.json /etc/heliopause/pod-init.json
haa_install_ci_client "$state_root/qualification/qualified-timing-bootstrap.test"
haa_start_policy_helper
mkdir -p "$state_root/source-home"
cd internal/bootstrap
env -i PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin HOME="$state_root/source-home" DOCKER_HOST=unix:///run/docker.sock DOCKER_CONFIG="$original_root/docker-client-home" HELOX_TERRAFORM_INIT_INTEGRATION=1 HELOX_GITHUB_RELEASE_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.timeout=5m -test.run "${1:-^TestLinuxTerraformInitIntegration$}"
