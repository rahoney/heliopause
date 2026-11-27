package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/testutil/fakeworkflow"
)

func TestSyntheticInspectCLIContracts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		scenario       fakeworkflow.Scenario
		exitCode       int
		operationError error
		decision       string
	}{
		{name: "safe", scenario: fakeworkflow.Safe, exitCode: 0, decision: "ALLOW"},
		{name: "review", scenario: fakeworkflow.Review, exitCode: 10, decision: "MANUAL_REVIEW"},
		{name: "blocked", scenario: fakeworkflow.Blocked, exitCode: 20, decision: "BLOCK"},
		{name: "acquire error", scenario: fakeworkflow.AcquireError, exitCode: 1, operationError: fakeworkflow.ErrAcquire},
		{name: "resolve error", scenario: fakeworkflow.ResolveError, exitCode: 1, operationError: fakeworkflow.ErrResolve},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			service, request := cliInspectFixture(t, test.scenario)
			var machine bytes.Buffer
			exitCode, err := cli.ExecuteInspect(context.Background(), service, request, true, &machine)
			if exitCode != test.exitCode || !errors.Is(err, test.operationError) {
				t.Fatalf("ExecuteInspect() exit=%d error=%v", exitCode, err)
			}
			var document map[string]any
			if err := json.Unmarshal(machine.Bytes(), &document); err != nil {
				t.Fatalf("machine JSON = %q: %v", machine.String(), err)
			}
			if document["schema_version"] != "helox.operation-result/v1" || document["operation_id"] != "op_aaaaaaaaaaaaaaaaaaaaaaaaaa" {
				t.Fatalf("machine identity = %#v", document)
			}
			if test.decision != "" {
				policyDocument, ok := document["policy"].(map[string]any)
				if !ok || policyDocument["decision"] != test.decision {
					t.Fatalf("machine Policy = %#v", document["policy"])
				}
			} else if _, ok := document["policy"]; ok {
				t.Fatal("operational failure JSON contains Policy")
			}
			if test.operationError != nil && strings.Contains(machine.String(), test.operationError.Error()) {
				t.Fatalf("machine result disclosed raw operational cause: %q", machine.String())
			}

			service, request = cliInspectFixture(t, test.scenario)
			var human bytes.Buffer
			if humanExit, humanErr := cli.ExecuteInspect(context.Background(), service, request, false, &human); humanExit != exitCode || !errors.Is(humanErr, test.operationError) {
				t.Fatalf("human ExecuteInspect() exit=%d error=%v", humanExit, humanErr)
			}
			for _, shared := range []string{"op_aaaaaaaaaaaaaaaaaaaaaaaaaa", string(domain.OperationInspect), string(document["operation_status"].(string))} {
				if !strings.Contains(human.String(), shared) {
					t.Fatalf("human result missing %q: %q", shared, human.String())
				}
			}
			if run, ok := document["run"].(map[string]any); ok && !strings.Contains(human.String(), run["id"].(string)) {
				t.Fatalf("human result missing Run ID: %q", human.String())
			}
			artifactDocument := document["artifact"].(map[string]any)
			if identity, ok := artifactDocument["resolved_identity"].(map[string]any); ok {
				for _, shared := range []string{identity["source_id"].(string), identity["name"].(string), identity["version"].(string), identity["variant"].(string)} {
					if !strings.Contains(human.String(), shared) {
						t.Fatalf("human result missing Artifact identity %q: %q", shared, human.String())
					}
				}
			}
			if digest, ok := artifactDocument["digest"].(map[string]any); ok && !strings.Contains(human.String(), digest["value"].(string)) {
				t.Fatalf("human result missing digest: %q", human.String())
			}
			if test.decision != "" && !strings.Contains(human.String(), test.decision) {
				t.Fatalf("human result missing decision: %q", human.String())
			}
			if test.operationError != nil && strings.Contains(human.String(), test.operationError.Error()) {
				t.Fatalf("human result disclosed raw operational cause: %q", human.String())
			}
		})
	}
}

func TestExecuteInspectRejectsMissingAdapterDependency(t *testing.T) {
	t.Parallel()

	if exitCode, err := cli.ExecuteInspect(context.Background(), nil, application.InspectRequest{}, true, &bytes.Buffer{}); exitCode != 2 || err == nil {
		t.Fatalf("ExecuteInspect() exit=%d error=%v", exitCode, err)
	}
}

