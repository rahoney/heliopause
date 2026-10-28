package npm

import (
	"context"
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

// ReadNodeExecutionEntry reuses the complete bounded archive inspection to
// obtain one selected member's bytes. It binds the measurement to
// the acquired subject and the complete original inventory, never a Host path.
// It does not provide Policy approval or permission to execute the result.
func (i *Inspector) ReadNodeExecutionEntry(ctx context.Context, entry artifactnpm.ExecutionEntry) (artifactnpm.NodeExecutionEntry, error) {
	if ctx == nil || !entry.Valid() {
		return artifactnpm.NodeExecutionEntry{}, errors.New("node entry read requires a context and exact selected inventory")
	}
	capture := &archiveCapture{selectedPath: entry.Target().Path}
	report, err := i.inspect(ctx, entry.Artifact(), capture)
	if err != nil {
		return artifactnpm.NodeExecutionEntry{}, errors.Join(err, context.Cause(ctx))
	}
	if len(report.Findings()) != 0 || !entry.Matches(capture.inventory) {
		return artifactnpm.NodeExecutionEntry{}, errors.New("node entry source or complete inventory differs from its selection")
	}
	if err := ctx.Err(); err != nil {
		return artifactnpm.NodeExecutionEntry{}, errors.Join(err, context.Cause(ctx))
	}
	proof, err := entry.BindNodeTarget(capture.selectedBody)
	if err != nil {
		return artifactnpm.NodeExecutionEntry{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactnpm.NodeExecutionEntry{}, errors.Join(err, context.Cause(ctx))
	}
	return proof, nil
}
