package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestResponderSourceRequiresCurrentBuildAndEarlyPreparation(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("../..", workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	original := string(body)
	if findings := validateResponderSourceOrder(original); len(findings) != 0 {
		t.Fatal(findings)
	}
	for _, command := range []string{ciResponderBuild, "source scripts/ci-responder-source.sh", "haa_ci_fetch_semgrep_supply", "haa_ci_prepare_responder_base"} {
		for _, replacement := range []string{"# " + command, "echo " + command, command + " || true", command + "\n          " + command} {
			candidate := strings.Replace(original, command, replacement, 1)
			if candidate == original || len(validateResponderSourceOrder(candidate)) == 0 {
				t.Fatal("unprepared responder source accepted", replacement)
			}
		}
	}
}

// Model only: real filesystem/command failures, with the fixed root relocated
// and owner UID/GID modeled. The protected reader has separate native evidence.
func TestResponderSourceShellPreservesForeignStateAndRejectsUnknownQueries(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("hosted Linux fixture uses GNU stat/find")
	}
	body, err := os.ReadFile(filepath.Join("..", "ci-responder-source.sh"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	fixture := filepath.Join(root, "native")
	module := filepath.Join(root, "source.sh")
	if err := os.WriteFile(module, []byte(strings.ReplaceAll(string(body), "/usr/lib/heliopause", fixture)), 0o600); err != nil {
		t.Fatal(err)
	}
	const script = `set -euo pipefail
source "$1"
export RUNNER_TEMP=$2/run
mkdir "$RUNNER_TEMP"
printf 'model-only-source\n' > "$RUNNER_TEMP/haa-sandbox-responder"
printf 'model-core\n' > "$RUNNER_TEMP/semgrep-core-linux-amd64"
printf 'model-rules\n' > "$RUNNER_TEMP/semgrep-rules.json"
jq -n --arg c "$(sha256sum < "$RUNNER_TEMP/semgrep-core-linux-amd64" | cut -d ' ' -f 1)" --arg r "$(sha256sum < "$RUNNER_TEMP/semgrep-rules.json" | cut -d ' ' -f 1)" \
  '{semgrep_tool: {tool: {sha256: $c, size: 11}, rules: {sha256: $r, size: 12}}}' > "$2/runtimes.lock.json"
sudo() { [[ $1 == -n ]] || return 1; shift; "$@"; }
stat() {
  local value
  value=$(/usr/bin/stat "$@") || return 1
  printf '%s\n' "${value/#$MODEL_UID:$MODEL_GID:/0:0:}"
}
haa_ci_prepare_responder_base
haa_ci_require_responder_base
base=$(haa_ci_responder_source)
native=${base%/versions/*}
if haa_ci_prepare_responder_base; then exit 11; fi
printf foreign > "$native/foreign"
if haa_ci_require_responder_base; then exit 12; fi
test "$(cat "$native/foreign")" = foreign
rm "$native/foreign"
for version in v1.0.0-pnpm-ci v1.0.0-bun-ci; do
  mkdir -m 0755 "$native/versions/$version"
  cp "$base" "$native/versions/$version/haa-sandbox-responder"
  ln -sfnT "versions/$version" "$native/current"
  haa_ci_restore_responder_base "$version"
  rm "$native/versions/$version/haa-sandbox-responder"
  rmdir "$native/versions/$version"
  haa_ci_require_responder_base
done
ln -sfnT foreign "$native/current"
if haa_ci_restore_responder_base v1.0.0-pnpm-ci; then exit 13; fi
test "$(readlink "$native/current")" = foreign
ln -sfnT versions/v0.0.0-responder-ci "$native/current"
chmod 0644 "$base"
if haa_ci_responder_source; then exit 14; fi
chmod 0555 "$base"
ln "$base" "$native/extra-link"
if haa_ci_responder_source; then exit 15; fi
rm "$native/extra-link"
chmod 0755 "$base"
printf changed > "$base"
chmod 0555 "$base"
if haa_ci_responder_source; then exit 16; fi
chmod 0755 "$base"
cp "$RUNNER_TEMP/haa-sandbox-responder" "$base"
chmod 0555 "$base"
if (stat() { printf '0:0:755\n'; return 1; }; haa_ci_responder_source); then exit 17; fi
if (find() { command find "$@"; return 1; }; haa_ci_require_responder_base); then exit 18; fi
core="${base%/*}/semgrep-core-linux-amd64"
chmod 0755 "$core"
printf 'model-cor2\n' > "$core"
chmod 0555 "$core"
if haa_ci_semgrep_source core || haa_ci_require_responder_base; then exit 19; fi
chmod 0755 "$core"
cp "$RUNNER_TEMP/semgrep-core-linux-amd64" "$core"
chmod 0555 "$core"
if haa_ci_semgrep_source other; then exit 20; fi
haa_ci_require_responder_base
printf 'CI_RESPONDER_SOURCE_MODEL_OK\n'
`
	command := exec.Command("bash", "-c", script, "ci-source-model", module, root)
	command.Env = append(os.Environ(), "MODEL_UID="+strconv.Itoa(os.Getuid()), "MODEL_GID="+strconv.Itoa(os.Getgid()))
	output, err := command.CombinedOutput()
	if err != nil || string(output) != "CI_RESPONDER_SOURCE_MODEL_OK\n" {
		t.Fatal(string(output), err)
	}
}
