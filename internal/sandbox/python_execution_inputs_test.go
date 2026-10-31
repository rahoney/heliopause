package sandbox

import (
	"context"
	"errors"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestPythonExecutionInputsRejectIncompleteAndPreserveCallerCause(t *testing.T) {
	var absent *PythonDynamicBackend
	var missing context.Context
	for _, row := range []struct {
		backend *PythonDynamicBackend
		ctx     context.Context
	}{{absent, context.Background()}, {&PythonDynamicBackend{}, missing}, {&PythonDynamicBackend{}, context.Background()}} {
		if err := row.backend.ValidatePythonExecutionInputs(row.ctx, domain.ExecutionContext{}, artifactpypi.ConsoleExecutionEntry{}); err == nil {
			t.Fatal("incomplete readonly execution inputs accepted")
		}
	}
	cause := errors.New("caller stopped Python input preparation")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	err := absent.ValidatePythonExecutionInputs(ctx, domain.ExecutionContext{}, artifactpypi.ConsoleExecutionEntry{})
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal("original caller cause lost", err)
	}
}

func TestPythonClosurePreparationPreservesExistingInspectionFailureMapping(t *testing.T) {
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("original preparation failure")
	for _, row := range []struct {
		code, phase, expected string
		keepsCause            bool
	}{
		{"M5_PYPI_DYNAMIC_INTRODUCTION_FAILED", "", "M5_PYPI_DYNAMIC_INTRODUCTION_FAILED", false},
		{"M5_PYPI_DYNAMIC_INSTALL_FAILED", "", "M5_PYPI_DYNAMIC_INSTALL_FAILED", true},
		{"M5_PYPI_DYNAMIC_ANCHOR_FAILED", "", "M5_PYPI_DYNAMIC_ANCHOR_FAILED", false},
		{"M5_PYPI_DYNAMIC_PREPARATION_FAILED", "", "M5_PYPI_DYNAMIC_PREPARATION_FAILED", true},
		{"M5_PYPI_DYNAMIC_CLOSURE_CHANGED", "", "M5_PYPI_DYNAMIC_CLOSURE_CHANGED", false},
		{"", "verify prepared installation", "M5_PYPI_DYNAMIC_CLOSURE_INVALID", true},
	} {
		failure := &pythonClosurePreparationFailure{code: row.code, phase: row.phase, cause: cause}
		result, err := failure.inspectionResult(session)
		code, incomplete := result.LimitationCode()
		if result.Status() != domain.SandboxIncomplete || !incomplete || code != row.expected || errors.Is(err, cause) != row.keepsCause || !errors.Is(failure, cause) {
			t.Fatal("existing inspection failure meaning changed", code, err)
		}
	}
}

func TestPythonPreparationZeroUnitsStillRequiresRealLifecycleEvidence(t *testing.T) {
	now := time.Now()
	policy, err := observationPolicyForPython(artifactpypi.PublicPyPIProfile().ResourcePolicy(), "pypi-wheel", 0, now)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := newObservationTransaction(policy, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := ledger.finalize(0, now); err == nil {
		t.Fatal("zero observation units bypassed preparation/accounting/cleanup")
	}
	ledger.preparationOK = true
	if err := ledger.finalize(0, now); err == nil {
		t.Fatal("preparation flag bypassed independent counter/teardown evidence")
	}
}
