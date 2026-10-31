package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Reuse the existing UV pip complete-project materializer with the native
// project caller. Only authenticated staged wheels enter this private output.
func withApprovedPythonProjectOutput(ctx context.Context, promoter *PyPIPromotion, transaction *pypiVenvTransaction, root string, material pythonProjectMaterial, staged domain.StagedProjectSet, commit func([]pypiDestination) error) (resultErr error) {
	var desired []pypiDestination
	if !staged.Set().Inspected().Snapshot().DependencyFree() {
		temporary, err := os.MkdirTemp(transaction.plan.root, ".haa-pypi-work-")
		if err != nil {
			return err
		}
		defer func() {
			if !transaction.recovery {
				resultErr = errors.Join(resultErr, os.RemoveAll(temporary))
			}
		}()
		if err := os.Chmod(temporary, 0o700); err != nil {
			return err
		}
		if err := copyExactRecord(filepath.Join(root, manifestFilename), filepath.Join(temporary, manifestFilename), material.document); err != nil {
			return err
		}
		if err := copyExactRecord(filepath.Join(root, sbomFilename), filepath.Join(temporary, sbomFilename), material.sbom); err != nil {
			return err
		}
		requirements, expected, err := preparePythonWheels(temporary, filepath.Join(root, "artifacts"), material.nodes, staged.Set().Inspected().Inspections())
		if err != nil {
			return err
		}
		desired, err = promoter.materializePythonProject(ctx, temporary, requirements, expected)
		if err != nil {
			return err
		}
	}
	return commit(desired)
}
