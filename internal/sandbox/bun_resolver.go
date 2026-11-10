package sandbox

import (
	"context"
	"errors"
	"runtime"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// BunResolver produces unapproved controls through the existing fixed native
// resource owner. Registry binding, quarantine and publication remain separate.
type BunResolver struct{ scope *nativeProjectScope }

type BunNativeSelection struct {
	controls  []domain.ProjectControlFile
	selection artifactnpm.BunLock
}

func (s BunNativeSelection) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), s.controls...)
}
func (s BunNativeSelection) Selection() artifactnpm.BunLock { return s.selection }

func BunResolverRuntimeIdentity() string {
	return "bun:" + runtimeidentity.BunVersion + ";tool-sha256:" + runtimeidentity.BunToolSHA256 + ";node-image:" + runtimeidentity.NodeImageReference + ";runtime-lock-sha256:" + runtimeidentity.RuntimeLockSHA256
}

func (s BunNativeSelection) BindProjectSnapshot(ctx context.Context, source *artifactnpm.Resolver, install domain.InstallContext) (domain.ProjectDependencySnapshot, error) {
	frozen, err := s.BindInspectionContext(ctx, source, install)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	return frozen.Snapshot(), nil
}

// BindInspectionContext preserves native tree placements alongside the exact
// independently source-bound snapshot. Archive inspection remains required.
func (s BunNativeSelection) BindInspectionContext(ctx context.Context, source *artifactnpm.Resolver, install domain.InstallContext) (artifactnpm.BunInspectionContext, error) {
	if ctx == nil || source == nil || !install.Valid() || (install.Mode() != domain.InstallNewTarget && install.Mode() != domain.InstallNPMProject) {
		return artifactnpm.BunInspectionContext{}, errors.New("bun public source binding unavailable")
	}
	bound, err := source.BindBunLock(ctx, s.selection)
	if err != nil {
		return artifactnpm.BunInspectionContext{}, err
	}
	frozen, err := bound.InspectionContext(install, s.controls, BunResolverRuntimeIdentity())
	if ctx.Err() != nil {
		return artifactnpm.BunInspectionContext{}, errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	return frozen, err
}

func NewBunResolver(runner CommandRunner, endpoints EndpointResolver, observer TraceObserver, capability CapabilityProbe, policy ResolverPolicyService) (*BunResolver, error) {
	scope, err := newBunNativeProjectScope(runner, endpoints, observer, capability, policy)
	if err != nil {
		return nil, err
	}
	return &BunResolver{scope}, nil
}

func NewLinuxBunResolverWithExecutorAndPolicy(executor TrustedExecutor, observer TraceObserver, policy ResolverPolicyService) (*BunResolver, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return nil, errors.New("bun native target unsupported")
	}
	return NewBunResolver(executor, systemEndpointResolver{}, observer, func(ctx context.Context) (Capability, error) { return probe(ctx, runtime.GOOS, executor) }, policy)
}

func (r *BunResolver) SelectProject(ctx context.Context, input artifactnpm.BunSelectionInput) (BunNativeSelection, error) {
	if r == nil || r.scope == nil || r.scope.kind != nativeRuntimeBun || ctx == nil || input.ClientVersion() != runtimeidentity.BunVersion {
		return BunNativeSelection{}, errors.New("bun selection input unavailable")
	}
	if err := ctx.Err(); err != nil {
		return BunNativeSelection{}, errors.Join(err, context.Cause(ctx))
	}
	validated, err := artifactnpm.NewBunSelectionInput(input.Intent(), input.Selector(), input.Controls(), runtimeidentity.BunVersion)
	if err != nil {
		return BunNativeSelection{}, err
	}
	var selected BunNativeSelection
	err = r.scope.run(ctx, false, func(ctx context.Context, container, _ string, _ *closureVolume) error {
		if err := copyBunProjectControls(ctx, r.scope.runner, container, validated.Controls()); err != nil {
			return err
		}
		args, err := bunSelectionArguments(validated)
		if err != nil {
			return err
		}
		if _, err := r.scope.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, args...)...); err != nil {
			return errors.Join(errors.New("native bun selection failed"), err)
		}
		controls, err := readBunProjectControls(ctx, r.scope.runner.(streamingDockerOutput), container)
		if err != nil {
			return err
		}
		lock, err := validated.ParseOutput(controls[0].Body(), controls[1])
		if err != nil {
			return err
		}
		selected = BunNativeSelection{controls, lock}
		return nil
	})
	if err != nil {
		return BunNativeSelection{}, err
	}
	return selected, nil
}

func bunSelectionArguments(input artifactnpm.BunSelectionInput) ([]string, error) {
	if input.ClientVersion() != runtimeidentity.BunVersion || len(input.Controls()) != 2 {
		return nil, errors.New("bun client identity differs")
	}
	args := []string{bunToolGuestPath, "--cwd=" + bunProjectDirectory, string(input.Intent()), "--lockfile-only", "--ignore-scripts", "--no-env-file", "--registry=https://registry.npmjs.org/", "--cache-dir=" + bunProjectRoot + "/cache", "--backend=copyfile", "--linker=hoisted", "--save-text-lockfile", "--no-progress", "--no-summary"}
	switch input.Intent() {
	case artifactnpm.BunInstall:
		if input.Controls()[1].Present() {
			args = append(args, "--frozen-lockfile")
		}
	case artifactnpm.BunAdd, artifactnpm.BunUpdate:
	default:
		return nil, errors.New("bun selection intent unsupported")
	}
	if input.Selector() != "" {
		args = append(args, "--", input.Selector())
	}
	return args, nil
}
