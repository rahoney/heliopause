//go:build linux && realcorpus

package bootstrap

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestPNPMNativeOriginalIntegration(t *testing.T) {
	if os.Getenv("HELOX_PNPM_NATIVE_ORIGINAL_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user native original qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "user-cache")
	t.Setenv("XDG_CACHE_HOME", cache)
	for _, item := range []struct {
		name, manifest, code, action, selector, version string
		count                                           int
	}{
		{"single", `{"scripts":{"install":"exit 99"},"dependencies":{"is-number":"6.0.0"}}`, "node_modules/.pnpm/is-number@6.0.0/node_modules/is-number/index.js", "add", "is-number@7.0.0", "7.0.0", 1},
		{"native-update", `{"scripts":{"install":"exit 99"},"dependencies":{"is-number":"6.0.0"}}`, "node_modules/.pnpm/is-number@6.0.0/node_modules/is-number/index.js", "update", "is-number", "6.0.0", 1},
		{"peer", `{"scripts":{"install":"exit 99"},"dependencies":{"react":"18.3.1","use-sync-external-store":"1.2.0"}}`, "node_modules/.pnpm/react@18.3.1/node_modules/react/index.js", "install", "", "18.3.1", 4},
	} {
		if !t.Run(item.name, func(t *testing.T) {
			target := filepath.Join(root, item.name)
			if err := os.Mkdir(target, 0o700); err != nil {
				t.Fatal(err)
			}
			input := pnpmNativeOriginalFixture(t, ctx, item.manifest, target, filepath.Join(root, item.name+"-fixture-intake"))
			if err := os.WriteFile(filepath.Join(target, "user-note"), []byte("preserve native user file\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			code := filepath.Join(target, item.code)
			original, err := os.ReadFile(code)
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.Lstat(code)
			if err != nil {
				t.Fatal(err)
			}
			refuse := func(adopt bool, expected string) {
				t.Helper()
				args := []string{"--output", "json", "pnpm", "install", "--target", target}
				if adopt {
					args = append(args, "--adopt")
				}
				var stdout, stderr bytes.Buffer
				if err := Run(ctx, args, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), expected) {
					t.Fatal("native refusal did not reach its intended boundary", expected, err, stdout.String(), stderr.String())
				}
				after, err := os.Lstat(code)
				if err != nil || !os.SameFile(before, after) {
					t.Fatal("refusal replaced original", err)
				}
				if _, err := os.Lstat(filepath.Join(target, ".heliopause")); !os.IsNotExist(err) {
					t.Fatal("refusal created target marker", err)
				}
			}
			refuse(false, "requires independently proven native adoption")
			if item.name == "single" {
				if err := os.WriteFile(code, []byte("untrusted old code"), before.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
				refuse(true, "acquired payload")
				body, err := os.ReadFile(code)
				if err != nil || string(body) != "untrusted old code" {
					t.Fatal("failed proof changed tampered original", err)
				}
				if err := os.WriteFile(code, original, before.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
				lockpath := filepath.Join(target, "pnpm-lock.yaml")
				lockbody, err := os.ReadFile(lockpath)
				if err != nil {
					t.Fatal(err)
				}
				integrity := input.Context().Lock().Packages()[0].Integrity
				forged := bytes.ReplaceAll(lockbody, []byte(integrity), []byte("sha512-"+strings.Repeat("A", 86)+"=="))
				if bytes.Equal(forged, lockbody) {
					t.Fatal("SRI negative did not change original lock")
				}
				if err := os.WriteFile(lockpath, forged, 0o600); err != nil {
					t.Fatal(err)
				}
				refuse(true, "independent public registry metadata")
				if err := os.WriteFile(lockpath, lockbody, 0o600); err != nil {
					t.Fatal(err)
				}
				foreign := filepath.Join(target, "node_modules/foreign.js")
				if err := os.WriteFile(foreign, []byte("foreign"), 0o600); err != nil {
					t.Fatal(err)
				}
				refuse(true, "entry coverage")
				if err := os.Remove(foreign); err != nil {
					t.Fatal(err)
				}
			}
			var stdout, stderr bytes.Buffer
			operation, selector := item.action, item.selector
			args := []string{"--lang", "ko", "--output", "json", "pnpm", operation, "--target", target, "--adopt"}
			if selector != "" {
				args = append(args, selector)
			}
			if err := Run(ctx, args, &stdout, &stderr); err != nil {
				t.Fatal("actual native original handoff failed", err, stderr.String(), stdout.String())
			}
			var report struct {
				Schema    string                    `json:"schema_version"`
				Operation string                    `json:"operation"`
				Status    string                    `json:"operation_status"`
				Promotion string                    `json:"promotion_status"`
				Source    string                    `json:"source_id"`
				Policy    struct{ Decision string } `json:"policy"`
				Entries   []struct {
					Run    string `json:"run_id"`
					Checks []struct {
						Required   bool
						Capability string
						Status     string `json:"execution_status"`
					}
					Evidence []json.RawMessage `json:"evidence_references"`
				} `json:"entries"`
				Records struct {
					Manifest string `json:"manifest_id"`
					SBOM     string `json:"sbom_handle"`
					Staging  string `json:"staging_handle"`
					Count    int    `json:"entry_count"`
				} `json:"verified_set"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatal("non-JSON CLI stdout", err, stdout.String())
			}
			wantOp, wantSchema := "INSTALL", "helox.project-install-result/v1"
			if operation == "update" {
				wantOp, wantSchema = "UPDATE", "helox.project-update-result/v1"
			}
			if report.Schema != wantSchema || report.Operation != wantOp || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || report.Source != "npm" || report.Policy.Decision != "ALLOW" || len(report.Entries) != item.count || report.Records.Count != item.count || !strings.HasPrefix(report.Records.Staging, "project-cache:") || report.Records.SBOM != report.Records.Staging+":sbom.cdx.json" {
				t.Fatal("incomplete actual CLI result", stdout.String())
			}
			for _, entry := range report.Entries {
				if entry.Run == "" || len(entry.Evidence) == 0 || len(entry.Checks) < 3 {
					t.Fatal("missing actual per-entry facts", stdout.String())
				}
				for _, check := range entry.Checks {
					if check.Required && (check.Status != "COMPLETED" || check.Capability != "SUPPORTED") {
						t.Fatal("incomplete required check", stdout.String())
					}
				}
			}
			stateHash := sha256.Sum256([]byte(target))
			stateName := hex.EncodeToString(stateHash[:]) + ".json"
			stateBody, err := os.ReadFile(filepath.Join(cache, "heliopause/pnpm-projects", stateName))
			if err != nil {
				t.Fatal(err)
			}
			var state struct{ OriginalProof string }
			if json.Unmarshal(stateBody, &state) != nil || state.OriginalProof == "" {
				t.Fatal("native independent original source proof absent for this exact target")
			}
			if item.name != "single" {
				after, err := os.Lstat(code)
				if err != nil || os.SameFile(before, after) {
					t.Fatal("first adoption retained unapproved payload instead of publishing clean inspected copy", err)
				}
			}
			note, err := os.ReadFile(filepath.Join(target, "user-note"))
			if err != nil || string(note) != "preserve native user file\n" {
				t.Fatal("native user file not preserved", err)
			}
			recordBody, err := os.ReadFile(filepath.Join(target, ".heliopause/manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			var record struct {
				Manifest  string `json:"manifest_id"`
				Operation string `json:"operation_id"`
			}
			if json.Unmarshal(recordBody, &record) != nil || record.Manifest != report.Records.Manifest || record.Operation == "" {
				t.Fatal("actual native handoff Manifest differs from completion")
			}
			packagePath := "node_modules/is-number/package.json"
			if item.name == "peer" {
				packagePath = "node_modules/react/package.json"
			}
			packageBody, err := os.ReadFile(filepath.Join(target, packagePath))
			var installed struct{ Version string }
			if err != nil || json.Unmarshal(packageBody, &installed) != nil || installed.Version != item.version {
				t.Fatal("native selection did not preserve its actual requested semantics", err, string(packageBody))
			}
			if item.name == "single" {
				if _, err := os.Lstat(filepath.Join(target, "node_modules/.pnpm/is-number@6.0.0")); !os.IsNotExist(err) {
					t.Fatal("retired native source payload survived", err)
				}
			}
			t.Logf("pnpm_native_original case=%s old_archives=%d action=%s fresh_policy=ALLOW full_source_proof=true native_copy_fixture=true root_script=NOT_EXECUTED user_file=preserved", item.name, len(input.Artifacts()), operation)
		}) {
			return
		}
	}
}

// This producer owns and drains its observer before the production CLI starts.
func pnpmNativeOriginalFixture(t *testing.T, ctx context.Context, manifestBody, target, intake string) artifactnpm.PNPMInspectionInput {
	t.Helper()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	launcher, err := hosttool.NewSystemObserverLauncher()
	if err != nil {
		t.Fatal(err)
	}
	observer, err := sandbox.NewObserverSupervisor(ctx, func(start context.Context, remote, output string) (sandbox.ObserverProcess, error) {
		return launcher.StartObserver(start, remote, output)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := observer.Close(); err != nil {
			t.Error(err)
		}
	}()
	policyService, err := newSystemResolverPolicyAdapter()
	if err != nil {
		t.Fatal(err)
	}
	native, err := sandbox.NewLinuxPNPMResolverWithExecutorAndPolicy(executor, observer.Observer(), policyService)
	if err != nil {
		t.Fatal(err)
	}

	source, err := artifactnpm.NewPublicResolver(intake)
	if err != nil {
		t.Fatal(err)
	}
	static, err := inspectionnpm.NewInspector(intake)
	if err != nil {
		t.Fatal(err)
	}
	prepare, err := sandbox.NewPNPMInspectionPreparer(native, intake)
	if err != nil {
		t.Fatal(err)
	}
	destination, err := domain.NewInstallTarget(target)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewNPMProjectInstallContext(destination)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := domain.NewProjectControlFile("package.json", []byte(manifestBody), true)
	if err != nil {
		t.Fatal(err)
	}
	absent, err := domain.NewProjectControlFile("pnpm-lock.yaml", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	selectionInput, err := artifactnpm.NewPNPMSelectionInput(artifactnpm.PNPMInstall, "", []domain.ProjectControlFile{manifest, absent}, runtimeidentity.PNPMVersion)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := native.SelectProject(ctx, selectionInput)
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := selected.BindInspectionContext(ctx, source, install)
	if err != nil {
		t.Fatal(err)
	}
	acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	for _, resolved := range frozen.Snapshot().Dependencies() {
		run, err := domain.NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		subject, err := source.Acquire(ctx, run, resolved)
		if err != nil {
			t.Fatal(err)
		}
		node, err := domain.ProjectDependencyNodeID(resolved.Identity())
		if err != nil {
			t.Fatal(err)
		}
		acquired[node] = subject
	}
	input, err := frozen.BindAcquired(frozen.Snapshot(), acquired)
	if err != nil {
		t.Fatal(err)
	}
	var inventories []artifactnpm.ArchiveInventory
	for _, subject := range input.Artifacts() {
		inventory, report, err := static.InspectArchive(ctx, subject)
		if err != nil || len(report.Findings()) != 0 || !inventory.Matches(subject) {
			t.Fatal("native original fixture inventory", err)
		}
		inventories = append(inventories, inventory)
	}
	layout, err := prepare.PrepareInspection(ctx, input, inventories)
	if err != nil {
		t.Fatal(err)
	}
	// Copy actual isolated native output as a safe single-link original fixture.
	// No Host native execution, marker or independent HAA approval is created.
	for _, entry := range layout.Entries() {
		if entry.Kind == tar.TypeDir {
			if err := os.MkdirAll(filepath.Join(target, entry.Name), os.FileMode(entry.Mode)); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, entry := range layout.Entries() {
		name := filepath.Join(target, entry.Name)
		switch entry.Kind {
		case tar.TypeReg, tar.TypeLink:
			body, err := layout.ReadFile(entry.Name)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(name, body, os.FileMode(entry.Mode)); err != nil {
				t.Fatal(err)
			}
		case tar.TypeSymlink:
			if err := os.Symlink(entry.Link, name); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, control := range selected.Controls() {
		if err := os.WriteFile(filepath.Join(target, control.Name()), control.Body(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return input
}
