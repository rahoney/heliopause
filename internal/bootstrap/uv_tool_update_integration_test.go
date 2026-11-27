//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestUVPublicToolApprovedUpgradeIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_UPDATE_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user managed UV tool upgrade qualification")
	}
	ctx, bindings, root, interpreter := uvToolApprovedIntegrationFixture(t)
	venv, bin := filepath.Join(root, "managed tool", "cowsay"), filepath.Join(root, "commands")
	install := prepareUVToolTargetFixture(t, venv, bin, interpreter)
	note := filepath.Join(venv, "user-note.txt")
	if err := os.WriteFile(note, []byte("keep user data"), 0600); err != nil {
		t.Fatal(err)
	}
	fresh, err := newUVToolInstallService(bindings, bin)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := domain.NewArtifactReference(artifactpypi.PublicPyPIProfile().Source(), "cowsay==6.0")
	seed, err := application.NewInstallRequest(reference, install)
	if err != nil {
		t.Fatal(err)
	}
	previous, err := fresh.Install(ctx, seed)
	if err != nil || !previous.Bundle().Valid() || previous.Inspection().Decision().Decision() != domain.DecisionAllow {
		t.Fatal("authentic managed old tool preparation", err)
	}
	// Only this historical receipt requirement is synthetic: authentic current
	// HAA installation owns the real public6.0 files. A native exact6.0 pin would
	// remain pinned; represent a prior unversioned installation explicitly.
	receiptPath := filepath.Join(venv, "uv-receipt.toml")
	body, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := toml.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	wire["tool"].(map[string]any)["requirements"] = []map[string]any{{"name": "cowsay"}}
	body, err = toml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifactpypi.ParseUVToolReceipt(body); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("uv_tool_original_requirement=synthetic_unversioned; old_files=authenticated_current_HAA_public6.0; no_native_adoption")
	service, err := newUVToolUpdateService(bindings, bin)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, entry := range previous.Bundle().Set().Inspected().Inspections() {
		seen[entry.RunID().String()] = true
	}
	for _, step := range []string{"changed-version", "same-version-current-approval", "managed-explicit-adopt-current-approval"} {
		t.Run(step, func(t *testing.T) {
			request, err := application.NewInstallUpdateRequest(artifactpypi.PublicPyPIProfile().Source(), "cowsay==6.1", install, step == "managed-explicit-adopt-current-approval")
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := service.Install(ctx, request)
			bundle := outcome.Bundle()
			if err != nil || outcome.FailureCode() != "" || !bundle.Valid() || len(bundle.Set().Inspected().Inspections()) != 1 || outcome.Inspection().Decision().Decision() != domain.DecisionAllow || outcome.Promoted().ManifestID() != bundle.ManifestID() {
				t.Fatal("current whole tool update/approval missing", err, outcome.FailureCode())
			}
			for _, entry := range bundle.Set().Inspected().Inspections() {
				if entry.Artifact().Identity().Version() != "6.1" || seen[entry.RunID().String()] || len(entry.Evidence()) < 3 {
					t.Fatal("old or incomplete inspection reused")
				}
				seen[entry.RunID().String()] = true
				for _, check := range entry.Checks() {
					if check.Required() && check.Status() != domain.ExecutionCompleted {
						t.Fatal("required current check incomplete")
					}
				}
				for _, ref := range entry.Evidence() {
					if _, _, err := bindings.evidence.ReadReference(ctx, entry.RunID(), ref, entry.Artifact().Identity(), entry.Artifact().Digest()); err != nil {
						t.Fatal(err)
					}
				}
			}
			var manifest struct {
				Resolver struct {
					Runtime string `json:"runtime_identity"`
				} `json:"resolver"`
				Controls []struct {
					Name    string
					Content []byte
				} `json:"project_controls"`
			}
			if json.Unmarshal(bundle.ManifestDocument(), &manifest) != nil || len(manifest.Controls) != 8 || !strings.HasPrefix(manifest.Resolver.Runtime, "uv-tool-upgrade:uv:") {
				t.Fatal("complete upgrade provenance missing")
			}
			published, err := os.ReadFile(receiptPath)
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := artifactpypi.ParseUVToolReceipt(published)
			if err != nil || receipt.Requirement().Requirement() != "cowsay" {
				t.Fatal("native original constraint lost", err)
			}
			if err := receipt.ValidateExposedCommands(venv, bin, filepath.Join(venv, "bin/python"), map[string]string{filepath.Join(bin, "cowsay"): filepath.Join(venv, "bin/cowsay")}); err != nil {
				t.Fatal(err)
			}
			bound := false
			for _, c := range manifest.Controls {
				if c.Name == "uv-tool-host-receipt.toml" {
					bound = bytes.Equal(c.Content, published)
				}
			}
			if !bound {
				t.Fatal("Host receipt not currently approved")
			}
			metadata, err := os.ReadFile(filepath.Join(venv, "lib/python3.14/site-packages/cowsay-6.1.dist-info/METADATA"))
			if err != nil || !bytes.Contains(metadata, []byte("Version: 6.1")) {
				t.Fatal("selected installed version absent", err)
			}
			if data, err := os.ReadFile(note); err != nil || string(data) != "keep user data" {
				t.Fatal("user data changed")
			}
			t.Logf("uv_tool_current_upgrade case=%s selected=6.1 controls=8 current_runs=%d", step, len(seen))
		})
	}
}
