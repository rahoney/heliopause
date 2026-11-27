package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pnpmProofDirectory(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "internal", "verification", "releaseprovenance", "testdata", "pnpm-12.15.0"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPNPMOfflineSupplyRejectsUnverifiedArchive(t *testing.T) {
	for _, name := range []string{"relative", "missing", "wrong-bytes", "symlink-file", "symlink-directory", "oversize", "unknown-identity", "trailing-identity", "duplicate-identity", "extra-argument"} {
		t.Run(name, func(t *testing.T) {
			input := t.TempDir()
			for _, file := range []string{"identity.json", "bundle.json"} {
				body, err := os.ReadFile(filepath.Join(pnpmProofDirectory(t), file))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(input, file), body, 0600); err != nil {
					t.Fatal(err)
				}
			}
			archive := filepath.Join(input, "pnpm-linux-x64.tar.gz")
			args := []string{"verify-pnpm-release", "--input", input}
			var err error
			switch name {
			case "relative":
				args[2] = "."
			case "missing":
			case "wrong-bytes":
				err = os.WriteFile(archive, []byte("unsigned archive"), 0600)
			case "symlink-file":
				err = os.Symlink(filepath.Join(pnpmProofDirectory(t), "bundle.json"), archive)
			case "symlink-directory":
				link := filepath.Join(t.TempDir(), "proof")
				err = os.Symlink(input, link)
				args[2] = link
			case "oversize":
				var file *os.File
				file, err = os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if err == nil {
					err = file.Truncate((64 << 20) + 1)
					if closeErr := file.Close(); err == nil {
						err = closeErr
					}
				}
			case "unknown-identity":
				err = os.WriteFile(filepath.Join(input, "identity.json"), []byte(`{"unknown":true}`), 0600)
			case "trailing-identity":
				body, readErr := os.ReadFile(filepath.Join(input, "identity.json"))
				if readErr != nil {
					t.Fatal(readErr)
				}
				err = os.WriteFile(filepath.Join(input, "identity.json"), append(body, []byte("\n{}")...), 0600)
			case "duplicate-identity":
				body, readErr := os.ReadFile(filepath.Join(input, "identity.json"))
				if readErr != nil {
					t.Fatal(readErr)
				}
				err = os.WriteFile(filepath.Join(input, "identity.json"), bytes.Replace(body, []byte(`"version":`), []byte(`"version":"0.0.1","version":`), 1), 0600)
			case "extra-argument":
				args = append(args, "unexpected")
			}
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := run(args, &out); err == nil || out.Len() != 0 {
				t.Fatalf("unverified input returned receipt: %v %s", err, out.String())
			}
		})
	}
}

func TestPNPMSupplyReceiptRequiresActualArchiveDigest(t *testing.T) {
	// A valid proof alone must not produce a tool supply receipt.
	var out bytes.Buffer
	err := run([]string{"verify-pnpm-release", "--input", pnpmProofDirectory(t)}, &out)
	if err == nil || !strings.Contains(err.Error(), "frozen supply file") || out.Len() != 0 {
		t.Fatalf("missing archive was not rejected after provenance verification: %v %s", err, out.String())
	}
}
