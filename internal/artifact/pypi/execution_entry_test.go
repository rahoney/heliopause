package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func consoleExecutionWheel(t *testing.T, points, extra string) ([]byte, domain.AcquiredArtifact, ConsoleExecutionInventory) {
	t.Helper()
	entries := []wheelTestEntry{{name: "example-1.0.0.dist-info/entry_points.txt", body: []byte(points)}}
	if extra != "" {
		entries = append(entries, wheelTestEntry{name: "example/state.txt", body: []byte(extra)})
	}
	body := validWheelArchive(t, "example", "1.0.0", "example/__init__.py", "example-1.0.0.dist-info", entries...)
	artifact := consoleExecutionArtifact(t, body, "example", "1.0.0", "wheel", PublicPyPIProfile().Source())
	inventory, err := InspectConsoleExecutionWheel(artifact, bytes.NewReader(body), "example-1.0.0-cp314-cp314-manylinux_2_36_x86_64.whl", WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	return body, artifact, inventory
}

func consoleExecutionArtifact(t *testing.T, body []byte, name, version, variant string, source domain.SourceID) domain.AcquiredArtifact {
	t.Helper()
	identity, err := domain.NewResolvedArtifactIdentity(source, name, version, variant)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":"+variant, uint64(len(body)), "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func TestConsoleExecutionEntryBindsCanonicalWheelMetadata(t *testing.T) {
	_, artifact, inventory := consoleExecutionWheel(t, "[console_scripts]\nexample = example.cli:main.run\n", "")
	selected, err := SelectConsoleExecutionEntry(inventory, "")
	if err != nil || !inventory.Valid() || !selected.Valid() || !selected.Matches(inventory) || selected.Artifact() != artifact || selected.Command() != "example" {
		t.Fatal("canonical selection incomplete", err)
	}
	entry := selected.EntryPoint()
	metadata := selected.Metadata()
	if entry.Module != "example.cli" || entry.Attr != "main.run" || metadata.Path != "example-1.0.0.dist-info/entry_points.txt" || !validSHA256(metadata.SHA256) || metadata.Size <= 0 {
		t.Fatal("declaration/member facts lost")
	}
	entry.Module = "foreign"
	entry.Extras = append(entry.Extras, "foreign")
	metadata.Path = "foreign"
	if selected.EntryPoint().Module != "example.cli" || len(selected.EntryPoint().Extras) != 0 || selected.Metadata().Path == "foreign" {
		t.Fatal("getter changed private source facts")
	}
}

func TestConsoleExecutionEntryRequiresExactSupportedConsoleSelection(t *testing.T) {
	for _, row := range []struct {
		name, points, command string
		valid                 bool
	}{
		{"different default", "[console_scripts]\nother = example:main\n", "", false},
		{"different explicit", "[console_scripts]\nother = example:main\n", "other", true},
		{"multi default", "[console_scripts]\nexample = example:main\nother = example:other\n", "", false},
		{"multi explicit", "[console_scripts]\nexample = example:main\nother = example:other\n", "other", true},
		{"same-target aliases", "[console_scripts]\nexample = example:main\nother = example:main\n", "", false},
		{"absent", "[console_scripts]\nexample = example:main\n", "absent", false},
		{"gui", "[gui_scripts]\nexample = example:main\n", "example", false},
		{"plugin", "[example.plugin]\nexample = example:main\n", "example", false},
		{"no callable", "[console_scripts]\nexample = example\n", "example", false},
		{"extras", "[console_scripts]\nexample = example:main [extra]\n", "example", false},
		{"unsafe command", "[console_scripts]\nexample = example:main\n", "../example", false},
		{"leading dash", "[console_scripts]\nexample = example:main\n", "-example", false},
		{"long command", "[console_scripts]\nexample = example:main\n", strings.Repeat("x", 215), false},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, _, inventory := consoleExecutionWheel(t, row.points, "")
			entry, err := SelectConsoleExecutionEntry(inventory, row.command)
			if (err == nil) != row.valid || entry.Valid() != row.valid {
				t.Fatal("selection scope changed", err)
			}
		})
	}
	if entry, err := SelectConsoleExecutionEntry(ConsoleExecutionInventory{}, "example"); err == nil || entry.Valid() || (ConsoleExecutionEntry{}).Valid() {
		t.Fatal("zero proof accepted")
	}
}

