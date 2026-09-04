#!/usr/bin/env sh
# Build the patched runsc container runtime against the exact gVisor/Bazel
# and HAA patch identities in the canonical runtime lock.
# This script produces one Linux amd64 binary, verifies its capability probe,
# and does not install or activate it on the Host.
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 ABSOLUTE_OUTPUT_PATH" >&2
  exit 2
fi

output_path=$1
case "$output_path" in
  /*) ;;
  *) echo "runsc output must be absolute" >&2; exit 2 ;;
esac

runtime_lock=scripts/runtimes.lock.json
test -f "$runtime_lock"
gvisor_repository=$(jq -er '.gvisor.source_repository' "$runtime_lock")
gvisor_commit=$(jq -er '.gvisor.commit' "$runtime_lock")
gvisor_patch_path=$(jq -er '.gvisor.patch.path' "$runtime_lock")
gvisor_patch_sha256=$(jq -er '.gvisor.patch.sha256' "$runtime_lock")
bazel_url=$(jq -er '.bazel.linux_x86_64_url' "$runtime_lock")
bazel_sha512=$(jq -er '.bazel.linux_x86_64_sha512' "$runtime_lock")

case "$gvisor_commit" in
  *[!0-9a-f]*)
    echo "runtime lock gVisor commit is invalid" >&2
    exit 1
    ;;
esac
test "$(printf %s "$gvisor_commit" | wc -c | tr -d ' ')" = 40 || {
  echo "runtime lock gVisor commit is invalid" >&2
  exit 1
}

case "$gvisor_patch_path" in
  tools/gvisor/*.patch) ;;
  *) echo "runtime lock gVisor patch path is invalid" >&2; exit 1 ;;
esac
case "$gvisor_patch_path" in
  *..*) echo "runtime lock gVisor patch path cannot escape" >&2; exit 1 ;;
esac
test -f "$gvisor_patch_path" || {
  echo "runtime lock gVisor patch file missing: $gvisor_patch_path" >&2
  exit 1
}
actual_patch_sha256="$(sha256sum "$gvisor_patch_path" | awk '{print $1}')"
test "$actual_patch_sha256" = "$gvisor_patch_sha256" || {
  echo "runtime lock gVisor patch sha256 mismatch: expected $gvisor_patch_sha256, got $actual_patch_sha256" >&2
  exit 1
}
patch_abs="$(cd "$(dirname "$gvisor_patch_path")" && pwd)/$(basename "$gvisor_patch_path")"

work_root=$(mktemp -d "${TMPDIR:-/tmp}/helox-release-runsc.XXXXXX")
cleanup() { rm -rf "$work_root"; }
trap cleanup EXIT HUP INT TERM

gvisor_source="${GVISOR_LOCAL_REPO:-$gvisor_repository}"
git clone --filter=blob:none "$gvisor_source" "$work_root/gvisor"
git -C "$work_root/gvisor" checkout --detach "$gvisor_commit"
test "$(git -C "$work_root/gvisor" rev-parse HEAD)" = "$gvisor_commit"
test -z "$(git -C "$work_root/gvisor" status --porcelain)" || {
  echo "gVisor upstream source is not clean" >&2
  exit 1
}

git -C "$work_root/gvisor" apply --check "$patch_abs" || {
  echo "gVisor patch failed application check" >&2
  exit 1
}
git -C "$work_root/gvisor" apply "$patch_abs" || {
  echo "gVisor patch application failed" >&2
  exit 1
}
test -n "$(git -C "$work_root/gvisor" status --porcelain)" || {
  echo "gVisor patch produced no changes" >&2
  exit 1
}

if [ -n "${BAZEL_PATH:-}" ] && [ -x "${BAZEL_PATH}" ]; then
  cp "${BAZEL_PATH}" "$work_root/bazel"
else
  curl --fail --location --silent --show-error --output "$work_root/bazel" "$bazel_url"
fi
test "$(sha512sum "$work_root/bazel" | awk '{print $1}')" = "$bazel_sha512"
chmod 0755 "$work_root/bazel"

bazel_output_args=""
if [ -n "${BAZEL_OUTPUT_USER_ROOT:-}" ]; then
  bazel_output_args="--output_user_root=${BAZEL_OUTPUT_USER_ROOT}"
fi

(
  cd "$work_root/gvisor"
  # shellcheck disable=SC2086
  "$work_root/bazel" $bazel_output_args build -c opt //runsc:runsc
)

built_binary="$work_root/gvisor/bazel-bin/runsc/runsc_/runsc"
test -f "$built_binary"

# Capability probe verification:
# Built binary MUST advertise all required observation points in trace metadata.
meta_output="$("$built_binary" trace metadata)"
for point in "syscall/open_result" "sentry/mount_topology_snapshot" "sentry/mount_topology_mutation"; do
  case "$meta_output" in
    *"$point"*) ;;
    *) echo "built runsc binary missing required capability point: $point" >&2; exit 1 ;;
  esac
done

test ! -e "$output_path"
install -m 0755 "$built_binary" "$output_path"
