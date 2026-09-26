package sandbox

import (
	"context"
	"io"
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
	if len(runner.inputCalls) != 1 || !sameStrings(runner.inputCalls[0].arguments, []string{"exec", "-i", "--user", boundaryBootstrapUser, "0123456789abcdef", boundaryHelperPath, boundaryOriginLaunchMode, "/bin/sh", "-ceu", "umask 077; cat > /tmp/artifact.tgz"}) || string(runner.input) != "fixture" {
		t.Fatalf("docker input stream = %#v/%q", runner.inputCalls, runner.input)
	}
}

func (r *recordingRunner) RunInput(_ context.Context, input io.Reader, binary string, arguments ...string) error {
	call := commandCall{binary: binary, arguments: append([]string(nil), arguments...)}
	r.inputCalls = append(r.inputCalls, call)
	r.timeline = append(r.timeline, call)
	r.input, _ = io.ReadAll(input)
	return nil
}
