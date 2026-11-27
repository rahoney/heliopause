package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"path/filepath"
	"strings"
)

// materializePipxCandidate emits DATA for an explicit later isolated probe.
func materializePipxCandidate(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime materialize-pipx-candidate", flag.ContinueOnError)
	flags.SetOutput(out)
	root := flags.String("root", "", "absolute canonical source directory")
	input := flags.String("input", "", "absolute frozen pipx closure directory")
	output := flags.String("output", "", "absolute new external pipx-runtime.zip")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*root) || !filepath.IsAbs(*input) || !filepath.IsAbs(*output) || filepath.Clean(*output) != *output || filepath.Base(*output) != "pipx-runtime.zip" {
		return errors.New("pipx candidate requires explicit absolute source, input and output paths")
	}
	canonical, err := filepath.EvalSymlinks(*root)
	if err != nil || canonical != *root {
		return errors.New("pipx candidate source must be canonical and available")
	}
	if relative, err := filepath.Rel(canonical, *output); err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("pipx candidate output must be outside source tree")
	}
	supply, err := readPipxClosure(*input)
	if err != nil {
		return err
	}
	body, err := supply.closure.ZIP(supply.inputs)
	if err != nil {
		return err
	}
	if err := publishNativeAsset(*output, body, 0444); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Schema           string `json:"schema"`
		Asset            string `json:"asset"`
		SHA256           string `json:"sha256"`
		Size             int    `json:"size"`
		Files            int    `json:"files"`
		RuntimeQualified bool   `json:"runtime_qualified"`
	}{"helox.pipx-candidate/v1", filepath.Base(*output), hash(body), len(body), len(supply.closure.Files()), false})
}
