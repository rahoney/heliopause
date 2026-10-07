package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

// Exercise the public metadata adapter through the actual composed CLI. npm
// install uses a different resolver and cannot qualify this inspect boundary.
func TestLinuxNPMInspectCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_NPM_INSPECT_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("npm qualification requires Linux amd64")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var stdout, stderr bytes.Buffer
	if err := bootstrap.Run(ctx, []string{"npm", "inspect", "is-number@7.0.0"}, &stdout, &stderr); err != nil {
		t.Fatalf("actual npm inspect failed: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	var result struct {
		Schema   string `json:"schema_version"`
		Status   string `json:"operation_status"`
		Artifact struct {
			Identity struct {
				SourceID      string `json:"source_id"`
				Name, Version string
			} `json:"resolved_identity"`
		} `json:"artifact"`
		Policy struct{ Decision string } `json:"policy"`
		Checks []struct {
			Required bool
			Status   string `json:"execution_status"`
		} `json:"checks"`
		Evidence []struct{ ID, Handle string } `json:"evidence_references"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Schema != "helox.operation-result/v1" || result.Status != "COMPLETED" || result.Policy.Decision != "ALLOW" || result.Artifact.Identity.SourceID != "npm" || result.Artifact.Identity.Name != "is-number" || result.Artifact.Identity.Version != "7.0.0" || len(result.Checks) < 3 || len(result.Evidence) < 3 {
		t.Fatalf("incomplete inspect result: %s", stdout.String())
	}
	for _, check := range result.Checks {
		if check.Required && check.Status != "COMPLETED" {
			t.Fatalf("required check is incomplete: %s", stdout.String())
		}
	}
	files, err := filepath.Glob(filepath.Join(root, "cache", "heliopause", "evidence", "*", "*.json"))
	if err != nil || len(files) != len(result.Evidence) {
		t.Fatal("inspect did not retain its recorded Evidence")
	}
	t.Logf("actual_npm_inspect_result=%s", stdout.String())
}
