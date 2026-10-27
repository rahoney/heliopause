package githubrelease

import (
	"context"
	"errors"
	"io"

	"github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// pipxRuntimeInput retains the original HAA subject and the plan derived from
// its exact acquired bytes. It cannot issue inspection completion or Evidence.
type pipxRuntimeInput struct {
	artifact domain.AcquiredArtifact
	plan     pypi.PipxRuntimePlan
}

func (i *StaticInspector) readPipxRuntimeInput(ctx context.Context, artifact domain.AcquiredArtifact) (input pipxRuntimeInput, resultErr error) {
	if i == nil || ctx == nil {
		return input, errors.New("pipx runtime intake reader is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return input, err
	}
	identity := artifact.Identity()
	if identity.Source().String() != "github-release" || identity.Name() != "rahoney-heliopause" || identity.Version() == "" || identity.Variant() != "pipx-runtime-linux-amd64.zip" || artifact.SizeBytes() != runtimeidentity.PipxToolSize || artifact.Digest().String() != runtimeidentity.PipxToolSHA256 {
		return input, errors.New("pipx runtime intake subject differs from selected asset")
	}
	file, err := i.openAsset(artifact)
	if err != nil {
		return input, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			input = pipxRuntimeInput{}
			resultErr = errors.Join(resultErr, errors.New("close pipx runtime intake failed"))
		}
	}()
	body, err := io.ReadAll(ports.DownloadReader(ctx, io.LimitReader(file, runtimeidentity.PipxToolSize+1), runtimeidentity.PipxToolSize))
	if err != nil {
		return input, errors.Join(errors.New("read pipx runtime intake failed"), err, ctx.Err())
	}
	if err := ctx.Err(); err != nil {
		return input, err
	}
	value, err := pypi.ParsePipxRuntimeZIP(body)
	if err != nil {
		return input, err
	}
	plan, err := value.ObservationPlan()
	if err != nil {
		return input, err
	}
	if err := ctx.Err(); err != nil {
		return input, err
	}
	return pipxRuntimeInput{artifact: artifact, plan: plan}, nil
}
