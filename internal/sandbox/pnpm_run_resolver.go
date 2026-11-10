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

const pnpmRunCacheBase = pnpmProjectRoot + "/cache/pnpm"
const pnpmRunQueryPath = "/haa-runtime/helox-pnpm-inert-query"

func pnpmRunSelectionArguments(reference domain.ArtifactReference) ([]string, error) {
	if err := artifactnpm.ValidatePNPMRunReference(reference); err != nil {
		return nil, err
	}
	return []string{pnpmToolGuestPath, "--dir", pnpmProjectRoot, "dlx", "--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--registry=https://registry.npmjs.org/", "--store-dir", pnpmProjectRoot + "/store", "--state-dir", pnpmProjectRoot + "/state", "--reporter=silent", "--package", reference.Locator(), "--", pnpmRunQueryPath}, nil
}

// SelectToolRun collects a fresh native cold cache as unapproved DATA. Only
// a missing absolute command in the sealed helper mount is requested after
// installation; no artifact bin or caller argv is passed.
// Independent source binding and current quarantine/execution remain separate.
func (r *PNPMResolver) SelectToolRun(ctx context.Context, reference domain.ArtifactReference) (artifactnpm.PNPMRunCache, error) {
	if ctx == nil {
		return artifactnpm.PNPMRunCache{}, errors.New("pnpm one-off selection requires context")
	}
	args, err := pnpmRunSelectionArguments(reference)
	if err != nil {
		return artifactnpm.PNPMRunCache{}, err
	}
	var selected artifactnpm.PNPMRunCache
	err = r.withNativeProject(ctx, true, func(ctx context.Context, container, _ string, _ *closureVolume) error {
		_, err := r.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, args...)...)
		var exited *exec.ExitError
		if !errors.As(err, &exited) || !pnpmRunMissingTerminal(exited.ExitCode(), exited.Stderr) {
			return errors.Join(errors.New("native pnpm one-off selection failed"), err)
		}
		output := nativeToolArchiveOutput{limit: artifactnpm.PNPMLayoutArchiveLimit}
		if err := r.runner.(streamingDockerOutput).RunOutput(ctx, &output, "docker", "cp", container+":"+pnpmRunCacheBase+"/dlx", "-"); err != nil || output.failed || ctx.Err() != nil {
			return errors.Join(errors.New("pnpm one-off cache readback incomplete"), err, ctx.Err(), context.Cause(ctx))
		}
		selected, err = artifactnpm.ParsePNPMRunCache(ctx, output.body, reference, runtimeidentity.PNPMVersion)
		return err
	})
	if err != nil {
		return artifactnpm.PNPMRunCache{}, err
	}
	return selected, nil
}

// This classifies the pinned producer's preparation diagnostic only. Cache
// validation and the scope's complete external trace/cleanup remain required.
func pnpmRunMissingTerminal(code int, diagnostic []byte) bool {
	return code == 1 && len(diagnostic) <= 4096 &&
		bytes.Contains(diagnostic, []byte("ERR_PNPM_DLX_COMMAND_NOT_FOUND")) &&
		bytes.Contains(diagnostic, []byte("Command \""+pnpmRunQueryPath+"\" not found"))
}
