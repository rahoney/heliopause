package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type cargoProjectTestRunner struct {
	t                        *testing.T
	original                 string
	metadata, manifest, lock []byte
	workspaces, homes        []string
	mutateMetadata           string
	failure                  error
}

func TestCargoProjectRejectsLockSourceBeforeRunner(t *testing.T) {
	for _, operation := range []string{"metadata", "add"} {
		for _, mutation := range []string{"syntax", "git", "alternate-registry", "missing-checksum", "dependency-source"} {
			t.Run(operation+"/"+mutation, func(t *testing.T) {
				root, install, runner := cargoProjectResolverFixture(t, true)
				lock := string(runner.lock)
				switch mutation {
				case "syntax":
					lock = "invalid [["
				case "git":
					lock = strings.ReplaceAll(lock, "registry+https://github.com/rust-lang/crates.io-index", "git+https://example.invalid/repository")
				case "alternate-registry":
					lock = strings.ReplaceAll(lock, "registry+https://github.com/rust-lang/crates.io-index", "registry+https://example.invalid/index")
				case "missing-checksum":
					lock = strings.ReplaceAll(lock, "92ecc6618181def0457392ccd0ee51198e065e016d1d527a7ac1b6dc7c1f09d2", "")
				case "dependency-source":
					lock = strings.ReplaceAll(lock, `dependencies = ["itoa"]`, `dependencies = ["itoa 1.0.17 (git+https://example.invalid/repository)"]`)
				}
				if lock == string(runner.lock) {
					t.Fatal("fixture did not mutate the lock")
				}
				if err := os.WriteFile(filepath.Join(root, "Cargo.lock"), []byte(lock), 0o600); err != nil {
					t.Fatal(err)
				}
				runner.failure = errors.New("SDK runner reached")
				resolver, _ := NewCargoResolver(runner)
				var err error
				if operation == "metadata" {
					_, err = resolver.ResolveProjectDependencies(context.Background(), install)
				} else {
					manifest, readErr := os.ReadFile(filepath.Join(root, "Cargo.toml"))
					if readErr != nil {
						t.Fatal(readErr)
					}
					controls, controlErr := cargoFrozenControls(map[string][]byte{"Cargo.toml": manifest, "Cargo.lock": []byte(lock)})
					if controlErr != nil {
						t.Fatal(controlErr)
					}
					reference, _ := artifactcargo.ParseReference("itoa@1.0.17")
					_, err = resolver.ResolveProjectDependencyUpdate(context.Background(), reference, install, controls)
				}
				if err == nil || len(runner.workspaces) != 0 {
					t.Fatalf("unsupported lock reached SDK runner: calls=%d err=%v", len(runner.workspaces), err)
				}
			})
		}
	}
}

func (r *cargoProjectTestRunner) RunCargo(_ context.Context, workspace string, environment []string, arguments ...string) ([]byte, error) {
	r.t.Helper()
	if workspace == r.original {
		r.t.Fatal("original project passed to Cargo")
	}
	home := strings.TrimPrefix(environment[0], "CARGO_HOME=")
	if ValidateCargoResolverEnvironment(environment, home) != nil {
		r.t.Fatal("noncanonical environment passed to Cargo")
	}
	r.workspaces = append(r.workspaces, workspace)
	r.homes = append(r.homes, home)
	data, err := os.ReadFile(filepath.Join(workspace, "src/main.rs"))
	if err != nil || string(data) != "fn main() {}\n" {
		r.t.Fatal("anchored local source not copied")
	}
	if r.failure != nil {
		return nil, r.failure
	}
	if arguments[0] == "add" {
		if strings.Join(arguments, " ") != "add itoa@=1.0.17" {
			r.t.Fatal("exact request was changed")
		}
		if err := os.WriteFile(filepath.Join(workspace, "Cargo.toml"), r.manifest, 0o600); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(workspace, "Cargo.lock"), r.lock, 0o600); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if r.mutateMetadata != "" {
		if err := os.WriteFile(filepath.Join(workspace, r.mutateMetadata), []byte("changed\n"), 0o600); err != nil {
			return nil, err
		}
	}
	return []byte(strings.ReplaceAll(string(r.metadata), "/fixture", workspace)), nil
}

func cargoProjectResolverFixture(t *testing.T, locked bool) (string, domain.InstallContext, *cargoProjectTestRunner) {
	t.Helper()
	root := cargoSourceFixture(t)
	manifest := []byte("[package]\nname='haa_schema_fixture'\nversion='0.1.0'\nedition='2021'\n[dependencies]\nitoa='=1.0.17'\n")
	metadata, err := os.ReadFile("../artifact/cargo/testdata/metadata-v1-rust-1.99.0.json")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := os.ReadFile("../artifact/cargo/testdata/metadata-v1-rust-1.99.0.lock")
	if err != nil {
		t.Fatal(err)
	}
	initial := manifest
	if !locked {
		initial = []byte("[package]\nname='haa_schema_fixture'\nversion='0.1.0'\nedition='2021'\n")
	}
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), initial, 0o600); err != nil {
		t.Fatal(err)
	}
	if locked {
		if err := os.WriteFile(filepath.Join(root, "Cargo.lock"), lock, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	target, _ := domain.NewInstallTarget(root)
	install, _ := domain.NewInstallContext(target)
	return root, install, &cargoProjectTestRunner{t: t, original: root, manifest: manifest, lock: lock, metadata: metadata}
}

func assertCargoPrivateCleanup(t *testing.T, runner *cargoProjectTestRunner) {
	t.Helper()
	for _, directory := range append(append([]string(nil), runner.workspaces...), runner.homes...) {
		if _, err := os.Lstat(directory); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("private Cargo data remains")
		}
	}
}

