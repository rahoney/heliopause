package promotion

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func makePypiVenvFixture(t *testing.T) (string, pypiVenvPlan) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("production venv transaction is Linux amd64 only")
	}
	root := realPromotionRoot(t)
	if err := os.WriteFile(filepath.Join(root, "pyvenv.cfg"), []byte("version = 3.14.7\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	site := filepath.Join(root, "lib", "python3.14", "site-packages")
	if err := os.MkdirAll(site, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	plan, err := discoverPythonVenv(root)
	if err != nil {
		t.Fatal(err)
	}
	return root, plan
}

// Legacy unit fixtures exercise the transaction through test-only adapters.
// Production can construct destinations only through validated RECORD output.
func (p pypiVenvPlan) outputState(output string) ([]pypiDestination, error) {
	result := []pypiDestination{}
	err := filepath.WalkDir(output, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		snapshot, err := snapshotPyPIFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(output, path)
		if err != nil {
			return err
		}
		result = append(result, pypiDestination{Scheme: "site", Relative: filepath.ToSlash(relative), Source: path, Distribution: "fixture", Version: "1.0", Digest: snapshot.Digest, Size: snapshot.Size})
		return nil
	})
	return result, err
}

func (p pypiVenvPlan) commit(_ string, desired []pypiDestination) (resultErr error) {
	tx, err := beginPyPIVenvTransaction(p)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, tx.close()) }()
	return tx.commit(desired)
}

func venvDestinations(t *testing.T, files map[string]string, distribution string) []pypiDestination {
	t.Helper()
	output := realPromotionRoot(t)
	result := []pypiDestination{}
	for key, body := range files {
		scheme, relative, ok := strings.Cut(key, "/")
		if !ok {
			t.Fatal(key)
		}
		source := filepath.Join(output, key)
		if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		s, err := snapshotPyPIFile(source)
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, pypiDestination{Scheme: scheme, Relative: relative, Source: source, Distribution: distribution, Version: "1.0", Digest: s.Digest, Size: s.Size})
	}
	return result
}

func runVenvFixture(t *testing.T, p pypiVenvPlan, files map[string]string, owner string) {
	t.Helper()
	if err := p.commit("", venvDestinations(t, files, owner)); err != nil {
		t.Fatal(err)
	}
}

