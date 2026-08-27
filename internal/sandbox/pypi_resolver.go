package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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

var pypiResolverEndpoints = []string{"pypi.org", "files.pythonhosted.org"}

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
	profile   artifactpypi.SourceProfile
}

// NewPyPIResolver constructs the narrow M5 resolver boundary. Every injected
// collaborator is required because missing network, runtime or observation
// controls must fail closed before a graph can be emitted.
func NewPyPIResolver(runner CommandRunner, endpoints NamedEndpointResolver, observer TraceObserver, probe func(context.Context) (PythonCapability, error), policy ResolverPolicyService) (*PyPIResolver, error) {
	return newPythonResolver(runner, endpoints, observer, probe, policy, artifactpypi.PublicPyPIProfile())
}

func newPythonResolver(runner CommandRunner, endpoints NamedEndpointResolver, observer TraceObserver, probe func(context.Context) (PythonCapability, error), policy ResolverPolicyService, profile artifactpypi.SourceProfile) (*PyPIResolver, error) {
	if runner == nil || endpoints == nil || observer == nil || probe == nil || policy == nil {
		return nil, errors.New("PyPI resolver requires runner, endpoint resolver, observer, runtime probe and network policy service")
	}
	return &PyPIResolver{runner: runner, endpoints: endpoints, observer: observer, probe: probe, policy: policy, profile: profile}, nil
}

// NewPyTorchResolver constructs a resolver for one immutable official
// PyTorch source profile. It shares the PyPI runtime but not source identity.
func NewPyTorchResolver(runner CommandRunner, endpoints NamedEndpointResolver, observer TraceObserver, probe func(context.Context) (PythonCapability, error), policy ResolverPolicyService, profile artifactpypi.SourceProfile) (*PyPIResolver, error) {
	if !artifactpypi.IsPyTorchSource(profile.Source()) {
		return nil, errors.New("PyTorch resolver requires an official source profile")
	}
	return newPythonResolver(runner, endpoints, observer, probe, policy, profile)
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

// NewLinuxPyTorchResolverWithExecutorAndPolicy binds one named PyTorch source
// profile to the same trusted Linux resolver infrastructure.
func NewLinuxPyTorchResolverWithExecutorAndPolicy(executor TrustedExecutor, observer TraceObserver, policy ResolverPolicyService, profile artifactpypi.SourceProfile) (*PyPIResolver, error) {
	if policy == nil || !artifactpypi.IsPyTorchSource(profile.Source()) {
		return nil, errors.New("PyTorch resolver requires network policy and official profile")
	}
	if observer == nil {
		return nil, errors.New("process-scoped observer is required")
	}
	capabilityProbe := func(ctx context.Context) (PythonCapability, error) {
		return probePython(ctx, runtime.GOOS, runtime.GOARCH, executor)
	}
	return newPythonResolver(executor, systemNamedEndpointResolver{}, observer, capabilityProbe, policy, profile)
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
	if r == nil || r.runner == nil || r.endpoints == nil || r.observer == nil || r.probe == nil || ctx == nil || reference.Source() != r.profile.Source() {
		return domain.DependencyResolution{}, errors.New("valid PyPI resolver request is required")
	}
	capability, err := r.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedPythonRuntime() {
		return domain.DependencyResolution{}, errors.New("PyPI resolver runtime is unavailable")
	}
	endpointNames := artifactpypiProfileEndpoints(r.profile)
	addressesByName, err := r.endpoints.Resolve(ctx, endpointNames)
	if err != nil {
		return domain.DependencyResolution{}, errors.New("PyPI resolver endpoint preflight failed")
	}
	addresses, hostArguments, err := resolverNetworkArguments(r.profile, addressesByName)
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
	resolveTimeout := pypiResolverTimeout
	if resourcePolicy := artifactpypi.ResourcePolicyFromContext(ctx); resourcePolicy.Duration() > defaultPyPIResolverDuration {
		resolveTimeout = resourcePolicy.Duration()
	}
	resolveCtx, cancel := context.WithTimeout(ctx, resolveTimeout)
	defer cancel()
	if artifactpypi.IsPyTorchSource(r.profile.Source()) {
		candidates, reportBytes, resolveErr := r.resolvePyTorchGraph(resolveCtx, containerID, capability.Runtime, reference)
		if resolveErr != nil {
			return domain.DependencyResolution{}, resolveErr
		}
		graph, graphErr := artifactpypi.BuildLockedGraph(reference, candidates)
		if graphErr != nil {
			return domain.DependencyResolution{}, graphErr
		}
		sum := sha256.Sum256(reportBytes)
		digest, digestErr := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		if digestErr != nil {
			return domain.DependencyResolution{}, digestErr
		}
		runtimeIdentity := "python:" + capability.Runtime.PythonVersion + ";pip:" + capability.Runtime.PipVersion + ";target:" + capability.Runtime.InterpreterTag + "/" + capability.Runtime.ABITag + "/" + capability.Runtime.PlatformTag + ";source:" + r.profile.Name()
		return domain.NewDependencyResolution(graph, runtimeIdentity, digest)
	}
	pipArguments := pypiResolveArguments(r.profile, request)
	if _, err := r.runner.Output(resolveCtx, "docker", append([]string{"exec", containerID, "python", "-I", "-m", "pip"}, pipArguments...)...); err != nil {
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
		fetchScript := simpleJSONFetchScript
		fetchArguments := []string{candidate.Project()}
		profile := artifactpypi.PublicPyPIProfile()
		if candidate.Source() != artifactpypi.PublicPyPIProfile().Source() {
			fetchScript = pytorchHTMLFetchScript
			fetchArguments = []string{r.profile.IndexURL(), candidate.Project()}
			profile = r.profile
		}
		body, err := r.runner.Output(resolveCtx, "docker", append([]string{"exec", containerID, "python", "-I", "-c", fetchScript}, fetchArguments...)...)
		if err != nil {
			return domain.DependencyResolution{}, errors.New("fetch PyPI Simple metadata failed")
		}
		page, err := artifactpypi.ParseSimpleProjectForProfile(candidate.Project(), body, profile)
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
	runtimeIdentity := "python:" + capability.Runtime.PythonVersion + ";pip:" + capability.Runtime.PipVersion + ";target:" + capability.Runtime.InterpreterTag + "/" + capability.Runtime.ABITag + "/" + capability.Runtime.PlatformTag + ";source:" + r.profile.Name()
	return domain.NewDependencyResolution(graph, runtimeIdentity, digest)
}

const defaultPyPIResolverDuration = 5 * time.Minute

func (r *PyPIResolver) resolvePyTorchGraph(ctx context.Context, containerID string, runtime PythonRuntime, reference domain.ArtifactReference) ([]artifactpypi.Candidate, []byte, error) {
	request, err := pypiRequirement(reference)
	if err != nil {
		return nil, nil, errors.New("PyTorch resolver reference is invalid")
	}
	type pending struct {
		request string
		profile artifactpypi.SourceProfile
		primary bool
	}
	queue := []pending{{request: request, profile: r.profile, primary: true}}
	candidates := make(map[string]artifactpypi.Candidate)
	requests := make(map[string]string)
	reports := make([]byte, 0)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		project, err := artifactpypi.DependencyProject(current.request)
		if err != nil {
			return nil, nil, errors.New("PyTorch dependency requirement is invalid")
		}
		if existing, seen := requests[project]; seen {
			if existing != current.request {
				return nil, nil, errors.New("PyTorch dependency requirements are ambiguous")
			}
			continue
		}
		requests[project] = current.request
		candidateReference, err := artifactpypi.ParseReferenceForSource(project, current.profile.Source())
		if err != nil {
			return nil, nil, err
		}
		candidate, report, err := r.resolvePyTorchCandidate(ctx, containerID, runtime, candidateReference, current.profile, current.request)
		if err != nil {
			return nil, nil, err
		}
		candidates[project] = candidate.WithPrimary(current.primary)
		reports = append(reports, report...)
		for _, requirement := range candidate.DependencyRequirements() {
			dependency, err := artifactpypi.DependencyProject(requirement)
			if err != nil {
				return nil, nil, err
			}
			profile := artifactpypi.PublicPyPIProfile()
			if artifactpypi.IsPyTorchOwnedProject(dependency) {
				profile = r.profile
			}
			queue = append(queue, pending{request: requirement, profile: profile})
		}
	}
	result := make([]artifactpypi.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, candidate)
	}
	return result, reports, nil
}