func TestConsoleExecutionEntryRejectsOtherAcquisitionAndWholeSource(t *testing.T) {
	points := "[console_scripts]\nexample = example:main\n"
	body, _, first := consoleExecutionWheel(t, points, "first")
	entry, err := SelectConsoleExecutionEntry(first, "example")
	if err != nil {
		t.Fatal(err)
	}
	_, _, other := consoleExecutionWheel(t, points, "other")
	if entry.Matches(other) {
		t.Fatal("same console metadata hid another archive member")
	}
	another := consoleExecutionArtifact(t, body, "example", "1.0.0", "wheel", PublicPyPIProfile().Source())
	rebound, err := InspectConsoleExecutionWheel(another, bytes.NewReader(body), "example-1.0.0-cp314-cp314-manylinux_2_36_x86_64.whl", WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	if entry.Matches(rebound) {
		t.Fatal("same bytes substituted another acquisition handle")
	}
}

func TestConsoleExecutionEntryMatchesRepeatedOriginalMeasurement(t *testing.T) {
	body, artifact, first := consoleExecutionWheel(t, "[console_scripts]\nexample = example:main\n", "same original resource")
	entry, err := SelectConsoleExecutionEntry(first, "example")
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 64; attempt++ {
		measured, err := InspectConsoleExecutionWheel(artifact, bytes.NewReader(body), "example-1.0.0-cp314-cp314-manylinux_2_36_x86_64.whl", WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
		if err != nil || !entry.Matches(measured) {
			t.Fatalf("identical acquired bytes diverged at measurement %d: %v", attempt, err)
		}
	}
}

func TestConsoleExecutionEntryMatchesRepeatedHookSource(t *testing.T) {
	body := validWheelArchive(t, "example", "1.0.0", "example/__init__.py", "example-1.0.0.dist-info",
		wheelTestEntry{name: "example-1.0.0.dist-info/entry_points.txt", body: []byte("[console_scripts]\nexample = example:main\n")},
		wheelTestEntry{name: "alpha.pth", body: []byte("import alpha_hook\nimport other_hook\n")},
		wheelTestEntry{name: "beta.pth", body: []byte("import beta_hook\n")})
	artifact := consoleExecutionArtifact(t, body, "example", "1.0.0", "wheel", PublicPyPIProfile().Source())
	var entry ConsoleExecutionEntry
	for attempt := 0; attempt < 64; attempt++ {
		inventory, err := InspectConsoleExecutionWheel(artifact, bytes.NewReader(body), "example-1.0.0-cp314-cp314-manylinux_2_36_x86_64.whl", WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
		if err != nil {
			t.Fatal(err)
		}
		if attempt == 0 {
			entry, err = SelectConsoleExecutionEntry(inventory, "example")
			if err != nil {
				t.Fatal(err)
			}
		}
		if !entry.Matches(inventory) {
			t.Fatal("same hook source diverged across canonical parser reads")
		}
	}
}

func TestConsoleExecutionInventoryRejectsUnboundAndCorruptInputs(t *testing.T) {
	body, artifact, _ := consoleExecutionWheel(t, "[console_scripts]\nexample = example:main\n", "")
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	filename := "example-1.0.0-cp314-cp314-manylinux_2_36_x86_64.whl"
	foreign, _ := domain.NewSourceID("foreign")
	for _, subject := range []domain.AcquiredArtifact{
		{}, consoleExecutionArtifact(t, body, "other", "1.0.0", "wheel", PublicPyPIProfile().Source()), consoleExecutionArtifact(t, body, "example", "2.0.0", "wheel", PublicPyPIProfile().Source()), consoleExecutionArtifact(t, body, "example", "1.0.0", "sdist", PublicPyPIProfile().Source()), consoleExecutionArtifact(t, body, "example", "1.0.0", "derived-wheel", PublicPyPIProfile().Source()), consoleExecutionArtifact(t, body, "example", "1.0.0", "wheel", foreign),
	} {
		if inventory, err := InspectConsoleExecutionWheel(subject, bytes.NewReader(body), filename, target, DefaultWheelLimits()); err == nil || inventory.Valid() {
			t.Fatal("unbound source/identity accepted")
		}
	}
	corrupt := append([]byte(nil), body...)
	corrupt[len(corrupt)/2] ^= 1
	if inventory, err := InspectConsoleExecutionWheel(artifact, bytes.NewReader(corrupt), filename, target, DefaultWheelLimits()); err == nil || inventory.Valid() {
		t.Fatal("wrong original bytes accepted")
	}
	if inventory, err := InspectConsoleExecutionWheel(artifact, nil, filename, target, DefaultWheelLimits()); err == nil || inventory.Valid() {
		t.Fatal("missing reader accepted")
	}
	limited := DefaultWheelLimits()
	limited.MaxCompressed = int64(len(body) - 1)
	if inventory, err := InspectConsoleExecutionWheel(artifact, bytes.NewReader(body), filename, target, limited); err == nil || inventory.Valid() {
		t.Fatal("caller resource bound ignored")
	}
	without := validWheelArchive(t, "example", "1.0.0", "example/__init__.py", "example-1.0.0.dist-info")
	if inventory, err := InspectConsoleExecutionWheel(consoleExecutionArtifact(t, without, "example", "1.0.0", "wheel", PublicPyPIProfile().Source()), bytes.NewReader(without), filename, target, DefaultWheelLimits()); err == nil || inventory.Valid() {
		t.Fatal("missing exact console metadata member accepted")
	}
	undeclared, err := domain.NewAcquiredArtifact(artifact.Identity(), artifact.Digest(), artifact.ContentHandle(), artifact.SizeBytes())
	if err != nil {
		t.Fatal(err)
	}
	if inventory, err := InspectConsoleExecutionWheel(undeclared, bytes.NewReader(body), filename, target, DefaultWheelLimits()); err == nil || inventory.Valid() {
		t.Fatal("observed-only acquisition became a bound declaration")
	}
	entries := validWheelEntries(t, "example", "1.0.0", "example/__init__.py", "example-1.0.0.dist-info", wheelTestEntry{name: "example-1.0.0.dist-info/entry_points.txt", body: []byte("[console_scripts]\nexample = example:main\n")})
	for i := range entries {
		if entries[i].name == "example-1.0.0.dist-info/entry_points.txt" {
			entries[i].body = []byte("[console_scripts]\nexample = example:other\n")
		}
	}
	badRecord := makeWheelEntries(t, entries...)
	if inventory, err := InspectConsoleExecutionWheel(consoleExecutionArtifact(t, badRecord, "example", "1.0.0", "wheel", PublicPyPIProfile().Source()), bytes.NewReader(badRecord), filename, target, DefaultWheelLimits()); err == nil || inventory.Valid() {
		t.Fatal("valid archive digest bypassed original RECORD member binding")
	}
}
