// Package hosttool owns the production Host executable and local Docker
// daemon trust boundary. It deliberately exposes only the narrow command
// methods consumed by infrastructure adapters.
package hosttool

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const (
	defaultDockerEndpoint = "unix:///run/docker.sock"
	systemConfigPath      = "/etc/heliopause/host-tools.json"
	defaultObserverHelper = "/usr/libexec/heliopause/haa_gvisor_observer"
)

var (
	minimumDockerEngine = runtimeidentity.DockerMinimumEngine
	gVisorRelease       = runtimeidentity.GVisorRelease
)

// Config is trusted installation configuration, not user request data. Paths
// are absolute and the Docker endpoint must use a local Unix socket.
type Config struct {
	DockerPath         string `json:"docker_path"`
	DockerEndpoint     string `json:"docker_endpoint"`
	ObserverHelperPath string `json:"observer_helper_path"`
	GoPath             string `json:"go_path,omitempty"`
}

// GoCommandRunner is the explicit environment/working-directory boundary
// consumed by ecosystem adapters. It never inherits the caller environment.
func (e *Executor) RunGo(ctx context.Context, directory string, environment []string, arguments ...string) ([]byte, error) {
	if e == nil || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || verifyNoSymlinkPath(directory, false) != nil {
		return nil, errors.New("trusted Go working directory is unavailable")
	}
	if _, err := os.Stat(directory); err != nil {
		return nil, errors.New("trusted Go working directory is unavailable")
	}
	if _, err := e.tool("go"); err != nil {
		for _, candidate := range []string{"/usr/local/go/bin/go", "/usr/bin/go", "/bin/go"} {
			if verified, verifyErr := verifySystemExecutable(candidate); verifyErr == nil {
				e.tools["go"] = verified
				break
			}
		}
	}
	command, err := e.command(ctx, "go", arguments...)
	if err != nil {
		return nil, err
	}
	command.Dir = directory
	command.Env = append(minimalEnvironment(e.clientHome), append([]string(nil), environment...)...)
	return command.Output()
}

type identity struct {
	path          string
	info          os.FileInfo
	digest        string
	systemSymlink bool
}

// Executor executes only registered trusted tools with a fresh minimal
// environment. It never searches PATH or consumes Docker context variables.
type Executor struct {
	tools      map[string]identity
	endpoint   string
	endpointID os.FileInfo
	clientHome string
}

// NewSystem validates the supported Host installation and the daemon's actual
// runsc-trace registration before returning an executor.
func NewSystem(ctx context.Context) (*Executor, error) {
	if runtime.GOOS != "linux" {
		return nil, errors.New("trusted Host tools require Linux")
	}
	config, err := loadSystemConfig()
	if err != nil {
		return nil, err
	}
	return New(ctx, config)
}

