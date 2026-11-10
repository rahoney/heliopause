package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pipxProofDirectory(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "internal", "verification", "releaseprovenance", "testdata", "pipx-1.18.4"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPipxOfflineSupplyRequiresActualWheel(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"verify-pipx-publish", "--input", pipxProofDirectory(t)}, &out)
	if err == nil || !strings.Contains(err.Error(), "frozen supply file") || out.Len() != 0 {
		t.Fatalf("proof alone produced receipt: %v %s", err, out.String())
	}
}

func TestPipxOfflineSupplyRejectsUnverifiedInputs(t *testing.T) {
	for _, name := range []string{"relative", "missing-identity", "wrong-bytes", "symlink-wheel", "symlink-proof", "symlink-directory", "oversize", "unknown-identity", "duplicate-identity", "case-identity", "trailing-identity", "extra-argument"} {
		t.Run(name, func(t *testing.T) {
			input := t.TempDir()
			for _, file := range []string{"identity.json", "provenance.json"} {
				body, err := os.ReadFile(filepath.Join(pipxProofDirectory(t), file))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(input, file), body, 0600); err != nil {
					t.Fatal(err)
				}
			}
			wheel := filepath.Join(input, "pipx-1.18.4-py3-none-any.whl")
			args := []string{"verify-pipx-publish", "--input", input}
			var err error
			switch name {
			case "relative":
				args[2] = "."
			case "missing-identity":
				err = os.Remove(filepath.Join(input, "identity.json"))
			case "wrong-bytes":
				err = os.WriteFile(wheel, []byte("unverified bytes"), 0600)
			case "symlink-wheel":
				err = os.Symlink(filepath.Join(pipxProofDirectory(t), "provenance.json"), wheel)
			case "symlink-proof":
				err = os.Remove(filepath.Join(input, "provenance.json"))
				if err == nil {
					err = os.Symlink(filepath.Join(pipxProofDirectory(t), "provenance.json"), filepath.Join(input, "provenance.json"))
				}
			case "symlink-directory":
				link := filepath.Join(t.TempDir(), "supply")
				err = os.Symlink(input, link)
				args[2] = link
			case "oversize":
				err = os.WriteFile(wheel, make([]byte, (1<<20)+1), 0600)
			case "unknown-identity", "duplicate-identity", "case-identity", "trailing-identity":
				var body []byte
				body, err = os.ReadFile(filepath.Join(input, "identity.json"))
				if err == nil {
					switch name {
					case "unknown-identity":
						body = bytes.Replace(body, []byte("{"), []byte(`{"unknown":true,`), 1)
					case "duplicate-identity":
						body = bytes.Replace(body, []byte(`"version":`), []byte(`"version":"1.0.0","version":`), 1)
					case "case-identity":
						body = bytes.Replace(body, []byte(`"version":`), []byte(`"Version":`), 1)
					case "trailing-identity":
						body = append(body, []byte("{}")...)
					}
					err = os.WriteFile(filepath.Join(input, "identity.json"), body, 0600)
				}
			case "extra-argument":
				args = append(args, "unexpected")
			}
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := run(args, &out); err == nil || out.Len() != 0 {
				t.Fatalf("unverified input produced receipt: %v %s", err, out.String())
			}
		})
	}
}
