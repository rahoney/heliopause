package pypi

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func pipxToolUpgradeFixture(t *testing.T, arguments []string) (ToolReceipt, []InstalledMetadataFile) {
	t.Helper()
	var wire map[string]any
	if err := json.Unmarshal(nativeToolFixture(t, "pipx-installed.json"), &wire); err != nil {
		t.Fatal(err)
	}
	wire["main_package"].(map[string]any)["pip_args"] = arguments
	body, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := ParsePipxToolReceipt(body)
	if err != nil {
		t.Fatal(err)
	}
	return receipt, []InstalledMetadataFile{{Directory: "haa_tool_fixture-1.0.0.dist-info", Body: []byte("Metadata-Version: 2.4\nName: haa-tool-fixture\nVersion: 1.0.0\n")}}
}

func TestPipxUpgradeInputKeepsOriginalConstraintAndDerivesUnconstrainedRequest(t *testing.T) {
	for _, arguments := range [][]string{nil, {}, {"--no-build", "--no-sources", "--no-config", "--no-cache", "--no-python-downloads", "--no-managed-python", "--link-mode", "copy", "--index-url", "https://pypi.org/simple/"}, {"--link-mode=copy", "--index-url=https://pypi.org/simple/"}} {
		receipt, files := pipxToolUpgradeFixture(t, arguments)
		input, err := NewPipxToolUpgradeInput(receipt, files, runtimeidentity.PythonVersion)
		if err != nil {
			t.Fatal(err)
		}
		if input.Request().Requirement() != "haa-tool-fixture" || !input.Request().AcceptsVersion("2.0.0") || input.Original().Requirement().AcceptsVersion("2.0.0") || input.Receipt().Version() != "1.0.0" {
			t.Fatal("old receipt constraint/version and native upgrade request collapsed")
		}
		originalBody := receipt.Body()
		control, err := input.Control()
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParsePipxToolUpgradeInput([]domain.ProjectControlFile{control}, runtimeidentity.PythonVersion)
		if err != nil || !bytes.Equal(parsed.Receipt().Body(), originalBody) || !reflect.DeepEqual(input.Request(), parsed.Request()) {
			t.Fatal("canonical round trip lost original receipt or request", err)
		}
		files[0].Body[0] = '!'
		changed := parsed.Original().MetadataFiles()
		changed[0].Body[0] = '!'
		changedBody := parsed.Receipt().Body()
		changedBody[0] = '!'
		after, err := parsed.Control()
		if err != nil || !bytes.Equal(after.Body(), control.Body()) {
			t.Fatal("original input aliases caller data", err)
		}
	}
}

func TestPipxUpgradeInputRejectsUnsupportedRecipeBeforeReplay(t *testing.T) {
	for _, arguments := range [][]string{
		{"--index-url", "https://unapproved.invalid/simple/"}, {"--index-url=https://pypi.org/simple/?secret=x"},
		{"--extra-index-url=https://pypi.org/simple/"}, {"--find-links=/host/private"}, {"--offline"}, {"--no-index"},
		{"--no-build=false"}, {"--no-build", "--no-build"}, {"--link-mode=copy", "--link-mode", "copy"},
		{"--index-url"}, {"--link-mode", "symlink"}, {"--python=/host/python"}, {"--config-file=/host/config"},
		{"--trusted-host=pypi.org"}, {"--no-binary=:all:"}, {"--pre"}, {"--editable"}, {"--", "foreign"},
		{"--index-url", "https://pypi.org/simple/", "--index-url=https://pypi.org/simple/"},
	} {
		receipt, files := pipxToolUpgradeFixture(t, arguments)
		if result, err := NewPipxToolUpgradeInput(receipt, files, runtimeidentity.PythonVersion); err == nil || result.Receipt().Valid() {
			t.Fatal("unreviewed opaque recipe accepted as native input", arguments)
		}
	}
	// The original parser deliberately remains an inert, opaque data parser.
	receipt, err := ParsePipxToolReceipt(nativeToolFixture(t, "pipx-installed.json"))
	if err != nil || !receipt.Valid() {
		t.Fatal("opaque historical parser contract changed", err)
	}
}