func TestCargoCompleteProjectResolutionUsesPrivateSource(t *testing.T) {
	root, install, runner := cargoProjectResolverFixture(t, true)
	resolver, _ := NewCargoResolver(runner)
	snapshot, err := resolver.ResolveProjectDependencies(context.Background(), install)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Valid() || len(snapshot.Dependencies()) != 1 || snapshot.Context() != install || snapshot.Source() != artifactcargo.Source() {
		t.Fatal("whole project snapshot incomplete")
	}
	current, err := os.ReadFile(filepath.Join(root, "Cargo.lock"))
	if err != nil || !bytes.Equal(current, runner.lock) {
		t.Fatal("original lock changed")
	}
	assertCargoPrivateCleanup(t, runner)
}

func TestCargoProjectUpdateFreezesOnePrivateSelection(t *testing.T) {
	root, install, runner := cargoProjectResolverFixture(t, false)
	originalManifest, err := os.ReadFile(filepath.Join(root, "Cargo.toml"))
	if err != nil {
		t.Fatal(err)
	}
	original, err := cargoFrozenControls(map[string][]byte{"Cargo.toml": originalManifest})
	if err != nil {
		t.Fatal(err)
	}
	resolver, _ := NewCargoResolver(runner)
	reference, _ := artifactcargo.ParseReference("itoa@1.0.17")
	update, err := resolver.ResolveProjectDependencyUpdate(context.Background(), reference, install, original)
	if err != nil {
		t.Fatal(err)
	}
	if !update.Valid() || !artifactcargo.EqualControls(update.OriginalControls(), original) || len(update.Snapshot().Dependencies()) != 1 || update.Resolution().LockfileDigest() != update.Snapshot().GraphDigest() {
		t.Fatal("selection is incomplete")
	}
	if len(runner.workspaces) != 2 || runner.workspaces[0] != runner.workspaces[1] {
		t.Fatal("selection replayed in different private projects")
	}
	current, err := os.ReadFile(filepath.Join(root, "Cargo.toml"))
	if err != nil || !bytes.Equal(current, originalManifest) {
		t.Fatal("original manifest changed before transaction")
	}
	if _, err := os.Lstat(filepath.Join(root, "Cargo.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("original missing lock was published before approval")
	}
	assertCargoPrivateCleanup(t, runner)
}

func TestCargoProjectResolverRejectsPrivateDriftAndPreservesFailure(t *testing.T) {
	for _, name := range []string{"Cargo.toml", "Cargo.lock", "src/main.rs", "data/extra.txt", "trusted failure"} {
		t.Run(name, func(t *testing.T) {
			_, install, runner := cargoProjectResolverFixture(t, true)
			if name == "trusted failure" {
				runner.failure = &isolatedCargoResolverFailure{cause: context.DeadlineExceeded, phase: "COMMAND", diagnostic: TraceDiagnostic{Reason: "STREAM_FAULT"}}
			} else {
				runner.mutateMetadata = name
			}
			resolver, _ := NewCargoResolver(runner)
			snapshot, err := resolver.ResolveProjectDependencies(context.Background(), install)
			if err == nil || snapshot.Valid() {
				t.Fatal("failed selection returned snapshot")
			}
			if name == "trusted failure" && (!errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "phase=COMMAND") || !strings.Contains(err.Error(), "STREAM_FAULT")) {
				t.Fatal("first trusted failure lost")
			}
			assertCargoPrivateCleanup(t, runner)
		})
	}
	for _, name := range []string{"Cargo.toml", "Cargo.lock", "src/main.rs"} {
		t.Run("update metadata "+name, func(t *testing.T) {
			root, install, runner := cargoProjectResolverFixture(t, false)
			manifest, err := os.ReadFile(filepath.Join(root, "Cargo.toml"))
			if err != nil {
				t.Fatal(err)
			}
			original, _ := cargoFrozenControls(map[string][]byte{"Cargo.toml": manifest})
			runner.mutateMetadata = name
			resolver, _ := NewCargoResolver(runner)
			reference, _ := artifactcargo.ParseReference("itoa@1.0.17")
			update, err := resolver.ResolveProjectDependencyUpdate(context.Background(), reference, install, original)
			if err == nil || update.Valid() {
				t.Fatal("metadata changed already selected source")
			}
			assertCargoPrivateCleanup(t, runner)
		})
	}
}