func (r *PyPIResolver) resolvePyTorchCandidate(ctx context.Context, containerID string, runtime PythonRuntime, reference domain.ArtifactReference, profile artifactpypi.SourceProfile, requirement string) (artifactpypi.Candidate, []byte, error) {
	arguments := append([]string{"install", "--dry-run", "--report", pypiResolverProjectDir + "/report.json", "--disable-pip-version-check", "--no-input", "--no-cache-dir", "--isolated", "--no-deps", "--index-url", profile.IndexURL()}, requirement)
	if _, err := r.runner.Output(ctx, "docker", append([]string{"exec", containerID, "python", "-I", "-m", "pip"}, arguments...)...); err != nil {
		return artifactpypi.Candidate{}, nil, errors.New("run source-pinned pip resolution failed")
	}
	reportBytes, err := r.runner.Output(ctx, "docker", "exec", containerID, "python", "-I", "-c", boundedReadScript, pypiResolverProjectDir+"/report.json", "4194304")
	if err != nil {
		return artifactpypi.Candidate{}, nil, errors.New("read source-pinned pip report failed")
	}
	report, err := artifactpypi.ParseInstallationReportForProfile(reference, reportBytes, runtime.PipVersion, runtime.PythonVersion, profile)
	if err != nil {
		return artifactpypi.Candidate{}, nil, fmt.Errorf("source-pinned pip report is invalid: %w", err)
	}
	if len(report.Candidates()) != 1 {
		return artifactpypi.Candidate{}, nil, errors.New("source-pinned pip report candidate count is invalid")
	}
	candidate := report.Candidates()[0]
	fetchScript, fetchArguments := simpleJSONFetchScript, []string{candidate.Project()}
	if artifactpypi.IsPyTorchSource(profile.Source()) {
		fetchScript, fetchArguments = pytorchHTMLFetchScript, []string{profile.IndexURL(), candidate.Project()}
	}
	body, err := r.runner.Output(ctx, "docker", append([]string{"exec", containerID, "python", "-I", "-c", fetchScript}, fetchArguments...)...)
	if err != nil {
		return artifactpypi.Candidate{}, nil, errors.New("fetch source-pinned Simple metadata failed")
	}
	page, err := artifactpypi.ParseSimpleProjectForProfile(candidate.Project(), body, profile)
	if err != nil {
		return artifactpypi.Candidate{}, nil, err
	}
	if _, err := artifactpypi.CrossCheckReport(report, []artifactpypi.SimpleProject{page}); err != nil {
		return artifactpypi.Candidate{}, nil, err
	}
	return candidate, reportBytes, nil
}

