#!/usr/bin/env sh
# Prepare the exact upstream source and HAA patch, then select its SDK from the
# HAA Go owner. CI, observer and complete-bundle builds share this preparation.
set -eu

test "$(uname -s)" = Linux && test "$(uname -m)" = x86_64 || {
  echo "gVisor source SDK preparation supports Linux amd64 builders" >&2
  exit 1
}

if [ "$#" -ne 2 ]; then
  echo "usage: $0 RUNTIME_LOCK ABSOLUTE_NEW_SOURCE_DIRECTORY" >&2
  exit 2
fi
runtime_lock=$1
source_directory=$2
case "$source_directory" in /*) ;; *) echo "gVisor source directory must be absolute" >&2; exit 2 ;; esac
test ! -e "$source_directory" || { echo "gVisor source destination already exists" >&2; exit 1; }
preparation_manifest="$source_directory.haa-source.json"
sdk_patch_original="$source_directory.haa-sdk-patch.upstream"
test ! -e "$preparation_manifest" && test ! -e "$sdk_patch_original" || { echo "gVisor preparation evidence already exists" >&2; exit 1; }

gvisor_repository=$(jq -er '.gvisor.source_repository' "$runtime_lock")
gvisor_commit=$(jq -er '.gvisor.commit' "$runtime_lock")
gvisor_patch_path=$(jq -er '.gvisor.patch.path' "$runtime_lock")
gvisor_patch_sha256=$(jq -er '.gvisor.patch.sha256' "$runtime_lock")
selected_go=$(jq -er '.go_image.go_version' "$runtime_lock")
go_image=$(jq -er '.go_image.reference' "$runtime_lock")
test "$gvisor_repository" = https://github.com/google/gvisor.git || { echo "unexpected gVisor source repository" >&2; exit 1; }
printf '%s\n' "$gvisor_commit" | LC_ALL=C grep -Eq '^[0-9a-f]{40}$' || { echo "invalid gVisor commit" >&2; exit 1; }
printf '%s\n' "$gvisor_patch_sha256" | LC_ALL=C grep -Eq '^[0-9a-f]{64}$' || { echo "invalid gVisor patch identity" >&2; exit 1; }
printf '%s\n' "$selected_go" | LC_ALL=C grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$' || { echo "invalid selected Go owner" >&2; exit 1; }
case "$go_image" in "golang:$selected_go-bookworm@sha256:"*) ;; *) echo "partial selected Go image/version input" >&2; exit 1 ;; esac
go_image_digest=${go_image##*@sha256:}
printf '%s\n' "$go_image_digest" | LC_ALL=C grep -Eq '^[0-9a-f]{64}$' || { echo "invalid selected Go image digest" >&2; exit 1; }
test "$go_image" = "golang:$selected_go-bookworm@sha256:$go_image_digest"
case "$gvisor_patch_path" in tools/gvisor/*.patch) ;; *) echo "invalid gVisor patch path" >&2; exit 1 ;; esac
case "$gvisor_patch_path" in *..*) echo "gVisor patch path cannot escape" >&2; exit 1 ;; esac
test -f "$gvisor_patch_path" && test ! -L "$gvisor_patch_path" || { echo "gVisor patch is not a regular source file" >&2; exit 1; }
test "$(sha256sum "$gvisor_patch_path" | awk '{print $1}')" = "$gvisor_patch_sha256" || { echo "gVisor patch sha256 mismatch" >&2; exit 1; }
patch_absolute="$(cd "$(dirname "$gvisor_patch_path")" && pwd)/$(basename "$gvisor_patch_path")"

git init --quiet "$source_directory"
git -C "$source_directory" fetch --quiet --depth=1 --no-tags "${GVISOR_LOCAL_REPO:-$gvisor_repository}" "$gvisor_commit"
git -C "$source_directory" checkout --quiet --detach FETCH_HEAD
test "$(git -C "$source_directory" rev-parse HEAD)" = "$gvisor_commit"
test -z "$(git -C "$source_directory" tag --contains HEAD)"
test -z "$(git -C "$source_directory" status --porcelain)"
git -C "$source_directory" apply --check "$patch_absolute"
git -C "$source_directory" apply "$patch_absolute"
test -n "$(git -C "$source_directory" status --porcelain)" || { echo "gVisor patch produced no changes" >&2; exit 1; }

# This edits one known field in an exact, trusted upstream control file. Unknown
# selectors/grammar are rejected; arbitrary user go.mod files are not rewritten.
module_path="$source_directory/go.mod"
test "$(LC_ALL=C grep -Ec '^go [0-9]+\.[0-9]+\.[0-9]+$' "$module_path")" = 1 || { echo "unknown gVisor Go directive" >&2; exit 1; }
test "$(LC_ALL=C grep -Ec '^[[:space:]]*go[[:space:]]' "$module_path")" = 1
if LC_ALL=C grep -Eq '^[[:space:]]*toolchain[[:space:]]' "$module_path"; then
  echo "unexpected independent gVisor toolchain selector" >&2
  exit 1
fi
upstream_go=$(awk '$1 == "go" {print $2}' "$module_path")
awk -v upstream="$upstream_go" -v selected="$selected_go" 'BEGIN {
  split(upstream, a, "."); split(selected, b, ".")
  for (i=1; i<=3; i++) { if (b[i]+0 > a[i]+0) exit 0; if (b[i]+0 < a[i]+0) exit 1 }
}' || { echo "selected Go is below the pinned gVisor minimum" >&2; exit 1; }
test -f "$source_directory/MODULE.bazel"
grep -Fq 'go_sdk.from_file(' "$source_directory/MODULE.bazel" || { echo "unknown gVisor SDK selector" >&2; exit 1; }
grep -Fq 'go_mod = "//:go.mod"' "$source_directory/MODULE.bazel" || { echo "gVisor SDK is not owned by go.mod" >&2; exit 1; }
sdk_patch="$source_directory/tools/go_types_memoize.patch"
test -f "$sdk_patch" && test ! -L "$sdk_patch"
git -C "$source_directory" show "$gvisor_commit:tools/go_types_memoize.patch" >"$sdk_patch_original"
upstream_sdk_patch_sha256=$(sha256sum "$sdk_patch_original" | awk '{print $1}')
test "$(sha256sum "$sdk_patch" | awk '{print $1}')" = "$upstream_sdk_patch_sha256" || { echo "unreviewed gVisor SDK patch mutation" >&2; exit 1; }
module_temporary=$(mktemp "$source_directory/go.mod.haa-sdk.XXXXXX")
awk -v selected="$selected_go" '/^go [0-9]+\.[0-9]+\.[0-9]+$/ { print "go " selected; next } { print }' "$module_path" >"$module_temporary"
chmod --reference="$module_path" "$module_temporary"
mv "$module_temporary" "$module_path"
test "$(awk '$1 == "go" {print $2}' "$module_path")" = "$selected_go"
expected_stamp="$(printf %.12s "$gvisor_commit")-dirty"
test "$(git -C "$source_directory" describe --always --tags --abbrev=12 --dirty)" = "$expected_stamp"
jq -cn --arg commit "$gvisor_commit" --arg patch "$gvisor_patch_sha256" \
  --arg selected_go "$selected_go" --arg upstream_go "$upstream_go" \
  --arg sdk_patch "$upstream_sdk_patch_sha256" \
  --arg module "$(sha256sum "$module_path" | awk '{print $1}')" \
  '{schema_version: 1, gvisor_commit: $commit, haa_patch_sha256: $patch, selected_go: $selected_go, upstream_go_minimum: $upstream_go, sdk_patch_sha256: $sdk_patch, prepared_go_mod_sha256: $module}' \
  >"$preparation_manifest"
