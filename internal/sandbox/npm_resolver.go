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

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	// nodeImageReference is the M3-pinned Node 22.23.1 image. Its upstream
	// bundled npm version is part of that exact runtime identity.
	resolverNPMVersion      = "10.9.8"
	resolverProjectDir      = "/tmp/haa-resolver"
	resolverRuntimeIdentity = "node:22.23.1-slim@sha256:6c74791e557ce11fc957704f6d4fe134a7bc8d6f5ca4403205b2966bd488f6b3;npm:10.9.8"
)

// EndpointResolver supplies the trusted, preflight-resolved address set for
// the registry endpoints allowed by one resolver network policy.
type EndpointResolver interface {
	Resolve(context.Context, []string) ([]netip.Addr, error)
}

// NPMResolver is a disposable Docker resolver. It returns only the parsed
// Domain resolution; package-lock bytes and npm output never leave this adapter.
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
	return &NPMResolver{runner: runner, endpoints: endpoints, policy: policy}, nil
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
func (r *NPMResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, _ domain.InstallContext) (resolution domain.DependencyResolution, resultErr error) {
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
				collectCtx, collectCancel := context.WithTimeout(context.Background(), cleanupTimeout)
				_, limitation := collectTrace(collectCtx, trace)
				collectCancel()
				if limitation != "" {
					cleanupErr = errors.Join(cleanupErr, errors.New("resolver observation is incomplete"))
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
			resultErr = cleanupErr
		}
	}()

	createArguments := []string{"create", "--runtime", gVisorRuntimeName, "--network", network, "--user", "1000:1000", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", "512m", "--cpus", "1", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=128m,uid=1000,gid=1000,mode=0700"}
	createArguments = append(createArguments, hostArguments...)
	createArguments = append(createArguments, nodeImageReference, "/bin/sh", "-ceu", "sleep infinity")
	created, err := r.runner.Output(ctx, "docker", createArguments...)
	if err != nil || !containerIDPattern.MatchString(strings.TrimSpace(string(created))) {
		return domain.DependencyResolution{}, errors.New("create resolver container failed")
	}
	containerID = strings.TrimSpace(string(created))
	if r.observer == nil {
		return domain.DependencyResolution{}, errors.New("resolver observation is unavailable")
	}
	trace, err = r.observer.Start(ctx, containerID)
	if err != nil {
		return domain.DependencyResolution{}, errors.New("start resolver observer failed")
	}
	if _, err := r.runner.Output(ctx, "docker", "start", containerID); err != nil {
		return domain.DependencyResolution{}, errors.New("start resolver container failed")
	}
	version, err := r.runner.Output(ctx, "docker", "exec", containerID, "npm", "--version")
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
	if err := input.RunInput(ctx, bytes.NewReader(manifest), "docker", "exec", "-i", containerID, "/bin/sh", "-ceu", "umask 077; mkdir -p "+resolverProjectDir+"; cat > "+resolverProjectDir+"/package.json"); err != nil {
		return domain.DependencyResolution{}, errors.New("write resolver manifest failed")
	}
	command := "cd " + resolverProjectDir + "; HOME=/tmp npm_config_cache=/tmp/cache npm install --package-lock-only --ignore-scripts --no-audit --no-fund --registry=https://registry.npmjs.org/ --userconfig=/tmp/haa-user.npmrc --globalconfig=/tmp/haa-global.npmrc"
	if _, err := r.runner.Output(ctx, "docker", "exec", containerID, "/bin/sh", "-ceu", command); err != nil {
		return domain.DependencyResolution{}, errors.New("run locked npm resolution failed")
	}
	lock, err := r.runner.Output(ctx, "docker", "exec", containerID, "cat", resolverProjectDir+"/package-lock.json")
	if err != nil {
		return domain.DependencyResolution{}, errors.New("read resolver lockfile failed")
	}
	graph, parseErr := artifactnpm.ParsePackageLockV3(reference, lock)
	if parseErr != nil {
		return domain.DependencyResolution{}, fmt.Errorf("parse resolver lockfile: %w", parseErr)
	}
	sum := sha256.Sum256(lock)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return domain.NewDependencyResolution(graph, resolverRuntimeIdentity, digest)
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
