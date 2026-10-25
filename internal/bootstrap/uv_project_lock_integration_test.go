//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func qualifyUVProjectLockOnly(t *testing.T, ctx context.Context, b uvProjectBindings, root string) {
	t.Helper()
	project := filepath.Join(root, "lock-only-project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.uv.lock")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(manifest, []byte("dependencies = []")) {
		t.Fatal("fixed empty original manifest changed")
	}
	manifest = bytes.Replace(manifest, []byte("dependencies = []"), []byte("dependencies = ['packaging==25.0']"), 1)
	manifestPath := filepath.Join(project, "pyproject.toml")
	if err := os.WriteFile(manifestPath, manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "uv.lock"), lock, 0o600); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(project, "user-note")
	if err := os.WriteFile(note, []byte("preserved user file"), 0o600); err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(project)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewPythonProjectControlContext(target)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for index, count := range []int{1, 1, 0} {
		if index == 1 {
			if err := os.Mkdir(filepath.Join(project, ".venv"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(project, ".venv", "foreign-user-data"), []byte("not a venv"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if count == 0 {
			manifest = bytes.Replace(manifest, []byte("dependencies = ['packaging==25.0']"), []byte("dependencies = []"), 1)
			if err := os.WriteFile(manifestPath, manifest, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		original, err := os.Lstat(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		service, err := newUVProjectLockService(b, project)
		if err != nil {
			t.Fatal(err)
		}
		approved, err := service.Apply(ctx, install)
		if err != nil {
			t.Fatal("actual control-only project lock publication", err)
		}
		if !approved.Valid() || approved.Decision().Decision() != domain.DecisionAllow || approved.Inspected().Snapshot().Context() != install || len(approved.Inspected().Inspections()) != count || approved.Inspected().Snapshot().DependencyFree() != (count == 0) {
			t.Fatal("control-only approval changed context or completeness")
		}
		for _, entry := range approved.Inspected().Inspections() {
			if seen[entry.RunID().String()] || len(entry.Evidence()) < 3 {
				t.Fatal("control-only update skipped fresh complete inspection")
			}
			seen[entry.RunID().String()] = true
			completed := 0
			for _, check := range entry.Checks() {
				if check.Required() {
					if check.Status() != domain.ExecutionCompleted {
						t.Fatal("control-only required check incomplete")
					}
					completed++
				}
			}
			if completed < 3 {
				t.Fatal("control-only current required coverage missing")
			}
		}
		id, handle, sbom, ok := service.promoter.PublishedRecords()
		cacheID, prefixOK := strings.CutPrefix(strings.TrimSuffix(handle, ":manifest.json"), "project-cache:")
		if !ok || !prefixOK || id.String() == "" || sbom != "project-cache:"+cacheID+":sbom.cdx.json" {
			t.Fatal("control-only completed records missing")
		}
		if _, err := domain.ParseRunID(cacheID); err != nil {
			t.Fatal(err)
		}
		payload := filepath.Join(b.cacheRoot, cacheID, "payload")
		body, err := os.ReadFile(filepath.Join(payload, "manifest.json"))
		var records struct {
			Manifest string                `json:"manifest_id"`
			Target   struct{ Path string } `json:"target"`
			Controls []struct {
				Name    string
				Content []byte
			} `json:"project_controls"`
		}
		if err != nil || json.Unmarshal(body, &records) != nil || records.Manifest != id.String() || records.Target.Path != project || len(records.Controls) != 6 {
			t.Fatal("control-only provenance incomplete", err)
		}
		for _, control := range records.Controls {
			if control.Name == "pyproject.toml" || control.Name == "uv.lock" {
				current, err := os.ReadFile(filepath.Join(project, control.Name))
				if err != nil || !bytes.Equal(current, control.Content) {
					t.Fatal("control-only approved bytes differ from project", err)
				}
			}
		}
		body, err = os.ReadFile(filepath.Join(b.cacheRoot, cacheID, "receipt.json"))
		var receipt struct{ Mode, Target, Graph string }
		if err != nil || json.Unmarshal(body, &receipt) != nil || receipt.Mode != string(domain.InstallPythonProjectControls) || receipt.Target != project || receipt.Graph != approved.Inspected().Snapshot().GraphDigest().String() {
			t.Fatal("approval receipt lost distinct control-only context", err)
		}
		current, err := os.Lstat(manifestPath)
		currentBody, readErr := os.ReadFile(manifestPath)
		if err != nil || readErr != nil || !os.SameFile(original, current) || original.Mode() != current.Mode() || !bytes.Equal(currentBody, manifest) {
			t.Fatal("lock-only update changed manifest identity or bytes", err, readErr)
		}
		if info, err := os.Lstat(filepath.Join(project, "uv.lock")); err != nil || info.Mode().Perm() != 0o600 {
			t.Fatal("lock-only original lock mode changed", err)
		}
		if body, err := os.ReadFile(note); err != nil || string(body) != "preserved user file" {
			t.Fatal("lock-only update changed user data", err)
		}
		if index == 0 {
			if _, err := os.Lstat(filepath.Join(project, ".venv")); !os.IsNotExist(err) {
				t.Fatal("lock-only update provisioned venv", err)
			}
		} else if body, err := os.ReadFile(filepath.Join(project, ".venv", "foreign-user-data")); err != nil || string(body) != "not a venv" {
			t.Fatal("lock-only update touched foreign venv", err)
		}
		if _, err := os.Lstat(filepath.Join(project, ".heliopause-uv-project-transaction.lock")); !os.IsNotExist(err) {
			t.Fatal("lock-only guard remains", err)
		}
		t.Logf("actual_uv_project_lock_only=PASS entries=%d mode=%s manifest=%s current complete inspection/Policy; only lock published; no venv installation", count, install.Mode(), id.String())
	}
}
