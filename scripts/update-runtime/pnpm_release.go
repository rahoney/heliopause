package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	identityBody, err := readFrozenSupplyFile(*input, "identity.json", 4096)
	if err != nil {
		return err
	}
	var identity releaseprovenance.PNPMReleaseIdentity
	decoder := json.NewDecoder(bytes.NewReader(identityBody))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&identity) != nil || decoder.Decode(new(any)) != io.EOF {
		return errors.New("pnpm release identity is invalid")
	}
	proof, err := readFrozenSupplyFile(*input, "bundle.json", 1<<20)
	if err != nil {
		return err
	}
	receipt, err := releaseprovenance.VerifyPNPMRelease(context.Background(), proof, identity)
	if err != nil {
		return err
	}
	archive, err := readFrozenSupplyFile(*input, "pnpm-linux-x64.tar.gz", 64<<20)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(archive)
	if hex.EncodeToString(sum[:]) != identity.ArchiveSHA256 {
		return errors.New("pnpm acquired archive differs from signed subject")
	}
	return json.NewEncoder(out).Encode(struct {
		Schema     string                                `json:"schema"`
		Identity   releaseprovenance.PNPMReleaseIdentity `json:"identity"`
		Bundle     string                                `json:"bundle_sha256"`
		Bytes      int                                   `json:"archive_bytes"`
		Invocation string                                `json:"invocation"`
	}{"helox.pnpm-release-supply/v1", receipt.Identity(), receipt.BundleSHA256(), len(archive), receipt.Invocation()})
}
