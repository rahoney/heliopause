package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGoTidySourceIsAnchoredAndRefusesUnsupportedInputs(t *testing.T) {
	for _, kind := range []string{"normal", "replace", "workspace", "nested", "symlink", "hardlink", "source-change", "same-bytes-replacement"} {
		t.Run(kind, func(t *testing.T) {
			project := t.TempDir()
			mod := []byte("module example.com/app\ngo 1.26.0\n")
			if kind == "replace" {
				mod = append(mod, []byte("replace example.com/other => ../other\n")...)
			}
			if err := os.WriteFile(filepath.Join(project, "go.mod"), mod, 0o600); err != nil {
				t.Fatal(err)
			}
			code := filepath.Join(project, "app.go")
			body := []byte("package app\nfunc init(){panic(\"must never execute\")}\n")
			if err := os.WriteFile(code, body, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(project, ".env"), []byte("NOT_SOURCE"), 0o600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "workspace":
				if err := os.WriteFile(filepath.Join(project, "go.work"), []byte("go 1.26.0\nuse .\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "nested":
				if err := os.Mkdir(filepath.Join(project, "nested"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(project, "nested/go.mod"), mod, 0o600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(code, filepath.Join(project, "link.go")); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(code, filepath.Join(project, "alias.go")); err != nil {
					t.Fatal(err)
				}
			}
			source, err := CaptureGoProjectSource(context.Background(), project)
			refused := kind == "replace" || kind == "workspace" || kind == "nested" || kind == "symlink" || kind == "hardlink"
			if refused {
				if err == nil {
					_ = source.Close()
					t.Fatal("unsupported source accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer source.Close()
			if _, ok := source.Files()[".env"]; ok {
				t.Fatal("private environment copied")
			}
			if !bytes.Equal(source.Files()["app.go"], body) {
				t.Fatal("source was rewritten")
			}
			if kind == "source-change" {
				if err := os.WriteFile(code, []byte("package changed\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "same-bytes-replacement" {
				if err := os.Rename(code, code+".old"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(code, body, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			err = source.Verify(context.Background())
			if (err != nil) != (kind != "normal") {
				t.Fatalf("source witness: %v", err)
			}
		})
	}
}

type goTidyFixture struct {
	goEmptyProjectRunner
	project, mode string
	calls         []string
}

func (r *goTidyFixture) RunGoProject(_ context.Context, workspace string, env []string, args ...string) ([]byte, error) {
	if workspace == r.project {
		return nil, errors.New("live project reached native runner")
	}
	code, err := os.ReadFile(filepath.Join(workspace, "app.go"))
	if err != nil || !bytes.Contains(code, []byte("must never execute")) {
		return nil, errors.New("frozen source not provided")
	}
	if _, err := os.ReadFile(filepath.Join(workspace, "go.sum")); err != nil {
		return nil, errors.New("private Go controls are incomplete")
	}
	command := strings.Join(args, " ")
	r.calls = append(r.calls, command)
	if command == "mod tidy" {
		if r.mode == "failure" {
			return nil, errors.New("native failed")
		}
		if r.mode == "source-change" {
			if err := os.WriteFile(filepath.Join(r.project, "app.go"), []byte("package changed\n"), 0o600); err != nil {
				return nil, err
			}
		}
		if r.mode == "directive" {
			if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/app\ngo 1.27.0\n"), 0o600); err != nil {
				return nil, err
			}
		}
		if err := os.WriteFile(filepath.Join(workspace, "go.sum"), nil, 0o600); err != nil {
			return nil, err
		}
		return nil, nil
	}
	return r.goEmptyProjectRunner.RunGo(context.Background(), workspace, env, args...)
}

func TestGoTidyResolverRejectsDriftAndKeepsNoPrimary(t *testing.T) {
	for _, mode := range []string{"normal", "failure", "source-change", "directive"} {
		t.Run(mode, func(t *testing.T) {
			project := t.TempDir()
			mod := []byte("module example.com/app\ngo 1.26.0\n")
			if err := os.WriteFile(filepath.Join(project, "go.mod"), mod, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(project, "app.go"), []byte("package app\n// must never execute\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			source, err := CaptureGoProjectSource(context.Background(), project)
			if err != nil {
				t.Fatal(err)
			}
			defer source.Close()
			original, err := readGoFrozenControls(project)
			if err != nil {
				t.Fatal(err)
			}
			target, _ := domain.NewInstallTarget(project)
			install, _ := domain.NewInstallContext(target)
			runner := &goTidyFixture{project: project, mode: mode}
			resolver, _ := NewGoModuleResolver(runner)
			update, err := resolver.ResolveProjectTidy(context.Background(), install, original, source)
			if mode != "normal" {
				if err == nil || update.Valid() {
					t.Fatal("failed selection returned update")
				}
				return
			}
			if err != nil || !update.Valid() || update.HasRequestedResolution() || !update.Snapshot().DependencyFree() {
				t.Fatalf("tidy result: %v", err)
			}
			if strings.Join(runner.calls, ";") != "mod tidy;mod download -json all;mod graph" {
				t.Fatalf("commands=%v", runner.calls)
			}
			after, err := readGoFrozenControls(project)
			if err != nil || !sameGoFrozenControls(original, after) {
				t.Fatal("resolver changed original controls")
			}
		})
	}
}
