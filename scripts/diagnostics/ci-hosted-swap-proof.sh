#!/usr/bin/env bash
set -euo pipefail

[[ ${RUNNER_ENVIRONMENT-} == github-hosted && ${RUNNER_OS-} == Linux && ${RUNNER_ARCH-} == X64 ]]
[[ ${GITHUB_REPOSITORY-} == rahoney/heliopause && ${GITHUB_REF-} == refs/heads/diagnostic/r8-hosted-swap-20261127 ]]
[[ $(getconf PAGESIZE) == 4096 ]]
[[ $(git hash-object scripts/ci-integration-cleanup.sh) == 564fb0fea0c6d81f31cb7480df90718b3aec8b7f ]]
printf 'checkout=%s kernel=%s runner_image=%s\n' "$(git rev-parse HEAD)" "$(uname -r)" "${ImageVersion-}"
sudo -n true

swap_table() {
  swapon --bytes --show=NAME,TYPE,SIZE,USED,PRIO --noheadings --raw
}

initial=$(swap_table)
stock=''
priority=''
if [[ -n $initial ]]; then
  [[ $initial != *$'\n'* ]]
  read -r stock type size used priority extra <<< "$initial"
  [[ $type == file && $size =~ ^[0-9]+$ && $used == 0 && -z $extra ]]
  [[ $stock == /* && -f $stock && ! -L $stock && $priority =~ ^(-2|[0-9]+)$ ]]
fi
printf 'before_swap=%s\n' "${initial:-none}"

root=$(mktemp -d "$RUNNER_TEMP/haa-swap-proof.XXXXXXXX")
stock_paused=0
finish() {
  local original=$1 cleanup=0
  trap - EXIT
  if (( stock_paused )); then
    if [[ $priority == -2 ]]; then
      sudo -n swapon -- "$stock" || cleanup=$?
    else
      sudo -n swapon --priority "$priority" -- "$stock" || cleanup=$?
    fi
  fi
  local after leftovers
  after=$(swap_table) || cleanup=$?
  leftovers=$(find "$root" -mindepth 1 -maxdepth 1 -print) || cleanup=$?
  if [[ -n $leftovers ]]; then
    printf 'diagnostic temporary files remain: %s\n' "$leftovers" >&2
    cleanup=1
  fi
  if [[ $after != "$initial" ]]; then
    printf 'runner swap differs after diagnostic: %s\n' "$after" >&2
    cleanup=1
  fi
  rmdir -- "$root" || cleanup=$?
  if (( original != 0 )); then exit "$original"; fi
  (( cleanup == 0 )) || exit "$cleanup"
  printf 'after_swap=%s temporary_directory_absent=yes\n' "${after:-none}"
}
trap 'finish "$?"' EXIT

if [[ -n $stock ]]; then
  sudo -n swapoff -- "$stock"
  stock_paused=1
fi
current=$(swap_table)
[[ -z $current ]]
[[ $(awk '$1 == "SwapFree:" {print $2}' /proc/meminfo) == 0 ]]
printf 'before_prepare_swap_free_kib=0\n'

RUNNER_TEMP="$root" bash -euo pipefail <<'PROBE'
source scripts/ci-integration-cleanup.sh
haa_ci_initialize
time haa_ci_prepare_cuda_swap
[[ -n $haa_cuda_swap_file && $haa_cuda_swap_file == "$RUNNER_TEMP"/haa-cuda-swap.* ]]
[[ $(stat -c '%a:%s' "$haa_cuda_swap_file") == 600:3221225472 ]]
active=$(swapon --bytes --show=NAME,TYPE,SIZE,USED,PRIO --noheadings --raw)
[[ $active != *$'\n'* ]]
read -r name type size used priority extra <<< "$active"
[[ $name == "$haa_cuda_swap_file" && $type == file && $size == 3221221376 && -z $extra ]]
printf 'active_swap=%s\n' "$active"
PROBE

current=$(swap_table)
[[ -z $current ]]
[[ $(awk '$1 == "SwapFree:" {print $2}' /proc/meminfo) == 0 ]]
printf 'after_ci_exit_swap_free_kib=0\n'
