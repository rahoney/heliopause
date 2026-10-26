package pypi

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/pelletier/go-toml/v2"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const UVToolUpgradeInputControlName = "uv-tool-upgrade-input.json"

// UVToolUpgradeInput freezes original receipt/metadata and the separate native
// positional constraint. It grants no ownership, runtime admission or approval.
// Original installed code, interpreter, RECORD and startup files are never input.
type UVToolUpgradeInput struct {
	receipt  ToolReceipt
	original UpgradeInput
	request  UpgradeRequirement
}

func NewUVToolUpgradeInput(receipt ToolReceipt, request UpgradeRequirement, files []InstalledMetadataFile, python string) (UVToolUpgradeInput, error) {
	parsed, err := ParseUVToolReceipt(receipt.Body())
	if err != nil || receipt.Client() != "uv" || request.Reference().Source() != PublicPyPIProfile().Source() || parsed.Requirement().Project() != request.Project() {
		return UVToolUpgradeInput{}, errors.New("UV tool upgrade requires a receipt and positional requirement for the same public root")
	}
	// Parsing native recipe data is distinct from authorizing it as input.
	// Never forward an original local/index override into the controlled resolver.
	var wire uvToolReceiptWire
	if err := toml.Unmarshal(parsed.Body(), &wire); err != nil || wire.Tool == nil || wire.Tool.Options == nil {
		return UVToolUpgradeInput{}, errors.New("UV tool original recipe is unavailable")
	}
	options := wire.Tool.Options
	if options.NoIndex || len(options.FindLinks) != 0 || options.LinkMode != "copy" || !options.NoSources || !options.NoBuild {
		return UVToolUpgradeInput{}, errors.New("UV tool original recipe differs from the fixed public wheel-only resolver")
	}
	canonical, err := ParseUpgradeRequirement(request.Requirement())
	if err != nil || canonical.Requirement() != request.Requirement() {
		return UVToolUpgradeInput{}, errors.New("UV tool positional requirement is not canonical")
	}
	original, err := NewUpgradeInput(parsed.Requirement(), files, python)
	if err != nil {
		return UVToolUpgradeInput{}, err
	}
	root := false
	for _, installed := range original.Installed() {
		if installed.Project() == parsed.Requirement().Project() {
			if !parsed.Requirement().AcceptsVersion(installed.Version()) {
				return UVToolUpgradeInput{}, errors.New("UV tool original main version violates its receipt")
			}
			root = true
		}
	}
	if !root {
		return UVToolUpgradeInput{}, errors.New("UV tool original main distribution is absent")
	}
	input := UVToolUpgradeInput{receipt: parsed, original: original, request: canonical}
	if _, err := input.Control(); err != nil {
		return UVToolUpgradeInput{}, err
	}
	return input, nil
}

func (u UVToolUpgradeInput) Receipt() ToolReceipt        { return u.receipt }
func (u UVToolUpgradeInput) Original() UpgradeInput      { return u.original }
func (u UVToolUpgradeInput) Request() UpgradeRequirement { return u.request }

type uvToolUpgradeInputWire struct {
	Version  int             `json:"version"`
	Receipt  []byte          `json:"receipt"`
	Original json.RawMessage `json:"original"`
	Request  string          `json:"request"`
}

func (u UVToolUpgradeInput) Control() (domain.ProjectControlFile, error) {
	if !u.receipt.Valid() || u.request.Project() == "" {
		return domain.ProjectControlFile{}, errors.New("validated UV tool original input is required")
	}
	original, err := u.original.Control()
	if err != nil {
		return domain.ProjectControlFile{}, err
	}
	body, err := json.Marshal(uvToolUpgradeInputWire{1, u.receipt.Body(), original.Body(), u.request.Requirement()})
	if err != nil || len(body) > upgradeInputLimit {
		return domain.ProjectControlFile{}, errors.New("UV tool original input exceeds combined byte bounds")
	}
	return domain.NewProjectControlFile(UVToolUpgradeInputControlName, body, true)
}

func ParseUVToolUpgradeInput(controls []domain.ProjectControlFile, python string) (UVToolUpgradeInput, error) {
	if len(controls) != 1 || controls[0].Name() != UVToolUpgradeInputControlName || !controls[0].Present() {
		return UVToolUpgradeInput{}, errors.New("UV tool upgrade requires exactly one original input")
	}
	body := controls[0].Body()
	if len(body) == 0 || len(body) > upgradeInputLimit {
		return UVToolUpgradeInput{}, errors.New("UV tool original input exceeds byte bounds")
	}
	var wire uvToolUpgradeInputWire
	if json.Unmarshal(body, &wire) != nil || wire.Version != 1 {
		return UVToolUpgradeInput{}, errors.New("UV tool original input is invalid")
	}
	receipt, err := ParseUVToolReceipt(wire.Receipt)
	if err != nil {
		return UVToolUpgradeInput{}, err
	}
	request, err := ParseUpgradeRequirement(wire.Request)
	if err != nil {
		return UVToolUpgradeInput{}, err
	}
	control, err := domain.NewProjectControlFile(UpgradeInputControlName, wire.Original, true)
	if err != nil {
		return UVToolUpgradeInput{}, err
	}
	original, err := ParseUpgradeInput([]domain.ProjectControlFile{control}, python)
	if err != nil || !original.Requirement().SameConstraints(receipt.Requirement()) {
		return UVToolUpgradeInput{}, errors.New("UV tool original selection differs from its receipt")
	}
	input, err := NewUVToolUpgradeInput(receipt, request, original.MetadataFiles(), python)
	if err != nil {
		return UVToolUpgradeInput{}, err
	}
	canonical, err := input.Control()
	if err != nil || !bytes.Equal(canonical.Body(), body) {
		return UVToolUpgradeInput{}, errors.New("UV tool original input is not canonical")
	}
	return input, nil
}
