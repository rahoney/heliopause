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

func TestPNPMFrozenActualSupplyPlanAndInterruptedApply(t *testing.T) {
	input := os.Getenv("HELOX_PNPM_RELEASE_ROOT")
	if !filepath.IsAbs(input) {
		t.Fatal("HELOX_PNPM_RELEASE_ROOT must declare the official proof/archive input")
	}
	root, state := fixture(t)
	// Synthetic previous pin proves coordination, not an official old release
	// or execution/qualification of that version.
	var lock map[string]any
	body, err := os.ReadFile(filepath.Join(root, runtimeFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		t.Fatal(err)
	}
	pnpm := lock["pnpm_tool"].(map[string]any)
	current := pnpm["pnpm_version"].(string)
	pnpm["pnpm_version"] = "0.0.1"
	archive := pnpm["archive"].(map[string]any)
	archive["url"] = strings.Replace(archive["url"].(string), "/v"+current+"/", "/v0.0.1/", 1)
	body, err = json.MarshalIndent(lock, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeTest(t, filepath.Join(root, runtimeFile), append(body, '\n'))
	var support map[string]any
	body, err = os.ReadFile(filepath.Join(root, supportFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &support); err != nil {
		t.Fatal(err)
	}
	var review map[string]any
	for _, raw := range support["records"].([]any) {
		row := raw.(map[string]any)
		if row["component"] == "pnpm" {
			review = make(map[string]any)
			for k, v := range row {
				review[k] = v
			}
			row["pinned_version"] = "0.0.1"
			row["reason"] = "synthetic previous fixture pin; not an upstream release claim"
		}
	}
	if review == nil {
		t.Fatal("pnpm support owner missing")
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
	reviewPath := filepath.Join(state, "review.json")
	body, err = json.Marshal([]map[string]any{review})
	if err != nil {
		t.Fatal(err)
	}
	writeTest(t, reviewPath, body)
	p, err := makePlanResolving(root, state, "pnpm", current, reviewPath, func(component, version string) (metadata, error) { return freezePNPMSupply(version, input, state) })
	if err != nil {
		t.Fatal(err)
	}
	if p.Metadata.PNPM == nil || p.Metadata.Image != "" || len(p.Changes) != 3 || len(p.RequiredQualification) == 0 {
		t.Fatalf("bad native plan: %+v", p)
	}
	first := errors.New("forced native plan interruption")
	if err := applyPlan(p, state, &bytes.Buffer{}, func(index int) error {
		if index == 0 {
			return first
		}
		return nil
	}); !errors.Is(err, first) {
		t.Fatalf("first failure lost: %v", err)
	}
	if err := applyPlan(p, state, &bytes.Buffer{}, nil); err != nil {
		t.Fatal(err)
	}
	if err := applyPlan(p, state, &bytes.Buffer{}, nil); err != nil {
		t.Fatal(err)
	}
	if err := compareSnapshot(p); err != nil {
		t.Fatal("original repository changed:", err)
	}
	frozen := filepath.Join(state, "pnpm-supply", p.Metadata.PNPM.Identity.ArchiveSHA256+".tar.gz")
	file, err := os.OpenFile(frozen, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte{0}, 0); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := renderChanges(p, state); err == nil {
		t.Fatal("changed frozen archive accepted")
	}
}
