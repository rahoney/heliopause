package main

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type semgrepSourcePackageInput struct {
	Schema          string                     `json:"schema"`
	SourceCommit    string                     `json:"source_commit"`
	SourceDateEpoch int64                      `json:"source_date_epoch"`
	Files           []semgrepSourcePackageFile `json:"files"`
}

type semgrepSourcePackageFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Mode   int64  `json:"mode"`
}

// This packages curated source inputs, not a build attestation or runtime
// admission. The selected runtime owner must independently bind the result.
func packageSemgrepSource(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime package-semgrep-source", flag.ContinueOnError)
	flags.SetOutput(out)
	input := flags.String("input", "", "absolute frozen source input directory containing source-inputs.json")
	output := flags.String("output", "", "absolute new external semgrep-SOURCES.tar")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*input) || !filepath.IsAbs(*output) || filepath.Clean(*input) != *input || filepath.Clean(*output) != *output || filepath.Base(*output) != "semgrep-SOURCES.tar" {
		return errors.New("semgrep source package requires explicit canonical input and new external output paths")
	}
	canonical, err := filepath.EvalSymlinks(*input)
	if err != nil || canonical != *input {
		return errors.New("semgrep source input must be canonical and available")
	}
	if relative, err := filepath.Rel(canonical, *output); err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("semgrep source output must be outside the input directory")
	}
	body, err := readFrozenSupplyFile(canonical, "source-inputs.json", 2<<20)
	if err != nil {
		return err
	}
	if _, err := jsonSpans(body); err != nil {
		return err
	}
	var selected semgrepSourcePackageInput
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&selected) != nil || decoder.Decode(new(any)) != io.EOF {
		return errors.New("semgrep source manifest is malformed or has unknown fields")
	}
	commit, err := hex.DecodeString(selected.SourceCommit)
	if err != nil || len(commit) != 20 || strings.ToLower(selected.SourceCommit) != selected.SourceCommit || selected.Schema != "helox.semgrep-source-inputs/v1" || selected.SourceDateEpoch < 0 || selected.SourceDateEpoch >= 1<<33 || len(selected.Files) == 0 || len(selected.Files) > 4096 {
		return errors.New("semgrep source manifest identity or bounds differ")
	}
	// USTAR emits exactly one header and 512-byte padding per ordinary member.
	// Requiring this format also rejects names needing additional PAX records.
	size := int64(1024 + 512 + (len(body)+511)/512*512)
	previous := ""
	for _, file := range selected.Files {
		if file.Path <= previous || file.Path == "SOURCE-MANIFEST.json" || file.Path == "." || file.Path == ".." || path.IsAbs(file.Path) || path.Clean(file.Path) != file.Path || strings.HasPrefix(file.Path, "../") || strings.ContainsAny(file.Path, "\\:") || len(file.Path) > 240 || !digestPattern.MatchString(file.SHA256) || file.Size < 0 || file.Size > 256<<20 || file.Mode != 0444 && file.Mode != 0555 {
			return errors.New("semgrep source member is unsafe, duplicated or outside bounds")
		}
		for _, character := range file.Path {
			if character < 33 || character > 126 {
				return errors.New("semgrep source member path has unsupported characters")
			}
		}
		previous = file.Path
		size += 512 + (file.Size+511)/512*512
		if size > 512<<20 {
			return errors.New("semgrep source package exceeds its supply bound")
		}
	}
	root, err := os.OpenRoot(canonical)
	if err != nil {
		return err
	}
	defer root.Close()
	digest := sha256.New()
	if err := publishNativeAssetWith(*output, size, 0444, func(output io.Writer) error {
		archive := tar.NewWriter(io.MultiWriter(output, digest))
		stamp := time.Unix(selected.SourceDateEpoch, 0).UTC()
		if err := archive.WriteHeader(&tar.Header{Name: "SOURCE-MANIFEST.json", Mode: 0444, Size: int64(len(body)), ModTime: stamp, Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}); err != nil {
			return err
		}
		if _, err := archive.Write(body); err != nil {
			return err
		}
		for _, member := range selected.Files {
			if err := writeSemgrepSourceMember(root, archive, member, stamp); err != nil {
				return err
			}
		}
		return archive.Close()
	}); err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Schema           string `json:"schema"`
		SourceCommit     string `json:"source_commit"`
		ManifestSHA256   string `json:"manifest_sha256"`
		SHA256           string `json:"sha256"`
		Size             int64  `json:"size"`
		Files            int    `json:"files"`
		RuntimeQualified bool   `json:"runtime_qualified"`
	}{"helox.semgrep-source-package/v1", selected.SourceCommit, hash(body), hex.EncodeToString(digest.Sum(nil)), size, len(selected.Files), false})
}

func writeSemgrepSourceMember(root *os.Root, archive *tar.Writer, member semgrepSourcePackageFile, stamp time.Time) error {
	before, err := root.Lstat(member.Path)
	if err != nil || !before.Mode().IsRegular() || before.Size() != member.Size {
		return errors.New("semgrep source member is absent, a link or has changed size")
	}
	file, err := root.Open(member.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	held, err := file.Stat()
	if err != nil || !os.SameFile(before, held) || held.Size() != member.Size {
		return errors.New("semgrep source member changed before opening")
	}
	if err := archive.WriteHeader(&tar.Header{Name: member.Path, Mode: member.Mode, Size: member.Size, ModTime: stamp, Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}); err != nil {
		return err
	}
	digest := sha256.New()
	count, readErr := io.CopyN(io.MultiWriter(archive, digest), file, member.Size)
	var tail [1]byte
	extra, endErr := file.Read(tail[:])
	after, statErr := root.Lstat(member.Path)
	if readErr != nil || count != member.Size || extra != 0 || endErr != io.EOF || statErr != nil || !os.SameFile(before, after) || after.Size() != before.Size() || after.Mode() != before.Mode() || !after.ModTime().Equal(before.ModTime()) || hex.EncodeToString(digest.Sum(nil)) != member.SHA256 {
		return errors.New("semgrep source member bytes or identity changed")
	}
	return nil
}
