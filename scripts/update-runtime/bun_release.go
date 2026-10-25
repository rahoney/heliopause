package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

type bunSupplyDiscovery struct {
	Version              string `json:"version"`
	SourceCommitObserved string `json:"source_commit_observed"`
	ArchiveSHA256        string `json:"archive_sha256"`
	ArchiveSize          int64  `json:"archive_size"`
}

// This offline command verifies vendor checksum/archive bytes only. Version
// and source commit remain discovery metadata until separate native/build proof.
func verifyBunRelease(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime verify-bun-release", flag.ContinueOnError)
	flags.SetOutput(out)
	input := flags.String("input", "", "absolute frozen directory: identity.json, SHASUMS256.txt.asc, bun-linux-x64.zip")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*input) {
		return errors.New("bun supply verification requires an explicit absolute directory")
	}
	body, err := readFrozenSupplyFile(*input, "identity.json", 4096)
	if err != nil {
		return err
	}
	if _, err := jsonSpans(body); err != nil {
		return err
	}
	var discovery bunSupplyDiscovery
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&discovery) != nil || decoder.Decode(new(any)) != io.EOF {
		return errors.New("bun discovery JSON is invalid")
	}
	validVersion := len(discovery.Version) <= 64 && versionPattern.MatchString(discovery.Version)
	for _, part := range strings.Split(discovery.Version, ".") {
		if len(part) > 1 && part[0] == '0' {
			validVersion = false
		}
	}
	sourceCommit, decodeErr := hex.DecodeString(discovery.SourceCommitObserved)
	if decodeErr != nil || len(sourceCommit) != 20 || hex.EncodeToString(sourceCommit) != discovery.SourceCommitObserved || !validVersion || discovery.ArchiveSize <= 0 || discovery.ArchiveSize > releaseprovenance.BunArchiveLimit {
		return errors.New("bun frozen discovery metadata is invalid")
	}
	signed, err := readFrozenSupplyFile(*input, "SHASUMS256.txt.asc", 64<<10)
	if err != nil {
		return err
	}
	receipt, err := releaseprovenance.VerifyBunChecksums(context.Background(), signed, discovery.ArchiveSHA256, time.Now())
	if err != nil {
		return err
	}
	archive, err := readFrozenSupplyFile(*input, releaseprovenance.BunArchiveFilename, releaseprovenance.BunArchiveLimit)
	if err != nil {
		return err
	}
	if int64(len(archive)) != discovery.ArchiveSize {
		return errors.New("bun archive size differs from frozen discovery")
	}
	tool, _, err := receipt.ReadToolArchive(context.Background(), archive)
	if err != nil {
		return err
	}
	type nativeRecord struct {
		ArchiveSHA256 string   `json:"archive_sha256"`
		SHA256        string   `json:"sha256"`
		Size          int64    `json:"size"`
		Libraries     []string `json:"libraries"`
	}
	return json.NewEncoder(out).Encode(struct {
		Tool                          nativeRecord       `json:"tool"`
		Schema                        string             `json:"schema"`
		Discovery                     bunSupplyDiscovery `json:"discovery"`
		Publisher                     string             `json:"publisher_fingerprint"`
		Signer                        string             `json:"signing_fingerprint"`
		Document                      string             `json:"signed_document_sha256"`
		Checksums                     string             `json:"authenticated_checksums_sha256"`
		SignedAt                      time.Time          `json:"signed_at"`
		PublisherChecksumVerified     bool               `json:"publisher_checksum_verified"`
		ArchiveBytesVerified          bool               `json:"archive_bytes_verified"`
		NativeVersionVerified         bool               `json:"native_version_verified"`
		SourceBuildProvenanceVerified bool               `json:"source_build_provenance_verified"`
	}{nativeRecord{tool.ArchiveSHA256(), tool.SHA256(), tool.Size(), tool.Libraries()}, "helox.bun-release-supply/v2", discovery, releaseprovenance.BunPublisherFingerprint, receipt.SigningFingerprint(), receipt.SignedDocumentSHA256(), receipt.ChecksumsSHA256(), receipt.SignatureTime(), true, true, false, false})
}
