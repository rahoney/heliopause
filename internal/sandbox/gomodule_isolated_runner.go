package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
)

const (
	goResolverProfile      = "go-module-resolver"
	goResolverGuestProject = "/tmp/haa-go-project"
	goResolverGuestCache   = "/tmp/haa-go-modules"
	goResolverBinary       = "/usr/local/go/bin/go"
	goResolverOutputLimit  = 4 << 20
)

// IsolatedGoModuleRunner executes only resolver grammar, never project source,
// in a disposable pinned gVisor runtime. It cannot be used as a build runner.
type IsolatedGoModuleRunner struct {
	runner    CommandRunner
	endpoints NamedEndpointResolver
	observer  TraceObserver
	probe     func(context.Context) (GoCapability, error)
	policy    ResolverPolicyService
}

// Only this registered runner can produce this failure type. Upstream resolver
// wrappers may retain its bounded diagnostics without trusting arbitrary
// errors from a Host/stub runner or exposing command stderr.
type isolatedGoResolverFailure struct {
	cause      error
	phase      string
	diagnostic TraceDiagnostic
}

func (e *isolatedGoResolverFailure) Error() string {
	if e.diagnostic.Reason != "" {
		return "isolated go resolver failed: phase=" + e.phase + " " + e.diagnostic.String()
	}
	return "isolated go resolver failed: phase=" + e.phase
}
func (e *isolatedGoResolverFailure) Unwrap() error { return e.cause }

func NewLinuxGoModuleRunner(executor TrustedExecutor, observer TraceObserver, policy ResolverPolicyService) (*IsolatedGoModuleRunner, error) {
	if executor == nil {
		return nil, errors.New("go resolver requires trusted executor")
	}
	return newIsolatedGoModuleRunner(executor, goNamedEndpointResolver{}, observer, func(ctx context.Context) (GoCapability, error) { return ProbeGo(ctx, executor) }, policy)
}

func newIsolatedGoModuleRunner(runner CommandRunner, endpoints NamedEndpointResolver, observer TraceObserver, probe func(context.Context) (GoCapability, error), policy ResolverPolicyService) (*IsolatedGoModuleRunner, error) {
	if runner == nil || endpoints == nil || observer == nil || probe == nil || policy == nil {
		return nil, errors.New("go resolver requires runtime, network and observation boundaries")
	}
	if _, ok := runner.(inputCommandRunner); !ok {
		return nil, errors.New("go resolver requires bounded control input")
	}
	if _, ok := runner.(interface {
		RunOutput(context.Context, io.Writer, string, ...string) error
	}); !ok {
		return nil, errors.New("go resolver requires bounded output")
	}
	return &IsolatedGoModuleRunner{admissionAwareRunner(runner), endpoints, observer, probe, policy}, nil
}

