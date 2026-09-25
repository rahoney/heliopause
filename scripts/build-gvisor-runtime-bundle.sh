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
gvisor_module_lock_sha256=$(jq -er '.gvisor.build.bazel_module_lock_sha256' "$runtime_lock")
builder_repository=$(jq -er '.gvisor.build.builder.image_repository' "$runtime_lock")
builder_tag=$(jq -er '.gvisor.build.builder.image_tag' "$runtime_lock")
builder_digest=$(jq -er '.gvisor.build.builder.image_digest' "$runtime_lock")
builder_architecture=$(jq -er '.gvisor.build.builder.architecture' "$runtime_lock")
bundle_architecture=$(jq -er '.gvisor.runtime_bundle.architecture' "$runtime_lock")
bazel_sha512=$(jq -er '.bazel.linux_x86_64_sha512' "$runtime_lock")
bazel_version=$(jq -er '.bazel.version' "$runtime_lock")

test "$bundle_architecture" = amd64
test "$builder_architecture" = amd64
test "$builder_repository" = us-central1-docker.pkg.dev/gvisor-presubmit/gvisor-presubmit-images/default_x86_64 || {
  echo "gVisor builder repository is not the upstream default amd64 image" >&2
  exit 1
}
case "$builder_digest" in sha256:????????????????????????????????????????????????????????????????) ;; *) echo "gVisor builder digest is invalid" >&2; exit 1 ;; esac
case "$builder_tag" in ????????????????) ;; *) echo "gVisor builder tag is invalid" >&2; exit 1 ;; esac
case "${builder_digest#sha256:}" in *[!0-9a-f]* ) echo "gVisor builder digest is invalid" >&2; exit 1 ;; esac
case "$builder_tag" in *[!0-9a-f]* ) echo "gVisor builder tag is invalid" >&2; exit 1 ;; esac
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
cleanup() {
  chmod -R u+w "$work_root" 2>/dev/null || true
  rm -rf "$work_root"
}
trap cleanup EXIT HUP INT TERM

gvisor_source="${GVISOR_LOCAL_REPO:-$gvisor_repository}"
# A depth-one, tagless checkout fixes the stamped source description to the
# pinned commit. Full-clone tag reachability is mutable repository metadata
# and must not influence an exact runtime identity.
git init "$work_root/gvisor" >/dev/null
git -C "$work_root/gvisor" fetch --quiet --depth=1 --no-tags "$gvisor_source" "$gvisor_commit"
git -C "$work_root/gvisor" checkout --quiet --detach FETCH_HEAD
test "$(git -C "$work_root/gvisor" rev-parse HEAD)" = "$gvisor_commit"
test -z "$(git -C "$work_root/gvisor" tag --contains HEAD)" || {
  echo "canonical gVisor source checkout unexpectedly contains tags" >&2
  exit 1
}
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
expected_stamp="$(printf %.12s "$gvisor_commit")-dirty"
actual_stamp="$(git -C "$work_root/gvisor" describe --always --tags --abbrev=12 --dirty)"
test "$actual_stamp" = "$expected_stamp" || {
  echo "gVisor stamped version mismatch: expected=$expected_stamp actual=$actual_stamp" >&2
  exit 1
}

# Match upstream tools/images.mk:tag for the pinned images/default tree, then
# require its published tag to resolve to the locked immutable amd64 manifest.
actual_builder_tag="$(cd "$work_root/gvisor/images" && find default -type f | LC_ALL=C sort -f -d | xargs -n 1 sha256sum | sha256sum - | cut -c 1-16)"
test "$actual_builder_tag" = "$builder_tag" || {
  echo "gVisor upstream builder definition mismatch: expected=$builder_tag actual=$actual_builder_tag" >&2
  exit 1
}
if published_builder_digest="$(docker manifest inspect --verbose "$builder_repository:$builder_tag" 2>/dev/null | jq -er '.Descriptor.digest' 2>/dev/null)"; then
  test "$published_builder_digest" = "$builder_digest" || {
    echo "gVisor published builder digest mismatch: expected=$builder_digest actual=$published_builder_digest" >&2
    exit 1
  }
