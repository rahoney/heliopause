package sandbox

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
)

type admissionFirstErrorRunner struct {
	*testRealExecRunner
	prefix []byte
}

func (r *admissionFirstErrorRunner) Output(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	_, err := r.testRealExecRunner.Output(ctx, binary, arguments...)
	return append([]byte(nil), r.prefix...), err
}

func (r *admissionFirstErrorRunner) RunInput(ctx context.Context, input io.Reader, binary string, arguments ...string) error {
	if _, err := io.Copy(io.Discard, input); err != nil {
		return err
	}
	_, err := r.Output(ctx, binary, arguments...)
	return err
}

func (r *admissionFirstErrorRunner) RunBounded(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	return r.Output(ctx, binary, arguments...)
}

func (r *admissionFirstErrorRunner) RunOutput(ctx context.Context, output io.Writer, binary string, arguments ...string) error {
	body, err := r.Output(ctx, binary, arguments...)
	_, writeErr := output.Write(body)
	return errors.Join(err, writeErr)
}

func TestDirectExecFinisherPreservesNativeFailure(t *testing.T) {
	for _, method := range []string{"Output", "RunInput", "RunBounded", "RunOutput", "RunDiscard"} {
		for _, consumed := range []bool{false, true} {
			t.Run(method+"/"+map[bool]string{false: "unconsumed", true: "consumed"}[consumed], func(t *testing.T) {
				withShortAdmissionCompletionTimeout(t)
				const id = "0123456789abcdef"
				observer := startRealObserverForTest(t, id)
				registerAdmissionSession(t, id)
				// This local test-owned shell supplies a genuine OS ExitError. The
				// observer below is the existing protocol fixture, not real gVisor.
				nativeErr := exec.Command("/bin/sh", "-c", "exit 37").Run()
				if nativeErr == nil {
					t.Fatal("local nonzero fixture unexpectedly succeeded")
				}
				underlying := &admissionFirstErrorRunner{testRealExecRunner: &testRealExecRunner{err: nativeErr}, prefix: []byte("private-partial-prefix")}
				if consumed {
					underlying.onExec = func(nonce string) { observer.sendSentryExec(t, id, boundaryLaunchMode, nonce) }
				}
				runner := admissionAwareRunner(underlying)
				args := boundaryExecArguments(id, boundaryLaunchMode, "/bin/false")
				ctx := context.Background()
				var output []byte
				var resultErr error
				switch method {
				case "Output":
					output, resultErr = runner.Output(ctx, "docker", args...)
				case "RunInput":
					resultErr = runner.(inputCommandRunner).RunInput(ctx, strings.NewReader("inert-input"), "docker", args...)
				case "RunBounded":
					output, resultErr = runner.(boundedCommandRunner).RunBounded(ctx, "docker", args...)
				case "RunOutput":
					var buffer bytes.Buffer
					resultErr = runner.(streamingDockerOutput).RunOutput(ctx, &buffer, "docker", args...)
					output = buffer.Bytes()
				case "RunDiscard":
					resultErr = runner.(discardCommandRunner).RunDiscard(ctx, "docker", args...)
				}
				var nativeExit *exec.ExitError
				if !errors.Is(resultErr, nativeErr) || !errors.As(resultErr, &nativeExit) || nativeExit.ExitCode() != 37 {
					t.Fatal("admission finalization discarded the original native ExitError", resultErr)
				}
				if errors.Is(resultErr, errDirectExecNotStarted) != !consumed {
					t.Fatal("native error preservation changed admission uncertainty", resultErr)
				}
				if method == "Output" || method == "RunBounded" || method == "RunOutput" {
					if !bytes.Equal(output, underlying.prefix) {
						t.Fatal("first-error preservation discarded partial private output")
					}
				}
			})
		}
	}
}
