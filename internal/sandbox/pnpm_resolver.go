package sandbox

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

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

func (r *PNPMResolver) SelectProject(ctx context.Context, input artifactnpm.PNPMSelectionInput) (result PNPMNativeSelection, resultErr error) {
	if r == nil || r.runner == nil || r.endpoints == nil || r.observer == nil || r.probe == nil || r.policy == nil || r.prepare == nil || ctx == nil || input.ClientVersion() != runtimeidentity.PNPMVersion {
		return PNPMNativeSelection{}, errors.New("pnpm selection input unavailable")
	}
	// Recheck the immutable value before all tool, DNS and resource operations.
	validated, err := artifactnpm.NewPNPMSelectionInput(input.Intent(), input.Selector(), input.Controls(), runtimeidentity.PNPMVersion)
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	defer func() {
		if err := ctx.Err(); err != nil {
			result = PNPMNativeSelection{}
			resultErr = errors.Join(resultErr, err)
		}
	}()
	if err := ctx.Err(); err != nil {
		return PNPMNativeSelection{}, err
	}
	capability, err := r.probe(ctx)
	if err != nil || !capability.Available {
		return PNPMNativeSelection{}, errors.Join(errors.New("pnpm fixed runtime unavailable"), err)
	}
	tool, err := r.prepare(ctx)
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	addresses, err := r.endpoints.Resolve(ctx, []string{"registry.npmjs.org"})
	if err != nil {
		return PNPMNativeSelection{}, errors.Join(errors.New("pnpm endpoint preflight failed"), ctx.Err())
	}
	hosts, err := npmNetworkArguments(addresses)
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	policy, err := NewResolverNetworkPolicy(r.runner, r.policy)
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	network, err := policy.Prepare(ctx, addresses)
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	var native nativeRuntimeVolume
	var project closureVolume
	var trace TraceReader
	name, owner, container := "", "", ""
	createAttempted := false
	defer func() {
		var cleanupErr error
		if createAttempted {
			cleanup, done := resolverCleanupContext()
			removeErr := removeOwnedNativeResolverContainer(cleanup, r.runner, name, owner, container, nativeRuntimePNPM)
			done()
			if removeErr != nil {
				cleanupErr = fmt.Errorf("pnpm resolver container cleanup unproven name=%s: %w", name, removeErr)
			}
		}
		if trace != nil {
			cleanup, done := resolverCleanupContext()
			_, limitation, diagnostic := collectTraceDiagnostic(cleanup, trace)
			done()
			if limitation != "" {
				cleanupErr = errors.Join(cleanupErr, fmt.Errorf("pnpm observation incomplete: %s", diagnostic))
			}
			for _, kind := range []string{"process-exec-unexpected", "filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
				if diagnostic.KindCounts[kind] != 0 {
					cleanupErr = errors.Join(cleanupErr, fmt.Errorf("pnpm selection contains forbidden observation %s", kind))
				}
			}
		}
		if project.name != "" {
			cleanup, done := resolverCleanupContext()
			cleanupErr = errors.Join(cleanupErr, closePNPMProjectVolume(cleanup, r.runner, &project))
			done()
		}
		if native.name != "" {
			cleanup, done := resolverCleanupContext()
			cleanupErr = errors.Join(cleanupErr, native.close(cleanup, r.runner))
			done()
		}
		cleanup, done := resolverCleanupContext()
		cleanupErr = errors.Join(cleanupErr, policy.Close(cleanup))
		done()
		if cleanupErr != nil {
			result = PNPMNativeSelection{}
			resultErr = errors.Join(resultErr, cleanupErr)
		}
	}()
	native, err = createPNPMRuntimeVolume(ctx, r.runner)
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	owner, name = session.String(), "haa-pnpm-resolver-"+session.String()
	project, err = createPNPMProjectVolume(ctx, r.runner, owner)
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	arguments, err := pnpmCreateArguments(network, hosts, native, project)
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	listed, err := r.runner.Output(ctx, "docker", "ps", "--all", "--quiet", "--no-trunc", "--filter", "name=^/"+name+"$")
	if err != nil || len(listed) > 1024 || strings.TrimSpace(string(listed)) != "" {
		return PNPMNativeSelection{}, errors.New("pnpm resolver name preflight unavailable or nonempty")
	}
	arguments = append([]string{"create", "--name", name, "--label", pnpmResolverOwnerLabel + "=" + owner}, arguments[1:]...)
	createAttempted = true
	created, err := r.runner.Output(ctx, "docker", arguments...)
	if err != nil || !exactObservationContainerID(strings.TrimSpace(string(created))) {
		return PNPMNativeSelection{}, errors.Join(errors.New("pnpm create outcome unavailable"), err, ctx.Err())
	}
	container = strings.TrimSpace(string(created))
	if err := verifyOwnedNativeResolverContainer(ctx, r.runner, name, owner, container, nativeRuntimePNPM); err != nil {
		return PNPMNativeSelection{}, err
	}
	trace, err = startTrace(ctx, r.observer, container, pnpmResolverProfile)
	if err != nil {
		return PNPMNativeSelection{}, errors.Join(errors.New("pnpm observer start failed"), err)
	}
	if _, err := r.runner.Output(ctx, "docker", "start", container); err != nil {
		return PNPMNativeSelection{}, errors.Join(errors.New("pnpm container start failed"), err)
	}
	if err := awaitMountAnchors(ctx, r.observer, container); err != nil {
		return PNPMNativeSelection{}, err
	}
	if err := awaitBoundaryHelper(ctx, r.runner, container); err != nil {
		return PNPMNativeSelection{}, err
	}
	if err := native.verifyTarget(ctx, r.runner, container); err != nil {
		return PNPMNativeSelection{}, err
	}
	if err := project.verifyContainerMount(ctx, r.runner, container, false); err != nil {
		return PNPMNativeSelection{}, err
	}
	if err := project.verifyAttachments(ctx, r.runner, map[string]bool{container: false}); err != nil {
		return PNPMNativeSelection{}, err
	}
	if err := tool.introduce(ctx, r.runner, container); err != nil {
		return PNPMNativeSelection{}, err
	}
	if err := native.verifyTarget(ctx, r.runner, container); err != nil {
		return PNPMNativeSelection{}, err
	}
	if err := copyPNPMProjectControls(ctx, r.runner, container, validated.Controls()); err != nil {
		return PNPMNativeSelection{}, err
	}
	arguments, err = pnpmSelectionArguments(validated)
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	if _, err := r.runner.Output(ctx, "docker", boundaryExecArguments(container, boundaryLaunchMode, arguments...)...); err != nil {
		return PNPMNativeSelection{}, errors.Join(errors.New("native pnpm selection failed"), err)
	}
	streams := r.runner.(streamingDockerOutput)
	controls := make([]domain.ProjectControlFile, 0, 2)
	for _, name := range []string{"package.json", "pnpm-lock.yaml"} {
		control, err := readPNPMProjectControl(ctx, streams, container, name)
		if err != nil {
			return PNPMNativeSelection{}, err
		}
		controls = append(controls, control)
	}
	selected, err := validated.ParseOutput(controls[0].Body(), controls[1].Body())
	if err != nil {
		return PNPMNativeSelection{}, err
	}
	if err := project.verifyAttachments(ctx, r.runner, map[string]bool{container: false}); err != nil {
		return PNPMNativeSelection{}, err
	}
	return PNPMNativeSelection{controls, selected}, nil
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
