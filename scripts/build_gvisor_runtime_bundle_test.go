package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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
	verify := func(output string) ([]byte, error) {
		t.Helper()
		return exec.Command("sh", "verify-gvisor-runtime-bundle.sh", lockPath, built, output).CombinedOutput()
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
	output, err := verify(filepath.Join(root, "mismatch"))
	if err == nil {
		t.Fatal("SHA-512 mismatch unexpectedly passed")
	}
	for _, want := range []string{
		"sha512 mismatch", "path=" + wantPath,
		"expected=" + strings.Repeat("0", 128), "actual=" + actualSHA,
	} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("mismatch diagnostic missing %q: %s", want, output)
		}
	}
}
