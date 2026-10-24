package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strings"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const (
	resolverProjectDir = "/tmp/haa-resolver"
)

var (
	resolverNPMVersion      = runtimeidentity.NodeNPMVersion
	resolverRuntimeIdentity = runtimeidentity.NodeImageReference + ";npm:" + runtimeidentity.NodeNPMVersion
)

// EndpointResolver supplies the trusted, preflight-resolved address set for
// the registry endpoints allowed by one resolver network policy.
type EndpointResolver interface {
	Resolve(context.Context, []string) ([]netip.Addr, error)
}

// NPMResolver is a disposable Docker resolver. It returns the normalized graph
// and bounded opaque selection controls for exact publication. Runtime output
// never becomes Evidence, Policy or Host-execution authority.
type NPMResolver struct {
	runner    CommandRunner
	endpoints EndpointResolver
	observer  TraceObserver
	policy    ResolverPolicyService
}

func NewNPMResolver(runner CommandRunner, endpoints EndpointResolver, policy ResolverPolicyService) (*NPMResolver, error) {
	if runner == nil || endpoints == nil || policy == nil {
		return nil, errors.New("npm resolver requires command runner, endpoint resolver and network policy service")
	}
	if _, ok := runner.(inputCommandRunner); !ok {
		return nil, errors.New("npm resolver requires input stream runner")
	}
	return &NPMResolver{runner: admissionAwareRunner(runner), endpoints: endpoints, policy: policy}, nil
}

// NewNPMResolverWithObserver constructs the production resolver with the
// process-scoped trace receiver required for its runsc-trace container.
func NewNPMResolverWithObserver(runner CommandRunner, endpoints EndpointResolver, observer TraceObserver, policy ResolverPolicyService) (*NPMResolver, error) {
	resolver, err := NewNPMResolver(runner, endpoints, policy)
	if err != nil || observer == nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("npm resolver requires process-scoped observer")
	}
	resolver.observer = observer
	return resolver, nil
}

// NewLinuxNPMResolverWithExecutorAndPolicy constructs the production resolver
// with the ordinary-to-privileged typed network policy port.
func NewLinuxNPMResolverWithExecutorAndPolicy(executor interface {
	CommandRunner
	inputCommandRunner
}, observer TraceObserver, policy ResolverPolicyService) (*NPMResolver, error) {
	if policy == nil {
		return nil, errors.New("npm resolver requires network policy service")
	}
	resolver, err := NewNPMResolverWithObserver(executor, systemEndpointResolver{}, observer, policy)
	if err != nil {
		return nil, err
	}
	resolver.policy = policy
	return resolver, nil
}

type systemEndpointResolver struct{}

func (systemEndpointResolver) Resolve(ctx context.Context, names []string) ([]netip.Addr, error) {
	if ctx == nil || len(names) == 0 {
		return nil, errors.New("resolver endpoint names are required")
	}
	seen := map[netip.Addr]bool{}
	addresses := make([]netip.Addr, 0, len(names))
	for _, name := range names {
		resolved, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", name)
		if err != nil {
			return nil, errors.New("resolver endpoint lookup failed")
		}
		for _, address := range resolved {
			if !seen[address] {
				seen[address] = true
				addresses = append(addresses, address)
			}
		}
	}
	if err := validateResolverEndpoints(addresses); err != nil {
		return nil, errors.New("resolver endpoint lookup is unsafe")
	}
	return addresses, nil
}

// ResolveDependencies runs npm only in a policy-protected disposable
// container, parses its lockfile locally, and always disposes policy resources.
func (r *NPMResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, _ domain.InstallContext) (domain.DependencyResolution, error) {
	return r.resolveDependencies(ctx, reference, nil)
}

