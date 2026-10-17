package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	manifestFileName = "helox-release-manifest.json"
	checksumFileName = "helox-release-checksums.txt"
	sbomFileName     = "helox-release-sbom.cdx.json"
	imageFileName    = "helox-runtime-images.json"
)

func manifestRecords(root, version, commit, workflow string) ([]releaseRecord, error) {
	info, err := os.Lstat(root)
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("bundle must be an absolute non-symlink directory")
	}
	if !releaseVersion.MatchString(version) || !commitID.MatchString(commit) || !workflowRunURL.MatchString(workflow) {
		return nil, errors.New("bundle verification requires exact version, source and workflow context")
	}
	manifestPath := filepath.Join(root, manifestFileName)
	metadata, err := os.Lstat(manifestPath)
	if err != nil || !metadata.Mode().IsRegular() || metadata.Size() > 1<<20 {
		return nil, errors.New("manifest is unavailable or outside metadata bounds")
	}
	body, err := readRegularFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var value releaseManifest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&value) != nil || decoder.Decode(&struct{}{}) != io.EOF || value.Schema != manifestSchema || value.Version != version || value.SourceCommit != commit || value.WorkflowRun != workflow {
		return nil, errors.New("manifest does not bind the requested version/source/run")
	}
	digest, decodeErr := hex.DecodeString(value.RuntimeLock.SHA256)
	if value.SBOM.Name != sbomFileName || value.RuntimeImages.Name != imageFileName || value.SBOM.Size <= 0 || value.RuntimeImages.Size <= 0 || decodeErr != nil || len(digest) != 32 || strings.ToLower(value.RuntimeLock.SHA256) != value.RuntimeLock.SHA256 || !commitID.MatchString(value.RuntimeLock.GVisorCommit) || value.RuntimeLock.GVisorRelease == "" {
		return nil, errors.New("manifest metadata binding is invalid")
	}
	specs := assetSpecs()
	seen := map[string]bool{}
	paths := make([]string, 0, len(value.Assets))
	for _, asset := range value.Assets {
		spec, ok := specs[asset.Name]
		if !ok || seen[asset.Name] || asset.Platform != spec.platform || asset.Role != spec.role {
			return nil, errors.New("manifest asset identity is invalid or duplicated")
		}
		seen[asset.Name] = true
		paths = append(paths, filepath.Join(root, asset.Name))
	}
	actual, _, err := collectAssets(paths)
	if err != nil {
		return nil, err
	}
	declared := append([]releaseAsset(nil), value.Assets...)
	sort.Slice(declared, func(i, j int) bool { return declared[i].Name < declared[j].Name })
	for i, asset := range actual {
		if asset != declared[i] {
			return nil, errors.New("manifest asset size/digest does not match bundle")
		}
	}
	result := make([]releaseRecord, 0, len(actual)+3)
	for _, asset := range actual {
		result = append(result, releaseRecord{Name: asset.Name, SHA256: asset.SHA256, Size: asset.Size})
	}
	for _, record := range []releaseRecord{value.SBOM, value.RuntimeImages} {
		actual, err := fileRecord(filepath.Join(root, record.Name))
		if err != nil || actual != record {
			return nil, errors.New("bundle metadata size/digest is invalid")
		}
		result = append(result, record)
	}
	self, err := fileRecord(manifestPath)
	if err != nil {
		return nil, err
	}
	result = append(result, self)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func checksumText(records []releaseRecord) []byte {
	var text strings.Builder
	for _, record := range records {
		fmt.Fprintf(&text, "%s  %s\n", record.SHA256, record.Name)
	}
	return []byte(text.String())
}

func writeChecksums(manifest, output, version, commit, workflow string) error {
	if filepath.Base(manifest) != manifestFileName || filepath.Base(output) != checksumFileName || filepath.Dir(manifest) != filepath.Dir(output) {
		return errors.New("checksum output must accompany the canonical manifest")
	}
	records, err := manifestRecords(filepath.Dir(manifest), version, commit, workflow)
	if err != nil {
		return err
	}
	return writeNewFile(output, checksumText(records))
}

func verifyBundle(root, version, commit, workflow string) ([]string, error) {
	records, err := manifestRecords(root, version, commit, workflow)
	if err != nil {
		return nil, err
	}
	checksums, err := readRegularFile(filepath.Join(root, checksumFileName))
	if err != nil || !bytes.Equal(checksums, checksumText(records)) {
		return nil, errors.New("checksum file does not describe exactly the manifest-bound bundle")
	}
	names := make([]string, 0, len(records)+1)
	allowed := map[string]bool{checksumFileName: true}
	for _, record := range records {
		names = append(names, record.Name)
		allowed[record.Name] = true
	}
	names = append(names, checksumFileName)
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != len(names) {
		return nil, errors.New("bundle file set is not exact")
	}
	for _, entry := range entries {
		if !allowed[entry.Name()] || !entry.Type().IsRegular() {
			return nil, errors.New("bundle contains an unexpected or non-regular entry")
		}
	}
	sort.Strings(names)
	return names, nil
}
