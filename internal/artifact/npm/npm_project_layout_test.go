package npm

import (
	"archive/tar"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func npmWholeLayoutFixture(t *testing.T) (NPMProjectInspectionInput, []ArchiveInventory, []layoutTarTestEntry) {
	t.Helper()
	frozen, acquired, inventories := npmWholeInspectionFixture(t)
	input, err := frozen.BindAcquired(frozen.Snapshot(), acquired)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]ArchiveInventory{}
	for _, inventory := range inventories {
		byName[inventory.Artifact().Identity().Name()] = inventory
	}
	// Expected locations are independent of the production tree builder.
	entries := []layoutTarTestEntry{
		layoutTestEntry("node_modules", tar.TypeDir, "", ""),
		layoutTestEntry("node_modules/alpha", tar.TypeDir, "", ""),
		layoutTestEntry("node_modules/alpha/node_modules", tar.TypeDir, "", ""),
		layoutTestEntry("node_modules/alpha/node_modules/shared", tar.TypeDir, "", ""),
		layoutTestEntry("node_modules/alpha/node_modules/shared/package.json", tar.TypeReg, string(byName["shared"].Manifest()), ""),
		layoutTestEntry("node_modules/alpha/package.json", tar.TypeReg, string(byName["alpha"].Manifest()), ""),
		layoutTestEntry("node_modules/alpha/cli.js", tar.TypeReg, npmWholeCLIContent, ""),
		layoutTestEntry("node_modules/beta", tar.TypeDir, "", ""),
		layoutTestEntry("node_modules/beta/package.json", tar.TypeReg, string(byName["beta"].Manifest()), ""),
		layoutTestEntry("node_modules/shared", tar.TypeDir, "", ""),
		layoutTestEntry("node_modules/shared/package.json", tar.TypeReg, string(byName["shared"].Manifest()), ""),
		layoutTestEntry("node_modules/.bin", tar.TypeDir, "", ""),
		layoutTestEntry("node_modules/.bin/alpha-cli", tar.TypeSymlink, "", "../alpha/cli.js"),
	}
	entries[6].header.Mode = 0o755
	return input, inventories, entries
}

func TestNPMWholeLayoutBindsAllPlacementsBinsAndFreshSubjects(t *testing.T) {
	input, inventories, entries := npmWholeLayoutFixture(t)
	native, err := ParsePNPMLayoutArchive(context.Background(), layoutTarBody(t, entries))
	if err != nil {
		t.Fatal(err)
	}
	bound, err := input.BindLayout(context.Background(), inventories, native)
	if err != nil || bound.Digest().String() == "" || !bound.MatchesInput(input) {
		t.Fatal("complete native layout rejected", err)
	}
	for _, subject := range input.Artifacts() {
		instances, _ := input.InstancesFor(subject)
		for _, instance := range instances {
			if target, err := bound.Target(subject, instance.Location); err != nil || target != instance.Location {
				t.Fatal("logical instance lost its target", err)
			}
		}
	}
	if _, err := bound.Target(input.Artifacts()[0], "node_modules/foreign"); err == nil {
		t.Fatal("unbound target admitted")
	}
	var manifest map[string]json.RawMessage
	if json.Unmarshal(bound.InspectionManifest(), &manifest) != nil || manifest["scripts"] != nil {
		t.Fatal("dependency inspection inherited root scripts")
	}
	body, err := bound.ReadFile("node_modules/alpha/cli.js")
	if err != nil || string(body) != npmWholeCLIContent {
		t.Fatal("bound bytes unavailable", err)
	}
	body[0] = '!'
	again, _ := bound.ReadFile("node_modules/alpha/cli.js")
	if string(again) != npmWholeCLIContent {
		t.Fatal("caller changed sealed bytes")
	}
	if (NPMProjectInspectionLayout{}).MatchesInput(input) {
		t.Fatal("zero layout matched current context")
	}
}

func TestNPMWholeLayoutRejectsMissingChangedForeignAndBinLinks(t *testing.T) {
	input, inventories, original := npmWholeLayoutFixture(t)
	for _, fault := range []string{"missing", "changed", "foreign", "wrong bin", "unknown bin", "source metadata"} {
		t.Run(fault, func(t *testing.T) {
			entries := append([]layoutTarTestEntry(nil), original...)
			switch fault {
			case "missing":
				entries = append(entries[:4], entries[5:]...)
			case "changed":
				entries[6].body = strings.Repeat("!", len(entries[6].body))
			case "foreign":
				entries = append(entries, layoutTestEntry("node_modules/alpha/extra.js", tar.TypeReg, "injected", ""))
			case "wrong bin":
				entries[len(entries)-1].header.Linkname = "../beta/package.json"
			case "unknown bin":
				entries = append(entries, layoutTestEntry("node_modules/.bin/unknown", tar.TypeSymlink, "", "../alpha/cli.js"))
			case "source metadata":
				entries[5].body = strings.Replace(entries[5].body, "^1", "^2", 1)
			}
			native, err := ParsePNPMLayoutArchive(context.Background(), layoutTarBody(t, entries))
			if err != nil {
				t.Fatal("fixture did not reach source/tree binder", err)
			}
			if _, err := input.BindLayout(context.Background(), inventories, native); err == nil {
				t.Fatal("changed/unbound layout admitted")
			}
		})
	}
}

func TestNPMWholeOfflineControlsRetainSourceSemanticsAndBindLocalTarballs(t *testing.T) {
	input, _, _ := npmWholeLayoutFixture(t)
	original := input.Context().Controls()
	controls, err := input.OfflineControls()
	if err != nil || len(controls) != 2 || string(controls[0].Body()) != string(original[0].Body()) {
		t.Fatal("private recipe changed manifest", err)
	}
	var lock struct {
		Packages map[string]struct{ Resolved string }
	}
	if json.Unmarshal(controls[1].Body(), &lock) != nil {
		t.Fatal("private recipe lock malformed")
	}
	for _, artifact := range input.Artifacts() {
		instances, _ := input.InstancesFor(artifact)
		for _, instance := range instances {
			if lock.Packages[instance.Location].Resolved != "file:.heliopause/artifacts/"+artifact.Digest().String()+".tgz" {
				t.Fatal("local recipe omitted a placement or reconstructed a source URL")
			}
		}
	}
	if string(input.Context().Controls()[1].Body()) != string(original[1].Body()) {
		t.Fatal("private recipe mutated original public lock")
	}
	if _, err := (NPMProjectInspectionInput{}).OfflineControls(); err == nil {
		t.Fatal("unbound input rendered a recipe")
	}
}
