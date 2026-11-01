package pypi

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestRequirementsInputFreezesOnlyCompleteRelativeClosure(t *testing.T) {
	files := map[string][]byte{
		"requirements.txt": []byte("-r nested/extra.txt\n-c constraints.txt\nrequests>=2\n"),
		"nested/extra.txt": []byte("idna==3.10\n-r ../shared.txt\n"),
		"constraints.txt":  []byte("urllib3<3\n"),
		"shared.txt":       []byte("certifi\n"),
	}
	sources := []RequirementsSource{{Name: "requirements.txt"}}
	input, err := NewRequirementsInput(sources, files, []string{"packaging==25.0"})
	if err != nil {
		t.Fatal(err)
	}
	control, err := input.Control()
	if err != nil {
		t.Fatal(err)
	}
	before := control.Body()
	files["shared.txt"][0] = '!'
	sources[0].Name = "changed"
	if after, _ := input.Control(); !bytes.Equal(before, after.Body()) {
		t.Fatal("source mutation changed frozen request")
	}
	replayed, err := ParseRequirementsControl(control)
	if err != nil {
		t.Fatal(err)
	}
	if after, _ := replayed.Control(); !bytes.Equal(before, after.Body()) {
		t.Fatal("source replay differs")
	}
}

func TestRequirementsInstallationBindsIntentAndOriginalMetadata(t *testing.T) {
	base, err := NewRequirementsInput(nil, nil, []string{"alpha>=1"})
	if err != nil {
		t.Fatal(err)
	}
	files := []InstalledMetadataFile{{"alpha-1.0.dist-info", []byte("Metadata-Version: 2.1\nName: alpha\nVersion: 1.0\n\n")}}
	for _, client := range []string{"pip", "uv"} {
		input, err := base.WithInstallation(client, true, files)
		if err != nil {
			t.Fatal(err)
		}
		control, _ := input.Control()
		replayed, err := ParseRequirementsControl(control)
		if err != nil {
			t.Fatal(err)
		}
		actual, upgrade, metadata, err := replayed.Installation()
		if err != nil || actual != client || !upgrade || len(metadata) != 1 {
			t.Fatal("installation intent lost", err)
		}
		metadata[0].Body[0] = '!'
		_, _, again, _ := replayed.Installation()
		if again[0].Body[0] == '!' {
			t.Fatal("metadata mutation changed frozen input")
		}
		parser, _ := input.ParserControl()
		if strings.Contains(string(parser.Body()), "Metadata-Version") || strings.Contains(string(parser.Body()), "dist-info") {
			t.Fatal("original metadata entered grammar-only payload")
		}
		if _, err := input.WithInstallation(client, false, nil); err == nil {
			t.Fatal("installation intent rebound")
		}
	}
	for _, invalid := range []struct {
		client string
		files  []InstalledMetadataFile
	}{
		{"other", nil}, {"pip", append(files, files...)}, {"uv", []InstalledMetadataFile{{Directory: "../alpha.dist-info", Body: files[0].Body}}},
	} {
		if _, err := base.WithInstallation(invalid.client, false, invalid.files); err == nil {
			t.Fatal("invalid original metadata accepted")
		}
	}
}

func TestRequirementsInputRefusesMissingCycleEscapeAndUnusedFiles(t *testing.T) {
	for _, files := range []map[string][]byte{
		{"root.txt": []byte("-r absent.txt")},
		{"root.txt": []byte("-r root.txt")},
		{"root.txt": []byte("-c nested.txt"), "nested.txt": []byte("-r root.txt")},
		{"root.txt": []byte("-r ../outside.txt")},
		{"root.txt": []byte("packaging"), "unused.txt": []byte("private data")},
	} {
		if _, err := NewRequirementsInput([]RequirementsSource{{Name: "root.txt"}}, files, nil); err == nil {
			t.Fatal("incomplete or unsafe closure accepted")
		}
	}
	if _, err := NewRequirementsInput(nil, nil, nil); err == nil {
		t.Fatal("absent request accepted")
	}
	if _, err := NewRequirementsInput([]RequirementsSource{{Name: "empty.txt"}}, map[string][]byte{"empty.txt": nil}, nil); err != nil {
		t.Fatal("explicit empty source refused", err)
	}
}

