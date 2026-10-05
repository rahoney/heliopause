package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
)

func cargoSourceFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range map[string]string{
		"Cargo.toml":     "[package]\nname = 'local'\nversion = '0.1.0'\nedition = '2024'\n",
		"src/main.rs":    "fn main() {}\n",
		"data/input.txt": "data only\n",
	} {
		filename := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCargoProjectSourceAnchoredArchive(t *testing.T) {
	root := cargoSourceFixture(t)
	for _, name := range []string{".git/config", "target/old-binary", ".haa/approval.json"} {
		filename := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte("excluded"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s, err := captureCargoProject(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.close(); err != nil {
			t.Error(err)
		}
	}()
	if err := s.verify(context.Background()); err != nil {
		t.Fatal(err)
	}
	body, err := s.archive()
	if err != nil {
		t.Fatal(err)
	}
	reader := tar.NewReader(bytes.NewReader(body))
	got := map[string]string{}
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Typeflag == tar.TypeDir {
			if header.Mode != 0o700 {
				t.Fatal("directory authority not reconstructed")
			}
			continue
		}
		if header.Typeflag != tar.TypeReg || header.Mode != 0o600 || header.Uid != 0 || header.Gid != 0 || header.Linkname != "" {
			t.Fatal("file authority not reconstructed")
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		got[header.Name] = string(data)
	}
	want := map[string]string{"Cargo.toml": "[package]\nname = 'local'\nversion = '0.1.0'\nedition = '2024'\n", "src/main.rs": "fn main() {}\n", "data/input.txt": "data only\n"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("consumed source = %#v", got)
	}
	files := s.files()
	files["src/main.rs"][0] = '!'
	if bytes.Equal(s.files()["src/main.rs"], files["src/main.rs"]) {
		t.Fatal("mutable exported source aliases frozen bytes")
	}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.archive(); err == nil {
		t.Fatal("closed snapshot returned input")
	}
	if err := s.verify(context.Background()); err == nil {
		t.Fatal("closed snapshot verified")
	}
}

func TestCargoProjectSourceRejectsUntrustedMembers(t *testing.T) {
	for _, name := range []string{"symbolic link", "hard link", "configuration", "nested git source", "nested outside path", "noncanonical name", "oversized file"} {
		t.Run(name, func(t *testing.T) {
			root := cargoSourceFixture(t)
			var err error
			switch name {
			case "symbolic link":
				err = os.Symlink("src/main.rs", filepath.Join(root, "alias.rs"))
			case "hard link":
				err = os.Link(filepath.Join(root, "src/main.rs"), filepath.Join(root, "alias.rs"))
			case "configuration":
				err = os.Mkdir(filepath.Join(root, ".cargo"), 0o700)
			case "nested git source", "nested outside path":
				if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname='local'\nversion='0.1.0'\n[dependencies]\nmember={path='member'}\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(root, "member"), 0o700); err != nil {
					t.Fatal(err)
				}
				dependency := "git = 'https://example.invalid/repository'"
				if name == "nested outside path" {
					dependency = "path = '../../outside'"
				}
				err = os.WriteFile(filepath.Join(root, "member/Cargo.toml"), []byte("[package]\nname='member'\nversion='0.1.0'\n[dependencies]\nother={"+dependency+"}\n"), 0o600)
			case "noncanonical name":
				err = os.WriteFile(filepath.Join(root, "bad:name"), nil, 0o600)
			case "oversized file":
				var file *os.File
				file, err = os.Create(filepath.Join(root, "large"))
				if err == nil {
					err = file.Truncate(64<<20 + 1)
					err = errors.Join(err, file.Close())
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if s, err := captureCargoProject(context.Background(), root); err == nil || s != nil {
				t.Fatal("untrusted project input accepted")
			}
		})
	}
}

func TestCargoSourcePreservesUnselectedManifests(t *testing.T) {
	for _, size := range []int{32, 5 << 20} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			root := cargoSourceFixture(t)
			name := "tests/fixtures/invalid/Cargo.toml"
			body := bytes.Repeat([]byte("["), size)
			if err := os.MkdirAll(filepath.Join(root, path.Dir(name)), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, name), body, 0o600); err != nil {
				t.Fatal(err)
			}
			source, err := captureCargoProject(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := source.close(); err != nil {
					t.Error(err)
				}
			}()
			archive, err := source.archive()
			if err != nil {
				t.Fatal(err)
			}
			reader := tar.NewReader(bytes.NewReader(archive))
			found := false
			for {
				header, err := reader.Next()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				if header.Name == name {
					data, err := io.ReadAll(reader)
					if err != nil || !bytes.Equal(data, body) {
						t.Fatal("unselected source data changed")
					}
					found = true
				}
			}
			if !found {
				t.Fatal("unselected manifest was deleted")
			}
			if err := os.WriteFile(filepath.Join(root, name), []byte("changed"), 0o600); err != nil {
				t.Fatal(err)
			}
			if source.verify(context.Background()) == nil {
				t.Fatal("data mutation was approved")
			}
		})
	}
}

func TestCargoProjectSourceRejectsDrift(t *testing.T) {
	for _, kind := range []string{"content", "file replacement", "file mode", "directory replacement", "root replacement", "addition", "removal", "configuration"} {
		t.Run(kind, func(t *testing.T) {
			root := cargoSourceFixture(t)
			s, err := captureCargoProject(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := s.close(); err != nil {
					t.Error(err)
				}
			}()
			filename := filepath.Join(root, "src/main.rs")
			switch kind {
			case "content":
				err = os.WriteFile(filename, []byte("fn main() { panic!(); }\n"), 0o600)
			case "file replacement":
				err = os.Rename(filename, filename+".original")
				if err == nil {
					err = os.WriteFile(filename, []byte("fn main() {}\n"), 0o600)
				}
			case "file mode":
				err = os.Chmod(filename, 0o400)
			case "directory replacement":
				err = os.Rename(filepath.Join(root, "src"), filepath.Join(root, "old-src"))
				if err == nil {
					err = os.Mkdir(filepath.Join(root, "src"), 0o700)
				}
				if err == nil {
					err = os.WriteFile(filename, []byte("fn main() {}\n"), 0o600)
				}
			case "root replacement":
				err = os.Rename(root, root+".original")
				t.Cleanup(func() {
					if err := os.RemoveAll(root + ".original"); err != nil {
						t.Error(err)
					}
				})
				if err == nil {
					err = os.Mkdir(root, 0o700)
				}
				if err == nil {
					err = os.WriteFile(filepath.Join(root, "Cargo.toml"), s.files()["Cargo.toml"], 0o600)
				}
			case "addition":
				err = os.WriteFile(filepath.Join(root, "new.rs"), nil, 0o600)
			case "removal":
				err = os.Remove(filename)
			case "configuration":
				err = os.Mkdir(filepath.Join(root, ".cargo"), 0o700)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := s.verify(context.Background()); err == nil {
				t.Fatal("changed original project verified")
			}
		})
	}
}

func TestCargoProjectSourceCancelledAndRootLink(t *testing.T) {
	root := cargoSourceFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if s, err := captureCargoProject(ctx, root); !errors.Is(err, context.Canceled) || s != nil {
		t.Fatal("cancelled capture produced source")
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	if s, err := captureCargoProject(context.Background(), alias); err == nil || s != nil {
		t.Fatal("root alias accepted")
	}
}
