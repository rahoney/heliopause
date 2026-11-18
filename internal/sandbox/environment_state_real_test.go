//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// This tests DATA transport only, with no artifact input. The explicit runsc
// variant uses the existing boundary/admission and observer for the writer;
// the readonly reader remains a fixed, isolated runc DATA keeper. Neither
// variant qualifies artifact detection or the complete two-experiment flow.
func TestEnvironmentStateWorkspaceTransportPrimitive(t *testing.T) {
	mode := os.Getenv("HELOX_ENVIRONMENT_DATA_PRIMITIVE")
	if mode != "1" && mode != "runsc" {
		t.Skip("explicit owned workspace DATA primitive")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	runner, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	var commands CommandRunner = runner
	var supervisor *ObserverSupervisor
	var environmentObserver *SharedObserver
	var trace TraceReader
	if mode == "runsc" {
		supervisor = integrationObserverSupervisor(t)
		var ok bool
		environmentObserver, ok = supervisor.Observer().(*SharedObserver)
		if !ok {
			t.Fatal("DATA shared observer unavailable")
		}
		commands = admissionAwareRunner(runner)
	}
	owner := session.String()
	profile, err := newInspectionEnvironmentProfile(developmentEnvironment, session)
	if err != nil {
		t.Fatal(err)
	}
	var volume closureVolume
	owned := make(map[string]bool)
	volumeCreated := false
	remove := func(cleanup context.Context, id string) error {
		if !owned[id] {
			return errors.New("unowned DATA container")
		}
		body, err := commands.Output(cleanup, "docker", "inspect", "--format", `{{.Id}}|{{.Config.Image}}|{{index .Config.Labels "io.heliopause.observation-transaction"}}`, id)
		if err != nil || strings.TrimSpace(string(body)) != id+"|"+runtimeidentity.NodeImageReference+"|"+owner {
			return errors.New("DATA container cleanup identity unavailable")
		}
		if _, err := commands.Output(cleanup, "docker", "rm", "--force", id); err != nil {
			return err
		}
		remaining, err := commands.Output(cleanup, "docker", "ps", "--all", "--no-trunc", "--quiet", "--filter", "label="+closureVolumeLabel+"="+owner)
		if err != nil || len(remaining) > 4096 {
			return errors.New("DATA container absence unproven")
		}
		for _, candidate := range strings.Fields(string(remaining)) {
			if candidate == id {
				return errors.New("DATA container remains")
			}
		}
		delete(owned, id)
		return nil
	}
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 20*time.Second)
		defer done()
		for id := range owned {
			if err := remove(cleanup, id); err != nil {
				t.Error(err)
			}
		}
		if trace != nil {
			_, limitation, diagnostic := collectTraceDiagnostic(cleanup, trace)
			trace = nil
			if limitation != "" {
				t.Errorf("DATA observer drain incomplete: %s %s", limitation, diagnostic.String())
			}
		}
		if volumeCreated {
			if err := volume.verify(cleanup, commands); err != nil {
				t.Error(err)
			} else if len(owned) != 0 {
				t.Error("DATA volume retained for uncertain containers")
			} else if _, err := commands.Output(cleanup, "docker", "volume", "rm", volume.name); err != nil {
				t.Error(err)
			}
		}
		if supervisor != nil {
			if err := supervisor.Close(); err != nil {
				t.Error(err)
			}
		}
		if err := runner.Close(); err != nil {
			t.Error(err)
		}
	}()
	volume, err = createEnvironmentWorkspace(ctx, commands, session, profile, 64<<10)
	if volume.name != "" {
		t.Logf("DATA owned volume allocation=%s", volume.name)
		volumeCreated = true
	}
	if err != nil {
		t.Fatal("DATA owned workspace create failed", err)
	}
	create := func(role string, readonly bool) string {
		t.Helper()
		mount, err := volume.mountArgument(readonly)
		if err != nil {
			t.Fatal(err)
		}
		name := "haa-env-data-" + role + "-" + owner
		if mode == "runsc" && !readonly {
			name = "heliopause-" + owner
		}
		t.Logf("DATA owned container allocation=%s", name)
		args := []string{"create", "--pull", "never", "--name", name, "--label", closureVolumeLabel + "=" + owner, "--label", closureManifestLabel + "=" + volume.manifestID, "--runtime", "runc", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--user", "1000:1000", "--pids-limit", "32", "--memory", "64m", "--cpus", "1", "--mount", mount}
		args = append(args, profile.environment...)
		args = append(args, "--entrypoint", "/bin/sleep", runtimeidentity.NodeImageReference, "90")
		if mode == "runsc" && !readonly {
			args, err = environmentContainerArguments(session, profile, volume, false)
			if err != nil {
				t.Fatal(err)
			}
		}
		body, err := commands.Output(ctx, "docker", args...)
		id := strings.TrimSpace(string(body))
		if err != nil || !exactObservationContainerID(id) {
			t.Fatal("DATA container create failed", err)
		}
		owned[id] = true
		if mode == "runsc" && !readonly {
			budget, err := newObservationTraceLedger("npm-lifecycle")
			if err != nil {
				t.Fatal(err)
			}
			trace, err = environmentObserver.StartEnvironmentProfile(ctx, id, false, budget)
			if err != nil {
				t.Fatal("DATA observer start", err)
			}
		}
		if _, err := commands.Output(ctx, "docker", "start", id); err != nil {
			t.Fatal(err)
		}
		if mode == "runsc" && !readonly {
			if err := awaitBoundaryHelper(ctx, commands, id); err != nil {
				t.Fatal("DATA boundary readiness", err)
			}
			if err := environmentObserver.AwaitMountAnchors(ctx, id); err != nil {
				t.Fatal("DATA mount readiness", err)
			}
		}
		if err := volume.verifyContainerMount(ctx, commands, id, readonly); err != nil {
			t.Fatal(err)
		}
		return id
	}
	writer := create("writer", false)
	payload, err := profile.payload()
	if err != nil {
		t.Fatal(err)
	}
	if err := commands.(inputCommandRunner).RunInput(ctx, bytes.NewReader(payload), "docker", "cp", "--archive", "-", writer+":/tmp"); err != nil {
		t.Fatal("DATA synthetic profile introduction", err)
	}
	write := func(files map[string]string) {
		t.Helper()
		var data bytes.Buffer
		archive := tar.NewWriter(&data)
		for name, body := range files {
			if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(body)), Typeflag: tar.TypeReg, Uid: 1000, Gid: 1000}); err != nil {
				t.Fatal(err)
			}
			if _, err := archive.Write([]byte(body)); err != nil {
				t.Fatal(err)
			}
		}
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
		if err := commands.(inputCommandRunner).RunInput(ctx, &data, "docker", "cp", "--archive", "-", writer+":/tmp"); err != nil {
			t.Fatal("DATA introduction", err)
		}
	}
	write(map[string]string{"state.txt": "before", "old.txt": "remove"})
	// A backing tmpfs option alone does not prove guest noexec. A tiny
	// shebang script tests direct execution without loader/mapping ambiguity.
	probeFiles := 0
	if mode == "runsc" {
		write(map[string]string{"noexec-probe": "#!/bin/sh\nexit 99\n"})
		args := boundaryExecArguments(writer, boundaryLaunchMode, "/bin/sh", "-ceu", "chmod 0700 /tmp/noexec-probe; set +e; /tmp/noexec-probe; code=$?; printf 'DATA_EXEC_EXIT=%s\\n' \"$code\"; [ \"$code\" = 126 ]")
		outputBody, err := commands.Output(ctx, "docker", args...)
		t.Logf("DATA fixed execution status=%q", string(outputBody[:min(len(outputBody), 1024)]))
		if err != nil {
			var native *exec.ExitError
			if errors.As(err, &native) {
				t.Logf("DATA fixed execution stderr=%q", string(native.Stderr[:min(len(native.Stderr), 4096)]))
			}
			t.Fatal("DATA guest noexec refusal unproven", err)
		}
		probeFiles = 1
	}
	reader := create("reader", true)
	if err := volume.verifyAttachments(ctx, commands, map[string]bool{writer: false, reader: true}); err != nil {
		t.Fatal(err)
	}
	capture := func() environmentState {
		t.Helper()
		state, err := captureEnvironmentState(ctx, commands, reader, environmentStateLimits{entries: 64, bytes: 64 << 10, archiveBytes: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		return state
	}
	first := capture()
	if err := profile.verifyPlacement(first); err != nil {
		t.Fatal("DATA synthetic profile bytes/mode/ownership verification", err)
	}
	beforeHash := sha256.Sum256([]byte("before"))
	if len(first.files) != len(profile.fixtures)+3+probeFiles || first.files["state.txt"].digest != hex.EncodeToString(beforeHash[:]) || first.files["state.txt"].mode != 0600 {
		t.Fatal("DATA baseline content/mode differs")
	}
	write(map[string]string{"state.txt": "after", "new.txt": "created"})
	mutate := []string{"exec", "--user", "1000:1000", writer, "/bin/sh", "-ceu", "rm /tmp/old.txt; chmod 0640 /tmp/state.txt"}
	if mode == "runsc" {
		mutate = boundaryExecArguments(writer, boundaryLaunchMode, "/bin/sh", "-ceu", "rm /tmp/old.txt; chmod 0640 /tmp/state.txt")
	}
	if _, err := commands.Output(ctx, "docker", mutate...); err != nil {
		t.Fatal("DATA fixed content mutation", err)
	}
	if err := remove(ctx, writer); err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyAttachments(ctx, commands, map[string]bool{reader: true}); err != nil {
		t.Fatal(err)
	}
	if trace != nil {
		observed, limitation, diagnostic := collectTraceDiagnostic(ctx, trace)
		trace = nil
		if limitation != "" || len(retainedSuspiciousTraceFacts(diagnostic.KindCounts)) != 0 {
			t.Fatalf("DATA observer completion: %s %s", limitation, diagnostic.String())
		}
		t.Logf("DATA complete observer facts=%d %s", len(observed), diagnostic.String())
	}
	second := capture()
	afterHash := sha256.Sum256([]byte("after"))
	changes, err := compareEnvironmentStates(first, second)
	if err != nil || len(second.files) != len(profile.fixtures)+3+probeFiles || second.files["state.txt"].digest != hex.EncodeToString(afterHash[:]) || second.files["state.txt"].mode != 0640 || changes != (environmentChanges{created: 1, deleted: 1, modified: 1}) {
		t.Fatalf("DATA final comparison differs after writer removal: %+v, %v", changes, err)
	}
	// The reader is an inert runc DATA keeper in both variants. No artifact
	// command is admitted through this fixed readonly-write negative.
	_, err = runner.Output(ctx, "docker", "exec", "--user", "1000:1000", reader, "/usr/bin/touch", "/tmp/forbidden-write")
	var exited *exec.ExitError
	if !errors.As(err, &exited) || exited.ExitCode() != 1 {
		t.Fatal("readonly DATA write refusal unproven", err)
	}
	if err := volume.verifyAttachments(ctx, commands, map[string]bool{reader: true}); err != nil {
		t.Fatal(err)
	}
	t.Logf("DATA transport: baseline/final content+mode/create/delete, readonly keeper after writer removal; artifact code=0; writer runtime=%s; two-experiment/detection NOT_TESTED", mode)
}
