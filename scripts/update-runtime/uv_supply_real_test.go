//go:build realcorpus

package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestUVActualFrozenSupplyAndTamperRefusals(t *testing.T) {
	layers := os.Getenv("HELOX_UV_LAYER_ROOT")
	if !filepath.IsAbs(layers) {
		t.Fatal("HELOX_UV_LAYER_ROOT must declare complete official layers")
	}
	state, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := freezeUVSupply("0.13.4", uvProofDirectory(t), layers, state)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var frozen metadata
	if err := json.Unmarshal(encoded, &frozen); err != nil {
		t.Fatal(err)
	}
	if err := verifyFrozenUVSupply("0.13.4", frozen, state); err != nil {
		t.Fatal(err)
	}
	frozen.UV.ToolSHA256 = "wrong"
	if err := verifyFrozenUVSupply("0.13.4", frozen, state); err == nil {
		t.Fatal("substituted native identity accepted")
	}
	if err := verifyFrozenUVSupply("0.0.1", m, state); err == nil {
		t.Fatal("substituted version accepted")
	}
	files, err := os.ReadDir(filepath.Join(state, "uv-supply"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(state, "uv-supply", files[0].Name())
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte{0}, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := verifyFrozenUVSupply("0.13.4", m, state); err == nil {
		t.Fatal("mutated frozen layer accepted")
	}
	if _, err := freezeUVSupply("0.13.4", uvProofDirectory(t), layers, state); err == nil {
		t.Fatal("preexisting supply state overwritten")
	}
	root, next := fixture(t)
	body, err := os.ReadFile(filepath.Join(root, runtimeFile))
	if err != nil {
		t.Fatal(err)
	}
	// Synthetic previous pin only; no claim that this version/image exists.
	body, err = editJSON(body, map[string]any{"/uv_image/uv_version": "0.0.1", "/uv_image/reference": "ghcr.io/astral-sh/uv:0.0.1@sha256:" + m.UV.Identity.ManifestSHA256})
	if err != nil {
		t.Fatal(err)
	}
	writeTest(t, filepath.Join(root, runtimeFile), body)
	body, err = os.ReadFile(filepath.Join(root, supportFile))
	if err != nil {
		t.Fatal(err)
	}
	var support struct {
		Records []map[string]any `json:"records"`
	}
	if err := json.Unmarshal(body, &support); err != nil {
		t.Fatal(err)
	}
	var review []map[string]any
	for i, record := range support.Records {
		if record["component"] == "uv" {
			review = append(review, record)
			body, err = editJSON(body, map[string]any{"/records/" + strconv.Itoa(i) + "/pinned_version": "0.0.1"})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	writeTest(t, filepath.Join(root, supportFile), body)
	if _, err := command(root, nil, "go", "run", "./scripts/generate-runtime-lock.go"); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "--all")
	commitTest(t, root)
	reviewPath := filepath.Join(t.TempDir(), "review.json")
	body, err = json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	writeTest(t, reviewPath, body)
	p, err := makePlanResolving(root, next, "uv", "0.13.4", reviewPath, func(_, version string) (metadata, error) {
		return freezeUVSupply(version, uvProofDirectory(t), layers, next)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Changes) != 3 {
		t.Fatal("unexpected coordinated paths", len(p.Changes))
	}
	if err := applyPlan(p, next, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	files, err = os.ReadDir(filepath.Join(next, "uv-supply"))
	if err != nil {
		t.Fatal(err)
	}
	path = filepath.Join(next, "uv-supply", files[0].Name())
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(p, next, io.Discard, nil); err == nil {
		t.Fatal("changed frozen file permissions accepted on reapply")
	}
	if err := verifyPlan(p, next, "", io.Discard); err == nil {
		t.Fatal("verify accepted mutated frozen supply")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(layers, files[0].Name()), path); err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(p, next, io.Discard, nil); err == nil {
		t.Fatal("symlink frozen layer accepted on reapply")
	}
	if err := compareSnapshot(p); err != nil {
		t.Fatal("source mutated", err)
	}
}
