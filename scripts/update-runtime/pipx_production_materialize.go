package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// materializePipx binds all supplied bytes and derived assets to the single
// selected source owner. It does not execute Python or activate a runtime.
func materializePipx(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime materialize-pipx", flag.ContinueOnError)
	flags.SetOutput(out)
	root := flags.String("root", "", "absolute canonical source directory")
	input := flags.String("input", "", "absolute frozen full pipx supply directory")
	output := flags.String("output", "", "absolute new external pipx-runtime-linux-amd64.zip")
	noticesOutput := flags.String("notices-output", "", "absolute new sibling pipx-NOTICE.txt")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*root) || !filepath.IsAbs(*input) || !filepath.IsAbs(*output) || filepath.Clean(*output) != *output || filepath.Base(*output) != "pipx-runtime-linux-amd64.zip" || !filepath.IsAbs(*noticesOutput) || filepath.Dir(*noticesOutput) != filepath.Dir(*output) || filepath.Base(*noticesOutput) != "pipx-NOTICE.txt" {
		return errors.New("pipx materialization requires explicit absolute source/input/new ZIP/notices paths")
	}
	canonical, err := filepath.EvalSymlinks(*root)
	if err != nil || canonical != *root {
		return errors.New("pipx source must be canonical and available")
	}
	if relative, err := filepath.Rel(canonical, *output); err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("pipx output must be outside source tree")
	}
	pin, err := readPipxMaterializationOwner(canonical)
	if err != nil {
		return err
	}
	for _, f := range pin.Inputs {
		if strings.HasSuffix(f.Path, ".whl") {
			continue
		}
		data, err := readFrozenSupplyFile(canonical, "tools/pipx/supply/"+f.Path, 1<<20)
		if err != nil || hash(data) != f.SHA256 {
			return errors.New("pipx canonical supply metadata differs from owner")
		}
	}
	supply, err := readPipxClosure(*input)
	if err != nil {
		return err
	}
	body, err := supply.closure.ZIP(supply.inputs)
	if err != nil {
		return err
	}
	notices, err := pipxNotices(supply.closure, body)
	if err != nil {
		return err
	}
	if err := bindPipxOwner(pin, supply, body, notices); err != nil {
		return err
	}
	canonicalNotices, err := readFrozenSupplyFile(canonical, pin.Notices.Path, pin.Notices.Size)
	if err != nil || hash(canonicalNotices) != pin.Notices.SHA256 || int64(len(canonicalNotices)) != pin.Notices.Size {
		return errors.New("pipx canonical notices differ from owner")
	}
	for _, name := range []string{*output, *noticesOutput} {
		if _, err := os.Lstat(name); !errors.Is(err, os.ErrNotExist) {
			return errors.New("pipx output exists or cannot be checked")
		}
	}
	if err := publishNativeAsset(*output, body, 0444); err != nil {
		return err
	}
	if err := publishNativeAsset(*noticesOutput, notices, 0444); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Schema  string            `json:"schema"`
		Version string            `json:"version"`
		ZIP     pipxAssetIdentity `json:"zip"`
		Notices pipxAssetIdentity `json:"notices"`
	}{"helox.pipx-runtime-asset/v1", pin.Version, pin.Tool, pin.Notices})
}
