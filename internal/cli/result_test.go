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
