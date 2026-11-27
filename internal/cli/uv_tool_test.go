package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func uvToolCLIPaths(t *testing.T) (string, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target, bin := filepath.Join(root, "demo-package"), filepath.Join(root, "commands")
	for _, path := range []string{target, bin} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return target, bin
}

func TestUVToolClientRoutesCanonicalCurrentInstallRequest(t *testing.T) {
	for _, input := range []string{"Demo_Package", "Demo_Package==1.0", "Demo_Package@1.0", "Demo_Package>=1,<2"} {
		target, bin := uvToolCLIPaths(t)
		var out bytes.Buffer
		root, err := New(&out, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		recorder := &uvClientRequestRecorder{}
		if err := AddUVToolInstall(root, recorder); err != nil {
			t.Fatal(err)
		}
		args := []string{"--output", "json", "--lang", "ko", "uv", "tool", "install", input, "--target", target, "--bin-dir", bin}
		source, err := PreparePresentation(root, args)
		if err != nil || source != "pypi" || SelectedPackageClient(root) != UVToolPackageClient {
			t.Fatal("wrong tool route", source, err)
		}
		root.SetArgs(args)
		_ = root.ExecuteContext(context.Background())
		if len(recorder.requests) != 1 {
			t.Fatal("tool request not routed once")
		}
		request := recorder.requests[0]
		requirement, err := artifactpypi.ParseUpgradeRequirement(input)
		if err != nil {
			t.Fatal(err)
		}
		if request.Operation() != domain.OperationInstall || request.Reference().Locator() != requirement.Requirement() || request.Reference().Source() != artifactpypi.PublicPyPIProfile().Source() || request.Context().Mode() != domain.InstallPythonVenv || request.Context().Target().String() != target {
			t.Fatal("tool request was changed")
		}
	}
}

func TestUVToolClientRejectsImplicitUnsafeAndUnsupportedInputBeforeBinding(t *testing.T) {
	target, bin := uvToolCLIPaths(t)
	t.Setenv("VIRTUAL_ENV", target)
	alias := filepath.Join(filepath.Dir(target), "alias")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"uv", "tool", "install", "demo-package"},
		{"uv", "tool", "install", "demo-package", "--bin-dir", bin},
		{"uv", "tool", "install", "demo-package", "--target", target},
		{"uv", "tool", "install", "other", "--target", target, "--bin-dir", bin},
		{"uv", "tool", "install", "demo-package", "--target", target, "--bin-dir", target},
		{"uv", "tool", "install", "demo-package", "--target", alias, "--bin-dir", bin},
		{"uv", "tool", "install", "demo-package", "--target", "relative", "--bin-dir", bin},
		{"uv", "tool", "install", "demo-package", "--target", target, "--bin-dir", bin, "--bin-dir", bin},
		{"uv", "tool", "install", "demo-package", "--target", target, "--target", target, "--bin-dir", bin},
		{"uv", "tool", "install", "demo-package", "--target", target, "--bin-dir", bin, "--with", "other"},
		{"uv", "tool", "install", "demo-package", "--target", target, "--bin-dir", bin, "--upgrade"},
		{"uv", "tool", "install", "https://example.invalid/tool.whl", "--target", target, "--bin-dir", bin},
	} {
		root, err := New(io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := PreparePresentation(root, args); err == nil {
			t.Fatalf("unsupported tool input passed preflight: %q", args)
		}
	}
}
