package main

import "strings"

const pipxInstalledNativeCommand = `"${ordinary[@]}" HELOX_PIPX_NATIVE_SESSION_INTEGRATION=1 HELOX_PIPX_NATIVE_INSTALLED_SUPPLY=1 HELOX_PIPX_QUERY_SUPPLY_ROOT="$(dirname -- "$input")/supply" HELOX_PIPX_RUNTIME_ZIP=/usr/lib/heliopause/current/pipx-runtime-linux-amd64.zip HELOX_PIPX_PUBLIC_RESOLVER_INTEGRATION=1 HELOX_PIPX_PUBLIC_UPGRADE_INTEGRATION=1 /usr/libexec/heliopause/helox -test.v -test.failfast -test.timeout=5m -test.run '^(TestLinuxPipxObservedNativeInstallIntegration|TestLinuxPipxPublicInstallResolverIntegration|TestLinuxPipxPublicOriginalUpgradeIntegration)$'`

// This owns concrete CI dispatch/fixture conventions, not artifact admission.
// Actual namespace/ordinary-user/cleanup behavior is qualified separately.
func validatePipxInstalledToolsScript(contents string) []string {
	counts := map[string]int{}
	var active []string
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		counts[line]++
		if line != "" && !strings.HasPrefix(line, "#") {
			active = append(active, line)
		}
	}
	var findings []string
	for _, command := range []string{
		`set -euo pipefail`, `case "$#" in 5|6) ;; *) exit 2 ;; esac`, `case "$1" in tools-native|tools-bootstrap|run) ;; *) exit 2 ;; esac`,
		`test "$(id -u)" = 0`, `[[ "$user_id" =~ ^[0-9]+$ && "$group_id" =~ ^[0-9]+$ ]]`, `test "$user_id" != 0`,
		`test "$(readlink /proc/self/ns/mnt)" != "$(readlink /proc/1/ns/mnt)"`,
		`test "$(readlink /proc/self/ns/net)" = "$(readlink /proc/1/ns/net)"`,
		`test -f "$input"`, `test ! -L "$input"`, `test "$(stat -c '%h' "$input")" = 1`, `test "$(stat -c '%u' "$input")" = "$user_id"`,
		`test ! -e /usr/lib/heliopause`, `test ! -L /usr/lib/heliopause`,
		`expected_sha=$(jq -er '.pipx_tool.tool.sha256' "$repo/scripts/runtimes.lock.json")`,
		`expected_size=$(jq -er '.pipx_tool.tool.size' "$repo/scripts/runtimes.lock.json")`,
		`test "$(stat -c '%s' "$input")" = "$expected_size"`,
		`work=$(mktemp -d "$RUNNER_TEMP/haa-pipx-installed.XXXXXXXX")`,
		`work_identity=$(stat -c '%d:%i:%u:%a' "$work")`,
		`unshare --mount --fork --kill-child /bin/bash "$0" "$@" "$work"`,
		`namespace=$(cat "$work/mount-namespace") || cleanup_status=1`,
		`if [[ ! "$namespace" =~ ^mnt:\[[0-9]+\]$ ]]; then cleanup_status=1; fi`,
		`for path in /proc/[0-9]*/ns/mnt; do`,
		`if current=$(readlink "$path"); then`,
		`if [[ "$current" == "$namespace" ]]; then cleanup_status=1; fi`,
		`elif [[ -d "${path%/ns/mnt}" ]]; then`,
		`if [[ "$(stat -c '%d:%i:%u:%a' "$work")" != "$work_identity" ]]; then cleanup_status=1; fi`,
		`test "$(stat -c '%u:%a' "$work")" = 0:700`,
		`readlink /proc/self/ns/mnt > "$work/mount-namespace"`, `trap cleanup EXIT`,

		`if ((cleanup_status == 0)); then rm -rf -- "$work" || cleanup_status=1; fi`,
		`if ((primary == 0)); then primary=$cleanup_status; fi`, `exit "$primary"`,
		`mount --make-rprivate /`, `mkdir -m 0755 "$work/upper"`, `mkdir -m 0700 "$work/mount-work"`,
		`mount -t overlay overlay -o "lowerdir=/usr/lib,upperdir=$work/upper,workdir=$work/mount-work" /usr/lib`,
		`test "$(stat -c '%a' /usr/lib)" = 755`,
		`install -m 0444 -o 0 -g 0 -- "$input" "$installed"`,
		`test "$(sha256sum "$installed" | cut -d ' ' -f 1)" = "$expected_sha"`,
		`mount --bind /usr/lib/heliopause /usr/lib/heliopause`,
		`mount -o remount,bind,ro,noexec,nosuid,nodev /usr/lib/heliopause`,
		pipxInstalledNativeCommand,
		`"${ordinary[@]}" /bin/bash scripts/ci-uv-tools.sh pipx-bootstrap`,
		`"${ordinary[@]}" /bin/bash scripts/ci-uv-tools.sh pipx-run-workflow`,
		`"${ordinary[@]}" /bin/bash scripts/ci-uv-tools.sh pipx-run-cli`,
	} {
		if counts[command] != 1 {
			findings = append(findings, "pipx installed tools CI requires one exact command: "+command)
		}
	}
	if counts[`test "$(sha256sum "$input" | cut -d ' ' -f 1)" = "$expected_sha"`] != 2 {
		findings = append(findings, "pipx DATA input requires before/after digest verification")
	}
	joined := strings.Join(active, "\n")
	for _, scope := range []string{strings.Join([]string{`case "$profile" in`, `run)`, `"${ordinary[@]}" /bin/bash scripts/ci-uv-tools.sh pipx-run-workflow`, `"${ordinary[@]}" /bin/bash scripts/ci-uv-tools.sh pipx-run-cli`, `;;`, `tools-native)`, `cd internal/sandbox`, pipxInstalledNativeCommand, `;;`, `tools-bootstrap)`, `"${ordinary[@]}" /bin/bash scripts/ci-uv-tools.sh pipx-bootstrap`, `;;`, `esac`}, "\n")} {
		if strings.Count(joined, scope) != 1 {
			findings = append(findings, "pipx installed tools requires exact exclusive package dispatch")
		}
	}
	if strings.Count(contents, `ordinary=(/usr/bin/setpriv --reuid="$user_id" --regid="$group_id" --init-groups /usr/bin/env -i `) != 1 {
		findings = append(findings, "pipx installed tools requires explicit ordinary UID/GID and reset environment")
	}
	return findings
}
