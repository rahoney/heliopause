//go:build linux && realcorpus

package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// The operator supplies the actual verified bytes at the fixed path inside
// disposable mount/network namespaces. No Host package manager is executed.
func TestPNPMProtectedSourceInDisposableNamespace(t *testing.T) {
	if os.Getenv("HELOX_PNPM_SOURCE_NAMESPACE") != "1" {
		t.Skip("explicit protected PNPM source qualification")
	}
	if os.Geteuid() != 0 {
		t.Fatal("requires Root in a disposable namespace")
	}
	for _, kind := range []string{"mnt", "net"} {
		self, err := os.Readlink("/proc/self/ns/" + kind)
		host, other := os.Readlink("/proc/1/ns/" + kind)
		if err != nil || other != nil || self == host {
			t.Fatal("refusing source mutation outside disposable namespaces")
		}
	}
	tool, err := preparePNPMTool(context.Background())
	if err != nil {
		t.Fatal("fixed installed source unavailable", err)
	}
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.Chmod(root, 0755))
	versions := filepath.Join(root, "versions")
	version := filepath.Join(versions, "v1.0.0-probe")
	name := filepath.Join(version, runtimeidentity.PNPMNativeAssetName)
	current := filepath.Join(root, "current")
	must(os.Mkdir(versions, 0755))
	must(os.Mkdir(version, 0755))
	must(os.WriteFile(name, tool.body, 0555))
	must(os.Symlink("versions/v1.0.0-probe", current))
	if got, err := readInstalledPNPMTool(context.Background(), root); err != nil || got == nil {
		t.Fatal("exact Root fixture refused", err)
	}
	tests := []struct {
		name   string
		mutate func(func(error)) func()
	}{
		{"writable-file", func(must func(error)) func() {
			must(os.Chmod(name, 0755))
			return func() { must(os.Chmod(name, 0555)) }
		}},
		{"foreign-file-owner", func(must func(error)) func() {
			must(os.Chown(name, 1000, 1000))
			return func() { must(os.Chown(name, 0, 0)) }
		}},
		{"hardlink", func(must func(error)) func() {
			link := filepath.Join(version, "other")
			must(os.Link(name, link))
			return func() { must(os.Remove(link)) }
		}},
		{"file-symlink", func(must func(error)) func() {
			saved := name + ".saved"
			must(os.Rename(name, saved))
			must(os.Symlink(filepath.Base(saved), name))
			return func() { must(os.Remove(name)); must(os.Rename(saved, name)) }
		}},
		{"changed-byte", func(must func(error)) func() {
			f, err := os.OpenFile(name, os.O_WRONLY, 0)
			must(err)
			_, err = f.WriteAt([]byte{tool.body[0] ^ 1}, 0)
			must(err)
			must(f.Close())
			return func() {
				f, err := os.OpenFile(name, os.O_WRONLY, 0)
				must(err)
				_, err = f.WriteAt(tool.body[:1], 0)
				must(err)
				must(f.Close())
			}
		}},
		{"wrong-size", func(must func(error)) func() {
			must(os.Truncate(name, runtimeidentity.PNPMToolSize-1))
			return func() {
				f, err := os.OpenFile(name, os.O_WRONLY, 0)
				must(err)
				_, err = f.WriteAt(tool.body[len(tool.body)-1:], runtimeidentity.PNPMToolSize-1)
				must(err)
				must(f.Close())
			}
		}},
		{"foreign-current", func(must func(error)) func() {
			must(os.Lchown(current, 1000, 1000))
			return func() { must(os.Lchown(current, 0, 0)) }
		}},
		{"escaping-current", func(must func(error)) func() {
			must(os.Remove(current))
			must(os.Symlink("../outside", current))
			return func() { must(os.Remove(current)); must(os.Symlink("versions/v1.0.0-probe", current)) }
		}},
		{"mutable-version", func(must func(error)) func() {
			must(os.Chmod(version, 0775))
			return func() { must(os.Chmod(version, 0755)) }
		}},
		{"version-symlink", func(must func(error)) func() {
			saved := version + ".saved"
			must(os.Rename(version, saved))
			must(os.Symlink(filepath.Base(saved), version))
			return func() { must(os.Remove(version)); must(os.Rename(saved, version)) }
		}},
		{"mutable-root", func(must func(error)) func() {
			must(os.Chmod(root, 0777))
			return func() { must(os.Chmod(root, 0755)) }
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			restore := test.mutate(func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			})
			defer restore()
			if got, err := readInstalledPNPMTool(context.Background(), root); err == nil || got != nil {
				t.Fatal("untrusted source produced native bytes")
			}
		})
	}
	if got, err := readInstalledPNPMTool(context.Background(), root); err != nil || got == nil {
		t.Fatal("restored exact source refused", err)
	}
}
