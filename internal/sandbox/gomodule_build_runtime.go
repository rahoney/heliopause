package sandbox

import (
	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
)

const (
	goBuildProfile              = "go-module-build"
	goBuildGuestInput           = "/tmp/haa-go-input"
	goBuildGuestProject         = goBuildGuestInput + "/project"
	goBuildGuestCache           = goBuildGuestInput + "/cache"
	goBuildGuestOutput          = "/tmp/haa-go-output"
	goBuildInputCapacity  int64 = 512 << 20
	goBuildOutputCapacity int64 = artifactgo.MaxBuildOutputBytes
)

// Build and resolver share the immutable toolchain image, but their process
// roles and network contracts are separate. Project compilation is ARTIFACT.
func goBuildCreateArguments() []string {
	return goBuildCreateArgumentsForCache(goResolverGuestCache)
}

func goBuildCreateArgumentsForCache(cache string) []string {
	arguments := []string{"create", "--pull", "never", "--runtime", gVisorRuntimeName, "--network", "none",
		"--read-only", "--cap-drop", "ALL", "--cap-add", "SETUID", "--cap-add", "SETGID", "--cap-add", "SETPCAP", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", "512m", "--cpus", "1",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=512m,uid=1000,gid=1000,mode=0700", "--tmpfs", boundaryHelperMount, "--workdir", "/tmp"}
	arguments = append(arguments, isolatedContainerEnvironmentArguments()...)
	environment, _ := artifactgo.BuildEnvironmentForCache(cache)
	environment = append(environment, "HOME=/tmp", "XDG_CONFIG_HOME=/tmp/.config", "GOPATH=/tmp/haa-go-path", "GOCACHE=/tmp/haa-go-build", "TMPDIR=/tmp", "GOROOT=/usr/local/go", "GODEBUG=netdns=go")
	for _, value := range environment {
		arguments = append(arguments, "--env", value)
	}
	return append(arguments, PinnedGoRuntime().ImageReference, "/bin/sh", "-ceu", boundaryContainerCommand())
}

func goBuildVolumesCreateArguments(volume closureVolume, output *closureVolume, readOnly bool) ([]string, error) {
	mount, err := volume.mountArgument(readOnly)
	if err != nil || !volume.goBuild || volume.goBuildOutput {
		return nil, errGoBuildInput
	}
	arguments := goBuildCreateArgumentsForCache(goBuildGuestCache)
	// Mount options precede the image and fixed keeper command.
	image := len(arguments) - 4
	result := append([]string(nil), arguments[:image]...)
	result = append(result, "--mount", mount)
	if output != nil {
		if !output.goBuild || !output.goBuildOutput || output.name == volume.name || output.transaction != volume.transaction+"-output" {
			return nil, errGoBuildInput
		}
		outputMount, err := output.mountArgument(false)
		if err != nil {
			return nil, err
		}
		result = append(result, "--mount", outputMount)
	}
	return append(result, arguments[image:]...), nil
}
