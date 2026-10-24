package gomodule

import (
	"strings"
	"testing"
)

func TestUpgradeFreezesRealNamedModuleAndPreservesDirectives(t *testing.T) {
	original := []byte("module example.com/root\ngo 1.26.0\nrequire example.com/Case/v2 v2.0.0 // indirect\n")
	selected := []byte(strings.ReplaceAll(string(original), "v2.0.0", "v2.1.0"))
	records := []DownloadRecord{{Path: "example.com/Case/v2", Version: "v2.1.0"}}
	ref, err := FreezeUpgradeReference(original, selected, records[0].Path, records)
	if err != nil || ref.Locator() != "example.com/Case/v2@v2.1.0" {
		t.Fatalf("exact named selection: %v", err)
	}
	for _, selector := range []string{"example.com/missing", "example.com/Case/v2@v2.0.0", "example.com/Case/v2/...", "go", "toolchain", "", "./..."} {
		if ValidateUpgradeMembership(original, selector) == nil {
			t.Fatalf("selector admitted: %q", selector)
		}
	}
	for _, change := range []string{
		strings.ReplaceAll(string(selected), "go 1.26.0", "go 1.26.9"),
		strings.ReplaceAll(string(selected), "example.com/root", "example.com/changed"),
		string(selected) + "toolchain go1.26.9\n",
		string(selected) + "exclude example.com/other v1.0.0\n",
		string(selected) + "retract v1.0.0\n",
		string(selected) + "tool example.com/tool\n",
		string(selected) + "godebug panicnil=1\n",
		string(selected) + "ignore ./assets\n",
		string(selected) + "replace example.com/Case/v2 => ../local\n",
		string(selected) + "require example.com/Case/v2 v2.2.0\n",
	} {
		if _, err := FreezeUpgradeReference(original, []byte(change), records[0].Path, records); err == nil {
			t.Fatalf("directive/source/duplicate drift admitted: %s", change)
		}
	}
	for _, changed := range [][]DownloadRecord{nil, {records[0], records[0]}, {{Path: "example.com/case/v2", Version: "v2.1.0"}}} {
		if _, err := FreezeUpgradeReference(original, selected, records[0].Path, changed); err == nil {
			t.Fatal("missing/duplicate/substituted record admitted")
		}
	}
}
