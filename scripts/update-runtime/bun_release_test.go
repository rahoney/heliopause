package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBunOfflineSupplyRejectsUnverifiedOrAmbiguousInputs(t *testing.T) {
	fixture, err := filepath.Abs(filepath.Join("..", "..", "internal", "verification", "releaseprovenance", "testdata", "bun-1.4.4"))
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"relative", "missing", "wrong-bytes", "symlink-file", "symlink-directory", "oversize", "unknown-identity", "duplicate-identity", "trailing-identity", "extra-argument"} {
		t.Run(change, func(t *testing.T) {
			input := t.TempDir()
			for _, name := range []string{"identity.json", "SHASUMS256.txt.asc"} {
				body, err := os.ReadFile(filepath.Join(fixture, name))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(input, name), body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			args := []string{"verify-bun-release", "--input", input}
			archive := filepath.Join(input, "bun-linux-x64.zip")
			switch change {
			case "relative":
				args[2] = "."
			case "wrong-bytes":
				if err := os.WriteFile(archive, []byte("unsigned archive"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "symlink-file":
				if err := os.Symlink(filepath.Join(input, "SHASUMS256.txt.asc"), archive); err != nil {
					t.Fatal(err)
				}
			case "symlink-directory":
				link := filepath.Join(t.TempDir(), "proof")
				if err := os.Symlink(input, link); err != nil {
					t.Fatal(err)
				}
				args[2] = link
			case "oversize":
				f, err := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
				if err != nil {
					t.Fatal(err)
				}
				err = f.Truncate((64 << 20) + 1)
				closeErr := f.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
			case "unknown-identity":
				if err := os.WriteFile(filepath.Join(input, "identity.json"), []byte(`{"unknown":true}`), 0o600); err != nil {
					t.Fatal(err)
				}
			case "duplicate-identity":
				body, err := os.ReadFile(filepath.Join(input, "identity.json"))
				if err != nil {
					t.Fatal(err)
				}
				body = bytes.Replace(body, []byte(`"version":`), []byte(`"version":"0.0.1","version":`), 1)
				if err := os.WriteFile(filepath.Join(input, "identity.json"), body, 0o600); err != nil {
					t.Fatal(err)
				}
			case "trailing-identity":
				body, err := os.ReadFile(filepath.Join(input, "identity.json"))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(input, "identity.json"), append(body, []byte("\n{}")...), 0o600); err != nil {
					t.Fatal(err)
				}
			case "extra-argument":
				args = append(args, "unexpected")
			}
			var out bytes.Buffer
			if err := run(args, &out); err == nil {
				t.Fatal("unverified Bun supply admitted", change, out.String())
			}
		})
	}
}
