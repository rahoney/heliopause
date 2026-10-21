package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

// verifyUVImage is an offline supply check. It does not adopt pins, acquire
// images, execute upstream tools or modify a running installation.
func verifyUVImage(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("update-runtime verify-uv-image", flag.ContinueOnError)
	flags.SetOutput(out)
	input := flags.String("input", "", "absolute frozen proof directory: identity.json, bundle.json, index.json, manifest.json, config.json")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !filepath.IsAbs(*input) {
		return errors.New("uv image proof requires an explicit absolute directory")
	}
	info, err := os.Lstat(*input)
	if err != nil || !info.IsDir() {
		return errors.New("uv image proof directory is unavailable or a symlink")
	}
	read := func(name string, limit int64) ([]byte, error) {
		location := filepath.Join(*input, name)
		before, err := os.Lstat(location)
		if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > limit {
			return nil, errors.New("uv image proof file is unavailable or exceeds bound")
		}
		file, err := os.Open(location)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		after, err := file.Stat()
		if err != nil || !os.SameFile(before, after) || after.Size() != before.Size() {
			return nil, errors.New("uv image proof file changed")
		}
		body, err := io.ReadAll(io.LimitReader(file, limit+1))
		if err != nil || int64(len(body)) != before.Size() {
			return nil, errors.New("uv image proof read failed or changed")
		}
		return body, nil
	}
	identityBody, err := read("identity.json", 4096)
	if err != nil {
		return err
	}
	var identity releaseprovenance.UVImageIdentity
	decoder := json.NewDecoder(bytes.NewReader(identityBody))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&identity) != nil || decoder.Decode(new(any)) != io.EOF {
		return errors.New("uv image identity is invalid")
	}
	var documents [][]byte
	for _, item := range []struct {
		name  string
		limit int64
	}{{"bundle.json", 1 << 20}, {"index.json", 1 << 20}, {"manifest.json", 64 << 10}, {"config.json", 64 << 10}} {
		body, err := read(item.name, item.limit)
		if err != nil {
			return err
		}
		documents = append(documents, body)
	}
	receipt, err := releaseprovenance.VerifyUVImage(context.Background(), documents[0], documents[1], documents[2], documents[3], identity)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		Schema   string                            `json:"schema"`
		Identity releaseprovenance.UVImageIdentity `json:"identity"`
		Files    []struct {
			Role   string `json:"role"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
		Invocation string                           `json:"invocation"`
		Layers     []releaseprovenance.UVImageLayer `json:"layers"`
	}{"helox.uv-image-supply/v1", receipt.Identity(), []struct {
		Role   string `json:"role"`
		SHA256 string `json:"sha256"`
	}{{"bundle", receipt.BundleSHA256()}, {"config", receipt.ConfigSHA256()}}, receipt.Invocation(), receipt.Layers()})
}
