package sandbox

import (
	"context"
	"errors"
	"runtime"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// PNPMResolver produces unapproved controls with the pinned native client.
// Registry binding, archive quarantine and publication remain separate steps.
type PNPMResolver struct {
	runner    CommandRunner
	endpoints EndpointResolver
	observer  TraceObserver
	probe     CapabilityProbe
	policy    ResolverPolicyService
	prepare   func(context.Context) (*pnpmPreparedTool, error)
}

type PNPMNativeSelection struct {
	controls  []domain.ProjectControlFile
	selection artifactnpm.PNPMLock
}

func (s PNPMNativeSelection) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), s.controls...)
}
func (s PNPMNativeSelection) Selection() artifactnpm.PNPMLock { return s.selection }

// PNPMResolverRuntimeIdentity is derived solely from the reviewed runtime lock.
func PNPMResolverRuntimeIdentity() string {
	return "pnpm:" + runtimeidentity.PNPMVersion + ";tool-sha256:" + runtimeidentity.PNPMToolSHA256 + ";node-image:" + runtimeidentity.NodeImageReference + ";runtime-lock-sha256:" + runtimeidentity.RuntimeLockSHA256
}

// BindProjectSnapshot connects the producer's frozen controls and independently
// authenticated registry metadata. The returned snapshot is still unapproved.
func (s PNPMNativeSelection) BindProjectSnapshot(ctx context.Context, source *artifactnpm.Resolver, install domain.InstallContext) (domain.ProjectDependencySnapshot, error) {
	frozen, err := s.BindInspectionContext(ctx, source, install)
	return frozen.Snapshot(), err
}

// BindInspectionContext preserves the complete native wiring and controls for
// the project's later static-all and closure-aware inspection consumers.
func (s PNPMNativeSelection) BindInspectionContext(ctx context.Context, source *artifactnpm.Resolver, install domain.InstallContext) (artifactnpm.PNPMInspectionContext, error) {
	if ctx == nil || source == nil || !install.Valid() || (install.Mode() != domain.InstallNewTarget && install.Mode() != domain.InstallNPMProject) {
		return artifactnpm.PNPMInspectionContext{}, errors.New("pnpm public source binding is unavailable")
	}
	bound, err := source.BindPNPMLock(ctx, s.selection)
	if err != nil {
		return artifactnpm.PNPMInspectionContext{}, err
	}
	frozen, err := bound.InspectionContext(install, s.controls, PNPMResolverRuntimeIdentity())
	if ctx.Err() != nil {
		return artifactnpm.PNPMInspectionContext{}, errors.Join(err, ctx.Err())
	}
	return frozen, err
}

func NewPNPMResolver(runner CommandRunner, endpoints EndpointResolver, observer TraceObserver, capability CapabilityProbe, policy ResolverPolicyService) (*PNPMResolver, error) {
	if runner == nil || endpoints == nil || observer == nil || capability == nil || policy == nil {
		return nil, errors.New("pnpm resolver infrastructure unavailable")
	}
	if _, ok := runner.(inputCommandRunner); !ok {
		return nil, errors.New("pnpm resolver input transport unavailable")
	}
	if _, ok := runner.(streamingDockerOutput); !ok {
		return nil, errors.New("pnpm resolver bounded transport unavailable")
	}
	return &PNPMResolver{admissionAwareRunner(runner), endpoints, observer, capability, policy, preparePNPMTool}, nil
}

func NewLinuxPNPMResolverWithExecutorAndPolicy(executor TrustedExecutor, observer TraceObserver, policy ResolverPolicyService) (*PNPMResolver, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return nil, errors.New("pnpm native target is unsupported")
	}
	return NewPNPMResolver(executor, systemEndpointResolver{}, observer, func(ctx context.Context) (Capability, error) {
		return probe(ctx, runtime.GOOS, executor)
	}, policy)
}

func (r *PNPMResolver) SelectProject(ctx context.Context, input artifactnpm.PNPMSelectionInput) (PNPMNativeSelection, error) {
	if r == nil || r.runner == nil || r.endpoints == nil || r.observer == nil || r.probe == nil || r.policy == nil || r.prepare == nil || ctx == nil || input.ClientVersion() != runtimeidentity.PNPMVersion {
		return PNPMNativeSelection{}, errors.New("pnpm selection input unavailable")
	}
	// Recheck the immutable value before all tool, DNS and resource operations.
	validated, err := artifactnpm.NewPNPMSelectionInput(input.Intent(), input.Selector(), input.Controls(), runtimeidentity.PNPMVersion)
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	var selectedResult PNPMNativeSelection
	err = r.withNativeProject(ctx, false, func(ctx context.Context, container, _ string, _ *closureVolume) error {
		if err := copyPNPMProjectControls(ctx, r.runner, container, validated.Controls()); err != nil {
			return err
		}
		arguments, err := pnpmSelectionArguments(validated)
		if err != nil {
			return err
		}
		if _, err := r.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, arguments...)...); err != nil {
			return errors.Join(errors.New("native pnpm selection failed"), err)
		}
		streams := r.runner.(streamingDockerOutput)
		controls := make([]domain.ProjectControlFile, 0, 2)
		for _, name := range []string{"package.json", "pnpm-lock.yaml"} {
			control, err := readPNPMProjectControl(ctx, streams, container, name)
			if err != nil {
				return err
			}
			controls = append(controls, control)
		}
		selected, err := validated.ParseOutput(controls[0].Body(), controls[1].Body())
		if err != nil {
			return err
		}
		selectedResult = PNPMNativeSelection{controls, selected}
		return nil
	})
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	return selectedResult, nil
}

func pnpmSelectionArguments(input artifactnpm.PNPMSelectionInput) ([]string, error) {
	if input.ClientVersion() != runtimeidentity.PNPMVersion {
		return nil, errors.New("pnpm client identity differs")
	}
	args := []string{pnpmToolGuestPath, "--dir", pnpmProjectDirectory, string(input.Intent()), "--lockfile-only", "--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--registry=https://registry.npmjs.org/", "--store-dir", pnpmProjectRoot + "/store", "--state-dir", pnpmProjectRoot + "/state", "--reporter=silent"}
	switch input.Intent() {
	case artifactnpm.PNPMInstall:
		if input.Controls()[1].Present() {
			args = append(args, "--frozen-lockfile")
		} else {
			args = append(args, "--no-frozen-lockfile")
		}
	case artifactnpm.PNPMAdd, artifactnpm.PNPMUpdate:
	default:
		return nil, errors.New("pnpm selection intent unsupported")
	}
	if input.Selector() != "" {
		args = append(args, "--", input.Selector())
	}
	return args, nil
}
