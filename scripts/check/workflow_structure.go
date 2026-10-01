package main

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// This policy is owned by the checker, independently of the workflow being checked.
var requiredJobBindings = []struct{ job, env string }{
	{"quick", "QUICK_RESULT"}, {"docs", "DOCS_RESULT"}, {"security", "SECURITY_RESULT"},
	{"vulnerability", "VULNERABILITY_RESULT"}, {"minimum-go", "MINIMUM_GO_RESULT"},
	{"macos", "MACOS_RESULT"}, {"gvisor-observer", "GVISOR_OBSERVER_RESULT"},
	{"gvisor-integration", "GVISOR_INTEGRATION_RESULT"}, {"wheel-corpus", "CORPUS_RESULT"},
}

func parseWorkflow(contents string) (map[string]any, error) {
	decoder := yaml.NewDecoder(strings.NewReader(contents))
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("workflow must contain exactly one YAML document")
	}
	if document == nil {
		return nil, errors.New("empty workflow")
	}
	return document, nil
}

func workflowMap(value any) map[string]any { result, _ := value.(map[string]any); return result }

// Validate the Actions structures used by this repository before textual policy
// checks. Expressions are scalar strings, not YAML keys or evaluated host code.
func validateWorkflowStructure(contents string, ordinaryCI bool) []string {
	doc, err := parseWorkflow(contents)
	if err != nil {
		return []string{"invalid workflow YAML: " + err.Error()}
	}
	var findings []string
	bad := func(format string, args ...any) { findings = append(findings, fmt.Sprintf(format, args...)) }
	checkKeys := func(kind string, object map[string]any, allowed string) {
		for key := range object {
			if !strings.Contains(" "+allowed+" ", " "+key+" ") {
				bad("unsupported Actions %s field %s", kind, key)
			}
		}
	}
	checkKeys("workflow", doc, "name on permissions defaults concurrency jobs run-name env")
	jobs := workflowMap(doc["jobs"])
	events := workflowMap(doc["on"])
	if len(jobs) == 0 || len(events) == 0 {
		bad("Actions jobs and on must be nonempty mappings")
	}
	for event, config := range events {
		if event == "schedule" {
			if _, ok := config.([]any); !ok {
				bad("schedule must be a sequence")
			}
			continue
		}
		if config != nil && workflowMap(config) == nil {
			bad("event %s must be a mapping or null", event)
		}
	}
	for id, value := range jobs {
		job := workflowMap(value)
		if job == nil {
			bad("job %s must be a mapping", id)
			continue
		}
		checkKeys("job", job, "name env if needs runs-on steps timeout-minutes environment permissions concurrency defaults strategy outputs container services continue-on-error")
		if _, ok := job["runs-on"]; !ok {
			bad("job %s omits runs-on", id)
		}
		steps, ok := job["steps"].([]any)
		if !ok || len(steps) == 0 {
			bad("job %s requires nonempty steps", id)
		}
		for _, value := range steps {
			step := workflowMap(value)
			checkKeys("step", step, "name id env if run uses with shell working-directory timeout-minutes continue-on-error")
			_, run := step["run"].(string)
			_, uses := step["uses"].(string)
			if step == nil || run == uses {
				bad("job %s step must have exactly one string run or uses", id)
			}
		}
		if value, exists := job["needs"]; exists {
			needs, ok := value.([]any)
			if !ok {
				bad("job %s needs must be a sequence", id)
				continue
			}
			seen := map[string]bool{}
			for _, value := range needs {
				need, ok := value.(string)
				if !ok || jobs[need] == nil || need == id || seen[need] {
					bad("job %s has invalid/duplicate needs %v", id, value)
				}
				seen[need] = true
			}
		}
	}
	if !ordinaryCI {
		return findings
	}
	wantJobs := []string{"required"}
	wantNeeds := []any{}
	wantEnv := map[string]any{"GOTOOLCHAIN": "local"}
	command := "go run ./scripts/check required"
	for _, binding := range requiredJobBindings {
		wantJobs = append(wantJobs, binding.job)
		wantNeeds = append(wantNeeds, binding.job)
		wantEnv[binding.env] = "${{ needs." + binding.job + ".result }}"
		command += " \"$" + binding.env + "\""
	}
	sort.Strings(wantJobs)
	if !reflect.DeepEqual(workflowJobIDs(contents), wantJobs) {
		bad("ordinary CI requires exactly ten policy-owned jobs")
	}
	required := workflowMap(jobs["required"])
	if required["if"] != "${{ always() }}" {
		bad("Required must run with always()")
	}
	if !reflect.DeepEqual(required["needs"], wantNeeds) {
		bad("Required needs must bind exactly nine policy prerequisites in order")
	}
	if !reflect.DeepEqual(workflowMap(required["env"]), wantEnv) {
		bad("Required environment must bind each exact prerequisite result once")
	}
	if len(requiredJobBindings) != requiredResultCount {
		bad("checker aggregation policy count differs from result validator")
	}
	if !reflect.DeepEqual(workflowMap(doc["permissions"]), map[string]any{"contents": "read"}) {
		bad("ordinary CI requires contents: read only")
	}
	if !reflect.DeepEqual(workflowMap(events["push"]), map[string]any{"branches": []any{"main"}}) {
		bad("ordinary push must target main")
	}
	if events["pull_request"] != nil || events["merge_group"] != nil {
		bad("ordinary PR and merge group triggers must be unconditional")
	}
	count := 0
	steps, _ := required["steps"].([]any)
	for _, value := range steps {
		step := workflowMap(value)
		if run, ok := step["run"].(string); ok && strings.Contains(run, "./scripts/check required") && run != command {
			bad("unexpected Required aggregation invocation")
		}
		if step["run"] == command {
			count++
		}
	}
	if count != 1 {
		bad("Required must invoke the exact nine-result aggregation once")
	}
	wantEvents := []string{"merge_group", "pull_request", "push", "workflow_dispatch"}
	actualEvents := []string{}
	for event := range events {
		actualEvents = append(actualEvents, event)
	}
	sort.Strings(actualEvents)
	if !reflect.DeepEqual(actualEvents, wantEvents) {
		bad("ordinary CI events differ from policy")
	}
	dispatch := workflowMap(events["workflow_dispatch"])
	inputs := workflowMap(dispatch["inputs"])
	if len(dispatch) != 1 || len(inputs) != 3 {
		bad("dispatch requires exactly three qualification inputs")
	}
	for _, profile := range []string{"cu126", "cu130", "cu132"} {
		input := workflowMap(inputs["pytorch_"+profile+"_qualification"])
		if len(input) != 4 || input["type"] != "boolean" || input["required"] != false || input["default"] != false || input["description"] != "Run only the bounded "+profile+" full qualification." {
			bad("invalid %s dispatch input structure", profile)
		}
	}
	for id, value := range jobs {
		job := workflowMap(value)
		if _, exists := job["if"]; exists && id != "required" {
			bad("mandatory job %s cannot have a skip condition", id)
		}
		if _, exists := job["continue-on-error"]; exists {
			bad("mandatory job %s cannot ignore errors", id)
		}
		steps, _ := job["steps"].([]any)
		for _, value := range steps {
			step := workflowMap(value)
			if _, exists := step["if"]; exists && id == "required" {
				bad("Required steps cannot skip aggregation")
			}
			if _, exists := step["continue-on-error"]; exists {
				bad("job %s step cannot ignore errors", id)
			}
		}
	}
	corpus := workflowMap(jobs["wheel-corpus"])
	steps, _ = corpus["steps"].([]any)
	preparation, execution := -1, -1
	for i, value := range steps {
		step := workflowMap(value)
		switch step["run"] {
		case `python3 scripts/prepare-wheel-corpus.py --root "$HELOX_CORPUS_ROOT"`:
			preparation = i
		case "go run ./scripts/check corpus":
			execution = i
		}
		if step["if"] != nil {
			bad("corpus steps must be unconditional")
		}
	}
	if preparation < 0 || execution <= preparation {
		bad("corpus preparation must precede mandatory offline validation")
	}
	return findings
}
