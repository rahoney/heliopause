//go:build realcorpus

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipxFrozenActualSupplyPlanAndInterruptedApply(t *testing.T) {
	input := os.Getenv("HELOX_PIPX_SUPPLY_ROOT")
	if !filepath.IsAbs(input) {
		t.Fatal("HELOX_PIPX_SUPPLY_ROOT must declare frozen official wheel/proof inputs")
	}
	root, state := fixture(t)
	body, err := os.ReadFile(filepath.Join(root, runtimeFile))
	if err != nil {
		t.Fatal(err)
	}
	var lock map[string]any
	if json.Unmarshal(body, &lock) != nil {
		t.Fatal("lock")
	}
	pin := lock["pipx_tool"].(map[string]any)
	current := pin["pipx_version"].(string)
	// This synthetic previous owner exercises coordinated patching, not an
	// authenticated historical pipx release or its runtime qualification.
	pin["pipx_version"] = "0.0.1"
	for _, v := range pin["inputs"].([]any) {
		f := v.(map[string]any)
		f["path"] = strings.Replace(f["path"].(string), "pipx/pipx-"+current+"-", "pipx/pipx-0.0.1-", 1)
	}
	body, err = json.MarshalIndent(lock, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeTest(t, filepath.Join(root, runtimeFile), append(body, '\n'))
	body, err = os.ReadFile(filepath.Join(root, supportFile))
	if err != nil {
		t.Fatal(err)
	}
	var support map[string]any
	if json.Unmarshal(body, &support) != nil {
		t.Fatal("support")
	}
	var review map[string]any
	for _, v := range support["records"].([]any) {
		r := v.(map[string]any)
		if r["component"] == "pipx" {
			review = map[string]any{}
			for k, v := range r {
				review[k] = v
			}
			r["pinned_version"] = "0.0.1"
			r["reason"] = "synthetic previous fixture owner; not an upstream release claim"
		}
	}
	if review == nil {
		t.Fatal("pipx review missing")
	}
	body, err = json.MarshalIndent(support, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeTest(t, filepath.Join(root, supportFile), append(body, '\n'))
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	goBin, err := filepath.EvalSymlinks(goPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := command(root, goEnvironment(goBin), goBin, "run", "./scripts/generate-runtime-lock.go"); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "--all")
	commitTest(t, root)
	body, err = json.Marshal([]map[string]any{review})
	if err != nil {
		t.Fatal(err)
	}
	reviewPath := filepath.Join(state, "review.json")
	writeTest(t, reviewPath, body)
	p, err := makePlanResolving(root, state, "pipx", current, reviewPath, func(component, version string) (metadata, error) { return freezePipxSupply(version, input, state) })
	if err != nil {
		t.Fatal(err)
	}
	if p.Metadata.Pipx == nil || len(p.Metadata.Pipx.Documents) != 20 || p.Metadata.Image != "" || len(p.Changes) != 3 || len(p.RequiredQualification) == 0 {
		t.Fatalf("unexpected frozen plan: changes=%d", len(p.Changes))
	}
	primary := errors.New("forced pipx apply interruption")
	if err := applyPlan(p, state, &bytes.Buffer{}, func(index int) error {
		if index == 0 {
			return primary
		}
		return nil
	}); !errors.Is(err, primary) {
		t.Fatalf("primary lost: %v", err)
	}
	if err := applyPlan(p, state, &bytes.Buffer{}, nil); err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(p, state, &bytes.Buffer{}, nil); err != nil {
		t.Fatal(err)
	}
	if err := compareSnapshot(p); err != nil {
		t.Fatal("source changed", err)
	}
	if err := verifyFrozenPipxSupply(current, p.Metadata, state); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(state, "candidate")
	actual, err := readPipxMaterializationOwner(candidate)
	if err != nil || actual.Version != current {
		t.Fatal("candidate owner differs", err)
	}
	output := t.TempDir()
	var receipt bytes.Buffer
	args := []string{"--root", candidate, "--input", filepath.Join(state, "pipx-supply"), "--output", filepath.Join(output, "pipx-runtime-linux-amd64.zip"), "--notices-output", filepath.Join(output, "pipx-NOTICE.txt")}
	if err := materializePipx(args, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Len() == 0 {
		t.Fatal("missing materialization facts")
	}
	zipBody, err := os.ReadFile(filepath.Join(output, "pipx-runtime-linux-amd64.zip"))
	if err != nil || hash(zipBody) != actual.Tool.SHA256 {
		t.Fatal("candidate ZIP changed", err)
	}
	before, err := os.Stat(filepath.Join(output, "pipx-runtime-linux-amd64.zip"))
	if err != nil {
		t.Fatal(err)
	}
	receipt.Reset()
	if err := materializePipx(args, &receipt); err == nil || receipt.Len() != 0 {
		t.Fatal("existing output replaced")
	}
	after, err := os.Stat(filepath.Join(output, "pipx-runtime-linux-amd64.zip"))
	if err != nil || !os.SameFile(before, after) || after.Mode().Perm() != 0444 {
		t.Fatal("protected ZIP identity changed", err)
	}
	frozen := filepath.Join(state, "pipx-supply", "userpath", "userpath-1.9.2-py3-none-any.whl")
	f, err := os.OpenFile(frozen, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := f.WriteAt([]byte{0}, 0)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatal(writeErr, closeErr)
	}
	if err := verifyFrozenPipxSupply(current, p.Metadata, state); err == nil {
		t.Fatal("frozen wheel tamper accepted")
	}
	if err := verifyPlan(p, state, goBin, &bytes.Buffer{}); err == nil {
		t.Fatal("tampered supply reached post-apply checks")
	}
	if err := compareSnapshot(p); err != nil {
		t.Fatal("source changed after refusal", err)
	}
}
