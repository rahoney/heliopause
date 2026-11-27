package pypi

import (
	"bytes"
	"errors"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type uvToolReceiptWire struct {
	Tool *struct {
		Requirements []struct {
			Name      string `toml:"name"`
			Specifier string `toml:"specifier"`
		} `toml:"requirements"`
		Python      string `toml:"python"`
		Entrypoints []struct {
			Name string `toml:"name"`
			Path string `toml:"install-path"`
			From string `toml:"from"`
		} `toml:"entrypoints"`
		// Inert recipe data only. A later resolver must compare the exact recipe;
		// these claims must never be forwarded as native arguments.
		Options *struct {
			IndexURL  string   `toml:"index-url"`
			NoIndex   bool     `toml:"no-index"`
			FindLinks []string `toml:"find-links"`
			LinkMode  string   `toml:"link-mode"`
			NoSources bool     `toml:"no-sources"`
			NoBuild   bool     `toml:"no-build"`
		} `toml:"options"`
	} `toml:"tool"`
}

// ParseUVToolReceipt reads the reviewed UV 0.13.4 single-public-root subset.
// The native receipt has no schema version or installed distribution version;
// caller supply identity and independent archive/inventory binding remain required.
func ParseUVToolReceipt(body []byte) (ToolReceipt, error) {
	if !validToolText(body) {
		return ToolReceipt{}, errors.New("UV tool receipt is absent, invalid or exceeds bounds")
	}
	var wire uvToolReceiptWire
	if err := toml.NewDecoder(bytes.NewReader(body)).DisallowUnknownFields().Decode(&wire); err != nil {
		return ToolReceipt{}, err
	}
	if wire.Tool == nil || len(wire.Tool.Requirements) != 1 || !validToolAbsolutePath(wire.Tool.Python) {
		return ToolReceipt{}, errors.New("UV tool requires one public root and explicit interpreter")
	}
	if wire.Tool.Options != nil && wire.Tool.Options.IndexURL != "" && wire.Tool.Options.IndexURL != PublicPyPIProfile().IndexURL() && wire.Tool.Options.IndexURL != strings.TrimSuffix(PublicPyPIProfile().IndexURL(), "/") {
		return ToolReceipt{}, errors.New("UV tool receipt index is outside the public source profile")
	}
	root := wire.Tool.Requirements[0]
	project, err := NormalizeProjectName(root.Name)
	requirement, reqErr := parseUVProjectRequirement(root.Name + root.Specifier)
	if err != nil || project != root.Name || reqErr != nil || requirement.Project() != project {
		return ToolReceipt{}, errors.New("UV tool root requirement is unsupported")
	}
	var names, paths []string
	for _, ep := range wire.Tool.Entrypoints {
		if ep.From != project {
			return ToolReceipt{}, errors.New("UV tool command is not from its main package")
		}
		names = append(names, ep.Name)
		paths = append(paths, ep.Path)
	}
	commands, err := toolCommands(names, paths)
	if err != nil {
		return ToolReceipt{}, err
	}
	return ToolReceipt{client: "uv", interpreter: wire.Tool.Python, requirement: requirement, commands: commands, body: bytes.Clone(body)}, nil
}
