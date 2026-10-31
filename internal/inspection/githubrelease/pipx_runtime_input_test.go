package githubrelease

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func pipxInputArtifact(t *testing.T, source, name, variant, digest, handle string, size uint64) domain.AcquiredArtifact {
	t.Helper()
	sourceID, err := domain.NewSourceID(source)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := domain.NewResolvedArtifactIdentity(sourceID, name, "v9.8.7", variant)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := domain.NewSHA256Digest(digest)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, hash, handle, size, "sha256:"+digest)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func pipxInputFixture(t *testing.T, body []byte) (*StaticInspector, domain.AcquiredArtifact, string) {
	t.Helper()
	root := t.TempDir()
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, run.String())
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(dir, "asset")
	if err := os.WriteFile(filename, body, 0600); err != nil {
		t.Fatal(err)
	}
	reader, err := NewStaticInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	artifact := pipxInputArtifact(t, "github-release", "rahoney-heliopause", "pipx-runtime-linux-amd64.zip", runtimeidentity.PipxToolSHA256, "intake:"+run.String()+":github-release", runtimeidentity.PipxToolSize)
	return reader, artifact, filename
}

func TestPipxRuntimeInputRejectsSubstitutionAndInvalidIntake(t *testing.T) {
	for _, mode := range []string{"source", "name", "variant", "digest", "declared-size", "handle", "symlink", "actual-size", "changed-bytes"} {
		t.Run(mode, func(t *testing.T) {
			reader, artifact, filename := pipxInputFixture(t, make([]byte, runtimeidentity.PipxToolSize))
			source, name, variant, digest, handle, size := "github-release", "rahoney-heliopause", "pipx-runtime-linux-amd64.zip", runtimeidentity.PipxToolSHA256, artifact.ContentHandle(), uint64(runtimeidentity.PipxToolSize)
			expected := "subject differs"
			switch mode {
			case "source":
				source = "pypi"
			case "name":
				name = "pipx"
			case "variant":
				variant = "pipx.whl"
			case "digest":
				digest = strings.Repeat("0", 64)
			case "declared-size":
				size++
			case "handle":
				handle = "invalid"
				expected = "handle is invalid"
			case "symlink":
				if err := os.Rename(filename, filename+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filename+".original", filename); err != nil {
					t.Fatal(err)
				}
				expected = "asset is invalid"
			case "actual-size":
				if err := os.Truncate(filename, 1); err != nil {
					t.Fatal(err)
				}
				expected = "asset is invalid"
			case "changed-bytes":
				expected = "digest differs"
			}
			artifact = pipxInputArtifact(t, source, name, variant, digest, handle, size)
			input, err := reader.readPipxRuntimeInput(context.Background(), artifact)
			if err == nil || !strings.Contains(err.Error(), expected) || input.plan.Valid() || input.artifact.ContentHandle() != "" {
				t.Fatal("invalid input accepted or wrong refusal", err)
			}
		})
	}
}

func TestPipxRuntimeInputPreservesCancellation(t *testing.T) {
	reader, artifact, _ := pipxInputFixture(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if input, err := reader.readPipxRuntimeInput(ctx, artifact); !errors.Is(err, context.Canceled) || input.plan.Valid() {
		t.Fatal("cancellation lost", err)
	}
	var absent *StaticInspector
	if _, err := absent.readPipxRuntimeInput(context.Background(), artifact); err == nil {
		t.Fatal("nil reader accepted")
	}
}
