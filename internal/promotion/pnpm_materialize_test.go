package promotion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPNPMPreparedProjectCopiesApprovedBytesAndKeepsTargetAbsent(t *testing.T) {
	for _, empty := range []bool{false, true} {
		f := newPNPMCacheFixture(t, empty)
		ctx := context.Background()
		staged, err := f.cache.StageProject(ctx, f.set)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := f.cache.PrepareProject(ctx, staged)
		if err != nil {
			t.Fatal(err)
		}
		name := prepared.Directory()
		for _, control := range f.cache.frozen.Controls() {
			body, err := os.ReadFile(filepath.Join(name, control.Name()))
			if err != nil || !bytes.Equal(body, control.Body()) {
				t.Fatal("prepared controls changed", err)
			}
		}
		if !empty {
			link, err := os.Readlink(filepath.Join(name, "node_modules/is-number"))
			if err != nil || link != ".pnpm/is-number@6.0.0/node_modules/is-number" {
				t.Fatal("internal native link changed", err)
			}
			body, err := os.ReadFile(filepath.Join(name, "node_modules/is-number/package.json"))
			if err != nil || !bytes.Contains(body, []byte("is-number")) {
				t.Fatal("owned internal link is unusable", err)
			}
		}
		if err := prepared.Verify(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(f.cache.frozen.Snapshot().Context().Target().String()); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("private preparation published target", err)
		}
		if err := prepared.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(name); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("owned preparation not removed", err)
		}
		if prepared.Directory() != "" || prepared.Verify(ctx) == nil {
			t.Fatal("closed preparation reused")
		}
	}
}

