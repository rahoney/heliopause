//go:build linux && realcorpus

package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestUVPublicProjectApprovedPublicationIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_PROJECT_PUBLICATION_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user UV project publication qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
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
	native, err := sandbox.NewLinuxUVResolverWithExecutorAndPolicy(executor, observer.Observer(), policyService)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := sandbox.NewUVProjectResolver(native)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Error(err)
		}
	}()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, "project")
	venv := filepath.Join(project, ".venv")
	site := filepath.Join(venv, "lib/python3.14/site-packages")
	for _, path := range []string{site, filepath.Join(venv, "bin")} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(venv, "pyvenv.cfg"), []byte("version = "+sandbox.PinnedPythonRuntime().PythonVersion+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(fmt.Sprintf("[project]\nname='haa-published-project'\nversion='0.1.0'\nrequires-python='==%s'\ndependencies=[]\n", sandbox.PinnedPythonRuntime().PythonVersion))
	manifestPath := filepath.Join(project, "pyproject.toml")
	if err := os.WriteFile(manifestPath, manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	user := filepath.Join(site, "user-note.txt")
	if err := os.WriteFile(user, []byte("preserve user data"), 0o600); err != nil {
		t.Fatal(err)
	}
	intakeRoot, evidenceRoot, cacheRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "cache")
	intake, err := artifactpypi.NewPublicIntake(intakeRoot)
	if err != nil {
		t.Fatal(err)
	}
	runtime := sandbox.PinnedPythonRuntime()
	static, err := inspectionpypi.NewStaticInspector(intakeRoot, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
	if err != nil {
		t.Fatal(err)
	}
	backend, err := sandbox.NewLinuxPyPIDynamicBackendWithExecutor(intakeRoot, executor, observer.Observer(), newObservationResourceAdapter())
	if err != nil {
		t.Fatal(err)
	}
	dynamic, err := inspectionpypi.NewDynamicInspector(backend)
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := inspectionpypi.NewCompositeInspector(static, dynamic)
	if err != nil {
		t.Fatal(err)
	}
	store, err := local.NewStore(evidenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	promoter, err := promotion.NewPyPIPromotionWithRunner(cacheRoot, executor)
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(venv)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewPythonVenvInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	b := uvProjectBindings{intakeRoot, evidenceRoot, cacheRoot, intake, inspector, store, promoter, resolver}
	seen := map[string]bool{}
	for _, step := range []struct {
		name     string
		intent   artifactpypi.UVProjectIntent
		selector string
		count    int
	}{
		{"fresh exact add", artifactpypi.UVProjectAdd, "packaging==24.2", 1},
		{"version change", artifactpypi.UVProjectAdd, "packaging==25.0", 1},
		{"same version sync", artifactpypi.UVProjectSync, "", 1},
		{"second root", artifactpypi.UVProjectAdd, "idna==3.10", 2},
		{"edited empty sync", artifactpypi.UVProjectSync, "", 0},
	} {
		if !t.Run(step.name, func(t *testing.T) {
			if step.count == 0 {
				body, err := os.ReadFile(manifestPath)
				if err != nil {
					t.Fatal(err)
				}
				lines := strings.Split(string(body), "\n")
				start, end := -1, -1
				for i, line := range lines {
					if strings.HasPrefix(strings.TrimSpace(line), "dependencies") {
						start = i
						for j := i; j < len(lines); j++ {
							if strings.Contains(lines[j], "]") {
								end = j
								break
							}
						}
						break
					}
				}
				if start < 0 || end < start {
					t.Fatal("native manifest dependency array missing")
				}
				lines = append(append(lines[:start], "dependencies = []"), lines[end+1:]...)
				if err := os.WriteFile(manifestPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			service, err := newUVProjectService(b, project, step.intent, step.selector)
			if err != nil {
				t.Fatal(err)
			}
			approved, err := service.Apply(ctx, install)
			if err != nil {
				t.Fatal("actual native project publication", err)
			}
			if !approved.Valid() || len(approved.Inspected().Inspections()) != step.count || approved.Inspected().Snapshot().DependencyFree() != (step.count == 0) {
				t.Fatal("incomplete actual project approval")
			}
			for _, entry := range approved.Inspected().Inspections() {
				if seen[entry.RunID().String()] || len(entry.Evidence()) < 3 {
					t.Fatal("missing fresh complete inspection")
				}
				seen[entry.RunID().String()] = true
				completed := 0
				for _, check := range entry.Checks() {
					if check.Required() {
						if check.Status() != domain.ExecutionCompleted {
							t.Fatal("required check incomplete")
						}
						completed++
					}
				}
				if completed < 3 {
					t.Fatal("required wheel coverage missing")
				}
				path := filepath.Join(site, entry.Artifact().Identity().Name()+"-"+entry.Artifact().Identity().Version()+".dist-info", "METADATA")
				if body, err := os.ReadFile(path); err != nil || !bytes.Contains(body, []byte("Version: "+entry.Artifact().Identity().Version())) {
					t.Fatal("approved version not installed", err)
				}
			}
			id, handle, sbomHandle, ok := service.promoter.PublishedRecords()
			if !ok || id.String() == "" || sbomHandle == "" {
				t.Fatal("actual publication records unavailable")
			}
			cacheID, ok := strings.CutPrefix(strings.TrimSuffix(handle, ":manifest.json"), "project-cache:")
			if !ok {
				t.Fatal("wrong approved cache handle")
			}
			if _, err := domain.ParseRunID(cacheID); err != nil {
				t.Fatal(err)
			}
			payload := filepath.Join(cacheRoot, cacheID, "payload")
			body, err := os.ReadFile(filepath.Join(payload, "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Manifest string `json:"manifest_id"`
				Controls []struct {
					Name, SHA256 string
					Content      []byte
				} `json:"project_controls"`
			}
			if json.Unmarshal(body, &doc) != nil || doc.Manifest != id.String() || len(doc.Controls) != 6 {
				t.Fatal("actual native provenance records changed")
			}
			for _, control := range doc.Controls {
				if control.Name == "pyproject.toml" || control.Name == "uv.lock" {
					body, err := os.ReadFile(filepath.Join(project, control.Name))
					sum := sha256.Sum256(body)
					if err != nil || !bytes.Equal(body, control.Content) || hex.EncodeToString(sum[:]) != control.SHA256 {
						t.Fatal("published control differs from approved record", err)
					}
				}
			}
			if body, err := os.ReadFile(filepath.Join(payload, "sbom.cdx.json")); err != nil || !json.Valid(body) || !bytes.Contains(body, []byte(id.String())) {
				t.Fatal("actual SBOM differs", err)
			}
			if body, err := os.ReadFile(user); err != nil || string(body) != "preserve user data" {
				t.Fatal("user file changed", err)
			}
			if _, err := os.Lstat(filepath.Join(project, ".heliopause-uv-project-transaction.lock")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("project guard remains", err)
			}
			if step.count == 0 {
				for _, path := range []string{filepath.Join(site, "packaging"), filepath.Join(site, "idna")} {
					err := filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
						if errors.Is(err, os.ErrNotExist) {
							return nil
						}
						if err != nil {
							return err
						}
						if !entry.IsDir() {
							return errors.New("unselected installed file remains")
						}
						return nil
					})
					if err != nil {
						t.Fatal("unselected installed files remain", path, err)
					}
				}
			}
			t.Logf("actual_uv_project_publication=PASS entries=%d manifest=%s native/current inspection/atomic controls+venv; original ownership verified", step.count, id.String())
		}) {
			t.FailNow()
		}
	}
}