func TestPipxUpgradeInputRejectsVersionRuntimeAndReceiptSubstitution(t *testing.T) {
	for _, fault := range []string{"missing root", "duplicate root", "wrong version within old constraint", "wrong directory", "changed cached receipt", "UV receipt", "zero", "wrong Python"} {
		t.Run(fault, func(t *testing.T) {
			receipt, files := pipxToolUpgradeFixture(t, nil)
			python := runtimeidentity.PythonVersion
			switch fault {
			case "missing root":
				files = nil
			case "duplicate root":
				files = append(files, files[0])
			case "wrong version within old constraint":
				files[0].Directory = "haa_tool_fixture-1.1.0.dist-info"
				files[0].Body = bytes.ReplaceAll(files[0].Body, []byte("1.0.0"), []byte("1.1.0"))
			case "wrong directory":
				files[0].Directory = "foreign-1.0.0.dist-info"
			case "changed cached receipt":
				receipt.version = "1.1.0"
			case "UV receipt":
				receipt, _ = ParseUVToolReceipt(nativeToolFixture(t, "uv-installed.toml"))
			case "zero":
				receipt = ToolReceipt{}
			case "wrong Python":
				python = "3.14.8"
			}
			if result, err := NewPipxToolUpgradeInput(receipt, files, python); err == nil || result.Receipt().Valid() {
				t.Fatal("changed original accepted")
			}
		})
	}
}

func TestPipxUpgradeInputRejectsMixedNoncanonicalAndConstrainedControls(t *testing.T) {
	receipt, files := pipxToolUpgradeFixture(t, nil)
	input, err := NewPipxToolUpgradeInput(receipt, files, runtimeidentity.PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	control, err := input.Control()
	if err != nil {
		t.Fatal(err)
	}
	var wire pipxToolUpgradeInputWire
	if err := json.Unmarshal(control.Body(), &wire); err != nil {
		t.Fatal(err)
	}
	wire.Request = "haa-tool-fixture<2"
	constrained, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range [][]byte{
		constrained,
		append(bytes.Clone(control.Body()), '\n'),
		bytes.Replace(control.Body(), []byte("\"version\":1"), []byte("\"version\":2"), 1),
		bytes.Replace(control.Body(), []byte("\"version\":1"), []byte("\"version\":1,\"version\":1"), 1),
		bytes.Replace(control.Body(), []byte("\"version\":1"), []byte("\"version\":1,\"unknown\":true"), 1),
	} {
		c, err := domain.NewProjectControlFile(PipxToolUpgradeInputControlName, body, true)
		if err != nil {
			t.Fatal(err)
		}
		if result, err := ParsePipxToolUpgradeInput([]domain.ProjectControlFile{c}, runtimeidentity.PythonVersion); err == nil || result.Receipt().Valid() {
			t.Fatal("changed/noncanonical control accepted")
		}
	}
	uv, err := domain.NewProjectControlFile(UVToolUpgradeInputControlName, control.Body(), true)
	if err != nil {
		t.Fatal(err)
	}
	for _, controls := range [][]domain.ProjectControlFile{nil, {uv}, {control, control}, {control, uv}} {
		if result, err := ParsePipxToolUpgradeInput(controls, runtimeidentity.PythonVersion); err == nil || result.Receipt().Valid() {
			t.Fatal("mixed/absent original controls accepted")
		}
	}
	if _, err := (PipxToolUpgradeInput{}).Control(); err == nil {
		t.Fatal("zero input has a control")
	}
	if _, err := ParseUVToolUpgradeInput([]domain.ProjectControlFile{control}, runtimeidentity.PythonVersion); err == nil {
		t.Fatal("pipx input accepted as UV")
	}
}

func TestPipxUpgradeInputEnforcesCombinedEncodedByteBound(t *testing.T) {
	receipt, files := pipxToolUpgradeFixture(t, nil)
	padded := append(receipt.Body(), bytes.Repeat([]byte(" "), 3<<20)...)
	receipt, err := ParsePipxToolReceipt(padded)
	if err != nil {
		t.Fatal("individually bounded receipt", err)
	}
	if _, err := NewPipxToolUpgradeInput(receipt, files, runtimeidentity.PythonVersion); err == nil {
		t.Fatal("combined base64 bound ignored")
	}
}
