package sandbox

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const pnpmResolverProfile = "pnpm-resolver"
const pnpmProjectRoot = "/tmp/haa-pnpm-input"
const pnpmProjectDirectory = pnpmProjectRoot + "/project"

func createPNPMProjectVolume(ctx context.Context, runner CommandRunner, owner string) (closureVolume, error) {
	return createInputVolume(ctx, runner, owner+"-pnpm-project", runtimeidentity.PNPMToolSHA256, 128<<20, false, projectBuildVolumeConfig{pnpmProjectRoot, false})
}

func closePNPMProjectVolume(ctx context.Context, runner CommandRunner, volume *closureVolume) error {
	return closeWritableOutputVolume(ctx, runner, volume, pnpmProjectRoot)
}

// The pinned helper and private project mounts reuse the qualified Node boundary.
// PNPM never inherits Python closure or artifact execution authority.
func pnpmCreateArguments(network string, hostArguments []string, volume nativeRuntimeVolume, project closureVolume) ([]string, error) {
	if volume.kind != nativeRuntimePNPM {
		return nil, errors.New("pnpm requires its own attested runtime volume")
	}
	mount, err := volume.mountArgument()
	if err != nil {
		return nil, err
	}
	if project.destination() != pnpmProjectRoot || project.projectBuildExecutable {
		return nil, errors.New("pnpm project mount is invalid")
	}
	projectMount, err := project.mountArgument(false)
	if err != nil {
		return nil, err
	}
	return nodeResolverCreateArguments(network, hostArguments, []string{"--mount", mount, "--mount", projectMount,
		"--env", "NPM_CONFIG_MANAGE_PACKAGE_MANAGER_VERSIONS=false",
		"--env", "PNPM_CONFIG_UPDATE_NOTIFIER=false"}), nil
}
