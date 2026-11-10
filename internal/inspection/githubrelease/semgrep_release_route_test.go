package githubrelease

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestSemgrepSelectedRulesUsePassiveChecksAndRejectChangedIntake(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0", "helox-release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	assets := value["assets"].([]any)
	for name, role := range map[string]string{"haa-release-installer-linux-amd64": "release-installer", "pnpm-linux-amd64": "pnpm", "pnpm-NOTICE.txt": "release-metadata", "bun-linux-amd64": "bun", "bun-NOTICE.txt": "release-metadata", runtimeidentity.PipxRuntimeAssetName: "pipx-runtime", "pipx-NOTICE.txt": "release-metadata", runtimeidentity.SandboxResponderAssetName: "sandbox-responder"} {
		assets = append(assets, map[string]any{"name": name, "platform": "linux/amd64", "role": role, "sha256": strings.Repeat("a", 64), "size": 123})
	}
	pin := runtimeidentity.SelectedSemgrepSupply()
	for _, asset := range runtimeidentity.SemgrepReleaseAssets(pin) {
		role := "semgrep-data"
		if asset.Path == pin.Tool.Path {
			role = "semgrep"
		}
		assets = append(assets, map[string]any{"name": asset.Path, "platform": "linux/amd64", "role": role, "sha256": asset.SHA256, "size": asset.Size})
	}
	value["assets"] = assets
	body, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := releaseinstall.ParseReleaseManifest(body)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := os.ReadFile(filepath.Join("..", "..", "..", pin.Rules.Path))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	native := &pipxNativeRouteSpy{}
	inspector, err := NewReleaseSetInspector(root, func() (releaseinstall.ReleaseManifest, error) { return manifest, nil }, native, nil)
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("github-release")
	id, _ := domain.NewResolvedArtifactIdentity(source, "rahoney-heliopause", manifest.Version(), runtimeidentity.SemgrepRulesAssetName)
	digest, _ := domain.NewSHA256Digest(pin.Rules.SHA256)
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, run.String())
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "asset")
	if err := os.WriteFile(path, rules, 0600); err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(id, digest, "intake:"+run.String()+":github-release", uint64(len(rules)), "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	requirements, err := inspector.ExpectedChecks(context.Background(), artifact, domain.CheckInspection)
	if err != nil || len(requirements) != 1 {
		t.Fatal("missing passive declaration", err)
	}
	report, err := inspector.Inspect(context.Background(), artifact)
	if err != nil || native.calls != 0 || len(report.Findings()) != 0 || report.Execution().ID().String() != "github-release-static" || report.Execution().Status() != domain.ExecutionCompleted {
		t.Fatal("passive supply routed incorrectly", err)
	}
	rules[0] ^= 1
	if err := os.WriteFile(path, rules, 0600); err != nil {
		t.Fatal(err)
	}
	report, err = inspector.Inspect(context.Background(), artifact)
	if err != nil || native.calls != 0 || len(report.Findings()) != 1 || report.Findings()[0].Code() != "SELF_UPDATE_METADATA_INVALID" {
		t.Fatal("changed passive bytes accepted", err)
	}
}
