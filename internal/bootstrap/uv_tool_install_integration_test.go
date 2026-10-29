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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	evidencerecord "github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

// The target interpreter fixture is authenticated image DATA. Neither it nor
// any exposed command is executed on the Host. This qualifies current whole
// inspection and installed-file publication, not Host runtime provisioning.
func TestUVPublicToolApprovedInstallationIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_INSTALL_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user UV tool installation qualification")
	}
	runPublicToolApprovedInstallationIntegration(t, false)
}

func TestPipxPublicToolApprovedInstallationIntegration(t *testing.T) {
	if os.Getenv("HELOX_PIPX_TOOL_INSTALL_INTEGRATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user pipx tool installation qualification")
	}
	runPublicToolApprovedInstallationIntegration(t, true)
}

func runPublicToolApprovedInstallationIntegration(t *testing.T, pipx bool) {
	t.Helper()
	ctx, bindings, root, interpreter, nativePipx := toolApprovedIntegrationFixture(t, pipx)
	receiptName, receiptControl := "uv-receipt.toml", "uv-tool-host-receipt.toml"
	parseReceipt := artifactpypi.ParseUVToolReceipt
	if pipx {
		receiptName, receiptControl = "pipx_metadata.json", "pipx-tool-host-receipt.json"
		parseReceipt = artifactpypi.ParsePipxToolReceipt
	}
	store, stagingRoot := bindings.evidence, bindings.stagingRoot
	seen := map[string]bool{}
	for _, step := range []struct{ name, requirement string }{{"plain", "cowsay==6.1"}, {"env with spaces", "cowsay>=6.1,<=6.1"}} {
		t.Run(step.name, func(t *testing.T) {
			venv := filepath.Join(root, step.name, "cowsay")
			bin := filepath.Join(root, step.name, "commands")
			install := prepareUVToolTargetFixture(t, venv, bin, interpreter)
			user := filepath.Join(venv, "user-note.txt")
			if err := os.WriteFile(user, []byte("keep user data"), 0600); err != nil {
				t.Fatal(err)
			}
			var service *application.PreparedInstallService
			var err error
			if pipx {
				service, err = newPipxToolInstallService(bindings, nativePipx, bin)
			} else {
				service, err = newUVToolInstallService(bindings, bin)
			}
			if err != nil {
				t.Fatal(err)
			}
			requirement, err := artifactpypi.ParseUpgradeRequirement(step.requirement)
			if err != nil {
				t.Fatal(err)
			}
			reference, err := domain.NewArtifactReference(artifactpypi.PublicPyPIProfile().Source(), requirement.Requirement())
			if err != nil {
				t.Fatal(err)
			}
			request, err := application.NewInstallRequest(reference, install)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := service.Install(ctx, request)
			if err != nil {
				t.Fatal("actual approved tool install", err, outcome.FailureCode())
			}
			bundle := outcome.Bundle()
			if outcome.FailureCode() != "" || !bundle.Valid() || outcome.Inspection().Decision().Decision() != domain.DecisionAllow || outcome.Promoted().ManifestID() != bundle.ManifestID() || outcome.Promoted().Target() != install.Target() || len(bundle.Set().Inspected().Inspections()) != 1 {
				t.Fatal("current whole tool approval/publication missing")
			}
			for _, entry := range bundle.Set().Inspected().Inspections() {
				if seen[entry.RunID().String()] || len(entry.Evidence()) < 3 {
					t.Fatal("fresh current inspection evidence missing")
				}
				seen[entry.RunID().String()] = true
				completed := 0
				for _, check := range entry.Checks() {
					if check.Required() {
						if check.Status() != domain.ExecutionCompleted {
							t.Fatal("required tool check incomplete")
						}
						completed++
					}
				}
				if completed < 3 {
					t.Fatal("whole required coverage missing")
				}
				for _, ref := range entry.Evidence() {
					if _, _, err := store.ReadReference(ctx, entry.RunID(), ref, entry.Artifact().Identity(), entry.Artifact().Digest()); err != nil {
						t.Fatal("current Evidence readback", err)
					}
				}
			}
			manifest, err := os.ReadFile(filepath.Join(stagingRoot, bundle.ManifestID().String(), "manifest.json"))
			if err != nil || !bytes.Equal(manifest, bundle.ManifestDocument()) {
				t.Fatal("staged current manifest differs", err)
			}
			if sbom, err := os.ReadFile(filepath.Join(stagingRoot, bundle.ManifestID().String(), "sbom.cdx.json")); err != nil || !bytes.Equal(sbom, bundle.SBOMDocument()) {
				t.Fatal("staged current SBOM differs", err)
			}
			var doc struct {
				Controls []struct {
					Name    string
					Content []byte
				} `json:"project_controls"`
			}
			if err := json.Unmarshal(manifest, &doc); err != nil || len(doc.Controls) != 6 {
				t.Fatal("native and Host tool controls missing")
			}
			published, err := os.ReadFile(filepath.Join(venv, receiptName))
			if err != nil {
				t.Fatal(err)
			}
			bound, err := parseReceipt(published)
			if err != nil || !bound.Requirement().SameConstraints(requirement) {
				t.Fatal("published native requirement differs", err)
			}
			if err := bound.ValidateExposedCommands(venv, bin, filepath.Join(venv, "bin/python"), map[string]string{filepath.Join(bin, "cowsay"): filepath.Join(venv, "bin/cowsay")}); err != nil {
				t.Fatal(err)
			}
			matched := false
			for _, control := range doc.Controls {
				if control.Name == receiptControl {
					matched = bytes.Equal(control.Content, published)
				}
			}
			if !matched {
				t.Fatal("published receipt differs from approved control")
			}
			link, err := os.Readlink(filepath.Join(bin, "cowsay"))
			if err != nil || link != filepath.Join(venv, "bin/cowsay") {
				t.Fatal("exposed command link differs")
			}
			script, err := os.ReadFile(link)
			if err != nil {
				t.Fatal(err)
			}
			python := filepath.Join(venv, "bin/python")
			validPrefix := bytes.HasPrefix(script, []byte("#!"+python+"\n")) || bytes.HasPrefix(script, []byte("#!/bin/sh\n'''exec' \""+python+"\" \"$0\" \"$@\"\n' '''\n")) || bytes.HasPrefix(script, []byte("#!/bin/sh\n'''exec' "+python+" \"$0\" \"$@\"\n' '''\n"))
			if !validPrefix {
				t.Fatalf("fixed pip wrapper has wrong final interpreter: %q", script)
			}
			if data, err := os.ReadFile(user); err != nil || string(data) != "keep user data" {
				t.Fatal("user data changed")
			}
			for _, directory := range []string{venv, bin} {
				entries, err := os.ReadDir(directory)
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), ".haa-") || strings.HasPrefix(entry.Name(), ".heliopause-") {
						t.Fatal("tool temporary data or lock retained", entry.Name())
					}
				}
			}
			before := uvToolTargetState(t, venv, bin)
			rejected, err := service.Install(ctx, request)
			if err == nil || rejected.Bundle().Valid() || rejected.Promoted().ManifestID().String() != "" {
				t.Fatal("existing tool routed through fresh install")
			}
			if !reflect.DeepEqual(before, uvToolTargetState(t, venv, bin)) {
				t.Fatal("fresh refusal changed existing tool")
			}
			t.Logf("current_tool_approved pipx=%t request=%q controls=%d inspections=%d manifest=%s decision=ALLOW published_receipt=%s repeat_refused_unchanged=true", pipx, step.requirement, len(doc.Controls), len(bundle.Set().Inspected().Inspections()), bundle.ManifestID().String(), receiptName)
		})
	}
}

