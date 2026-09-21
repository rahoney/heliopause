#!/usr/bin/env sh
# Build the complete patched gVisor runtime bundle against the exact source,
# patch, Bazel, and member identities in the canonical runtime lock. The bundle
# is self-contained: no installation step may download a missing sidecar.
set -eu

test "$(uname -s)" = Linux && test "$(uname -m)" = x86_64 || {
  echo "local gVisor bundle build supports Linux amd64 only" >&2
  exit 1
}

if [ "$#" -ne 1 ]; then
  echo "usage: $0 ABSOLUTE_OUTPUT_DIRECTORY" >&2
  exit 2
fi

output_directory=$1
case "$output_directory" in
  /*) ;;
  *) echo "gVisor bundle output must be absolute" >&2; exit 2 ;;
esac
test ! -e "$output_directory" || {
  echo "gVisor bundle output already exists" >&2
  exit 1
}

runtime_lock=scripts/runtimes.lock.json
test -f "$runtime_lock"
gvisor_repository=$(jq -er '.gvisor.source_repository' "$runtime_lock")
gvisor_commit=$(jq -er '.gvisor.commit' "$runtime_lock")
gvisor_patch_path=$(jq -er '.gvisor.patch.path' "$runtime_lock")
gvisor_patch_sha256=$(jq -er '.gvisor.patch.sha256' "$runtime_lock")
bundle_architecture=$(jq -er '.gvisor.runtime_bundle.architecture' "$runtime_lock")
bazel_url=$(jq -er '.bazel.linux_x86_64_url' "$runtime_lock")
bazel_sha512=$(jq -er '.bazel.linux_x86_64_sha512' "$runtime_lock")
bazel_version=$(jq -er '.bazel.version' "$runtime_lock")

test "$bundle_architecture" = amd64
case "$gvisor_commit" in
  *[!0-9a-f]*) echo "runtime lock gVisor commit is invalid" >&2; exit 1 ;;
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
  echo "runtime lock gVisor patch sha256 mismatch" >&2
  exit 1
}
patch_abs="$(cd "$(dirname "$gvisor_patch_path")" && pwd)/$(basename "$gvisor_patch_path")"

work_root=$(mktemp -d "${TMPDIR:-/tmp}/helox-release-gvisor.XXXXXX")
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
git -C "$work_root/gvisor" apply --check "$patch_abs"
git -C "$work_root/gvisor" apply "$patch_abs"
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
test "$("$work_root/bazel" --version)" = "bazel $bazel_version" || {
  echo "pinned Bazel version mismatch" >&2
  exit 1
}

bazel_output_args=""
if [ -n "${BAZEL_OUTPUT_USER_ROOT:-}" ]; then
  bazel_output_args="--output_user_root=${BAZEL_OUTPUT_USER_ROOT}"
fi
(
  cd "$work_root/gvisor"
  # shellcheck disable=SC2086
  "$work_root/bazel" $bazel_output_args build -c opt //:release
)

built_directory="$work_root/gvisor/bazel-bin/release"
test -d "$built_directory"
expected_paths=$(jq -er '.gvisor.runtime_bundle.members[].path' "$runtime_lock")
actual_paths=$(cd "$built_directory" && find . -type f -printf '%P\n' | LC_ALL=C sort)
test "$actual_paths" = "$expected_paths" || {
  echo "gVisor release output inventory differs from locked bundle" >&2
  exit 1
}

mkdir -p "$output_directory/gvisor-bin"
jq -er '.gvisor.runtime_bundle.members[] | "\(.path) \(.size) \(.sha512)"' "$runtime_lock" |
while IFS=' ' read -r member_path member_size member_sha512; do
  case "$member_path" in
    runsc|containerd-shim-runsc-v1|gvisor-bin/checkpointgofer|gvisor-bin/gvisor-sentry-prewarmer|gvisor-bin/gvisor_sentry|gvisor-bin/runsc-metric-server) ;;
    *) echo "runtime lock bundle member path is invalid" >&2; exit 1 ;;
  esac
  source_path="$built_directory/$member_path"
  test -f "$source_path" && test ! -L "$source_path"
  test "$(wc -c <"$source_path" | tr -d ' ')" = "$member_size"
  test "$(sha512sum "$source_path" | awk '{print $1}')" = "$member_sha512"
  install -m 0755 "$source_path" "$output_directory/$member_path"
done

# Required observer capabilities are verified from the built, locked runsc.
meta_output="$("$output_directory/runsc" trace metadata)"
for point in syscall/open_result sentry/mount_topology_snapshot sentry/mount_topology_mutation; do
  case "$meta_output" in *"$point"*) ;; *) echo "built runsc missing $point" >&2; exit 1 ;; esac
done

umask 022
jq -cn \
  --arg commit "$gvisor_commit" \
  --arg patch "$gvisor_patch_sha256" \
  --arg bazel_version "$bazel_version" \
  --arg bazel_sha512 "$bazel_sha512" \
  --arg architecture "$bundle_architecture" \
  --argjson members "$(jq -c '.gvisor.runtime_bundle.members' "$runtime_lock")" \
  '{schema_version: 1, architecture: $architecture, gvisor_commit: $commit, gvisor_patch_sha256: $patch, bazel_version: $bazel_version, bazel_binary_sha512: $bazel_sha512, members: $members}' \
  >"$output_directory/gvisor-bundle.manifest.json"
