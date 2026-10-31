package main

import (
	"io"
	"path/filepath"
	"testing"
)

func TestUVSupplyRefusesUnverifiedOrWrongVersionInputs(t *testing.T) {
	for _, paths := range [][2]string{{".", "."}, {uvProofDirectory(t), t.TempDir()}} {
		if _, err := freezeUVSupply("0.13.4", paths[0], paths[1], t.TempDir()); err == nil {
			t.Fatal("missing authenticated supply accepted")
		}
	}
	if _, err := freezeUVSupply("0.0.1", uvProofDirectory(t), t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("foreign version accepted")
	}
	if err := verifyFrozenUVSupply("0.13.4", metadata{}, t.TempDir()); err == nil {
		t.Fatal("missing proof accepted")
	}
}

func TestUVSameVersionDoesNotAcquireAndRejectsNewSupplyFlags(t *testing.T) {
	root, state := fixture(t)
	version, err := selectedVersion(root, "uv")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	p, err := makePlanResolving(root, state, "uv", version, "", func(string, string) (metadata, error) { calls++; return metadata{}, nil })
	if err != nil || calls != 0 || len(p.Changes) != 0 || p.Metadata.UV != nil {
		t.Fatal("same-version acquired or changed supply", calls, err)
	}
	for _, flag := range []string{"--uv-proof", "--uv-layers"} {
		args := []string{"plan", "--root", root, "--state", filepath.Join(t.TempDir(), "state"), "--component", "uv", "--version", version, flag, uvProofDirectory(t)}
		if err := run(args, io.Discard); err == nil {
			t.Fatal("same-version accepted new supply flag", flag)
		}
	}
}
