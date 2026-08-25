package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/netip"
	"runtime"
	"sort"
	"strings"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	pypiIndexEndpoint        = "pypi.org"
	pypiDistributionEndpoint = "files.pythonhosted.org"
	pypiResolverProjectDir   = "/tmp/haa-pypi-resolver"
	pypiResolverTimeout      = 2 * time.Minute
)

var pypiResolverEndpoints = []string{pypiIndexEndpoint, pypiDistributionEndpoint}

// NamedEndpointResolver supplies trusted Host-resolved IPv4 addresses keyed
// by the only public endpoints an isolated PyPI resolver may contact.
type NamedEndpointResolver interface {
	Resolve(context.Context, []string) (map[string][]netip.Addr, error)
}

// PyPIResolver runs pip only in the locked Python gVisor runtime and returns
// a parser-normalized generic graph after Simple API cross-checking. Raw pip
// reports and index responses never leave this adapter boundary.
type PyPIResolver struct {
	runner    CommandRunner
	endpoints NamedEndpointResolver
	observer  TraceObserver
	probe     func(context.Context) (PythonCapability, error)
	close     func() error
	policy    ResolverPolicyService
}

// NewPyPIResolver constructs the narrow M5 resolver boundary. Every injected
// collaborator is required because missing network, runtime or observation
// controls must fail closed before a graph can be emitted.
func NewPyPIResolver(runner CommandRunner, endpoints NamedEndpointResolver, observer TraceObserver, probe func(context.Context) (PythonCapability, error), policy ResolverPolicyService) (*PyPIResolver, error) {
	if runner == nil || endpoints == nil || observer == nil || probe == nil || policy == nil {
		return nil, errors.New("PyPI resolver requires runner, endpoint resolver, observer, runtime probe and network policy service")
	}
	return &PyPIResolver{runner: runner, endpoints: endpoints, observer: observer, probe: probe, policy: policy}, nil
}

// NewLinuxPyPIResolverWithExecutorAndPolicy constructs the production
// resolver with the ordinary-to-privileged typed network policy port.
func NewLinuxPyPIResolverWithExecutorAndPolicy(executor TrustedExecutor, observer TraceObserver, policy ResolverPolicyService) (*PyPIResolver, error) {
	if policy == nil {
		return nil, errors.New("PyPI resolver requires network policy service")
	}
	resolver, err := newLinuxPyPIResolver(executor, observer, policy)
	if err != nil {
		return nil, err
	}
	resolver.policy = policy
	return resolver, nil
}

func newLinuxPyPIResolver(executor TrustedExecutor, observer TraceObserver, policy ResolverPolicyService) (*PyPIResolver, error) {
	if observer == nil {
		return nil, errors.New("process-scoped observer is required")
	}
	capabilityProbe := func(ctx context.Context) (PythonCapability, error) {
		return probePython(ctx, runtime.GOOS, runtime.GOARCH, executor)
	}
	resolver, err := NewPyPIResolver(executor, systemNamedEndpointResolver{}, observer, capabilityProbe, policy)
	if err != nil {
		return nil, err
	}
	return resolver, nil
}

// Close releases the HAA-only observer listener created by the Linux factory.
// It is a no-op for test or externally owned observers.
func (r *PyPIResolver) Close() error {
	if r == nil || r.close == nil {
		return nil
	}
	return r.close()
}

