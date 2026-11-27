package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPipxCandidateRefusesInvalidPathsAndUnverifiedInputs(t *testing.T) {
	for _, change := range []string{"relative-root", "relative-input", "relative-output", "source-output", "wrong-name", "extra-arg", "symlink-source", "missing-proof"} {
		t.Run(change, func(t *testing.T) {
			root, input, parent := t.TempDir(), t.TempDir(), t.TempDir()
			output := filepath.Join(parent, "pipx-runtime.zip")
			args := []string{"materialize-pipx-candidate", "--root", root, "--input", input, "--output", output}
			switch change {
			case "relative-root":
				args[2] = "."
			case "relative-input":
				args[4] = "."
			case "relative-output":
				args[6] = "pipx-runtime.zip"
			case "source-output":
				args[6] = filepath.Join(root, "pipx-runtime.zip")
			case "wrong-name":
				args[6] = filepath.Join(parent, "wrong")
			case "extra-arg":
				args = append(args, "unexpected")
			case "symlink-source":
				link := filepath.Join(t.TempDir(), "linked")
				if err := os.Symlink(root, link); err != nil {
					t.Fatal(err)
				}
				args[2] = link
			}
			var out bytes.Buffer
			if err := run(args, &out); err == nil || out.Len() != 0 {
				t.Fatalf("unverified candidate succeeded: %v %s", err, out.String())
			}
			for _, directory := range []string{root, input, parent} {
				entries, err := os.ReadDir(directory)
				if err != nil || len(entries) != 0 {
					t.Fatalf("failed candidate changed inputs/output: %s %v", directory, err)
				}
			}
		})
	}
}