func (r *IsolatedGoModuleRunner) RunGo(ctx context.Context, workspace string, environment []string, arguments ...string) (output []byte, resultErr error) {
	if r == nil || r.runner == nil || ctx == nil || !validGoResolverCommand(arguments) {
		return nil, errors.New("go resolver command is unsupported")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(environment) == 0 {
		return nil, errors.New("go resolver environment is missing")
	}
	cache := strings.TrimPrefix(environment[len(environment)-1], "GOMODCACHE=")
	if artifactgomodule.ValidateResolverEnvironmentForCache(environment, cache) != nil {
		return nil, errors.New("go resolver environment is not canonical")
	}
	if !filepath.IsAbs(workspace) || filepath.Clean(workspace) != workspace || workspace == "/" {
		return nil, errors.New("go resolver workspace is invalid")
	}
	beforeControls, err := readGoFrozenControls(workspace)
	if err != nil {
		return nil, err
	}
	beforeMod, beforeSum, err := goFrozenControlBodies(beforeControls)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(workspace)
	if err != nil || len(entries) != 2 {
		return nil, errors.New("go resolver accepts private controls only")
	}
	capability, err := r.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedGoRuntime() {
		return nil, errors.New("go resolver pinned runtime unavailable")
	}
	addresses, err := r.endpoints.Resolve(ctx, []string{"proxy.golang.org", "sum.golang.org"})
	if err != nil {
		return nil, errors.New("go resolver endpoint lookup failed")
	}
	allowed, hosts, err := goResolverNetworkArguments(addresses)
	if err != nil {
		return nil, err
	}
	policy, err := NewResolverNetworkPolicy(r.runner, r.policy)
	if err != nil {
		return nil, err
	}
	network, err := policy.Prepare(ctx, allowed)
	if err != nil {
		return nil, err
	}
	containerID := ""
	var trace TraceReader
	var selectedMod, selectedSum []byte
	var firstDiagnostic TraceDiagnostic
	phase := "CREATE"
	defer func() {
		primaryFailed := resultErr != nil
		var cleanupErr error
		if containerID != "" {
			cleanupCtx, cancel := resolverCleanupContext()
			_, removeErr := r.runner.Output(cleanupCtx, "docker", "rm", "--force", containerID)
			cancel()
			if removeErr != nil {
				cleanupErr = errors.New("go resolver container cleanup failed")
			}
			if trace != nil {
				collectCtx, cancel := resolverCleanupContext()
				_, limitation, diagnostic := collectTraceDiagnostic(collectCtx, trace)
				firstDiagnostic = diagnostic
				cancel()
				if limitation != "" {
					cleanupErr = errors.Join(cleanupErr, fmt.Errorf("go resolver observation failed: %s", diagnostic))
				}
				for _, kind := range []string{"process-exec-unexpected", "filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
					if diagnostic.KindCounts[kind] != 0 {
						cleanupErr = errors.Join(cleanupErr, errors.New("go resolver crossed its controlled boundary"))
					}
				}
			}
		}
		cleanupCtx, cancel := resolverCleanupContext()
		cleanupErr = errors.Join(cleanupErr, policy.Close(cleanupCtx))
		cancel()
		resultErr = errors.Join(resultErr, cleanupErr)
		if !primaryFailed && cleanupErr != nil {
			phase = "CLEANUP"
		}
		if resultErr == nil {
			phase = "PUBLISH_CONTROLS"
			resultErr = storeGoResolverControls(workspace, beforeMod, beforeSum, selectedMod, selectedSum)
		}
		if resultErr != nil {
			output = nil
			resultErr = &isolatedGoResolverFailure{cause: resultErr, phase: phase, diagnostic: firstDiagnostic}
		}
	}()
	created, err := r.runner.Output(ctx, "docker", goResolverCreateArguments(network, hosts)...)
	if err != nil || !containerIDPattern.MatchString(strings.TrimSpace(string(created))) {
		return nil, errors.New("create go resolver container failed")
	}
	containerID = strings.TrimSpace(string(created))
	phase = "REGISTER"
	trace, err = startTrace(ctx, r.observer, containerID, goResolverProfile)
	if err != nil {
		return nil, fmt.Errorf("register go resolver observer failed: %s", pythonCommandErrorReason(err))
	}
	phase = "START"
	if _, err := r.runner.Output(ctx, "docker", "start", containerID); err != nil {
		return nil, errors.New("start go resolver container failed")
	}
	phase = "HELPER"
	if err := awaitBoundaryHelper(ctx, r.runner, containerID); err != nil {
		return nil, err
	}
	phase = "TOPOLOGY"
	if err := awaitMountAnchors(ctx, r.observer, containerID); err != nil {
		return nil, errors.New("go resolver topology unavailable")
	}
	// The locked Go tool's default local telemetry starts a sidecar even for
	// version. Disable it in the disposable, HAA-owned configuration before
	// any Go command; telemetry is outside the resolver command grammar.
	input := r.runner.(inputCommandRunner)
	phase = "CONFIG"
	if err := input.RunInput(ctx, strings.NewReader("off\n"), "docker", boundaryInputExecArguments(containerID, boundaryLaunchMode, "/bin/sh", "-ceu", "umask 077; mkdir -p /tmp/.config/go/telemetry; cat > /tmp/.config/go/telemetry/mode")...); err != nil {
		return nil, errors.New("initialize go resolver configuration failed")
	}
	phase = "TOOLCHAIN_QUERY"
	version, err := r.boundedOutput(ctx, containerID, goResolverBinary, "version")
	if err != nil {
		return nil, errors.New("go resolver toolchain query failed")
	}
	phase = "TOOLCHAIN_IDENTITY"
	if strings.TrimSpace(string(version)) != "go version go"+capability.Runtime.GoVersion+" linux/amd64" {
		return nil, errors.New("go resolver toolchain identity mismatch")
	}
	phase = "CONTROLS"
	for _, control := range []struct {
		name string
		body []byte
	}{{"go.mod", beforeMod}, {"go.sum", beforeSum}} {
		command := "umask 077; mkdir -p " + goResolverGuestProject + "; cat > " + goResolverGuestProject + "/" + control.name
		if err := input.RunInput(ctx, bytes.NewReader(control.body), "docker", boundaryInputExecArguments(containerID, boundaryLaunchMode, "/bin/sh", "-ceu", command)...); err != nil {
			return nil, errors.New("copy go resolver control failed")
		}
	}
	operationCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	phase = "COMMAND"
	output, err = r.boundedOutput(operationCtx, containerID, append([]string{goResolverBinary, "-C", goResolverGuestProject}, arguments...)...)
	if err != nil {
		return nil, errors.New("isolated go resolver command failed")
	}
	phase = "SELECTED_MOD"
	selectedMod, err = r.boundedOutput(ctx, containerID, "/bin/cat", goResolverGuestProject+"/go.mod")
	if err != nil || artifactgomodule.ValidateProjectMod(selectedMod) != nil {
		return nil, errors.New("go resolver selected module control invalid")
	}
	phase = "SELECTED_SUM"
	selectedSum, err = r.boundedOutput(ctx, containerID, "/bin/cat", goResolverGuestProject+"/go.sum")
	if err != nil || len(selectedSum) == 0 {
		return nil, errors.New("go resolver selected checksum control invalid")
	}
	return output, nil
}

func validGoResolverCommand(arguments []string) bool {
	if len(arguments) == 2 && arguments[0] == "get" {
		_, err := artifactgomodule.ParseReference(arguments[1])
		return err == nil
	}
	return strings.Join(arguments, "\x00") == "mod\x00download\x00-json\x00all" || strings.Join(arguments, "\x00") == "mod\x00graph"
}

func (r *IsolatedGoModuleRunner) boundedOutput(ctx context.Context, containerID string, command ...string) ([]byte, error) {
	writer := &goResolverBoundedOutput{limit: goResolverOutputLimit}
	runner := r.runner.(interface {
		RunOutput(context.Context, io.Writer, string, ...string) error
	})
	if err := runner.RunOutput(ctx, writer, "docker", boundaryExecArguments(containerID, boundaryLaunchMode, command...)...); err != nil || writer.exceeded {
		return nil, errors.New("go resolver output incomplete or excessive")
	}
	return append([]byte(nil), writer.Bytes()...), nil
}

type goResolverBoundedOutput struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (w *goResolverBoundedOutput) Write(body []byte) (int, error) {
	if len(body) > w.limit-w.Len() {
		w.exceeded = true
		return 0, errors.New("go resolver output exceeds bound")
	}
	return w.Buffer.Write(body)
}

func goResolverCreateArguments(network string, hosts []string) []string {
	arguments := []string{"create", "--pull", "never", "--runtime", gVisorRuntimeName, "--network", network,
		"--read-only", "--cap-drop", "ALL", "--cap-add", "SETUID", "--cap-add", "SETGID", "--cap-add", "SETPCAP", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", "512m", "--cpus", "1",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=512m,uid=1000,gid=1000,mode=0700", "--tmpfs", boundaryHelperMount, "--workdir", "/tmp"}
	arguments = append(arguments, isolatedContainerEnvironmentArguments()...)
	environment, _ := artifactgomodule.ResolverEnvironmentForCache(goResolverGuestCache)
	environment = append(environment, "HOME=/tmp", "XDG_CONFIG_HOME=/tmp/.config", "GOPATH=/tmp/haa-go-path", "GOCACHE=/tmp/haa-go-build", "TMPDIR=/tmp", "GOROOT=/usr/local/go", "GODEBUG=netdns=go")
	for _, value := range environment {
		arguments = append(arguments, "--env", value)
	}
	arguments = append(arguments, hosts...)
	return append(arguments, PinnedGoRuntime().ImageReference, "/bin/sh", "-ceu", boundaryContainerCommand())
}

func storeGoResolverControls(workspace string, beforeMod, beforeSum, selectedMod, selectedSum []byte) error {
	currentControls, err := readGoFrozenControls(workspace)
	if err != nil {
		return errors.New("go resolver private controls changed")
	}
	currentMod, currentSum, err := goFrozenControlBodies(currentControls)
	if err != nil || !bytes.Equal(currentMod, beforeMod) || !bytes.Equal(currentSum, beforeSum) {
		return errors.New("go resolver private controls changed")
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return errors.New("go resolver private control root unavailable")
	}
	defer root.Close()
	for _, control := range []struct {
		name string
		body []byte
	}{{"go.mod", selectedMod}, {"go.sum", selectedSum}} {
		if len(control.body) == 0 || len(control.body) > artifactgomodule.MaxProjectControlBytes {
			return errors.New("go resolver selected controls exceed bound")
		}
		info, err := root.Lstat(control.name)
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("go resolver control identity changed")
		}
		file, err := root.OpenFile(control.name, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return errors.New("open go resolver private control")
		}
		opened, statErr := file.Stat()
		if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
			_ = file.Close()
			return errors.New("go resolver control identity changed")
		}
		if err := file.Truncate(0); err != nil {
			_ = file.Close()
			return errors.New("truncate go resolver private control")
		}
		_, writeErr := file.Write(control.body)
		syncErr := file.Sync()
		closeErr := file.Close()
		if errors.Join(writeErr, syncErr, closeErr) != nil {
			return errors.New("write go resolver private control")
		}
	}
	return nil
}

