#!/usr/bin/env bash
set -euo pipefail
cd /tmp/haa-m12-004-source
state_root=/home/winamp/.local/state/heliopause/m12-004
original_root=/home/winamp/.local/state/heliopause/m12-003
if [[ -e "$state_root/qualification/hold-between-consumers" ]]; then
 echo "qualification preparation held after prior completed restoration" >&2
 exit 98
fi
RUNNER_TEMP="$state_root/lifecycle"
export RUNNER_TEMP
if [[ -e /run/heliopause-network-policy/helper.sock || -e /run/heliopause-network-policy/observation.sock || -e /run/heliopause-observer/gvisor-remote.sock || -e /run/heliopause-observer/.heliopause-supervisor.lock ]]; then
  echo 'existing lifecycle: refuse replacement' >&2
  exit 1
fi
python3 "$state_root/verify-qualified-candidate.py"
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
qualification_package=${4:-bootstrap}
case "$qualification_package" in bootstrap|sandbox|promotion) ;; *) exit 2 ;; esac
haa_install_ci_client "$state_root/qualification/qualified-timing-$qualification_package.test"
haa_start_policy_helper
mkdir -p "$state_root/source-home"
qualification_package=${4:-bootstrap}
case "$qualification_package" in bootstrap|sandbox|promotion) ;; *) exit 2 ;; esac
cd "internal/$qualification_package"
qualification_profile=${2:-none}
qualification_timeout=${3:-5m}
case "$qualification_timeout" in 5m|10m|15m|30m|40m) ;; *) exit 2 ;; esac
qualification_env=(HELOX_TERRAFORM_PROVIDER_INTEGRATION=1 HELOX_GO_BUILD_CLI_INTEGRATION=1 HELOX_GITHUB_RELEASE_INTEGRATION=1 HELOX_GVISOR_INTEGRATION=1 HELOX_NPM_DYNAMIC_INTEGRATION=1 HELOX_PYPI_PROFILE_TRANSACTION_INTEGRATION=1 HELOX_PYPI_PROMOTION_INTEGRATION=1 HELOX_GO_BUILD_INTEGRATION=1 HELOX_PROMOTION_INTEGRATION=1 HELOX_PYPI_RESOLVER_INTEGRATION=1 HELOX_PYTORCH_RESOLVER_INTEGRATION=1 HELOX_NPM_RESOLVER_INTEGRATION=1)
case "$qualification_profile" in
 none) qualification_env+=(HELOX_PYTORCH_PROFILE=cpu) ;;
 cpu)
  qualification_env+=(HELOX_PYTORCH_FULL_INTEGRATION=1 HELOX_PYTORCH_PROFILE=cpu HELOX_PYTORCH_INSPECTION_PREREQUISITE= HELOX_PYTORCH_NOT_ATTESTED_MODULE=)
  ;;
 cu126)
  qualification_env+=(HELOX_PYTORCH_FULL_INTEGRATION=1 HELOX_PYTORCH_PROFILE=cu126 HELOX_PYTORCH_NOT_ATTESTED_MODULE=triton.profiler.viewer)
  qualification_env+=('HELOX_PYTORCH_INSPECTION_PREREQUISITE={"target_sha256":"94e4f9bd6b9b21aad545e0d441707094e4ff88ab420d62b85fcdc8b6da1e039f","source":"pypi","project":"numpy","version":"2.4.6","filename":"numpy-2.4.6-cp314-cp314-manylinux_2_27_x86_64.manylinux_2_28_x86_64.whl","sha256":"a2c306dea656c12c68f51f4cea133cbe78ca7435eb28c735eac1d3ebe73be6e8","reason":"BROADER_MODULE_PROBES"}')
  ;;
 cu130|cu132) qualification_env+=("HELOX_PYTORCH_PROFILE=$qualification_profile") ;;
 *) exit 2 ;;
esac
qualification_env+=("HELOX_INTEGRATION_EVIDENCE_ROOT=/tmp/haa-m12-004-timing-native-e2e" "HELOX_INTEGRATION_CACHE_ROOT=")
env -i PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin HOME="$state_root/source-home" DOCKER_HOST=unix:///run/docker.sock DOCKER_CONFIG="$original_root/docker-client-home" HELOX_CARGO_RESOLVER_INTEGRATION=1 HELOX_CARGO_ADD_INTEGRATION=1 HELOX_CARGO_BUILD_CLI_INTEGRATION=1 HELOX_GO_RESOLVER_INTEGRATION=1 HELOX_GO_RESOLVER_PROJECT_INTEGRATION=1 HELOX_PYPI_DYNAMIC_INTEGRATION=1 "${qualification_env[@]}" /usr/libexec/heliopause/helox -test.v -test.timeout="$qualification_timeout" -test.run "${1:-^TestLinuxCargoDependencyFreeBuildCLIIntegration$}"
