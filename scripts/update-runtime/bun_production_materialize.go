package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

// materializeBun seals native/passive release files from the selected source
// owner. It grants no Host execution, installed activation or guest role.
func materializeBun(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime materialize-bun", flag.ContinueOnError)
	flags.SetOutput(out)
	root := flags.String("root", "", "absolute canonical source directory")
	archive := flags.String("archive", "", "absolute acquired official archive")
	output := flags.String("output", "", "absolute new bun-linux-amd64 outside the source tree")
	noticesOutput := flags.String("notices-output", "", "absolute new bun-NOTICE.txt alongside the native output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*root) || !filepath.IsAbs(*archive) || !filepath.IsAbs(*output) || filepath.Base(*output) != "bun-linux-amd64" || !filepath.IsAbs(*noticesOutput) || filepath.Dir(*noticesOutput) != filepath.Dir(*output) || filepath.Base(*noticesOutput) != "bun-NOTICE.txt" {
		return errors.New("bun materialization requires explicit absolute source, archive and native/notices output paths")
	}
	canonical, err := filepath.EvalSymlinks(*root)
	if err != nil || canonical != filepath.Clean(*root) {
		return errors.New("bun source must be canonical and available")
	}
	if relative, err := filepath.Rel(canonical, *output); err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("bun materialization output must be outside the source tree")
	}
	pin, err := readBunMaterializationOwner(canonical)
	if err != nil {
		return err
	}
	signed, err := readFrozenSupplyFile(canonical, pin.Publisher.ChecksumsPath, 64<<10)
	if err != nil || hash(signed) != pin.Publisher.ChecksumsSHA256 {
		return errors.New("bun production signed checksums are missing or changed")
	}
	notices, err := readFrozenSupplyFile(canonical, pin.Notices.Path, pin.Notices.Size)
	if err != nil || int64(len(notices)) != pin.Notices.Size || hash(notices) != pin.Notices.SHA256 || !validBunNotices(pin.SourceCommitObserved, notices) {
		return errors.New("bun production passive notices are missing, changed or from another source")
	}
	receipt, err := releaseprovenance.VerifyBunChecksums(context.Background(), signed, pin.Archive.SHA256, time.Now())
	if err != nil {
		return err
	}
	compressed, err := readFrozenSupplyFile(filepath.Dir(*archive), filepath.Base(*archive), pin.Archive.Size)
	if err != nil || int64(len(compressed)) != pin.Archive.Size {
		return errors.New("bun official archive is missing or has a different size")
	}
	tool, body, err := receipt.ReadToolArchive(context.Background(), compressed)
	if err != nil {
		return err
	}
	if tool.SHA256() != pin.Tool.SHA256 || tool.Size() != pin.Tool.Size {
		return errors.New("bun native bytes differ from the selected runtime owner")
	}
	if _, err := os.Lstat(*noticesOutput); !errors.Is(err, os.ErrNotExist) {
		return errors.New("bun notices output already exists or cannot be checked")
	}
	if err := publishNativeAsset(*output, body, 0555); err != nil {
		return err
	}
	if err := publishNativeAsset(*noticesOutput, notices, 0444); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Schema        string `json:"schema"`
		Asset         string `json:"asset"`
		Version       string `json:"version"`
		SHA256        string `json:"sha256"`
		Size          int64  `json:"size"`
		NoticesSHA256 string `json:"notices_sha256"`
	}{"helox.bun-native-asset/v1", filepath.Base(*output), pin.Version, tool.SHA256(), tool.Size(), pin.Notices.SHA256})
}
