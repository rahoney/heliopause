package sandbox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGoUpgradeCommandAdmission(t *testing.T) {
	for _, args := range [][]string{{"get", "-u", "example.com/mod"}, {"get", "-u", "example.com/Case/v2"}} {
		if !validGoResolverCommand(args) {
			t.Fatalf("reviewed exact named upgrade refused: %q", args)
		}
	}
	for _, args := range [][]string{{"get", "-u"}, {"get", "-u=patch", "example.com/mod"}, {"get", "-u", "example.com/mod@v1.2.3"}, {"get", "-u", "./..."}, {"get", "-u", "example.com/mod/..."}, {"get", "-u", "example.com/mod", "example.com/other"}, {"get", "-u", "go"}, {"get", "-u", "toolchain"}, {"get", "-u", "https://example.com/mod"}, {"get", "-u", "-modfile=other"}} {
		if validGoResolverCommand(args) {
			t.Fatalf("unsupported upgrade admitted: %q", args)
		}
	}
}

type goUpgradeRunner struct {
	goModuleRunnerFixture
	project  string
	mode     string
	selected []byte
}

func (r *goUpgradeRunner) RunGo(ctx context.Context, directory string, env []string, args ...string) ([]byte, error) {
	if directory == r.project {
		panic("user project executed")
	}
	body, err := r.goModuleRunnerFixture.RunGo(ctx, directory, env, args...)
	if err != nil {
		return nil, err
	}
	command := strings.Join(args, " ")
	if command == "get -u example.com/mod" {
		selected := r.selected
		if r.mode == "directive" {
			selected = []byte(strings.ReplaceAll(string(selected), "go 1.25", "go 1.26.9"))
		}
		if err := os.WriteFile(filepath.Join(directory, "go.mod"), selected, 0o600); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(directory, "go.sum"), upgradeSums(), 0o600); err != nil {
			return nil, err
		}
		if r.mode == "original-drift" {
			if err := os.WriteFile(filepath.Join(r.project, "go.mod"), []byte("module example.com/foreign\n"), 0o600); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	if command == "mod download -json all" {
		var result []byte
		for _, pair := range upgradeModules() {
			record := artifactgo.DownloadRecord{Path: pair[0], Version: pair[1], GoMod: "/private/mod", Zip: "/private/zip", Sum: upgradeH1(), GoModSum: upgradeH1()}
			data, err := json.Marshal(record)
			if err != nil {
				return nil, err
			}
			result = append(append(result, data...), '\n')
		}
		return result, nil
	}
	if command == "mod graph" {
		return []byte("example.com/app example.com/mod@v1.2.3\nexample.com/app example.com/dep@v1.1.0\nexample.com/app example.com/retained@v1.0.0\nexample.com/mod@v1.2.3 example.com/dep@v1.0.0\nexample.com/app go@1.25\ngo@1.25 toolchain@go1.25\n"), nil
	}
	return body, nil
}

func upgradeModules() [][2]string {
	return [][2]string{{"example.com/mod", "v1.2.3"}, {"example.com/dep", "v1.1.0"}, {"example.com/retained", "v1.0.0"}}
}
func upgradeH1() string {
	return "h1:" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32)))
}
func upgradeSums() []byte {
	var result string
	for _, pair := range upgradeModules() {
		result += pair[0] + " " + pair[1] + " " + upgradeH1() + "\n" + pair[0] + " " + pair[1] + "/go.mod " + upgradeH1() + "\n"
	}
	return []byte(result)
}

func TestGoUpgradeSelectsOnceWithCompleteTransitiveAndRetainedProject(t *testing.T) {
	for _, mode := range []string{"changed", "noop", "directive", "original-drift", "missing"} {
		t.Run(mode, func(t *testing.T) {
			project := t.TempDir()
			selected := []byte("module example.com/app\ngo 1.25\nrequire (\nexample.com/mod v1.2.3\nexample.com/dep v1.1.0 // indirect\nexample.com/retained v1.0.0\n)\n")
			original := selected
			if mode != "noop" {
				original = []byte(strings.ReplaceAll(strings.ReplaceAll(string(selected), "v1.2.3", "v1.0.0"), "v1.1.0", "v1.0.0"))
			}
			for name, body := range map[string][]byte{"go.mod": original, "go.sum": upgradeSums()} {
				if err := os.WriteFile(filepath.Join(project, name), body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			controls, err := readGoFrozenControls(project)
			if err != nil {
				t.Fatal(err)
			}
			target, _ := domain.NewInstallTarget(project)
			install, _ := domain.NewInstallContext(target)
			runner := &goUpgradeRunner{project: project, mode: mode, selected: selected}
			resolver, _ := NewGoModuleResolver(runner)
			selector := "example.com/mod"
			if mode == "missing" {
				selector = "example.com/missing"
			}
			update, err := resolver.ResolveProjectDependencyUpgrade(context.Background(), install, controls, selector)
			if mode == "directive" || mode == "original-drift" || mode == "missing" {
				if err == nil || update.Valid() {
					t.Fatalf("invalid selection returned success: %v", err)
				}
				if mode == "missing" && len(runner.calls) != 0 {
					t.Fatal("missing selector reached runner")
				}
				return
			}
			if err != nil || !update.HasRequestedResolution() || len(update.Snapshot().Dependencies()) != 3 {
				t.Fatalf("complete selection: %v", err)
			}
			if strings.Join(runner.calls, ";") != "get -u example.com/mod;mod download -json all;mod graph" {
				t.Fatal(runner.calls)
			}
			for _, node := range update.Resolution().Graph().Nodes() {
				if node.Node() == update.Resolution().Graph().Primary() && (node.Artifact().Identity().Name() != selector || node.Artifact().Identity().Version() != "v1.2.3") {
					t.Fatal("primary substituted")
				}
			}
			current, err := readGoFrozenControls(project)
			if err != nil || !sameGoFrozenControls(controls, current) {
				t.Fatal("original controls changed")
			}
			for _, cache := range runner.caches {
				if _, err := os.Stat(cache); !os.IsNotExist(err) {
					t.Fatal("private cache remains")
				}
			}
		})
	}
}
