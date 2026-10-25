package main

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

// Discovery is observed metadata. Signed checksums authenticate archive bytes,
// and never turn this version/source into source-build provenance.
type bunSupplyMetadata struct {
	Discovery  bunSupplyDiscovery `json:"discovery"`
	Signed     []byte             `json:"signed_checksums"`
	ToolSHA256 string             `json:"tool_sha256"`
	ToolSize   int64              `json:"tool_size"`
	Libraries  []string           `json:"libraries"`
	Notices    []byte             `json:"notices"`
}

func freezeBunSupply(version, input, state string) (metadata, error) {
	if !filepath.IsAbs(input) {
		return metadata{}, errors.New("bun supply requires an absolute frozen proof directory")
	}
	supply, err := readBunSupply(input, time.Now())
	if err != nil {
		return metadata{}, err
	}
	notices, err := readFrozenSupplyFile(input, "NOTICE", 64<<10)
	if err != nil || !validBunNotices(supply.discovery.SourceCommitObserved, notices) {
		return metadata{}, errors.New("bun plan requires explicit source-bound passive notices")
	}
	if supply.discovery.Version != version {
		return metadata{}, errors.New("bun selected version and observed supply version differ")
	}
	directory := filepath.Join(state, "bun-supply")
	if err := os.Mkdir(directory, 0700); err != nil {
		return metadata{}, err
	}
	file, err := os.OpenFile(filepath.Join(directory, supply.discovery.ArchiveSHA256+".zip"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return metadata{}, err
	}
	count, writeErr := file.Write(supply.archive)
	syncErr, closeErr := file.Sync(), file.Close()
	if count != len(supply.archive) || writeErr != nil || syncErr != nil || closeErr != nil {
		return metadata{}, errors.Join(errors.New("bun frozen archive write is incomplete"), writeErr, syncErr, closeErr)
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
	m := metadata{Architecture: "amd64", Bun: &bunSupplyMetadata{supply.discovery, supply.signed, supply.tool.SHA256(), supply.tool.Size(), supply.tool.Libraries(), notices}}
	if err := verifyFrozenBunSupply(version, m, state); err != nil {
		return metadata{}, err
	}
	return m, nil
}

func verifyFrozenBunSupply(version string, m metadata, state string) error {
	u := m.Bun
	if u == nil || !versionPattern.MatchString(version) || u.Discovery.Version != version || !validBunNotices(u.Discovery.SourceCommitObserved, u.Notices) || u.Discovery.ArchiveSize <= 0 || u.Discovery.ArchiveSize > releaseprovenance.BunArchiveLimit {
		return errors.New("frozen bun supply/version is missing or excessive")
	}
	receipt, err := releaseprovenance.VerifyBunChecksums(context.Background(), u.Signed, u.Discovery.ArchiveSHA256, time.Now())
	if err != nil {
		return err
	}
	directory := filepath.Join(state, "bun-supply")
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil || resolved != directory {
		return errors.New("frozen bun directory changed or is a symlink")
	}
	info, err := os.Lstat(directory)
	if err != nil || !frozenSupplyOwned(info, true) {
		return errors.New("frozen bun directory ownership or mode changed")
	}
	name := u.Discovery.ArchiveSHA256 + ".zip"
	info, err = os.Lstat(filepath.Join(directory, name))
	if err != nil || !frozenSupplyOwned(info, false) {
		return errors.New("frozen bun archive ownership or mode changed")
	}
	archive, err := readFrozenSupplyFile(directory, name, u.Discovery.ArchiveSize)
	if err != nil || int64(len(archive)) != u.Discovery.ArchiveSize {
		return errors.New("frozen bun archive is unavailable or has changed size")
	}
	tool, _, err := receipt.ReadToolArchive(context.Background(), archive)
	if err != nil {
		return err
	}
	expected := &bunSupplyMetadata{u.Discovery, u.Signed, tool.SHA256(), tool.Size(), tool.Libraries(), append([]byte(nil), u.Notices...)}
	if !reflect.DeepEqual(m, metadata{Architecture: "amd64", Bun: expected}) {
		return errors.New("frozen bun metadata differs from authenticated native data")
	}
	return nil
}

func validBunNotices(source string, body []byte) bool {
	commit, err := hex.DecodeString(source)
	prefix := "Bun upstream source: https://github.com/oven-sh/bun/tree/" + source + "\n\n"
	if err != nil || len(commit) != 20 || hex.EncodeToString(commit) != source || len(body) <= len(prefix) || len(body) > 64<<10 || !utf8.Valid(body) || !strings.HasPrefix(string(body), prefix) {
		return false
	}
	for _, value := range string(body) {
		if unicode.IsControl(value) && value != '\n' && value != '\t' {
			return false
		}
	}
	return true
}
