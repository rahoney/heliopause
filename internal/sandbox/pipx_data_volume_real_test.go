//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// This test executes only fixed image sleep/touch, never Python or ZIP code.
// It qualifies Docker DATA mounts; gVisor/external observation remains separate.
func TestPipxActualReadonlyDataVolume(t *testing.T) {
	if os.Getenv("HELOX_PIPX_DATA_VOLUME") != "1" {
		t.Skip("explicit actual pipx DATA volume qualification")
	}
	input := os.Getenv("HELOX_PIPX_RUNTIME_ZIP")
	if !filepath.IsAbs(input) {
		t.Fatal("explicit selected ZIP required")
	}
	body, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if int64(len(body)) != runtimeidentity.PipxToolSize || hex.EncodeToString(sum[:]) != runtimeidentity.PipxToolSHA256 {
		t.Fatal("selected ZIP mismatch")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
	label := "io.heliopause.pipx-data-qualification"
	owned := map[string]bool{}
	var volume *pipxDataVolume
	remove := func(cleanup context.Context, id string) error {
		if !owned[id] {
			return errors.New("container is not qualification-owned")
		}
		b, e := runner.Output(cleanup, "docker", "inspect", "--format", "{{json .}}", id)
		var actual struct {
			ID     string `json:"Id"`
			Config struct {
				Image  string
				Labels map[string]string
			}
		}
		if e != nil || len(b) > 64<<10 || json.Unmarshal(b, &actual) != nil || actual.ID != id || actual.Config.Image != runtimeidentity.PythonImageReference || actual.Config.Labels[label] != owner {
			return errors.New("cleanup identity unavailable")
		}
		if _, e = runner.Output(cleanup, "docker", "rm", "--force", id); e != nil {
			return e
		}
		b, e = runner.Output(cleanup, "docker", "ps", "--all", "--no-trunc", "--format", "{{.ID}}")
		if e != nil || len(b) > 64<<10 {
			return errors.New("cleanup inventory unavailable")
		}
		for _, candidate := range strings.Fields(string(b)) {
			if candidate == id {
				return errors.New("owned container remains")
			}
		}
		delete(owned, id)
		return nil
	}
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		for id := range owned {
			if err := remove(cleanup, id); err != nil {
				t.Error(err)
			}
		}
		if volume != nil {
			if err := volume.close(cleanup, runner); err != nil {
				t.Error("DATA volume cleanup", err)
			}
		}
		if err := runner.Close(); err != nil {
			t.Error(err)
		}
		after, err := os.ReadFile(input)
		if err != nil || sha256.Sum256(after) != sum {
			t.Error("source changed")
		}
	}()
	volume, err = createPipxDataVolume(ctx, runner, owner)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("owned_volume=%s", volume.volume.name)
	create := func(role, mount string) string {
		t.Helper()
		user := "1000:1000"
		if role == "writer" {
			user = "0:0"
		}
		b, err := runner.Output(ctx, "docker", "create", "--pull", "never", "--name", "haa-pipx-data-"+role+"-"+owner,
			"--label", label+"="+owner, "--runtime", "runc", "--network", "none", "--read-only", "--cap-drop", "ALL",
			"--security-opt", "no-new-privileges", "--user", user, "--pids-limit", "64", "--memory", "256m", "--cpus", "1",
			"--mount", mount, "--entrypoint", "/bin/sleep", runtimeidentity.PythonImageReference, "120")
		id := strings.TrimSpace(string(b))
		if err != nil || !exactObservationContainerID(id) {
			t.Fatal("keeper create", role, err)
		}
		owned[id] = true
		if _, err := runner.Output(ctx, "docker", "start", id); err != nil {
			t.Fatal("keeper start", role, err)
		}
		t.Logf("owned_container role=%s id=%s", role, id)
		return id
	}
	rw, err := volume.preparationMount()
	if err != nil {
		t.Fatal(err)
	}
	writer := create("writer", rw)
	if err := volume.seed(ctx, runner, &pipxPreparedTool{body: body}, writer); err != nil {
		captured := nativeToolArchiveOutput{limit: runtimeidentity.PipxToolSize + 64<<10}
		diagnosticErr := runner.RunOutput(ctx, &captured, "docker", "cp", writer+":"+pipxToolGuestPath, "-")
		header, headerErr := tar.NewReader(bytes.NewReader(captured.body)).Next()
		t.Logf("seed diagnostic: copy=%v header_error=%v header=%+v transport_bytes=%d", diagnosticErr, headerErr, header, len(captured.body))
		t.Fatal("seed", err)
	}
	ro, err := volume.readerMount()
	if err != nil {
		t.Fatal(err)
	}
	anchor := create("anchor", ro)
	if err := volume.bindAnchor(ctx, runner, anchor); err != nil {
		t.Fatal("anchor", err)
	}
	if err := volume.seal(ctx, runner); err == nil {
		t.Fatal("writer still attached but sealed")
	}
	if err := remove(ctx, writer); err != nil {
		t.Fatal(err)
	}
	if err := volume.seal(ctx, runner); err != nil {
		t.Fatal("seal after writer removal", err)
	}
	if err := volume.verifyReader(ctx, runner, anchor); err != nil {
		t.Fatal("anchor readback", err)
	}
	reader := create("reader", ro)
	if err := volume.verifyReader(ctx, runner, reader); err != nil {
		t.Fatal("reader readback", err)
	}
	foreign := create("foreign", rw)
	if err := volume.verifyReader(ctx, runner, reader); err == nil {
		t.Fatal("foreign writable attachment accepted")
	}
	if err := remove(ctx, foreign); err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyReader(ctx, runner, reader); err != nil {
		t.Fatal("readback after foreign removal", err)
	}
	// Root still cannot create a file on the read-only mount. Only fixed image
	// touch is executed; arbitrary command or artifact execution is not allowed.
	_, err = runner.Output(ctx, "docker", "exec", "--user", "0:0", reader, "/usr/bin/touch", pipxRuntimeRoot+"/forbidden-write")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || !strings.Contains(string(exit.Stderr), "Read-only file system") {
		t.Fatalf("readonly write refusal not proven: %v", err)
	}
	if err := volume.verifyReader(ctx, runner, reader); err != nil {
		t.Fatal("readback after refused write", err)
	}
	if err := volume.close(ctx, runner); err == nil {
		t.Fatal("attached volume removed")
	}
	if err := remove(ctx, reader); err != nil {
		t.Fatal(err)
	}
	if err := volume.verifyReader(ctx, runner, anchor); err != nil {
		t.Fatal("retained anchor", err)
	}
	t.Log("PASS selected ZIP DATA: writer removed; live RO anchor/reader exact bytes; root write EROFS; source unchanged; no Python/ZIP execution")
}
