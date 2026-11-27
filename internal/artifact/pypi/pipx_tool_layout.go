package pypi

import (
	"encoding/json"
	"errors"
	"path"
	"reflect"
)

// WithPipxLayout relocates inert receipt claims to caller-held paths. Pipx
// records its source interpreter separately from the environment interpreter;
// neither that claim nor this projection authorizes opening or executing it.
// The original receipt must remain a separate, unchanged control.
func (r ToolReceipt) WithPipxLayout(environment, binDirectory, sourceInterpreter string) (ToolReceipt, error) {
	if !r.Valid() || r.client != "pipx" || !validToolAbsolutePath(environment) || !validToolAbsolutePath(binDirectory) || !validToolAbsolutePath(sourceInterpreter) {
		return ToolReceipt{}, errors.New("pipx layout requires a parsed receipt and explicit roots and source interpreter")
	}
	original, err := ParsePipxToolReceipt(r.Body())
	if err != nil || !reflect.DeepEqual(original, r) {
		return ToolReceipt{}, errors.New("pipx receipt projection input changed")
	}
	// Raw fields preserve omission and null, including opaque stored pip_args.
	// Only app paths and source_interpreter are projection fields. No original
	// recipe data is returned as executable argv.
	var wire, main map[string]json.RawMessage
	if err := json.Unmarshal(r.Body(), &wire); err != nil {
		return ToolReceipt{}, err
	}
	if err := json.Unmarshal(wire["main_package"], &main); err != nil {
		return ToolReceipt{}, err
	}
	var entries []pipxPathWire
	if err := json.Unmarshal(main["app_paths"], &entries); err != nil {
		return ToolReceipt{}, err
	}
	for i := range entries {
		entries[i].Path = path.Join(environment, "bin", path.Base(entries[i].Path))
	}
	main["app_paths"], err = json.Marshal(entries)
	if err != nil {
		return ToolReceipt{}, err
	}
	wire["main_package"], err = json.Marshal(main)
	if err != nil {
		return ToolReceipt{}, err
	}
	wire["source_interpreter"], err = json.Marshal(pipxPathWire{Type: "Path", Path: sourceInterpreter})
	if err != nil {
		return ToolReceipt{}, err
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return ToolReceipt{}, err
	}
	projected, err := ParsePipxToolReceipt(body)
	if err != nil || !reflect.DeepEqual(projected.requirement, r.requirement) || projected.version != r.version || len(projected.commands) != len(r.commands) {
		return ToolReceipt{}, errors.New("pipx layout projection changed receipt semantics")
	}
	for i, command := range projected.commands {
		if command.Name != r.commands[i].Name {
			return ToolReceipt{}, errors.New("pipx layout projection changed command set")
		}
	}
	if err := projected.ValidateLayout(environment, binDirectory, sourceInterpreter); err != nil {
		return ToolReceipt{}, err
	}
	return projected, nil
}