func goResolverNetworkArguments(byName map[string][]netip.Addr) ([]netip.Addr, []string, error) {
	if len(byName) != 2 {
		return nil, nil, errors.New("go resolver endpoint set invalid")
	}
	var addresses []netip.Addr
	var arguments []string
	seen := map[netip.Addr]bool{}
	for _, name := range []string{"proxy.golang.org", "sum.golang.org"} {
		values := append([]netip.Addr(nil), byName[name]...)
		if len(values) == 0 {
			return nil, nil, errors.New("go resolver endpoint missing")
		}
		sort.Slice(values, func(i, j int) bool { return values[i].Less(values[j]) })
		for i, address := range values {
			if !address.IsValid() || !address.Is4() || address.IsPrivate() || address.IsLoopback() || address.IsMulticast() || address.IsUnspecified() || (i > 0 && values[i-1] == address) {
				return nil, nil, errors.New("go resolver endpoint unsafe")
			}
			arguments = append(arguments, "--add-host", name+":"+address.String())
			if !seen[address] {
				addresses = append(addresses, address)
				seen[address] = true
			}
		}
	}
	if err := validateResolverEndpoints(addresses); err != nil {
		return nil, nil, err
	}
	return addresses, arguments, nil
}

type goNamedEndpointResolver struct{}

func (goNamedEndpointResolver) Resolve(ctx context.Context, names []string) (map[string][]netip.Addr, error) {
	if ctx == nil || len(names) != 2 || names[0] != "proxy.golang.org" || names[1] != "sum.golang.org" {
		return nil, errors.New("go resolver endpoint names invalid")
	}
	result := make(map[string][]netip.Addr, 2)
	for _, name := range names {
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", name)
		if err != nil {
			return nil, errors.New("go resolver endpoint lookup failed")
		}
		result[name] = addresses
	}
	return result, nil
}
