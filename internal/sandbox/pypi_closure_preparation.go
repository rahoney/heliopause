package sandbox

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// This error retains the existing M5 limitation/phase mapping while allowing
// a preparation-only consumer to preserve causes without creating a result.
type pythonClosurePreparationFailure struct {
	code, phase string
	cause       error
}

func (e *pythonClosurePreparationFailure) Error() string { return "python closure preparation failed" }
func (e *pythonClosurePreparationFailure) Unwrap() error { return e.cause }

func (e *pythonClosurePreparationFailure) inspectionResult(session domain.SandboxSessionID) (domain.SandboxResult, error) {
	if e.phase != "" {
		return pythonClosureFailure(session, e.phase, e.cause)
	}
	result, err := pythonIncomplete(session, e.code)
	if e.code == "M5_PYPI_DYNAMIC_INSTALL_FAILED" || e.code == "M5_PYPI_DYNAMIC_PREPARATION_FAILED" {
		err = errors.Join(err, e.cause)
	}
	return result, err
}

// Shared with the existing inspection transaction: the same authenticated
// stream, offline installer arguments and complete installed-tree census.
func (t *pythonTransactionExecutor) prepareWheelClosure(ctx context.Context, preparation string, artifact domain.AcquiredArtifact, closure []domain.AcquiredArtifact) *pythonClosurePreparationFailure {
	validated, err := t.backend.introducer.validatedWheelDestinations(artifact, closure)
	if err != nil {
		return &pythonClosurePreparationFailure{phase: "validate wheel destinations", cause: err}
	}
	wheelPaths := make([]string, 0, len(validated))
	for _, item := range validated {
		if err := t.backend.introducer.introduceWheelAt(ctx, preparation, item.artifact, item.destination); err != nil {
			return &pythonClosurePreparationFailure{code: "M5_PYPI_DYNAMIC_INTRODUCTION_FAILED", cause: err}
		}
		wheelPaths = append(wheelPaths, item.destination)
	}
	install := append(boundaryExecArguments(preparation, boundaryLaunchMode, "python", "-I", "-B", "-m", "pip", "install",
		"--no-index", "--no-deps", "--no-compile", "--disable-pip-version-check", "--no-cache-dir", "--target", pythonSitePath), wheelPaths...)
	if err := discardCommand(ctx, t.backend.runner, "docker", install...); err != nil {
		return &pythonClosurePreparationFailure{code: "M5_PYPI_DYNAMIC_INSTALL_FAILED", cause: t.diagnostic("install", errors.New(pythonCommandErrorReason(err)))}
	}
	installed, err := verifyClosureInstallation(ctx, t.backend.runner, preparation, t.manifest, t.policy.RuntimeTmpfs())
	if err != nil {
		return &pythonClosurePreparationFailure{phase: "verify prepared installation", cause: err}
	}
	t.installed = installed
	return nil
}

// Keep a real read-only reader attached before removing the writer. Both
// existing wheel and pipx transactions retain their original sealing checks.
func (t *pythonTransactionExecutor) sealPreparedClosure(ctx context.Context, preparation string) *pythonClosurePreparationFailure {
	anchor, err := t.startPhase(ctx, phaseAnchor, 1)
	if err != nil {
		return &pythonClosurePreparationFailure{code: "M5_PYPI_DYNAMIC_ANCHOR_FAILED", cause: err}
	}
	t.anchor = anchor.id
	if t.pipxVolume != nil {
		if err := t.pipxVolume.bindAnchor(ctx, t.backend.runner, anchor.id); err != nil {
			return &pythonClosurePreparationFailure{phase: "bind pipx DATA anchor", cause: err}
		}
	} else {
		if err := t.volume.verifyAttachments(ctx, t.backend.runner, map[string]bool{preparation: false, anchor.id: true}); err != nil {
			return &pythonClosurePreparationFailure{phase: "verify closure attachments", cause: err}
		}
	}
	if err := t.terminatePhase(preparation); err != nil {
		return &pythonClosurePreparationFailure{code: "M5_PYPI_DYNAMIC_PREPARATION_FAILED", cause: err}
	}
	if t.pipxVolume != nil {
		if err := t.pipxVolume.seal(ctx, t.backend.runner); err != nil {
			return &pythonClosurePreparationFailure{phase: "seal acquired pipx DATA", cause: err}
		}
	} else {
		if after, err := verifyClosureInstallation(ctx, t.backend.runner, anchor.id, t.manifest, t.policy.RuntimeTmpfs()); err != nil || after.identity != t.installed.identity {
			return &pythonClosurePreparationFailure{code: "M5_PYPI_DYNAMIC_CLOSURE_CHANGED", cause: err}
		}
	}
	t.ledger.mu.Lock()
	t.ledger.preparationOK, t.ledger.anchorAlive = true, true
	t.ledger.mu.Unlock()
	return nil
}
