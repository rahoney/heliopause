package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

// Plans contain only authenticated small metadata. The complete archive stays
// in owned external state and is authenticated again on every render/apply.
type pnpmSupplyMetadata struct {
	Identity    releaseprovenance.PNPMReleaseIdentity `json:"identity"`
	Bundle      []byte                                `json:"bundle"`
	ArchiveSize int64                                 `json:"archive_size"`
	ToolSHA256  string                                `json:"tool_sha256"`
	ToolSize    int64                                 `json:"tool_size"`
	Libraries   []string                              `json:"libraries"`
	Notices     []byte                                `json:"notices,omitempty"`
	Invocation  string                                `json:"invocation"`
}

func readPNPMSupply(input string) (*pnpmSupplyMetadata, []byte, error) {
	if !filepath.IsAbs(input) {
		return nil, nil, errors.New("pnpm supply requires an absolute frozen proof directory")
	}
	body, err := readFrozenSupplyFile(input, "identity.json", 4096)
	if err != nil {
		return nil, nil, err
	}
	if _, err := jsonSpans(body); err != nil {
		return nil, nil, err
	}
	u := &pnpmSupplyMetadata{}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(&u.Identity) != nil || d.Decode(new(any)) != io.EOF {
		return nil, nil, errors.New("pnpm release identity is invalid")
	}
	u.Bundle, err = readFrozenSupplyFile(input, "bundle.json", 1<<20)
	if err != nil {
		return nil, nil, err
	}
	receipt, err := releaseprovenance.VerifyPNPMRelease(context.Background(), u.Bundle, u.Identity)
	if err != nil {
		return nil, nil, err
	}
	archive, err := readFrozenSupplyFile(input, "pnpm-linux-x64.tar.gz", 64<<20)
	if err != nil {
		return nil, nil, err
	}
	tool, err := receipt.VerifyToolArchive(context.Background(), archive)
	if err != nil {
		return nil, nil, err
	}
	u.ArchiveSize = int64(len(archive))
	u.ToolSHA256 = tool.SHA256()
	u.ToolSize = tool.Size()
	u.Libraries = tool.Libraries()
	u.Invocation = receipt.Invocation()
	return u, archive, nil
}

func freezePNPMSupply(version, input, state string) (metadata, error) {
	u, archive, err := readPNPMSupply(input)
	if err != nil {
		return metadata{}, err
	}
	u.Notices, err = readFrozenSupplyFile(input, "NOTICE", 64<<10)
	if err != nil || !validPNPMNotices(u.Identity.SourceCommit, u.Notices) {
		return metadata{}, errors.New("pnpm plan requires explicit source-bound notices")
	}
	if u.Identity.Version != version {
		return metadata{}, errors.New("pnpm selected version and supply identity differ")
	}
	directory := filepath.Join(state, "pnpm-supply")
	if err := os.Mkdir(directory, 0700); err != nil {
		return metadata{}, err
	}
	name := filepath.Join(directory, u.Identity.ArchiveSHA256+".tar.gz")
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return metadata{}, err
	}
	_, writeErr := file.Write(archive)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return metadata{}, err
	}
	for _, path := range []string{directory, state} {
		dir, err := os.Open(path)
		if err != nil {
			return metadata{}, err
		}
		if err := errors.Join(dir.Sync(), dir.Close()); err != nil {
			return metadata{}, err
		}
	}
	m := metadata{Architecture: "amd64", PNPM: u}
	if err := verifyFrozenPNPMSupply(version, m, state); err != nil {
		return metadata{}, err
	}
	return m, nil
}

func verifyFrozenPNPMSupply(version string, m metadata, state string) error {
	u := m.PNPM
	if u == nil || !validPNPMNotices(u.Identity.SourceCommit, u.Notices) || u.Identity.Version != version || u.ArchiveSize <= 0 || u.ArchiveSize > 64<<20 {
		return errors.New("frozen pnpm supply/version missing or excessive")
	}
	receipt, err := releaseprovenance.VerifyPNPMRelease(context.Background(), u.Bundle, u.Identity)
	if err != nil {
		return err
	}
	directory := filepath.Join(state, "pnpm-supply")
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil || resolved != directory {
		return errors.New("frozen pnpm directory changed or is a symlink")
	}
	info, err := os.Lstat(directory)
	if err != nil || !frozenSupplyOwned(info, true) {
		return errors.New("frozen pnpm directory ownership or mode changed")
	}
	name := u.Identity.ArchiveSHA256 + ".tar.gz"
	info, err = os.Lstat(filepath.Join(directory, name))
	if err != nil || !frozenSupplyOwned(info, false) {
		return errors.New("frozen pnpm archive ownership or mode changed")
	}
	archive, err := readFrozenSupplyFile(directory, name, u.ArchiveSize)
	if err != nil {
		return err
	}
	tool, err := receipt.VerifyToolArchive(context.Background(), archive)
	if err != nil {
		return err
	}
	expected := &pnpmSupplyMetadata{Identity: receipt.Identity(), Bundle: u.Bundle, ArchiveSize: int64(len(archive)), ToolSHA256: tool.SHA256(), ToolSize: tool.Size(), Libraries: tool.Libraries(), Invocation: receipt.Invocation(), Notices: append([]byte(nil), u.Notices...)}
	if !reflect.DeepEqual(m, metadata{Architecture: "amd64", PNPM: expected}) {
		return errors.New("frozen pnpm metadata differs from authenticated native supply")
	}
	return nil
}

// Notices are reviewed passive source data, not an archive-signature claim or
// execution authority. The frozen plan binds their complete bytes and source.
func validPNPMNotices(source string, body []byte) bool {
	prefix := "PNPM upstream source: https://github.com/pnpm/pnpm/tree/" + source + "\n\n"
	if len(body) <= len(prefix) || len(body) > 64<<10 || !utf8.Valid(body) || !strings.HasPrefix(string(body), prefix) {
		return false
	}
	for _, value := range string(body) {
		if unicode.IsControl(value) && value != '\n' && value != '\t' {
			return false
		}
	}
	return true
}
