//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	evidencerecord "github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

// Actual-only composition; the production CLI remains a separate work item.
func pipxToolActualUpdateService(b pythonClientBindings, native *sandbox.PipxResolver, bin string) (*application.InstallUpdateService, error) {
	return newPipxToolUpdateService(b, native, bin)
}

func TestPipxPublicToolApprovedUpgradeIntegration(t *testing.T) {
	if os.Getenv("HELOX_PIPX_TOOL_UPDATE_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user managed pipx current upgrade qualification")
	}
	ctx, bindings, root, interpreter, native := toolApprovedIntegrationFixture(t, true)
	venv, bin := filepath.Join(root, "managed tool", "cowsay"), filepath.Join(root, "commands")
	install := prepareUVToolTargetFixture(t, venv, bin, interpreter)
	note := filepath.Join(venv, "user-note.txt")
	if err := os.WriteFile(note, []byte("keep user data"), 0600); err != nil {
		t.Fatal(err)
	}
	fresh, err := newPipxToolInstallService(bindings, native, bin)
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
		t.Fatal("genuine current managed pipx6.0 preparation", err)
	}
	path := filepath.Join(venv, "pipx_metadata.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := artifactpypi.ParsePipxToolReceipt(body)
	if err != nil || initial.Version() != "6.0" || initial.Requirement().Requirement() != "cowsay==6.0" {
		t.Fatal("authentic original receipt/constraint absent", err)
	}
	service, err := pipxToolActualUpdateService(bindings, native, bin)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, entry := range previous.Bundle().Set().Inspected().Inspections() {
		seen[entry.RunID().String()] = true
	}
	for _, step := range []struct{ name, oldVersion, oldRequirement string }{{"changed-version", "6.0", "cowsay==6.0"}, {"same-version-current-approval", "6.1", "cowsay"}} {
		t.Run(step.name, func(t *testing.T) {
			request, err := application.NewInstallUpdateRequest(artifactpypi.PublicPyPIProfile().Source(), "cowsay", install, false)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := service.Install(ctx, request)
			bundle := outcome.Bundle()
			if err != nil || outcome.FailureCode() != "" || !bundle.Valid() || outcome.Inspection().Decision().Decision() != domain.DecisionAllow || outcome.Promoted().ManifestID() != bundle.ManifestID() || len(bundle.Set().Inspected().Inspections()) != 1 {
				t.Fatal("current pipx whole approval/publication missing", err, outcome.FailureCode())
			}
			for _, entry := range bundle.Set().Inspected().Inspections() {
				if entry.Artifact().Identity().Version() != "6.1" || seen[entry.RunID().String()] || len(entry.Evidence()) < 3 {
					t.Fatal("old or incomplete current inspection reused")
				}
				seen[entry.RunID().String()] = true
				for _, check := range entry.Checks() {
					if check.Required() && check.Status() != domain.ExecutionCompleted {
						t.Fatal("required current check incomplete")
					}
				}
				for _, ref := range entry.Evidence() {
					if _, _, err := bindings.evidence.ReadReference(ctx, entry.RunID(), ref, entry.Artifact().Identity(), entry.Artifact().Digest()); err != nil {
						t.Fatal("current Evidence readback", err)
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
			if json.Unmarshal(bundle.ManifestDocument(), &manifest) != nil || len(manifest.Controls) != 9 || !strings.HasPrefix(manifest.Resolver.Runtime, "pipx-tool-upgrade:pipx:") {
				t.Fatal("pipx current native7/Host2 provenance missing")
			}
			published, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := artifactpypi.ParsePipxToolReceipt(published)
			if err != nil || receipt.Version() != "6.1" || receipt.Requirement().Requirement() != "cowsay" {
				t.Fatal("new bare upgrade differs", err)
			}
			if err := receipt.ValidateExposedCommands(venv, bin, filepath.Join(venv, "bin/python"), map[string]string{filepath.Join(bin, "cowsay"): filepath.Join(venv, "bin/cowsay")}); err != nil {
				t.Fatal(err)
			}
			bound, originalBound := false, false
			for _, c := range manifest.Controls {
				if c.Name == "pipx-tool-host-receipt.json" {
					bound = bytes.Equal(c.Content, published)
				}
				if c.Name == artifactpypi.PipxToolUpgradeInputControlName {
					control, err := domain.NewProjectControlFile(c.Name, c.Content, true)
					if err != nil {
						t.Fatal(err)
					}
					input, err := artifactpypi.ParsePipxToolUpgradeInput([]domain.ProjectControlFile{control}, sandbox.PinnedPythonRuntime().PythonVersion)
					if err != nil || input.Receipt().Version() != step.oldVersion || input.Receipt().Requirement().Requirement() != step.oldRequirement || input.Request().Requirement() != "cowsay" {
						t.Fatal("old and current native intent conflated", err)
					}
					originalBound = true
				}
			}
			if !bound || !originalBound {
				t.Fatal("current approved receipt/original controls missing")
			}
			metadata, err := os.ReadFile(filepath.Join(venv, "lib/python3.14/site-packages/cowsay-6.1.dist-info/METADATA"))
			if err != nil || !bytes.Contains(metadata, []byte("Version: 6.1")) {
				t.Fatal("selected installed version missing", err)
			}
			if data, err := os.ReadFile(note); err != nil || string(data) != "keep user data" {
				t.Fatal("user data changed")
			}
			t.Logf("pipx_tool_current_upgrade case=%s original_version=%s original_requirement=%s selected=6.1 controls=9 current_runs=%d manifest=%s", step.name, step.oldVersion, step.oldRequirement, len(seen), bundle.ManifestID().String())
		})
	}
	// A real current native no-op still enters current inspection. Old ALLOW and
	// cached wheel bytes cannot replace a failed current Evidence write.
	before := uvToolTargetState(t, venv, bin)
	primary := errors.New("current pipx Evidence storage unavailable")
	inspection, err := application.NewInstallInspectService(bindings.resolver, bindings.intake, verificationpypi.IntegrityVerifier{}, bindings.inspector, uvToolUnavailableEvidence{primary}, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	staging, err := promotion.NewLocalStaging(bindings.intakeRoot, bindings.evidenceRoot, bindings.stagingRoot)
	if err != nil {
		t.Fatal(err)
	}
	base, err := application.NewInstallService(inspection, evidencerecord.Generator{}, staging, bindings.promoter)
	if err != nil {
		t.Fatal(err)
	}
	bindings.installer = base
	failing, err := pipxToolActualUpdateService(bindings, native, bin)
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewInstallUpdateRequest(artifactpypi.PublicPyPIProfile().Source(), "cowsay", install, false)
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := failing.Install(ctx, request)
	if !errors.Is(err, primary) || rejected.Bundle().Valid() || rejected.Promoted().ManifestID().String() != "" {
		t.Fatal("prior approval bypassed failed current Evidence", err)
	}
	if !reflect.DeepEqual(before, uvToolTargetState(t, venv, bin)) {
		t.Fatal("failed current approval changed old pipx state")
	}
	t.Log("pipx_tool_noop_current_evidence_failure refused=true original_inodes_bytes_modes_links_unchanged=true; no native adoption/CLI/Host interpreter execution")
}
