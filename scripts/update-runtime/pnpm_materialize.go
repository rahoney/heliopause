package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

// materializePNPM consumes the production owner/proof and an explicitly acquired
// archive. No network request, package-manager process or general extraction is
// permitted. A completed executable is published once outside the source tree.
func materializePNPM(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime materialize-pnpm", flag.ContinueOnError)
	flags.SetOutput(out)
	root := flags.String("root", "", "absolute canonical source directory")
	archive := flags.String("archive", "", "absolute acquired official archive")
	output := flags.String("output", "", "absolute new pnpm-linux-amd64 asset outside the source tree")
	noticesOutput := flags.String("notices-output", "", "absolute new pnpm-NOTICE.txt asset alongside the native output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*root) || !filepath.IsAbs(*archive) || !filepath.IsAbs(*output) || filepath.Base(*output) != "pnpm-linux-amd64" || !filepath.IsAbs(*noticesOutput) || filepath.Dir(*noticesOutput) != filepath.Dir(*output) || filepath.Base(*noticesOutput) != "pnpm-NOTICE.txt" {
		return errors.New("pnpm materialization requires explicit absolute source, archive and native output paths")
	}
	canonical, err := filepath.EvalSymlinks(*root)
	if err != nil || canonical != filepath.Clean(*root) {
		return errors.New("pnpm source must be canonical and available")
	}
	if relative, err := filepath.Rel(canonical, *output); err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("pnpm materialization output must be outside the source tree")
	}
	pin, err := readPNPMMaterializationOwner(canonical)
	if err != nil {
		return err
	}
	bundle, err := readFrozenSupplyFile(canonical, pin.Provenance.BundlePath, 1<<20)
	if err != nil || hash(bundle) != pin.Provenance.BundleSHA256 {
		return errors.New("pnpm production proof is missing or changed")
	}
	notices, err := readFrozenSupplyFile(canonical, pin.Notices.Path, pin.Notices.Size)
	if err != nil || int64(len(notices)) != pin.Notices.Size || hash(notices) != pin.Notices.SHA256 || !validPNPMNotices(pin.Provenance.SourceCommit, notices) {
		return errors.New("pnpm production notices are missing, changed or from another source")
	}
	identity := releaseprovenance.PNPMReleaseIdentity{
		Version: pin.Version, SourceCommit: pin.Provenance.SourceCommit,
		WorkflowRun: pin.Provenance.WorkflowRun, ArchiveSHA256: pin.Archive.SHA256,
	}
	receipt, err := releaseprovenance.VerifyPNPMRelease(context.Background(), bundle, identity)
	if err != nil {
		return err
	}
	compressed, err := readFrozenSupplyFile(filepath.Dir(*archive), filepath.Base(*archive), pin.Archive.Size)
	if err != nil || int64(len(compressed)) != pin.Archive.Size {
		return errors.New("pnpm official archive is missing or has a different size")
	}
	tool, body, err := receipt.ReadToolArchive(context.Background(), compressed)
	if err != nil {
		return err
	}
	if tool.SHA256() != pin.Tool.SHA256 || tool.Size() != pin.Tool.Size {
		return errors.New("pnpm authenticated native identity differs from the production owner")
	}
	if _, err := os.Lstat(*noticesOutput); !errors.Is(err, os.ErrNotExist) {
		return errors.New("pnpm notices output already exists or cannot be checked")
	}
	if err := publishNativeAsset(*output, body, 0555); err != nil {
		return err
	}
	if err := publishNativeAsset(*noticesOutput, notices, 0444); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Schema        string `json:"schema"`
		Name          string `json:"name"`
		Version       string `json:"version"`
		SHA256        string `json:"sha256"`
		Size          int64  `json:"size"`
		NoticesSHA256 string `json:"notices_sha256"`
	}{"helox.pnpm-native-asset/v1", filepath.Base(*output), identity.Version, tool.SHA256(), tool.Size(), pin.Notices.SHA256})
}

// Maintenance reads the selected source owner rather than importing the
// controller's generated runtime tuple. The canonical generator/drift checker
// remain responsible for the complete lock, including unrelated components.
type pnpmMaterializationOwner struct {
	Version      string `json:"pnpm_version"`
	Architecture string `json:"architecture"`
	Archive      struct {
		URL    string `json:"url"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
	} `json:"archive"`
	Provenance struct {
		SourceCommit string `json:"source_commit"`
		WorkflowRun  string `json:"workflow_run"`
		BundlePath   string `json:"bundle_path"`
		BundleSHA256 string `json:"bundle_sha256"`
	} `json:"provenance"`
	Tool struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
	} `json:"tool"`
	Notices struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
	} `json:"notices"`
}

func readPNPMMaterializationOwner(root string) (pnpmMaterializationOwner, error) {
	var pin pnpmMaterializationOwner
	body, err := readFrozenSupplyFile(root, runtimeFile, 1<<20)
	if err != nil {
		return pin, err
	}
	if _, err := jsonSpans(body); err != nil {
		return pin, err
	}
	var lock map[string]json.RawMessage
	if err := json.Unmarshal(body, &lock); err != nil {
		return pin, err
	}
	var schema int
	if json.Unmarshal(lock["schema_version"], &schema) != nil || schema != 5 {
		return pin, errors.New("pnpm runtime owner schema is invalid")
	}
	selected := lock["pnpm_tool"]
	fields, err := jsonSpans(selected)
	if err != nil {
		return pin, err
	}
	expected := []string{"", "/pnpm_version", "/architecture", "/archive", "/archive/url", "/archive/sha256", "/archive/size", "/provenance", "/provenance/source_commit", "/provenance/workflow_run", "/provenance/bundle_path", "/provenance/bundle_sha256", "/tool", "/tool/path", "/tool/sha256", "/tool/size", "/notices", "/notices/path", "/notices/sha256", "/notices/size"}
	if len(fields) != len(expected) {
		return pin, errors.New("pnpm selected owner fields are missing or unknown")
	}
	for _, field := range expected {
		if _, ok := fields[field]; !ok {
			return pin, errors.New("pnpm selected owner has a noncanonical field")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(selected))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&pin) != nil || decoder.Decode(new(any)) != io.EOF || !versionPattern.MatchString(pin.Version) || pin.Architecture != "amd64" || pin.Archive.URL != "https://github.com/pnpm/pnpm/releases/download/v"+pin.Version+"/pnpm-linux-x64.tar.gz" || !digestPattern.MatchString(pin.Archive.SHA256) || pin.Archive.Size <= 0 || pin.Archive.Size > 64<<20 || pin.Provenance.BundlePath != "tools/pnpm/release-proof.json" || !digestPattern.MatchString(pin.Provenance.BundleSHA256) || pin.Tool.Path != "pnpm" || !digestPattern.MatchString(pin.Tool.SHA256) || pin.Tool.Size <= 0 || pin.Tool.Size > 64<<20 || pin.Notices.Path != "tools/pnpm/NOTICE" || !digestPattern.MatchString(pin.Notices.SHA256) || pin.Notices.Size <= 0 || pin.Notices.Size > 64<<10 {
		return pnpmMaterializationOwner{}, errors.New("pnpm selected runtime owner is invalid")
	}
	return pin, nil
}
