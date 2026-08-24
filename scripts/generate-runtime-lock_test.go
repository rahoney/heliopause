package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeLockAndGeneratedIdentityAreCurrent(t *testing.T) {
	t.Parallel()
	lock, err := readLock(filepath.Join("runtimes.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(filepath.Join("..", outputPath))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, render(lock)) {
		t.Fatal("generated runtime identity differs from the canonical lock")
	}
}

func TestRuntimeLockRejectsUnknownFieldAndMissingPlatform(t *testing.T) {
	t.Parallel()
	current, err := os.ReadFile("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	unknown := filepath.Join(directory, "unknown.json")
	if err := os.WriteFile(unknown, append(current[:len(current)-2], []byte(",\n  \"unexpected\": true\n}\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readLock(unknown); err == nil {
		t.Fatal("unknown runtime lock field was accepted")
	}

	lock, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	delete(lock.GVisor.Binaries, "aarch64")
	if err := validate(lock); err == nil {
		t.Fatal("missing runtime architecture was accepted")
	}
}
