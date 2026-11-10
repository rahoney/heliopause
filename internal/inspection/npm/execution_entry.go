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
	body, err := i.readExecutionEntryBody(ctx, entry)
	if err != nil {
		return artifactnpm.NodeExecutionEntry{}, err
	}
	proof, err := entry.BindNodeTarget(body)
	if err != nil {
		return artifactnpm.NodeExecutionEntry{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactnpm.NodeExecutionEntry{}, errors.Join(err, context.Cause(ctx))
	}
	return proof, nil
}

func (i *Inspector) ReadBunExecutionEntry(ctx context.Context, entry artifactnpm.ExecutionEntry) (artifactnpm.BunExecutionEntry, error) {
	body, err := i.readExecutionEntryBody(ctx, entry)
	if err != nil {
		return artifactnpm.BunExecutionEntry{}, err
	}
	proof, err := entry.BindBunTarget(body)
	if err != nil {
		return artifactnpm.BunExecutionEntry{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactnpm.BunExecutionEntry{}, errors.Join(err, context.Cause(ctx))
	}
	return proof, nil
}

func (i *Inspector) readExecutionEntryBody(ctx context.Context, entry artifactnpm.ExecutionEntry) ([]byte, error) {
	if ctx == nil || !entry.Valid() {
		return nil, errors.New("entry read requires a context and exact selected inventory")
	}
	capture := &archiveCapture{selectedPath: entry.Target().Path}
	report, err := i.inspect(ctx, entry.Artifact(), capture)
	if err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	if len(report.Findings()) != 0 || !entry.Matches(capture.inventory) {
		return nil, errors.New("entry source or complete inventory differs from its selection")
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	return capture.selectedBody, nil
}