func TestVenvMultiRootPositiveAndCumulativeOwnership(t *testing.T) {
	for _, extra := range []map[string]string{{}, {"scripts/isympy": "script"}, {"data/man/man1/isympy.1": "man"}, {"scripts/isympy": "script", "data/man/man1/isympy.1": "man"}} {
		t.Run(strings.Join(sortedVenvTestKeys(extra), ","), func(t *testing.T) {
			root, p := makePypiVenvFixture(t)
			for _, relative := range []string{"bin/python", "bin/pip", "bin/activate", "share/unrelated"} {
				path := filepath.Join(root, relative)
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("baseline"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			files := map[string]string{"site/sympy/module.py": "module"}
			for key, value := range extra {
				files[key] = value
			}
			runVenvFixture(t, p, files, "sympy")
			runVenvFixture(t, p, map[string]string{"site/other.py": "other"}, "other")
			state, _, err := p.readState()
			if err != nil || len(state.Files) != len(files)+1 {
				t.Fatalf("cumulative state: %#v %v", state, err)
			}
			if err := p.verifyState(state); err != nil {
				t.Fatal(err)
			}
			for _, relative := range []string{"bin/python", "bin/pip", "bin/activate", "share/unrelated"} {
				body, err := os.ReadFile(filepath.Join(root, relative))
				if err != nil || string(body) != "baseline" {
					t.Fatalf("baseline changed: %s", relative)
				}
			}
		})
	}
}

func sortedVenvTestKeys(files map[string]string) []string {
	keys := []string{}
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestVenvManagedReplacementRemovesObsoleteOutputs(t *testing.T) {
	_, p := makePypiVenvFixture(t)
	runVenvFixture(t, p, map[string]string{"site/old.py": "old", "scripts/old": "old script", "data/old": "old data"}, "demo")
	runVenvFixture(t, p, map[string]string{"site/old.py": "replacement", "scripts/new": "new"}, "demo")
	for _, relative := range []string{"bin/old", "share/old"} {
		if _, err := os.Lstat(filepath.Join(p.root, relative)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("obsolete output remains: %s", relative)
		}
	}
	state, _, err := p.readState()
	if err != nil || len(state.Files) != 2 {
		t.Fatalf("replacement metadata: %v %v", state, err)
	}
}

func TestVenvRollbackPhases(t *testing.T) {
	for _, phase := range []string{"before-mutation", "after-publish:site", "after-publish:scripts", "after-publish:data", "metadata", "sync", "reconcile"} {
		t.Run(phase, func(t *testing.T) {
			_, p := makePypiVenvFixture(t)
			runVenvFixture(t, p, map[string]string{"site/old.py": "original", "scripts/old": "original script"}, "demo")
			oldMeta, err := os.ReadFile(filepath.Join(p.root, pypiVenvMetadata))
			if err != nil {
				t.Fatal(err)
			}
			tx, err := beginPyPIVenvTransaction(p)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.close()
			tx.fault = func(step string) error {
				if step == phase {
					return errors.New("injected failure")
				}
				return nil
			}
			ds := venvDestinations(t, map[string]string{"site/old.py": "replacement", "site/new/deep.py": "new", "scripts/new": "new script", "data/man/man1/new.1": "new man"}, "demo")
			if err := tx.commit(ds); err == nil {
				t.Fatal("fault returned success")
			}
			if tx.recovery {
				t.Fatal("ordinary injected failure did not roll back")
			}
			body, err := os.ReadFile(filepath.Join(p.site, "old.py"))
			if err != nil || string(body) != "original" {
				t.Fatal("original not restored")
			}
			meta, err := os.ReadFile(filepath.Join(p.root, pypiVenvMetadata))
			if err != nil || !bytes.Equal(meta, oldMeta) {
				t.Fatal("metadata not restored")
			}
			for _, path := range []string{filepath.Join(p.site, "new"), filepath.Join(p.root, "bin", "new"), filepath.Join(p.root, "share")} {
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("partial state remains: %s", path)
				}
			}
		})
	}
}

func TestVenvRollbackFailureRetainsRecovery(t *testing.T) {
	_, p := makePypiVenvFixture(t)
	tx, err := beginPyPIVenvTransaction(p)
	if err != nil {
		t.Fatal(err)
	}
	tx.fault = func(step string) error {
		if step == "after-publish:data" || step == "rollback" {
			return errors.New("injected")
		}
		return nil
	}
	if err := tx.commit(venvDestinations(t, map[string]string{"site/new.py": "new", "data/man/man1/new.1": "man"}, "demo")); err == nil || !tx.recovery {
		t.Fatal("rollback uncertainty reported clean")
	}
	if _, err := os.Stat(filepath.Join(tx.backup, "ledger.json")); err != nil {
		t.Fatal("recovery ledger lost")
	}
	body, err := os.ReadFile(filepath.Join(tx.backup, "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	created := 0
	for _, line := range bytes.Split(bytes.TrimSpace(body), []byte("\n")) {
		var entry struct {
			Created, Scheme    string
			Order              int
			Inode, ParentInode uint64
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatal(err)
		}
		if entry.Created == "" {
			continue
		}
		info, err := os.Lstat(entry.Created)
		if err != nil {
			t.Fatal(err)
		}
		_, inode, ok := pypiObjectIdentity(info)
		if !ok || entry.Inode != inode || entry.ParentInode == 0 || entry.Scheme != "data" || entry.Order != created {
			t.Fatal("created-directory identity/order lost")
		}
		created++
	}
	if created != 3 {
		t.Fatalf("created directories recorded: %d", created)
	}
	if err := tx.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := beginPyPIVenvTransaction(p); err == nil {
		t.Fatal("interrupted transaction reused")
	}
}

func TestVenvRejectsDestinationTypesAndAliases(t *testing.T) {
	for _, kind := range []string{"regular", "directory", "symlink", "symlink-parent", "hardlink", "fifo", "socket"} {
		t.Run(kind, func(t *testing.T) {
			_, p := makePypiVenvFixture(t)
			path := filepath.Join(p.root, "bin", "tool")
			switch kind {
			case "directory":
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink("python", path); err != nil {
					t.Fatal(err)
				}
			case "symlink-parent":
				if err := os.Symlink(realPromotionRoot(t), filepath.Join(p.root, "bin", "nested")); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := unix.Mkfifo(path, 0o600); err != nil {
					t.Fatal(err)
				}
			case "socket":
				listener, err := net.Listen("unix", path)
				if err != nil {
					t.Fatal(err)
				}
				defer listener.Close()
			default:
				if err := os.WriteFile(path, []byte("baseline"), 0o600); err != nil {
					t.Fatal(err)
				}
				if kind == "hardlink" {
					if err := os.Link(path, filepath.Join(p.root, "alias")); err != nil {
						t.Fatal(err)
					}
				}
			}
			key := "scripts/tool"
			if kind == "symlink-parent" {
				key = "scripts/nested/tool"
			}
			if err := p.commit("", venvDestinations(t, map[string]string{key: "new"}, "demo")); err == nil {
				t.Fatal("unsafe collision accepted")
			}
		})
	}
	if _, err := snapshotPyPIFile("/dev/null"); err == nil {
		t.Fatal("device accepted as regular output")
	}
}

func TestVenvRejectsConfigAndSchemeIdentity(t *testing.T) {
	for _, kind := range []string{"missing", "malformed", "wrong-version", "version-substring", "duplicate", "root-symlink", "site-symlink", "scripts-symlink", "data-symlink"} {
		t.Run(kind, func(t *testing.T) {
			root, p := makePypiVenvFixture(t)
			cfg := filepath.Join(root, "pyvenv.cfg")
			switch kind {
			case "missing":
				if err := os.Remove(cfg); err != nil {
					t.Fatal(err)
				}
			case "root-symlink":
				alias := filepath.Join(realPromotionRoot(t), "alias")
				if err := os.Symlink(root, alias); err != nil {
					t.Fatal(err)
				}
				root = alias
			case "site-symlink", "scripts-symlink", "data-symlink":
				path := map[string]string{"site-symlink": p.site, "scripts-symlink": filepath.Join(root, "bin"), "data-symlink": filepath.Join(root, "share")}[kind]
				if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
					t.Fatal(err)
				}
				if err := os.Symlink(realPromotionRoot(t), path); err != nil {
					t.Fatal(err)
				}
			default:
				body := map[string]string{"malformed": "invalid", "wrong-version": "version = 3.13.7\n", "version-substring": "note = version = 3.14.7\n", "duplicate": "version = 3.14.7\nversion = 3.14.7\n"}[kind]
				if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := discoverPythonVenv(root); err == nil {
				t.Fatal("invalid venv accepted")
			}
		})
	}
}

func TestVenvDriftAndConcurrentGuard(t *testing.T) {
	for _, kind := range []string{"root", "config", "site", "scripts", "data", "concurrent", "appeared", "reconcile-drift"} {
		t.Run(kind, func(t *testing.T) {
			_, p := makePypiVenvFixture(t)
			if err := os.Mkdir(filepath.Join(p.root, "share"), 0o700); err != nil {
				t.Fatal(err)
			}
			tx, err := beginPyPIVenvTransaction(p)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.close()
			if kind == "concurrent" {
				if _, err := beginPyPIVenvTransaction(p); err == nil {
					t.Fatal("concurrent transaction accepted")
				}
				return
			}
			ds := venvDestinations(t, map[string]string{"site/new.py": "new"}, "demo")
			switch kind {
			case "config":
				if err := os.WriteFile(filepath.Join(p.root, "pyvenv.cfg"), []byte("version = 3.14.8\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "appeared":
				tx.fault = func(step string) error {
					if step == "before-publish:site/new.py" {
						return os.WriteFile(filepath.Join(p.site, "new.py"), []byte("outside"), 0o600)
					}
					return nil
				}
			case "reconcile-drift":
				tx.fault = func(step string) error {
					if step == "reconcile" {
						return os.WriteFile(filepath.Join(p.root, "pyvenv.cfg"), []byte("version = 3.14.8\n"), 0o600)
					}
					return nil
				}
			default:
				path := map[string]string{"root": p.root, "site": p.site, "scripts": filepath.Join(p.root, "bin"), "data": filepath.Join(p.root, "share")}[kind]
				if err := os.Rename(path, path+"-old"); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := tx.commit(ds); err == nil {
				t.Fatal("drift returned success")
			}
			if kind == "appeared" {
				body, err := os.ReadFile(filepath.Join(p.site, "new.py"))
				if err != nil || string(body) != "outside" {
					t.Fatal("concurrent file overwritten or deleted")
				}
			}
		})
	}
}

func TestVenvRejectsGlobalDuplicateAndProvenanceCollision(t *testing.T) {
	_, p := makePypiVenvFixture(t)
	ds := venvDestinations(t, map[string]string{"scripts/tool": "tool"}, "demo")
	if err := p.commit("", append(ds, ds[0])); err == nil {
		t.Fatal("duplicate accepted")
	}
	other := ds[0]
	other.Distribution = "other"
	if err := p.commit("", append(ds, other)); err == nil {
		t.Fatal("two distributions accepted for same destination")
	}
	for _, relative := range []string{"../escape", "nested/../../escape", "/outside", "."} {
		bad := ds[0]
		bad.Relative = relative
		if err := p.commit("", []pypiDestination{bad}); err == nil {
			t.Fatalf("escape accepted: %s", relative)
		}
	}
	runVenvFixture(t, p, map[string]string{"scripts/tool": "tool"}, "demo")
	if err := p.commit("", venvDestinations(t, map[string]string{"scripts/tool": "other"}, "other")); err == nil {
		t.Fatal("another package's file replaced")
	}
	if err := os.Link(filepath.Join(p.root, "bin", "tool"), filepath.Join(p.root, "alias")); err != nil {
		t.Fatal(err)
	}
	if _, err := beginPyPIVenvTransaction(p); err == nil {
		t.Fatal("managed hardlink accepted")
	}
}

func TestVenvLegacyDigestOnlyOwnershipFailsClosed(t *testing.T) {
	_, p := makePypiVenvFixture(t)
	if err := os.Mkdir(filepath.Join(p.root, ".heliopause"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"files":{"old.py":"` + strings.Repeat("a", 64) + `"}}`)
	meta := filepath.Join(p.root, pypiVenvMetadata)
	if err := os.WriteFile(meta, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := beginPyPIVenvTransaction(p); err == nil {
		t.Fatal("distribution ownership inferred from legacy digests")
	}
	after, err := os.ReadFile(meta)
	if err != nil || !bytes.Equal(body, after) {
		t.Fatal("legacy metadata modified")
	}
}

func TestVenvSameContentReplacementFailsReconciliation(t *testing.T) {
	for _, kind := range []string{"file", "metadata"} {
		t.Run(kind, func(t *testing.T) {
			_, p := makePypiVenvFixture(t)
			tx, err := beginPyPIVenvTransaction(p)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.close()
			tx.fault = func(step string) error {
				if step != "reconcile" {
					return nil
				}
				path := filepath.Join(p.site, "new.py")
				if kind == "metadata" {
					path = filepath.Join(p.root, pypiVenvMetadata)
				}
				body, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if err := os.Rename(path, path+"-outside"); err != nil {
					return err
				}
				return os.WriteFile(path, body, 0o600)
			}
			if err := tx.commit(venvDestinations(t, map[string]string{"site/new.py": "new"}, "demo")); err == nil || !tx.recovery {
				t.Fatal("same-content inode replacement reported clean")
			}
		})
	}
}

func TestVenvRollbackPreservesExistingDirectories(t *testing.T) {
	_, p := makePypiVenvFixture(t)
	dir := filepath.Join(p.root, "share", "man", "man1")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "baseline.1"), []byte("baseline"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := beginPyPIVenvTransaction(p)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.close()
	tx.fault = func(step string) error {
		if step == "reconcile" {
			return errors.New("injected")
		}
		return nil
	}
	if err := tx.commit(venvDestinations(t, map[string]string{"data/man/man1/new.1": "new", "data/new/deep/file": "new"}, "demo")); err == nil || tx.recovery {
		t.Fatal("rollback failed")
	}
	after, err := os.Stat(dir)
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("pre-existing directory replaced")
	}
	body, err := os.ReadFile(filepath.Join(dir, "baseline.1"))
	if err != nil || string(body) != "baseline" {
		t.Fatal("baseline modified")
	}
	for _, path := range []string{filepath.Join(dir, "new.1"), filepath.Join(p.root, "share", "new")} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("new file/directory remains")
		}
	}
}

func TestDiscoverPythonVenvRequiresPinnedMinorAndLayout(t *testing.T) {
	_, plan := makePypiVenvFixture(t)
	if plan.site == "" {
		t.Fatal("site-packages was not discovered")
	}
}

func TestDiscoverPythonVenvRejectsMissingConfiguration(t *testing.T) {
	t.Parallel()
	root := realPromotionRoot(t)
	if _, err := discoverPythonVenv(root); err == nil {
		t.Fatal("missing pyvenv.cfg was accepted as a Python virtual environment")
	}
}

func TestPypiVenvCommitTracksOnlyHAAOwnedFiles(t *testing.T) {
	root, plan := makePypiVenvFixture(t)
	output := filepath.Join(root, "output")
	if err := os.MkdirAll(filepath.Join(output, "demo"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "demo", "__init__.py"), []byte("v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	desired, err := plan.outputState(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.commit(output, desired); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(filepath.Join(plan.site, "demo", "__init__.py")); err != nil || string(body) != "v1\n" {
		t.Fatalf("committed file=%q error=%v", body, err)
	}
	state, managed, err := plan.readState()
	if err != nil || !managed || len(state.Files) != 1 {
		t.Fatalf("state=%#v managed=%v error=%v", state, managed, err)
	}
	if err := plan.verifyState(state); err != nil {
		t.Fatal(err)
	}
}

func TestPypiVenvRejectsBaselineCollisionAndRollsBackPartialPublish(t *testing.T) {
	_, plan := makePypiVenvFixture(t)
	if err := os.MkdirAll(filepath.Join(plan.site, "baseline"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plan.site, "baseline", "__init__.py"), []byte("host\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(plan.root, "output")
	if err := os.MkdirAll(filepath.Join(output, "baseline"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "baseline", "__init__.py"), []byte("replace\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	desired, err := plan.outputState(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.commit(output, desired); err == nil {
		t.Fatal("baseline collision was accepted")
	}
	body, err := os.ReadFile(filepath.Join(plan.site, "baseline", "__init__.py"))
	if err != nil || string(body) != "host\n" {
		t.Fatalf("baseline changed: %q error=%v", body, err)
	}
}

func TestPypiVenvRejectsInterruptedTransaction(t *testing.T) {
	root, plan := makePypiVenvFixture(t)
	if err := os.Mkdir(filepath.Join(root, ".heliopause-pypi-commit-stale"), 0o700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "output")
	if err := os.Mkdir(output, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "new.py"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	desired, err := plan.outputState(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.commit(output, desired); err == nil {
		t.Fatal("interrupted transaction was reused")
	}
}
