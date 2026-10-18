package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

type presentedError struct{ cause error }

func (e presentedError) Error() string { return "operation failed; see the displayed report" }
func (e presentedError) Unwrap() error { return e.cause }
func (e presentedError) ExitCode() int { return 1 }

// PresentRuntimeError reports startup/operational failure without replaying raw
// backend errors. The cause is retained for errors.Is and never becomes ALLOW.
func PresentRuntimeError(command *cobra.Command, cause error) error {
	if cause == nil {
		return nil
	}
	p := presentationFor(command.OutOrStdout())
	if !p.human() {
		return cause
	}
	p.interruption = interruptionCode(cause)
	var coded interface{ ExitCode() int }
	if errors.As(cause, &coded) {
		return cause
	}
	if !p.reportedError {
		code := "OPERATION_FAILED"
		message := strings.ToLower(cause.Error())
		if p.interruption != "" {
			code = p.interruption
		} else if strings.Contains(message, "network policy") {
			code = "NETWORK_POLICY_SERVICE_UNAVAILABLE"
		} else if strings.Contains(message, "observer") {
			code = "OBSERVER_HELPER_UNAVAILABLE"
		} else if strings.Contains(message, "runtime") {
			code = "TRUSTED_HOST_RUNTIME_UNAVAILABLE"
		}
		p.reportHeader("REQUEST", "FAILED", "", "")
		p.line("%s %s", p.text("Error:", "오류:"), code)
		p.line("%s", p.errorExplanation(code))
	}
	return presentedError{cause: cause}
}

// PresentFinalizationError makes actual deferred cleanup uncertainty visible even
// after an individually successful operation or a separately coded Policy exit.
// It never replaces or interprets the original cleanup error returned by Bootstrap.
func PresentFinalizationError(command *cobra.Command, cause error) {
	if cause == nil {
		return
	}
	p := presentationFor(command.OutOrStdout())
	if !p.human() {
		return
	}
	p.reportHeader("FINALIZATION", "FAILED", "", "")
	p.line("Error: PROCESS_FINALIZATION_FAILED")
	p.line("%s", p.text("The operation result above does not confirm clean process termination. Check the controlled runtime cleanup before retrying.", "위 개별 작업 결과와 별개로 정상 종료·정리를 확인하지 못했습니다. 재실행 전에 실행 환경의 정리 상태를 확인하세요."))
	p.reportedError = true
}

func interruptionCode(cause error) string {
	if errors.Is(cause, context.Canceled) {
		return "OPERATION_CANCELLED"
	}
	if errors.Is(cause, context.DeadlineExceeded) {
		return "OPERATION_TIMED_OUT"
	}
	return ""
}
