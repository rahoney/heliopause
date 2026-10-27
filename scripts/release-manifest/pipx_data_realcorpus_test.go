//go:build realcorpus

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// This verifies a local unsigned producer candidate. A claimed workflow identity
// in its manifest is fixture data, never publisher proof or hosted qualification.
func TestPipxActualProducerBundleAndOriginalZIPCensus(t *testing.T) {
	root := os.Getenv("HELOX_PIPX_RELEASE_ACTUAL_DIR")
	if root == "" {
		t.Skip("explicit complete local producer candidate required")
	}
	body, err := os.ReadFile(filepath.Join(root, "helox-release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var declared releaseManifest
	if err := json.Unmarshal(body, &declared); err != nil {
		t.Fatal(err)
	}
	manifest, err := releaseinstall.ParseReleaseManifest(body)
	if err != nil {
		t.Fatal(err)
	}
	if !manifest.HasPipxRuntime() || len(declared.Assets) != 15 {
		t.Fatal("incomplete DATA release", len(declared.Assets))
	}
	names, err := verifyBundle(root, declared.Version, declared.SourceCommit, declared.WorkflowRun)
	if err != nil || len(names) != 19 {
		t.Fatal("exact bundle rejected", names, err)
	}
	images, err := os.ReadFile(filepath.Join(root, declared.RuntimeImages.Name))
	if err != nil {
		t.Fatal(err)
	}
	if err := releaseinstall.ValidateRuntimeImagesForRelease(images, manifest); err != nil {
		t.Fatal(err)
	}
	sbomBody, err := os.ReadFile(filepath.Join(root, declared.SBOM.Name))
	if err != nil {
		t.Fatal(err)
	}
	if err := releaseinstall.ValidateReleaseSBOM(sbomBody, manifest); err != nil {
		t.Fatal(err)
	}
	zipBody, err := os.ReadFile(filepath.Join(root, runtimeidentity.PipxRuntimeAssetName))
	if err != nil {
		t.Fatal(err)
	}
	original, err := pypi.ParsePipxRuntimeZIP(zipBody)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := original.ObservationPlan()
	if err != nil {
		t.Fatal(err)
	}
	var sbom cyclonedxSBOM
	if err := json.Unmarshal(sbomBody, &sbom); err != nil {
		t.Fatal(err)
	}
	var actual []pypi.PipxRuntimeDistribution
	seen := map[string]bool{}
	for _, component := range sbom.Components {
		project := strings.TrimPrefix(component.Name, "pypi/")
		if component.Name != "pipx" && project == component.Name {
			continue
		}
		wantType := "library"
		if project == "pipx" {
			wantType = "application"
		}
		if component.Type != wantType || seen[project] {
			t.Fatal("duplicate or incorrect SBOM distribution role", component)
		}
		seen[project] = true
		actual = append(actual, pypi.PipxRuntimeDistribution{Project: project, Version: component.Version})
	}
	sort.Slice(actual, func(i, j int) bool { return actual[i].Project < actual[j].Project })
	if len(actual) != 11 || !reflect.DeepEqual(actual, plan.Distributions()) {
		t.Fatal("SBOM differs from original ZIP metadata", actual, plan.Distributions())
	}
	t.Logf("unsigned local producer: assets=%d exact bundle=%d runtime manifest v3 tools3; original ZIP members=%d distributions=%v; no artifact execution/authenticity claim", len(declared.Assets), len(names), len(original.Files()), actual)
}
