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
	"strings"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type uvTransportQualificationRunner struct {
	*hosttool.Executor
	calls [][]string
}

func (r *uvTransportQualificationRunner) allow(name string, args []string) error {
	if name != "docker" || len(args) == 0 {
		return errors.New("data-only qualification command rejected")
	}
	switch args[0] {
	case "create", "rm", "cp":
	case "image", "container":
		if len(args) < 2 || args[1] != "inspect" {
			return errors.New("data-only inspection required")
		}
	default:
		return errors.New("UV supply qualification forbids container execution")
	}
	r.calls = append(r.calls, append([]string(nil), args...))
	return nil
}
func (r *uvTransportQualificationRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if err := r.allow(name, args); err != nil {
		return nil, err
	}
	return r.Executor.Output(ctx, name, args...)
}
func (r *uvTransportQualificationRunner) RunInput(ctx context.Context, input io.Reader, name string, args ...string) error {
	if err := r.allow(name, args); err != nil {
		return err
	}
	return r.Executor.RunInput(ctx, input, name, args...)
}
func (r *uvTransportQualificationRunner) RunOutput(ctx context.Context, output io.Writer, name string, args ...string) error {
	if err := r.allow(name, args); err != nil {
		return err
	}
	return r.Executor.RunOutput(ctx, output, name, args...)
}

func TestUVActualDockerToolTransportWithoutExecution(t *testing.T) {
	if os.Getenv("HELOX_UV_SUPPLY_DOCKER") != "1" {
		t.Fatal("declare HELOX_UV_SUPPLY_DOCKER=1 in the prepared data-only namespace")
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
	runner := &uvTransportQualificationRunner{Executor: executor}
	defer func() {
		for _, args := range runner.calls {
			if args[0] == "create" {
				for i, a := range args {
					if a == "--name" {
						t.Logf("owned_data_container name=%s", args[i+1])
					}
				}
			}
			if args[0] == "rm" {
				t.Logf("removed_owned_data_container id=%s", args[1])
			}
		}
	}()
	tool, err := prepareUVTool(ctx, runner)
	if err != nil {
		t.Fatal(err)
	}
	id, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	owner := id.String()
	name := "haa-uv-transport-target-" + owner
	created, err := runner.Output(ctx, "docker", "create", "--pull", "never", "--name", name, "--label", "io.heliopause.uv-transport-target="+owner, "--network", "none", "--cap-drop", "ALL", runtimeidentity.PythonImageReference)
	target := strings.TrimSpace(string(created))
	if err != nil || !exactObservationContainerID(target) {
		t.Fatal("target create outcome unavailable", name, err)
	}
	defer func() {
		cleanup, done := resolverCleanupContext()
		defer done()
		body, err := runner.Output(cleanup, "docker", "container", "inspect", target, "--format", "{{json .}}")
		var value struct {
			ID     string `json:"Id"`
			Config struct {
				Image  string
				Labels map[string]string
			}
			State struct{ Status, StartedAt string }
		}
		if err != nil || len(body) > 64<<10 || json.Unmarshal(body, &value) != nil || value.ID != target || value.Config.Image != runtimeidentity.PythonImageReference || value.Config.Labels["io.heliopause.uv-transport-target"] != owner || value.State.Status != "created" || value.State.StartedAt != "0001-01-01T00:00:00Z" {
			t.Error("data-only target cleanup unproven", name, err)
			return
		}
		if _, err := runner.Output(cleanup, "docker", "rm", target); err != nil {
			t.Error("remove exact data-only target", err)
		}
	}()
	// This stopped, writable-root fixture qualifies only byte transport and
	// root file metadata. It is not a running tmpfs/admission qualification.
	var directory bytes.Buffer
	archive := tar.NewWriter(&directory)
	if err := archive.WriteHeader(&tar.Header{Name: "haa-runtime", Typeflag: tar.TypeDir, Mode: 0755, Uid: 0, Gid: 0}); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runner.RunInput(ctx, bytes.NewReader(directory.Bytes()), "docker", "cp", "--archive", "-", target+":/"); err != nil {
		t.Fatal(err)
	}
	if err := tool.introduce(ctx, runner, target); err != nil {
		t.Fatal(err)
	}
	for _, args := range runner.calls {
		if args[0] == "start" || args[0] == "run" || args[0] == "exec" {
			t.Fatal("unexpected execution", args)
		}
	}
	t.Logf("actual signed-native data transport: bytes=%d source_mode=0755 target_mode=0555 owner=0:0; no container/UV execution; runtime admission remains", len(tool.body))
}

var _ streamingDockerOutput = (*uvTransportQualificationRunner)(nil)
var _ interface {
	RunInput(context.Context, io.Reader, string, ...string) error
} = (*uvTransportQualificationRunner)(nil)
