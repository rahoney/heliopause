package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os/exec"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const bunRunCacheBase = bunProjectRoot + "/cache/.bunx-1000"
const bunRunQueryPath = "/haa-runtime/helox-bunx-inert-query"

func bunRunSelectionArguments(reference domain.ArtifactReference) ([]string, error) {
	if err := artifactnpm.ValidateBunRunReference(reference); err != nil {
		return nil, err
	}
	return []string{bunToolGuestPath, "--cwd=" + bunProjectRoot, "x", "--ignore-scripts", "--registry=https://registry.npmjs.org/", "--package=" + reference.Locator(), bunRunQueryPath}, nil
}

// SelectToolRun collects fresh native DATA through the existing Bun resource,
// network and complete-observation owner. Only a sealed missing command is
// requested. Independent source binding, quarantine and execution are separate.
func (r *BunResolver) SelectToolRun(ctx context.Context, reference domain.ArtifactReference) (artifactnpm.BunRunCache, error) {
	if r == nil || r.scope == nil || r.scope.kind != nativeRuntimeBun || ctx == nil {
		return artifactnpm.BunRunCache{}, errors.New("bun one-off resolver requires context and fixed infrastructure")
	}
	args, err := bunRunSelectionArguments(reference)
	if err != nil {
		return artifactnpm.BunRunCache{}, err
	}
	var selected artifactnpm.BunRunCache
	err = r.scope.run(ctx, false, func(ctx context.Context, container, _ string, _ *closureVolume) error {
		_, producerErr := r.scope.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, args...)...)
		var exited *exec.ExitError
		if !errors.As(producerErr, &exited) || !bunRunMissingTerminal(exited.ExitCode(), exited.Stderr) {
			return errors.Join(errors.New("native bun one-off selection failed"), producerErr)
		}
		output := nativeToolArchiveOutput{limit: artifactnpm.PNPMLayoutArchiveLimit}
		if err := r.scope.runner.(streamingDockerOutput).RunOutput(ctx, &output, "docker", "cp", container+":"+bunRunCacheBase, "-"); err != nil || output.failed || ctx.Err() != nil {
			return errors.Join(errors.New("bun one-off cache readback incomplete"), err, ctx.Err(), context.Cause(ctx))
		}
		selected, err = artifactnpm.ParseBunRunCache(ctx, output.body, reference, runtimeidentity.BunVersion)
		return err
	})
	if err != nil {
		return artifactnpm.BunRunCache{}, err
	}
	return selected, nil
}

func bunRunMissingTerminal(code int, diagnostic []byte) bool {
	return code == 1 && len(diagnostic) <= 4096 &&
		bytes.Contains(diagnostic, []byte("does not provide a binary named "+bunRunQueryPath))
}

// ResolveToolRun binds the complete native selection to the independent public
// source. Ephemeral execution has no installation target or project snapshot.
func (r *BunResolver) ResolveToolRun(ctx context.Context, reference domain.ArtifactReference, source *artifactnpm.Resolver) (artifactnpm.BunInspectionContext, error) {
	if ctx == nil || source == nil {
		return artifactnpm.BunInspectionContext{}, errors.New("bun one-off resolution requires context and public source")
	}
	cache, err := r.SelectToolRun(ctx, reference)
	if err != nil {
		return artifactnpm.BunInspectionContext{}, err
	}
	return source.BindBunRunCache(ctx, cache, reference, BunResolverRuntimeIdentity())
}
