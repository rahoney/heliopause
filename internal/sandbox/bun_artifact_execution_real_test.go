//go:build linux && realcorpus

package sandbox

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// Structural source/SRI fixture, with actual static inventory and complete
// layout binding. This is an execution boundary, not registry authentication.
func bunArtifactExecutionLayout(t *testing.T, failure bool) (artifactnpm.BunInspectionLayout, domain.AcquiredArtifact, string) {
	t.Helper()
	_, input, _ := bunPreparationInputFixture(t, BunResolverRuntimeIdentity())
	subject := input.Artifacts()[0]
	name := subject.Identity().Name()
	root := "node_modules/" + name
	manifest, err := json.Marshal(map[string]any{"name": name, "version": subject.Identity().Version(), "scripts": map[string]string{"install": "bun " + bunProjectDirectory + "/" + root + "/probe.js"}})
	if err != nil {
		t.Fatal(err)
	}
	script := []byte(`if (typeof Bun !== "object" || Bun.version !== "` + runtimeidentity.BunVersion + `") throw new Error("fixed Bun runtime missing"); console.log("native-Bun:" + Bun.version);`)
	if failure {
		script = append(script, []byte(`process.exit(37);`)...)
	}
	bodies := map[string][]byte{"package.json": manifest, "probe.js": script}
	var entries []artifactnpm.ArchiveEntry
	files := []pnpmPayloadTestFile{{header: tar.Header{Name: "node_modules", Typeflag: tar.TypeDir, Mode: 0o755, Uid: 1000, Gid: 1000}}, {header: tar.Header{Name: root, Typeflag: tar.TypeDir, Mode: 0o755, Uid: 1000, Gid: 1000}}}
	for _, path := range []string{"package.json", "probe.js"} {
		body := bodies[path]
		sum := sha256.Sum256(body)
		digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, artifactnpm.ArchiveEntry{Path: path, Size: int64(len(body)), Mode: 0o644, Digest: digest})
		files = append(files, pnpmPayloadTestFile{header: tar.Header{Name: root + "/" + path, Typeflag: tar.TypeReg, Mode: 0o644, Uid: 1000, Gid: 1000, Size: int64(len(body))}, body: body})
	}
	inventory, err := artifactnpm.NewArchiveInventory(subject, entries, manifest)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := artifactnpm.ParseBunLayoutArchive(context.Background(), pnpmPayloadTestTar(t, files))
	if err != nil {
		t.Fatal(err)
	}
	layout, err := input.BindLayout(context.Background(), []artifactnpm.ArchiveInventory{inventory}, archive)
	if err != nil {
		t.Fatal("whole controlled Bun layout", err)
	}
	instances, err := input.InstancesFor(subject)
	if err != nil || len(instances) != 1 {
		t.Fatal("controlled exact instance", err)
	}
	return layout, subject, instances[0].Key
}

// Capture only the exact controller command's bounded output as a test oracle.
// It grants no completion, role, evidence or Policy authority.
type bunArtifactExecutionCapture struct {
	CommandRunner
	output string
}

func (r *bunArtifactExecutionCapture) Output(ctx context.Context, binary string, args ...string) ([]byte, error) {
	body, err := r.CommandRunner.Output(ctx, binary, args...)
	for _, arg := range args {
		if arg == bunTargetLifecycleCommand || arg == bunRuntimeTargetLifecycleCommand {
			r.output = string(body)
		}
	}
	return body, err
}
func (r *bunArtifactExecutionCapture) RunInput(ctx context.Context, input io.Reader, binary string, args ...string) error {
	return r.CommandRunner.(inputCommandRunner).RunInput(ctx, input, binary, args...)
}
func (r *bunArtifactExecutionCapture) RunOutput(ctx context.Context, output io.Writer, binary string, args ...string) error {
	return r.CommandRunner.(streamingDockerOutput).RunOutput(ctx, output, binary, args...)
}

func TestBunActualArtifactExecutionBoundary(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_BUN_ARTIFACT_EXECUTION") != "1" {
		t.Fatal("declare ordinary-user controlled Bun Artifact execution")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, failure := range []bool{false, true} {
		label := "normal"
		if failure {
			label = "script-failure"
		}
		if !t.Run(label, func(t *testing.T) {
			layout, subject, instance := bunArtifactExecutionLayout(t, failure)
			backend, err := NewBackend(executor, &recordingIntroducer{}, supervisor.Observer(), integrationCapabilityProbe(executor))
			if err != nil {
				t.Fatal(err)
			}
			capture := &bunArtifactExecutionCapture{CommandRunner: backend.runner}
			backend.runner = capture
			result, err := backend.ExecuteBunRuntimeInstance(ctx, layout, subject, instance)
			if shared, ok := supervisor.Observer().(*SharedObserver); ok {
				shared.mu.Lock()
				fault := shared.fault
				shared.mu.Unlock()
				t.Logf("existing_observer_fault=%#v", fault)
			}
			t.Logf("native_Bun_artifact failure=%t status=%v cause=%v observations=%+v command_output=%s", failure, result.Status(), err, result.Observations(), capture.output)
			if !strings.Contains(capture.output, "native-Bun:"+runtimeidentity.BunVersion) {
				t.Fatal("fixed native Bun JS body not reached", err)
			}
			if failure {
				if err == nil || result.Status() != domain.SandboxIncomplete {
					t.Fatal("failed JS completed", err)
				}
				var exit interface{ ExitCode() int }
				if !errors.As(err, &exit) || exit.ExitCode() != 37 {
					t.Fatal("original native JS exit status lost", err)
				}
			} else if err != nil || result.Status() != domain.SandboxCompleted {
				t.Fatal("native Bun execution incomplete", err)
			}
			artifactExecs := uint64(0)
			for _, fact := range result.Observations() {
				switch fact.Subject() {
				case "process-exec-unexpected":
					artifactExecs += fact.Count()
				case "filesystem-outside-workspace", "honeytoken-access", "network-attempt":
					t.Errorf("Bun runtime baseline has %s=%d", fact.Subject(), fact.Count())
				}
			}
			if artifactExecs < 2 {
				t.Fatal("native Bun or script shell gained trusted attribution", artifactExecs)
			}
			t.Logf("native_Bun_actual_JS=true fixed_version=%s Artifact_process_observations=%d Policy=NOT_RUN Host_install=NOT_RUN", runtimeidentity.BunVersion, artifactExecs)
		}) {
			return
		}
	}
}
