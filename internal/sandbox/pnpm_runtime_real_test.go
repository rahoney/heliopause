//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const pnpmProjectDirectory = pnpmProjectRoot + "/project"

func TestPNPMActualIsolatedNativeStartup(t *testing.T) {
	withActualPNPMNativeRuntime(t, func(ctx context.Context, runner CommandRunner, id string) {
		version, err := runner.Output(ctx, "docker", boundaryExecArguments(id, boundaryLaunchMode, pnpmToolGuestPath, "--version")...)
		if err != nil || strings.TrimSpace(string(version)) != runtimeidentity.PNPMVersion {
			t.Fatal("isolated native PNPM startup failed", err)
		}
		t.Logf("isolated_native_pnpm_version=%s", strings.TrimSpace(string(version)))
	})
}

// Empty, controller-owned input tests native configuration and Node runtime
// query without a public graph, artifact scripts, downloads or publication.
func TestPNPMActualIsolatedEmptyLock(t *testing.T) { runActualPNPMLockProbe(t, false) }
func TestPNPMActualIsolatedNodeQuery(t *testing.T) { runActualPNPMLockProbe(t, true) }

func runActualPNPMLockProbe(t *testing.T, requireNode bool) {
	t.Helper()
	withActualPNPMNativeRuntime(t, func(ctx context.Context, runner CommandRunner, id string) {
		input, ok := runner.(inputCommandRunner)
		if !ok {
			t.Fatal("PNPM input transport unavailable")
		}
		nodeVersion, _, valid := strings.Cut(strings.TrimPrefix(runtimeidentity.NodeImageReference, "node:"), "-slim@sha256:")
		if !valid || nodeVersion == "" {
			t.Fatal("unknown pinned Node image tag format")
		}
		manifest := map[string]any{"name": "haa-pnpm-probe", "version": "0.0.0", "private": true, "engines": map[string]string{"node": nodeVersion}}
		if requireNode {
			manifest["devEngines"] = map[string]any{"runtime": map[string]string{"name": "node", "version": nodeVersion, "onFail": "error"}}
		}
		body, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		var transport bytes.Buffer
		archive := tar.NewWriter(&transport)
		if err := archive.WriteHeader(&tar.Header{Name: "project/", Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}); err != nil {
			t.Fatal(err)
		}
		if err := archive.WriteHeader(&tar.Header{Name: "project/package.json", Typeflag: tar.TypeReg, Mode: 0600, Uid: 1000, Gid: 1000, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write(body); err != nil {
			t.Fatal(err)
		}
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
		if err := input.RunInput(ctx, &transport, "docker", "cp", "--archive", "-", id+":"+pnpmProjectRoot); err != nil {
			t.Fatal("PNPM empty input copy failed", err)
		}
		args := []string{pnpmToolGuestPath, "--dir", pnpmProjectDirectory, "install", "--offline", "--lockfile-only", "--ignore-scripts", "--ignore-workspace", "--store-dir", pnpmProjectRoot + "/store", "--state-dir", pnpmProjectRoot + "/state", "--reporter=silent"}
		if _, err := runner.Output(ctx, "docker", boundaryExecArguments(id, boundaryLaunchMode, args...)...); err != nil {
			t.Fatal("PNPM isolated empty lock failed", err)
		}
		// Use the existing bounded Docker transport. This is format observation;
		// the public lock/graph parser and Policy binding remain P2c.
		streams, ok := runner.(streamingDockerOutput)
		if !ok {
			t.Fatal("PNPM bounded lock transport unavailable")
		}
		output := nativeToolArchiveOutput{limit: 64 << 10}
		if err := streams.RunOutput(ctx, &output, "docker", "cp", id+":"+pnpmProjectDirectory+"/pnpm-lock.yaml", "-"); err != nil {
			t.Fatal("PNPM lock read failed", err)
		}
		reader := tar.NewReader(bytes.NewReader(output.body))
		h, err := reader.Next()
		if err != nil || h.Name != "pnpm-lock.yaml" || h.Typeflag != tar.TypeReg || h.Uid != 1000 || h.Gid != 1000 || h.Size <= 0 || h.Size > 16<<10 {
			t.Fatal("PNPM empty lock transport differs", err)
		}
		lock, err := io.ReadAll(reader)
		if err != nil || int64(len(lock)) != h.Size {
			t.Fatal("PNPM empty lock read failed", err)
		}
		if _, err := reader.Next(); err != io.EOF {
			t.Fatal("PNPM lock transport has extra members", err)
		}
		if !bytes.Contains(lock, []byte("lockfileVersion: '9.0'")) || !bytes.Contains(lock, []byte("importers:")) {
			t.Fatal("PNPM empty lock format differs", string(lock))
		}
		t.Logf("isolated_native_pnpm_empty_lock=%q require_node=%t", lock, requireNode)
	})
}

func withActualPNPMNativeRuntime(t *testing.T, operation func(context.Context, CommandRunner, string)) {
	t.Helper()
	if os.Geteuid() == 0 || os.Getenv("HELOX_PNPM_NATIVE_RUNTIME") != "1" {
		t.Fatal("requires declared ordinary-user isolated PNPM runtime qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
	runner := admissionAwareRunner(executor)
	tool, err := preparePNPMTool(ctx)
	if err != nil {
		t.Fatal(err)
	}
	volume, err := createPNPMRuntimeVolume(ctx, runner)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("owned_pnpm_runtime_volume name=%s", volume.name)
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if err := volume.close(cleanup, runner); err != nil {
			t.Error(err)
		}
	}()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	name := "haa-pnpm-native-" + session.String()
	project, err := createPNPMProjectVolume(ctx, runner, session.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("owned_pnpm_project_volume name=%s", project.name)
	defer func() {
		cleanup, done := resolverCleanupContext()
		defer done()
		if err := closePNPMProjectVolume(cleanup, runner, &project); err != nil {
			t.Error(err)
		}
	}()
	baseArguments, err := pnpmCreateArguments("none", nil, volume, project)
	if err != nil {
		t.Fatal(err)
	}
	arguments := append([]string{"create", "--name", name, "--label", "io.heliopause.pnpm-native=" + session.String()}, baseArguments[1:]...)
	created, err := runner.Output(ctx, "docker", arguments...)
	id := strings.TrimSpace(string(created))
	if err != nil || !exactObservationContainerID(id) {
		t.Fatal("PNPM native create outcome unavailable", name, err)
	}
	t.Logf("owned_pnpm_native_container name=%s id=%s", name, id)
	var trace TraceReader
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if _, err := runner.Output(cleanup, "docker", "rm", "--force", id); err != nil {
			t.Error("PNPM native cleanup unproven", err)
		}
		if trace != nil {
			_, limitation, diagnostic := collectTraceDiagnostic(cleanup, trace)
			t.Logf("pnpm_native_trace limitation=%s diagnostic=%+v", limitation, diagnostic)
			if limitation != "" {
				t.Error("PNPM native observation incomplete", limitation)
			}
			for _, kind := range []string{"process-exec-unexpected", "filesystem-outside-workspace", "honeytoken-access", "network-attempt"} {
				if diagnostic.KindCounts[kind] != 0 {
					t.Errorf("PNPM control probe contains %s=%d", kind, diagnostic.KindCounts[kind])
				}
			}
		}
	}()
	trace, err = startTrace(ctx, supervisor.Observer(), id, pnpmResolverProfile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Output(ctx, "docker", "start", id); err != nil {
		t.Fatal(err)
	}
	if err := awaitMountAnchors(ctx, supervisor.Observer(), id); err != nil {
		t.Fatal(err)
	}
	if err := awaitBoundaryHelper(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyTarget(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	if err := project.verifyAttachments(ctx, runner, map[string]bool{id: false}); err != nil {
		t.Fatal(err)
	}
	if err := tool.introduce(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyTarget(ctx, runner, id); err != nil {
		t.Fatal(err)
	}
	operation(ctx, runner, id)
}

// The same pinned payload is deliberately inspected as ARTIFACT through the
// existing GitHub ELF backend; its resolver CONTROL pin supplies no exception.
func TestPNPMActualReleaseArtifactProbe(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_PNPM_NATIVE_RUNTIME") != "1" {
		t.Fatal("requires declared ordinary-user isolated PNPM qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	tool, err := preparePNPMTool(ctx)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, run.String())
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "asset"), tool.body, 0600); err != nil {
		t.Fatal(err)
	}
	digest, err := domain.NewSHA256Digest(runtimeidentity.PNPMToolSHA256)
	if err != nil {
		t.Fatal(err)
	}
	source, err := domain.NewSourceID("github-release")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := domain.NewResolvedArtifactIdentity(source, "rahoney-heliopause", "v1.0.0-probe", runtimeidentity.PNPMNativeAssetName)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+run.String()+":github-release", uint64(len(tool.body)))
	if err != nil {
		t.Fatal(err)
	}
	request, err := domain.NewSandboxRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	backend, err := NewLinuxHeliopauseNativeBackendWithExecutor(root, executor, supervisor.Observer())
	if err != nil {
		t.Fatal(err)
	}
	result, err := backend.Execute(ctx, request)
	code, _ := result.LimitationCode()
	if err != nil || result.Status() != domain.SandboxCompleted {
		supervisor.observer.mu.Lock()
		fault := supervisor.observer.fault
		supervisor.observer.mu.Unlock()
		diagnostic := TraceDiagnostic{}
		var reason traceFault
		if errors.As(fault, &reason) {
			diagnostic.Reason = reason.TraceFaultReason()
		}
		var site interface{ TraceFaultSite() string }
		if errors.As(fault, &site) {
			diagnostic.FaultSite = site.TraceFaultSite()
		}
		var open interface{ TraceFaultOpen() FaultOpenDiagnostic }
		if errors.As(fault, &open) {
			diagnostic.FaultOpen = open.TraceFaultOpen()
		}
		t.Logf("pnpm_artifact_fault=%s", diagnostic)
		t.Fatalf("PNPM ARTIFACT probe=%s/%s observer_reason=%s error=%v", result.Status(), code, integrationObserverFaultReason(supervisor), err)
	}
	for _, observation := range result.Observations() {
		switch observation.Subject() {
		case "process-exec-unexpected", "filesystem-outside-workspace", "network-attempt", "honeytoken-access":
			t.Fatalf("PNPM artifact probe contains suspicious observation %s", observation.Subject())
		}
	}
	t.Logf("isolated_native_pnpm_artifact_probe=%s observations=%d", result.Status(), len(result.Observations()))
}