// ResolveDependencies performs the complete default-deny resolver lifecycle.
// Any failure to create, verify, attribute, collect or clean up that lifecycle
// clears the result and returns an error.
func (r *PyPIResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, _ domain.InstallContext) (resolution domain.DependencyResolution, resultErr error) {
	if r == nil || r.runner == nil || r.endpoints == nil || r.observer == nil || r.probe == nil || ctx == nil || reference.Source().String() != "pypi" {
		return domain.DependencyResolution{}, errors.New("valid PyPI resolver request is required")
	}
	capability, err := r.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedPythonRuntime() {
		return domain.DependencyResolution{}, errors.New("PyPI resolver runtime is unavailable")
	}
	addressesByName, err := r.endpoints.Resolve(ctx, append([]string(nil), pypiResolverEndpoints...))
	if err != nil {
		return domain.DependencyResolution{}, errors.New("PyPI resolver endpoint preflight failed")
	}
	addresses, hostArguments, err := pypiNetworkArguments(addressesByName)
	if err != nil {
		return domain.DependencyResolution{}, errors.New("PyPI resolver endpoint preflight is unsafe")
	}
	policy, err := NewResolverNetworkPolicy(r.runner, r.policy)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	network, err := policy.Prepare(ctx, addresses)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	containerID := ""
	var trace TraceReader
	defer func() {
		var cleanupErr error
		if containerID != "" {
			cleanupCtx, cleanupCancel := resolverCleanupContext()
			_, removeErr := r.runner.Output(cleanupCtx, "docker", "rm", "--force", containerID)
			cleanupCancel()
			if removeErr != nil {
				cleanupErr = errors.New("PyPI resolver container cleanup failed")
			}
			if trace != nil {
				collectCtx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
				_, limitation := collectTrace(collectCtx, trace)
				cancel()
				if limitation != "" {
					cleanupErr = errors.Join(cleanupErr, errors.New("PyPI resolver observation is incomplete"))
				}
			}
		}
		cleanupCtx, cleanupCancel := resolverCleanupContext()
		closeErr := policy.Close(cleanupCtx)
		cleanupCancel()
		if closeErr != nil {
			cleanupErr = errors.Join(cleanupErr, closeErr)
		}
		if cleanupErr != nil {
			resolution = domain.DependencyResolution{}
			resultErr = cleanupErr
		}
	}()

	created, err := r.runner.Output(ctx, "docker", pypiCreateArguments(network, hostArguments)...)
	if err != nil || !containerIDPattern.MatchString(strings.TrimSpace(string(created))) {
		return domain.DependencyResolution{}, errors.New("create PyPI resolver container failed")
	}
	containerID = strings.TrimSpace(string(created))
	trace, err = startTrace(ctx, r.observer, containerID, "pypi-wheel")
	if err != nil {
		return domain.DependencyResolution{}, errors.New("start PyPI resolver observer failed")
	}
	if _, err := r.runner.Output(ctx, "docker", "start", containerID); err != nil {
		return domain.DependencyResolution{}, errors.New("start PyPI resolver container failed")
	}
	if err := verifyPyPIResolverRuntime(ctx, r.runner, containerID, capability.Runtime); err != nil {
		return domain.DependencyResolution{}, err
	}
	request, err := pypiRequirement(reference)
	if err != nil {
		return domain.DependencyResolution{}, errors.New("PyPI resolver reference is invalid")
	}
	resolveCtx, cancel := context.WithTimeout(ctx, pypiResolverTimeout)
	defer cancel()
	if _, err := r.runner.Output(resolveCtx, "docker", "exec", containerID, "python", "-I", "-m", "pip", "install", "--dry-run", "--report", pypiResolverProjectDir+"/report.json", "--disable-pip-version-check", "--no-input", "--no-cache-dir", "--isolated", "--index-url", "https://pypi.org/simple/", request); err != nil {
		return domain.DependencyResolution{}, errors.New("run locked pip resolution failed")
	}
	reportBytes, err := r.runner.Output(resolveCtx, "docker", "exec", containerID, "python", "-I", "-c", boundedReadScript, pypiResolverProjectDir+"/report.json", "4194304")
	if err != nil {
		return domain.DependencyResolution{}, errors.New("read pip installation report failed")
	}
	report, err := artifactpypi.ParseInstallationReport(reference, reportBytes, capability.Runtime.PipVersion, capability.Runtime.PythonVersion)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	pages := make([]artifactpypi.SimpleProject, 0, len(report.Candidates()))
	for _, candidate := range report.Candidates() {
		body, err := r.runner.Output(resolveCtx, "docker", "exec", containerID, "python", "-I", "-c", simpleJSONFetchScript, candidate.Project())
		if err != nil {
			return domain.DependencyResolution{}, errors.New("fetch PyPI Simple metadata failed")
		}
		page, err := artifactpypi.ParseSimpleProject(candidate.Project(), body)
		if err != nil {
			return domain.DependencyResolution{}, err
		}
		pages = append(pages, page)
	}
	candidates, err := artifactpypi.CrossCheckReport(report, pages)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	graph, err := artifactpypi.BuildLockedGraph(reference, candidates)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	sum := sha256.Sum256(reportBytes)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	runtimeIdentity := "python:" + capability.Runtime.PythonVersion + ";pip:" + capability.Runtime.PipVersion + ";target:" + capability.Runtime.InterpreterTag + "/" + capability.Runtime.ABITag + "/" + capability.Runtime.PlatformTag
	return domain.NewDependencyResolution(graph, runtimeIdentity, digest)
}

func pypiCreateArguments(network string, hostArguments []string) []string {
	arguments := []string{
		"create", "--pull", "never", "--runtime", gVisorRuntimeName,
		"--network", network,
		"--user", "1000:1000",
		"--read-only",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--pids-limit", "64",
		"--memory", "512m",
		"--cpus", "1",
		"--ulimit", "cpu=60:60",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=128m,uid=1000,gid=1000,mode=0700",
	}
	arguments = append(arguments, hostArguments...)
	// Prepare the bounded report directory on the container tmpfs before pip runs.
	// The command is fixed infrastructure wiring; no request-controlled input is
	// interpolated into it.
	arguments = append(arguments, pythonImageReference, "sh", "-ceu", "umask 077; mkdir -p "+pypiResolverProjectDir+"; exec sleep infinity")
	return arguments
}

