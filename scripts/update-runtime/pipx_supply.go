package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type pipxFrozenDocument struct {
	Path string `json:"path"`
	Body []byte `json:"body"`
}
type pipxSupplyMetadata struct {
	Owner     pipxMaterializationOwner `json:"owner"`
	Documents []pipxFrozenDocument     `json:"documents"`
	Notices   []byte                   `json:"notices"`
}

func freezePipxSupply(version, input, state string) (metadata, error) {
	supply, err := readPipxClosure(input)
	if err != nil {
		return metadata{}, err
	}
	body, err := supply.closure.ZIP(supply.inputs)
	if err != nil {
		return metadata{}, err
	}
	notices, err := pipxNotices(supply.closure, body)
	if err != nil {
		return metadata{}, err
	}
	owner := pipxMaterializationOwner{version, "amd64", "python_image", supply.files, pipxAssetIdentity{"pipx-runtime.zip", hash(body), int64(len(body))}, pipxAssetIdentity{"tools/pipx/NOTICE", hash(notices), int64(len(notices))}}
	if err := bindPipxOwner(owner, supply, body, notices); err != nil {
		return metadata{}, err
	}
	directory := filepath.Join(state, "pipx-supply")
	if err := os.Mkdir(directory, 0700); err != nil {
		return metadata{}, err
	}
	documents := []pipxFrozenDocument{}
	dirs := map[string]bool{}
	for _, fact := range supply.files {
		data, err := readFrozenSupplyFile(input, fact.Path, 1<<20)
		if err != nil || hash(data) != fact.SHA256 {
			return metadata{}, errors.New("pipx input changed while freezing")
		}
		project, _, _ := strings.Cut(fact.Path, "/")
		parent := filepath.Join(directory, project)
		if !dirs[parent] {
			if err := os.Mkdir(parent, 0700); err != nil {
				return metadata{}, err
			}
			dirs[parent] = true
		}
		f, err := os.OpenFile(filepath.Join(directory, fact.Path), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return metadata{}, err
		}
		n, writeErr := f.Write(data)
		syncErr := f.Sync()
		closeErr := f.Close()
		if n != len(data) || writeErr != nil || syncErr != nil || closeErr != nil {
			return metadata{}, errors.Join(errors.New("pipx frozen write incomplete"), writeErr, syncErr, closeErr)
		}
		if !strings.HasSuffix(fact.Path, ".whl") {
			documents = append(documents, pipxFrozenDocument{fact.Path, data})
		}
	}
	dirs[directory] = true
	dirs[state] = true
	for name := range dirs {
		f, err := os.Open(name)
		if err != nil {
			return metadata{}, err
		}
		if err := errors.Join(f.Sync(), f.Close()); err != nil {
			return metadata{}, err
		}
	}
	m := metadata{Architecture: "amd64", Pipx: &pipxSupplyMetadata{owner, documents, notices}}
	if err := verifyFrozenPipxSupply(version, m, state); err != nil {
		return metadata{}, err
	}
	return m, nil
}

func verifyFrozenPipxSupply(version string, m metadata, state string) error {
	u := m.Pipx
	if u == nil || u.Owner.Version != version || !validPipxOwner(u.Owner) || len(u.Documents) != 20 {
		return errors.New("pipx frozen owner is missing or invalid")
	}
	directory := filepath.Join(state, "pipx-supply")
	canonical, err := filepath.EvalSymlinks(directory)
	if err != nil || canonical != directory {
		return errors.New("pipx frozen directory is not canonical")
	}
	dirs := map[string]bool{directory: true}
	for _, fact := range u.Owner.Inputs {
		project, _, _ := strings.Cut(fact.Path, "/")
		dirs[filepath.Join(directory, project)] = true
		info, err := os.Lstat(filepath.Join(directory, fact.Path))
		if err != nil || !frozenSupplyOwned(info, false) {
			return errors.New("pipx frozen file ownership/mode differs")
		}
	}
	for name := range dirs {
		info, err := os.Lstat(name)
		if err != nil || !frozenSupplyOwned(info, true) {
			return errors.New("pipx frozen directory ownership/mode differs")
		}
	}
	supply, err := readPipxClosure(directory)
	if err != nil {
		return err
	}
	body, err := supply.closure.ZIP(supply.inputs)
	if err != nil {
		return err
	}
	notices, err := pipxNotices(supply.closure, body)
	if err != nil {
		return err
	}
	if err := bindPipxOwner(u.Owner, supply, body, notices); err != nil {
		return err
	}
	documents := []pipxFrozenDocument{}
	for _, fact := range supply.files {
		if strings.HasSuffix(fact.Path, ".whl") {
			continue
		}
		data, err := readFrozenSupplyFile(directory, fact.Path, 1<<20)
		if err != nil || hash(data) != fact.SHA256 {
			return errors.New("pipx frozen metadata changed")
		}
		documents = append(documents, pipxFrozenDocument{fact.Path, data})
	}
	expected := metadata{Architecture: "amd64", Pipx: &pipxSupplyMetadata{u.Owner, documents, notices}}
	if !reflect.DeepEqual(m, expected) {
		return errors.New("pipx frozen plan metadata differs from supply")
	}
	return nil
}