// ResolveDependencyUpdate selects once from frozen existing public controls.
func (r *NPMResolver) ResolveDependencyUpdate(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext, controls []domain.ProjectControlFile) (resolution domain.DependencyResolution, resultErr error) {
	if ctx == nil || !install.Valid() || install.Mode() != domain.InstallNPMProject || len(controls) != 2 {
		return domain.DependencyResolution{}, errors.New("npm update requires an existing project and two frozen controls")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	defer func() {
		if err := ctx.Err(); err != nil {
			resolution = domain.DependencyResolution{}
			resultErr = errors.Join(resultErr, err)
		}
	}()
	if err := ctx.Err(); err != nil {
		return domain.DependencyResolution{}, err
	}
	values := map[string][]byte{}
	for _, control := range controls {
		if !control.Present() || control.Name() != "package.json" && control.Name() != "package-lock.json" || values[control.Name()] != nil {
			return domain.DependencyResolution{}, errors.New("npm update controls are invalid")
		}
		values[control.Name()] = control.Body()
	}
	if _, err := artifactnpm.ParseProjectControls(reference, values["package.json"], values["package-lock.json"]); err != nil {
		return domain.DependencyResolution{}, err
	}
	return r.resolveDependencies(ctx, reference, values)
}

func (r *NPMResolver) resolveDependencies(ctx context.Context, reference domain.ArtifactReference, original map[string][]byte) (resolution domain.DependencyResolution, resultErr error) {
	if ctx == nil || reference.Source().String() != "npm" {
		return domain.DependencyResolution{}, errors.New("valid npm resolver request is required")
	}
	addresses, err := r.endpoints.Resolve(ctx, []string{"registry.npmjs.org"})
	if err != nil {
		return domain.DependencyResolution{}, errors.New("resolver endpoint preflight failed")
	}
	hostArguments, err := npmNetworkArguments(addresses)
	if err != nil {
		return domain.DependencyResolution{}, errors.New("resolver endpoint preflight failed")
	}
	policy, err := NewResolverNetworkPolicy(r.runner, r.policy)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	network, err := policy.Prepare(ctx, addresses)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	var cleanupErr error
	containerID := ""
	var trace TraceReader
	defer func() {
		if containerID != "" {
			cleanupCtx, cancel := resolverCleanupContext()
			_, removeErr := r.runner.Output(cleanupCtx, "docker", "rm", "--force", containerID)
			cancel()
			if removeErr != nil {
				cleanupErr = errors.New("resolver container cleanup failed")
			}
			if trace != nil {
				collectCtx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
				_, limitation, diagnostic := collectTraceDiagnostic(collectCtx, trace)
				cancel()
				if limitation != "" {
					cleanupErr = errors.Join(cleanupErr, fmt.Errorf("npm resolver observation is incomplete: %s", diagnostic))
				}
			}
		}
		cleanupCtx, cancel := resolverCleanupContext()
		closeErr := policy.Close(cleanupCtx)
		cancel()
		if closeErr != nil {
			cleanupErr = errors.Join(cleanupErr, closeErr)
		}
		if cleanupErr != nil {
			resolution = domain.DependencyResolution{}
			if resultErr != nil {
				resultErr = errors.Join(resultErr, cleanupErr)
			} else {
				resultErr = cleanupErr
			}
		}
	}()

	created, err := r.runner.Output(ctx, "docker", npmCreateArguments(network, hostArguments)...)
	if err != nil || !containerIDPattern.MatchString(strings.TrimSpace(string(created))) {
		return domain.DependencyResolution{}, errors.New("create resolver container failed")
	}
	containerID = strings.TrimSpace(string(created))
	if r.observer == nil {
		return domain.DependencyResolution{}, errors.New("resolver observation is unavailable")
	}
	trace, err = startTrace(ctx, r.observer, containerID, "npm-lifecycle")
	if err != nil {
		return domain.DependencyResolution{}, errors.New("start resolver observer failed")
	}
	if _, err := r.runner.Output(ctx, "docker", "start", containerID); err != nil {
		return domain.DependencyResolution{}, errors.New("start resolver container failed")
	}
	if err := awaitBoundaryHelper(ctx, r.runner, containerID); err != nil {
		return domain.DependencyResolution{}, errors.New("resolver boundary helper failed")
	}
	if err := awaitMountAnchors(ctx, r.observer, containerID); err != nil {
		return domain.DependencyResolution{}, errors.New("resolver observer mount anchors failed")
	}
	version, err := r.runner.Output(ctx, "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "npm", "--version")...)
	if err != nil || strings.TrimSpace(string(version)) != resolverNPMVersion {
		return domain.DependencyResolution{}, errors.New("resolver npm runtime version mismatch")
	}
	input, ok := r.runner.(inputCommandRunner)
	if !ok {
		return domain.DependencyResolution{}, errors.New("resolver input stream unavailable")
	}
	packageName, err := artifactnpm.RequestedPackageName(reference)
	if err != nil {
		return domain.DependencyResolution{}, errors.New("resolver package reference is invalid")
	}
	packageSpec, err := artifactnpm.RequestedPackageSpec(reference)
	if err != nil {
		return domain.DependencyResolution{}, errors.New("resolver package reference is invalid")
	}
	manifest := []byte("{\"name\":\"haa-resolver\",\"version\":\"1.0.0\",\"private\":true,\"dependencies\":{\"" + packageName + "\":\"" + packageSpec + "\"}}")
	if original != nil {
		manifest = original["package.json"]
	}
	if err := input.RunInput(ctx, bytes.NewReader(manifest), "docker", boundaryInputExecArguments(containerID, boundaryLaunchMode, "/bin/sh", "-ceu", "umask 077; mkdir -p "+resolverProjectDir+"; cat > "+resolverProjectDir+"/package.json")...); err != nil {
		return domain.DependencyResolution{}, errors.New("write resolver manifest failed")
	}
	selection := "npm install"
	selectionFlags := ""
	if original != nil {
		if err := input.RunInput(ctx, bytes.NewReader(original["package-lock.json"]), "docker", boundaryInputExecArguments(containerID, boundaryLaunchMode, "/bin/sh", "-ceu", "umask 077; cat > "+resolverProjectDir+"/package-lock.json")...); err != nil {
			return domain.DependencyResolution{}, errors.New("write frozen npm lock failed")
		}
		// The supported subset has one direct root; zero/one CLI selections both
		// update that complete graph, respecting all original dependency constraints.
		selection = "npm update"
		selectionFlags = " --save=false"
	}
	command := "cd " + resolverProjectDir + "; HOME=/tmp npm_config_cache=/tmp/cache " + selection + " --package-lock-only --ignore-scripts --no-audit --no-fund" + selectionFlags + " --registry=https://registry.npmjs.org/ --userconfig=/tmp/haa-user.npmrc --globalconfig=/tmp/haa-global.npmrc"
	if _, err := r.runner.Output(ctx, "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/sh", "-ceu", command)...); err != nil {
		return domain.DependencyResolution{}, errors.New("run locked npm resolution failed")
	}
	lock, err := r.runner.Output(ctx, "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "cat", resolverProjectDir+"/package-lock.json")...)
	if err != nil {
		return domain.DependencyResolution{}, errors.New("read resolver lockfile failed")
	}
	graph, parseErr := artifactnpm.ParseProjectControls(reference, manifest, lock)
	if parseErr != nil {
		return domain.DependencyResolution{}, fmt.Errorf("parse resolver lockfile: %w", parseErr)
	}
	sum := sha256.Sum256(lock)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	resolution, err = domain.NewDependencyResolution(graph, resolverRuntimeIdentity, digest)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	controls := make([]domain.ProjectControlFile, 0, 2)
	for _, item := range []struct {
		name string
		body []byte
	}{{"package.json", manifest}, {"package-lock.json", lock}} {
		control, err := domain.NewProjectControlFile(item.name, item.body, true)
		if err != nil {
			return domain.DependencyResolution{}, err
		}
		controls = append(controls, control)
	}
	return resolution.WithProjectControls(controls)
}

func npmCreateArguments(network string, hostArguments []string) []string {
	return nodeResolverCreateArguments(network, hostArguments, []string{"--tmpfs", boundaryHelperMount})
}

func nodeResolverCreateArguments(network string, hostArguments, helperMount []string) []string {
	arguments := []string{
		"create", "--pull", "never", "--runtime", gVisorRuntimeName,
		"--network", network,
		"--read-only",
		"--cap-drop", "ALL",
		"--cap-add", "SETUID", "--cap-add", "SETGID", "--cap-add", "SETPCAP",
		"--security-opt", "no-new-privileges",
		"--pids-limit", "64",
		"--memory", "512m",
		"--cpus", "1",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=128m,uid=1000,gid=1000,mode=0700",
	}
	arguments = append(arguments, helperMount...)
	arguments = append(arguments, isolatedContainerEnvironmentArguments()...)
	arguments = append(arguments, hostArguments...)
	arguments = append(arguments, runtimeidentity.NodeImageReference, "/bin/sh", "-ceu", boundaryContainerCommand())
	return arguments
}

// npmNetworkArguments fixes the preflight-resolved registry address set into
// the sandbox. Resolver egress deliberately denies DNS, so registry lookup
// cannot be deferred to the untrusted container network namespace.
func npmNetworkArguments(addresses []netip.Addr) ([]string, error) {
	if err := validateResolverEndpoints(addresses); err != nil {
		return nil, err
	}
	ordered := append([]netip.Addr(nil), addresses...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Less(ordered[j]) })
	arguments := make([]string, 0, len(ordered)*2)
	for _, address := range ordered {
		arguments = append(arguments, "--add-host", "registry.npmjs.org:"+address.String())
	}
	return arguments, nil
}
