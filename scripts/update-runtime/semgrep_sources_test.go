package main

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func semgrepSourceFixture(t *testing.T) (string, semgrepSourcePackageInput) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(base, "inputs")
	if err := os.Mkdir(input, 0700); err != nil {
		t.Fatal(err)
	}
	selected := semgrepSourcePackageInput{
		Schema: "helox.semgrep-source-inputs/v1", SourceCommit: strings.Repeat("a", 40), SourceDateEpoch: 1793297858,
	}
	for name, body := range map[string]string{"build.sh": "set -eu\n", "source.ml": "let main = ()\n", "empty": ""} {
		if err := os.WriteFile(filepath.Join(input, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		mode := int64(0444)
		if name == "build.sh" {
			mode = 0555
		}
		selected.Files = append(selected.Files, semgrepSourcePackageFile{name, hash([]byte(body)), int64(len(body)), mode})
	}
	sort.Slice(selected.Files, func(i, j int) bool { return selected.Files[i].Path < selected.Files[j].Path })
	return input, selected
}

func writeSemgrepSourceFixture(t *testing.T, input string, selected semgrepSourcePackageInput) []byte {
	t.Helper()
	body, err := json.Marshal(selected)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(input, "source-inputs.json"), body, 0600); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestSemgrepSourcePackageDeterministicPassiveArchive(t *testing.T) {
	input, selected := semgrepSourceFixture(t)
	manifest := writeSemgrepSourceFixture(t, input, selected)
	var previous []byte
	for attempt := 0; attempt < 2; attempt++ {
		directory, err := os.MkdirTemp(filepath.Dir(input), "output-")
		if err != nil {
			t.Fatal(err)
		}
		output := filepath.Join(directory, "semgrep-SOURCES.tar")
		var receipt bytes.Buffer
		if err := run([]string{"package-semgrep-source", "--input", input, "--output", output}, &receipt); err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		var fact struct {
			SHA256           string `json:"sha256"`
			Size             int64  `json:"size"`
			RuntimeQualified bool   `json:"runtime_qualified"`
		}
		if json.Unmarshal(receipt.Bytes(), &fact) != nil || fact.SHA256 != hash(body) || fact.Size != int64(len(body)) || fact.RuntimeQualified {
			t.Fatal("source package receipt granted admission or lost byte identity")
		}
		info, err := os.Stat(output)
		if err != nil || info.Mode().Perm() != 0444 || attempt > 0 && !bytes.Equal(body, previous) {
			t.Fatal("source package is not passive and deterministic")
		}
		previous = body
		archive := tar.NewReader(bytes.NewReader(body))
		for i := -1; i < len(selected.Files); i++ {
			header, err := archive.Next()
			if err != nil {
				t.Fatal(err)
			}
			name, mode := "SOURCE-MANIFEST.json", int64(0444)
			expected := manifest
			if i >= 0 {
				name, mode = selected.Files[i].Path, selected.Files[i].Mode
				expected, err = os.ReadFile(filepath.Join(input, name))
				if err != nil {
					t.Fatal(err)
				}
			}
			contents, err := io.ReadAll(archive)
			if err != nil || header.Name != name || header.Mode != mode || header.Typeflag != tar.TypeReg || header.Uid != 0 || header.Gid != 0 || header.ModTime.Unix() != selected.SourceDateEpoch || !bytes.Equal(contents, expected) {
				t.Fatal("source member bytes or normalized metadata differ")
			}
		}
		if _, err := archive.Next(); err != io.EOF {
			t.Fatal("source package has an undeclared member")
		}
	}
}

func TestSemgrepSourcePackageRejectsUntrustedChanges(t *testing.T) {
	for _, scenario := range []string{"content", "size", "traversal", "symlink", "duplicate", "unsorted", "unknown-schema", "unknown-field", "duplicate-json-key", "member-budget", "aggregate-budget", "epoch", "existing-output"} {
		t.Run(scenario, func(t *testing.T) {
			input, selected := semgrepSourceFixture(t)
			output := filepath.Join(filepath.Dir(input), "semgrep-SOURCES.tar")
			switch scenario {
			case "content":
				if err := os.WriteFile(filepath.Join(input, "build.sh"), []byte("set +eu\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "size":
				selected.Files[0].Size++
			case "traversal":
				selected.Files[0].Path = "../foreign"
			case "symlink":
				link := filepath.Join(input, "build.sh")
				if err := os.Remove(link); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(input, "source.ml"), link); err != nil {
					t.Fatal(err)
				}
			case "duplicate":
				selected.Files[1] = selected.Files[0]
			case "unsorted":
				selected.Files[0], selected.Files[1] = selected.Files[1], selected.Files[0]
			case "unknown-schema":
				selected.Schema = "foreign"
			case "member-budget":
				selected.Files[0].Size = 256<<20 + 1
			case "aggregate-budget":
				for i := range selected.Files {
					selected.Files[i].Size = 256 << 20
				}
			case "epoch":
				selected.SourceDateEpoch = 1 << 33
			case "existing-output":
				if err := os.WriteFile(output, []byte("foreign"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			body := writeSemgrepSourceFixture(t, input, selected)
			if scenario == "unknown-field" || scenario == "duplicate-json-key" {
				field := `"unreviewed":true,`
				if scenario == "duplicate-json-key" {
					field = `"schema":"foreign",`
				}
				if err := os.WriteFile(filepath.Join(input, "source-inputs.json"), append([]byte("{"+field), body[1:]...), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := packageSemgrepSource([]string{"--input", input, "--output", output}, io.Discard); err == nil {
				t.Fatal("untrusted source input was packaged")
			}
			if scenario == "existing-output" {
				if body, err := os.ReadFile(output); err != nil || string(body) != "foreign" {
					t.Fatal("existing source output was changed")
				}
			} else if _, err := os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed source packaging published an output")
			}
			entries, err := os.ReadDir(filepath.Dir(output))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".native-") {
					t.Fatal("failed source packaging retained its private temporary file")
				}
			}
		})
	}
}

func TestNativeAssetWriterFailureCannotPublish(t *testing.T) {
	first := errors.New("source input read failed")
	for _, scenario := range []string{"error", "short", "long"} {
		t.Run(scenario, func(t *testing.T) {
			directory, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(directory, "source.tar")
			err = publishNativeAssetWith(output, 4, 0444, func(output io.Writer) error {
				body := "abc"
				if scenario == "long" {
					body = "abcde"
				}
				if _, err := io.WriteString(output, body); err != nil {
					return err
				}
				if scenario == "error" {
					return first
				}
				return nil
			})
			if err == nil || scenario == "error" && !errors.Is(err, first) {
				t.Fatal("incomplete writer succeeded or lost the first failure")
			}
			entries, err := os.ReadDir(filepath.Dir(output))
			if err != nil || len(entries) != 0 {
				t.Fatal("incomplete writer left a temporary or published asset")
			}
		})
	}
}

func semgrepNoticeFixture(t *testing.T) (string, semgrepSourcePackageInput, []byte) {
	t.Helper()
	input, selected := semgrepSourceFixture(t)
	if err := os.Mkdir(filepath.Join(input, "notices"), 0700); err != nil {
		t.Fatal(err)
	}
	// Copyright bytes and page separators must survive without charset guessing.
	original := []byte("Copyright \xa9 source authors\n\fOriginal terms\n")
	for name, body := range map[string][]byte{"catalogue.json": []byte("{}\n"), hash(original) + ".txt": original} {
		name = "notices/" + name
		if err := os.WriteFile(filepath.Join(input, name), body, 0600); err != nil {
			t.Fatal(err)
		}
		selected.Files = append(selected.Files, semgrepSourcePackageFile{name, hash(body), int64(len(body)), 0444})
	}
	sort.Slice(selected.Files, func(i, j int) bool { return selected.Files[i].Path < selected.Files[j].Path })
	return input, selected, original
}

func TestSemgrepNoticePackagePreservesOriginalBytes(t *testing.T) {
	input, selected, original := semgrepNoticeFixture(t)
	sourceManifest := writeSemgrepSourceFixture(t, input, selected)
	output := filepath.Join(filepath.Dir(input), "semgrep-NOTICES.tar")
	var receipt bytes.Buffer
	if err := packageSemgrepSource([]string{"--notices", "--input", input, "--output", output}, &receipt); err != nil {
		t.Fatal(err)
	}
	var fact struct {
		Schema               string `json:"schema"`
		SourceManifestSHA256 string `json:"source_manifest_sha256"`
		ManifestSHA256       string `json:"manifest_sha256"`
		RuntimeQualified     bool   `json:"runtime_qualified"`
	}
	if json.Unmarshal(receipt.Bytes(), &fact) != nil || fact.Schema != "helox.semgrep-notice-package/v1" || fact.SourceManifestSHA256 != hash(sourceManifest) || fact.RuntimeQualified {
		t.Fatal("notice package lost its source binding or granted runtime admission")
	}
	body, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	archive := tar.NewReader(bytes.NewReader(body))
	count, found := 0, false
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(archive)
		if err != nil || header.Typeflag != tar.TypeReg || header.Mode != 0444 {
			t.Fatal("notice archive contains a non-passive member")
		}
		if count == 0 {
			var projected semgrepSourcePackageInput
			if header.Name != "NOTICE-MANIFEST.json" || hash(contents) != fact.ManifestSHA256 || json.Unmarshal(contents, &projected) != nil || projected.Schema != "helox.semgrep-notice-inputs/v1" || len(projected.Files) != 2 {
				t.Fatal("notice projection manifest differs")
			}
		} else if header.Name == "notices/"+hash(original)+".txt" {
			found = bytes.Equal(contents, original)
		} else if header.Name != "notices/catalogue.json" {
			t.Fatal("notice projection included unrelated source data")
		}
		count++
	}
	if count != 3 || !found {
		t.Fatal("original notice bytes were lost or transformed")
	}
}

func TestSemgrepNoticePackageRejectsMissingOrExecutableNotices(t *testing.T) {
	for _, scenario := range []string{"catalogue", "text", "name", "mode", "entry-budget", "aggregate-budget"} {
		t.Run(scenario, func(t *testing.T) {
			input, selected, _ := semgrepNoticeFixture(t)
			var files []semgrepSourcePackageFile
			for _, member := range selected.Files {
				if !strings.HasPrefix(member.Path, "notices/") {
					files = append(files, member)
					continue
				}
				catalogue := member.Path == "notices/catalogue.json"
				if scenario == "catalogue" && catalogue || scenario == "text" && !catalogue {
					continue
				}
				if !catalogue {
					switch scenario {
					case "name":
						member.Path = "notices/foreign.txt"
					case "mode":
						member.Mode = 0555
					case "entry-budget":
						body := bytes.Repeat([]byte{'a'}, 1<<20+1)
						member = semgrepSourcePackageFile{"notices/" + hash(body) + ".txt", hash(body), int64(len(body)), 0444}
						if err := os.WriteFile(filepath.Join(input, member.Path), body, 0600); err != nil {
							t.Fatal(err)
						}
					case "aggregate-budget":
						for n := byte(0); n < 4; n++ {
							body := bytes.Repeat([]byte{n}, 1<<20)
							name := "notices/" + hash(body) + ".txt"
							if err := os.WriteFile(filepath.Join(input, name), body, 0600); err != nil {
								t.Fatal(err)
							}
							files = append(files, semgrepSourcePackageFile{name, hash(body), int64(len(body)), 0444})
						}
					}
				}
				files = append(files, member)
			}
			selected.Files = files
			sort.Slice(selected.Files, func(i, j int) bool { return selected.Files[i].Path < selected.Files[j].Path })
			writeSemgrepSourceFixture(t, input, selected)
			output := filepath.Join(filepath.Dir(input), "semgrep-NOTICES.tar")
			if err := packageSemgrepSource([]string{"--notices", "--input", input, "--output", output}, io.Discard); err == nil {
				t.Fatal("invalid notice package succeeded")
			}
			if _, err := os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("invalid notice package published output")
			}
		})
	}
}
