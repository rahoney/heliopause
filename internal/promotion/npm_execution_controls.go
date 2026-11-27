package promotion

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// BuildNPMExecutionControls renders only private controller data from the
// current whole set. It does not publish, install, access source bytes or grant
// runtime authority. The existing npm generator owns all lock semantics.
func BuildNPMExecutionControls(ctx context.Context, execution domain.ExecutionContext) ([]domain.ProjectControlFile, error) {
	if ctx == nil || !execution.Valid() {
		return nil, errors.New("npm execution controls require current execution context")
	}
	if ctx.Err() != nil {
		return nil, errors.Join(ctx.Err(), context.Cause(ctx))
	}
	manifest, lock, err := generatedNPMProjectSet(execution.Set())
	if err != nil {
		return nil, errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	controls := make([]domain.ProjectControlFile, 0, 2)
	for _, input := range []struct {
		name string
		body []byte
	}{{"package.json", manifest}, {"package-lock.json", lock}} {
		file, err := domain.NewProjectControlFile(input.name, input.body, true)
		if err != nil {
			return nil, errors.Join(err, ctx.Err(), context.Cause(ctx))
		}
		controls = append(controls, file)
	}
	if ctx.Err() != nil {
		return nil, errors.Join(ctx.Err(), context.Cause(ctx))
	}
	return controls, nil
}
