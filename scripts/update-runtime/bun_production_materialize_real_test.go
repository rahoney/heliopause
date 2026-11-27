//go:build realcorpus

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestBunProductionMaterializationFromOfficialArchive(t *testing.T) {
	input := os.Getenv("HELOX_BUN_RELEASE_ROOT")
	if !filepath.IsAbs(input) {
		t.Fatal("official archive input must be explicitly declared")
	}
	root, _ := fixture(t)
	pin, err := readBunMaterializationOwner(root)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "bun-linux-amd64")
	args := []string{"materialize-bun", "--root", root, "--archive", filepath.Join(input, "bun-linux-x64.zip"), "--output", output, "--notices-output", filepath.Join(filepath.Dir(output), "bun-NOTICE.txt")}
	var report bytes.Buffer
	if err := run(args, &report); err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Schema string
		SHA256 string
		Size   int64
	}
	if err := json.Unmarshal(report.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(output)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0555 || info.Size() != pin.Tool.Size || hash(body) != pin.Tool.SHA256 || receipt.Schema != "helox.bun-native-asset/v1" || receipt.SHA256 != hash(body) || receipt.Size != int64(len(body)) {
		t.Fatalf("production native identity or mode mismatch: %v %s", err, report.String())
	}
	noticeBody, err := os.ReadFile(args[8])
	if err != nil || hash(noticeBody) != pin.Notices.SHA256 {
		t.Fatal("production notice bytes were not copied")
	}
	noticeInfo, err := os.Lstat(args[8])
	if err != nil || noticeInfo.Mode().Perm() != 0444 {
		t.Fatal("notice was not passive read-only data")
	}
	before := info
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != uint32(os.Geteuid()) || owner.Nlink != 1 {
		t.Fatal("native asset inherited archive ownership or retained another link")
	}
	report.Reset()
	if err := run(args, &report); err == nil || report.Len() != 0 {
		t.Fatal("second materialization overwrote the asset or reported success")
	}
	after, err := os.Lstat(output)
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("existing native asset identity changed")
	}
	entries, err := os.ReadDir(filepath.Dir(output))
	if err != nil || len(entries) != 2 {
		t.Fatalf("materialization leaked another archive member or temporary file: %v", err)
	}
	// No execution: the host consumes only authenticated regular-file bytes.
	for _, failure := range []string{"archive-digest", "production-proof"} {
		t.Run(failure, func(t *testing.T) {
			candidate := append([]string(nil), args...)
			candidate[6] = filepath.Join(t.TempDir(), "bun-linux-amd64")
			candidate[8] = filepath.Join(filepath.Dir(candidate[6]), "bun-NOTICE.txt")
			if failure == "archive-digest" {
				fake := filepath.Join(t.TempDir(), "changed.tar.gz")
				file, err := os.OpenFile(fake, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Truncate(pin.Archive.Size); err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
				candidate[4] = fake
			} else {
				writeTest(t, filepath.Join(root, pin.Publisher.ChecksumsPath), []byte("changed production proof"))
			}
			var rejected bytes.Buffer
			if err := run(candidate, &rejected); err == nil || rejected.Len() != 0 {
				t.Fatal("changed production input produced a success receipt")
			}
			if _, err := os.Lstat(candidate[6]); !os.IsNotExist(err) {
				t.Fatal("changed production input wrote a native asset")
			}
		})
	}
}
