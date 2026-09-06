package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type venvGuardCheckingRunner struct {
	t      *testing.T
	plan   pypiVenvPlan
	called bool
}

func (r *venvGuardCheckingRunner) Run(_ context.Context, _ string, _ []string) error {
	r.called = true
	if _, err := os.Lstat(filepath.Join(r.plan.root, ".heliopause-pypi-transaction.lock")); err != nil {
		r.t.Fatal("guard absent during private execution")
	}
	if tx, err := beginPyPIVenvTransaction(r.plan); err == nil {
		_ = tx.close()
		r.t.Fatal("concurrent transaction entered during private execution")
	}
	return errors.New("deterministic private runner failure")
}
func TestVenvGuardCoversPrivateExecution(t *testing.T) {
	_, p := makePypiVenvFixture(t)
	root := realPromotionRoot(t)
	bundle, staged := validStagedPyPIFixture(t, root)
	runner := &venvGuardCheckingRunner{t: t, plan: p}
	promoter, err := newPyPIPromotion(filepath.Join(root, "staging"), runner, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(p.root)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewPythonVenvInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := promoter.Promote(context.Background(), staged, bundle, install); err == nil || !runner.called {
		t.Fatal("private execution guard test did not reach runner")
	}
	if _, err := os.Lstat(filepath.Join(p.root, ".heliopause-pypi-transaction.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("ordinary failed transaction leaked guard")
	}
}

func TestRecordDestinationsDriveExactSympyVenvTransaction(t *testing.T) {
	_, p := makePypiVenvFixture(t)
	project := realPromotionRoot(t)
	site := filepath.Join(project, "site")
	files := map[string]string{
		"sympy/__init__.py":               "sympy fixture\n",
		"sympy-1.14.0.dist-info/METADATA": "Name: sympy\nVersion: 1.14.0\n",
		"../../bin/isympy":                "#!/bin/sh\n",
		"../../share/man/man1/isympy.1":   "sympy man\n",
	}
	record := []string{}
	for relative, body := range files {
		path := filepath.Join(site, relative)
		switch relative {
		case "../../bin/isympy":
			path = filepath.Join(project, "bin", "isympy")
		case "../../share/man/man1/isympy.1":
			path = filepath.Join(project, "share", "man", "man1", "isympy.1")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(body))
		record = append(record, fmt.Sprintf("%s,sha256=%s,%d", relative, base64.RawURLEncoding.EncodeToString(sum[:]), len(body)))
	}
	record = append(record, "sympy-1.14.0.dist-info/RECORD,,")
	if err := os.WriteFile(filepath.Join(site, "sympy-1.14.0.dist-info", "RECORD"), []byte(strings.Join(record, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expected := map[string]pypiExpected{"sympy": {name: "sympy", version: "1.14.0", digest: strings.Repeat("a", 64)}}
	ds, err := validatedPyPIDestinations(site, expected, []byte("sympy==1.14.0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 5 || len(expected) != 1 {
		t.Fatal("distribution/file count conflated")
	}
	for _, d := range ds {
		if d.Distribution != "sympy" || d.Version != "1.14.0" || d.ArtifactDigest != strings.Repeat("a", 64) {
			t.Fatal("inspected provenance lost")
		}
	}
	if err := p.commit("", ds); err != nil {
		t.Fatal(err)
	}
	state, _, err := p.readState()
	if err != nil {
		t.Fatal(err)
	}
	for key, path := range map[string]string{"scripts/isympy": filepath.Join(p.root, "bin", "isympy"), "data/man/man1/isympy.1": filepath.Join(p.root, "share", "man", "man1", "isympy.1")} {
		if state.Files[key].Final != path {
			t.Fatalf("canonical destination changed: %s", key)
		}
		if _, err := os.Lstat(path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRecordRejectsCrossDistributionDestinationCollision(t *testing.T) {
	_, site, firstDist := validPromotionOutputFixture(t)
	second := filepath.Join(site, "other-1.0.0.dist-info")
	if err := os.Mkdir(second, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, "METADATA"), []byte("Name: other\nVersion: 1.0.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(firstDist, "RECORD"))
	if err != nil {
		t.Fatal(err)
	}
	var shared string
	for _, row := range strings.Split(string(first), "\n") {
		if strings.HasPrefix(row, "../../bin/") {
			shared = row
		}
	}
	if shared == "" {
		t.Fatal("missing fixture script")
	}
	if err := os.WriteFile(filepath.Join(second, "RECORD"), []byte(shared+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expected := fixtureExpected()
	expected["other"] = pypiExpected{name: "other", version: "1.0.0"}
	if _, err := validatedPyPIDestinations(site, expected, []byte("fixture other")); err == nil {
		t.Fatal("cross-distribution destination collision accepted")
	}
}

func TestRecordRejectsOutputOutsideSchemeRoots(t *testing.T) {
	root, site, _ := validPromotionOutputFixture(t)
	if err := os.Mkdir(filepath.Join(root, "libexec"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "libexec", "unexpected"), []byte("unexpected"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := validatedPyPIDestinations(site, fixtureExpected(), []byte("fixture==1.0.0")); err == nil {
		t.Fatal("outside-scheme output accepted")
	}
}

func TestPreparePyPIProjectBindsExactWheelAndHashRequirement(t *testing.T) {
	t.Parallel()
	root := realPromotionRoot(t)
	bundle, _ := validStagedPyPIFixture(t, root)
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0o700); err != nil {
		t.Fatal(err)
	}
	requirements, expected, err := preparePyPIProject(project, filepath.Join(root, "staging", bundle.ManifestID().String()), bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(expected) != 1 || !strings.Contains(string(requirements), "haa-promotion-fixture==1.0.0 --hash=sha256:") {
		t.Fatalf("requirements=%q expected=%v", requirements, expected)
	}
	if _, err := os.Stat(filepath.Join(project, "wheels", "haa_promotion_fixture-1.0.0-py3-none-any.whl")); err != nil {
		t.Fatalf("exact wheel not copied: %v", err)
	}
}

func TestValidatePyPIOutputRejectsUnrecordedOrMismatchedOutput(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	site := filepath.Join(root, "site")
	dist := filepath.Join(site, "haa_promotion_fixture-1.0.0.dist-info")
	if err := os.MkdirAll(dist, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "METADATA"), []byte("Name: haa-promotion-fixture\nVersion: 1.0.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "RECORD"), []byte("../escape,sha256=AAAA,1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expected := map[string]pypiExpected{"haa-promotion-fixture": {name: "haa-promotion-fixture", version: "1.0.0"}}
	if err := validatePyPIOutput(site, expected, []byte("haa-promotion-fixture==1.0.0 --hash=sha256:abc\n")); err == nil {
		t.Fatal("validatePyPIOutput accepted unsafe RECORD")
	}
}

func TestValidatePyPIOutputAcceptsWheelDataScript(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	site := filepath.Join(root, "site")
	dist := filepath.Join(site, "haa_promotion_fixture-1.0.0.dist-info")
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(dist, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	scriptContent := []byte("#!/bin/sh\necho fixture\n")
	if err := os.WriteFile(filepath.Join(binDir, "tool"), scriptContent, 0o600); err != nil {
		t.Fatal(err)
	}
	scriptSum := sha256.Sum256(scriptContent)
	scriptDigest := base64.RawURLEncoding.EncodeToString(scriptSum[:])

	metaContent := []byte("Name: haa-promotion-fixture\nVersion: 1.0.0\n")
	if err := os.WriteFile(filepath.Join(dist, "METADATA"), metaContent, 0o600); err != nil {
		t.Fatal(err)
	}
	metaSum := sha256.Sum256(metaContent)
	metaDigest := base64.RawURLEncoding.EncodeToString(metaSum[:])

	recordContent := fmt.Sprintf("../../bin/tool,sha256=%s,%d\nhaa_promotion_fixture-1.0.0.dist-info/METADATA,sha256=%s,%d\nhaa_promotion_fixture-1.0.0.dist-info/RECORD,,\n", scriptDigest, len(scriptContent), metaDigest, len(metaContent))
	if err := os.WriteFile(filepath.Join(dist, "RECORD"), []byte(recordContent), 0o600); err != nil {
		t.Fatal(err)
	}
	expected := map[string]pypiExpected{"haa-promotion-fixture": {name: "haa-promotion-fixture", version: "1.0.0"}}
	if err := validatePyPIOutput(site, expected, []byte("haa-promotion-fixture==1.0.0 --hash=sha256:abc\n")); err != nil {
		t.Fatalf("validatePyPIOutput rejected valid wheel data script: %v", err)
	}

	// Unsafe traversal must be rejected
	unsafeRecord := fmt.Sprintf("../../../bin/escape,sha256=%s,%d\n", scriptDigest, len(scriptContent))
	if err := os.WriteFile(filepath.Join(dist, "RECORD"), []byte(unsafeRecord), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validatePyPIOutput(site, expected, []byte("haa-promotion-fixture==1.0.0 --hash=sha256:abc\n")); err == nil {
		t.Fatal("validatePyPIOutput accepted unsafe traversed RECORD path")
	}
}

func TestValidatePyPIOutputRejectsUnsafeRecordForms(t *testing.T) {
	t.Parallel()
	for _, recordPath := range []string{
		"../../site/escape", // escape then re-enter is an alias, not a scheme path.
		"/tmp/escape",
		"C:\\escape",
		"..\\..\\bin\\escape",
	} {
		t.Run(recordPath, func(t *testing.T) {
			root := t.TempDir()
			site := filepath.Join(root, "site")
			dist := filepath.Join(site, "fixture-1.0.0.dist-info")
			if err := os.MkdirAll(dist, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dist, "METADATA"), []byte("Name: fixture\nVersion: 1.0.0\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dist, "RECORD"), []byte(recordPath+",sha256=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA,0\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			expected := map[string]pypiExpected{"fixture": {name: "fixture", version: "1.0.0"}}
			if err := validatePyPIOutput(site, expected, []byte("fixture==1.0.0\n")); err == nil {
				t.Fatal("unsafe RECORD path was accepted")
			}
		})
	}
}

func TestParseInstalledMetadataRejectsDuplicateOrMalformedIdentity(t *testing.T) {
	t.Parallel()
	for _, body := range [][]byte{
		[]byte("Name: fixture\nName: fixture\nVersion: 1.0.0\n"),
		[]byte("Name: fixture\nVersion: 1.0.0\nVersion: 2.0.0\n"),
		[]byte("Name fixture\nVersion: 1.0.0\n"),
		[]byte("Name: fixture\nVersion: 1.0.0\x00\n"),
	} {
		if _, _, err := parseInstalledMetadata(body); err == nil {
			t.Fatalf("unsafe METADATA was accepted: %q", body)
		}
	}
}

func TestValidatePyPIOutputRejectsRecordIntegrityAndOutputMismatches(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, root, site, dist string)
	}{
		{"duplicate canonical path", func(t *testing.T, _ string, _ string, dist string) {
			appendRecord(t, dist, "../fixture/__init__.py,sha256=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA,1\n")
		}},
		{"duplicate record row", func(t *testing.T, _ string, _ string, dist string) {
			appendRecord(t, dist, "fixture/__init__.py,sha256=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA,1\n")
		}},
		{"incorrect hash", func(t *testing.T, _ string, _ string, dist string) {
			replaceRecord(t, dist, "sha256=", "sha256=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		}},
		{"incorrect size", func(t *testing.T, _ string, _ string, dist string) {
			replaceRecord(t, dist, ",5\n", ",6\n")
		}},
		{"malformed size", func(t *testing.T, _ string, _ string, dist string) {
			replaceRecord(t, dist, ",5\n", ",not-a-size\n")
		}},
		{"unrecorded output", func(t *testing.T, _ string, site, _ string) {
			if err := os.WriteFile(filepath.Join(site, "unexpected.py"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"nested fake dist info", func(t *testing.T, _ string, site, _ string) {
			if err := os.MkdirAll(filepath.Join(site, "fixture", "fake-1.0.dist-info"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(site, "fixture", "fake-1.0.dist-info", "METADATA"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"malformed csv", func(t *testing.T, _ string, _ string, dist string) {
			if err := os.WriteFile(filepath.Join(dist, "RECORD"), []byte("only,two\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			root, site, dist := validPromotionOutputFixture(t)
			test.mutate(t, root, site, dist)
			if err := validatePyPIOutput(site, fixtureExpected(), []byte("fixture==1.0.0\n")); err == nil {
				t.Fatal("unsafe RECORD/output mismatch was accepted")
			}
		})
	}
}

func TestPromotionMetadataReadIsBounded(t *testing.T) {
	t.Parallel()
	filename := filepath.Join(t.TempDir(), "METADATA")
	if err := os.WriteFile(filename, make([]byte, artifactpypi.DefaultWheelLimits().MaxMetadata+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBoundedPromotionFile(filename, artifactpypi.DefaultWheelLimits().MaxMetadata); err == nil {
		t.Fatal("oversized METADATA was accepted")
	}
}

func fixtureExpected() map[string]pypiExpected {
	return map[string]pypiExpected{"fixture": {name: "fixture", version: "1.0.0"}}
}

func validPromotionOutputFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	site := filepath.Join(root, "site")
	dist := filepath.Join(site, "fixture-1.0.0.dist-info")
	if err := os.MkdirAll(filepath.Join(site, "fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "share", "man"), 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		filepath.Join(site, "fixture", "__init__.py"):    []byte("value"),
		filepath.Join(root, "bin", "fixture-tool"):       []byte("#!/bin/sh\n"),
		filepath.Join(root, "share", "man", "fixture.1"): []byte("fixture man\n"),
		filepath.Join(dist, "METADATA"):                  []byte("Name: fixture\nVersion: 1.0.0\n"),
	}
	for filename, body := range files {
		if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	record := make([]string, 0, len(files)+1)
	for filename, body := range files {
		relative := ""
		switch filename {
		case filepath.Join(root, "bin", "fixture-tool"):
			relative = "../../bin/fixture-tool"
		case filepath.Join(root, "share", "man", "fixture.1"):
			relative = "../../share/man/fixture.1"
		default:
			value, err := filepath.Rel(site, filename)
			if err != nil {
				t.Fatal(err)
			}
			relative = filepath.ToSlash(value)
		}
		sum := sha256.Sum256(body)
		record = append(record, fmt.Sprintf("%s,sha256=%s,%d", relative, base64.RawURLEncoding.EncodeToString(sum[:]), len(body)))
	}
	record = append(record, "fixture-1.0.0.dist-info/RECORD,,")
	if err := os.WriteFile(filepath.Join(dist, "RECORD"), []byte(strings.Join(record, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, site, dist
}

func appendRecord(t *testing.T, dist, row string) {
	t.Helper()
	filename := filepath.Join(dist, "RECORD")
	body, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, append(body, []byte(row)...), 0o600); err != nil {
		t.Fatal(err)
	}
}

func replaceRecord(t *testing.T, dist, old, replacement string) {
	t.Helper()
	filename := filepath.Join(dist, "RECORD")
	body, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(body), old, replacement, 1)
	if updated == string(body) {
		t.Fatalf("record replacement %q was not found", old)
	}
	if err := os.WriteFile(filename, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRelocateStagedSchemeRoots(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	site := filepath.Join(root, "site")
	bin := filepath.Join(site, "bin")
	share := filepath.Join(site, "share", "man")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(share, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "tool"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(share, "tool.1"), []byte("man\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := relocateStagedSchemeRoots(root); err != nil {
		t.Fatalf("relocateStagedSchemeRoots failed: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(site, "bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("site/bin still exists after relocation")
	}
	if _, err := os.Lstat(filepath.Join(site, "share")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("site/share still exists after relocation")
	}
	if body, err := os.ReadFile(filepath.Join(root, "bin", "tool")); err != nil || string(body) != "#!/bin/sh\n" {
		t.Fatalf("root/bin/tool content mismatch: %q, %v", body, err)
	}
	if body, err := os.ReadFile(filepath.Join(root, "share", "man", "tool.1")); err != nil || string(body) != "man\n" {
		t.Fatalf("root/share/man/tool.1 content mismatch: %q, %v", body, err)
	}

	// No-op when neither exists
	emptyRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(emptyRoot, "site"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := relocateStagedSchemeRoots(emptyRoot); err != nil {
		t.Fatalf("no-op relocate failed: %v", err)
	}

	// Rejects pre-existing destination
	collisionRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(collisionRoot, "site", "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(collisionRoot, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := relocateStagedSchemeRoots(collisionRoot); err == nil {
		t.Fatal("relocate accepted pre-existing destination")
	}
}

func TestRelocateStagedSchemeRootsRejectsHostilePathsBeforeMutation(t *testing.T) {
	for _, kind := range []string{"root-symlink", "site-symlink", "bin-symlink", "share-symlink", "share-fifo", "share-file", "share-collision"} {
		t.Run(kind, func(t *testing.T) {
			root, external := t.TempDir(), t.TempDir()
			for _, base := range []string{filepath.Join(root, "site"), external} {
				for _, scheme := range []string{"bin", "share"} {
					dir := filepath.Join(base, scheme)
					if err := os.MkdirAll(dir, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, "sentinel"), []byte("unchanged"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			argument := root
			switch kind {
			case "root-symlink":
				argument = filepath.Join(t.TempDir(), "root")
				if err := os.Symlink(root, argument); err != nil {
					t.Fatal(err)
				}
			case "site-symlink":
				if err := os.Rename(filepath.Join(root, "site"), filepath.Join(root, "original-site")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(external, filepath.Join(root, "site")); err != nil {
					t.Fatal(err)
				}
			case "share-collision":
				if err := os.Mkdir(filepath.Join(root, "share"), 0o700); err != nil {
					t.Fatal(err)
				}
			default:
				scheme := "share"
				if kind == "bin-symlink" {
					scheme = "bin"
				}
				src := filepath.Join(root, "site", scheme)
				if err := os.Rename(src, src+"-original"); err != nil {
					t.Fatal(err)
				}
				var err error
				switch kind {
				case "share-fifo":
					err = unix.Mkfifo(src, 0o600)
				case "share-file":
					err = os.WriteFile(src, []byte("not a directory"), 0o600)
				default:
					err = os.Symlink(filepath.Join(external, scheme), src)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			// WalkDir does not follow symlinks. Capture every directory, link,
			// special object and regular file before calling the product code.
			type snapshot struct {
				info os.FileInfo
				body string
			}
			capture := func() map[string]snapshot {
				result := map[string]snapshot{}
				for _, base := range []string{root, external} {
					if err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
						if err != nil {
							return err
						}
						info, err := entry.Info()
						if err != nil {
							return err
						}
						value := snapshot{info: info}
						if info.Mode().IsRegular() {
							body, err := os.ReadFile(path)
							if err != nil {
								return err
							}
							value.body = string(body)
						} else if info.Mode()&os.ModeSymlink != 0 {
							value.body, err = os.Readlink(path)
							if err != nil {
								return err
							}
						}
						result[path] = value
						return nil
					}); err != nil {
						t.Fatal(err)
					}
				}
				return result
			}
			before := capture()
			if err := relocateStagedSchemeRoots(argument); err == nil {
				t.Fatal("hostile staging accepted")
			}
			after := capture()
			if len(before) != len(after) {
				t.Fatal("rejected relocation added or removed objects")
			}
			for path, old := range before {
				now, ok := after[path]
				if !ok || !os.SameFile(old.info, now.info) || old.info.Mode() != now.info.Mode() || old.info.ModTime() != now.info.ModTime() || old.body != now.body {
					t.Fatalf("rejected relocation mutated %s", path)
				}
			}
		})
	}
}
