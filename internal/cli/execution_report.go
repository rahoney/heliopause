package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type executionReport struct {
	Schema          string                `json:"schema"`
	Quarantine      string                `json:"quarantine"`
	Policy          string                `json:"policy"`
	Preparation     string                `json:"preparation"`
	Execution       string                `json:"execution"`
	ExitCode        *uint8                `json:"exit_code,omitempty"`
	Signal          *uint8                `json:"signal,omitempty"`
	OutputPromotion string                `json:"output_promotion"`
	GuestCleanup    string                `json:"execution_guest_cleanup"`
	FailureCode     string                `json:"failure_code,omitempty"`
	ProgramOutput   executionOutputReport `json:"program_output"`
}

type executionOutputReport struct {
	Untrusted   bool   `json:"untrusted"`
	Encoding    string `json:"encoding"`
	StdoutBytes int    `json:"stdout_bytes"`
	StderrBytes int    `json:"stderr_bytes"`
	Stdout      string `json:"stdout_escaped,omitempty"`
	Stderr      string `json:"stderr_escaped,omitempty"`
}

// Display-only data; it never creates inspection, admission or terminal facts.
func executionOutput(stdout, stderr []byte, show bool) executionOutputReport {
	r := executionOutputReport{Untrusted: true, Encoding: "OMITTED", StdoutBytes: len(stdout), StderrBytes: len(stderr)}
	if show {
		r.Encoding = "ASCII_QUOTED"
		r.Stdout, r.Stderr = strconv.QuoteToASCII(string(stdout)), strconv.QuoteToASCII(string(stderr))
	}
	return r
}

func newExecutionReport(result application.ExecutionResult, cause error, show bool) executionReport {
	r := executionReport{Schema: "helox.execution-result/v1", Quarantine: "INCOMPLETE", Policy: "NOT_AVAILABLE", Preparation: "INCOMPLETE", Execution: "NOT_STARTED", OutputPromotion: "NOT_REQUESTED", GuestCleanup: "NOT_STARTED", FailureCode: result.FailureCode(), ProgramOutput: executionOutput(result.Stdout(), result.Stderr(), show)}
	prepared := result.Preparation()
	if inspected := prepared.Inspection(); inspected.Valid() {
		r.Quarantine, r.Policy = "COMPLETED", string(inspected.Set().Decision().Decision())
	}
	var denied *application.ExecutionPolicyFailure
	if errors.As(cause, &denied) {
		r.Quarantine, r.Policy = "COMPLETED", string(denied.Decision().Decision())
	}
	if prepared.Valid() {
		r.Preparation = "COMPLETED"
	}
	if result.RuntimeAttempted() {
		r.Execution, r.GuestCleanup = "INCOMPLETE", "NOT_CONFIRMED"
	}
	if result.Complete() {
		r.Execution, r.GuestCleanup = "COMPLETED", "CONFIRMED"
		if code, ok := result.ExitCode(); ok {
			r.ExitCode = &code
		}
		if signal, ok := result.Signal(); ok {
			r.Signal = &signal
		}
	}
	return r
}

func executionProcessCode(result application.ExecutionResult, cause error) int {
	if result.Complete() {
		if code, ok := result.ExitCode(); ok {
			if code == 0 && cause != nil {
				return 1
			}
			return int(code)
		}
		if signal, ok := result.Signal(); ok {
			return 128 + int(signal)
		}
	}
	var denied *application.ExecutionPolicyFailure
	if errors.As(cause, &denied) {
		if denied.Decision().Decision() == domain.DecisionManualReview {
			return 10
		}
		if denied.Decision().Decision() == domain.DecisionBlock {
			return 20
		}
	}
	return 1
}

func writeExecutionReport(output io.Writer, r executionReport, machine bool) error {
	if output == nil {
		return errors.New("execution reporting requires an output writer")
	}
	if machine {
		return json.NewEncoder(output).Encode(r)
	}
	p := presentationFor(output)
	code, signal := "NOT_AVAILABLE", "NOT_AVAILABLE"
	if r.ExitCode != nil {
		code = strconv.Itoa(int(*r.ExitCode))
	}
	if r.Signal != nil {
		signal = strconv.Itoa(int(*r.Signal))
	}
	_, err := fmt.Fprintf(output, "%s: %s / Policy: %s\n%s: %s\n%s: %s / exit: %s / signal: %s\n%s: %s\n%s: %s\n%s: stdout=%dB stderr=%dB (%s)\n",
		p.text("Quarantine", "검역"), r.Quarantine, r.Policy, p.text("Storage preparation", "보관 준비"), r.Preparation, p.text("Execution", "실행"), r.Execution, code, signal, p.text("Output promotion", "출력 반출"), r.OutputPromotion, p.text("Execution guest cleanup", "실행 guest 정리"), r.GuestCleanup, p.text("Untrusted program output", "신뢰하지 않는 프로그램 출력"), r.ProgramOutput.StdoutBytes, r.ProgramOutput.StderrBytes, r.ProgramOutput.Encoding)
	if err != nil {
		return err
	}
	if r.FailureCode != "" {
		if _, err := fmt.Fprintf(output, "%s: %s\n", p.text("Failure", "실패"), safeTerminal(r.FailureCode)); err != nil {
			return err
		}
	}
	if r.ProgramOutput.Encoding == "ASCII_QUOTED" {
		_, err = fmt.Fprintf(output, "program stdout (untrusted): %s\nprogram stderr (untrusted): %s\n", r.ProgramOutput.Stdout, r.ProgramOutput.Stderr)
	}
	return err
}
