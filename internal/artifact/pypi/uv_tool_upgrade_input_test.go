package pypi

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func uvToolUpgradeFixture(t *testing.T) (ToolReceipt, []InstalledMetadataFile) {
	t.Helper()
	body, err := os.ReadFile("testdata/persistent-tool/uv-installed.toml")
	if err != nil {
		t.Fatal(err)
	}
	body = bytes.ReplaceAll(body, []byte("no-index = true\n"), nil)
	body = bytes.ReplaceAll(body, []byte("find-links = [\"file:///opt/haa-query/wheels-v1\"]\n"), nil)
	receipt, err := ParseUVToolReceipt(body)
	if err != nil {
		t.Fatal(err)
	}
	return receipt, []InstalledMetadataFile{{"haa_tool_fixture-1.0.0.dist-info", []byte("Metadata-Version: 2.4\nName: haa-tool-fixture\nVersion: 1.0.0\n")}}
}

func TestUVToolUpgradeInputKeepsOriginalAndPositionalConstraintsSeparate(t *testing.T) {
	receipt, files := uvToolUpgradeFixture(t)
	request, _ := ParseUpgradeRequirement("haa-tool-fixture>=1.1,<2")
	input, err := NewUVToolUpgradeInput(receipt, request, files, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	// The old version is allowed to lie outside the new positional constraint.
	if request.AcceptsVersion(input.Original().Installed()[0].Version()) || !input.Receipt().Requirement().AcceptsVersion("1.0.0") {
		t.Fatal("original and requested constraints collapsed")
	}
	control, err := input.Control()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseUVToolUpgradeInput([]domain.ProjectControlFile{control}, "3.14.7")
	if err != nil || parsed.Request().Requirement() != request.Requirement() || !bytes.Equal(parsed.Receipt().Body(), receipt.Body()) {
		t.Fatal("canonical input did not preserve intent", err)
	}
	files[0].Body[0] = 'x'
	copy := parsed.Original().MetadataFiles()
	copy[0].Body[0] = 'x'
	body := parsed.Receipt().Body()
	body[0] = 'x'
	after, err := parsed.Control()
	if err != nil || !bytes.Equal(after.Body(), control.Body()) {
		t.Fatal("caller changed immutable input", err)
	}
}

func TestUVToolUpgradeInputRejectsOriginalMismatchAndUnreviewedRecipe(t *testing.T) {
	for name, mutate := range map[string]func(*ToolReceipt, *[]InstalledMetadataFile, *UpgradeRequirement){
		"missing root": func(_ *ToolReceipt, f *[]InstalledMetadataFile, _ *UpgradeRequirement) { *f = nil },
		"wrong distribution": func(_ *ToolReceipt, f *[]InstalledMetadataFile, _ *UpgradeRequirement) {
			(*f)[0].Directory = "other-1.0.0.dist-info"
		},
		"duplicate root": func(_ *ToolReceipt, f *[]InstalledMetadataFile, _ *UpgradeRequirement) { *f = append(*f, (*f)[0]) },
		"receipt version mismatch": func(_ *ToolReceipt, f *[]InstalledMetadataFile, _ *UpgradeRequirement) {
			(*f)[0].Directory = "haa_tool_fixture-3.0.0.dist-info"
			(*f)[0].Body = bytes.ReplaceAll((*f)[0].Body, []byte("1.0.0"), []byte("3.0.0"))
		},
		"different requested root": func(_ *ToolReceipt, _ *[]InstalledMetadataFile, r *UpgradeRequirement) {
			*r, _ = ParseUpgradeRequirement("other")
		},
		"local find links": func(r *ToolReceipt, _ *[]InstalledMetadataFile, _ *UpgradeRequirement) {
			*r, _ = ParseUVToolReceipt(append(r.Body(), []byte("find-links = [\"/private/wheels\"]\n")...))
		},
		"no index": func(r *ToolReceipt, _ *[]InstalledMetadataFile, _ *UpgradeRequirement) {
			*r, _ = ParseUVToolReceipt(append(r.Body(), []byte("no-index = true\n")...))
		},
		"source execution": func(r *ToolReceipt, _ *[]InstalledMetadataFile, _ *UpgradeRequirement) {
			*r, _ = ParseUVToolReceipt(bytes.ReplaceAll(r.Body(), []byte("no-build = true"), []byte("no-build = false")))
		},
		"mutable recipe": func(r *ToolReceipt, _ *[]InstalledMetadataFile, _ *UpgradeRequirement) {
			*r, _ = ParseUVToolReceipt(bytes.ReplaceAll(r.Body(), []byte("no-sources = true"), []byte("no-sources = false")))
		},
		"link mode": func(r *ToolReceipt, _ *[]InstalledMetadataFile, _ *UpgradeRequirement) {
			*r, _ = ParseUVToolReceipt(bytes.ReplaceAll(r.Body(), []byte("link-mode = \"copy\""), []byte("link-mode = \"symlink\"")))
		},
		"no receipt": func(r *ToolReceipt, _ *[]InstalledMetadataFile, _ *UpgradeRequirement) { *r = ToolReceipt{} },
	} {
		t.Run(name, func(t *testing.T) {
			receipt, files := uvToolUpgradeFixture(t)
			request, _ := ParseUpgradeRequirement("haa-tool-fixture")
			mutate(&receipt, &files, &request)
			if _, err := NewUVToolUpgradeInput(receipt, request, files, "3.14.7"); err == nil {
				t.Fatal("unsafe tool input accepted")
			}
		})
	}
}

func TestUVToolUpgradeInputRejectsNoncanonicalAndMixedControls(t *testing.T) {
	receipt, files := uvToolUpgradeFixture(t)
	request, _ := ParseUpgradeRequirement("haa-tool-fixture")
	input, err := NewUVToolUpgradeInput(receipt, request, files, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	control, _ := input.Control()
	for name, body := range map[string][]byte{
		"unknown":    append(append([]byte(nil), control.Body()[:len(control.Body())-1]...), []byte(",\"extra\":true}")...),
		"whitespace": append([]byte(" "), control.Body()...),
		"duplicate":  bytes.Replace(control.Body(), []byte("\"version\":1"), []byte("\"version\":1,\"version\":1"), 1),
		"oversize":   bytes.Repeat([]byte(" "), upgradeInputLimit+1),
		"trailing":   append(control.Body(), []byte("{}")...),
	} {
		t.Run(name, func(t *testing.T) {
			c, e := domain.NewProjectControlFile(UVToolUpgradeInputControlName, body, true)
			if e != nil {
				return
			}
			if _, err := ParseUVToolUpgradeInput([]domain.ProjectControlFile{c}, "3.14.7"); err == nil {
				t.Fatal("noncanonical input accepted")
			}
		})
	}
	if _, err := ParseUVToolUpgradeInput([]domain.ProjectControlFile{control}, "3.14.8"); err == nil {
		t.Fatal("different runtime accepted")
	}
	if _, err := ParseUVToolUpgradeInput([]domain.ProjectControlFile{control, control}, "3.14.7"); err == nil {
		t.Fatal("duplicate controls accepted")
	}
	var wire uvToolUpgradeInputWire
	if json.Unmarshal(control.Body(), &wire) != nil {
		t.Fatal("wire")
	}
	var originalWire upgradeInputWire
	if json.Unmarshal(wire.Original, &originalWire) != nil {
		t.Fatal("original wire")
	}
	originalWire.Requirement = "haa-tool-fixture>=1.1,<2"
	wire.Original, _ = json.Marshal(originalWire)
	body, _ := json.Marshal(wire)
	c, _ := domain.NewProjectControlFile(UVToolUpgradeInputControlName, body, true)
	if originalWire.Requirement != "haa-tool-fixture>=1.1,<2" {
		t.Fatal("mismatch fixture did not change")
	}
	if _, err := ParseUVToolUpgradeInput([]domain.ProjectControlFile{c}, "3.14.7"); err == nil {
		t.Fatal("receipt/input mismatch accepted")
	}
}

func TestUVToolUpgradeInputEnforcesCombinedEncodedByteBound(t *testing.T) {
	receipt, files := uvToolUpgradeFixture(t)
	padded, err := ParseUVToolReceipt(append(receipt.Body(), []byte("#"+strings.Repeat("x", 3<<20)+"\n")...))
	if err != nil {
		t.Fatal("individually bounded receipt fixture", err)
	}
	request, _ := ParseUpgradeRequirement("haa-tool-fixture")
	if _, err := NewUVToolUpgradeInput(padded, request, files, "3.14.7"); err == nil {
		t.Fatal("combined base64-encoded input exceeded control limit")
	}
}
