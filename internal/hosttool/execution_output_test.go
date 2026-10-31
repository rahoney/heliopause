package hosttool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func captureFixture(t *testing.T) *Executor {
	t.Helper()
	for _, path := range []string{"/usr/bin/dash", "/usr/bin/bash", "/bin/bash", "/bin/sh"} {
		tool, err := verifyExecutable(path, "")
		if err == nil {
			return &Executor{tools: map[string]identity{"fixture": tool}, clientHome: t.TempDir()}
		}
	}
	t.Skip("no root-owned shell fixture executable")
	return nil
}

func TestRunCapturedSeparatesBinaryStreamsAndPreservesExit(t *testing.T) {
	t.Parallel()
	executor := captureFixture(t)
	output, err := executor.RunCaptured(context.Background(), "fixture", "-c", "printf 'out\\000\\377'; printf 'err\\033]private' >&2; exit 17")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 17 || len(exit.Stderr) != 0 {
		t.Fatalf("original exit was lost: %v", err)
	}
	if !bytes.Equal(output.Stdout(), []byte{'o', 'u', 't', 0, 255}) || string(output.Stderr()) != "err\x1b]private" {
		t.Fatal("binary streams were changed or combined")
	}
	if output.StdoutOverflow() || output.StderrOverflow() {
		t.Fatal("small output overflowed")
	}
	first, second := output.Stdout(), output.Stderr()
	first[0], second[0] = 'X', 'X'
	if output.Stdout()[0] != 'o' || output.Stderr()[0] != 'e' {
		t.Fatal("capture getters exposed mutable storage")
	}
	for _, rendered := range []string{fmt.Sprint(output), fmt.Sprintf("%+v", output), fmt.Sprintf("%#v", &output)} {
		if strings.Contains(rendered, "err") || strings.Contains(rendered, "out\x00") || strings.Contains(rendered, "\x1b") {
			t.Fatal("default formatting exposed raw output")
		}
	}
	if body, marshalErr := json.Marshal(output); marshalErr == nil || len(body) != 0 {
		t.Fatal("private output was JSON serializable")
	}
}

func TestRunCapturedExactLimitAndOverflow(t *testing.T) {
	t.Parallel()
	for _, stream := range []string{"stdout", "stderr"} {
		t.Run(stream, func(t *testing.T) {
			executor := captureFixture(t)
			redirect := ""
			if stream == "stderr" {
				redirect = " >&2"
			}
			block := strings.Repeat("x", 64<<10)
			for _, overflow := range []bool{false, true} {
				script := "i=0; while [ \"$i\" -lt 16 ]; do printf '%s' \"$1\"" + redirect + "; i=$((i+1)); done"
				if overflow {
					script += "; printf x" + redirect + "; while :; do :; done"
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				output, err := executor.RunCaptured(ctx, "fixture", "-c", script, "fixture", block)
				cancel()
				if overflow != errors.Is(err, ErrExecutionOutputLimit) || !overflow && err != nil {
					t.Fatalf("overflow=%v error=%v", overflow, err)
				}
				body, other, flag := output.Stdout(), output.Stderr(), output.StdoutOverflow()
				if stream == "stderr" {
					body, other, flag = output.Stderr(), output.Stdout(), output.StderrOverflow()
				}
				if len(body) != 1<<20 || len(other) != 0 || flag != overflow || !bytes.Equal(body, bytes.Repeat([]byte{'x'}, 1<<20)) {
					t.Fatal("stream bound, prefix or overflow fact was incorrect")
				}
			}
		})
	}
}

func TestRunCapturedCancellationAndSignal(t *testing.T) {
	t.Parallel()
	executor := captureFixture(t)
	cause := errors.New("fixture cancellation cause")
	ctx, cancel := context.WithCancelCause(context.Background())
	finished := make(chan error, 1)
	go func() {
		_, err := executor.RunCaptured(ctx, "fixture", "-c", "while :; do :; done")
		finished <- err
	}()
	time.AfterFunc(50*time.Millisecond, func() { cancel(cause) })
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
			t.Fatalf("cancellation cause was lost: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled Host command did not return")
	}
	output, err := executor.RunCaptured(context.Background(), "fixture", "-c", "printf before; kill -TERM $$")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != -1 || string(output.Stdout()) != "before" {
		t.Fatalf("Host signal/partial output was lost: %v", err)
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := executor.RunCaptured(deadline, "fixture", "-c", "while :; do :; done"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline was lost: %v", err)
	}
}

func TestRunCapturedBoundsInheritedPipeWait(t *testing.T) {
	t.Parallel()
	executor := captureFixture(t)
	sleepPath := ""
	for _, path := range []string{"/usr/bin/sleep", "/bin/sleep"} {
		if _, err := verifyExecutable(path, ""); err == nil {
			sleepPath = path
			break
		}
	}
	if sleepPath == "" {
		t.Skip("no root-owned sleep fixture executable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	started := time.Now()
	output, err := executor.RunCaptured(ctx, "fixture", "-c", `"$1" 30 & printf '%s' "$!"`, "fixture", sleepPath)
	pid, parseErr := strconv.Atoi(string(output.Stdout()))
	if parseErr != nil || pid <= 1 {
		t.Fatal("controlled child identity was not captured")
	}
	child, findErr := os.FindProcess(pid)
	if findErr != nil {
		t.Fatal(findErr)
	}
	// WaitDelay bounds pipes, not child lifetime. This controlled test owns
	// and terminates the independently surviving child explicitly.
	if killErr := child.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
		t.Fatal(killErr)
	}
	if !errors.Is(err, exec.ErrWaitDelay) || time.Since(started) > 5*time.Second {
		t.Fatalf("inherited pipe wait was not bounded: %v", err)
	}
}

func TestRunCapturedClosedInputAndMinimalEnvironment(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "private-ambient-value")
	executor := captureFixture(t)
	output, err := executor.RunCaptured(context.Background(), "fixture", "-c", "if read value; then exit 91; fi; printf '%s|%s' \"$HOME\" \"${HTTPS_PROXY-unset}\"")
	if err != nil || string(output.Stdout()) != executor.clientHome+"|unset" {
		t.Fatalf("closed input or minimal environment changed: %v", err)
	}
}

func TestRunCapturedRejectsUnavailableInputAndIdentity(t *testing.T) {
	t.Parallel()
	executor := captureFixture(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("preflight cancellation")
	cancel(cause)
	contexts := []context.Context{nil, ctx}
	for _, invalid := range contexts {
		if _, err := executor.RunCaptured(invalid, "fixture", "-c", "exit 0"); err == nil {
			t.Fatal("invalid context reached execution")
		} else if invalid == ctx && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
			t.Fatal("preflight cancellation lost its cause")
		}
	}
	if _, err := executor.RunCaptured(context.Background(), "unregistered"); err == nil {
		t.Fatal("unregistered tool reached execution")
	}
	tool := executor.tools["fixture"]
	tool.info = nil
	executor.tools["fixture"] = tool
	if _, err := executor.RunCaptured(context.Background(), "fixture"); err == nil {
		t.Fatal("changed identity reached execution")
	}
	var unavailable *Executor
	if _, err := unavailable.RunCaptured(context.Background(), "fixture"); err == nil {
		t.Fatal("unavailable executor reached execution")
	}
}
