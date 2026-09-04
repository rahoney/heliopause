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

func TestRuntimeLockRejectsInvalidPatchIdentity(t *testing.T) {
	t.Parallel()
	baseLock, err := readLock("runtimes.lock.json")
	if err != nil {
		t.Fatal(err)
	}

	invalidCases := []struct {
		name   string
		path   string
		sha256 string
	}{
		{name: "missing path", path: "", sha256: baseLock.GVisor.Patch.SHA256},
		{name: "path outside tools/gvisor", path: "scripts/patch.patch", sha256: baseLock.GVisor.Patch.SHA256},
		{name: "directory traversal path", path: "tools/gvisor/../../etc/test.patch", sha256: baseLock.GVisor.Patch.SHA256},
		{name: "non-patch extension", path: "tools/gvisor/patch.txt", sha256: baseLock.GVisor.Patch.SHA256},
		{name: "missing sha256", path: baseLock.GVisor.Patch.Path, sha256: ""},
		{name: "short sha256", path: baseLock.GVisor.Patch.Path, sha256: "abc"},
		{name: "sha512 instead of sha256", path: baseLock.GVisor.Patch.Path, sha256: "4463ce276e207f5a516a08ec627a768a19cf7bed0094d522b0810bee3424585caa8d344e093204012b974f5c508ab2362dcb0d7236f0c1992fccc426beeb7ffc"},
		{name: "invalid characters in sha256", path: baseLock.GVisor.Patch.Path, sha256: "z98f802d74a6ee42e4090957373ec30c432b64b7106a589f94bfdd1f384f8162"},
	}

	for _, tc := range invalidCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			badLock := baseLock
			badLock.GVisor.Patch.Path = tc.path
			badLock.GVisor.Patch.SHA256 = tc.sha256
			if err := validate(badLock); err == nil {
				t.Fatalf("expected error for case %q, but got nil", tc.name)
			}
		})
	}
}