func TestRequirementsInputChecksDepthAcrossSharedIncludes(t *testing.T) {
	files := map[string][]byte{"shared.txt": []byte("-r leaf.txt"), "leaf.txt": []byte("packaging")}
	for i := 1; i <= maximumRequirementsIncludeDepth-1; i++ {
		next := "shared.txt"
		if i < maximumRequirementsIncludeDepth-1 {
			next = fmt.Sprintf("file%d.txt", i+1)
		}
		files[fmt.Sprintf("file%d.txt", i)] = []byte("-r " + next)
	}
	// The shared suffix was visited first at shallow depth. Its memoized
	// traversal must not conceal an over-limit second route.
	if _, err := NewRequirementsInput([]RequirementsSource{{Name: "shared.txt"}, {Name: "file1.txt"}}, files, nil); err == nil {
		t.Fatal("shared include concealed excessive depth")
	}
}

func TestRequirementsInputRejectsNonUTF8SourceIdentity(t *testing.T) {
	name := string([]byte{0xff})
	if _, err := NewRequirementsInput([]RequirementsSource{{Name: name}}, map[string][]byte{name: []byte("alpha")}, nil); err == nil {
		t.Fatal("JSON would replace source identity")
	}
}

func TestRequirementsParserControlOmitsCommentsAndRejectsCredentialNamespaces(t *testing.T) {
	body := []byte("# synthetic private comment\nalpha==1 # second synthetic comment\n")
	input, err := NewRequirementsInput([]RequirementsSource{{Name: "requirements.txt"}}, map[string][]byte{"requirements.txt": body}, nil)
	if err != nil {
		t.Fatal(err)
	}
	control, err := input.ParserControl()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(control.Body(), []byte("synthetic")) || bytes.Contains(control.Body(), []byte("comment")) {
		t.Fatal("unneeded source comments entered parser payload")
	}
	for _, name := range []string{".env", "nested/.aws/credentials", "id_ed25519", "secrets.txt", "production.env"} {
		if _, err := NewRequirementsInput([]RequirementsSource{{Name: name}}, map[string][]byte{name: []byte("alpha")}, nil); err == nil {
			t.Fatal("credential namespace accepted", name)
		}
	}
}

func TestRequirementsInstallationPreservesLiteralDistInfoIdentity(t *testing.T) {
	source, err := NewRequirementsInput(nil, nil, []string{"other"})
	if err != nil {
		t.Fatal(err)
	}
	metadata := []byte("Metadata-Version: 2.1\nName: PySocks\nVersion: 1.7.1\n\n")
	for _, client := range []string{"pip", "uv"} {
		t.Run(client, func(t *testing.T) {
			for _, directory := range []string{"PySocks-1.7.1.dist-info", "pysocks-1.7.1.dist-info"} {
				bound, err := source.WithInstallation(client, false, []InstalledMetadataFile{{directory, metadata}})
				if err != nil {
					t.Fatal("literal matching identity rejected", directory, err)
				}
				_, _, files, err := bound.Installation()
				if err != nil || len(files) != 1 || files[0].Directory != directory {
					t.Fatal("literal path was rewritten", err)
				}
			}
			for _, files := range [][]InstalledMetadataFile{
				{{"../PySocks-1.7.1.dist-info", metadata}}, {{"PySocks-1.7.2.dist-info", metadata}}, {{"other-1.7.1.dist-info", metadata}}, {{"PySocks-1.7.1.dist-info", metadata}, {"pysocks-1.7.1.dist-info", metadata}},
			} {
				if _, err := source.WithInstallation(client, false, files); err == nil {
					t.Fatal("foreign or duplicate metadata identity accepted")
				}
			}
		})
	}
}

func TestRequirementsOriginalMetadataRetainsGeneralMarkerSemantics(t *testing.T) {
	source, err := NewRequirementsInput(nil, nil, []string{"other"})
	if err != nil {
		t.Fatal(err)
	}
	for _, requirement := range []string{`child>=1 ; os_name == "posix"`, `child>=1 ; implementation_name == "cpython"`, `child[feature]>=1 ; python_full_version >= "3.14.0"`} {
		files := []InstalledMetadataFile{{"retained-1.0.dist-info", []byte("Metadata-Version: 2.1\nName: retained\nVersion: 1.0\nRequires-Dist: " + requirement + "\n\n")}}
		if _, err := source.WithInstallation("pip", false, files); err != nil {
			t.Fatal("general requirements cannot survive the next installation", requirement, err)
		}
	}
}
