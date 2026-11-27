#!/usr/bin/env bash
set -euo pipefail

[[ ${RUNNER_ENVIRONMENT-} == github-hosted && ${RUNNER_OS-} == Linux && ${RUNNER_ARCH-} == X64 ]]
[[ ${GITHUB_REPOSITORY-} == rahoney/heliopause && ${GITHUB_REF-} == refs/heads/diagnostic/r8-hosted-swap-20261127 ]]
[[ $(getconf PAGESIZE) == 4096 ]]
[[ $(git hash-object scripts/ci-integration-cleanup.sh) == 564fb0fea0c6d81f31cb7480df90718b3aec8b7f ]]
printf 'checkout=%s kernel=%s runner_image=%s\n' "$(git rev-parse HEAD)" "$(uname -r)" "${ImageVersion-}"

preflight_failure() {
  printf 'swap_preflight check=%s exit=%s\n' "$1" "$2" >&2
  exit "$2"
}

if sudo -n true; then
  printf 'swap_preflight check=sudo exit=0\n'
else
  preflight_failure sudo "$?"
fi

swap_table() {
  swapon --bytes --show=NAME,TYPE,SIZE,USED,PRIO --noheadings --raw
}

# Swap usage can change during the probe. Compare the identities instead.
swap_identities() {
  awk -v omit="${1-}" '
    NF == 0 { next }
    NF != 5 || $1 !~ /^\// || $3 !~ /^[0-9]+$/ ||
      $4 !~ /^[0-9]+$/ || $5 !~ /^-?[0-9]+$/ || seen[$1]++ { exit 1 }
    $1 != omit { print $1, $2, $3, $5 }
  ' | LC_ALL=C sort
}

if initial=$(swap_table); then
  printf 'swap_preflight check=query exit=0\nbefore_swap=%s\n' "${initial:-none}"
else
  preflight_failure query "$?"
fi
baseline=$(swap_identities <<< "$initial") || preflight_failure malformed_table "$?"
awk '/^(MemAvailable|SwapTotal|SwapFree):/' /proc/meminfo ||
  preflight_failure meminfo "$?"
printf 'swap_preflight check=eligible exit=0\n'

temp_dir=$(CDPATH= cd -- "$RUNNER_TEMP" && pwd -P)
root=$(mktemp -d "$temp_dir/haa-swap-proof.XXXXXXXX")
finish() {
  local original=$1 cleanup=0 after identities
  trap - EXIT
  if after=$(swap_table) && identities=$(swap_identities <<< "$after"); then
    if [[ $identities != "$baseline" ]]; then
      printf 'runner swap identities differ after diagnostic: %s\n' "$after" >&2
      cleanup=1
    fi
  else
    cleanup=$?
    printf 'final swap table could not be verified (exit=%s)\n' "$cleanup" >&2
  fi
  rm -f -- "$root/first-swap-read" || cleanup=$?
  rmdir -- "$root" || cleanup=$?
  if (( original != 0 )); then exit "$original"; fi
  (( cleanup == 0 )) || exit "$cleanup"
  printf 'after_swap=%s temporary_directory_absent=yes\n' "${after:-none}"
}
trap 'finish "$?"' EXIT

export -f swap_table swap_identities
PROBE_BASELINE="$baseline" RUNNER_TEMP="$root" bash -euo pipefail <<'PROBE'
minimum_kib=3145724
read_free='$1 == "SwapFree:" && $2 ~ /^[0-9]+$/ && $3 == "kB" { print $2 }'
first_read="$RUNNER_TEMP/first-swap-read"

# Keep the provider's swap enabled. Control only the helper's first read
# when the host already has enough. The read after swapon always uses procfs.
awk() {
  if [[ $# == 2 && $1 == "$read_free" && $2 == /proc/meminfo && ! -e $first_read ]]; then
    local actual input mode
    actual=$(command awk "$@") || return "$?"
    if [[ ! $actual =~ ^(0|[1-9][0-9]*)$ ]]; then
      printf 'invalid initial SwapFree: %s\n' "$actual" >&2
      return 1
    fi
    input=$actual
    mode=native-shortage
    if (( actual >= minimum_kib )); then
      input=0
      mode=controlled-shortage
    fi
    printf '%s\n' "$input" > "$first_read" || return "$?"
    printf 'probe mode=%s actual_swap_free_kib=%s initial_input_kib=%s\n' "$mode" "$actual" "$input" >&2
    printf '%s\n' "$input"
    return
  fi
  command awk "$@"
}

source scripts/ci-integration-cleanup.sh
haa_ci_initialize
time haa_ci_prepare_cuda_swap
IFS= read -r initial_kib < "$first_read"
allocated_bytes=$((((minimum_kib - initial_kib + 4 + 1023) / 1024) * 1024 * 1024))
usable_bytes=$((allocated_bytes - 4096))
if [[ $haa_cuda_swap_file != "$RUNNER_TEMP"/haa-cuda-swap.* ||
      $(stat -c '%a:%s' "$haa_cuda_swap_file") != "600:$allocated_bytes" ]]; then
  printf 'owned swapfile has the wrong path, permissions, or size\n' >&2
  exit 1
fi

active=$(swap_table)
rest=$(swap_identities "$haa_cuda_swap_file" <<< "$active")
if [[ $rest != "$PROBE_BASELINE" ]]; then
  printf 'existing swap identities changed during diagnostic\n' >&2
  exit 1
fi
owned=$(awk -v name="$haa_cuda_swap_file" -v size="$usable_bytes" '
  $1 == name && $2 == "file" && $3 == size { print }
' <<< "$active")
if [[ -z $owned ]]; then
  printf 'owned swapfile missing from kernel table or usable size differs\n' >&2
  exit 1
fi
printf 'active_swap=%s\nprior_swap_identities_unchanged=yes\n' "$owned"
PROBE
