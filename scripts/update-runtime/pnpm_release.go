package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

// verifyPNPMRelease compares acquired archive bytes with authenticated upstream
// provenance. It neither extracts/executes the tool nor selects a runtime pin.
func verifyPNPMRelease(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime verify-pnpm-release", flag.ContinueOnError)
	flags.SetOutput(out)
	input := flags.String("input", "", "absolute frozen directory: identity.json, bundle.json, pnpm-linux-x64.tar.gz")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*input) {
		return errors.New("pnpm release proof requires an explicit absolute directory")
	}
	supply, archive, err := readPNPMSupply(*input)
	if err != nil {
		return err
	}
	type nativeRecord struct {
		SHA256    string   `json:"sha256"`
		Size      int64    `json:"size"`
		Libraries []string `json:"libraries"`
	}
	return json.NewEncoder(out).Encode(struct {
		Schema     string                                `json:"schema"`
		Identity   releaseprovenance.PNPMReleaseIdentity `json:"identity"`
		Bundle     string                                `json:"bundle_sha256"`
		Bytes      int                                   `json:"archive_bytes"`
		Invocation string                                `json:"invocation"`
		Tool       nativeRecord                          `json:"tool"`
	}{"helox.pnpm-release-supply/v2", supply.Identity, hash(supply.Bundle), len(archive), supply.Invocation, nativeRecord{supply.ToolSHA256, supply.ToolSize, supply.Libraries}})
}
