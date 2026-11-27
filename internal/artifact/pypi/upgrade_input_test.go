package pypi

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestUpgradeRequirementKeepsNativeIntentAndRejectsUnsupportedOptions(t *testing.T) {
	for _, item := range []struct {
		input, canonical, allowed, rejected string
	}{
		{"Requests", "requests", "2.32.5", "3.0rc1"},
		{"Requests==02.032.05", "requests==2.32.5", "2.32.5", "2.32.6"},
		{"requests==2", "requests==2", "2.0.0", "2.1"},
		{"requests@2.32.5", "requests==2.32.5", "2.32.5", "2.32.6"},
		{"requests==3.0rc1", "requests==3.0rc1", "3.0rc1", "3.0"},
		{"requests>=2,<3", "requests>=2,<3", "2.32.5", "3.0"},
		{"Requests ~=2.32", "requests~=2.32", "2.33", "3.0"},
	} {
		requirement, err := ParseUpgradeRequirement(item.input)
		if err != nil || requirement.Project() != "requests" || requirement.Requirement() != item.canonical || !requirement.AcceptsVersion(item.allowed) || requirement.AcceptsVersion(item.rejected) || requirement.Reference().Source().String() != "pypi" {
			t.Fatal(item, requirement, err)
		}
	}
	for _, input := range []string{"", " requests", "requests ", "-U", "--index-url=https://elsewhere", "requests[security]", "requests;python_version>'3'", "requests @ https://elsewhere/a.whl", "../requests", "requests==", "requests===2", "requests!=2", "requests\n--no-deps", "requests\t==2", "requests==1+local", strings.Repeat("a", 513)} {
		if _, err := ParseUpgradeRequirement(input); err == nil {
			t.Fatal("unsupported upgrade requirement accepted", input)
		}
	}
}

func upgradeMetadataFile(name, version, requirements string) InstalledMetadataFile {
	return InstalledMetadataFile{wheelDistInfo(name, version), []byte("Metadata-Version: 2.4\nName: " + name + "\nVersion: " + version + "\n" + requirements + "\n")}
}

func TestUpgradeInputPreservesInertMetadataAndActiveRetainedConstraints(t *testing.T) {
	requirement, _ := ParseUpgradeRequirement("root>=1,<3")
	files := []InstalledMetadataFile{
		upgradeMetadataFile("retained", "1.0", "Requires-Dist: dependency<2\nRequires-Dist: ignored; sys_platform == 'darwin'\nRequires-Dist: extra-only; extra == 'test'\nRequires-Dist: linux-dep>=1; platform_system == 'Linux'\n"),
		upgradeMetadataFile("root", "1.0", "Requires-Dist: dependency>=1\n"),
	}
	input, err := NewUpgradeInput(requirement, files, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	control, err := input.Control()
	if err != nil || len(input.Installed()) != 2 {
		t.Fatal(err)
	}
	parsed, err := ParseUpgradeInput([]domain.ProjectControlFile{control}, "3.14.7")
	if err != nil || parsed.Requirement().Requirement() != "root>=1,<3" || parsed.Python() != "3.14.7" {
		t.Fatal(parsed, err)
	}
	for _, project := range parsed.Installed() {
		if project.Project() == "retained" && strings.Join(project.Requirements(), "|") != "dependency<2|linux-dep>=1" {
			t.Fatal("actual retained constraints lost or inactive extra activated", project.Requirements())
		}
	}
	files[0].Body[0] = 'X'
	copy := input.MetadataFiles()
	copy[0].Body[0] = 'Y'
	constraints := input.Installed()[0].Requirements()
	constraints[0] = "tampered"
	after, _ := input.Control()
	if !bytes.Equal(control.Body(), after.Body()) {
		t.Fatal("caller mutation changed frozen upgrade input")
	}
	altered := input.MetadataFiles()
	altered[0].Body = append(altered[0].Body, []byte("changed description")...)
	other, err := NewUpgradeInput(requirement, altered, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	changed, _ := other.Control()
	if changed.Digest() == control.Digest() {
		t.Fatal("changed actual metadata did not change input identity")
	}
}

func TestUpgradeInputRejectsAmbiguityUnsupportedMetadataAndNonCanonicalControls(t *testing.T) {
	requirement, _ := ParseUpgradeRequirement("root")
	good := upgradeMetadataFile("root", "1.0", "")
	for _, mode := range []string{"duplicate-project", "duplicate-name", "directory-escape", "directory-version", "direct-url", "folded-option", "active-extras", "unknown-marker", "metadata-limit", "count-limit", "total-limit"} {
		t.Run(mode, func(t *testing.T) {
			file := InstalledMetadataFile{good.Directory, append([]byte(nil), good.Body...)}
			files := []InstalledMetadataFile{file}
			switch mode {
			case "duplicate-project":
				files = append(files, file)
			case "duplicate-name":
				files[0].Body = []byte("Name: root\nName: another\nVersion: 1.0\n\n")
			case "directory-escape":
				files[0].Directory = "../../root-1.0.dist-info"
			case "directory-version":
				files[0].Directory = "root-2.0.dist-info"
			case "direct-url":
				files[0] = upgradeMetadataFile("root", "1.0", "Requires-Dist: dependency @ https://elsewhere/pkg.whl\n")
			case "folded-option":
				files[0] = upgradeMetadataFile("root", "1.0", "Requires-Dist: dependency>=1\n --index-url=https://elsewhere\n")
			case "active-extras":
				files[0] = upgradeMetadataFile("root", "1.0", "Requires-Dist: dependency[extra]>=1\n")
			case "unknown-marker":
				files[0] = upgradeMetadataFile("root", "1.0", "Requires-Dist: dependency; os_name == 'posix'\n")
			case "metadata-limit":
				files[0].Body = bytes.Repeat([]byte{'x'}, defaultWheelMaxMetadata+1)
			case "count-limit":
				files = make([]InstalledMetadataFile, maxPyPIReportEntries+1)
			case "total-limit":
				files = nil
				for _, name := range []string{"one", "two", "three", "four", "five"} {
					item := upgradeMetadataFile(name, "1.0", "")
					item.Body = append(item.Body, bytes.Repeat([]byte{'x'}, defaultWheelMaxMetadata-len(item.Body))...)
					files = append(files, item)
				}
			}
			if _, err := NewUpgradeInput(requirement, files, "3.14.7"); err == nil {
				t.Fatal("ambiguous or unsupported frozen environment accepted")
			}
		})
	}
	input, err := NewUpgradeInput(requirement, nil, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	control, _ := input.Control()
	for _, body := range [][]byte{append([]byte(" "), control.Body()...), bytes.Replace(control.Body(), []byte(`"version":1`), []byte(`"version":1,"version":1`), 1), bytes.Replace(control.Body(), []byte(`"version":1`), []byte(`"version":1,"extra":true`), 1)} {
		changed, _ := domain.NewProjectControlFile(UpgradeInputControlName, body, true)
		if _, err := ParseUpgradeInput([]domain.ProjectControlFile{changed}, "3.14.7"); err == nil {
			t.Fatal("noncanonical frozen input accepted")
		}
	}
	if _, err := ParseUpgradeInput([]domain.ProjectControlFile{control}, "3.14.8"); err == nil {
		t.Fatal("input from another runtime accepted")
	}
	for _, python := range []string{"3.13.7", "3.14.7.1", "3.14.7rc1", "3.14.07"} {
		if _, err := NewUpgradeInput(requirement, nil, python); err == nil {
			t.Fatal("unsupported runtime accepted", python)
		}
	}
}
