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

// This tests the Docker DATA transport only. It executes fixed image tools in
// runc, without artifact input or observer changes. It is not gVisor/detection
// qualification and does not authorize a new production mount/profile.
func TestEnvironmentStateWorkspaceTransportPrimitive(t *testing.T) {
	if os.Getenv("HELOX_ENVIRONMENT_DATA_PRIMITIVE") != "1" {
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
	owner := session.String()
	profile, err := newInspectionEnvironmentProfile(developmentEnvironment, session)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("environment-state-DATA-v1"))
	volume := closureVolume{transaction: owner, manifestID: hex.EncodeToString(digest[:]), capacity: 64 << 10, projectBuildDestination: "/tmp"}
	volume.name, err = closureVolumeName(owner)
	if err != nil {
		t.Fatal(err)
	}
	owned := make(map[string]bool)
	volumeCreated := false
	remove := func(cleanup context.Context, id string) error {
		if !owned[id] {
			return errors.New("unowned DATA container")
		}
		body, err := runner.Output(cleanup, "docker", "inspect", "--format", `{{.Id}}|{{.Config.Image}}|{{index .Config.Labels "io.heliopause.observation-transaction"}}`, id)
		if err != nil || strings.TrimSpace(string(body)) != id+"|"+runtimeidentity.NodeImageReference+"|"+owner {
			return errors.New("DATA container cleanup identity unavailable")
		}
		if _, err := runner.Output(cleanup, "docker", "rm", "--force", id); err != nil {
			return err
		}
		remaining, err := runner.Output(cleanup, "docker", "ps", "--all", "--no-trunc", "--quiet", "--filter", "label="+closureVolumeLabel+"="+owner)
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
		if volumeCreated {
			if err := volume.verify(cleanup, runner); err != nil {
				t.Error(err)
			} else if len(owned) != 0 {
				t.Error("DATA volume retained for uncertain containers")
			} else if _, err := runner.Output(cleanup, "docker", "volume", "rm", volume.name); err != nil {
				t.Error(err)
			}
		}
		if err := runner.Close(); err != nil {
			t.Error(err)
		}
	}()
	before, err := runner.Output(ctx, "docker", "volume", "ls", "--quiet", "--filter", "name=^"+volume.name+"$")
	if err != nil || len(strings.TrimSpace(string(before))) != 0 {
		t.Fatal("DATA volume absence unproven", err)
	}
	t.Logf("DATA owned volume allocation=%s", volume.name)
	created, err := runner.Output(ctx, "docker", "volume", "create", "--driver", "local", "--opt", "type=tmpfs", "--opt", "device=tmpfs", "--opt", "o="+volume.tmpfsOptions(), "--label", closureVolumeLabel+"="+owner, "--label", closureManifestLabel+"="+volume.manifestID, volume.name)
	if err != nil || strings.TrimSpace(string(created)) != volume.name {
		t.Fatal("DATA volume create failed", err)
	}
	volumeCreated = true
	if err := volume.verify(ctx, runner); err != nil {
		t.Fatal(err)
	}
	create := func(role string, readonly bool) string {
		t.Helper()
		mount, err := volume.mountArgument(readonly)
		if err != nil {
			t.Fatal(err)
		}
		name := "haa-env-data-" + role + "-" + owner
		t.Logf("DATA owned container allocation=%s", name)
		args := []string{"create", "--pull", "never", "--name", name, "--label", closureVolumeLabel + "=" + owner, "--label", closureManifestLabel + "=" + volume.manifestID, "--runtime", "runc", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--user", "1000:1000", "--pids-limit", "32", "--memory", "64m", "--cpus", "1", "--mount", mount}
		args = append(args, profile.environment...)
		args = append(args, "--entrypoint", "/bin/sleep", runtimeidentity.NodeImageReference, "90")
		body, err := runner.Output(ctx, "docker", args...)
		id := strings.TrimSpace(string(body))
		if err != nil || !exactObservationContainerID(id) {
			t.Fatal("DATA container create failed", err)
		}
		owned[id] = true
		if _, err := runner.Output(ctx, "docker", "start", id); err != nil {
			t.Fatal(err)
		}
		if err := volume.verifyContainerMount(ctx, runner, id, readonly); err != nil {
			t.Fatal(err)
		}
		return id
	}
	writer := create("writer", false)
	payload, err := profile.payload()
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.RunInput(ctx, bytes.NewReader(payload), "docker", "exec", "-i", "--user", "1000:1000", writer, "/bin/tar", "-xf", "-", "-C", "/tmp"); err != nil {
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
		if err := runner.RunInput(ctx, &data, "docker", "exec", "-i", "--user", "1000:1000", writer, "/bin/tar", "-xf", "-", "-C", "/tmp"); err != nil {
			t.Fatal("DATA introduction", err)
		}
	}
	write(map[string]string{"state.txt": "before", "old.txt": "remove"})
	reader := create("reader", true)
	if err := volume.verifyAttachments(ctx, runner, map[string]bool{writer: false, reader: true}); err != nil {
		t.Fatal(err)
	}
	capture := func() environmentState {
		t.Helper()
		state, err := captureEnvironmentState(ctx, runner, reader, environmentStateLimits{entries: 64, bytes: 64 << 10, archiveBytes: 1 << 20})
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
	if len(first.files) != len(profile.fixtures)+3 || first.files["state.txt"].digest != hex.EncodeToString(beforeHash[:]) || first.files["state.txt"].mode != 0600 {
		t.Fatal("DATA baseline content/mode differs")
	}
	write(map[string]string{"state.txt": "after", "new.txt": "created"})
	if _, err := runner.Output(ctx, "docker", "exec", "--user", "1000:1000", writer, "/bin/rm", "/tmp/old.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Output(ctx, "docker", "exec", "--user", "1000:1000", writer, "/bin/chmod", "0640", "/tmp/state.txt"); err != nil {
		t.Fatal(err)
	}
	if err := remove(ctx, writer); err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyAttachments(ctx, runner, map[string]bool{reader: true}); err != nil {
		t.Fatal(err)
	}
	second := capture()
	afterHash := sha256.Sum256([]byte("after"))
	changes, err := compareEnvironmentStates(first, second)
	if err != nil || len(second.files) != len(profile.fixtures)+3 || second.files["state.txt"].digest != hex.EncodeToString(afterHash[:]) || second.files["state.txt"].mode != 0640 || changes != (environmentChanges{created: 1, deleted: 1, modified: 1}) {
		t.Fatalf("DATA final comparison differs after writer removal: %+v, %v", changes, err)
	}
	_, err = runner.Output(ctx, "docker", "exec", "--user", "1000:1000", reader, "/usr/bin/touch", "/tmp/forbidden-write")
	var exited *exec.ExitError
	if !errors.As(err, &exited) || exited.ExitCode() != 1 {
		t.Fatal("readonly DATA write refusal unproven", err)
	}
	if err := volume.verifyAttachments(ctx, runner, map[string]bool{reader: true}); err != nil {
		t.Fatal(err)
	}
	t.Log("DATA transport: baseline/final content+mode/create/delete, readonly keeper after writer removal; artifact code=0; gVisor/observer NOT_TESTED")
}
