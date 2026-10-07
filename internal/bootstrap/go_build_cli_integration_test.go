package bootstrap_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/bootstrap"
)

func TestLinuxGoDependencyFreeBuildCLIIntegration(t *testing.T) {
	goBuildCLIIntegration(t, "", "package main\nfunc main() {}\n", false, true)
}
func TestLinuxGoPublicBuildCLIIntegration(t *testing.T) {
	goBuildCLIIntegration(t, "github.com/spf13/pflag@v1.0.9", "package main\nimport \"github.com/spf13/pflag\"\nfunc main() { pflag.Parse() }\n", false, false)
}
func TestLinuxGoTransitiveBuildCLIIntegration(t *testing.T) {
	goBuildCLIIntegration(t, "google.golang.org/grpc@v1.76.0", "package main\nimport \"google.golang.org/grpc\"\nfunc main() { _ = grpc.NewServer() }\n", false, false)
}
func TestLinuxGoInvalidTestdataBuildCLIIntegration(t *testing.T) {
	goBuildCLIIntegration(t, "", "package main\nfunc main() {}\n", true, false)
}

func TestLinuxGoCgoBuildCLIIntegration(t *testing.T) {
	goBuildCLIIntegration(t, "", "package main\n/* int answer(void) { return 42; } */\nimport \"C\"\nfunc main() { _ = C.answer() }\n", false, false)
}

func TestLinuxGoLibraryBuildCLIIntegration(t *testing.T) {
	goBuildCLIIntegration(t, "", "package library\nfunc Answer() int { return 42 }\n", false, false)
}

func TestLinuxGoMixedPackagesBuildCLIIntegration(t *testing.T) {
	goBuildCLIWithLibraryIntegration(t, "", "package main\nfunc main() {}\n", false, false, "package library\nfunc Answer() int { return 42 }\n", false)
}

func TestLinuxGoMixedInvalidPackageBuildCLIIntegration(t *testing.T) {
	goBuildCLIWithLibraryIntegration(t, "", "package main\nfunc main() {}\n", false, false, "package library\nfunc Broken() { undefinedFunction() }\n", true)
}

func TestLinuxGoBuildSecurityCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_GO_BUILD_CLI_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("Go qualification requires Linux amd64")
	}
	for _, failure := range []string{"filesystem", "missing-cache", "unmanaged"} {
		t.Run(failure, func(t *testing.T) {
			var ctx context.Context
			var project string
			if failure == "unmanaged" {
				root := t.TempDir()
				project = filepath.Join(root, "project")
				if err := os.Mkdir(project, 0o700); err != nil {
					t.Fatal(err)
				}
				for name, body := range map[string]string{"go.mod": "module example.com/unmanaged\ngo 1.26\n", "main.go": "package main\nfunc main() {}\n"} {
					if err := os.WriteFile(filepath.Join(project, name), []byte(body), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
				t.Chdir(project)
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)
				defer cancel()
			} else {
				reference, main := "", "package main\n/*\n#include \"/etc/passwd\"\n*/\nimport \"C\"\nfunc main() {}\n"
				if failure == "missing-cache" {
					reference = "github.com/spf13/pflag@v1.0.9"
					main = "package main\nimport \"github.com/spf13/pflag\"\nfunc main() { pflag.Parse() }\n"
				}
				var root string
				ctx, root, project = prepareGoBuildIntegrationProject(t, reference, main)
				if failure == "missing-cache" {
					removed := false
					err := filepath.WalkDir(filepath.Join(root, "cache", "heliopause", "go-verified-cache"), func(path string, entry os.DirEntry, err error) error {
						if err != nil {
							return err
						}
						if !removed && !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
							parent, err := os.Stat(filepath.Dir(path))
							if err != nil {
								return err
							}
							if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
								return err
							}
							if err := os.Remove(path); err != nil {
								return err
							}
							if err := os.Chmod(filepath.Dir(path), parent.Mode().Perm()); err != nil {
								return err
							}
							removed = true
						}
						return nil
					})
					if err != nil || !removed {
						t.Fatalf("missing-cache fixture: %v", err)
					}
				}
			}
			beforeMod, err := os.ReadFile(filepath.Join(project, "go.mod"))
			if err != nil {
				t.Fatal(err)
			}
			beforeSum, sumErr := os.ReadFile(filepath.Join(project, "go.sum"))
			if sumErr != nil && !os.IsNotExist(sumErr) {
				t.Fatal(sumErr)
			}
			var stdout, stderr bytes.Buffer
			err = bootstrap.Run(ctx, []string{"go", "build", "./..."}, &stdout, &stderr)
			if err == nil || strings.Contains(stdout.String(), "Build output:") || strings.Contains(stdout.String(), "Policy: ALLOW") {
				t.Fatalf("unsafe build completed: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
			}
			if failure == "filesystem" && !strings.Contains(err.Error(), "STREAM_FAULT") {
				t.Fatalf("filesystem fixture did not reach observer denial: %v", err)
			}
			if entries, e := os.ReadDir(filepath.Join(project, ".heliopause", "builds")); !os.IsNotExist(e) && (e != nil || len(entries) != 0) {
				t.Fatal("failed build published output")
			}
			afterMod, e := os.ReadFile(filepath.Join(project, "go.mod"))
			if e != nil || !bytes.Equal(beforeMod, afterMod) {
				t.Fatal("failed build changed go.mod")
			}
			afterSum, e := os.ReadFile(filepath.Join(project, "go.sum"))
			if (os.IsNotExist(sumErr) != os.IsNotExist(e)) || (e != nil && !os.IsNotExist(e)) || !bytes.Equal(beforeSum, afterSum) {
				t.Fatal("failed build changed go.sum")
			}
			if _, e := os.Lstat(filepath.Join(project, ".heliopause-go-transaction.lock")); !os.IsNotExist(e) {
				t.Fatal("failed build retained guard")
			}
		})
	}
}

