package pypi

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const PipxToolUpgradeInputControlName = "pipx-tool-upgrade-input.json"

// PipxToolUpgradeInput freezes inert original metadata separately from pipx's
// unconstrained upgrade request. It grants no source, ownership or execution
// authority. Stored interpreter paths and pip arguments are never argv.
type PipxToolUpgradeInput struct {
	receipt  ToolReceipt
	original UpgradeInput
	request  UpgradeRequirement
}

func NewPipxToolUpgradeInput(receipt ToolReceipt, files []InstalledMetadataFile, python string) (PipxToolUpgradeInput, error) {
	parsed, err := ParsePipxToolReceipt(receipt.Body())
	if err != nil || receipt.Client() != "pipx" || !reflect.DeepEqual(parsed, receipt) || python != runtimeidentity.PythonVersion {
		return PipxToolUpgradeInput{}, errors.New("pipx upgrade requires an unchanged receipt and pinned Python")
	}
	var wire pipxToolReceiptWire
	if err := json.Unmarshal(parsed.Body(), &wire); err != nil || wire.Main == nil || !supportedPipxUpgradeRecipe(wire.Main.PipArgs) {
		return PipxToolUpgradeInput{}, errors.New("pipx original recipe is outside the fixed public wheel-only lane")
	}
	original, err := NewUpgradeInput(parsed.Requirement(), files, python)
	if err != nil {
		return PipxToolUpgradeInput{}, err
	}
	root := false
	for _, installed := range original.Installed() {
		if installed.Project() == parsed.Requirement().Project() {
			if installed.Version() != parsed.Version() {
				return PipxToolUpgradeInput{}, errors.New("pipx original main version differs from receipt")
			}
			root = true
		}
	}
	if !root {
		return PipxToolUpgradeInput{}, errors.New("pipx original main distribution is absent")
	}
	request, err := parsed.UpgradeRequirement()
	if err != nil {
		return PipxToolUpgradeInput{}, err
	}
	input := PipxToolUpgradeInput{receipt: parsed, original: original, request: request}
	if _, err := input.Control(); err != nil {
		return PipxToolUpgradeInput{}, err
	}
	return input, nil
}

// This recognizes inert old recipe semantics, never an executable recipe. The
// native resolver must supply its own nonempty fixed options on every upgrade.
func supportedPipxUpgradeRecipe(arguments []string) bool {
	seen := map[string]bool{}
	for i := 0; i < len(arguments); i++ {
		name, value, equal := strings.Cut(arguments[i], "=")
		if seen[name] {
			return false
		}
		seen[name] = true
		switch name {
		case "--no-build", "--no-config", "--no-cache", "--no-python-downloads", "--no-managed-python", "--no-sources":
			if equal {
				return false
			}
		case "--index-url", "--link-mode":
			if !equal {
				i++
				if i >= len(arguments) {
					return false
				}
				value = arguments[i]
			}
			if name == "--index-url" && value != PublicPyPIProfile().IndexURL() || name == "--link-mode" && value != "copy" {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (u PipxToolUpgradeInput) Receipt() ToolReceipt        { return u.receipt }
func (u PipxToolUpgradeInput) Original() UpgradeInput      { return u.original }
func (u PipxToolUpgradeInput) Request() UpgradeRequirement { return u.request }

type pipxToolUpgradeInputWire struct {
	Version  int             `json:"version"`
	Receipt  []byte          `json:"receipt"`
	Original json.RawMessage `json:"original"`
	Request  string          `json:"request"`
}

func (u PipxToolUpgradeInput) Control() (domain.ProjectControlFile, error) {
	if !u.receipt.Valid() || u.request.Project() == "" {
		return domain.ProjectControlFile{}, errors.New("validated pipx original input is required")
	}
	original, err := u.original.Control()
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	body, err := json.Marshal(pipxToolUpgradeInputWire{1, u.receipt.Body(), original.Body(), u.request.Requirement()})
	if err != nil || len(body) > upgradeInputLimit {
		return domain.ProjectControlFile{}, errors.New("pipx original input exceeds combined byte bounds")
	}
	return domain.NewProjectControlFile(PipxToolUpgradeInputControlName, body, true)
}

func ParsePipxToolUpgradeInput(controls []domain.ProjectControlFile, python string) (PipxToolUpgradeInput, error) {
	if len(controls) != 1 || controls[0].Name() != PipxToolUpgradeInputControlName || !controls[0].Present() {
		return PipxToolUpgradeInput{}, errors.New("pipx upgrade requires exactly one original input")
	}
	body := controls[0].Body()
	if len(body) == 0 || len(body) > upgradeInputLimit {
		return PipxToolUpgradeInput{}, errors.New("pipx original input exceeds byte bounds")
	}
	var wire pipxToolUpgradeInputWire
	if json.Unmarshal(body, &wire) != nil || wire.Version != 1 {
		return PipxToolUpgradeInput{}, errors.New("pipx original input is invalid")
	}
	receipt, err := ParsePipxToolReceipt(wire.Receipt)
	if err != nil {
		return PipxToolUpgradeInput{}, err
	}
	control, err := domain.NewProjectControlFile(UpgradeInputControlName, wire.Original, true)
	if err != nil {
		return PipxToolUpgradeInput{}, err
	}
	original, err := ParseUpgradeInput([]domain.ProjectControlFile{control}, python)
	if err != nil || !original.Requirement().SameConstraints(receipt.Requirement()) {
		return PipxToolUpgradeInput{}, errors.New("pipx original selection differs from its receipt")
	}
	input, err := NewPipxToolUpgradeInput(receipt, original.MetadataFiles(), python)
	if err != nil {
		return PipxToolUpgradeInput{}, err
	}
	canonical, err := input.Control()
	if err != nil || !bytes.Equal(canonical.Body(), body) {
		return PipxToolUpgradeInput{}, errors.New("pipx original input is not canonical")
	}
	return input, nil
}

// NativeUVEnrollmentRecipeSupported narrows the first native original format.
// It is a static data-format gate, never source/ownership/execution authority.
func (u PipxToolUpgradeInput) NativeUVEnrollmentRecipeSupported() bool {
	if !u.receipt.Valid() || u.receipt.Client() != "pipx" {
		return false
	}
	var wire pipxToolReceiptWire
	if json.Unmarshal(u.receipt.Body(), &wire) != nil || wire.Main == nil || !supportedPipxUpgradeRecipe(wire.Main.PipArgs) {
		return false
	}
	noBuild, noSources, copyMode := false, false, false
	for i := 0; i < len(wire.Main.PipArgs); i++ {
		name, value, equal := strings.Cut(wire.Main.PipArgs[i], "=")
		switch name {
		case "--no-build":
			noBuild = true
		case "--no-sources":
			noSources = true
		case "--link-mode":
			if !equal {
				i++
				if i >= len(wire.Main.PipArgs) {
					return false
				}
				value = wire.Main.PipArgs[i]
			}
			copyMode = value == "copy"
		}
	}
	return noBuild && noSources && copyMode
}
