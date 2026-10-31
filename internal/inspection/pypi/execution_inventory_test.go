package pypi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func consoleIntakeFixture(t *testing.T) (*StaticInspector, domain.AcquiredArtifact, string) {
	t.Helper()
	root := t.TempDir()
	artifact := prerequisiteFixtureWheel(t, root, "example", "py3-none-any", map[string]string{"example-1.0.dist-info/entry_points.txt": "[console_scripts]\nexample = example:main\n"}, "")
	inspector, err := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, strings.Split(artifact.ContentHandle(), ":")[1])
	return inspector, artifact, directory
}

func TestConsoleIntakeInventoryBindsActualWheelAndOwnedSnapshot(t *testing.T) {
	inspector, artifact, directory := consoleIntakeFixture(t)
	inventory, err := inspector.ReadConsoleExecutionInventory(context.Background(), artifact)
	if err != nil || !inventory.Valid() || inventory.Artifact() != artifact {
		t.Fatal("controlled canonical inventory unavailable", err)
	}
	entry, err := artifactpypi.SelectConsoleExecutionEntry(inventory, "")
	if err != nil || entry.Command() != "example" || entry.EntryPoint().Attr != "main" || !entry.Matches(inventory) {
		t.Fatal("exact console declaration lost", err)
	}
	run, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer run.Close()
	snapshot, err := readConsoleIntakeSnapshot(context.Background(), run, "wheel.whl", artifactpypi.DefaultWheelLimits().MaxCompressed, artifact.SizeBytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "wheel.whl"), []byte("changed intake"), 0600); err != nil {
		t.Fatal(err)
	}
	measured, err := artifactpypi.InspectConsoleExecutionWheel(artifact, bytes.NewReader(snapshot), "example-1.0-py3-none-any.whl", inspector.target, artifactpypi.DefaultWheelLimits())
	if err != nil || !entry.Matches(measured) {
		t.Fatal("owned snapshot changed with original file", err)
	}
	if next, err := inspector.ReadConsoleExecutionInventory(context.Background(), artifact); err == nil || next.Valid() {
		t.Fatal("changed current intake reused old selection")
	}
}

func TestConsoleIntakeInventoryRejectsUncontrolledAndChangedFiles(t *testing.T) {
	for _, name := range []string{"archive symlink", "filename symlink", "Run symlink", "root symlink", "archive directory", "filename directory", "oversized filename", "foreign filename", "wrong archive digest"} {
		t.Run(name, func(t *testing.T) {
			inspector, artifact, directory := consoleIntakeFixture(t)
			archive := filepath.Join(directory, "wheel.whl")
			filename := filepath.Join(directory, "filename")
			switch name {
			case "archive symlink", "filename symlink":
				path := archive
				if name == "filename symlink" {
					path = filename
				}
				outside := filepath.Join(t.TempDir(), "original")
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(outside, body, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, path); err != nil {
					t.Fatal(err)
				}
			case "Run symlink", "root symlink":
				path := directory
				if name == "root symlink" {
					path = inspector.intakeRoot
				}
				outside := filepath.Join(t.TempDir(), "moved")
				if err := os.Rename(path, outside); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, path); err != nil {
					t.Fatal(err)
				}
			case "archive directory", "filename directory":
				path := archive
				if name == "filename directory" {
					path = filename
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "oversized filename", "foreign filename":
				body := "other-1.0-py3-none-any.whl"
				if name == "oversized filename" {
					body = strings.Repeat("a", 513)
				}
				if err := os.WriteFile(filename, []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			case "wrong archive digest":
				body, err := os.ReadFile(archive)
				if err != nil {
					t.Fatal(err)
				}
				body[len(body)/2] ^= 1
				if err := os.WriteFile(archive, body, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if inventory, err := inspector.ReadConsoleExecutionInventory(context.Background(), artifact); err == nil || inventory.Valid() {
				t.Fatal("uncontrolled source accepted")
			}
		})
	}
}

func TestConsoleIntakeInventoryPreservesCancellationAndRejectsInvalidRequests(t *testing.T) {
	inspector, artifact, _ := consoleIntakeFixture(t)
	cause := errors.New("caller stopped console measurement")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if inventory, err := inspector.ReadConsoleExecutionInventory(ctx, artifact); inventory.Valid() || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal("original caller cause lost", err)
	}
	ctx, cancel = context.WithCancelCause(context.Background())
	reader := consoleContextReader{ctx, consoleCancelReader{cancel, cause}}
	if _, err := reader.Read(make([]byte, 1)); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal("cancellation during source read lost", err)
	}
	var absent *StaticInspector
	var missing context.Context
	for _, row := range []struct {
		inspector *StaticInspector
		ctx       context.Context
		artifact  domain.AcquiredArtifact
	}{{absent, context.Background(), artifact}, {inspector, missing, artifact}, {inspector, context.Background(), domain.AcquiredArtifact{}}} {
		if inventory, err := row.inspector.ReadConsoleExecutionInventory(row.ctx, row.artifact); err == nil || inventory.Valid() {
			t.Fatal("invalid request accepted")
		}
	}
	invalid, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(artifact.Identity(), artifact.Digest(), "intake:invalid:wheel", artifact.SizeBytes(), "sha256:"+artifact.Digest().String())
	if err != nil {
		t.Fatal(err)
	}
	if inventory, err := inspector.ReadConsoleExecutionInventory(context.Background(), invalid); err == nil || inventory.Valid() {
		t.Fatal("uncontrolled handle accepted")
	}
}

type consoleCancelReader struct {
	cancel context.CancelCauseFunc
	cause  error
}

func (r consoleCancelReader) Read([]byte) (int, error) {
	r.cancel(r.cause)
	return 0, io.EOF
}