func goBuildCLIIntegration(t *testing.T, reference, main string, invalid, reuse bool) {
	t.Helper()
	goBuildCLIWithLibraryIntegration(t, reference, main, invalid, reuse, "", false)
}

func goBuildCLIWithLibraryIntegration(t *testing.T, reference, main string, invalid, reuse bool, library string, invalidLibrary bool) {
	t.Helper()
	if os.Getenv("HELOX_GO_BUILD_CLI_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and authenticated helper")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("Go qualification requires Linux amd64")
	}
	ctx, root, project := prepareGoBuildIntegrationProject(t, reference, main)
	if library != "" {
		if err := os.Mkdir(filepath.Join(project, "library"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, "library", "library.go"), []byte(library), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	controls := map[string][]byte{}
	names := []string{"go.mod", "go.sum", "main.go", "testdata/bad.go"}
	if library != "" {
		names = append(names, "library/library.go")
	}
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(project, name))
		if err != nil {
			t.Fatal(err)
		}
		controls[name] = body
	}
	// Ambient values must not enter the trusted fixed command/environment.
	t.Setenv("GOFLAGS", "-toolexec=/haa-forbidden-ambient-tool")
	t.Setenv("GOMODCACHE", filepath.Join(root, "ambient-poison-cache"))
	t.Setenv("GOPROXY", "https://invalid.example")
	t.Setenv("GOPRIVATE", "*")
	repeats := 1
	if reuse {
		repeats = 2
	}
	seen := map[string]bool{}
	for range repeats {
		selector := "./..."
		if invalid {
			selector = "./testdata"
		}
		var stdout, stderr bytes.Buffer
		err := bootstrap.Run(ctx, []string{"go", "build", selector}, &stdout, &stderr)
		if invalid || invalidLibrary {
			if err == nil || !strings.Contains(err.Error(), "command_status=EXIT_STATUS_1") || strings.Contains(stdout.String(), "Build output:") {
				t.Fatalf("invalid selected source changed failure: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
			}
			if entries, readErr := os.ReadDir(filepath.Join(project, ".heliopause", "builds")); !os.IsNotExist(readErr) && (readErr != nil || len(entries) != 0) {
				t.Fatal("failed compiler published output")
			}
			bound := false
			for _, line := range strings.Split(stdout.String(), "\n") {
				if digest, ok := strings.CutPrefix(line, "Digest: sha256:"); ok {
					decoded, e := hex.DecodeString(digest)
					if e != nil || len(decoded) != 32 {
						t.Fatal("failed compiler source binding is invalid")
					}
					t.Logf("actual selected compiler failure selector=%s source_sha256=%s trusted_failure=%v", selector, digest, err)
					bound = true
				}
			}
			if !bound {
				t.Fatal("failed compiler source binding missing")
			}
			break
		}
		if err != nil || !strings.Contains(stdout.String(), "Policy: ALLOW") {
			t.Fatalf("actual build CLI: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
		}
		directory := ""
		for _, line := range strings.Split(stdout.String(), "\n") {
			if suffix, ok := strings.CutPrefix(line, "Build output: "); ok {
				directory = suffix
			}
		}
		if !strings.HasPrefix(directory, ".heliopause/builds/run_") || seen[directory] {
			t.Fatalf("output destination missing/reused: %q", directory)
		}
		seen[directory] = true
		var receipt struct {
			Run, Source, Cache, Graph, Output, Recipe string
			Files                                     []artifactgo.BuildOutputFile
			Evidence                                  []struct{ ID, Handle, SHA256 string }
		}
		body, err := os.ReadFile(filepath.Join(project, directory, ".haa-build.json"))
		if err != nil {
			t.Fatal(err)
		}
		// Explicit field tags preserve the canonical digest role names.
		var values map[string]json.RawMessage
		if json.Unmarshal(body, &values) != nil || json.Unmarshal(values["run"], &receipt.Run) != nil || json.Unmarshal(values["source_sha256"], &receipt.Source) != nil || json.Unmarshal(values["cache_sha256"], &receipt.Cache) != nil || json.Unmarshal(values["graph_sha256"], &receipt.Graph) != nil || json.Unmarshal(values["output_sha256"], &receipt.Output) != nil || json.Unmarshal(values["recipe_sha256"], &receipt.Recipe) != nil || json.Unmarshal(values["files"], &receipt.Files) != nil || json.Unmarshal(values["evidence"], &receipt.Evidence) != nil {
			t.Fatal("build receipt is invalid")
		}
		outputFiles := 1
		if strings.HasPrefix(main, "package library\n") {
			outputFiles = 0
		}
		if filepath.Base(directory) != receipt.Run || len(receipt.Files) != outputFiles || len(receipt.Evidence) != 3 {
			t.Fatal("build output/Run/Evidence coverage differs")
		}
		for _, digest := range []string{receipt.Source, receipt.Cache, receipt.Graph, receipt.Output, receipt.Recipe} {
			if len(digest) != 64 {
				t.Fatal("build digest binding is missing")
			}
		}
		for _, file := range receipt.Files {
			body, err := os.ReadFile(filepath.Join(project, directory, file.Name))
			if err != nil || int64(len(body)) != file.Size || len(body) < 4 || !bytes.Equal(body[:4], []byte{0x7f, 'E', 'L', 'F'}) {
				t.Fatal("bounded compiler output is not exact ELF data")
			}
			hash := sha256.Sum256(body)
			if hex.EncodeToString(hash[:]) != file.SHA256 {
				t.Fatal("published output differs from receipt")
			}
		}
		for _, record := range receipt.Evidence {
			body, err := os.ReadFile(filepath.Join(root, "cache", "heliopause", "evidence", receipt.Run, record.ID+".json"))
			if err != nil || record.Handle != "evidence:"+receipt.Run+":"+record.ID {
				t.Fatal("actual build Evidence is missing")
			}
			hash := sha256.Sum256(body)
			if hex.EncodeToString(hash[:]) != record.SHA256 {
				t.Fatal("build Evidence binding changed")
			}
		}
		t.Logf("actual CLI retained build reference=%s graph=%s source=%s cache=%s output=%s recipe=%s", reference, receipt.Graph, receipt.Source, receipt.Cache, receipt.Output, receipt.Recipe)
	}
	for name, before := range controls {
		after, err := os.ReadFile(filepath.Join(project, name))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("build changed original input")
		}
	}
	if _, err := os.Lstat(filepath.Join(project, ".heliopause-go-transaction.lock")); !os.IsNotExist(err) {
		t.Fatal("build guard was not released")
	}
}
