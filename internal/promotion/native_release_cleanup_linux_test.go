//go:build linux

package promotion

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/releaseactivation"
	"github.com/rahoney/heliopause/internal/releaseinstall"
)

func TestNativePromotionCleanupKeepsRecordsAndRetainsUnknownPayload(t *testing.T) {
	for _, kind := range []string{"success", "installer-failed", "extra-file", "replaced-file", "hardlink", "replaced-directory"} {
		t.Run(kind, func(t *testing.T) {
			f := newNativePromotionFixture(t, "success")
			primary := errors.New("controlled installer failure")
			p := &NativeReleasePromotion{stagingRoot: filepath.Join(f.root, "staging"), proof: func(domain.AcquiredArtifact) (releaseinstall.ReleaseManifest, [][]byte, error) {
				return f.manifest, [][]byte{[]byte("fixture-only")}, nil
			}, activate: func(context.Context, []byte, [][]byte, func(string) (io.ReadCloser, error)) error {
				if kind == "installer-failed" {
					return primary
				}
				return nil
			}}
			_, err := p.Promote(context.Background(), f.staged, f.bundle, f.install)
			if kind == "installer-failed" {
				if !errors.Is(err, primary) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			root := filepath.Join(f.root, "staging", f.bundle.ManifestID().String())
			dir := filepath.Join(root, "artifacts")
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 11 {
				t.Fatal(entries, err)
			}
			asset := filepath.Join(dir, entries[0].Name())
			switch kind {
			case "extra-file":
				if err := os.WriteFile(filepath.Join(dir, "keep"), []byte("unknown"), 0400); err != nil {
					t.Fatal(err)
				}
			case "replaced-file":
				if err := os.Rename(asset, asset+".retained"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(asset, []byte("unknown"), 0400); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(asset, filepath.Join(f.root, "keep-link")); err != nil {
					t.Fatal(err)
				}
			case "replaced-directory":
				if err := os.Rename(dir, dir+".retained"); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "keep"), []byte("unknown"), 0400); err != nil {
					t.Fatal(err)
				}
			}
			err = p.Close()
			positive := kind == "success" || kind == "installer-failed"
			if (err == nil) != positive || !positive && !errors.Is(err, releaseactivation.ErrCleanup) {
				t.Fatal(kind, err)
			}
			if p.Close() != err {
				t.Fatal("first cleanup result lost")
			}
			if positive {
				if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("completed staging payload retained", err)
				}
			} else {
				if _, err := os.Lstat(dir); err != nil {
					t.Fatal("unknown directory removed", err)
				}
				if kind != "replaced-directory" {
					if body, err := os.ReadDir(dir); err != nil || len(body) < 9 {
						t.Fatal("partial deletion before rejecting unknown", body, err)
					}
				}
			}
			for name, want := range map[string][]byte{manifestFilename: f.bundle.ManifestDocument(), sbomFilename: f.bundle.SBOMDocument()} {
				body, err := os.ReadFile(filepath.Join(root, name))
				if err != nil || string(body) != string(want) {
					t.Fatal("final report record changed", name, err)
				}
			}
		})
	}
}
