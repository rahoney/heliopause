package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

// verifyPipxPublish is an offline single-wheel supply fact check. It does not
// execute Python, extract archives, adopt pins or approve a dependency closure.
func verifyPipxPublish(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime verify-pipx-publish", flag.ContinueOnError)
	flags.SetOutput(out)
	input := flags.String("input", "", "absolute frozen directory: identity.json, provenance.json, exact pipx wheel")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*input) {
		return errors.New("pipx publish proof requires an explicit absolute directory")
	}
	identityBody, err := readFrozenSupplyFile(*input, "identity.json", 4096)
	if err != nil {
		return err
	}
	var identity releaseprovenance.PipxPublishIdentity
	if json.Unmarshal(identityBody, &identity) != nil {
		return errors.New("pipx publish identity is invalid")
	}
	proof, err := readFrozenSupplyFile(*input, "provenance.json", 1<<20)
	if err != nil {
		return err
	}
	wheel, err := readFrozenSupplyFile(*input, identity.WheelFilename(), 1<<20)
	if err != nil {
		return err
	}
	if hash(wheel) != identity.WheelSHA256 {
		return errors.New("pipx publish acquired wheel digest mismatch")
	}
	receipt, err := releaseprovenance.VerifyPipxPublish(context.Background(), proof, identity)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Schema           string                                `json:"schema"`
		Scope            string                                `json:"scope"`
		Identity         releaseprovenance.PipxPublishIdentity `json:"identity"`
		ProvenanceSHA256 string                                `json:"provenance_sha256"`
		Invocation       string                                `json:"invocation"`
		WheelSize        int                                   `json:"wheel_size"`
	}{"helox.pipx-publish-supply/v1", "SINGLE_WHEEL_PUBLISH_PROOF_ONLY", receipt.Identity(), receipt.ProvenanceSHA256(), receipt.Invocation(), len(wheel)})
}
