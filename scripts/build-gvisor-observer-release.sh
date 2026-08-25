#!/usr/bin/env sh
# Build the release observer against the exact gVisor/Bazel identities in the
# canonical runtime lock. This script intentionally produces one Linux amd64
# helper and does not install or activate it on the Host.
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 ABSOLUTE_OUTPUT_PATH" >&2
  exit 2
fi

output_path=$1
case "$output_path" in
  /*) ;;
  *) echo "release observer output must be absolute" >&2; exit 2 ;;
esac

runtime_lock=scripts/runtimes.lock.json
test -f "$runtime_lock"
gvisor_repository=$(jq -er '.gvisor.source_repository' "$runtime_lock")
gvisor_commit=$(jq -er '.gvisor.commit' "$runtime_lock")
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

work_root=$(mktemp -d "${TMPDIR:-/tmp}/helox-release-observer.XXXXXX")
cleanup() { rm -rf "$work_root"; }
trap cleanup EXIT HUP INT TERM

git clone --filter=blob:none "$gvisor_repository" "$work_root/gvisor"
git -C "$work_root/gvisor" checkout --detach "$gvisor_commit"
test "$(git -C "$work_root/gvisor" rev-parse HEAD)" = "$gvisor_commit"

curl --fail --location --silent --show-error --output "$work_root/bazel" "$bazel_url"
test "$(sha512sum "$work_root/bazel" | awk '{print $1}')" = "$bazel_sha512"
chmod 0755 "$work_root/bazel"

cp -R tools/gvisor-observer "$work_root/gvisor/tools/haa_gvisor_observer"
(
  cd "$work_root/gvisor"
  "$work_root/bazel" build -c opt //tools/haa_gvisor_observer:haa_gvisor_observer
)

test ! -e "$output_path"
install -m 0755 "$work_root/gvisor/bazel-bin/tools/haa_gvisor_observer/haa_gvisor_observer" "$output_path"
