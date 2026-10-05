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

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
)

const (
	cargoResolverProfile      = "cargo-resolver"
	cargoResolverGuestProject = "/tmp/haa-cargo-project"
	cargoResolverGuestHome    = "/tmp/haa-cargo-home"
)

// IsolatedCargoRunner accepts source-selection commands only. Project build
// scripts, procedural macros and builds require the separate build boundary.
type IsolatedCargoRunner struct {
	runner    CommandRunner
	endpoints NamedEndpointResolver
	observer  TraceObserver
	probe     func(context.Context) (CargoCapability, error)
	policy    ResolverPolicyService
}

type isolatedCargoResolverFailure struct {
	cause      error
	phase      string
	diagnostic TraceDiagnostic
}

func (e *isolatedCargoResolverFailure) Error() string {
	if e.diagnostic.Reason != "" {
		return "isolated cargo resolver failed: phase=" + e.phase + " " + e.diagnostic.String()
	}
	return "isolated cargo resolver failed: phase=" + e.phase
}
func (e *isolatedCargoResolverFailure) Unwrap() error { return e.cause }

func NewLinuxCargoRunner(executor TrustedExecutor, observer TraceObserver, policy ResolverPolicyService) (*IsolatedCargoRunner, error) {
	if executor == nil {
		return nil, errors.New("cargo resolver requires trusted executor")
	}
	return newIsolatedCargoRunner(executor, cargoNamedEndpointResolver{}, observer, func(ctx context.Context) (CargoCapability, error) { return ProbeCargo(ctx, executor) }, policy)
}

func newIsolatedCargoRunner(runner CommandRunner, endpoints NamedEndpointResolver, observer TraceObserver, probe func(context.Context) (CargoCapability, error), policy ResolverPolicyService) (*IsolatedCargoRunner, error) {
	if runner == nil || endpoints == nil || observer == nil || probe == nil || policy == nil {
		return nil, errors.New("cargo resolver requires runtime, network and observation boundaries")
	}
	if _, ok := runner.(inputCommandRunner); !ok {
		return nil, errors.New("cargo resolver requires bounded source input")
	}
	if _, ok := runner.(interface {
		RunOutput(context.Context, io.Writer, string, ...string) error
	}); !ok {
		return nil, errors.New("cargo resolver requires bounded output")
	}
	return &IsolatedCargoRunner{admissionAwareRunner(runner), endpoints, observer, probe, policy}, nil
}

func validCargoResolverCommand(arguments []string) bool {
	if strings.Join(arguments, "\x00") == "metadata\x00--locked\x00--format-version\x001" {
		return true
	}
	if len(arguments) != 2 || arguments[0] != "add" {
		return false
	}
	parts := strings.Split(arguments[1], "@=")
	if len(parts) != 2 {
		return false
	}
	_, err := artifactcargo.ParseReference(parts[0] + "@" + parts[1])
	return err == nil
}

