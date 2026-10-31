package releaseprovenance

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func actualPipxDependency(t *testing.T, project string) ([]byte, PipxDependencyPublishIdentity) {
	t.Helper()
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("testdata", "pipx-dependencies", project, name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	var identity PipxDependencyPublishIdentity
	if err := json.Unmarshal(read("identity.json"), &identity); err != nil {
		t.Fatal(err)
	}
	return read("provenance.json"), identity
}

func TestPipxDependencyOfficialPublishPolicies(t *testing.T) {
	for _, project := range []string{"argcomplete", "click", "filelock", "packaging", "platformdirs", "pytest", "iniconfig", "pluggy"} {
		t.Run(project, func(t *testing.T) {
			body, id := actualPipxDependency(t, project)
			receipt, err := VerifyPipxDependencyPublish(context.Background(), body, id)
			policy, _ := pipxDependencyPolicy(id.Project, id.Version)
			if err != nil || receipt.Identity() != id || receipt.ProvenanceSHA256() != uvDigest(body) || !strings.HasPrefix(receipt.Invocation(), policy.repository+"/actions/runs/"+id.WorkflowRun+"/attempts/") {
				t.Fatalf("official dependency proof differs: %+v %v", receipt, err)
			}
			for _, field := range []string{"source", "run", "digest", "project", "version"} {
				wrong := id
				switch field {
				case "source":
					wrong.SourceCommit = strings.Repeat("a", 40)
				case "run":
					wrong.WorkflowRun = "1"
				case "digest":
					wrong.WheelSHA256 = strings.Repeat("b", 64)
				case "project":
					wrong.Project = "userpath"
				case "version":
					wrong.Version = "999.0.0"
				}
				if r, err := VerifyPipxDependencyPublish(context.Background(), body, wrong); err == nil || r != (PipxDependencyPublishReceipt{}) {
					t.Fatalf("wrong %s accepted", field)
				}
			}
			for _, mode := range []string{"publisher", "environment", "signature", "payload"} {
				changed := pipxProofMutation(t, body, func(o, g, a map[string]any) {
					switch mode {
					case "publisher":
						g["publisher"].(map[string]any)["repository"] = "pypa/pipx"
					case "environment":
						g["publisher"].(map[string]any)["environment"] = "other"
					case "signature":
						a["envelope"].(map[string]any)["signature"] = "AAAA"
					case "payload":
						e := a["envelope"].(map[string]any)
						p, err := base64.StdEncoding.DecodeString(e["statement"].(string))
						if err != nil {
							t.Fatal(err)
						}
						e["statement"] = base64.StdEncoding.EncodeToString(append(p, ' '))
					}
				})
				if r, err := VerifyPipxDependencyPublish(context.Background(), changed, id); err == nil || r != (PipxDependencyPublishReceipt{}) {
					t.Fatalf("tampered %s accepted", mode)
				}
			}
			_, mainID := actualPipxPublish(t)
			if _, err := VerifyPipxPublish(context.Background(), body, mainID); err == nil {
				t.Fatal("dependency gained main publisher authority")
			}
			other := "click"
			if project == other {
				other = "argcomplete"
			}
			otherBody, _ := actualPipxDependency(t, other)
			if _, err := VerifyPipxDependencyPublish(context.Background(), otherBody, id); err == nil {
				t.Fatal("different dependency publisher accepted")
			}
		})
	}
	policy, _ := pipxDependencyPolicy("packaging", "26.3")
	if policy.ref != "refs/heads/main" || policy.trigger != "workflow_dispatch" {
		t.Fatal("actual packaging publisher replaced with a synthetic tag")
	}
}

func TestPipxDependencyIdentityStrictSubset(t *testing.T) {
	_, id := actualPipxDependency(t, "packaging")
	body, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{`"project"`, `"Project"`}, {`"version":`, `"version":"1.0","version":`}, {`"packaging"`, `"userpath"`}, {`"26.3"`, `"26.3+build"`}, {`"26.3"`, `"26.3-rc1"`}, {`"26.3"`, `"../escape"`}, {`"wheel_sha256"`, `"unknown"`}} {
		var value PipxDependencyPublishIdentity
		if json.Unmarshal([]byte(strings.Replace(string(body), pair[0], pair[1], 1)), &value) == nil {
			t.Fatal("unsupported dependency identity accepted")
		}
	}
	if (PipxDependencyPublishIdentity{}).WheelFilename() != "" {
		t.Fatal("invalid identity generated a path")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	proof, _ := actualPipxDependency(t, "packaging")
	if _, err := VerifyPipxDependencyPublish(ctx, proof, id); err == nil {
		t.Fatal("cancelled dependency verification accepted")
	}
}
