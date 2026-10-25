package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

type bunMaterializationOwner struct {
	Version              string `json:"bun_version"`
	Architecture         string `json:"architecture"`
	RuntimeImage         string `json:"runtime_image"`
	SourceCommitObserved string `json:"source_commit_observed"`
	Archive              struct {
		URL    string `json:"url"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
	} `json:"archive"`
	Publisher struct {
		Fingerprint     string `json:"fingerprint"`
		ChecksumsPath   string `json:"checksums_path"`
		ChecksumsSHA256 string `json:"checksums_sha256"`
	} `json:"publisher"`
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

func readBunMaterializationOwner(root string) (bunMaterializationOwner, error) {
	var pin bunMaterializationOwner
	body, err := readFrozenSupplyFile(root, runtimeFile, 64<<10)
	if err != nil {
		return pin, err
	}
	selected, err := selectedBunOwner(body)
	if err != nil {
		return pin, err
	}
	decoder := json.NewDecoder(bytes.NewReader(selected))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&pin) != nil || decoder.Decode(new(any)) != io.EOF {
		return bunMaterializationOwner{}, errors.New("bun selected runtime owner is invalid")
	}
	commit, err := hex.DecodeString(pin.SourceCommitObserved)
	if !validBunVersion(pin.Version) || pin.Architecture != "amd64" || pin.RuntimeImage != "node_image" || err != nil || len(commit) != 20 || hex.EncodeToString(commit) != pin.SourceCommitObserved || pin.Archive.URL != "https://github.com/oven-sh/bun/releases/download/bun-v"+pin.Version+"/bun-linux-x64.zip" || !digestPattern.MatchString(pin.Archive.SHA256) || pin.Archive.Size <= 0 || pin.Archive.Size > releaseprovenance.BunArchiveLimit || pin.Publisher.Fingerprint != releaseprovenance.BunPublisherFingerprint || pin.Publisher.ChecksumsPath != "tools/bun/SHASUMS256.txt.asc" || !digestPattern.MatchString(pin.Publisher.ChecksumsSHA256) || pin.Tool.Path != "bun" || !digestPattern.MatchString(pin.Tool.SHA256) || pin.Tool.Size <= 0 || pin.Tool.Size > 128<<20 || pin.Notices.Path != "tools/bun/NOTICE" || !digestPattern.MatchString(pin.Notices.SHA256) || pin.Notices.Size <= 0 || pin.Notices.Size > 64<<10 {
		return bunMaterializationOwner{}, errors.New("bun selected publisher/archive/native/notices owner is invalid")
	}
	return pin, nil
}

// selectedBunOwner returns the exact schema4 owner bytes after duplicate and
// canonical-field checks. Both version selection and materialization consume it.
func selectedBunOwner(body []byte) ([]byte, error) {
	fields, err := jsonSpans(body)
	if err != nil {
		return nil, err
	}
	expected := map[string]bool{}
	for _, field := range []string{"", "/bun_version", "/architecture", "/runtime_image", "/archive", "/archive/url", "/archive/sha256", "/archive/size", "/publisher", "/publisher/fingerprint", "/publisher/checksums_path", "/publisher/checksums_sha256", "/source_commit_observed", "/tool", "/tool/path", "/tool/sha256", "/tool/size", "/notices", "/notices/path", "/notices/sha256", "/notices/size"} {
		expected["/bun_tool"+field] = true
	}
	count := 0
	for path := range fields {
		if path == "/bun_tool" || strings.HasPrefix(path, "/bun_tool/") {
			if !expected[path] {
				return nil, errors.New("bun selected owner has an unknown or noncanonical field")
			}
			count++
		}
	}
	if count != len(expected) {
		return nil, errors.New("bun selected owner fields are missing")
	}
	schema, ok := fields["/schema_version"]
	var version int
	if !ok || json.Unmarshal(body[schema.start:schema.end], &version) != nil || version != 4 {
		return nil, errors.New("bun runtime owner schema is invalid")
	}
	owner := fields["/bun_tool"]
	return body[owner.start:owner.end], nil
}