func (r *IsolatedCargoRunner) RunCargo(ctx context.Context, workspace string, environment []string, arguments ...string) (output []byte, resultErr error) {
	if r == nil || r.runner == nil || ctx == nil || !validCargoResolverCommand(arguments) {
		return nil, errors.New("cargo resolver command is unsupported")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(environment) == 0 || !strings.HasPrefix(environment[0], "CARGO_HOME=") || ValidateCargoResolverEnvironment(environment, strings.TrimPrefix(environment[0], "CARGO_HOME=")) != nil {
		return nil, errors.New("cargo resolver environment is not canonical")
	}
	source, err := captureCargoProject(ctx, workspace)
	if err != nil {
		return nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, source.close())
		if resultErr != nil {
			output = nil
		}
	}()
	frozen := source.files()
	if arguments[0] == "metadata" && len(frozen["Cargo.lock"]) == 0 {
		return nil, errors.New("cargo metadata requires a frozen lock")
	}
	archive, err := source.archive()
	if err != nil {
		return nil, err
	}
	capability, err := r.probe(ctx)
	if err != nil || !capability.Available || capability.Runtime != PinnedCargoRuntime() {
		return nil, errors.New("cargo resolver pinned runtime unavailable")
	}
	addresses, err := r.endpoints.Resolve(ctx, []string{"index.crates.io", "static.crates.io"})
	if err != nil {
		return nil, errors.New("cargo resolver endpoint lookup failed")
	}
	allowed, hosts, err := cargoResolverNetworkArguments(addresses)
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
	var selectedManifest, selectedLock []byte
	var diagnostic TraceDiagnostic
	phase := "CREATE"
	defer func() {
		primaryFailed := resultErr != nil
		var cleanupErr error
		if containerID != "" {
			cleanupCtx, cancel := resolverCleanupContext()
			_, removeErr := r.runner.Output(cleanupCtx, "docker", "rm", "--force", containerID)
			cancel()
			if removeErr != nil {
				cleanupErr = errors.New("cargo resolver container cleanup failed")
			}
			if trace != nil {
				collectCtx, cancel := resolverCleanupContext()
				_, limitation, observed := collectTraceDiagnostic(collectCtx, trace)
				cancel()
				diagnostic = observed
				if limitation != "" {
					cleanupErr = errors.Join(cleanupErr, fmt.Errorf("cargo resolver observation failed: %s", diagnostic))
				}
				for _, kind := range []string{"process-exec-unexpected", "filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
					if diagnostic.KindCounts[kind] != 0 {
						cleanupErr = errors.Join(cleanupErr, errors.New("cargo resolver crossed its controlled boundary"))
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
			resultErr = storeCargoResolverControls(ctx, source, selectedManifest, selectedLock)
		}
		if resultErr != nil {
			output = nil
			resultErr = &isolatedCargoResolverFailure{resultErr, phase, diagnostic}
		}
	}()
	created, err := r.runner.Output(ctx, "docker", cargoResolverCreateArguments(network, hosts)...)
	if err != nil || !containerIDPattern.MatchString(strings.TrimSpace(string(created))) {
		return nil, errors.New("create cargo resolver container failed")
	}
	containerID = strings.TrimSpace(string(created))
	phase = "REGISTER"
	trace, err = startTrace(ctx, r.observer, containerID, cargoResolverProfile)
	if err != nil {
		return nil, fmt.Errorf("register cargo resolver observer failed: %s", pythonCommandErrorReason(err))
	}
	phase = "START"
	if _, err := r.runner.Output(ctx, "docker", "start", containerID); err != nil {
		return nil, errors.New("start cargo resolver container failed")
	}
	phase = "HELPER"
	if err := awaitBoundaryHelper(ctx, r.runner, containerID); err != nil {
		return nil, err
	}
	phase = "TOPOLOGY"
	if err := awaitMountAnchors(ctx, r.observer, containerID); err != nil {
		return nil, errors.New("cargo resolver topology unavailable")
	}
	phase = "TOOLCHAIN_QUERY"
	version, err := r.boundedOutput(ctx, containerID, capability.Runtime.CargoBinary(), "--version")
	if err != nil || !strings.HasPrefix(string(version), "cargo "+capability.Runtime.RustVersion+" (") || len(version) > 256 {
		return nil, errors.New("cargo resolver toolchain identity mismatch")
	}
	phase = "SOURCE"
	input := r.runner.(inputCommandRunner)
	if err := input.RunInput(ctx, bytes.NewReader(archive), "docker", boundaryInputExecArguments(containerID, boundaryLaunchMode, "/bin/sh", "-ceu", "umask 077; mkdir -p "+cargoResolverGuestProject+"; tar -xf - -C "+cargoResolverGuestProject)...); err != nil {
		return nil, errors.New("copy cargo resolver source failed")
	}
	if err := source.verify(ctx); err != nil {
		return nil, err
	}
	operationCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	command := []string{capability.Runtime.CargoBinary(), arguments[0], "--manifest-path", cargoResolverGuestProject + "/Cargo.toml"}
	if arguments[0] == "add" {
		command = append(command, "--", arguments[1])
	} else {
		command = append(command, arguments[1:]...)
	}
	phase = "COMMAND"
	output, err = r.boundedOutput(operationCtx, containerID, command...)
	if err != nil {
		return nil, err
	}
	phase = "SELECTED_MANIFEST"
	selectedManifest, err = r.boundedOutput(ctx, containerID, "/bin/cat", cargoResolverGuestProject+"/Cargo.toml")
	if err != nil || artifactcargo.ValidateProjectManifest(selectedManifest, "Cargo.toml") != nil {
		return nil, errors.New("cargo resolver selected manifest invalid")
	}
	phase = "SELECTED_LOCK"
	selectedLock, err = r.boundedOutput(ctx, containerID, "/bin/cat", cargoResolverGuestProject+"/Cargo.lock")
	if err != nil || len(selectedLock) == 0 || len(selectedLock) > artifactcargo.MaxProjectControlBytes {
		return nil, errors.New("cargo resolver selected lock invalid")
	}
	if arguments[0] == "metadata" && (!bytes.Equal(selectedManifest, frozen["Cargo.toml"]) || !bytes.Equal(selectedLock, frozen["Cargo.lock"])) {
		return nil, errors.New("cargo metadata changed frozen controls")
	}
	return output, nil
}

func (r *IsolatedCargoRunner) boundedOutput(ctx context.Context, containerID string, command ...string) ([]byte, error) {
	w := &goResolverBoundedOutput{limit: 8 << 20}
	runner := r.runner.(interface {
		RunOutput(context.Context, io.Writer, string, ...string) error
	})
	err := runner.RunOutput(ctx, w, "docker", boundaryExecArguments(containerID, boundaryLaunchMode, command...)...)
	if err != nil || w.exceeded {
		return nil, errors.Join(errors.New("cargo resolver output incomplete or excessive"), err)
	}
	return bytes.Clone(w.Bytes()), nil
}

func cargoResolverCreateArguments(network string, hosts []string) []string {
	arguments := []string{"create", "--pull", "never", "--runtime", gVisorRuntimeName, "--network", network, "--read-only", "--cap-drop", "ALL", "--cap-add", "SETUID", "--cap-add", "SETGID", "--cap-add", "SETPCAP", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", "512m", "--cpus", "1", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=512m,uid=1000,gid=1000,mode=0700", "--tmpfs", boundaryHelperMount, "--workdir", "/tmp"}
	arguments = append(arguments, isolatedContainerEnvironmentArguments()...)
	environment, _ := CargoResolverEnvironmentForHome(cargoResolverGuestHome)
	environment = append(environment, "HOME=/tmp", "TMPDIR=/tmp", "RUSTC="+PinnedCargoRuntime().RustcBinary(), "RUSTDOC="+filepath.Dir(PinnedCargoRuntime().RustcBinary())+"/rustdoc", "CARGO_BUILD_TARGET="+PinnedCargoRuntime().Target, "CARGO_INCREMENTAL=0", "CARGO_TARGET_DIR=/tmp/haa-cargo-target")
	for _, value := range environment {
		arguments = append(arguments, "--env", value)
	}
	arguments = append(arguments, hosts...)
	return append(arguments, PinnedCargoRuntime().ImageReference, "/bin/sh", "-ceu", boundaryContainerCommand())
}

func storeCargoResolverControls(ctx context.Context, source *cargoProjectSource, manifest, lock []byte) (resultErr error) {
	if artifactcargo.ValidateProjectManifest(manifest, "Cargo.toml") != nil || len(lock) == 0 || len(lock) > artifactcargo.MaxProjectControlBytes {
		return errors.New("cargo resolver selected controls invalid")
	}
	if err := source.verify(ctx); err != nil {
		return err
	}
	root := source.roots[0]
	frozen := source.files()
	for _, control := range []struct {
		name string
		body []byte
	}{{"Cargo.toml", manifest}, {"Cargo.lock", lock}} {
		info, statErr := root.Lstat(control.name)
		flags := os.O_WRONLY | syscall.O_NONBLOCK
		if _, present := frozen[control.name]; !present {
			if !errors.Is(statErr, os.ErrNotExist) {
				return errors.New("cargo private control appeared")
			}
			flags |= os.O_CREATE | os.O_EXCL
		} else if statErr != nil || !info.Mode().IsRegular() || !goBuildSingleLink(info) {
			return errors.New("cargo private control changed")
		}
		file, err := root.OpenFile(control.name, flags, 0o600)
		if err != nil {
			return errors.New("open private Cargo selected control")
		}
		opened, err := file.Stat()
		if err != nil || !opened.Mode().IsRegular() || !goBuildSingleLink(opened) || (info != nil && !os.SameFile(info, opened)) {
			return errors.Join(errors.New("cargo private control identity changed"), file.Close())
		}
		if err := file.Truncate(0); err != nil {
			return errors.Join(errors.New("truncate private Cargo selected control"), file.Close())
		}
		_, writeErr := file.Write(control.body)
		if errors.Join(writeErr, file.Sync(), file.Close()) != nil {
			return errors.New("write private Cargo selected control")
		}
	}
	return nil
}

func cargoResolverNetworkArguments(byName map[string][]netip.Addr) ([]netip.Addr, []string, error) {
	if len(byName) != 2 {
		return nil, nil, errors.New("cargo resolver endpoint set invalid")
	}
	var addresses []netip.Addr
	var arguments []string
	seen := map[netip.Addr]bool{}
	for _, name := range []string{"index.crates.io", "static.crates.io"} {
		values := append([]netip.Addr(nil), byName[name]...)
		if len(values) == 0 || len(values) > 16 {
			return nil, nil, errors.New("cargo resolver endpoint missing or excessive")
		}
		sort.Slice(values, func(i, j int) bool { return values[i].Less(values[j]) })
		for index, address := range values {
			if !address.IsValid() || !address.Is4() || address.IsPrivate() || address.IsLoopback() || address.IsMulticast() || address.IsUnspecified() || (index > 0 && values[index-1] == address) {
				return nil, nil, errors.New("cargo resolver endpoint unsafe")
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

type cargoNamedEndpointResolver struct{}

func (cargoNamedEndpointResolver) Resolve(ctx context.Context, names []string) (map[string][]netip.Addr, error) {
	if ctx == nil || len(names) != 2 || names[0] != "index.crates.io" || names[1] != "static.crates.io" {
		return nil, errors.New("cargo resolver endpoint names invalid")
	}
	result := map[string][]netip.Addr{}
	for _, name := range names {
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", name)
		if err != nil {
			return nil, errors.New("cargo resolver endpoint lookup failed")
		}
		result[name] = addresses
	}
	return result, nil
}
