package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
)

type semgrepSupplyMetadata struct {
	Owner     semgrepMaterializationOwner `json:"owner"`
	Documents map[string][]byte           `json:"documents"`
}

func freezeSemgrepSupply(version, input, state string) (metadata, error) {
	owner, err := readSemgrepFrozenOwner(input)
	if err != nil || owner.Version != version {
		return metadata{}, errors.Join(errors.New("semgrep frozen owner differs from requested version"), err)
	}
	root, err := os.OpenRoot(input)
	if err != nil {
		return metadata{}, err
	}
	defer root.Close()
	directory := filepath.Join(state, "semgrep-input")
	if err := os.Mkdir(directory, 0700); err != nil {
		return metadata{}, err
	}
	for _, asset := range []semgrepAsset{owner.Tool, owner.Sources, owner.Notices} {
		if err := publishNativeAssetWith(filepath.Join(directory, asset.Path), asset.Size, 0444, func(output io.Writer) error {
			return copySemgrepSourceMember(root, output, semgrepSourcePackageFile{asset.Path, asset.SHA256, asset.Size, 0444})
		}); err != nil {
			return metadata{}, err
		}
	}
	canonicalOwner, err := json.Marshal(struct {
		Schema int                         `json:"schema_version"`
		Owner  semgrepMaterializationOwner `json:"semgrep_tool"`
	}{6, owner})
	if err != nil {
		return metadata{}, err
	}
	if err := publishNativeAsset(filepath.Join(directory, "owner.json"), canonicalOwner, 0444); err != nil {
		return metadata{}, err
	}
	m := metadata{Architecture: "amd64", Semgrep: &semgrepSupplyMetadata{Owner: owner}}
	documents, err := verifySemgrepFrozenFiles(version, owner, directory)
	if err != nil {
		return metadata{}, err
	}
	m.Semgrep.Documents = documents
	return m, nil
}

func verifyFrozenSemgrepSupply(version string, m metadata, state string) error {
	if m.Semgrep == nil || m.Architecture != "amd64" {
		return errors.New("semgrep frozen metadata is missing")
	}
	documents, err := verifySemgrepFrozenFiles(version, m.Semgrep.Owner, filepath.Join(state, "semgrep-input"))
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(documents, m.Semgrep.Documents) {
		return errors.New("semgrep frozen recipe/rules metadata changed")
	}
	return nil
}

func verifySemgrepFrozenFiles(version string, owner semgrepMaterializationOwner, directory string) (map[string][]byte, error) {
	if owner.Version != version {
		return nil, errors.New("semgrep frozen version differs")
	}
	body, err := readFrozenSupplyFile(directory, "owner.json", 128<<10)
	if err != nil {
		return nil, err
	}
	frozen, err := parseSemgrepMaterializationOwner(body)
	if err != nil || frozen != owner {
		return nil, errors.Join(errors.New("semgrep frozen owner changed"), err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	if err := copySemgrepSourceMember(root, io.Discard, semgrepSourcePackageFile{owner.Tool.Path, owner.Tool.SHA256, owner.Tool.Size, 0444}); err != nil {
		return nil, err
	}
	native, err := root.Open(owner.Tool.Path)
	if err != nil {
		return nil, err
	}
	err = validateSemgrepNative(native)
	if closeErr := native.Close(); err != nil || closeErr != nil {
		return nil, errors.Join(err, closeErr)
	}
	documents := map[string][]byte{}
	source, err := readSemgrepSupplyArchive(root, owner, false, documents)
	if err != nil {
		return nil, err
	}
	notices, err := readSemgrepSupplyArchive(root, owner, true, nil)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(source.Files, notices.Files) || len(documents) != 3 {
		return nil, errors.New("semgrep frozen source/notice projection differs")
	}
	return documents, nil
}

func readSemgrepFrozenOwner(input string) (semgrepMaterializationOwner, error) {
	var owner semgrepMaterializationOwner
	if !filepath.IsAbs(input) || filepath.Clean(input) != input {
		return owner, errors.New("semgrep requires explicit absolute frozen supply")
	}
	canonical, err := filepath.EvalSymlinks(input)
	if err != nil || canonical != input {
		return owner, errors.New("semgrep frozen supply is unavailable or a symlink")
	}
	body, err := readFrozenSupplyFile(input, "owner.json", 128<<10)
	if err != nil {
		return owner, err
	}
	return parseSemgrepMaterializationOwner(body)
}
