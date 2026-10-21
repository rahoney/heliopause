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
	"syscall"

	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

// The plan carries small authenticated metadata. Native layers remain bounded
// external-state files and are authenticated again on every render/apply.
type uvSupplyMetadata struct {
	Identity   releaseprovenance.UVImageIdentity `json:"identity"`
	Bundle     []byte                            `json:"bundle"`
	Index      []byte                            `json:"index"`
	Manifest   []byte                            `json:"manifest"`
	Config     []byte                            `json:"config"`
	ToolSHA256 string                            `json:"tool_sha256"`
	ToolSize   int64                             `json:"tool_size"`
}

func freezeUVSupply(version, proof, layers, state string) (metadata, error) {
	if !filepath.IsAbs(proof) || !filepath.IsAbs(layers) {
		return metadata{}, errors.New("changed UV requires absolute --uv-proof and --uv-layers")
	}
	u := &uvSupplyMetadata{}
	body, err := readUVSupplyFile(proof, "identity.json", 4096)
	if err != nil {
		return metadata{}, err
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(&u.Identity) != nil || d.Decode(new(any)) != io.EOF || u.Identity.Version != version {
		return metadata{}, errors.New("UV requested version and proof identity differ")
	}
	for _, file := range []struct {
		name   string
		limit  int64
		target *[]byte
	}{
		{"bundle.json", 1 << 20, &u.Bundle}, {"index.json", 1 << 20, &u.Index}, {"manifest.json", 64 << 10, &u.Manifest}, {"config.json", 64 << 10, &u.Config},
	} {
		*file.target, err = readUVSupplyFile(proof, file.name, file.limit)
		if err != nil {
			return metadata{}, err
		}
	}
	receipt, err := releaseprovenance.VerifyUVImage(context.Background(), u.Bundle, u.Index, u.Manifest, u.Config, u.Identity)
	if err != nil {
		return metadata{}, err
	}
	inputs, bodies, err := readUVSupplyLayers(layers, receipt)
	if err != nil {
		return metadata{}, err
	}
	tool, err := receipt.VerifyToolLayers(context.Background(), u.Config, inputs)
	if err != nil {
		return metadata{}, err
	}
	u.ToolSHA256, u.ToolSize = tool.SHA256(), tool.Size()
	destination := filepath.Join(state, "uv-supply")
	if err := os.Mkdir(destination, 0700); err != nil {
		return metadata{}, err
	}
	// A failed copy deliberately remains visible in its owned state. No prior
	// state files are overwritten and no user input directory is cleaned up.
	for i, layer := range receipt.Layers() {
		f, err := os.OpenFile(filepath.Join(destination, layer.SHA256+".tar.gz"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return metadata{}, err
		}
		_, writeErr := f.Write(bodies[i])
		syncErr := f.Sync()
		closeErr := f.Close()
		if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
			return metadata{}, err
		}
	}
	for _, path := range []string{destination, state} {
		dir, err := os.Open(path)
		if err != nil {
			return metadata{}, err
		}
		if err := errors.Join(dir.Sync(), dir.Close()); err != nil {
			return metadata{}, err
		}
	}
	m := metadata{Image: "ghcr.io/astral-sh/uv:" + version + "@sha256:" + u.Identity.ManifestSHA256, Architecture: "amd64", UV: u}
	if err := verifyFrozenUVSupply(version, m, state); err != nil {
		return metadata{}, err
	}
	return m, nil
}

func readUVSupplyLayers(directory string, receipt releaseprovenance.UVImageReceipt) ([]releaseprovenance.UVToolLayer, [][]byte, error) {
	var inputs []releaseprovenance.UVToolLayer
	var bodies [][]byte
	for _, layer := range receipt.Layers() {
		body, err := readUVSupplyFile(directory, layer.SHA256+".tar.gz", layer.Size)
		if err != nil {
			return nil, nil, err
		}
		inputs = append(inputs, releaseprovenance.UVToolLayer{Reader: bytes.NewReader(body), Size: int64(len(body))})
		bodies = append(bodies, body)
	}
	return inputs, bodies, nil
}

func verifyFrozenUVSupply(version string, m metadata, state string) error {
	u := m.UV
	if u == nil || u.Identity.Version != version {
		return errors.New("frozen UV proof/version missing")
	}
	receipt, err := releaseprovenance.VerifyUVImage(context.Background(), u.Bundle, u.Index, u.Manifest, u.Config, u.Identity)
	if err != nil {
		return err
	}
	directory := filepath.Join(state, "uv-supply")
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil || resolved != directory {
		return errors.New("frozen UV layer directory changed or is a symlink")
	}
	info, err := os.Lstat(directory)
	if err != nil || !uvStateOwned(info, true) {
		return errors.New("frozen UV state directory ownership or mode changed")
	}
	for _, layer := range receipt.Layers() {
		info, err := os.Lstat(filepath.Join(directory, layer.SHA256+".tar.gz"))
		if err != nil || !uvStateOwned(info, false) {
			return errors.New("frozen UV layer ownership or mode changed")
		}
	}
	inputs, _, err := readUVSupplyLayers(directory, receipt)
	if err != nil {
		return err
	}
	tool, err := receipt.VerifyToolLayers(context.Background(), u.Config, inputs)
	if err != nil {
		return err
	}
	expected := metadata{Image: "ghcr.io/astral-sh/uv:" + version + "@sha256:" + receipt.Identity().ManifestSHA256, Architecture: "amd64", UV: u}
	if tool.SHA256() != u.ToolSHA256 || tool.Size() != u.ToolSize || !reflect.DeepEqual(expected, m) {
		return errors.New("frozen UV metadata differs from authenticated complete supply")
	}
	return nil
}

func uvStateOwned(info os.FileInfo, directory bool) bool {
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != uint32(os.Geteuid()) {
		return false
	}
	if directory {
		return info.IsDir() && info.Mode().Perm() == 0700
	}
	return info.Mode().IsRegular() && info.Mode().Perm() == 0600
}