func verifyPyPIResolverRuntime(ctx context.Context, runner CommandRunner, containerID string, runtime PythonRuntime) error {
	python, err := runner.Output(ctx, "docker", "exec", containerID, "python", "-I", "-c", "import sys; print('.'.join(map(str, sys.version_info[:3])))")
	if err != nil || strings.TrimSpace(string(python)) != runtime.PythonVersion {
		return errors.New("PyPI resolver Python runtime version mismatch")
	}
	pip, err := runner.Output(ctx, "docker", "exec", containerID, "python", "-I", "-m", "pip", "--version")
	if err != nil || !strings.HasPrefix(strings.TrimSpace(string(pip)), "pip "+runtime.PipVersion+" ") {
		return errors.New("PyPI resolver pip runtime version mismatch")
	}
	tags, err := runner.Output(ctx, "docker", "exec", containerID, "python", "-I", "-m", "pip", "debug", "--verbose")
	if err != nil || !containsExactLine(string(tags), runtime.InterpreterTag+"-"+runtime.ABITag+"-"+runtime.PlatformTag) {
		return errors.New("PyPI resolver target tags are unavailable")
	}
	return nil
}

func containsExactLine(output, expected string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == expected {
			return true
		}
	}
	return false
}

func pypiRequirement(reference domain.ArtifactReference) (string, error) {
	project, err := artifactpypi.RequestedProject(reference)
	if err != nil {
		return "", err
	}
	version, present, err := artifactpypi.RequestedVersion(reference)
	if err != nil {
		return "", err
	}
	if present {
		return project + "==" + version, nil
	}
	return project, nil
}

func pypiNetworkArguments(addressesByName map[string][]netip.Addr) ([]netip.Addr, []string, error) {
	if len(addressesByName) != len(pypiResolverEndpoints) {
		return nil, nil, errors.New("unexpected endpoint set")
	}
	all := make([]netip.Addr, 0)
	seen := make(map[netip.Addr]bool)
	arguments := make([]string, 0)
	for _, name := range pypiResolverEndpoints {
		addresses := append([]netip.Addr(nil), addressesByName[name]...)
		if len(addresses) == 0 {
			return nil, nil, errors.New("endpoint address is missing")
		}
		sort.Slice(addresses, func(i, j int) bool { return addresses[i].Less(addresses[j]) })
		seenForName := make(map[netip.Addr]bool, len(addresses))
		for _, address := range addresses {
			if !address.IsValid() || !address.Is4() || address.IsPrivate() || address.IsLoopback() || address.IsMulticast() || address.IsUnspecified() {
				return nil, nil, errors.New("endpoint address is unsafe")
			}
			if seenForName[address] {
				return nil, nil, errors.New("endpoint address is ambiguous")
			}
			seenForName[address] = true
			arguments = append(arguments, "--add-host", name+":"+address.String())
			if !seen[address] {
				seen[address] = true
				all = append(all, address)
			}
		}
	}
	if err := validateResolverEndpoints(all); err != nil {
		return nil, nil, err
	}
	return all, arguments, nil
}

type systemNamedEndpointResolver struct{}

func (systemNamedEndpointResolver) Resolve(ctx context.Context, names []string) (map[string][]netip.Addr, error) {
	if ctx == nil || len(names) != len(pypiResolverEndpoints) {
		return nil, errors.New("endpoint names are required")
	}
	resolved := make(map[string][]netip.Addr, len(names))
	for _, name := range names {
		if name != pypiIndexEndpoint && name != pypiDistributionEndpoint {
			return nil, errors.New("endpoint is not trusted")
		}
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", name)
		if err != nil {
			return nil, errors.New("endpoint lookup failed")
		}
		resolved[name] = append([]netip.Addr(nil), addresses...)
	}
	return resolved, nil
}

const boundedReadScript = "import os,sys\np=sys.argv[1]\nlimit=int(sys.argv[2])\nsize=os.stat(p).st_size\nif size < 1 or size > limit: raise SystemExit(1)\nwith open(p, 'rb') as f: data=f.read(limit+1)\nif len(data) != size or len(data) > limit: raise SystemExit(1)\nsys.stdout.buffer.write(data)\n"

const simpleJSONFetchScript = "import sys,urllib.error,urllib.request\nclass NoRedirect(urllib.request.HTTPRedirectHandler):\n def redirect_request(self, req, fp, code, msg, headers, newurl): return None\nproject=sys.argv[1]\nurl='https://pypi.org/simple/'+project+'/'\nrequest=urllib.request.Request(url, headers={'Accept':'application/vnd.pypi.simple.v1+json'})\nresponse=urllib.request.build_opener(NoRedirect).open(request, timeout=15)\nif response.status != 200 or response.geturl() != url: raise SystemExit(1)\nif response.headers.get_content_type().lower() != 'application/vnd.pypi.simple.v1+json': raise SystemExit(1)\nlength=response.headers.get('Content-Length')\nif length is not None and (not length.isdigit() or int(length) < 1 or int(length) > 4194304): raise SystemExit(1)\nbody=response.read(4194305)\nif len(body) < 1 or len(body) > 4194304: raise SystemExit(1)\nsys.stdout.buffer.write(body)\n"
