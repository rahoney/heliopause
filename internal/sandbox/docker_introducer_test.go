package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDockerArtifactIntroducerCopiesOnlyControlledTarball(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "tarball.tgz")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	introducer, err := NewDockerArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	if err := introducer.Introduce(context.Background(), "0123456789abcdef", sandboxRequest(t).Artifact()); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 || !sameStrings(runner.calls[0].arguments, []string{"cp", path, "0123456789abcdef:/tmp/artifact.tgz"}) {
		t.Fatalf("docker cp = %#v", runner.calls)
	}
}
