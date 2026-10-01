#!/usr/bin/env bash
# Sourced in the owning shell. CI launches foreground sudo (never sudo -b):
# sudo forwards TERM and returns the command's status; only wait=0 proves a
# graceful end of that fixed, non-daemonizing helper lifecycle.
haa_ci_initialize() {
  if [[ ${haa_initialized-0} != 0 ]]; then echo 'cleanup: duplicate initialization' >&2; return 1; fi
  if [[ -n ${policy_helper_pid-} || -n ${haa_recorded_pid-} || ${haa_helper_error-0} != 0 || $BASH_SUBSHELL != 0 ]]; then
    haa_helper_unknown 1 'initialization conflicts with existing lifecycle evidence'; return $?
  fi
  haa_initialized=1; haa_owner=$$; haa_helper_state=NOT_STARTED
  haa_ever_started=0; haa_registered=0; haa_reaped=0
  haa_recorded_pid=''; policy_helper_pid=''
  haa_helper_error=0; haa_client_error=0
  haa_client_restore_required=0; haa_replacement_recorded=0
}

haa_helper_unknown() {
  local status=$1
  haa_helper_state=UNKNOWN
  if [[ ${haa_helper_error-0} == 0 ]]; then haa_helper_error=$status; fi
  printf 'cleanup: UNKNOWN: %s (status=%s pid=%s)\n' "$2" "$status" "${policy_helper_pid-}" >&2
  return "$haa_helper_error"
}

haa_observe_policy_helper() {
  local listing status line found=0 seen=' '
  if [[ ${haa_initialized-0} != 1 || ${haa_owner-} != "$$" || $BASH_SUBSHELL != 0 ]]; then
    haa_helper_unknown 1 'uninitialized or non-owning shell'; return $?
  fi
  if [[ ${haa_helper_error-0} != 0 ]]; then return "$haa_helper_error"; fi
  if [[ ${haa_ever_started-} == 0 && ${haa_registered-} == 0 && -z ${policy_helper_pid-} && -z ${haa_recorded_pid-} && ${haa_helper_state-} == NOT_STARTED ]]; then return 0; fi
  if [[ ${haa_registered-} != 1 || ${haa_ever_started-} != 1 || ! ${haa_recorded_pid-} =~ ^[1-9][0-9]*$ || ${policy_helper_pid-} != "${haa_recorded_pid-}" ]]; then
    haa_helper_unknown 1 'missing, invalid or unowned lifecycle handle'; return $?
  fi
  if (( haa_recorded_pid <= 1 )); then haa_helper_unknown 1 'invalid lifecycle PID'; return $?; fi
  if [[ ${haa_helper_state-} == CONFIRMED_STOPPED && ${haa_reaped-} == 1 ]]; then return 0; fi
  # Capture the producer status directly: no pipeline or matcher can turn an
  # observation error into a successful no-match. jobs -p includes stopped jobs.
  if listing=$(jobs -p); then :; else
    status=$?; haa_helper_unknown "$status" 'job-table query failed'; return $?
  fi
  if [[ -n "$listing" ]]; then
    while IFS= read -r line; do
      if [[ ! "$line" =~ ^[1-9][0-9]*$ || "$seen" == *" $line "* ]]; then
        haa_helper_unknown 1 'malformed job-table output'; return $?
      fi
      seen+="$line "
      if [[ "$line" == "$haa_recorded_pid" ]]; then found=1; fi
    done <<< "$listing"
  fi
  if (( found == 1 )); then haa_helper_state=OWNED_LIVE; return 0; fi
  # Absence from a successful snapshot is insufficient. Reap in this parent,
  # not in the query subshell. A failed/signalled sudo wrapper is uncertain.
  if wait "$haa_recorded_pid"; then
    haa_reaped=1; haa_helper_state=CONFIRMED_STOPPED; return 0
  else
    status=$?; haa_helper_unknown "$status" 'owned foreground wrapper did not complete cleanly'; return $?
  fi
}

haa_prepare_policy_helper_launch() {
  if haa_stop_policy_helper; then :; else return $?; fi
  if [[ ${haa_client_error-0} != 0 ]]; then return "$haa_client_error"; fi
}

