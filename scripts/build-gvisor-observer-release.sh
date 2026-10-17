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
bazel_sha512=$(jq -er '.bazel.linux_x86_64_sha512' "$runtime_lock")
bazel_version=$(jq -er '.bazel.version' "$runtime_lock")
builder_repository=$(jq -er '.gvisor.build.builder.image_repository' "$runtime_lock")
builder_digest=$(jq -er '.gvisor.build.builder.image_digest' "$runtime_lock")
test "$builder_repository" = us-central1-docker.pkg.dev/gvisor-presubmit/gvisor-presubmit-images/default_x86_64
printf '%s\n' "$builder_digest" | LC_ALL=C grep -Eq '^sha256:[0-9a-f]{64}$'


work_root=$(mktemp -d "${TMPDIR:-/tmp}/helox-release-observer.XXXXXX")
cleanup() {
  status=$1
  if [ "$status" -ne 0 ]; then
    echo "observer failed build inputs retained: $work_root" >&2
    return "$status"
  fi
  rm -rf "$work_root"
}
trap 'cleanup "$?"' EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

sh scripts/prepare-gvisor-build-source.sh "$runtime_lock" "$work_root/gvisor"

cp -R tools/gvisor-observer "$work_root/gvisor/tools/haa_gvisor_observer"
builder_ref="$builder_repository@$builder_digest"
if ! docker image inspect "$builder_ref" >/dev/null 2>&1; then
  docker pull --platform linux/amd64 "$builder_ref"
fi
test "$(docker image inspect --format '{{.Os}}/{{.Architecture}}' "$builder_ref")" = linux/amd64
docker image inspect --format '{{json .RepoDigests}}' "$builder_ref" | jq -e --arg ref "$builder_ref" 'index($ref) != null' >/dev/null
docker run --rm --pull=never --platform linux/amd64 \
  --user "$(id -u):$(id -g)" --env "HOME=$work_root" --env USER=builder \
  --env "EXPECTED_BAZEL_VERSION=$bazel_version" --env "EXPECTED_BAZEL_SHA512=$bazel_sha512" \
  --mount "type=bind,src=$work_root,dst=$work_root" \
  --workdir "$work_root/gvisor" --entrypoint /bin/sh "$builder_ref" -c '
    set -eu
    test "$(/usr/local/bin/bazel --version)" = "bazel $EXPECTED_BAZEL_VERSION"
    test "$(sha512sum /usr/local/bin/bazel | awk "{print \$1}")" = "$EXPECTED_BAZEL_SHA512"
    /usr/local/bin/bazel build -c opt --jobs=4 --local_ram_resources=8192 //tools/haa_gvisor_observer:haa_gvisor_observer //tools/haa_gvisor_observer:haa_gvisor_observer_latch_test
    bazel-bin/tools/haa_gvisor_observer/haa_gvisor_observer_latch_test
  '

test ! -e "$output_path"
install -m 0755 "$work_root/gvisor/bazel-bin/tools/haa_gvisor_observer/haa_gvisor_observer" "$output_path"
