//go:build realcorpus

package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// These are actual frozen official archives and independent release documents.
// The lock is a parser-binding fixture, not evidence of a native solver run.
// No artifact, pipx client or installed interpreter is executed here.
func TestPipxPreQueryActualFrozenWheelCorpus(t *testing.T) {
	directory := os.Getenv("HELOX_PIPX_QUERY_SUPPLY_ROOT")
	if directory == "" {
		t.Skip("explicit frozen official supply corpus required")
	}
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		t.Fatal("supply corpus must be explicit and absolute")
	}
	var owner struct {
		Pipx struct {
			Inputs []struct{ Path, SHA256 string } `json:"inputs"`
		} `json:"pipx_tool"`
	}
	body, err := os.ReadFile("../../../scripts/runtimes.lock.json")
	if err != nil || json.Unmarshal(body, &owner) != nil || len(owner.Pipx.Inputs) != 42 {
		t.Fatal("canonical fixed supply owner unavailable", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	originals := map[string][]byte{}
	for _, input := range owner.Pipx.Inputs {
		body, err := root.ReadFile(input.Path)
		sum := sha256.Sum256(body)
		if err != nil || hex.EncodeToString(sum[:]) != input.SHA256 {
			t.Fatal("frozen input differs from canonical owner", input.Path, err)
		}
		if strings.HasPrefix(input.Path, "userpath/") || strings.HasPrefix(input.Path, "click/") {
			originals[input.Path] = body
		}
	}
	var inputs []UVWheelInput
	var pages []SimpleProject
	var packages []UVLockedPackage
	for _, input := range owner.Pipx.Inputs {
		parts := strings.Split(input.Path, "/")
		if len(parts) != 2 || parts[0] != "userpath" && parts[0] != "click" || !strings.HasSuffix(parts[1], ".whl") {
			continue
		}
		project, version, _, _, _, err := ParseWheelFilename(parts[1])
		if err != nil {
			t.Fatal(err)
		}
		page, err := ParseUVReleaseMetadata(project, version, originals[project+"/metadata.json"])
		if err != nil {
			t.Fatal(err)
		}
		selected := ""
		archive := originals[input.Path]
		for _, file := range page.Files() {
			if file.Filename() == parts[1] && file.SHA256() == input.SHA256 && file.Size() == uint64(len(archive)) && !file.Yanked() {
				if selected != "" {
					t.Fatal("ambiguous frozen wheel")
				}
				selected = file.URL()
			}
		}
		if selected == "" {
			t.Fatal("original wheel absent from independent release metadata", project)
		}
		packages = append(packages, UVLockedPackage{Name: project, Version: version, Wheels: []UVLockedWheel{{Filename: parts[1], URL: selected, SHA256: input.SHA256, Size: int64(len(archive))}}})
		inputs = append(inputs, UVWheelInput{URL: selected, Reader: bytes.NewReader(archive), Size: int64(len(archive))})
		pages = append(pages, page)
	}
	if len(packages) != 2 {
		t.Fatal("expected exact userpath/click original closure")
	}
	var version string
	for _, p := range packages {
		if p.Name == "userpath" {
			version = p.Version
		}
	}
	reference, err := ParseReference("userpath@" + version)
	if err != nil {
		t.Fatal(err)
	}
	lock := UVLock{reference: reference, python: runtimeidentity.PythonVersion, packages: packages}
	target := WheelTarget{runtimeidentity.PythonInterpreterTag, runtimeidentity.PythonABITag, runtimeidentity.PythonPlatformTag}
	graph, err := lock.BindPipxQueryWheels(inputs, pages, target, DefaultWheelLimits())
	if err != nil || len(graph.Nodes()) != 2 {
		t.Fatal("frozen original corpus rejected", err)
	}
	for _, n := range graph.Nodes() {
		id := n.Artifact().Identity()
		if id.Name() != "userpath" && id.Name() != "click" {
			t.Fatal("foreign graph member")
		}
	}
	changed := append([]UVWheelInput(nil), inputs...)
	changed[0].Reader = bytes.NewReader(bytes.Repeat([]byte("!"), int(changed[0].Size)))
	if result, err := lock.BindPipxQueryWheels(changed, pages, target, DefaultWheelLimits()); err == nil || len(result.Nodes()) != 0 {
		t.Fatal("changed actual original archive accepted")
	}
	t.Logf("frozen_inputs=42 original_archive_nodes=%d main=userpath version=%s; parser-binding fixture only, no native solver/query/Policy", len(graph.Nodes()), version)
}