// pypiResolveArguments binds a PyTorch-owned root to exactly one official
// profile. Dependencies are deliberately not delegated to pip's multi-index
// selector: the resolver adds them through the canonical ownership table.
func pypiResolveArguments(profile artifactpypi.SourceProfile, request string) []string {
	arguments := []string{"install", "--dry-run", "--report", pypiResolverProjectDir + "/report.json", "--disable-pip-version-check", "--no-input", "--no-cache-dir", "--isolated", "--index-url", profile.IndexURL()}
	if artifactpypi.IsPyTorchSource(profile.Source()) {
		arguments = append(arguments, "--no-deps")
	}
	return append(arguments, request)
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
	return resolverNetworkArguments(artifactpypi.PublicPyPIProfile(), addressesByName)
}

func artifactpypiProfileEndpoints(profile artifactpypi.SourceProfile) []string {
	if profile.Source() == (artifactpypi.SourceProfile{}).Source() {
		return nil
	}
	if profile.Source() == artifactpypi.PublicPyPIProfile().Source() {
		return append([]string(nil), pypiResolverEndpoints...)
	}
	// The profile owns the endpoint set; PyTorch additionally reaches the
	// canonical PyPI endpoints for ordinary transitive dependencies.
	names := []string{profile.IndexHost()}
	names = append(names, profile.DistributionHosts()...)
	if artifactpypi.IsPyTorchSource(profile.Source()) {
		names = append(names, artifactpypi.PublicPyPIProfile().IndexHost())
		names = append(names, artifactpypi.PublicPyPIProfile().DistributionHosts()...)
	}
	sort.Strings(names)
	unique := names[:0]
	for _, name := range names {
		if len(unique) == 0 || unique[len(unique)-1] != name {
			unique = append(unique, name)
		}
	}
	return unique
}

func resolverNetworkArguments(profile artifactpypi.SourceProfile, addressesByName map[string][]netip.Addr) ([]netip.Addr, []string, error) {
	endpoints := artifactpypiProfileEndpoints(profile)
	if len(addressesByName) != len(endpoints) {
		return nil, nil, errors.New("unexpected endpoint set")
	}
	all := make([]netip.Addr, 0)
	seen := make(map[netip.Addr]bool)
	arguments := make([]string, 0)
	for _, name := range endpoints {
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
	if ctx == nil || len(names) == 0 {
		return nil, errors.New("endpoint names are required")
	}
	resolved := make(map[string][]netip.Addr, len(names))
	for _, name := range names {
		trusted := false
		for _, profile := range artifactpypi.AllSourceProfiles() {
			if profile.IndexHost() == name {
				trusted = true
			}
			for _, host := range profile.DistributionHosts() {
				if host == name {
					trusted = true
				}
			}
		}
		if !trusted {
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

const pytorchHTMLFetchScript = "import sys,urllib.request\nclass NoRedirect(urllib.request.HTTPRedirectHandler):\n def redirect_request(self, req, fp, code, msg, headers, newurl): return None\nbase=sys.argv[1]\nproject=sys.argv[2]\nurl=base+project+'/'\nrequest=urllib.request.Request(url, headers={'Accept':'text/html'})\nresponse=urllib.request.build_opener(NoRedirect).open(request, timeout=15)\nif response.status != 200 or response.geturl() != url: raise SystemExit(1)\nif response.headers.get_content_type().lower() != 'text/html': raise SystemExit(1)\nlength=response.headers.get('Content-Length')\nif length is not None and (not length.isdigit() or int(length) < 1 or int(length) > 4194304): raise SystemExit(1)\nbody=response.read(4194305)\nif len(body) < 1 or len(body) > 4194304: raise SystemExit(1)\nsys.stdout.buffer.write(body)\n"
