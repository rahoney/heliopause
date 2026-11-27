#!/bin/bash
# Sourced by the hosted CI parent and its existing native fixture children.
# This bounded runtime source lives for the job; it is not a release install.
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
  [[ $members == $'current\nversions\nversions/v0.0.0-responder-ci\nversions/v0.0.0-responder-ci/haa-sandbox-responder' ]]
}

haa_ci_prepare_responder_base() {
  [[ $# == 0 && ${RUNNER_TEMP-} == /* ]] || return 1
  [[ ! -e /usr/lib/heliopause && ! -L /usr/lib/heliopause ]] || return 1
  sudo -n mkdir -m 0755 -- /usr/lib/heliopause || return 1
  sudo -n install -d -m 0755 /usr/lib/heliopause/versions/v0.0.0-responder-ci || return 1
  sudo -n install -m 0555 -- "$RUNNER_TEMP/haa-sandbox-responder" /usr/lib/heliopause/versions/v0.0.0-responder-ci/haa-sandbox-responder || return 1
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
