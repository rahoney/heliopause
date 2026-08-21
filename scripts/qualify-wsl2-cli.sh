#!/usr/bin/env bash
# Run M7-003's native CLI qualification only inside a disposable WSL2 distro.
set -euo pipefail

if [[ $# -ne 1 ]]; then
	printf 'usage: %s EVIDENCE_FILE\n' "$0" >&2
	exit 2
fi
if [[ ! -f go.mod ]] || ! grep -qx 'module github.com/rahoney/heliopause' go.mod; then
	printf 'run from the Heliopause repository root\n' >&2
	exit 2
fi

evidence_file=$1
evidence_directory=$(dirname "$evidence_file")
if [[ ! -d "$evidence_directory" ]]; then
	printf 'evidence directory does not exist: %s\n' "$evidence_directory" >&2
	exit 2
fi
if [[ -e "$evidence_file" ]]; then
	printf 'refuse to overwrite existing evidence: %s\n' "$evidence_file" >&2
	exit 2
fi

kernel_release=$(uname -r)
case "$kernel_release" in
*[Ww][Ss][Ll]2*) ;;
*)
	printf 'M7-003 requires a WSL2 kernel, got: %s\n' "$kernel_release" >&2
	exit 1
	;;
esac

umask 077
: >"$evidence_file"
build_directory=$(mktemp -d)
qualification_status=FAILED
finish() {
	exit_code=$?
	printf 'qualification_status=%s\n' "$qualification_status" >>"$evidence_file"
	rm -rf "$build_directory"
	exit "$exit_code"
}
trap finish EXIT

{
	printf 'qualification=M7-003-wsl2-cli\n'
	printf 'kernel_release=%s\n' "$kernel_release"
	grep '^PRETTY_NAME=' /etc/os-release
	printf 'go_version='
	go version
	go env GOOS GOARCH
	if command -v wsl.exe >/dev/null 2>&1; then
		printf 'wsl_host_version:\n'
		wsl.exe --version
		printf 'wsl_distribution_list:\n'
		wsl.exe -l -v
	fi
	printf 'build_command=go build -trimpath -o <temporary>/helox ./cmd/helox\n'
	go build -trimpath -o "$build_directory/helox" ./cmd/helox
	printf 'default_command_output:\n'
	"$build_directory/helox"
	printf 'help_command_output:\n'
	"$build_directory/helox" --help
} >>"$evidence_file" 2>&1

qualification_status=COMPLETED
