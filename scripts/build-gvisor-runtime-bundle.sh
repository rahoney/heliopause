#!/usr/bin/env sh
# Build the complete patched gVisor runtime bundle against the exact source,
# patch, Bazel, and member identities in the canonical runtime lock. The bundle
# is self-contained: no installation step may download a missing sidecar.
set -eu

test "$(uname -s)" = Linux && test "$(uname -m)" = x86_64 || {
  echo "local gVisor bundle build supports Linux amd64 only" >&2
  exit 1
}

capture=0
if [ "${1:-}" = --capture ]; then
  capture=1
  shift
fi
if [ "$#" -ne 1 ]; then
  echo "usage: $0 [--capture] ABSOLUTE_OUTPUT_DIRECTORY" >&2
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
gvisor_commit=$(jq -er '.gvisor.commit' "$runtime_lock")
gvisor_patch_sha256=$(jq -er '.gvisor.patch.sha256' "$runtime_lock")
gvisor_module_lock_sha256=$(jq -er '.gvisor.build.bazel_module_lock_sha256' "$runtime_lock")
builder_repository=$(jq -er '.gvisor.build.builder.image_repository' "$runtime_lock")
builder_tag=$(jq -er '.gvisor.build.builder.image_tag' "$runtime_lock")
builder_digest=$(jq -er '.gvisor.build.builder.image_digest' "$runtime_lock")
builder_architecture=$(jq -er '.gvisor.build.builder.architecture' "$runtime_lock")
bundle_architecture=$(jq -er '.gvisor.runtime_bundle.architecture' "$runtime_lock")
bazel_sha512=$(jq -er '.bazel.linux_x86_64_sha512' "$runtime_lock")
bazel_version=$(jq -er '.bazel.version' "$runtime_lock")
selected_go=$(jq -er '.go_image.go_version' "$runtime_lock")

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

work_root=$(mktemp -d "${TMPDIR:-/tmp}/helox-release-gvisor.XXXXXX")
cleanup() {
  status=$1
  if [ "$status" -ne 0 ]; then
    echo "gVisor failed build inputs retained: $work_root" >&2
    return "$status"
  fi
  if [ "$capture" -eq 1 ]; then
    echo "gVisor capture inputs retained for frozen-plan verification: $work_root" >&2
    return 0
  fi
  chmod -R u+w "$work_root" 2>/dev/null || true
  rm -rf "$work_root"
}
trap 'cleanup "$?"' EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

sh scripts/prepare-gvisor-build-source.sh "$runtime_lock" "$work_root/gvisor"
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
verification_lock="$runtime_lock"
if [ "$capture" -eq 1 ]; then
  verification_lock="$work_root/candidate-runtime-lock.json"
  GOTOOLCHAIN=local go run ./scripts/update-runtime capture-gvisor \
    --root "$PWD" --built-dir "$work_root/gvisor/bazel-bin/release" \
    --module-lock "$module_lock_path" --source-manifest "$work_root/gvisor.haa-source.json" \
    --output "$verification_lock"
  gvisor_module_lock_sha256=$actual_module_lock_sha256
else
  test "$actual_module_lock_sha256" = "$gvisor_module_lock_sha256" || {
    echo "gVisor Bazel module lock sha256 mismatch: expected=$gvisor_module_lock_sha256 actual=$actual_module_lock_sha256" >&2
    exit 1
  }
fi
sh scripts/verify-gvisor-runtime-bundle.sh "$verification_lock" "$work_root/gvisor/bazel-bin/release" "$output_directory"
for member in containerd-shim-runsc-v1 gvisor-bin/checkpointgofer gvisor-bin/gvisor_sentry gvisor-bin/runsc-metric-server runsc; do
  binary="$output_directory/$member"
  test "$(GOTOOLCHAIN=local go version "$binary")" = "$binary: go$selected_go" || {
    echo "gVisor member compiler mismatch: $member" >&2
    exit 1
  }
done
if [ "$capture" -eq 1 ]; then
  cp "$verification_lock" "$output_directory/candidate-runtime-lock.json"
  cp "$verification_lock.receipt.json" "$output_directory/candidate-runtime-lock.json.receipt.json"
  cp "$module_lock_path" "$output_directory/MODULE.bazel.lock"
  cp "$work_root/gvisor.haa-source.json" "$output_directory/gvisor-source-preparation.json"
fi

# Required observer capabilities are verified from the built, locked runsc.
meta_output="$("$output_directory/runsc" trace metadata)"
for point in syscall/open_result syscall/socketpair_result sentry/mount_topology_snapshot sentry/mount_topology_mutation; do
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
  --argjson members "$(jq -c '.gvisor.runtime_bundle.members' "$verification_lock")" \
  --arg builder_repository "$builder_repository" \
  --arg builder_tag "$builder_tag" \
  --arg builder_digest "$builder_digest" \
  --arg builder_architecture "$builder_architecture" \
  '{schema_version: 2, architecture: $architecture, gvisor_commit: $commit, gvisor_patch_sha256: $patch, bazel_module_lock_sha256: $bazel_module_lock_sha256, builder_image_repository: $builder_repository, builder_image_tag: $builder_tag, builder_image_digest: $builder_digest, builder_architecture: $builder_architecture, bazel_version: $bazel_version, bazel_binary_sha512: $bazel_sha512, members: $members}' \
  >"$output_directory/gvisor-bundle.manifest.json"
