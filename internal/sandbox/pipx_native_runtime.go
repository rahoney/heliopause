package sandbox

import (
	"errors"
	"slices"
	"strconv"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

const pipxNativeResolverProfile = "pipx-native-resolver"

// These closed arguments provide no execution admission. The lifecycle must
// attest every mount/attachment and use the dedicated external profile before
// a native target can run. There is no caller network, cwd, argv or env input.
func pipxNativeCreateArguments(data *pipxDataVolume, wheels *pipxWheelVolume, runtime nativeRuntimeVolume, output closureVolume) ([]string, error) {
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	if !data.valid() || !data.sealed || !wheels.valid() || !wheels.sealed || runtime.kind != nativeRuntimeUV || output.destination() != pythonSitePath || output.capacity != policy.PromotionTmpfs() || output.manifestID != wheels.input.id || output.goBuild || output.goBuildOutput || output.projectBuildExecutable {
		return nil, errors.New("pipx native runtime requires sealed inputs and same-frozen-selection output")
	}
	zipMount, err := data.readerMount()
	if err != nil {
		return nil, err
	}
	wheelMount, err := wheels.readerMount()
	if err != nil {
		return nil, err
	}
	runtimeMount, err := runtime.mountArgument()
	if err != nil {
		return nil, err
	}
	outputMount, err := output.mountArgument(false)
	if err != nil {
		return nil, err
	}
	base := pythonResolverCreateArguments("none", nil, []string{"--mount", runtimeMount}, []string{"--mount", outputMount, "--mount", zipMount, "--mount", wheelMount})
	environment := isolatedContainerEnvironmentArguments()
	tail := []string{pythonImageReference, "/bin/sh", "-ceu", boundaryContainerCommand()}
	start := len(base) - len(environment) - len(tail)
	if start < 0 || !slices.Equal(base[start:start+len(environment)], environment) || !slices.Equal(base[start+len(environment):], tail) {
		return nil, errors.New("canonical resolver startup layout changed")
	}
	narrowed := 0
	for i := 0; i < start-1; i++ {
		if base[i] == "--ulimit" {
			if base[i+1] != "cpu=60:60" {
				return nil, errors.New("canonical resolver CPU contract changed")
			}
			seconds := strconv.Itoa(policy.RuntimeCPUSecs())
			base[i+1] = "cpu=" + seconds + ":" + seconds
			narrowed++
		}
	}
	if narrowed != 1 {
		return nil, errors.New("canonical resolver CPU contract is absent or duplicated")
	}
	result := append(base[:start:start], pipxNativeEnvironmentArguments()...)
	result = append(result, "--workdir", "/")
	return append(result, tail...), nil
}