func TestPNPMNativeHardlinkAliasesBecomeIndependentHostFiles(t *testing.T) {
	f := newPNPMCacheFixtureWithAliases(t, false, true)
	ctx := context.Background()
	staged, err := f.cache.StageProject(ctx, f.set)
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.cache.PrepareProject(ctx, staged)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(p.Directory(), "node_modules/.pnpm/is-number@6.0.0/node_modules/is-number")
	a, err := os.Lstat(filepath.Join(root, "LICENSE"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.Lstat(filepath.Join(root, "COPY"))
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(a, b) || !pypiSingleLink(a) || !pypiSingleLink(b) {
		t.Fatal("native alias remained a shared inode")
	}
	for _, name := range []string{"LICENSE", "COPY"} {
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(body) != "controlled shared license\n" {
			t.Fatal("native alias bytes changed", err)
		}
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPNPMPreparedCleanupRechecksBytesAfterInitialInventory(t *testing.T) {
	f := newPNPMCacheFixture(t, false)
	ctx := context.Background()
	staged, err := f.cache.StageProject(ctx, f.set)
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.cache.PrepareProject(ctx, staged)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(p.Directory(), "node_modules/.pnpm/is-number@6.0.0/node_modules/is-number/package.json")
	p.beforeRemove = func() error { return os.WriteFile(file, []byte("changed after initial inventory"), 0o644) }
	if err := p.Close(); err == nil {
		t.Fatal("cleanup deleted a member changed after its initial inventory")
	}
	if body, err := os.ReadFile(file); err != nil || string(body) != "changed after initial inventory" {
		t.Fatal("changed cleanup member was not preserved", err)
	}
}

func TestPNPMPreparedCleanupPreservesChangedParentAfterInitialInventory(t *testing.T) {
	f := newPNPMCacheFixture(t, false)
	ctx := context.Background()
	staged, err := f.cache.StageProject(ctx, f.set)
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.cache.PrepareProject(ctx, staged)
	if err != nil {
		t.Fatal(err)
	}
	packageRoot := filepath.Join(p.Directory(), "node_modules/.pnpm/is-number@6.0.0/node_modules/is-number")
	p.beforeRemove = func() error {
		if err := os.Rename(packageRoot, packageRoot+"-moved"); err != nil {
			return err
		}
		return os.Symlink("is-number-moved", packageRoot)
	}
	if err := p.Close(); err == nil {
		t.Fatal("changed cleanup parent was accepted")
	}
	if _, err := os.ReadFile(filepath.Join(packageRoot+"-moved", "package.json")); err != nil {
		t.Fatal("cleanup traversed changed parent and deleted original bytes", err)
	}
}

func TestPNPMPreparedProjectRejectsDriftAndPreservesUncertainTree(t *testing.T) {
	for _, mutation := range []string{"bytes", "mode", "symlink", "symlink-hardlink", "hardlink", "foreign", "missing", "root"} {
		t.Run(mutation, func(t *testing.T) {
			f := newPNPMCacheFixture(t, false)
			ctx := context.Background()
			staged, err := f.cache.StageProject(ctx, f.set)
			if err != nil {
				t.Fatal(err)
			}
			p, err := f.cache.PrepareProject(ctx, staged)
			if err != nil {
				t.Fatal(err)
			}
			name := p.Directory()
			file := filepath.Join(name, "node_modules/.pnpm/is-number@6.0.0/node_modules/is-number/package.json")
			switch mutation {
			case "bytes":
				err = os.WriteFile(file, []byte("changed"), 0o644)
			case "mode":
				err = os.Chmod(file, 0o600)
			case "symlink":
				link := filepath.Join(name, "node_modules/is-number")
				if err = os.Remove(link); err == nil {
					err = os.Symlink("../../outside", link)
				}
			case "symlink-hardlink":
				// Root.Link uses linkat flags0 on both Linux and Darwin.
				err = p.parent.Link(filepath.Join(p.name, "node_modules/is-number"), "outside-link")
			case "hardlink":
				err = os.Link(file, filepath.Join(filepath.Dir(name), "outside-link"))
			case "foreign":
				err = os.WriteFile(filepath.Join(name, "user-file"), []byte("preserve"), 0o600)
			case "missing":
				err = os.Remove(file)
			case "root":
				if err = os.Rename(name, name+"-original"); err == nil {
					err = os.Mkdir(name, 0o700)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if p.Verify(ctx) == nil {
				t.Fatal("prepared drift admitted", mutation)
			}
			if p.Close() == nil {
				t.Fatal("uncertain preparation deleted", mutation)
			}
			if _, err := os.Lstat(name); err != nil {
				t.Fatal("uncertain root not preserved", err)
			}
			if mutation == "foreign" {
				body, err := os.ReadFile(filepath.Join(name, "user-file"))
				if err != nil || string(body) != "preserve" {
					t.Fatal("unknown user member deleted", err)
				}
			}
		})
	}
}

func TestPNPMPreparationFailureKeepsCauseAndCleansOnlyOwnedData(t *testing.T) {
	for _, mutation := range []string{"cache", "late", "cancel", "foreign"} {
		t.Run(mutation, func(t *testing.T) {
			f := newPNPMCacheFixture(t, false)
			ctx, stop := context.WithCancelCause(context.Background())
			defer stop(nil)
			staged, err := f.cache.StageProject(ctx, f.set)
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("controlled prepare failure")
			name := ""
			if mutation == "cache" {
				data, err := f.cache.OpenProjectCache(ctx, staged)
				if err != nil {
					t.Fatal(err)
				}
				file := filepath.Join(data, pnpmCacheArchiveName)
				if err := os.Chmod(file, 0o644); err != nil {
					t.Fatal(err)
				}
			} else {
				f.cache.beforePrepareSync = func(p *PNPMPreparedProject) error {
					name = p.Directory()
					if mutation == "cancel" {
						stop(cause)
						return nil
					}
					if mutation == "foreign" {
						if err := os.WriteFile(filepath.Join(name, "user-file"), []byte("preserve"), 0o600); err != nil {
							return err
						}
					}
					return cause
				}
			}
			p, err := f.cache.PrepareProject(ctx, staged)
			if err == nil || p != nil {
				t.Fatal("failed prepare returned a project", err)
			}
			if mutation != "cache" && !errors.Is(err, cause) {
				t.Fatal("first prepare cause lost", err)
			}
			if mutation == "foreign" {
				if body, err := os.ReadFile(filepath.Join(name, "user-file")); err != nil || string(body) != "preserve" {
					t.Fatal("late foreign member deleted", err)
				}
			} else if name != "" {
				if _, err := os.Stat(name); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("owned failed preparation retained", err)
				}
			}
			if mutation == "cache" {
				entries, err := os.ReadDir(filepath.Dir(f.cache.frozen.Snapshot().Context().Target().String()))
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), ".haa-pnpm-prepare-") {
						t.Fatal("bad cache wrote a Host workspace")
					}
				}
			}
		})
	}
}
