package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func materializeSemgrep(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime materialize-semgrep", flag.ContinueOnError)
	flags.SetOutput(out)
	source := flags.String("root", "", "absolute source checkout with the selected runtime owner")
	input := flags.String("input", "", "absolute frozen core/source/notice input directory")
	output := flags.String("output", "", "absolute new external output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("semgrep supply does not accept positional arguments")
	}
	for _, directory := range []string{*source, *input, filepath.Dir(*output)} {
		if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
			return errors.New("semgrep supply directories must be absolute and canonical")
		}
		canonical, err := filepath.EvalSymlinks(directory)
		if err != nil || canonical != directory {
			return errors.New("semgrep supply directory is unavailable or a symlink")
		}
	}
	if !filepath.IsAbs(*output) || filepath.Clean(*output) != *output {
		return errors.New("semgrep supply output must be an absolute new directory")
	}
	for _, directory := range []string{*source, *input} {
		relative, err := filepath.Rel(directory, *output)
		if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return errors.New("semgrep supply output must be outside source and inputs")
		}
	}
	if _, err := os.Lstat(*output); !errors.Is(err, os.ErrNotExist) {
		return errors.New("semgrep supply output exists or cannot be checked")
	}
	pin, err := readSemgrepMaterializationOwner(*source)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(*input)
	if err != nil {
		return err
	}
	defer root.Close()
	repository, err := os.OpenRoot(*source)
	if err != nil {
		return err
	}
	defer repository.Close()
	for _, asset := range []semgrepAsset{pin.Rules, pin.Patch, pin.Worker} {
		if err := copySemgrepSourceMember(repository, io.Discard, semgrepSourcePackageFile{asset.Path, asset.SHA256, asset.Size, 0444}); err != nil {
			return err
		}
	}
	core := semgrepSourcePackageFile{pin.Tool.Path, pin.Tool.SHA256, pin.Tool.Size, 0555}
	if err := copySemgrepSourceMember(root, io.Discard, core); err != nil {
		return err
	}
	native, err := root.Open(pin.Tool.Path)
	if err != nil {
		return err
	}
	err = validateSemgrepNative(native)
	if closeErr := native.Close(); err != nil || closeErr != nil {
		return errors.Join(err, closeErr)
	}
	original, err := readSemgrepSupplyArchive(root, pin, false, nil)
	if err != nil {
		return err
	}
	notices, err := readSemgrepSupplyArchive(root, pin, true, nil)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(original.Files, notices.Files) {
		return errors.New("semgrep notice projection differs from its complete source")
	}
	if err := os.Mkdir(*output, 0700); err != nil {
		return err
	}
	assets := []struct {
		asset semgrepAsset
		root  *os.Root
		name  string
		mode  os.FileMode
	}{
		{pin.Tool, root, pin.Tool.Path, 0555},
		{pin.Sources, root, pin.Sources.Path, 0444},
		{pin.Notices, root, pin.Notices.Path, 0444},
		{pin.Rules, repository, "semgrep-rules.json", 0444},
	}
	for _, selected := range assets {
		asset := selected.asset
		if err := publishNativeAssetWith(filepath.Join(*output, selected.name), asset.Size, selected.mode, func(writer io.Writer) error {
			return copySemgrepSourceMember(selected.root, writer, semgrepSourcePackageFile{asset.Path, asset.SHA256, asset.Size, int64(selected.mode)})
		}); err != nil {
			return err
		}
	}
	return json.NewEncoder(out).Encode(struct {
		Schema           string `json:"schema"`
		Version          string `json:"semgrep_version"`
		NativeSHA256     string `json:"native_sha256"`
		SourceSHA256     string `json:"source_sha256"`
		NoticesSHA256    string `json:"notices_sha256"`
		RulesSHA256      string `json:"rules_sha256"`
		RuntimeQualified bool   `json:"runtime_qualified"`
	}{"helox.semgrep-supply/v1", pin.Version, pin.Tool.SHA256, pin.Sources.SHA256, pin.Notices.SHA256, pin.Rules.SHA256, false})
}
