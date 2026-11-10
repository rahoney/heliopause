package main

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
)

func validateSemgrepNative(reader io.ReaderAt) error {
	format, err := elf.NewFile(reader)
	if err != nil {
		return errors.New("semgrep source output is not a valid ELF")
	}
	defer format.Close()
	stacks := 0
	for _, program := range format.Progs {
		if program.Type == elf.PT_INTERP || program.Type == elf.PT_DYNAMIC || program.Type == elf.PT_GNU_STACK && program.Flags != elf.PF_R|elf.PF_W {
			return errors.New("semgrep source output has an unsupported dynamic/executable-stack profile")
		}
		if program.Type == elf.PT_GNU_STACK {
			stacks++
		}
	}
	if format.Class != elf.ELFCLASS64 || format.Machine != elf.EM_X86_64 || stacks != 1 {
		return errors.New("semgrep source native architecture or stack profile differs")
	}
	return nil
}

// Read the complete passive package without extracting or executing any member.
// The same fixed manifest contract is used by the producer and clean builder.
func readSemgrepSupplyArchive(root *os.Root, pin semgrepMaterializationOwner, notices bool, documents map[string][]byte) (semgrepSourcePackageInput, error) {
	var selected semgrepSourcePackageInput
	asset, name := pin.Sources, "SOURCE-MANIFEST.json"
	if notices {
		asset, name = pin.Notices, "NOTICE-MANIFEST.json"
	}
	member := semgrepSourcePackageFile{asset.Path, asset.SHA256, asset.Size, 0444}
	if err := copySemgrepSourceMember(root, io.Discard, member); err != nil {
		return selected, err
	}
	file, err := root.Open(asset.Path)
	if err != nil {
		return selected, err
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() != asset.Size {
		return selected, errors.New("semgrep supplied archive identity differs")
	}
	checksum := sha256.New()
	reader := io.TeeReader(file, checksum)
	archive := tar.NewReader(reader)
	header, err := archive.Next()
	if err != nil || header.Name != name || header.Typeflag != tar.TypeReg || header.Size <= 0 || header.Size > 2<<20 || header.Mode != 0444 || header.Uid != 0 || header.Gid != 0 || header.Format != tar.FormatUSTAR {
		return selected, errors.New("semgrep package manifest header differs")
	}
	body, err := io.ReadAll(archive)
	if err != nil {
		return selected, err
	}
	selected, size, err := parseSemgrepPackageManifest(body, notices)
	if err != nil {
		return selected, err
	}
	if size != asset.Size || selected.SourceCommit != pin.SourceCommit || selected.SourceDateEpoch != pin.SourceDateEpoch || header.ModTime.Unix() != pin.SourceDateEpoch || header.ModTime.Nanosecond() != 0 || !notices && hash(body) != pin.SourceManifestSHA256 {
		return selected, errors.New("semgrep package manifest differs from its selected owner")
	}
	required := map[string]semgrepAsset{}
	if !notices {
		required["patches/quarantine.patch"] = pin.Patch
		required["build/rebuild-worker.sh"] = pin.Worker
		required["rules/heliopause.json"] = pin.Rules
	}
	origins, sdkPrepare := notices, notices
	for _, row := range selected.Files {
		header, err = archive.Next()
		if err != nil || header.Name != row.Path || header.Typeflag != tar.TypeReg || header.Size != row.Size || header.Mode != row.Mode || header.Uid != 0 || header.Gid != 0 || header.ModTime.Unix() != pin.SourceDateEpoch || header.ModTime.Nanosecond() != 0 || header.Format != tar.FormatUSTAR {
			return selected, errors.New("semgrep package member header differs")
		}
		digest := sha256.New()
		var captured bytes.Buffer
		var output io.Writer = digest
		_, recipe := required[row.Path]
		capture := !notices && (row.Path == "SOURCE-ORIGINS.json" || row.Path == "build/sdk-prepare.sh" || documents != nil && recipe)
		if capture {
			if row.Size <= 0 || row.Size > 2<<20 {
				return selected, errors.New("semgrep build provenance exceeds bound")
			}
			output = io.MultiWriter(digest, &captured)
		}
		count, err := io.CopyN(output, archive, row.Size)
		if err != nil || count != row.Size || hex.EncodeToString(digest.Sum(nil)) != row.SHA256 {
			return selected, errors.New("semgrep package member differs from manifest")
		}
		if expected, ok := required[row.Path]; ok {
			if row.SHA256 != expected.SHA256 || row.Size != expected.Size {
				return selected, errors.New("semgrep recipe/rules differ from selected owner")
			}
			if documents != nil {
				documents[expected.Path] = bytes.Clone(captured.Bytes())
			}
			delete(required, row.Path)
		}
		if capture && row.Path == "build/sdk-prepare.sh" {
			if !bytes.Equal(captured.Bytes(), []byte(semgrepSDKPrepare)) {
				return selected, errors.New("semgrep SDK recipe differs from canonical builder")
			}
			sdkPrepare = true
		}
		if capture && row.Path == "SOURCE-ORIGINS.json" {
			if _, err := jsonSpans(captured.Bytes()); err != nil {
				return selected, err
			}
			var provenance struct {
				Schema string `json:"schema"`
				Source struct {
					Version   string `json:"version"`
					Tag       string `json:"tag"`
					TagObject string `json:"tag_object"`
					Commit    string `json:"commit"`
				} `json:"source"`
			}
			if json.Unmarshal(captured.Bytes(), &provenance) != nil || provenance.Schema != "helox.semgrep-source-origins/v1" || provenance.Source.Version != pin.Version || provenance.Source.Tag != "v"+pin.Version || provenance.Source.TagObject != pin.SourceTagObject || provenance.Source.Commit != pin.SourceCommit {
				return selected, errors.New("semgrep source origin differs from selected owner")
			}
			origins = true
		}
	}
	if _, err = archive.Next(); err != io.EOF {
		return selected, errors.New("semgrep package has extra or malformed members")
	}
	var tail [1]byte
	count, err := reader.Read(tail[:])
	after, statErr := root.Lstat(asset.Path)
	if count != 0 || err != io.EOF || statErr != nil || !os.SameFile(before, after) || after.Size() != before.Size() || after.Mode() != before.Mode() || !after.ModTime().Equal(before.ModTime()) || hex.EncodeToString(checksum.Sum(nil)) != asset.SHA256 {
		return selected, errors.New("semgrep supplied archive changed or has trailing data")
	}
	if !origins || !sdkPrepare || len(required) != 0 {
		return selected, errors.New("semgrep source omits canonical build inputs")
	}
	if !notices {
		projected := selected.Files[:0:0]
		for _, row := range selected.Files {
			if strings.HasPrefix(row.Path, "notices/") {
				projected = append(projected, row)
			}
		}
		selected.Files = projected
	}
	return selected, nil
}