func loadSystemConfig() (Config, error) {
	config := Config{DockerPath: firstExisting("/usr/bin/docker", "/usr/local/bin/docker"), DockerEndpoint: defaultDockerEndpoint, ObserverHelperPath: defaultObserverHelper}
	if _, err := os.Lstat(systemConfigPath); errors.Is(err, os.ErrNotExist) {
		return config, nil
	} else if err != nil {
		return Config{}, errors.New("inspect trusted Host tool configuration")
	}
	verified, err := verifyExecutable(systemConfigPath, "")
	if err != nil {
		return Config{}, fmt.Errorf("verify trusted Host tool configuration: %w", err)
	}
	file, err := os.Open(verified.path)
	if err != nil {
		return Config{}, errors.New("open bounded trusted Host tool configuration")
	}
	body, readErr := io.ReadAll(io.LimitReader(file, 16*1024+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(body) > 16*1024 {
		return Config{}, errors.New("read bounded trusted Host tool configuration")
	}
	current, err := os.Lstat(verified.path)
	if err != nil || !os.SameFile(verified.info, current) {
		return Config{}, errors.New("trusted Host tool configuration changed while reading")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return Config{}, errors.New("parse trusted Host tool configuration")
	}
	return config, nil
}

// New constructs a validated executor from trusted installation config.
func New(ctx context.Context, config Config) (*Executor, error) {
	return newExecutor(ctx, config, false)
}

func newExecutor(ctx context.Context, config Config, includeFirewall bool) (*Executor, error) {
	if ctx == nil {
		return nil, errors.New("trusted Host tool context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	docker, err := verifyExecutable(config.DockerPath, "")
	if err != nil {
		return nil, fmt.Errorf("verify Docker executable: %w", err)
	}
	endpoint, err := verifyLocalSocket(config.DockerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("verify local Docker endpoint: %w", err)
	}
	clientHome, err := os.MkdirTemp("", "heliopause-docker-client-")
	if err != nil {
		return nil, fmt.Errorf("create isolated Docker client configuration: %w", err)
	}
	if err := os.Chmod(clientHome, 0o700); err != nil {
		_ = os.RemoveAll(clientHome)
		return nil, fmt.Errorf("protect isolated Docker client configuration: %w", err)
	}
	endpointInfo, err := os.Lstat(strings.TrimPrefix(endpoint, "unix://"))
	if err != nil {
		_ = os.RemoveAll(clientHome)
		return nil, errors.New("capture local Docker endpoint identity")
	}
	executor := &Executor{tools: map[string]identity{"docker": docker}, endpoint: endpoint, endpointID: endpointInfo, clientHome: clientHome}
	if config.GoPath != "" {
		goTool, goErr := verifySystemExecutable(config.GoPath)
		if goErr != nil {
			_ = executor.Close()
			return nil, fmt.Errorf("verify Go executable: %w", goErr)
		}
		executor.tools["go"] = goTool
	}
	if err := executor.validateDaemon(ctx); err != nil {
		_ = executor.Close()
		return nil, err
	}
	if !includeFirewall {
		return executor, nil
	}
	for name, candidates := range map[string][]string{
		"iptables": {"/usr/sbin/iptables", "/usr/bin/iptables"},
		"nft":      {"/usr/sbin/nft", "/usr/bin/nft"},
	} {
		path := firstExisting(candidates...)
		if path == "" {
			continue
		}
		tool, toolErr := verifySystemExecutable(path)
		if toolErr != nil {
			_ = executor.Close()
			return nil, fmt.Errorf("verify %s executable: %w", name, toolErr)
		}
		executor.tools[name] = tool
	}
	return executor, nil
}

func (e *Executor) validateDaemon(ctx context.Context) error {
	version, err := e.output(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	if err != nil || !atLeastVersion(strings.TrimSpace(string(version)), minimumDockerEngine) {
		return errors.New("local Docker daemon version is unavailable or unsupported")
	}
	registration, err := e.output(ctx, "docker", "info", "--format", "{{json (index .Runtimes \"runsc-trace\")}}")
	if err != nil {
		return errors.New("local Docker daemon runtime registration is unavailable")
	}
	registeredPath, err := parseRunscRegistration(registration)
	if err != nil {
		return err
	}
	expected, ok := runtimeidentity.RunscSHA512(runtime.GOARCH)
	if !ok {
		return errors.New("runsc-trace architecture is unsupported")
	}
	runsc, err := verifyExecutable(registeredPath, expected)
	if err != nil {
		return fmt.Errorf("verify registered runsc-trace executable: %w", err)
	}
	e.tools["runsc"] = runsc
	output, err := e.output(ctx, "runsc", "--version")
	if err != nil || !strings.Contains(string(output), gVisorRelease) {
		delete(e.tools, "runsc")
		return errors.New("registered runsc-trace release mismatch")
	}
	return nil
}

// Close removes the isolated Docker client state. A failed cleanup is a
// production lifecycle failure and must be propagated by its owner.
func (e *Executor) Close() error {
	if e == nil || e.clientHome == "" {
		return nil
	}
	path := e.clientHome
	e.clientHome = ""
	return os.RemoveAll(path)
}

// LookPath returns only a registered absolute identity and revalidates it.
func (e *Executor) LookPath(name string) (string, error) {
	tool, err := e.tool(name)
	if err != nil {
		return "", err
	}
	return tool.path, nil
}

// Output executes a trusted tool and returns stdout.
func (e *Executor) Output(ctx context.Context, name string, arguments ...string) ([]byte, error) {
	return e.output(ctx, name, arguments...)
}

func (e *Executor) output(ctx context.Context, name string, arguments ...string) ([]byte, error) {
	command, err := e.command(ctx, name, arguments...)
	if err != nil {
		return nil, err
	}
	return command.Output()
}

// RunInput executes with an explicit input stream and closed output boundary.
func (e *Executor) RunInput(ctx context.Context, input io.Reader, name string, arguments ...string) error {
	command, err := e.command(ctx, name, arguments...)
	if err != nil {
		return err
	}
	command.Stdin = input
	return command.Run()
}

// RunOutput executes with an explicit output stream.
func (e *Executor) RunOutput(ctx context.Context, output io.Writer, name string, arguments ...string) error {
	command, err := e.command(ctx, name, arguments...)
	if err != nil {
		return err
	}
	command.Stdout = output
	return command.Run()
}

// RunDiscard executes while discarding both bounded diagnostic streams.
func (e *Executor) RunDiscard(ctx context.Context, name string, arguments ...string) error {
	command, err := e.command(ctx, name, arguments...)
	if err != nil {
		return err
	}
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	return command.Run()
}

// Run executes one Docker Promotion command. The project argument is already
// validated by Promotion and is not inherited as process working directory.
func (e *Executor) Run(ctx context.Context, _ string, arguments []string) error {
	command, err := e.command(ctx, "docker", arguments...)
	if err != nil {
		return err
	}
	if _, err := command.CombinedOutput(); err != nil {
		return errors.New("trusted Docker command failed")
	}
	return nil
}

func (e *Executor) command(ctx context.Context, name string, arguments ...string) (*exec.Cmd, error) {
	if ctx == nil {
		return nil, errors.New("trusted command context is required")
	}
	tool, err := e.tool(name)
	if err != nil {
		return nil, err
	}
	if name == "docker" {
		currentEndpoint, endpointErr := verifyLocalSocket(e.endpoint)
		if endpointErr != nil || currentEndpoint != e.endpoint {
			return nil, errors.New("local Docker endpoint changed after validation")
		}
		currentInfo, statErr := os.Lstat(strings.TrimPrefix(currentEndpoint, "unix://"))
		if statErr != nil || !os.SameFile(e.endpointID, currentInfo) {
			return nil, errors.New("local Docker endpoint identity changed after validation")
		}
		arguments = dockerArguments(e.endpoint, e.clientHome, arguments)
	}
	command := exec.CommandContext(ctx, tool.path, arguments...)
	command.Env = minimalEnvironment(e.clientHome)
	command.Stdin = nil
	return command, nil
}

func dockerArguments(endpoint, clientHome string, arguments []string) []string {
	return append([]string{"--host", endpoint, "--config", clientHome}, arguments...)
}

func minimalEnvironment(clientHome string) []string {
	return []string{"HOME=" + clientHome, "DOCKER_CONFIG=" + clientHome, "LANG=C", "LC_ALL=C"}
}

func parseRunscRegistration(body []byte) (string, error) {
	var registered struct {
		Path string `json:"path"`
	}
	if json.Unmarshal(body, &registered) != nil || !filepath.IsAbs(registered.Path) || filepath.Clean(registered.Path) != registered.Path {
		return "", errors.New("runsc-trace registration has no canonical absolute executable identity")
	}
	return registered.Path, nil
}

func (e *Executor) tool(name string) (identity, error) {
	if e == nil || e.clientHome == "" {
		return identity{}, errors.New("trusted host executor is closed or unavailable")
	}
	tool, ok := e.tools[name]
	if !ok {
		return identity{}, errors.New("host tool is not registered")
	}
	var current identity
	var err error
	if tool.systemSymlink {
		current, err = verifySystemExecutable(tool.path)
	} else {
		current, err = verifyExecutable(tool.path, tool.digest)
	}
	if err != nil || !os.SameFile(tool.info, current.info) {
		return identity{}, errors.New("host tool identity changed after validation")
	}
	return current, nil
}

// verifySystemExecutable permits only root-owned distribution alternatives
// whose protected parent chain and resolved executable remain non-writable by
// non-root. User-controlled or writable symlinks still fail closed.
func verifySystemExecutable(path string) (identity, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return identity{}, errors.New("system executable path is not canonical and absolute")
	}
	leaf, err := os.Lstat(path)
	if err != nil {
		return identity{}, errors.New("system executable is unavailable")
	}
	if leaf.Mode()&os.ModeSymlink == 0 {
		return verifyExecutable(path, "")
	}
	if err := verifyTrustedOwner(leaf); err != nil {
		return identity{}, err
	}
	if err := verifyTrustedParents(filepath.Dir(path)); err != nil {
		return identity{}, err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) {
		return identity{}, errors.New("system executable link cannot be resolved")
	}
	verified, err := verifyExecutable(resolved, "")
	if err != nil {
		return identity{}, err
	}
	verified.path = path
	verified.systemSymlink = true
	return verified, nil
}

func verifyExecutable(path, expectedDigest string) (identity, error) {
	if !filepath.IsAbs(path) {
		return identity{}, errors.New("executable path is not absolute")
	}
	clean := filepath.Clean(path)
	if clean != path {
		return identity{}, errors.New("executable path is not canonical")
	}
	if err := verifyNoSymlinkPath(clean, false); err != nil {
		return identity{}, err
	}
	info, err := os.Lstat(clean)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o022 != 0 {
		return identity{}, errors.New("executable is unavailable, non-regular, or writable by non-owner")
	}
	if err := verifyTrustedOwner(info); err != nil {
		return identity{}, err
	}
	if err := verifyTrustedParents(filepath.Dir(clean)); err != nil {
		return identity{}, err
	}
	digest := ""
	if expectedDigest != "" {
		file, openErr := os.Open(clean)
		if openErr != nil {
			return identity{}, errors.New("open executable for identity verification")
		}
		hash := sha512.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			return identity{}, errors.New("hash executable identity")
		}
		digest = hex.EncodeToString(hash.Sum(nil))
		if digest != expectedDigest {
			return identity{}, errors.New("executable digest mismatch")
		}
	}
	return identity{path: clean, info: info, digest: digest}, nil
}

func verifyLocalSocket(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "unix" || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || !filepath.IsAbs(parsed.Path) {
		return "", errors.New("docker endpoint is not an absolute local Unix socket")
	}
	resolved, err := filepath.EvalSymlinks(parsed.Path)
	if err != nil || !filepath.IsAbs(resolved) {
		return "", errors.New("docker endpoint cannot be resolved")
	}
	info, err := os.Lstat(resolved)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return "", errors.New("docker endpoint is not a Unix socket")
	}
	if err := verifyEndpointOwner(info); err != nil {
		return "", err
	}
	if err := verifyTrustedEndpointParents(filepath.Dir(resolved)); err != nil {
		return "", err
	}
	return "unix://" + resolved, nil
}

func verifyTrustedEndpointParents(path string) error {
	for current := filepath.Clean(path); current != filepath.Dir(current); current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("docker endpoint parent is unavailable or not a real directory")
		}
		if info.Mode()&0o022 != 0 {
			return errors.New("docker endpoint parent is writable by non-owner")
		}
		if err := verifyEndpointOwner(info); err != nil {
			return err
		}
	}
	return nil
}

func verifyTrustedParents(path string) error {
	for current := filepath.Clean(path); current != filepath.Dir(current); current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("tool parent is unavailable or not a real directory")
		}
		if info.Mode()&0o022 != 0 {
			return errors.New("tool parent is writable by non-owner")
		}
		if err := verifyTrustedOwner(info); err != nil {
			return err
		}
	}
	return nil
}

func verifyNoSymlinkPath(path string, allowLeaf bool) error {
	for current := filepath.Clean(path); current != filepath.Dir(current); current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return errors.New("host path component is unavailable")
		}
		if info.Mode()&os.ModeSymlink != 0 && (!allowLeaf || current == path) {
			return errors.New("host path contains a symbolic link")
		}
	}
	return nil
}

func firstExisting(paths ...string) string {
	for _, path := range paths {
		if _, err := os.Lstat(path); err == nil {
			return path
		}
	}
	return ""
}

func atLeastVersion(actual, minimum string) bool {
	var a, b, c, x, y, z int
	if _, err := fmt.Sscanf(actual, "%d.%d.%d", &a, &b, &c); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(minimum, "%d.%d.%d", &x, &y, &z); err != nil {
		return false
	}
	return a > x || a == x && (b > y || b == y && c >= z)
}
