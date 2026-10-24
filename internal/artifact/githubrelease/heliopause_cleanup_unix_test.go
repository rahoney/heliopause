//go:build linux || darwin

package githubrelease

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestHeliopauseReplayDiscardRetainsUnknownAndUnrelatedEvidence(t *testing.T) {
	for _, kind := range []string{"success", "extra-file", "replaced-directory", "symlink", "hardlink", "changed-mode", "changed-content", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			f := newReleaseSetFixture(t, nil)
			resolution, err := f.set.ResolveDependencies(context.Background(), f.set.Reference(), f.install)
			if err != nil {
				t.Fatal(err)
			}
			runs := make([]domain.RunID, 0, 2)
			for _, node := range resolution.Graph().Nodes()[:2] {
				run, _ := domain.NewRunID()
				if _, err := f.set.Acquire(context.Background(), run, node.Artifact()); err != nil {
					t.Fatal(err)
				}
				runs = append(runs, run)
			}
			foreign := filepath.Join(f.client.intakeRoot, "independent-evidence.json")
			if err := os.WriteFile(foreign, []byte("retained evidence"), 0600); err != nil {
				t.Fatal(err)
			}
			directory := filepath.Join(f.client.intakeRoot, runs[0].String())
			asset := filepath.Join(directory, "asset")
			switch kind {
			case "extra-file":
				if err := os.WriteFile(filepath.Join(directory, "keep"), []byte("unknown"), 0600); err != nil {
					t.Fatal(err)
				}
			case "replaced-directory":
				if err := os.Rename(directory, directory+".retained"); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(directory, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(directory, "keep"), []byte("unknown"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(asset, asset+".retained"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(asset+".retained", asset); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(asset, filepath.Join(f.client.intakeRoot, "retained-link")); err != nil {
					t.Fatal(err)
				}
			case "changed-mode":
				if err := os.Chmod(asset, 0400); err != nil {
					t.Fatal(err)
				}
			case "changed-content":
				if err := os.WriteFile(asset, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx := context.Background()
			if kind == "canceled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err = f.set.DiscardReplayRuns(ctx)
			if (err == nil) != (kind == "success") {
				t.Fatal(kind, err)
			}
			if kind == "canceled" {
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if _, err := os.Stat(filepath.Join(f.client.intakeRoot, runs[1].String(), "asset")); err != nil {
					t.Fatal("canceled cleanup removed payload", err)
				}
				if err := f.set.DiscardReplayRuns(context.Background()); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := os.Lstat(filepath.Join(f.client.intakeRoot, runs[1].String())); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("independent safe cleanup skipped", err)
				}
				if kind != "success" {
					if _, err := os.Lstat(directory); err != nil {
						t.Fatal("unknown directory deleted", err)
					}
				}
			}
			if kind == "success" || kind == "canceled" {
				if err := f.set.DiscardReplayRuns(context.Background()); err != nil {
					t.Fatal("repeat discard failed", err)
				}
			}
			if err := f.set.Close(); err != nil {
				t.Fatal(err)
			}
			if body, err := os.ReadFile(foreign); err != nil || string(body) != "retained evidence" {
				t.Fatal("unrelated evidence changed", err)
			}
		})
	}
}
