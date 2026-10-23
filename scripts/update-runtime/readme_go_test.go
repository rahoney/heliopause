package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoordinatedGoBuildReadmePreservesHistoricalSelections(t *testing.T) {
	root, state := fixture(t)
	current, err := selectedVersion(root, "go")
	if err != nil {
		t.Fatal(err)
	}
	paragraph := func(version string) string {
		return fmt.Sprintf("Heliopause는 Go `%s` 이상이 필요하며, 개발·CI toolchain은 Go `%s`으로 고정한다.", version, version)
	}
	document := func(build string) string {
		return "# Fixture\n\n## History\n\n" + paragraph(current) + "\n\n## Build\n\n" + paragraph(build) + "\n\n## Examples\n\n" + paragraph(current) + "\n"
	}
	writeTest(t, filepath.Join(root, "README.md"), []byte(document(current)))
	gitTest(t, root, "add", "README.md")
	commitTest(t, root)
	p := futurePlan(t, root, state, "go")
	if err := applyPlan(p, state, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(state, "candidate", "README.md"))
	if err != nil || string(body) != document(p.Version) {
		t.Fatalf("only the Build minimum/toolchain must change: %v\n%s", err, body)
	}
	if err := compareSnapshot(p); err != nil {
		t.Fatal("source checkout changed", err)
	}
}

func TestReadmeGoRenderingRejectsUnknownBaseline(t *testing.T) {
	const version = "1.26.10"
	paragraph := "Heliopause는 Go `1.26.10` 이상이 필요하며, 개발·CI toolchain은 Go `1.26.10`으로 고정한다."
	valid := "# Fixture\n\n## Build\n\n" + paragraph + "\n"
	for _, body := range []string{
		"# Fixture\n",
		valid + "\n## Build\n",
		strings.Replace(valid, "Go `1.26.10` 이상", "Go `1.26.9` 이상", 1),
		strings.Replace(valid, "toolchain은 Go `1.26.10`", "toolchain은 Go `1.26.9`", 1),
		strings.Replace(valid, "개발·CI toolchain", "compiler", 1),
		valid + paragraph + "\n",
	} {
		if _, err := editReadmeGoBuild([]byte(body), version, "1.27.3"); err == nil {
			t.Fatalf("unknown or mixed README input accepted: %q", body)
		}
	}
	after, err := editReadmeGoBuild([]byte(valid), version, version)
	if err != nil || !bytes.Equal(after, []byte(valid)) {
		t.Fatal("same-version reconciliation changed README", err)
	}
}
