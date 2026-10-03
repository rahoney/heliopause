package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGVisorBundleVerifier(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	built := filepath.Join(root, "built")
	lockPath := filepath.Join(root, "lock.json")
	paths := []string{
		"containerd-shim-runsc-v1",
		"gvisor-bin/checkpointgofer",
		"gvisor-bin/gvisor-sentry-prewarmer",
		"gvisor-bin/gvisor_sentry",
		"gvisor-bin/runsc-metric-server",
		"runsc",
	}
	type member struct {
		Path   string `json:"path"`
		Size   int    `json:"size"`
		SHA512 string `json:"sha512"`
	}
	var lock struct {
		GVisor struct {
			RuntimeBundle struct {
				Members []member `json:"members"`
			} `json:"runtime_bundle"`
		} `json:"gvisor"`
	}
	for _, path := range paths {
		contents := []byte("fixture:" + path)
		location := filepath.Join(built, path)
		if err := os.MkdirAll(filepath.Dir(location), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(location, contents, 0o600); err != nil {
			t.Fatal(err)
		}
		digest := sha512.Sum512(contents)
		lock.GVisor.RuntimeBundle.Members = append(lock.GVisor.RuntimeBundle.Members,
			member{Path: path, Size: len(contents), SHA512: hex.EncodeToString(digest[:])})
	}
	writeLock := func() {
		t.Helper()
		body, err := json.Marshal(lock)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(lockPath, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// The test executes the production verifier with a find implementation
	// that rejects GNU -printf, even when the host running this test has it.
	findPath, err := exec.LookPath("find")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "find"), []byte("#!/bin/sh\nfor arg do\n  if [ \"$arg\" = -printf ]; then exit 97; fi\ndone\nexec \"$FIND_REAL\" \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "sha512sum"), []byte("#!/bin/sh\nexit 97\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	verify := func(output string) ([]byte, error) {
		t.Helper()
		command := exec.Command("sh", "verify-gvisor-runtime-bundle.sh", lockPath, built, output)
		command.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "FIND_REAL="+findPath)
		return command.CombinedOutput()
	}
	expectFailure := func(name string, expected ...string) {
		t.Helper()
		output, err := verify(filepath.Join(root, name))
		if err == nil {
			t.Fatalf("%s unexpectedly passed", name)
		}
		for _, want := range expected {
			if !strings.Contains(string(output), want) {
				t.Fatalf("%s diagnostic missing %q: %s", name, want, output)
			}
		}
	}

	writeLock()
	successOutput := filepath.Join(root, "success")
	if output, err := verify(successOutput); err != nil {
		t.Fatalf("valid six-member fixture rejected: %v\n%s", err, output)
	}
	for _, path := range paths {
		if _, err := os.Stat(filepath.Join(successOutput, path)); err != nil {
			t.Fatalf("verified member %s was not copied: %v", path, err)
		}
	}

	wantPath := "gvisor-bin/checkpointgofer"
	actualSHA := lock.GVisor.RuntimeBundle.Members[1].SHA512
	lock.GVisor.RuntimeBundle.Members[1].SHA512 = strings.Repeat("0", 128)
	writeLock()
	expectFailure("hash-mismatch",
		"sha512 mismatch", "path="+wantPath,
		"expected="+strings.Repeat("0", 128), "actual="+actualSHA,
	)
	lock.GVisor.RuntimeBundle.Members[1].SHA512 = actualSHA
	actualSize := lock.GVisor.RuntimeBundle.Members[1].Size
	lock.GVisor.RuntimeBundle.Members[1].Size++
	writeLock()
	expectFailure("size-mismatch", "size mismatch", "path="+wantPath,
		"expected="+strconv.Itoa(actualSize+1), "actual="+strconv.Itoa(actualSize))
	lock.GVisor.RuntimeBundle.Members[1].Size--
	writeLock()

	memberPath := filepath.Join(built, wantPath)
	if err := os.Remove(memberPath); err != nil {
		t.Fatal(err)
	}
	expectFailure("missing-member", "inventory mismatch", wantPath)
	if err := os.WriteFile(memberPath, []byte("fixture:"+wantPath), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(built, "unexpected"), []byte("extra"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectFailure("unexpected-member", "inventory mismatch", "unexpected")
	if err := os.Remove(filepath.Join(built, "unexpected")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(memberPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("gvisor_sentry", memberPath); err != nil {
		t.Fatal(err)
	}
	expectFailure("symlink-member", "symlink", "path="+wantPath)
}
