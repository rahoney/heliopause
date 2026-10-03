#!/usr/bin/env sh
# Verify and copy the six release members using the canonical runtime lock.
# Kept separate from the builder so failure behavior can be exercised with
# small local fixtures without running Bazel or Docker.
set -eu

if [ "$#" -ne 3 ]; then
  echo "usage: $0 RUNTIME_LOCK BUILT_DIRECTORY OUTPUT_DIRECTORY" >&2
  exit 2
fi
runtime_lock=$1
built_directory=$2
output_directory=$3

test -d "$built_directory" || { echo "gVisor release output directory missing: $built_directory" >&2; exit 1; }
test "$(jq -er '.gvisor.runtime_bundle.members | length' "$runtime_lock")" = 6 || {
  echo "gVisor runtime lock must contain six members" >&2
  exit 1
}
expected_paths=$(jq -er '.gvisor.runtime_bundle.members[].path' "$runtime_lock" | LC_ALL=C sort)
actual_paths=$(cd "$built_directory" && find . ! -type d -print | sed 's#^\./##' | LC_ALL=C sort)
test "$actual_paths" = "$expected_paths" || {
  echo "gVisor release output inventory mismatch: expected=[$expected_paths] actual=[$actual_paths]" >&2
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
  test -f "$source_path" || { echo "gVisor bundle member missing: path=$member_path" >&2; exit 1; }
  test ! -L "$source_path" || { echo "gVisor bundle member is a symlink: path=$member_path" >&2; exit 1; }
  actual_member_size="$(wc -c <"$source_path" | tr -d ' ')"
  test "$actual_member_size" = "$member_size" || {
    echo "gVisor bundle member size mismatch: path=$member_path expected=$member_size actual=$actual_member_size" >&2
    exit 1
  }
  actual_member_sha512="$(shasum -a 512 "$source_path" | awk '{print $1}')"
  test "$actual_member_sha512" = "$member_sha512" || {
    echo "gVisor bundle member sha512 mismatch: path=$member_path expected=$member_sha512 actual=$actual_member_sha512" >&2
    exit 1
  }
  install -m 0755 "$source_path" "$output_directory/$member_path"
done