fi
builder_ref="$builder_repository@$builder_digest"
if ! docker image inspect "$builder_ref" >/dev/null 2>&1; then
  docker pull --platform linux/amd64 "$builder_ref"
fi
test "$(docker image inspect --format '{{.Os}}/{{.Architecture}}' "$builder_ref")" = linux/amd64 || {
  echo "gVisor builder architecture mismatch" >&2
  exit 1
}
docker image inspect --format '{{json .RepoDigests}}' "$builder_ref" | jq -e --arg ref "$builder_ref" 'index($ref) != null' >/dev/null || {
  echo "gVisor local builder digest mismatch: expected=$builder_ref" >&2
  exit 1
}

docker run --rm --pull=never --platform linux/amd64 \
  --user "$(id -u):$(id -g)" --env "HOME=$work_root" --env USER=builder \
  --env "EXPECTED_BAZEL_VERSION=$bazel_version" \
  --env "EXPECTED_BAZEL_SHA512=$bazel_sha512" \
  --env "EXPECTED_STAMP=$expected_stamp" \
  --mount "type=bind,src=$work_root,dst=$work_root" \
  --workdir "$work_root/gvisor" --entrypoint /bin/sh "$builder_ref" -c '
    set -eu
    test "$(/usr/local/bin/bazel --version)" = "bazel $EXPECTED_BAZEL_VERSION"
    test "$(sha512sum /usr/local/bin/bazel | awk "{print \$1}")" = "$EXPECTED_BAZEL_SHA512"
    test "$(git describe --always --tags --abbrev=12 --dirty)" = "$EXPECTED_STAMP"
    /usr/local/bin/bazel build -c opt //:release
  '

module_lock_path="$work_root/gvisor/MODULE.bazel.lock"
test -f "$module_lock_path" || {
  echo "gVisor Bazel module lock was not produced" >&2
  exit 1
}
actual_module_lock_sha256="$(sha256sum "$module_lock_path" | awk '{print $1}')"
test "$actual_module_lock_sha256" = "$gvisor_module_lock_sha256" || {
  echo "gVisor Bazel module lock sha256 mismatch: expected=$gvisor_module_lock_sha256 actual=$actual_module_lock_sha256" >&2
  exit 1
}

sh scripts/verify-gvisor-runtime-bundle.sh "$runtime_lock" "$work_root/gvisor/bazel-bin/release" "$output_directory"

# Required observer capabilities are verified from the built, locked runsc.
meta_output="$("$output_directory/runsc" trace metadata)"
for point in syscall/open_result sentry/mount_topology_snapshot sentry/mount_topology_mutation; do
  case "$meta_output" in *"$point"*) ;; *) echo "built runsc missing $point" >&2; exit 1 ;; esac
done

umask 022
jq -cn \
  --arg commit "$gvisor_commit" \
  --arg patch "$gvisor_patch_sha256" \
  --arg bazel_module_lock_sha256 "$gvisor_module_lock_sha256" \
  --arg bazel_version "$bazel_version" \
  --arg bazel_sha512 "$bazel_sha512" \
  --arg architecture "$bundle_architecture" \
  --argjson members "$(jq -c '.gvisor.runtime_bundle.members' "$runtime_lock")" \
  --arg builder_repository "$builder_repository" \
  --arg builder_tag "$builder_tag" \
  --arg builder_digest "$builder_digest" \
  --arg builder_architecture "$builder_architecture" \
  '{schema_version: 2, architecture: $architecture, gvisor_commit: $commit, gvisor_patch_sha256: $patch, bazel_module_lock_sha256: $bazel_module_lock_sha256, builder_image_repository: $builder_repository, builder_image_tag: $builder_tag, builder_image_digest: $builder_digest, builder_architecture: $builder_architecture, bazel_version: $bazel_version, bazel_binary_sha512: $bazel_sha512, members: $members}' \
  >"$output_directory/gvisor-bundle.manifest.json"
