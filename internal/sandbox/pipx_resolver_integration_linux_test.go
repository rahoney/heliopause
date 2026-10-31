//go:build linux && realcorpus

package sandbox

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/hosttool"
)

func TestLinuxPipxPublicInstallResolverIntegration(t *testing.T) {
	if os.Getenv("HELOX_PIPX_PUBLIC_RESOLVER_INTEGRATION") != "1" {
		t.Skip("requires explicit installed supply and observed public resolver namespace")
	}
	if os.Geteuid() == 0 {
		t.Fatal("ordinary caller required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Minute)
	defer cancel()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	client, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	resources := &pipxActualResources{ObservationResourceClient: integrationObservationResources{client: client}}
	resolver, err := NewLinuxPipxResolverWithExecutorAndPolicy(executor, supervisor.Observer(), integrationResolverPolicyService(t), resources)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Error(err)
		}
	}()
	requirement, _ := artifactpypi.ParseUpgradeRequirement("cowsay==6.1")
	selection, err := resolver.ResolveToolInstall(ctx, requirement)
	if err != nil {
		t.Fatal("public pipx selection/source/native binding failed", err)
	}
	resolution := selection.Resolution()
	if len(resolution.Graph().Nodes()) != 1 || len(selection.Controls()) != 4 || resolution.RuntimeIdentity() != "pipx-tool-install:"+PipxResolverRuntimeIdentity() || resolution.LockfileDigest().String() == "" {
		t.Fatal("public native graph/provenance differs")
	}
	var receipt artifactpypi.ToolReceipt
	for _, control := range selection.Controls() {
		t.Logf("pipx_public_control name=%s digest=%s", control.Name(), control.Digest().String())
		if control.Name() == "pipx-tool-receipt.json" {
			receipt, err = artifactpypi.ParsePipxToolReceipt(control.Body())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if receipt.Client() != "pipx" || receipt.Version() != "6.1" || len(receipt.Commands()) != 1 || receipt.Commands()[0].Name != "cowsay" || !receipt.Requirement().SameConstraints(requirement) {
		t.Fatal("native command differs")
	}
	for _, node := range resolution.Graph().Nodes() {
		id := node.Artifact().Identity()
		t.Logf("pipx_public_original name=%s version=%s locator=%q integrity=%q", id.Name(), id.Version(), node.Artifact().AcquisitionLocator(), node.Artifact().DeclaredIntegrity())
		if id.Name() != "cowsay" || id.Version() != "6.1" || id.Source() != artifactpypi.PublicPyPIProfile().Source() {
			t.Fatal("public source/root differs")
		}
	}
	t.Logf("public_native_pipx selector=%s graph=%d commands=%d controls=%d lock=%s runtime=%s", requirement.Requirement(), len(resolution.Graph().Nodes()), len(receipt.Commands()), len(selection.Controls()), resolution.LockfileDigest().String(), resolution.RuntimeIdentity())
	noCommand, _ := artifactpypi.ParseUpgradeRequirement("idna==3.11")
	refused, err := resolver.ResolveToolInstall(ctx, noCommand)
	if err == nil || len(refused.Controls()) != 0 || len(refused.Resolution().Graph().Nodes()) != 0 {
		t.Fatal("package without console command retained selection")
	}
	resources.mu.Lock()
	creates, registers, closes, cpu := resources.creates, resources.registers, resources.closes, resources.cpu
	resources.mu.Unlock()
	if creates != 2 || closes != 2 || registers != 2 || cpu == 0 || integrationObserverFaultReason(supervisor) != "" {
		t.Fatal("native resource accounting not reconciled")
	}
	t.Logf("public_native_pipx_no_command_refused error=%v creates=%d registers=%d closes=%d last_cpu_usec=%d reason=%s", err, creates, registers, closes, cpu, integrationObserverFaultReason(supervisor))
	t.Log("public live selection and independent original archive binding only; no inspection/Policy, target publication, original upgrade or CLI qualification")
	if !strings.Contains(resolution.RuntimeIdentity(), "backend:uv:") {
		t.Fatal("fixed backend absent")
	}
}

// Recover original inert controls only from the selected, independently bound
// archive snapshots; this never reads or imports a Host installation.
func pipxActualOriginalForSelection(t *testing.T, ctx context.Context, resolver *PipxResolver, selection PipxToolSelection) (artifactpypi.PipxToolUpgradeInput, []artifactpypi.UVInstalledWheelControls) {
	t.Helper()
	var receipt artifactpypi.ToolReceipt
	var lockBody, censusBody []byte
	for _, control := range selection.Controls() {
		switch control.Name() {
		case "pipx-tool-receipt.json":
			var err error
			receipt, err = artifactpypi.ParsePipxToolReceipt(control.Body())
			if err != nil {
				t.Fatal(err)
			}
		case "pipx-tool-pylock.toml":
			lockBody = control.Body()
		case "pipx-tool-census.json":
			censusBody = control.Body()
		}
	}
	if !receipt.Valid() || len(lockBody) == 0 || len(censusBody) == 0 {
		t.Fatal("actual original provenance absent")
	}
	lock, err := artifactpypi.ParseUVLock(receipt.Requirement().Reference(), lockBody, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]string{}
	for _, node := range selection.Resolution().Graph().Nodes() {
		wanted[node.Artifact().AcquisitionLocator()] = node.Artifact().DeclaredIntegrity()
	}
	var selected []artifactpypi.UVLockedWheel
	for _, pkg := range lock.Packages() {
		for _, wheel := range pkg.Wheels {
			if wanted[wheel.URL] == "sha256:"+wheel.SHA256 {
				selected = append(selected, wheel)
			}
		}
	}
	if len(selected) != len(wanted) {
		t.Fatal("original selected archive identity differs")
	}
	var census struct {
		Version int
		Wheels  []struct{ DistInfo, Metadata, Wheel string }
	}
	if err := json.Unmarshal(censusBody, &census); err != nil || census.Version != 1 {
		t.Fatal("actual native census invalid", err)
	}
	var original artifactpypi.PipxToolUpgradeInput
	var result []artifactpypi.UVInstalledWheelControls
	err = resolver.native.reader.withUVArchiveInputs(ctx, selected, func(inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) error {
		frozen, err := freezePipxNativeInput(ctx, lock, inputs, pages)
		if err != nil {
			return err
		}
		controls := map[string]*artifactpypi.UVInstalledWheelControls{}
		limit := artifactpypi.PublicPyPIProfile().ResourcePolicy().WheelLimits().MaxMetadata
		for _, file := range frozen.files {
			archive, err := zip.NewReader(bytes.NewReader(file.body), int64(len(file.body)))
			if err != nil {
				return err
			}
			for _, member := range archive.File {
				parts := strings.Split(member.Name, "/")
				if len(parts) != 2 || !strings.HasSuffix(parts[0], ".dist-info") || (parts[1] != "METADATA" && parts[1] != "WHEEL") {
					continue
				}
				if member.UncompressedSize64 > uint64(limit) {
					return errors.New("original metadata exceeds bound")
				}
				stream, err := member.Open()
				if err != nil {
					return err
				}
				body, readErr := io.ReadAll(io.LimitReader(stream, limit+1))
				closeErr := stream.Close()
				if err := errors.Join(readErr, closeErr); err != nil {
					return err
				}
				if int64(len(body)) > limit {
					return errors.New("original metadata overflow")
				}
				control := controls[parts[0]]
				if control == nil {
					control = &artifactpypi.UVInstalledWheelControls{DistInfo: parts[0]}
					controls[parts[0]] = control
				}
				if parts[1] == "METADATA" {
					control.Metadata = body
				} else {
					control.Wheel = body
				}
			}
		}
		var metadata []artifactpypi.InstalledMetadataFile
		for _, control := range controls {
			m, w := sha256.Sum256(control.Metadata), sha256.Sum256(control.Wheel)
			matched := false
			for _, expected := range census.Wheels {
				if expected.DistInfo == control.DistInfo && expected.Metadata == hex.EncodeToString(m[:]) && expected.Wheel == hex.EncodeToString(w[:]) {
					matched = true
				}
			}
			if !matched {
				return errors.New("original controls differ from observed native census")
			}
			metadata = append(metadata, artifactpypi.InstalledMetadataFile{Directory: control.DistInfo, Body: control.Metadata})
			result = append(result, *control)
		}
		if len(result) != len(census.Wheels) {
			return errors.New("original whole census incomplete")
		}
		original, err = artifactpypi.NewPipxToolUpgradeInput(receipt, metadata, PinnedPythonRuntime().PythonVersion)
		if err != nil {
			return err
		}
		runtime := PinnedPythonRuntime()
		result, err = original.ReplayWheelControls(result, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
		return err
	})
	if err != nil {
		t.Fatal("independent original archive binding failed", err)
	}
	return original, result
}

func TestLinuxPipxPublicOriginalUpgradeIntegration(t *testing.T) {
	if os.Getenv("HELOX_PIPX_PUBLIC_UPGRADE_INTEGRATION") != "1" {
		t.Skip("requires explicit protected supply/original-aware native namespace")
	}
	if os.Geteuid() == 0 {
		t.Fatal("ordinary caller required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	client, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	resources := &pipxActualResources{ObservationResourceClient: integrationObservationResources{client: client}}
	runner := &pipxNativeSessionActualRunner{CommandRunner: executor}
	resolver, err := NewLinuxPipxResolverWithExecutorAndPolicy(runner, supervisor.Observer(), integrationResolverPolicyService(t), resources)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Error(err)
		}
	}()
	logFailure := func(label string) {
		t.Helper()
		prefix := runner.nativeLayoutPrefix()
		name := filepath.Join(os.TempDir(), "haa-pipx-native-"+label+"-layout-prefix.tar")
		if len(prefix) != 0 {
			if err := os.WriteFile(name, prefix, 0600); err != nil {
				t.Logf("layout prefix write failed: %v", err)
			}
		}
		resources.mu.Lock()
		creates, registers, closes, cpu, last := resources.creates, resources.registers, resources.closes, resources.cpu, resources.lastReadCPU
		resources.mu.Unlock()
		t.Logf("native_original_failure stage=%s targets=%d creates=%d registers=%d closes=%d cpu_usec=%d last_read_cpu_usec=%d prefix_bytes=%d prefix_sha256=%x reason=%s", label, runner.targets.Load(), creates, registers, closes, cpu, last, len(prefix), sha256.Sum256(prefix), integrationObserverFaultReason(supervisor))
	}
	note := func(label string, expected int) {
		t.Helper()
		resources.mu.Lock()
		creates, registers, closes, cpu, last := resources.creates, resources.registers, resources.closes, resources.cpu, resources.lastReadCPU
		resources.mu.Unlock()
		t.Logf("pipx_original_%s creates=%d registers=%d closes=%d last_cpu_usec=%d last_read_cpu_usec=%d reason=%s", label, creates, registers, closes, cpu, last, integrationObserverFaultReason(supervisor))
		if creates != expected || registers != expected || closes != expected || cpu == 0 || cpu >= 13_000_000 || integrationObserverFaultReason(supervisor) != "" {
			t.Fatal("native original lifecycle/accounting incomplete")
		}
	}
	logSelection := func(label string, selection PipxToolSelection) {
		t.Helper()
		t.Logf("pipx_original_selection stage=%s controls=%d lock=%s runtime=%s", label, len(selection.Controls()), selection.Resolution().LockfileDigest().String(), selection.Resolution().RuntimeIdentity())
		for _, control := range selection.Controls() {
			t.Logf("pipx_original_control stage=%s name=%s digest=%s", label, control.Name(), control.Digest().String())
		}
		for _, node := range selection.Resolution().Graph().Nodes() {
			artifact := node.Artifact()
			t.Logf("pipx_upgrade_original stage=%s name=%s version=%s locator=%q integrity=%q", label, artifact.Identity().Name(), artifact.Identity().Version(), artifact.AcquisitionLocator(), artifact.DeclaredIntegrity())
		}
	}
	request, _ := artifactpypi.ParseUpgradeRequirement("cowsay==6.0")
	first, err := resolver.ResolveToolInstall(ctx, request)
	if err != nil {
		t.Fatal("original public native install failed", err)
	}
	note("fresh", 1)
	logSelection("fresh", first)
	original, wheels := pipxActualOriginalForSelection(t, ctx, resolver, first)
	if original.Receipt().Version() != "6.0" || original.Request().Requirement() != "cowsay" {
		t.Fatal("old constraint changed or new request not bare")
	}
	upgraded, err := resolver.ResolveToolUpgrade(ctx, original, wheels)
	if err != nil {
		logFailure("upgrade")
		t.Logf("native_upgrade_failure_output=%q", runner.nativeFailureOutput())
		t.Fatal("observed original-aware upgrade failed", err)
	}
	note("upgrade", 2)
	logSelection("upgrade", upgraded)
	if len(upgraded.Controls()) != 7 || upgraded.Resolution().RuntimeIdentity() != "pipx-tool-upgrade:"+PipxResolverRuntimeIdentity() {
		t.Fatal("original upgrade provenance missing")
	}
	current, currentWheels := pipxActualOriginalForSelection(t, ctx, resolver, upgraded)
	if current.Receipt().Version() == "6.0" || current.Receipt().Requirement().Requirement() != "cowsay" {
		t.Fatal("upgrade did not change version or retained old constraint")
	}
	t.Logf("public_native_pipx_upgrade old=6.0 new=%s graph=%d controls=%d lock=%s", current.Receipt().Version(), len(upgraded.Resolution().Graph().Nodes()), len(upgraded.Controls()), upgraded.Resolution().LockfileDigest().String())
	noop, err := resolver.ResolveToolUpgrade(ctx, current, currentWheels)
	if err != nil {
		t.Logf("native_upgrade_failure_output=%q", runner.nativeFailureOutput())
		logFailure("noop")
		t.Fatal("observed original-aware no-op failed", err)
	}
	note("noop", 3)
	var finalReceipt artifactpypi.ToolReceipt
	for _, control := range noop.Controls() {
		if control.Name() == "pipx-tool-receipt.json" {
			finalReceipt, err = artifactpypi.ParsePipxToolReceipt(control.Body())
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(noop.Controls()) != 7 || finalReceipt.Version() != current.Receipt().Version() || finalReceipt.Requirement().Requirement() != "cowsay" {
		t.Fatal("no-op changed selected native version/intent")
	}
	logSelection("noop", noop)
	t.Log("public original-aware version delta and no-op metadata only; no Host code/interpreter replay, inspection/Policy, publication, ownership or CLI qualification")
}
