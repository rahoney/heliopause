package sandbox

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const bunProjectResolverProfile = "bun-project-resolver"
const bunProjectRoot = "/tmp/haa-bun-input"
const bunProjectDirectory = bunProjectRoot + "/project"
const bunProjectWorkspaceLimit = 128 << 20

type nativeProjectSpec struct{ name, root, directory, lock, profile string }

func nativeProjectIdentity(kind nativeRuntimeToolKind) (nativeProjectSpec, bool) {
	switch kind {
	case nativeRuntimePNPM:
		return nativeProjectSpec{"pnpm", pnpmProjectRoot, pnpmProjectDirectory, "pnpm-lock.yaml", pnpmResolverProfile}, true
	case nativeRuntimeBun:
		return nativeProjectSpec{"bun", bunProjectRoot, bunProjectDirectory, "bun.lock", bunProjectResolverProfile}, true
	default:
		return nativeProjectSpec{}, false
	}
}

func createNativeProjectVolume(ctx context.Context, runner CommandRunner, owner string, kind nativeRuntimeToolKind) (closureVolume, error) {
	switch kind {
	case nativeRuntimePNPM:
		return createPNPMProjectVolume(ctx, runner, owner)
	case nativeRuntimeBun:
		return createInputVolume(ctx, runner, owner+"-bun-project", runtimeidentity.BunToolSHA256, bunProjectWorkspaceLimit, false, projectBuildVolumeConfig{bunProjectRoot, false})
	default:
		return closureVolume{}, errors.New("native project kind unavailable")
	}
}

func nativeProjectCreateArguments(network string, hosts []string, native nativeRuntimeVolume, project closureVolume, earlyStore bool) ([]string, error) {
	switch native.kind {
	case nativeRuntimePNPM:
		return pnpmCreateArgumentsWithStore(network, hosts, native, project, earlyStore)
	case nativeRuntimeBun:
		if earlyStore || project.destination() != bunProjectRoot || project.projectBuildExecutable {
			return nil, errors.New("bun project mount or mode differs")
		}
		mount, err := native.mountArgument()
		if err != nil {
			return nil, err
		}
		projectMount, err := project.mountArgument(false)
		if err != nil {
			return nil, err
		}
		extra := []string{"--mount", mount, "--mount", projectMount, "--env", "BUN_INSTALL_CACHE_DIR=" + bunProjectRoot + "/cache", "--env", "XDG_CACHE_HOME=" + bunProjectRoot + "/cache", "--env", "TMPDIR=" + bunProjectRoot + "/tmp"}
		return nodeResolverCreateArguments(network, hosts, extra), nil
	default:
		return nil, errors.New("native project kind unavailable")
	}
}

// The current consumer is the native lock qualification. This uses the same
// resource owner as PNPM; it grants no source/graph approval to native output.
func newBunNativeProjectScope(runner CommandRunner, endpoints EndpointResolver, observer TraceObserver, probe CapabilityProbe, policy ResolverPolicyService) (*nativeProjectScope, error) {
	if runner == nil || endpoints == nil || observer == nil || probe == nil || policy == nil {
		return nil, errors.New("bun native infrastructure unavailable")
	}
	if _, ok := runner.(inputCommandRunner); !ok {
		return nil, errors.New("bun native input transport unavailable")
	}
	if _, ok := runner.(streamingDockerOutput); !ok {
		return nil, errors.New("bun native output transport unavailable")
	}
	return &nativeProjectScope{kind: nativeRuntimeBun, runner: admissionAwareRunner(runner), endpoints: endpoints, observer: observer, probe: probe, policy: policy, prepare: func(ctx context.Context) (nativeProjectTool, error) { return prepareBunTool(ctx) }}, nil
}
