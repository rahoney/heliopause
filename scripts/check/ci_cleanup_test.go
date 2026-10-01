package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func cleanupScript(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../ci-integration-cleanup.sh")
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func runCleanup(t *testing.T, body string, env ...string) (int, string, string) {
	t.Helper()
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	shell := os.Getenv("HELOX_CI_BASH")
	if shell == "" {
		shell = "/bin/bash"
	}
	t.Logf("CI shell: %s", shell)
	args := []string{"--noprofile", "--norc", "-e", "-u", "-o", "pipefail"}
	body = `printf 'shell=%s version=%s options=%s\n' "$BASH" "$BASH_VERSION" "$-" >&2` + "\n" + body
	if os.Getenv("HELOX_CI_SCRIPT_FILE") == "1" {
		path := filepath.Join(dir, "case.sh")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		args = append(args, path)
	} else {
		args = append(args, "-c", body)
	}
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = append(os.Environ(), "SCRIPT="+cleanupScript(t), "CASE_DIR="+dir, "RUNNER_TEMP=/production")
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if ctx.Err() != nil {
		t.Fatalf("bounded child test timed out: %s", out)
	}
	actions, err := os.ReadFile(filepath.Join(dir, "actions"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return code, string(out), string(actions)
}

// Only observation and privileged commands are stubbed. State transitions,
// EXIT handling and restoration are the exact implementation sourced by CI.
const cleanupStubs = `
source "$SCRIPT"
haa_ci_initialize
printf '1' > "$CASE_DIR/live"
printf '0' > "$CASE_DIR/queries"
jobs() {
 local n live
 n=$(cat "$CASE_DIR/queries"); n=$((n+1)); printf '%s' "$n" > "$CASE_DIR/queries"
 live=$(cat "$CASE_DIR/live")
 case ${QUERY-normal} in
   error) return 2;; producer) return 3;; malformed) echo garbage; return 0;; duplicate) printf '42\n42\n'; return 0;;
   running-error) if [[ "$*" == "-r -p" ]]; then return 6; fi;;
   stopped-error) if [[ "$*" == "-s -p" ]]; then return 7; fi;;
   afterterm) if [[ $live == 0 ]]; then return 5; fi;;
 esac
 if [[ "$*" == "-s -p" ]]; then return 0; fi
 if [[ $live == 1 || ( ${COMPLETED_ENTRY-0} == 1 && "$*" == "-p" ) ]]; then echo 42; fi
}
grep() { echo matcher >> "$CASE_DIR/actions"; return "${MATCH_STATUS-2}"; }
wait() { echo "wait:$1" >> "$CASE_DIR/actions"; return "${WAIT_STATUS-0}"; }
sleep() { return "${SLEEP_STATUS-0}"; }
sudo() {
 if [[ $1 == kill ]]; then
   echo "term:$3" >> "$CASE_DIR/actions"
   if [[ ${STOP-0} != 0 ]]; then return "$STOP"; fi
   if [[ ${HANG-0} == 0 ]]; then printf '0' > "$CASE_DIR/live"; fi
   return 0
 fi
 echo "install:$4" >> "$CASE_DIR/actions"
 return "${RESTORE-0}"
}
# Fault injection starts from a recorded owning lifecycle. Separate real-child
# tests below exercise the actual $! registration and Bash job table.
haa_ever_started=1; haa_registered=1; haa_recorded_pid=42; policy_helper_pid=42
haa_helper_state=OWNED_LIVE
haa_client_restore_required=1; haa_replacement_recorded=1
`

func TestActualCIIntegrationCleanup(t *testing.T) {
	workflow, err := os.ReadFile("../../.github/workflows/heliopause-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range []string{"source scripts/ci-integration-cleanup.sh", "haa_ci_initialize", "haa_start_policy_helper", "haa_install_ci_client"} {
		if !strings.Contains(string(workflow), call) {
			t.Fatalf("missing shared lifecycle call %s", call)
		}
	}
	if strings.Contains(string(workflow), "sudo kill -0") || strings.Contains(string(workflow), "policy_helper_pid=$!") {
		t.Fatal("parallel helper-state logic remains")
	}
	for _, original := range []int{0, 37} {
		for _, restore := range []int{0, 19} {
			for _, stop := range []int{0, 23} {
				t.Run(fmt.Sprintf("original%d_restore%d_stop%d", original, restore, stop), func(t *testing.T) {
					code, out, actions := runCleanup(t, cleanupStubs+`exit "$ORIGINAL"`, fmt.Sprintf("ORIGINAL=%d", original), fmt.Sprintf("RESTORE=%d", restore), fmt.Sprintf("STOP=%d", stop))
					want := original
					if want == 0 {
						want = stop
					}
					if want == 0 {
						want = restore
					}
					expected := "term:42\nwait:42\ninstall:/production/helox\n"
					state := "CONFIRMED_STOPPED"
					result := "RESTORED"
					if restore != 0 {
						result = "FAILED"
					}
					if stop != 0 {
						expected = "term:42\n"
						state = "UNKNOWN"
						result = "SKIPPED_UNSAFE"
					}
					if code != want || actions != expected || !strings.Contains(out, "helper_state="+state) || !strings.Contains(out, "restore_result="+result) || !strings.Contains(out, fmt.Sprintf("original=%d", original)) {
						t.Fatalf("exit %d want %d actions %q\n%s", code, want, actions, out)
					}
				})
			}
		}
	}
}

func TestCIHelperStateFaults(t *testing.T) {
	cases := []struct {
		name, setup                string
		env                        []string
		code                       int
		actions, state, diagnostic string
	}{
		{"query-error", "", []string{"QUERY=error"}, 2, "", "UNKNOWN", "job-table query failed"},
		{"producer-error-not-matcher", "", []string{"QUERY=producer", "MATCH_STATUS=1"}, 3, "", "UNKNOWN", "job-table query failed"},
		{"running-query-error", "", []string{"QUERY=running-error"}, 6, "", "UNKNOWN", "running job-table query failed"},
		{"stopped-query-error", "", []string{"QUERY=stopped-error"}, 7, "", "UNKNOWN", "stopped job-table query failed"},
		{"completed-entry-retained", "printf 0 > \"$CASE_DIR/live\"", []string{"COMPLETED_ENTRY=1"}, 0, "wait:42\ninstall:/production/helox\n", "CONFIRMED_STOPPED", "RESTORED"},
		{"interrupted-wait", "printf 0 > \"$CASE_DIR/live\"", []string{"WAIT_STATUS=130"}, 130, "wait:42\n", "UNKNOWN", "did not complete cleanly"},
		{"query-after-term", "", []string{"QUERY=afterterm"}, 5, "term:42\n", "UNKNOWN", "job-table query failed"},
		{"malformed-output", "", []string{"QUERY=malformed"}, 1, "", "UNKNOWN", "malformed job-table"},
		{"duplicate-output", "", []string{"QUERY=duplicate"}, 1, "", "UNKNOWN", "malformed job-table"},
		{"missing-after-launch", "policy_helper_pid=''", nil, 1, "", "UNKNOWN", "unowned lifecycle"},
		{"malformed-pid", "policy_helper_pid=oops", nil, 1, "", "UNKNOWN", "unowned lifecycle"},
		{"one-pid", "haa_recorded_pid=1; policy_helper_pid=1", nil, 1, "", "UNKNOWN", "invalid lifecycle PID"},
		{"zero-pid", "haa_recorded_pid=0; policy_helper_pid=0", nil, 1, "", "UNKNOWN", "unowned lifecycle"},
		{"unowned", "haa_registered=0", nil, 1, "", "UNKNOWN", "unowned lifecycle"},
		{"wrong-owner", "haa_owner=0", nil, 1, "", "UNKNOWN", "non-owning shell"},
		{"signal-failure", "", []string{"STOP=23"}, 23, "term:42\n", "UNKNOWN", "termination request failed"},
		{"suspended-or-timeout", "", []string{"HANG=1"}, 1, "term:42\n", "UNKNOWN", "bounded wait"},
		{"sleep-failure", "", []string{"HANG=1", "SLEEP_STATUS=7"}, 7, "term:42\n", "UNKNOWN", "confirmation delay failed"},
		{"wrapper-signalled", "printf 0 > \"$CASE_DIR/live\"", []string{"WAIT_STATUS=143"}, 143, "wait:42\n", "UNKNOWN", "did not complete cleanly"},
		{"not-child", "printf 0 > \"$CASE_DIR/live\"", []string{"WAIT_STATUS=127"}, 127, "wait:42\n", "UNKNOWN", "did not complete cleanly"},
		{"already-terminated", "printf 0 > \"$CASE_DIR/live\"", nil, 0, "wait:42\ninstall:/production/helox\n", "CONFIRMED_STOPPED", "RESTORED"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, out, actions := runCleanup(t, cleanupStubs+c.setup+"\nexit 0", c.env...)
			if code != c.code || actions != c.actions || !strings.Contains(out, "helper_state="+c.state) || !strings.Contains(out, c.diagnostic) {
				t.Fatalf("exit %d want %d actions %q want %q\n%s", code, c.code, actions, c.actions, out)
			}
			if c.state == "UNKNOWN" && (!strings.Contains(out, "SKIPPED_UNSAFE") || !strings.Contains(out, "pid=")) {
				t.Fatal(out)
			}
			code, out, actions = runCleanup(t, cleanupStubs+c.setup+"\nexit 37", c.env...)
			if code != 37 || actions != c.actions || !strings.Contains(out, "original=37") {
				t.Fatalf("primary error lost: %d %q %s", code, actions, out)
			}
		})
	}
}

func TestCIHelperEarlyAndRepeatedCleanup(t *testing.T) {
	cases := []struct {
		name, body    string
		code          int
		actions, diag string
	}{
		{"never-started", `source "$SCRIPT"; haa_ci_initialize; exit 0`, 0, "", "NOT_STARTED"},
		{"conflicting-initialization", `source "$SCRIPT"; policy_helper_pid=42; haa_ci_initialize`, 1, "", "UNKNOWN"},
		{"missing-launch", `source "$SCRIPT"; haa_ci_initialize; haa_record_policy_helper_launch`, 1, "", "UNKNOWN"},
		{"subshell-owner", cleanupStubs + `if (haa_stop_policy_helper); then exit 99; fi
exit 37`, 37, "term:42\nwait:42\ninstall:/production/helox\n", "original=37"},
		{"before-initialization-zero", `source "$SCRIPT"; exit 0`, 1, "", "UNKNOWN"},
		{"before-initialization", `source "$SCRIPT"; exit 37`, 37, "", "UNKNOWN"},
		{"unproven-replacement", `source "$SCRIPT"; haa_ci_initialize; haa_client_restore_required=1; exit 0`, 1, "", "SKIPPED_UNSAFE"},
		{"legitimate-never-started-replacement", cleanupStubs + `haa_ever_started=0; haa_registered=0; haa_recorded_pid=''; policy_helper_pid=''; haa_helper_state=NOT_STARTED; haa_replacement_recorded=0; haa_client_restore_required=0
haa_install_ci_client /test-client
exit 0`, 0, "install:/test-client\ninstall:/production/helox\n", "RESTORED"},
		{"repeated-confirmed", cleanupStubs + `haa_stop_policy_helper; haa_stop_policy_helper; exit 0`, 0, "term:42\nwait:42\ninstall:/production/helox\n", "CONFIRMED_STOPPED"},
		{"repeated-unknown", cleanupStubs + `STOP=23
if haa_stop_policy_helper; then exit 99; fi
if haa_stop_policy_helper; then exit 98; fi
if haa_prepare_policy_helper_launch; then exit 97; fi
exit 37`, 37, "term:42\n", "SKIPPED_UNSAFE"},
		{"client-failure-blocks-rebind", cleanupStubs + `haa_ever_started=0; haa_registered=0; haa_recorded_pid=''; policy_helper_pid=''; haa_helper_state=NOT_STARTED
RESTORE=18
if haa_install_ci_client /test-client; then exit 99; fi
if haa_prepare_policy_helper_launch; then exit 98; fi
if haa_install_ci_client /second-client; then exit 97; fi
RESTORE=19
exit 37`, 37, "install:/test-client\ninstall:/production/helox\n", "client_error=18 client_restore=19"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, out, actions := runCleanup(t, c.body)
			if code != c.code || actions != c.actions || !strings.Contains(out, c.diag) {
				t.Fatalf("exit %d actions %q\n%s", code, actions, out)
			}
		})
	}
}

func TestCIClientReplacementStopsBeforeInstall(t *testing.T) {
	code, out, actions := runCleanup(t, cleanupStubs+`haa_install_ci_client /test-client; exit 0`)
	if code != 0 || actions != "term:42\nwait:42\ninstall:/test-client\ninstall:/production/helox\n" {
		t.Fatalf("%d %q %s", code, actions, out)
	}
	code, out, actions = runCleanup(t, cleanupStubs+`haa_install_ci_client /test-client`, "QUERY=error")
	if code != 2 || actions != "" || !strings.Contains(out, "SKIPPED_UNSAFE") {
		t.Fatalf("query error permitted replacement: %d %q %s", code, actions, out)
	}
	code, out, actions = runCleanup(t, cleanupStubs+`haa_install_ci_client /test-client`, "STOP=23")
	if code != 23 || actions != "term:42\n" {
		t.Fatalf("%d %q %s", code, actions, out)
	}
}

func TestCIHelperRealChildLifecycle(t *testing.T) {
	// Real jobs, $!, wait and kill; sudo is replaced with an exact recorded-child
	// signal, and install only records an action. No product helper or privilege.
	common := `source "$SCRIPT"
haa_ci_initialize
trap - EXIT
owned=''
cleanup_fixture() {
 trap - EXIT
 if [[ -n $owned ]]; then
   if kill -CONT "$owned" 2>/dev/null; then :; fi
   if kill -TERM "$owned" 2>/dev/null; then :; fi
   if wait "$owned" 2>/dev/null; then :; fi
 fi
}
trap cleanup_fixture EXIT
sudo() {
 if [[ $1 == kill ]]; then
  [[ $3 == "$owned" ]] || return 99
  echo term >> "$CASE_DIR/actions"
  kill -TERM "$3"
 else echo install >> "$CASE_DIR/actions"; fi
}
`
	cases := []struct {
		name, launch, exercise string
		actions                string
	}{
		{"normal-early-exit", `"$BASH" -c 'exit 0' &`, `sleep 0.1; haa_observe_policy_helper; [[ $haa_helper_state == CONFIRMED_STOPPED ]]; haa_stop_policy_helper`, ""},
		{"graceful-stop", `"$BASH" -c 'trap "exit 0" TERM; echo ready > "$CASE_DIR/ready"; for i in {1..30}; do sleep 0.1; done' &`, `while [[ ! -f "$CASE_DIR/ready" ]]; do sleep 0.01; done; haa_stop_policy_helper; [[ $haa_helper_state == CONFIRMED_STOPPED ]]`, "term\n"},
		{"suspended-is-live", `"$BASH" -c 'trap "exit 0" TERM; echo ready > "$CASE_DIR/ready"; for i in {1..30}; do sleep 0.1; done' &`, `while [[ ! -f "$CASE_DIR/ready" ]]; do sleep 0.01; done; kill -STOP "$owned"; sleep 0.05; haa_observe_policy_helper; [[ $haa_helper_state == OWNED_LIVE ]]; kill -CONT "$owned"; haa_stop_policy_helper; [[ $haa_helper_state == CONFIRMED_STOPPED ]]`, "term\n"},
		{"foreground-wrapper", `"$BASH" -c '
 "$BASH" -c '\''trap "exit 0" TERM; echo ready > "$CASE_DIR/ready"; for i in {1..30}; do sleep 0.1; done'\'' &
 child=$!
 trap '\''kill -TERM "$child"; if wait "$child"; then echo reaped > "$CASE_DIR/reaped"; exit 0; else exit 1; fi'\'' TERM
 wait "$child"
' &`, `while [[ ! -f "$CASE_DIR/ready" ]]; do sleep 0.01; done; haa_stop_policy_helper; [[ $haa_helper_state == CONFIRMED_STOPPED && -f "$CASE_DIR/reaped" ]]`, "term\n"},
		{"failed-wrapper", `"$BASH" -c 'exit 19' &`, `sleep 0.1; if haa_stop_policy_helper; then exit 99; fi; [[ $haa_helper_state == UNKNOWN && $haa_helper_error == 19 ]]; if haa_prepare_policy_helper_launch; then exit 98; fi`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body := common + c.launch + "\nowned=$!\nprintf 'owned pid=%s shell=%s\\n' \"$owned\" \"$$\" >&2\nhaa_record_policy_helper_launch \"$!\"\n" + c.exercise + "\nowned=''\n"
			// A very early failed wrapper may already fail registration; delay its exit
			// so registration exercises OWNED_LIVE before the failed wait.
			if c.name == "failed-wrapper" {
				body = strings.Replace(body, "\"$BASH\" -c 'exit 19' &", "\"$BASH\" -c 'sleep 0.05; exit 19' &", 1)
			}
			code, out, actions := runCleanup(t, body)
			if code != 0 || actions != c.actions {
				t.Fatalf("exit %d actions %q want %q\n%s", code, actions, c.actions, out)
			}
		})
	}
}
