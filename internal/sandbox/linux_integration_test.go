package sandbox

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

func TestLinuxGVisorLifecycleIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor runtime")
	}
	root := t.TempDir()
	path := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "tarball.tgz")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	writer := tar.NewWriter(gzipWriter)
	body := `{"name":"tiny","version":"1.2.3"}`
	if err := writer.WriteHeader(&tar.Header{Name: "package/package.json", Size: int64(len(body)), Mode: 0o600}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	runner := integrationRunner{t: t}
	introducer, err := NewDockerArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewBackend(runner, introducer, supervisor.Observer(), integrationCapabilityProbe(runner))
	if err != nil {
		t.Fatal(err)
	}
	result, err := backend.Execute(context.Background(), integrationRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status() != domain.SandboxCompleted {
		code, _ := result.LimitationCode()
		t.Fatalf("Sandbox result = %q/%q", result.Status(), code)
	}
}

func TestLinuxGitHubReleaseELFDynamicIntegration(t *testing.T) {
	if os.Getenv("HELOX_GITHUB_RELEASE_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor runtime for GitHub Release ELF")
	}
	root := t.TempDir()
	runID, _ := domain.NewRunID()
	directory := filepath.Join(root, runID.String())
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile("/bin/true")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "asset"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	source, _ := domain.NewSourceID("github-release")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "owner-repo", "v1", "tool")
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+runID.String()+":github-release", uint64(len(content)))
	if err != nil {
		t.Fatal(err)
	}
	request, err := domain.NewSandboxRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	runner := integrationRunner{t: t}
	backend, err := NewGitHubELFBackend(runner, root, supervisor.Observer(), integrationCapabilityProbe(runner))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := backend.Execute(ctx, request)
	if err != nil || result.Status() != domain.SandboxCompleted {
		code, _ := result.LimitationCode()
		t.Fatalf("GitHub ELF dynamic result = %q/%q, %v", result.Status(), code, err)
	}
}

func TestLinuxNPMResolverNetworkPolicyIntegration(t *testing.T) {
	if os.Getenv("HELOX_NPM_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires privileged Linux Docker firewall integration")
	}
	if os.Geteuid() != 0 {
		t.Fatal("resolver network policy integration requires explicit CAP_NET_ADMIN elevation")
	}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	resolver, err := NewNPMResolverWithObserver(integrationRunner{t: t}, systemEndpointResolver{}, supervisor.Observer(), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	reference, err := artifactnpm.ParseReference("is-number@7.0.0")
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/heliopause-resolver-target")
	installContext, _ := domain.NewInstallContext(target)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	resolution, err := resolver.ResolveDependencies(ctx, reference, installContext)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Graph().Nodes()) == 0 || resolution.Graph().Primary().String() == "" || resolution.RuntimeIdentity() == "" || resolution.LockfileDigest().String() == "" {
		t.Fatalf("resolver resolution = %#v", resolution)
	}
}

func TestLinuxPyPIResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYPI_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux Python/gVisor and Docker firewall integration")
	}
	if os.Geteuid() != 0 {
		t.Fatal("PyPI resolver network policy integration requires explicit CAP_NET_ADMIN elevation")
	}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	runner := integrationRunner{t: t}
	resolver, err := NewPyPIResolver(runner, systemNamedEndpointResolver{}, supervisor.Observer(), integrationPythonCapabilityProbe(runner), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resolver.Close() }()
	reference, err := artifactpypi.ParseReference("packaging@25.0")
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/heliopause-pypi-resolver-target")
	installContext, _ := domain.NewInstallContext(target)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	resolution, err := resolver.ResolveDependencies(ctx, reference, installContext)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Graph().Nodes()) != 1 || resolution.Graph().Primary().String() == "" || resolution.RuntimeIdentity() == "" || resolution.LockfileDigest().String() == "" {
		t.Fatalf("PyPI resolver resolution = %#v", resolution)
	}
}

func integrationResolverPolicyService(t *testing.T) ResolverPolicyService {
	t.Helper()
	return &recordingResolverPolicyService{}
}

func TestLinuxPyPIWheelDynamicIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYPI_DYNAMIC_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux Python/gVisor dynamic integration")
	}
	root := t.TempDir()
	wheel := linuxDynamicWheel(t)
	path := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "wheel.whl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, wheel, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(wheel)
	static, err := artifactpypi.InspectWheel(bytes.NewReader(wheel), int64(len(wheel)), "example-1.0-py3-none-any.whl", hex.EncodeToString(sum[:]), artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"}, artifactpypi.DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "wheel")
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:wheel", uint64(len(wheel)))
	if err != nil {
		t.Fatal(err)
	}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	runner := integrationRunner{t: t}
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewPythonDynamicBackend(runner, introducer, supervisor.Observer(), integrationPythonCapabilityProbe(runner))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := backend.InspectWheel(ctx, artifact, static.ImportNames)
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("dynamic result = %#v, %v", result, err)
	}
}

func TestLinuxPyPISdistBuildIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYPI_DYNAMIC_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux Python/gVisor dynamic integration")
	}
	root := t.TempDir()
	runID := "run_aaaaaaaaaaaaaaaaaaaaaaaaaa"
	sourceBytes := linuxBuildSdist(t)
	backendBytes := linuxBuildBackendWheel(t)
	for name, body := range map[string][]byte{"sdist.tar.gz": sourceBytes, "wheel.whl": backendBytes} {
		path := filepath.Join(root, runID, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	sourceSum := sha256.Sum256(sourceBytes)
	recipe, err := artifactpypi.InspectSdist(bytes.NewReader(sourceBytes), "example-1.0.tar.gz", hex.EncodeToString(sourceSum[:]), artifactpypi.DefaultSdistLimits())
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := domain.NewSourceID("pypi")
	sourceIdentity, _ := domain.NewResolvedArtifactIdentity(sourceID, "example", "1.0", "sdist")
	sourceDigest, _ := domain.NewSHA256Digest(hex.EncodeToString(sourceSum[:]))
	sourceArtifact, err := domain.NewAcquiredArtifact(sourceIdentity, sourceDigest, "intake:"+runID+":sdist", uint64(len(sourceBytes)))
	if err != nil {
		t.Fatal(err)
	}
	backendSum := sha256.Sum256(backendBytes)
	backendIdentity, _ := domain.NewResolvedArtifactIdentity(sourceID, "backend", "1.0", "wheel")
	backendDigest, _ := domain.NewSHA256Digest(hex.EncodeToString(backendSum[:]))
	backendArtifact, err := domain.NewAcquiredArtifact(backendIdentity, backendDigest, "intake:"+runID+":wheel", uint64(len(backendBytes)))
	if err != nil {
		t.Fatal(err)
	}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	runner := integrationRunner{t: t}
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	builder, err := NewPythonSdistBuilder(runner, introducer, supervisor.Observer(), integrationPythonCapabilityProbe(runner))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	derived, result, err := builder.Build(ctx, sourceArtifact, recipe, []domain.AcquiredArtifact{backendArtifact})
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("build result = %#v, %v", result, err)
	}
	derivedBytes, err := os.ReadFile(filepath.Join(root, runID, "derived.whl"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifactpypi.InspectWheel(bytes.NewReader(derivedBytes), int64(len(derivedBytes)), derived.Filename, derived.Artifact.Digest().String(), artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"}, artifactpypi.DefaultWheelLimits()); err != nil {
		t.Fatalf("derived wheel static reinspection: %v", err)
	}
}

func linuxDynamicWheel(t *testing.T) []byte {
	t.Helper()
	files := map[string][]byte{
		"example/__init__.py":            []byte("VALUE = 'ok'\n"),
		"example-1.0.dist-info/METADATA": []byte("Metadata-Version: 2.4\nName: example\nVersion: 1.0\nImport-Name: example\n"),
		"example-1.0.dist-info/WHEEL":    []byte("Wheel-Version: 1.0\nGenerator: heliopause-test\nRoot-Is-Purelib: true\nTag: py3-none-any\n"),
	}
	var record strings.Builder
	for _, name := range []string{"example/__init__.py", "example-1.0.dist-info/METADATA", "example-1.0.dist-info/WHEEL"} {
		sum := sha256.Sum256(files[name])
		record.WriteString(name + ",sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]) + "," + fmt.Sprintf("%d", len(files[name])) + "\n")
	}
	record.WriteString("example-1.0.dist-info/RECORD,,\n")
	files["example-1.0.dist-info/RECORD"] = []byte(record.String())
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, name := range []string{"example/__init__.py", "example-1.0.dist-info/METADATA", "example-1.0.dist-info/WHEEL", "example-1.0.dist-info/RECORD"} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func linuxBuildSdist(t *testing.T) []byte {
	t.Helper()
	files := map[string][]byte{
		"example-1.0/PKG-INFO":       []byte("Metadata-Version: 2.4\nName: example\nVersion: 1.0\n"),
		"example-1.0/pyproject.toml": []byte("[build-system]\nrequires = [\"backend\"]\nbuild-backend = \"backend\"\n"),
		"example-1.0/example.py":     []byte("VALUE = 'source'\n"),
	}
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	writer := tar.NewWriter(gzipWriter)
	for _, name := range []string{"example-1.0/PKG-INFO", "example-1.0/pyproject.toml", "example-1.0/example.py"} {
		body := files[name]
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg, Format: tar.FormatPAX, PAXRecords: map[string]string{"comment": "heliopause"}}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func linuxBuildBackendWheel(t *testing.T) []byte {
	t.Helper()
	backend := []byte("import base64,hashlib,os,zipfile\ndef build_wheel(wheel_directory,config_settings=None,metadata_directory=None):\n files={'example/__init__.py':b\"VALUE = 'built'\\n\",'example-1.0.dist-info/METADATA':b'Metadata-Version: 2.4\\nName: example\\nVersion: 1.0\\nImport-Name: example\\n','example-1.0.dist-info/WHEEL':b'Wheel-Version: 1.0\\nGenerator: heliopause-test\\nRoot-Is-Purelib: true\\nTag: py3-none-any\\n'}\n lines=[]\n for name in sorted(files):\n  body=files[name]; lines.append(name+',sha256='+base64.urlsafe_b64encode(hashlib.sha256(body).digest()).decode().rstrip('=')+','+str(len(body)))\n lines.append('example-1.0.dist-info/RECORD,,'); files['example-1.0.dist-info/RECORD']=('\\n'.join(lines)+'\\n').encode()\n path=os.path.join(wheel_directory,'example-1.0-py3-none-any.whl')\n with zipfile.ZipFile(path,'w',zipfile.ZIP_DEFLATED) as z:\n  for name in sorted(files): z.writestr(name,files[name])\n return os.path.basename(path)\n")
	files := map[string][]byte{
		"backend.py":                     backend,
		"backend-1.0.dist-info/METADATA": []byte("Metadata-Version: 2.4\nName: backend\nVersion: 1.0\nImport-Name: backend\n"),
		"backend-1.0.dist-info/WHEEL":    []byte("Wheel-Version: 1.0\nGenerator: heliopause-test\nRoot-Is-Purelib: true\nTag: py3-none-any\n"),
	}
	var record strings.Builder
	for _, name := range []string{"backend.py", "backend-1.0.dist-info/METADATA", "backend-1.0.dist-info/WHEEL"} {
		sum := sha256.Sum256(files[name])
		record.WriteString(name + ",sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]) + "," + fmt.Sprintf("%d", len(files[name])) + "\n")
	}
	record.WriteString("backend-1.0.dist-info/RECORD,,\n")
	files["backend-1.0.dist-info/RECORD"] = []byte(record.String())
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, name := range []string{"backend.py", "backend-1.0.dist-info/METADATA", "backend-1.0.dist-info/WHEEL", "backend-1.0.dist-info/RECORD"} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func integrationObserverSupervisor(t *testing.T) *ObserverSupervisor {
	t.Helper()
	launcher, err := hosttool.NewObserverLauncher("/usr/libexec/heliopause/haa_gvisor_observer")
	if err != nil {
		t.Fatalf("load pinned observer helper: %v", err)
	}
	supervisor, err := NewObserverSupervisor(context.Background(), func(ctx context.Context, remoteEndpoint, outputEndpoint string) (ObserverProcess, error) {
		return launcher.StartObserver(ctx, remoteEndpoint, outputEndpoint)
	})
	if err != nil {
		t.Fatalf("start observer supervisor: %v", err)
	}
	return supervisor
}

type integrationRunner struct {
	t *testing.T
}

func integrationCapabilityProbe(executor Executor) CapabilityProbe {
	return func(ctx context.Context) (Capability, error) {
		return probe(ctx, runtime.GOOS, executor)
	}
}

func integrationPythonCapabilityProbe(executor Executor) func(context.Context) (PythonCapability, error) {
	return func(ctx context.Context) (PythonCapability, error) {
		return probePython(ctx, runtime.GOOS, runtime.GOARCH, executor)
	}
}

func (r integrationRunner) LookPath(binary string) (string, error) {
	r.t.Helper()
	return exec.LookPath(integrationBinary(binary))
}

func integrationBinary(binary string) string {
	// The canonical CI installs the pinned runtime outside PATH so that every
	// parent directory has a trusted identity. Production resolves this through
	// hosttool; this raw integration runner mirrors only that exact test path.
	if binary == "runsc" {
		return "/usr/libexec/heliopause/runsc"
	}
	return binary
}

func (r integrationRunner) Output(ctx context.Context, binary string, arguments ...string) ([]byte, error) {
	r.t.Helper()
	command := exec.CommandContext(ctx, integrationBinary(binary), arguments...)
	output, err := command.Output()
	if err != nil {
		stderr := ""
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			stderr = strings.TrimSpace(string(exitError.Stderr))
		}
		r.t.Logf("command failed: %s %q: %v; stdout=%q; stderr=%q", binary, arguments, err, strings.TrimSpace(string(output)), stderr)
		// This is integration-test-only diagnostic output. Production adapter
		// errors remain sanitized and never carry command output.
		fmt.Fprintf(os.Stderr, "integration command failed: %s %q: %v; stdout=%q; stderr=%q\n", binary, arguments, err, strings.TrimSpace(string(output)), stderr)
	}
	if err == nil && binary == "docker" && len(arguments) == 2 && arguments[0] == "wait" && strings.TrimSpace(string(output)) != "0" {
		logs, logsErr := exec.CommandContext(ctx, "docker", "logs", arguments[1]).CombinedOutput()
		r.t.Logf("container exited with %q; logs=%q; logs error=%v", strings.TrimSpace(string(output)), strings.TrimSpace(string(logs)), logsErr)
	}
	return output, err
}

func (r integrationRunner) RunInput(ctx context.Context, input io.Reader, binary string, arguments ...string) error {
	r.t.Helper()
	command := exec.CommandContext(ctx, integrationBinary(binary), arguments...)
	command.Stdin = input
	output, err := command.CombinedOutput()
	if err != nil {
		r.t.Logf("input command failed: %s %q: %v; output=%q", binary, arguments, err, strings.TrimSpace(string(output)))
	}
	return err
}

func (r integrationRunner) RunDiscard(ctx context.Context, binary string, arguments ...string) error {
	r.t.Helper()
	command := exec.CommandContext(ctx, integrationBinary(binary), arguments...)
	output := &boundedIntegrationOutput{remaining: 16 << 10}
	command.Stdout = output
	command.Stderr = output
	err := command.Run()
	if err != nil {
		r.t.Logf("discard command failed: %s %q: %v; bounded output=%q", binary, arguments, err, output.String())
	}
	return err
}

func (r integrationRunner) RunOutput(ctx context.Context, output io.Writer, binary string, arguments ...string) error {
	r.t.Helper()
	command := exec.CommandContext(ctx, integrationBinary(binary), arguments...)
	command.Stdout = output
	stderr := &boundedIntegrationOutput{remaining: 16 << 10}
	command.Stderr = stderr
	err := command.Run()
	if err != nil {
		r.t.Logf("output command failed: %s %q: %v; bounded stderr=%q", binary, arguments, err, stderr.String())
	}
	return err
}

type boundedIntegrationOutput struct {
	bytes.Buffer
	remaining int
}

func (b *boundedIntegrationOutput) Write(data []byte) (int, error) {
	count := len(data)
	if b.remaining > 0 {
		accepted := data
		if len(accepted) > b.remaining {
			accepted = accepted[:b.remaining]
		}
		_, _ = b.Buffer.Write(accepted)
		b.remaining -= len(accepted)
	}
	return count, nil
}

func integrationRequest(t *testing.T) domain.SandboxRequest {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", 1)
	if err != nil {
		t.Fatal(err)
	}
	request, err := domain.NewSandboxRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	return request
}
