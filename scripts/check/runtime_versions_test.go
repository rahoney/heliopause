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
	for _, name := range []string{"go.mod", "README.md", "tools/pnpm/release-proof.json", "tools/pnpm/NOTICE", "tools/bun/SHASUMS256.txt.asc", "tools/bun/NOTICE", "scripts/ci-pnpm-runtime.sh", "scripts/ci-bun-runtime.sh", "scripts/ci-pipx-runtime.sh", "scripts/ci-uv-tools.sh", "scripts/runtimes.lock.json", "scripts/tools.lock.json", "scripts/version-support.lock.json", workflowRelativePath, securityWorkflowRelativePath, releaseWorkflowRelativePath, releasePublishWorkflowRelativePath} {
		body, err := os.ReadFile(filepath.Join("../..", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(root, name), string(body))
	}
	var pipxFixture struct {
		Pipx struct {
			Inputs []struct {
				Path string `json:"path"`
			} `json:"inputs"`
		} `json:"pipx_tool"`
	}
	lockBody, err := os.ReadFile(filepath.Join("../..", "scripts/runtimes.lock.json"))
	if err != nil || json.Unmarshal(lockBody, &pipxFixture) != nil {
		t.Fatal("pipx fixture owner unavailable", err)
	}
	names := []string{"tools/pipx/NOTICE"}
	for _, input := range pipxFixture.Pipx.Inputs {
		if !strings.HasSuffix(input.Path, ".whl") {
			names = append(names, "tools/pipx/supply/"+input.Path)
		}
	}
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join("../..", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(root, name), string(body))
	}
	return root
}

func TestPNPMProductionProofCannotDrift(t *testing.T) {
	for _, name := range []string{"missing", "changed", "symlink", "owner-path"} {
		t.Run(name, func(t *testing.T) {
			root := runtimeOwnerFixture(t)
			proof := filepath.Join(root, "tools", "pnpm", "release-proof.json")
			switch name {
			case "missing":
				if err := os.Remove(proof); err != nil {
					t.Fatal(err)
				}
			case "changed":
				if err := os.WriteFile(proof, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				body, err := os.ReadFile(proof)
				if err != nil {
					t.Fatal(err)
				}
				other := filepath.Join(t.TempDir(), "other.json")
				if err := os.WriteFile(other, body, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(proof); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, proof); err != nil {
					t.Fatal(err)
				}
			case "owner-path":
				mutateVersionJSON(t, root, "runtimes.lock.json", func(l map[string]any) {
					l["pnpm_tool"].(map[string]any)["provenance"].(map[string]any)["bundle_path"] = "testdata/bundle.json"
				})
			}
			if err := checkOwnerFixture(root); err == nil {
				t.Fatal("unowned production proof accepted")
			}
		})
	}
}

func TestPNPMProductionNoticesCannotDrift(t *testing.T) {
	for _, mode := range []string{"missing", "changed", "symlink", "size", "source"} {
		t.Run(mode, func(t *testing.T) {
			root := runtimeOwnerFixture(t)
			path := filepath.Join(root, "tools/pnpm/NOTICE")
			var lock map[string]any
			body, err := os.ReadFile(filepath.Join(root, "scripts/runtimes.lock.json"))
			if err != nil || json.Unmarshal(body, &lock) != nil {
				t.Fatal("runtime fixture unavailable")
			}
			pin := lock["pnpm_tool"].(map[string]any)
			switch mode {
			case "missing":
				err = os.Remove(path)
			case "changed":
				err = os.WriteFile(path, []byte("changed notices"), 0600)
			case "symlink":
				other := filepath.Join(t.TempDir(), "foreign")
				if err = os.WriteFile(other, []byte("foreign"), 0600); err != nil {
					t.Fatal(err)
				}
				if err = os.Remove(path); err != nil {
					t.Fatal(err)
				}
				err = os.Symlink(other, path)
			case "size":
				pin["notices"].(map[string]any)["size"] = float64(1)
			case "source":
				pin["provenance"].(map[string]any)["source_commit"] = strings.Repeat("b", 40)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := checkPNPMProofFile(root, lock); err == nil {
				t.Fatal("unowned notices accepted")
			}
		})
	}
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
	for _, component := range []string{"Go", "Node", "npm", "uv", "pnpm", "Python", "pip", "Rust/Cargo", "Docker", "containerd", "Bazel", "gVisor", "Staticcheck", "gosec", "govulncheck", "Gitleaks", "Actionlint"} {
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

func TestBunProductionProofCannotDrift(t *testing.T) {
	for _, name := range []string{"missing", "changed", "symlink", "owner-path"} {
		t.Run(name, func(t *testing.T) {
			root := runtimeOwnerFixture(t)
			proof := filepath.Join(root, "tools", "bun", "SHASUMS256.txt.asc")
			switch name {
			case "missing":
				if err := os.Remove(proof); err != nil {
					t.Fatal(err)
				}
			case "changed":
				if err := os.WriteFile(proof, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				body, err := os.ReadFile(proof)
				if err != nil {
					t.Fatal(err)
				}
				other := filepath.Join(t.TempDir(), "other.json")
				if err := os.WriteFile(other, body, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(proof); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, proof); err != nil {
					t.Fatal(err)
				}
			case "owner-path":
				mutateVersionJSON(t, root, "runtimes.lock.json", func(l map[string]any) {
					l["bun_tool"].(map[string]any)["publisher"].(map[string]any)["checksums_path"] = "testdata/bundle.json"
				})
			}
			if err := checkOwnerFixture(root); err == nil {
				t.Fatal("unowned production proof accepted")
			}
		})
	}
}