haa_record_policy_helper_launch() {
  local launched=${!-}
  if [[ ${haa_initialized-0} != 1 || ${haa_owner-} != "$$" || $BASH_SUBSHELL != 0 || ${haa_helper_error-0} != 0 || ${haa_client_error-0} != 0 || ( ${haa_helper_state-} != NOT_STARTED && ${haa_helper_state-} != CONFIRMED_STOPPED ) ]]; then
    haa_helper_unknown 1 'launch without a clean owning lifecycle'; return $?
  fi
  if [[ ${haa_ever_started-0} == 1 && "$launched" == "${haa_recorded_pid-}" ]]; then
    haa_helper_unknown 1 'stale launch handle'; return $?
  fi
  haa_ever_started=1; haa_registered=1; haa_reaped=0
  haa_recorded_pid=$launched; policy_helper_pid=$launched; haa_helper_state=OWNED_LIVE
  haa_observe_policy_helper
}

haa_stop_policy_helper() {
  local attempt status
  if haa_observe_policy_helper; then :; else return $?; fi
  if [[ "$haa_helper_state" == NOT_STARTED || "$haa_helper_state" == CONFIRMED_STOPPED ]]; then return 0; fi
  if sudo kill -TERM "$haa_recorded_pid"; then :; else
    status=$?; haa_helper_unknown "$status" 'termination request failed'; return $?
  fi
  for ((attempt=0; attempt<50; attempt++)); do
    if haa_observe_policy_helper; then :; else return $?; fi
    if [[ "$haa_helper_state" == CONFIRMED_STOPPED ]]; then return 0; fi
    if sleep 0.1; then :; else status=$?; haa_helper_unknown "$status" 'confirmation delay failed'; return $?; fi
  done
  haa_helper_unknown 1 'helper remains active after bounded wait'
}

haa_install_ci_client() {
  local status
  if haa_stop_policy_helper; then :; else return $?; fi
  if [[ ${haa_client_error-0} != 0 ]]; then return "$haa_client_error"; fi
  # Record possible replacement before install: even a failed install can have
  # changed the path. EXIT may restore only after a proven safe helper state.
  haa_client_restore_required=1; haa_replacement_recorded=1
  if sudo install -m 0755 "$1" /usr/libexec/heliopause/helox; then :; else
    status=$?; haa_client_error=$status; return "$status"
  fi
  if [[ "$1" == "$RUNNER_TEMP/helox" ]]; then haa_client_restore_required=0; fi
}

haa_ci_finish() {
  local original=$1 stop_status=0 restore_status=0 restore_result=NOT_REQUIRED
  trap - EXIT
  if haa_stop_policy_helper; then :; else stop_status=$?; fi
  if [[ ${haa_client_restore_required-0} == 1 ]]; then
    if (( stop_status == 0 )) && [[ ${haa_initialized-0} == 1 && ${haa_owner-} == "$$" && $BASH_SUBSHELL == 0 && ${haa_replacement_recorded-0} == 1 && -n ${RUNNER_TEMP-} && ( ${haa_helper_state-} == NOT_STARTED || ${haa_helper_state-} == CONFIRMED_STOPPED ) ]]; then
      if sudo install -m 0755 "$RUNNER_TEMP/helox" /usr/libexec/heliopause/helox; then
        haa_client_restore_required=0; restore_result=RESTORED
      else restore_status=$?; restore_result=FAILED; fi
    else
      restore_status=1; restore_result=SKIPPED_UNSAFE
      echo 'cleanup: production restoration incomplete; helper/replacement identity is uncertain' >&2
    fi
  fi
  printf 'cleanup: original=%s helper_state=%s helper_stop=%s client_error=%s client_restore=%s restore_result=%s pid=%s\n' "$original" "${haa_helper_state-UNKNOWN}" "$stop_status" "${haa_client_error-0}" "$restore_status" "$restore_result" "${policy_helper_pid-}" >&2
  if (( original != 0 )); then exit "$original"; fi
  if (( stop_status != 0 )); then exit "$stop_status"; fi
  if [[ ${haa_client_error-0} != 0 ]]; then exit "$haa_client_error"; fi
  if (( restore_status != 0 )); then exit "$restore_status"; fi
  exit 0
}
trap 'haa_ci_finish "$?"' EXIT
