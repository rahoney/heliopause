package pypi

import (
	"errors"
	"path"
	"reflect"

	"github.com/pelletier/go-toml/v2"
)

// WithUVLayout projects an unapproved native receipt onto explicitly held Host
// paths. It preserves the original requirement/options and command set. The
// original receipt must remain a separate control; neither value grants file
// ownership, execution permission or approval.
func (r ToolReceipt) WithUVLayout(environment, binDirectory string) (ToolReceipt, error) {
	return r.withUVLayout(environment, binDirectory, path.Join(environment, "bin", "python"))
}

// WithUVSelectionLayout relocates only inert native metadata into the private
// selection session. The interpreter is the existing pinned image Python, never
// a path supplied by the original installation. The original receipt remains a
// separate control. This projection grants no execution or ownership authority.
func (r ToolReceipt) WithUVSelectionLayout(environment, binDirectory string) (ToolReceipt, error) {
	return r.withUVLayout(environment, binDirectory, "/usr/local/bin/python")
}

func (r ToolReceipt) withUVLayout(environment, binDirectory, interpreter string) (ToolReceipt, error) {
	if !r.Valid() || r.client != "uv" || !validToolAbsolutePath(environment) || !validToolAbsolutePath(binDirectory) {
		return ToolReceipt{}, errors.New("UV tool layout requires a parsed receipt and explicit roots")
	}
	original, err := ParseUVToolReceipt(r.Body())
	if err != nil || !reflect.DeepEqual(original.requirement, r.requirement) || !reflect.DeepEqual(original.commands, r.commands) {
		return ToolReceipt{}, errors.New("UV tool receipt projection input changed")
	}
	// Map decoding preserves omission of native options such as index-url. A
	// typed marshal with zero-valued optional fields could invent an empty URL.
	var wire map[string]any
	if err := toml.Unmarshal(r.Body(), &wire); err != nil {
		return ToolReceipt{}, err
	}
	tool, ok := wire["tool"].(map[string]any)
	if !ok {
		return ToolReceipt{}, errors.New("UV tool receipt table is unavailable")
	}
	entries, ok := tool["entrypoints"].([]any)
	if !ok || len(entries) != len(r.commands) {
		return ToolReceipt{}, errors.New("UV tool receipt entrypoint table differs")
	}
	tool["python"] = interpreter
	for _, value := range entries {
		entry, ok := value.(map[string]any)
		if !ok {
			return ToolReceipt{}, errors.New("UV tool receipt entrypoint is unavailable")
		}
		name, ok := entry["name"].(string)
		if !ok || !validToolCommand(name) {
			return ToolReceipt{}, errors.New("UV tool receipt command is invalid")
		}
		entry["install-path"] = path.Join(binDirectory, name)
	}
	body, err := toml.Marshal(wire)
	if err != nil {
		return ToolReceipt{}, err
	}
	projected, err := ParseUVToolReceipt(body)
	if err != nil || !reflect.DeepEqual(projected.requirement, r.requirement) || len(projected.commands) != len(r.commands) {
		return ToolReceipt{}, errors.New("UV tool layout projection changed receipt semantics")
	}
	for i, command := range projected.commands {
		if command.Name != r.commands[i].Name {
			return ToolReceipt{}, errors.New("UV tool layout projection changed command set")
		}
	}
	if err := projected.ValidateLayout(environment, binDirectory, interpreter); err != nil {
		return ToolReceipt{}, err
	}
	return projected, nil
}
