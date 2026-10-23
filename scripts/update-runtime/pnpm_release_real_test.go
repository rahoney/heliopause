//go:build realcorpus

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPNPMOfficialReleaseOfflineCommand(t *testing.T) {
	input := os.Getenv("HELOX_PNPM_RELEASE_ROOT")
	if !filepath.IsAbs(input) {
		t.Fatal("HELOX_PNPM_RELEASE_ROOT must declare the acquired official proof/archive directory")
	}
	var out bytes.Buffer
	if err := run([]string{"verify-pnpm-release", "--input", input}, &out); err != nil {
		t.Fatal(err)
	}
	var value struct {
		Schema       string
		ArchiveBytes int `json:"archive_bytes"`
		Invocation   string
		Tool         struct {
			SHA256    string
			Size      int64
			Libraries []string
		}
	}
	if err := json.Unmarshal(out.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.Schema != "helox.pnpm-release-supply/v2" || value.ArchiveBytes != 25369328 || value.Invocation != "https://github.com/pnpm/pnpm/actions/runs/39583314875/attempts/1" || value.Tool.SHA256 != "8cca52dc2aab0081ac153a3c7ad36224cd72b6d7cfea413746b7b97f6a3d8942" || value.Tool.Size != 57822504 || len(value.Tool.Libraries) != 6 {
		t.Fatalf("unexpected official receipt: %s", out.String())
	}
}
