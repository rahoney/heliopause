#!/bin/bash
# Sourced by the hosted CI parent and its existing native fixture children.
# This bounded runtime source lives for the job; it is not a release install.
haa_ci_semgrep_lock=$(cd -- "${BASH_SOURCE[0]%/*}" && pwd -P)/runtimes.lock.json

# Self-built Semgrep core travels as a Release asset named by the selected lock
# owner; rules come from the checkout. Both are pinned by that same owner.
haa_ci_fetch_semgrep_supply() {
  [[ $# == 0 && ${RUNNER_TEMP-} == /* && ${GITHUB_REPOSITORY-} == */* ]] || return 1
  local version core_sha core_size
  version=$(jq -er '.semgrep_tool.semgrep_version' "$haa_ci_semgrep_lock") || return 1
  core_sha=$(jq -er '.semgrep_tool.tool.sha256' "$haa_ci_semgrep_lock") || return 1
  core_size=$(jq -er '.semgrep_tool.tool.size' "$haa_ci_semgrep_lock") || return 1
  curl --fail --show-error --silent --location --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 300 --max-filesize "$core_size" \
    --output "$RUNNER_TEMP/semgrep-core-linux-amd64" \
    "https://github.com/$GITHUB_REPOSITORY/releases/download/semgrep-$version-${core_sha:0:12}/semgrep-core-linux-amd64" || return 1
  install -m 0444 -- "${haa_ci_semgrep_lock%/scripts/*}/tools/semgrep/rules.json" "$RUNNER_TEMP/semgrep-rules.json"
}

haa_ci_semgrep_source() {
  [[ $# == 1 ]] || return 1
  local name mode owner installed metadata size hash
  case "$1" in
    core) name=semgrep-core-linux-amd64 mode=555 owner=tool ;;
    rules) name=semgrep-rules.json mode=444 owner=rules ;;
    *) return 1 ;;
  esac
  installed=/usr/lib/heliopause/versions/v0.0.0-responder-ci/$name
  [[ -f $installed && ! -L $installed ]] || return 1
  size=$(jq -er ".semgrep_tool.$owner.size" "$haa_ci_semgrep_lock") || return 1
  metadata=$(stat -c '%u:%g:%a:%h:%s' -- "$installed") || return 1
  [[ $metadata == "0:0:$mode:1:$size" ]] || return 1
  hash=$(sha256sum -- "$installed") || return 1
  [[ ${hash%% *} == "$(jq -er ".semgrep_tool.$owner.sha256" "$haa_ci_semgrep_lock")" ]] || return 1
  printf '%s\n' "$installed"
}

haa_ci_responder_source() {
  [[ $# == 0 && ${RUNNER_TEMP-} == /* ]] || return 1
  local directory metadata installed compiled installed_hash compiled_hash
  installed=/usr/lib/heliopause/versions/v0.0.0-responder-ci/haa-sandbox-responder
  compiled=$RUNNER_TEMP/haa-sandbox-responder
  for directory in /usr/lib/heliopause /usr/lib/heliopause/versions /usr/lib/heliopause/versions/v0.0.0-responder-ci; do
    [[ -d $directory && ! -L $directory ]] || return 1
    metadata=$(stat -c '%u:%g:%a' -- "$directory") || return 1
    [[ $metadata == 0:0:755 ]] || return 1
  done
  [[ -f $installed && ! -L $installed && -f $compiled && ! -L $compiled ]] || return 1
  metadata=$(stat -c '%u:%g:%a:%h' -- "$installed") || return 1
  [[ $metadata == 0:0:555:1 ]] || return 1
  installed_hash=$(sha256sum -- "$installed") || return 1
  compiled_hash=$(sha256sum -- "$compiled") || return 1
  [[ ${installed_hash%% *} == "${compiled_hash%% *}" ]] || return 1
  printf '%s\n' "$installed"
}

haa_ci_require_responder_base() {
  [[ $# == 0 ]] || return 1
  haa_ci_responder_source >/dev/null || return 1
  local current members
  current=$(readlink -- /usr/lib/heliopause/current) || return 1
  [[ $current == versions/v0.0.0-responder-ci ]] || return 1
  members=$(find /usr/lib/heliopause -mindepth 1 -maxdepth 3 -printf '%P\n' | LC_ALL=C sort) || return 1
  [[ $members == $'current\nversions\nversions/v0.0.0-responder-ci\nversions/v0.0.0-responder-ci/haa-sandbox-responder\nversions/v0.0.0-responder-ci/semgrep-core-linux-amd64\nversions/v0.0.0-responder-ci/semgrep-rules.json' ]] || return 1
  haa_ci_semgrep_source core >/dev/null && haa_ci_semgrep_source rules >/dev/null
}

haa_ci_prepare_responder_base() {
  [[ $# == 0 && ${RUNNER_TEMP-} == /* ]] || return 1
  [[ ! -e /usr/lib/heliopause && ! -L /usr/lib/heliopause ]] || return 1
  sudo -n mkdir -m 0755 -- /usr/lib/heliopause || return 1
  sudo -n install -d -m 0755 /usr/lib/heliopause/versions/v0.0.0-responder-ci || return 1
  sudo -n install -m 0555 -- "$RUNNER_TEMP/haa-sandbox-responder" /usr/lib/heliopause/versions/v0.0.0-responder-ci/haa-sandbox-responder || return 1
  sudo -n install -m 0555 -- "$RUNNER_TEMP/semgrep-core-linux-amd64" /usr/lib/heliopause/versions/v0.0.0-responder-ci/semgrep-core-linux-amd64 || return 1
  sudo -n install -m 0444 -- "$RUNNER_TEMP/semgrep-rules.json" /usr/lib/heliopause/versions/v0.0.0-responder-ci/semgrep-rules.json || return 1
  sudo -n ln -s versions/v0.0.0-responder-ci /usr/lib/heliopause/current || return 1
  haa_ci_require_responder_base
}

haa_ci_restore_responder_base() {
  [[ $# == 1 ]] || return 1
  case "$1" in v1.0.0-pnpm-ci|v1.0.0-bun-ci) ;; *) return 1 ;; esac
  haa_ci_responder_source >/dev/null || return 1
  local current
  current=$(readlink -- /usr/lib/heliopause/current) || return 1
  [[ $current == versions/"$1" || $current == versions/v0.0.0-responder-ci ]] || return 1
  sudo -n ln -sfnT versions/v0.0.0-responder-ci /usr/lib/heliopause/current
}
