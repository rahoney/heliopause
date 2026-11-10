package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"path/filepath"
	"strings"
	"time"
)

// materializeBunCandidate produces one authenticated candidate for subsequent
// isolated queries. It neither runs Bun nor registers a production runtime.
func materializeBunCandidate(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime materialize-bun-candidate", flag.ContinueOnError)
	flags.SetOutput(out)
	root := flags.String("root", "", "absolute canonical source directory")
	input := flags.String("input", "", "absolute frozen Bun supply directory")
	output := flags.String("output", "", "absolute new bun-candidate outside the source tree")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*root) || !filepath.IsAbs(*input) || !filepath.IsAbs(*output) || filepath.Base(*output) != "bun-candidate" {
		return errors.New("bun candidate requires explicit absolute source, input and output paths")
	}
	canonical, err := filepath.EvalSymlinks(*root)
	if err != nil || canonical != *root {
		return errors.New("bun candidate source must be canonical and available")
	}
	if relative, err := filepath.Rel(canonical, *output); err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("bun candidate output must be outside the source tree")
	}
	supply, err := readBunSupply(*input, time.Now())
	if err != nil {
		return err
	}
	if err := publishNativeAsset(*output, supply.native, 0555); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Schema                string `json:"schema"`
		Asset                 string `json:"asset"`
		ArchiveSHA256         string `json:"archive_sha256"`
		SHA256                string `json:"sha256"`
		Size                  int64  `json:"size"`
		NativeVersionVerified bool   `json:"native_version_verified"`
		RuntimeQualified      bool   `json:"runtime_qualified"`
	}{"helox.bun-native-candidate/v1", filepath.Base(*output), supply.tool.ArchiveSHA256(), supply.tool.SHA256(), supply.tool.Size(), false, false})
}
