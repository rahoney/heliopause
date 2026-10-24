package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

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
	if err := publishPNPMAsset(*output, body, 0555); err != nil {
		return err
	}
	if err := publishPNPMAsset(*noticesOutput, notices, 0444); err != nil {
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

func publishPNPMAsset(output string, body []byte, mode os.FileMode) error {
	if mode != 0555 && mode != 0444 || len(body) == 0 {
		return errors.New("pnpm asset mode or body is invalid")
	}
	parent := filepath.Dir(output)
	canonical, err := filepath.EvalSymlinks(parent)
	if err != nil || canonical != parent {
		return errors.New("pnpm asset parent is unavailable or not canonical")
	}
	directory, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer directory.Close()
	before, err := directory.Stat(".")
	if err != nil || !pnpmOutputDirectoryOwned(before) {
		return errors.New("pnpm asset parent is not an owned protected directory")
	}
	if _, err := directory.Lstat(filepath.Base(output)); !errors.Is(err, os.ErrNotExist) {
		return errors.New("pnpm native output already exists or cannot be checked")
	}
	temporary := ".pnpm-" + rand.Text()
	file, err := directory.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	created, err := file.Stat()
	if err != nil {
		return err
	}
	defer func() {
		// A foreign replacement is never removed, even on failure.
		if info, err := directory.Lstat(temporary); err == nil && os.SameFile(created, info) {
			_ = directory.Remove(temporary)
		}
	}()
	if count, err := file.Write(body); err != nil || count != len(body) {
		return errors.Join(errors.New("pnpm native asset write was incomplete"), err)
	}
	if err := errors.Join(file.Chmod(mode), file.Sync()); err != nil {
		return err
	}
	readback, err := directory.Open(temporary)
	if err != nil {
		return err
	}
	readInfo, statErr := readback.Stat()
	readBody, readErr := io.ReadAll(io.LimitReader(readback, int64(len(body))+1))
	closeErr := readback.Close()
	if statErr != nil || readErr != nil || closeErr != nil || !os.SameFile(created, readInfo) || len(readBody) != len(body) || hash(readBody) != hash(body) {
		return errors.New("pnpm native asset readback differs from the authenticated bytes")
	}
	current, err := os.Lstat(parent)
	if err != nil || !os.SameFile(before, current) || !pnpmOutputDirectoryOwned(current) {
		return errors.New("pnpm asset parent changed before publication")
	}
	staged, err := directory.Lstat(temporary)
	if err != nil || !staged.Mode().IsRegular() || !os.SameFile(created, staged) {
		return errors.New("pnpm temporary native asset changed")
	}
	// Link is atomic and refuses any existing destination. Both operands stay
	// within the opened parent, including if its pathname is renamed.
	if err := directory.Link(temporary, filepath.Base(output)); err != nil {
		return err
	}
	published, err := directory.Lstat(filepath.Base(output))
	if err != nil || !os.SameFile(created, published) || published.Mode().Perm() != mode || published.Size() != int64(len(body)) {
		return errors.New("pnpm native asset publication changed")
	}
	if err := directory.Remove(temporary); err != nil {
		return err
	}
	parentFile, err := directory.Open(".")
	if err != nil {
		return err
	}
	if err := errors.Join(parentFile.Sync(), parentFile.Close()); err != nil {
		return err
	}
	current, err = os.Lstat(parent)
	if err != nil || !os.SameFile(before, current) || !pnpmOutputDirectoryOwned(current) {
		return errors.New("pnpm asset parent changed after publication")
	}
	return nil
}

func pnpmOutputDirectoryOwned(info os.FileInfo) bool {
	owner, ok := info.Sys().(*syscall.Stat_t)
	return ok && owner.Uid == uint32(os.Geteuid()) && info.IsDir() && info.Mode().Perm()&0022 == 0
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
	if json.Unmarshal(lock["schema_version"], &schema) != nil || schema != 3 {
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
