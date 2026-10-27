package sandbox

import (
	"context"
	"errors"
	"strings"
)

const uvResolverProfile = "uv-pip-resolver"

func uvCreateArguments(network string, hostArguments []string, volume uvRuntimeVolume, output closureVolume) ([]string, error) {
	mount, err := volume.mountArgument()
	if err != nil {
		return nil, err
	}
	if output.destination() != pythonSitePath {
		return nil, errors.New("uv output volume destination is invalid")
	}
	outputMount, err := output.mountArgument(false)
	if err != nil {
		return nil, err
	}
	return pythonResolverCreateArguments(network, hostArguments, []string{"--mount", mount}, []string{"--mount", outputMount}), nil
}

func closeUVOutputVolume(ctx context.Context, runner CommandRunner, volume *closureVolume) error {
	return closeWritableOutputVolume(ctx, runner, volume, pythonSitePath)
}

func closeWritableOutputVolume(ctx context.Context, runner CommandRunner, volume *closureVolume, destination string) error {
	if destination != pythonSitePath && destination != pnpmProjectRoot && destination != bunProjectRoot && destination != pipxRuntimeRoot && destination != pipxNativeWheels {
		return errors.New("unknown private output destination")
	}
	if ctx == nil || runner == nil || volume == nil || volume.destination() != destination || volume.verify(ctx, runner) != nil {
		return errors.New("uv output volume cleanup identity is unavailable")
	}
	listed, err := runner.Output(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "volume="+volume.name, "--format", "{{.ID}}")
	if err != nil || len(listed) > 1024 || strings.TrimSpace(string(listed)) != "" {
		return errors.New("uv output volume cleanup attachment inventory is unavailable or nonempty")
	}
	_, err = runner.Output(ctx, "docker", "volume", "rm", volume.name)
	return err
}
