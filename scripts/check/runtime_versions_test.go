package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runtimeOwnerFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{"cmd", "internal", "scripts"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"go.mod", "README.md", "scripts/runtimes.lock.json", "scripts/tools.lock.json", "scripts/version-support.lock.json", workflowRelativePath, securityWorkflowRelativePath, releaseWorkflowRelativePath, releasePublishWorkflowRelativePath} {
		body, err := os.ReadFile(filepath.Join("../..", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(root, name), string(body))
	}
	return root
}

func mutateVersionJSON(t *testing.T, root, name string, mutate func(map[string]any)) {
	t.Helper()
	p := filepath.Join(root, "scripts", name)
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatal(err)
	}
	mutate(object)
	body, err = json.MarshalIndent(object, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, p, string(body)+"\n")
}

func checkOwnerFixture(root string) error {
	tools, err := readToolLock(filepath.Join(root, "scripts", "tools.lock.json"))
	if err != nil {
		return err
	}
	c := checker{root: root, toolLock: tools}
	if err := c.checkGoVersionConsistency(); err != nil {
		return err
	}
	return checkRuntimeVersionOwners(root, tools)
}

func TestRuntimeVersionOwnersRejectPartialOrUnownedUpdates(t *testing.T) {
	t.Parallel()
	for _, component := range []string{"Go", "Node", "npm", "uv", "Python", "pip", "Rust/Cargo", "Docker", "containerd", "Bazel", "gVisor", "Staticcheck", "gosec", "govulncheck", "Gitleaks", "Actionlint"} {
		t.Run(component, func(t *testing.T) {
			t.Parallel()
			root := runtimeOwnerFixture(t)
			if err := checkOwnerFixture(root); err != nil {
				t.Fatal(err)
			}
			mutateVersionJSON(t, root, "version-support.lock.json", func(value map[string]any) {
				for _, raw := range value["records"].([]any) {
					record := raw.(map[string]any)
					if record["component"] == component {
						record["pinned_version"] = "99.0.0"
					}
				}
			})
			if err := checkOwnerFixture(root); err == nil {
				t.Fatal("partial reviewed version update accepted")
			}
		})
	}
	for _, kind := range []string{"missing", "duplicate", "unknown"} {
		t.Run(kind, func(t *testing.T) {
			root := runtimeOwnerFixture(t)
			mutateVersionJSON(t, root, "version-support.lock.json", func(value map[string]any) {
				records := value["records"].([]any)
				switch kind {
				case "missing":
					value["records"] = records[1:]
				case "duplicate":
					value["records"] = append(records, records[0])
				case "unknown":
					records[0].(map[string]any)["component"] = "UnownedRuntime"
				}
			})
			if err := checkOwnerFixture(root); err == nil {
				t.Fatal("unowned/missing/duplicate support record accepted")
			}
		})
	}
}

func TestRuntimeVersionOwnersAllowCoordinatedFutureGoInputs(t *testing.T) {
	root := runtimeOwnerFixture(t)
	selected := selectedGoForTest(t)
	future := "1.99.1" // Logical fixture; not a claimed available SDK or image.
	mutateVersionJSON(t, root, "runtimes.lock.json", func(value map[string]any) {
		goImage := value["go_image"].(map[string]any)
		goImage["go_version"] = future
		goImage["reference"] = strings.Replace(goImage["reference"].(string), selected, future, 1)
	})
	mutateVersionJSON(t, root, "tools.lock.json", func(value map[string]any) {
		for _, raw := range value["tools"].([]any) {
			raw.(map[string]any)["setupGo"] = future
		}
	})
	mutateVersionJSON(t, root, "version-support.lock.json", func(value map[string]any) {
		for _, raw := range value["records"].([]any) {
			if raw.(map[string]any)["component"] == "Go" {
				raw.(map[string]any)["pinned_version"] = future
			}
		}
	})
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.test/fixture\n\ngo "+future+"\n")
	writeTestFile(t, filepath.Join(root, "README.md"), fmt.Sprintf("# Fixture\n\n## Build\n\nHeliopause는 Go `%s` 이상이 필요하며, 개발·CI toolchain은 Go `%s`으로 고정한다.\n", future, future))
	for _, path := range []string{workflowRelativePath, securityWorkflowRelativePath, releaseWorkflowRelativePath, releasePublishWorkflowRelativePath} {
		body, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(root, path), strings.ReplaceAll(string(body), "go-version: '"+selected+"'", "go-version: '"+future+"'"))
	}
	if err := checkOwnerFixture(root); err != nil {
		t.Fatal(err)
	}
	if err := checkCIWorkflow(root); err != nil {
		t.Fatal(err)
	}
	// One retained old consumer must fail even though other jobs are correct.
	p := filepath.Join(root, workflowRelativePath)
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, p, strings.Replace(string(body), "go-version: '"+future+"'", "go-version: '"+selected+"'", 1))
	if err := checkCIWorkflow(root); err == nil {
		t.Fatal("single old setup consumer accepted")
	}
}

func TestRuntimeVersionOwnersRejectNewExecutableImageConsumer(t *testing.T) {
	root := runtimeOwnerFixture(t)
	path := filepath.Join(root, "internal", "newconsumer", "runtime.go")
	writeTestFile(t, path, "package newconsumer\nconst image = \"node:24.21.0-slim@sha256:"+strings.Repeat("a", 64)+"\"\n")
	if err := checkOwnerFixture(root); err == nil || !strings.Contains(err.Error(), "unowned image literal") {
		t.Fatalf("unexpected result: %v", err)
	}
	if err := os.Rename(path, strings.TrimSuffix(path, ".go")+"_test.go"); err != nil {
		t.Fatal(err)
	}
	if err := checkOwnerFixture(root); err != nil {
		t.Fatal("historical test fixture treated as executable image owner:", err)
	}
	writeTestFile(t, filepath.Join(root, ".github/workflows/unowned.yml"), "name: unowned\n")
	if err := checkCIWorkflow(root); err == nil || !strings.Contains(err.Error(), "unowned workflow") {
		t.Fatalf("unexpected result: %v", err)
	}
}

func TestGoSetupCannotUseFloatingOrAlternateSelectors(t *testing.T) {
	root := runtimeOwnerFixture(t)
	body, err := os.ReadFile(filepath.Join(root, workflowRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	selected := selectedGoForTest(t)
	for _, replacement := range []string{"go-version: 'latest'", "go-version-file: 'go.mod'", "go-version: '${{ env.GO_VERSION }}'", "go-version: '" + selected + "'\n          go-version-file: 'go.mod'"} {
		fixture := strings.Replace(string(body), "go-version: '"+selected+"'", replacement, 1)
		if len(validateGoWorkflowVersions(fixture, selected)) == 0 {
			t.Fatal("alternate/floating selector accepted")
		}
	}
}
