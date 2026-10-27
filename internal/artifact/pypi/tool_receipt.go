package pypi

import (
	"bytes"
	"errors"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ToolReceipt is unapproved native metadata, never proof of ownership, source,
// runtime admission or Policy. Commands must also match authenticated wheels,
// the complete installed inventory and the held publication transaction.
type ToolReceipt struct {
	client, environment, interpreter, version string
	requirement                               UpgradeRequirement
	commands                                  []ToolCommand
	body                                      []byte
}

// ToolCommand records a native path claim. UV claims an exposed destination;
// pipx claims a script inside its environment. Neither path is opened here.
type ToolCommand struct {
	Name string
	Path string
}

func (r ToolReceipt) Valid() bool                     { return r.client != "" }
func (r ToolReceipt) Client() string                  { return r.client }
func (r ToolReceipt) Requirement() UpgradeRequirement { return r.requirement }
func (r ToolReceipt) Version() string                 { return r.version }

// InterpreterPath is an unapproved metadata claim; callers must bind a held
// alias/file snapshot independently and must never execute this path as input.
func (r ToolReceipt) InterpreterPath() string { return r.interpreter }
func (r ToolReceipt) Commands() []ToolCommand { return append([]ToolCommand(nil), r.commands...) }
func (r ToolReceipt) Body() []byte            { return bytes.Clone(r.body) }

// UpgradeRequirement preserves the actual client semantics: UV retains its
// receipt constraint; pipx 1.18.4 explicitly removes version constraints.
func (r ToolReceipt) UpgradeRequirement() (UpgradeRequirement, error) {
	if !r.Valid() {
		return UpgradeRequirement{}, errors.New("tool receipt is unavailable")
	}
	if r.client == "pipx" {
		return ParseUpgradeRequirement(r.requirement.Project())
	}
	return r.requirement, nil
}

// ValidateLayout compares claims with caller-held paths. Pipx metadata does not
// name its external bin directory; ValidateExposedCommands must bind that claim
// separately. Neither method proves filesystem ownership or stability.
func (r ToolReceipt) ValidateLayout(environment, binDirectory, interpreter string) error {
	if !r.Valid() || !validToolAbsolutePath(environment) || !validToolAbsolutePath(binDirectory) || !validToolAbsolutePath(interpreter) || r.interpreter != interpreter || path.Base(environment) != r.requirement.Project() || environment == binDirectory || strings.HasPrefix(binDirectory+"/", environment+"/") || strings.HasPrefix(environment+"/", binDirectory+"/") {
		return errors.New("native tool layout or interpreter differs from held target")
	}
	if r.client == "pipx" && r.environment != path.Base(environment) {
		return errors.New("pipx environment differs from held target")
	}
	for _, command := range r.commands {
		expected := path.Join(binDirectory, command.Name)
		if r.client == "pipx" {
			expected = path.Join(environment, "bin", command.Name)
		}
		if command.Path != expected {
			return errors.New("native tool command differs from held target")
		}
	}
	return nil
}

// ValidateExposedCommands compares a separately captured, exact command-link
// inventory. Capture identity and whole-filesystem ownership belong to the
// held transaction, not this inert parser.
func (r ToolReceipt) ValidateExposedCommands(environment, binDirectory, interpreter string, links map[string]string) error {
	if err := r.ValidateLayout(environment, binDirectory, interpreter); err != nil {
		return err
	}
	if len(links) != len(r.commands) {
		return errors.New("native exposed command inventory is incomplete or has extra members")
	}
	for _, command := range r.commands {
		if links[path.Join(binDirectory, command.Name)] != path.Join(environment, "bin", command.Name) {
			return errors.New("native exposed command link differs from held layout")
		}
	}
	return nil
}

func validToolText(body []byte) bool {
	return len(body) > 0 && len(body) <= upgradeInputLimit && utf8.Valid(body) && bytes.IndexByte(body, 0) < 0
}

func validToolAbsolutePath(value string) bool {
	if value == "/" || len(value) > 4096 || !path.IsAbs(value) || path.Clean(value) != value || strings.ContainsAny(value, "\\:") || !utf8.ValidString(value) {
		return false
	}
	for _, c := range value {
		if unicode.IsControl(c) {
			return false
		}
	}
	return true
}

func validToolCommand(name string) bool {
	if name == "" || len(name) > 255 || name == "." || name == ".." || strings.HasPrefix(name, "-") {
		return false
	}
	for _, c := range name {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.' {
			continue
		}
		return false
	}
	return true
}

func toolCommands(names []string, paths []string) ([]ToolCommand, error) {
	if len(names) == 0 || len(names) > 256 || len(names) != len(paths) {
		return nil, errors.New("native tool command set is absent or exceeds bounds")
	}
	seen := map[string]bool{}
	result := make([]ToolCommand, 0, len(names))
	for i, name := range names {
		key := strings.ToLower(name)
		if !validToolCommand(name) || seen[key] || !validToolAbsolutePath(paths[i]) || path.Base(paths[i]) != name {
			return nil, errors.New("native tool command is unsafe, repeated or mismatched")
		}
		seen[key] = true
		result = append(result, ToolCommand{Name: name, Path: paths[i]})
	}
	return result, nil
}
