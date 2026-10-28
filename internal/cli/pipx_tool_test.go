package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestPipxClientRoutesDistinctCanonicalInstallAndBareUpdate(t *testing.T) {
	for _, step := range []struct {
		operation, input string
		adopt            bool
	}{{"install", "Demo_Package@1.0", false}, {"install", "demo-package>=1,<2", false}, {"upgrade", "demo-package", false}, {"upgrade", "demo-package", true}} {
		target, bin := uvToolCLIPaths(t)
		root, err := New(&bytes.Buffer{}, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		recorder := &uvClientRequestRecorder{}
		if step.operation == "install" {
			err = AddPipxToolInstall(root, recorder)
		} else {
			err = AddPipxToolUpgrade(root, recorder)
		}
		if err != nil {
			t.Fatal(err)
		}
		args := []string{"--output", "json", "--lang", "ko", "pipx", step.operation, step.input, "--target", target, "--bin-dir", bin}
		if step.adopt {
			args = append(args, "--adopt")
		}
		if source, err := PreparePresentation(root, args); err != nil || source != "pypi" || SelectedPackageClient(root) != PipxPackageClient || SelectedPackageClientOperation(root) != step.operation {
			t.Fatal("pipx route/client/operation changed", err)
		}
		root.SetArgs(args)
		_ = root.ExecuteContext(context.Background())
		if len(recorder.requests) != 1 {
			t.Fatal("pipx request not routed once")
		}
		request := recorder.requests[0]
		requirement, err := artifactpypi.ParseUpgradeRequirement(step.input)
		if err != nil {
			t.Fatal(err)
		}
		if request.Context().Mode() != domain.InstallPythonVenv || request.Context().Target().String() != target || request.Reference().Locator() != requirement.Requirement() || request.Source() != artifactpypi.PublicPyPIProfile().Source() {
			t.Fatal("canonical request changed")
		}
		if step.operation == "upgrade" {
			expected, err := application.NewInstallUpdateRequest(request.Source(), step.input, request.Context(), step.adopt)
			if err != nil || request != expected {
				t.Fatal("pipx update/adoption intent changed", err)
			}
		} else if request.Operation() != domain.OperationInstall {
			t.Fatal("fresh request became update")
		}
	}
}

func TestPipxClientRefusesUnsafeImplicitAndNativeOptionsBeforeBinding(t *testing.T) {
	target, bin := uvToolCLIPaths(t)
	alias := filepath.Join(filepath.Dir(target), "alias")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VIRTUAL_ENV", target)
	t.Setenv("PIPX_HOME", filepath.Dir(target))
	t.Setenv("PIPX_BIN_DIR", bin)
	for _, op := range []string{"install", "upgrade"} {
		for _, tail := range [][]string{
			{"demo-package"}, {"demo-package", "--target", target}, {"demo-package", "--bin-dir", bin},
			{"demo-package", "--target", alias, "--bin-dir", bin}, {"demo-package", "--target", target, "--bin-dir", target},
			{"other", "--target", target, "--bin-dir", bin},
			{"demo-package", "--target", target, "--target", target, "--bin-dir", bin},
			{"demo-package", "--target", target, "--bin-dir", bin, "--bin-dir", bin},
			{"demo-package", "--target", target, "--bin-dir", bin, "--adopt", "--adopt=false"},
			{"demo-package", "--target", target, "--bin-dir", bin, "--backend", "uv"},
			{"demo-package", "--target", target, "--bin-dir", bin, "--python", "/foreign/python"},
			{"demo-package", "--target", target, "--bin-dir", bin, "--pip-args=--index-url=foreign"},
			{"demo-package", "--target", target, "--bin-dir", bin, "--global"},
			{"https://example.invalid/a.whl", "--target", target, "--bin-dir", bin},
		} {
			root, err := New(io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := PreparePresentation(root, append([]string{"pipx", op}, tail...)); err == nil {
				t.Fatal("unsupported pipx input accepted", op, tail)
			}
		}
	}
	root, _ := New(io.Discard, io.Discard)
	if _, err := PreparePresentation(root, []string{"pipx", "install", "demo-package", "--target", target, "--bin-dir", bin, "--adopt"}); err == nil {
		t.Fatal("fresh install accepted adoption")
	}
	for _, input := range []string{"Demo_Package", "demo-package==1", "demo-package>=1", "demo-package[extra]"} {
		root, _ := New(io.Discard, io.Discard)
		if _, err := PreparePresentation(root, []string{"pipx", "upgrade", input, "--target", target, "--bin-dir", bin}); err == nil {
			t.Fatal("non-bare/canonical upgrade accepted", input)
		}
	}
	for _, op := range []string{"run", "inject", "ensurepath"} {
		root, _ := New(io.Discard, io.Discard)
		if _, err := PreparePresentation(root, []string{"pipx", op, "demo-package"}); err == nil {
			t.Fatal("unimplemented operation accepted", op)
		}
	}
}