func cliInspectFixture(t *testing.T, scenario fakeworkflow.Scenario) (*application.InspectService, application.InspectRequest) {
	t.Helper()
	fake, err := fakeworkflow.New(scenario)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := fake.Reference()
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewInspectRequest(reference)
	if err != nil {
		t.Fatal(err)
	}
	operationID, err := domain.ParseOperationID("op_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	runID, err := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewInspectService(fake, fake, fake, fake, policy.M1{}, func() (domain.OperationID, error) { return operationID, nil }, func() (domain.RunID, error) { return runID, nil })
	if err != nil {
		t.Fatal(err)
	}
	return service, request
}

func TestHumanAndMachineKeepDeclaredCoverageOnMissingResult(t *testing.T) {
	for _, missing := range []bool{false, true} {
		service, request := cliInspectFixture(t, fakeworkflow.Safe)
		original, err := service.Inspect(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		run, _ := original.RunID()
		identity, _ := original.ResolvedIdentity()
		digest, _ := original.Digest()
		artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+run.String()+":tarball", 1)
		if err != nil {
			t.Fatal(err)
		}
		profile, _ := domain.CheckProfileDigest("cli-coverage-fixture-v1", "")
		var requirements []domain.CheckRequirement
		for index, name := range []string{"fixture-integrity", "fixture-static"} {
			id, _ := domain.NewCheckID(name)
			kind := domain.CheckInspection
			if index == 0 {
				kind = domain.CheckVerification
			}
			requirement, err := domain.NewCheckRequirement(id, kind, true, profile)
			if err != nil {
				t.Fatal(err)
			}
			requirements = append(requirements, requirement)
		}
		plan, err := domain.NewCheckPlan(run, artifact, requirements)
		if err != nil {
			t.Fatal(err)
		}
		var checks []domain.CheckExecution
		for _, check := range original.Checks() {
			bound, err := domain.NewCheckExecution(check.ID(), check.Kind(), check.Required(), check.Capability(), check.Status(), "", profile)
			if err != nil {
				t.Fatal(err)
			}
			checks = append(checks, bound)
		}
		decision, _ := original.PolicyDecision()
		data := domain.OperationResultData{OperationID: original.OperationID(), Status: domain.OperationCompleted, Reference: original.Reference(), ResolvedIdentity: identity, Digest: digest, RunID: run, RunOutcome: domain.RunCompleted, Checks: checks, CheckPlan: plan, Evidence: original.Evidence(), PolicyDecision: decision}
		if missing {
			failure, _ := domain.NewOperationError("POLICY_INPUT_INVALID", "Required check result missing.")
			data.Status = domain.OperationFailed
			data.RunOutcome = domain.RunFailed
			data.Checks = data.Checks[:1]
			data.PolicyDecision = domain.PolicyDecision{}
			data.OperationalError = failure
		}
		result, err := domain.NewInspectOperationResult(data)
		if err != nil {
			t.Fatal(err)
		}
		var machine, human bytes.Buffer
		if err := cli.WriteJSONResult(&machine, result); err != nil {
			t.Fatal(err)
		}
		if err := cli.WriteHumanResult(&human, result); err != nil {
			t.Fatal(err)
		}
		t.Logf("S09_SCHEMA_FIXTURE=%s", machine.Bytes())
		var value struct {
			SchemaVersion string `json:"schema_version"`
			Checks        []struct {
				Profile string `json:"profile_sha256"`
			} `json:"checks"`
			Plan struct {
				Run          string `json:"run_id"`
				SHA256       string `json:"sha256"`
				Requirements []struct {
					ID       string `json:"id"`
					Required bool   `json:"required"`
					Profile  string `json:"profile_sha256"`
				} `json:"requirements"`
			} `json:"check_plan"`
		}
		if err := json.Unmarshal(machine.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if value.SchemaVersion != "helox.operation-result/v2" || value.Plan.Run != run.String() || value.Plan.SHA256 != digest.String() || len(value.Plan.Requirements) != 2 || len(value.Checks) != len(data.Checks) {
			t.Fatal("declared and actual coverage collapsed")
		}
		for _, requirement := range value.Plan.Requirements {
			if !requirement.Required || requirement.Profile != profile.String() || !strings.Contains(human.String(), requirement.ID) || !strings.Contains(human.String(), requirement.Profile) {
				t.Fatal("human/JSON coverage facts differ")
			}
		}
	}
}
