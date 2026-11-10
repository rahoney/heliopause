//go:build realcorpus

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestBunOfficialCandidateIsMeasuredWithoutHostExecution(t *testing.T) {
	input := os.Getenv("HELOX_BUN_RELEASE_ROOT")
	if !filepath.IsAbs(input) {
		t.Fatal("explicit absolute official Bun supply is required")
	}
	supply, err := readBunSupply(input, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	root, parent := t.TempDir(), t.TempDir()
	output := filepath.Join(parent, "bun-candidate")
	args := []string{"materialize-bun-candidate", "--root", root, "--input", input, "--output", output}
	var out bytes.Buffer
	if err := run(args, &out); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Schema                string `json:"schema"`
		SHA256                string `json:"sha256"`
		Size                  int64  `json:"size"`
		NativeVersionVerified bool   `json:"native_version_verified"`
		RuntimeQualified      bool   `json:"runtime_qualified"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(output)
	info, statErr := os.Lstat(output)
	if err != nil || statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0555 || info.Size() != supply.tool.Size() || hash(body) != supply.tool.SHA256() || report.Schema != "helox.bun-native-candidate/v1" || report.SHA256 != hash(body) || report.Size != int64(len(body)) || report.NativeVersionVerified || report.RuntimeQualified {
		t.Fatal("candidate bytes/mode/claim differs", err, statErr, out.String())
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != uint32(os.Geteuid()) || owner.Nlink != 1 {
		t.Fatal("candidate inherited archive ownership or retained another link")
	}
	out.Reset()
	if err := run(args, &out); err == nil || out.Len() != 0 {
		t.Fatal("candidate replacement was permitted")
	}
	after, err := os.Lstat(output)
	if err != nil || !os.SameFile(info, after) {
		t.Fatal("candidate was replaced")
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 1 {
		t.Fatal("candidate publication leaked temporary/archive members")
	}
}
