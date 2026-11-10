package pypi

import (
	"bytes"
	"testing"
)

func TestWheelLegacyDistInfoPreservesMeasuredConsolePath(t *testing.T) {
	directory := "Example-1.0.0.dist-info"
	body := validWheelArchive(t, "example", "1.0.0", "example/__init__.py", directory, wheelTestEntry{name: directory + "/entry_points.txt", body: []byte("[console_scripts]\nexample = example:main\n")})
	artifact := consoleExecutionArtifact(t, body, "example", "1.0.0", "wheel", PublicPyPIProfile().Source())
	inventory, err := InspectConsoleExecutionWheel(artifact, bytes.NewReader(body), "example-1.0.0-cp314-cp314-manylinux_2_36_x86_64.whl", WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
	if err != nil {
		t.Fatal("legacy directory lost its original source/RECORD identity", err)
	}
	entry, err := SelectConsoleExecutionEntry(inventory, "example")
	if err != nil || entry.Metadata().Path != directory+"/entry_points.txt" {
		t.Fatal("literal authenticated console path changed", err)
	}
}

func TestUVInstalledControlsNormalizeLegacyDirectoryIdentityOnly(t *testing.T) {
	for _, directory := range []string{"Unidecode-1.4.0.dist-info", "unidecode-1.4.0.dist-info"} {
		value, err := readUVInstalledWheelControl(UVInstalledWheelControls{directory, []byte("Name: Unidecode\nVersion: 1.4.0\n"), []byte("Wheel-Version: 1.0\nTag: py3-none-any\n")})
		if err != nil || value.Name != "unidecode" || value.Version != "1.4.0" {
			t.Fatal("legacy control did not retain normalized identity", directory, err)
		}
	}
	for _, directory := range []string{"Other-1.4.0.dist-info", "Unidecode-1.4.1.dist-info", "../Unidecode-1.4.0.dist-info", "Unidecode-1.4.0.dist-info/", "Unidecode--1.4.0.dist-info", "Unidecode-1.4.0.other"} {
		if _, err := readUVInstalledWheelControl(UVInstalledWheelControls{directory, []byte("Name: Unidecode\nVersion: 1.4.0\n"), []byte("Wheel-Version: 1.0\nTag: py3-none-any\n")}); err == nil {
			t.Fatal("foreign/unsafe directory accepted", directory)
		}
	}
}

func TestWheelLegacyDistInfoRejectsSecondNormalizedDirectory(t *testing.T) {
	for _, member := range []string{"example-1.0.0.dist-info/METADATA", "example-1.0.0.dist-info/foreign.txt", "Other-1.0.0.dist-info/METADATA"} {
		body := validWheelArchive(t, "example", "1.0.0", "example/__init__.py", "Example-1.0.0.dist-info", wheelTestEntry{name: member, body: []byte("Name: example\nVersion: 1.0.0\n")})
		artifact := consoleExecutionArtifact(t, body, "example", "1.0.0", "wheel", PublicPyPIProfile().Source())
		if _, err := InspectWheel(bytes.NewReader(body), int64(len(body)), "example-1.0.0-cp314-cp314-manylinux_2_36_x86_64.whl", artifact.Digest().String(), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits()); err == nil {
			t.Fatal("foreign/second alias directory accepted", member)
		}
	}
}
