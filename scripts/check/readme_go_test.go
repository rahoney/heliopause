package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoBuildREADMETracksSelectedVersion(t *testing.T) {
	t.Parallel()
	const selected = "1.26.10"
	paragraph := func(minimum, compiler string) string {
		return fmt.Sprintf("Heliopause는 Go `%s` 이상이 필요하며, 개발·CI toolchain은 Go `%s`으로 고정한다.", minimum, compiler)
	}
	valid := "# Project\n\nHistorical Go `1.26.9` evidence stays unchanged.\n\n## Build\n\n" + paragraph(selected, selected) + "\n\n## Commands\n\nHistorical Go `1.26.9` example.\n"
	for _, test := range []struct {
		name, body string
		missing    bool
		wantError  bool
	}{
		{name: "selected and unrelated history", body: valid},
		{name: "stale minimum", body: strings.Replace(valid, paragraph(selected, selected), paragraph("1.26.9", selected), 1), wantError: true},
		{name: "stale compiler", body: strings.Replace(valid, paragraph(selected, selected), paragraph(selected, "1.26.9"), 1), wantError: true},
		{name: "missing README", missing: true, wantError: true},
		{name: "missing Build section", body: strings.Replace(valid, "## Build\n", "## Build notes\n", 1), wantError: true},
		{name: "duplicate Build section", body: valid + "\n## Build\n\n" + paragraph(selected, selected) + "\n", wantError: true},
		{name: "duplicate version paragraph", body: strings.Replace(valid, paragraph(selected, selected), paragraph(selected, selected)+"\n"+paragraph(selected, selected), 1), wantError: true},
		{name: "unknown field grammar", body: strings.Replace(valid, "개발·CI toolchain", "compiler", 1), wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeTestFile(t, filepath.Join(root, "go.mod"), "module example.test/project\n\ngo "+selected+"\n")
			writeTestFile(t, filepath.Join(root, "scripts", "runtimes.lock.json"), `{"go_image":{"go_version":"`+selected+`"}}`)
			if !test.missing {
				writeTestFile(t, filepath.Join(root, "README.md"), test.body)
			}
			c := checker{root: root, toolLock: toolLock{Tools: []toolSpec{{SetupGo: selected}}}}
			if err := c.checkGoVersionConsistency(); (err != nil) != test.wantError {
				t.Fatalf("README consumer: error = %v, want error %v", err, test.wantError)
			}
		})
	}
}
