package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	// nodeImageReference is the M3-pinned Node 22.23.1 image. Its upstream
	// bundled npm version is part of that exact runtime identity.
	resolverNPMVersion = "10.9.8"
	resolverProjectDir = "/tmp/haa-resolver"
)

// EndpointResolver supplies the trusted, preflight-resolved address set for
// the registry endpoints allowed by one resolver network policy.
type EndpointResolver interface {
	Resolve(context.Context, []string) ([]netip.Addr, error)
}

// NPMResolver is a disposable Docker resolver. It returns only the parsed
// Domain graph; package-lock bytes and npm output never leave this adapter.
type NPMResolver struct {
	runner    CommandRunner
	endpoints EndpointResolver
}

func NewNPMResolver(runner CommandRunner, endpoints EndpointResolver) (*NPMResolver, error) {
	if runner == nil || endpoints == nil {
		return nil, errors.New("npm resolver requires command runner and endpoint resolver")
	}
	if _, ok := runner.(inputCommandRunner); !ok {
		return nil, errors.New("npm resolver requires input stream runner")
	}
	return &NPMResolver{runner: runner, endpoints: endpoints}, nil
}

// NewLinuxNPMResolver composes the production command and trusted DNS
// preflight adapters. It is wired only by the Linux composition root.
func NewLinuxNPMResolver() (*NPMResolver, error) {
	return NewNPMResolver(systemExecutor{}, systemEndpointResolver{})
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
func (r *NPMResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, _ domain.InstallContext) (graph domain.LockedDependencyGraph, resultErr error) {
	if ctx == nil || reference.Source().String() != "npm" {
		return domain.LockedDependencyGraph{}, errors.New("valid npm resolver request is required")
	}
	addresses, err := r.endpoints.Resolve(ctx, []string{"registry.npmjs.org"})
	if err != nil {
		return domain.LockedDependencyGraph{}, errors.New("resolver endpoint preflight failed")
	}
	policy, err := NewResolverNetworkPolicy(ctx, r.runner)
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	network, err := policy.Prepare(ctx, addresses)
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	var cleanupErr error
	containerID := ""
	defer func() {
		if containerID != "" {
			_, removeErr := r.runner.Output(context.Background(), "docker", "rm", "--force", containerID)
			if removeErr != nil {
				cleanupErr = errors.New("resolver container cleanup failed")
			}
		}
		if closeErr := policy.Close(context.Background()); closeErr != nil {
			cleanupErr = errors.Join(cleanupErr, closeErr)
		}
		if cleanupErr != nil {
			graph = domain.LockedDependencyGraph{}
			resultErr = cleanupErr
		}
	}()

	created, err := r.runner.Output(ctx, "docker", "create", "--network", network, "--user", "1000:1000", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", "512m", "--cpus", "1", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=128m,uid=1000,gid=1000,mode=0700", nodeImageReference, "/bin/sh", "-ceu", "sleep infinity")
	if err != nil || !containerIDPattern.MatchString(strings.TrimSpace(string(created))) {
		return domain.LockedDependencyGraph{}, errors.New("create resolver container failed")
	}
	containerID = strings.TrimSpace(string(created))
	if _, err := r.runner.Output(ctx, "docker", "start", containerID); err != nil {
		return domain.LockedDependencyGraph{}, errors.New("start resolver container failed")
	}
	version, err := r.runner.Output(ctx, "docker", "exec", containerID, "npm", "--version")
	if err != nil || strings.TrimSpace(string(version)) != resolverNPMVersion {
		return domain.LockedDependencyGraph{}, errors.New("resolver npm runtime version mismatch")
	}
	input, ok := r.runner.(inputCommandRunner)
	if !ok {
		return domain.LockedDependencyGraph{}, errors.New("resolver input stream unavailable")
	}
	packageName, err := artifactnpm.RequestedPackageName(reference)
	if err != nil {
		return domain.LockedDependencyGraph{}, errors.New("resolver package reference is invalid")
	}
	manifest := []byte("{\"name\":\"haa-resolver\",\"version\":\"1.0.0\",\"private\":true,\"dependencies\":{\"" + packageName + "\":\"" + reference.Locator() + "\"}}")
	if err := input.RunInput(ctx, bytes.NewReader(manifest), "docker", "exec", "-i", containerID, "/bin/sh", "-ceu", "umask 077; mkdir -p "+resolverProjectDir+"; cat > "+resolverProjectDir+"/package.json"); err != nil {
		return domain.LockedDependencyGraph{}, errors.New("write resolver manifest failed")
	}
	command := "cd " + resolverProjectDir + "; HOME=/tmp npm_config_cache=/tmp/cache npm install --package-lock-only --ignore-scripts --no-audit --no-fund --registry=https://registry.npmjs.org/ --userconfig=/tmp/haa-user.npmrc --globalconfig=/tmp/haa-global.npmrc"
	if _, err := r.runner.Output(ctx, "docker", "exec", containerID, "/bin/sh", "-ceu", command); err != nil {
		return domain.LockedDependencyGraph{}, errors.New("run locked npm resolution failed")
	}
	lock, err := r.runner.Output(ctx, "docker", "exec", containerID, "cat", resolverProjectDir+"/package-lock.json")
	if err != nil {
		return domain.LockedDependencyGraph{}, errors.New("read resolver lockfile failed")
	}
	graph, parseErr := artifactnpm.ParsePackageLockV3(reference, lock)
	if parseErr != nil {
		return domain.LockedDependencyGraph{}, fmt.Errorf("parse resolver lockfile: %w", parseErr)
	}
	return graph, nil
}