func uvToolApprovedIntegrationFixture(t *testing.T) (context.Context, pythonClientBindings, string, []byte) {
	t.Helper()
	ctx, bindings, root, interpreter, _ := toolApprovedIntegrationFixture(t, false)
	return ctx, bindings, root, interpreter
}

func toolApprovedIntegrationFixture(t *testing.T, pipx bool) (context.Context, pythonClientBindings, string, []byte, *sandbox.PipxResolver) {
	ctx, bindings, root, interpreter, nativePipx, _ := toolApprovedIntegrationFixtureWithPythonBackend(t, pipx)
	return ctx, bindings, root, interpreter, nativePipx
}

func toolApprovedIntegrationFixtureWithPythonBackend(t *testing.T, pipx bool) (context.Context, pythonClientBindings, string, []byte, *sandbox.PipxResolver, *sandbox.PythonDynamicBackend) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	t.Cleanup(cancel)
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	})
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
	t.Cleanup(func() {
		if err := observer.Close(); err != nil {
			t.Error(err)
		}
	})
	resolverPolicy, err := newSystemResolverPolicyAdapter()
	if err != nil {
		t.Fatal(err)
	}
	native, err := sandbox.NewLinuxUVResolverWithExecutorAndPolicy(executor, observer.Observer(), resolverPolicy)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := native.Close(); err != nil {
			t.Error(err)
		}
	})
	var nativePipx *sandbox.PipxResolver
	if pipx {
		nativePipx, err = sandbox.NewLinuxPipxResolverWithExecutorAndPolicy(executor, observer.Observer(), resolverPolicy, newObservationResourceAdapter())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := nativePipx.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	intakeRoot, evidenceRoot, stagingRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging")
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
	inspection, err := application.NewInstallInspectService(native, intake, verificationpypi.IntegrityVerifier{}, inspector, store, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	staging, err := promotion.NewLocalStaging(intakeRoot, evidenceRoot, stagingRoot)
	if err != nil {
		t.Fatal(err)
	}
	promoter, err := promotion.NewPyPIPromotionWithRunner(stagingRoot, executor)
	if err != nil {
		t.Fatal(err)
	}
	base, err := application.NewInstallService(inspection, evidencerecord.Generator{}, staging, promoter)
	if err != nil {
		t.Fatal(err)
	}
	bindings := pythonClientBindings{installer: base, promoter: promoter, resolver: native, nativeLookup: native, intake: intake, intakeRoot: intakeRoot, evidenceRoot: evidenceRoot, stagingRoot: stagingRoot, inspector: inspector, evidence: store}
	interpreter := uvToolTargetInterpreterFixture(t)
	return ctx, bindings, root, interpreter, nativePipx, backend
}

func uvToolTargetInterpreterFixture(t *testing.T) []byte {
	t.Helper()
	path := os.Getenv("HELOX_TOOL_TARGET_PYTHON_FIXTURE")
	expected := os.Getenv("HELOX_TOOL_TARGET_PYTHON_SHA256")
	if !filepath.IsAbs(path) || len(expected) != 64 {
		t.Fatal("declare pinned-image interpreter data fixture")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 16<<20 {
		t.Fatal("interpreter data fixture is unavailable or exceeds bounds")
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(io.LimitReader(file, (16<<20)+1))
	closeErr := file.Close()
	after, statErr := os.Lstat(path)
	sum := sha256.Sum256(body)
	if readErr != nil || closeErr != nil || statErr != nil || !os.SameFile(info, after) || int64(len(body)) != info.Size() || hex.EncodeToString(sum[:]) != expected || !bytes.HasPrefix(body, []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatal("interpreter data fixture differs from image receipt")
	}
	return body
}

func prepareUVToolTargetFixture(t *testing.T, root, bin string, interpreter []byte) domain.InstallContext {
	t.Helper()
	for _, path := range []string{filepath.Join(root, "bin"), filepath.Join(root, "lib/python3.14/site-packages"), bin} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "pyvenv.cfg"), []byte("version = "+sandbox.PinnedPythonRuntime().PythonVersion+"\ninclude-system-site-packages = false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin/python3.14"), interpreter, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("python3.14", filepath.Join(root, "bin/python")); err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(root)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewPythonVenvInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	return install
}

func uvToolTargetState(t *testing.T, roots ...string) map[string]string {
	t.Helper()
	state := map[string]string{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, _ fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			info, err := os.Lstat(path)
			if err != nil {
				return err
			}
			identity, ok := info.Sys().(*syscall.Stat_t)
			if !ok {
				return errors.New("tool target identity unavailable")
			}
			stamp := fmt.Sprintf("%d:%d:%s", identity.Dev, identity.Ino, info.Mode())
			if info.IsDir() {
				state[path] = "directory:" + stamp
				return nil
			}
			if info.Mode()&os.ModeSymlink != 0 {
				link, err := os.Readlink(path)
				if err != nil {
					return err
				}
				state[path] = "link:" + stamp + ":" + link
				return nil
			}
			if !info.Mode().IsRegular() {
				return errors.New("unexpected tool target type")
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(body)
			state[path] = stamp + ":" + hex.EncodeToString(sum[:])
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return state
}
