package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestPyPIInstallParsesOnlyCanonicalReferenceAndTarget(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	root, err := cli.New(&stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	installer := &pypiCLIInstaller{}
	profile, _ := artifactpypi.PyTorchProfile("cpu")
	if err := cli.AddPyPIInstallSources(root, map[string]cli.Installer{"pypi": installer, profile.Source().String(): installer}); err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"pip", "install", "Demo_Package@1.0", "--target", "/tmp/haa-pypi-cli-target"})
	// The minimal installer intentionally has no completed result to render,
	// but receives the parsed request before presentation is attempted.
	_ = root.ExecuteContext(context.Background())
	if installer.request.Reference().Source().String() != "pypi" || installer.request.Reference().Locator() != "demo-package@1.0" || installer.request.Context().Target().String() != "/tmp/haa-pypi-cli-target" {
		t.Fatalf("request = %#v", installer.request)
	}
	if installer.request.Context().Mode() != domain.InstallPythonVenv {
		t.Fatalf("install mode = %s", installer.request.Context().Mode())
	}
	root.SetArgs([]string{"pip", "install", "torch@2.0.0+cpu", "--source", "pytorch:cpu", "--target", "/tmp/haa-pytorch-cli-target"})
	_ = root.ExecuteContext(context.Background())
	if installer.request.Reference().Source().String() != "pytorch-cpu" || installer.request.Reference().Locator() != "torch@2.0.0+cpu" {
		t.Fatalf("PyTorch request = %#v", installer.request)
	}
	root.SetArgs([]string{"pip", "install", "demo-package[extra]", "--target", "/tmp/other"})
	if err := root.ExecuteContext(context.Background()); err == nil || !strings.Contains(err.Error(), "PyPI") {
		t.Fatalf("unsafe reference error = %v", err)
	}
}

func TestPyPIInstallWithoutActiveVenvFailsClosed(t *testing.T) {
	t.Setenv("VIRTUAL_ENV", "")

	var stdout, stderr bytes.Buffer
	root, err := cli.New(&stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if err := cli.AddPyPIInstall(root, &pypiCLIInstaller{}); err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"pip", "install", "demo-package@1.0"})
	if err := root.ExecuteContext(context.Background()); err == nil || !strings.Contains(err.Error(), "active virtual environment") {
		t.Fatalf("error = %v, want missing active venv", err)
	}
}

type pypiCLIInstaller struct{ request application.InstallRequest }

func (p *pypiCLIInstaller) Install(_ context.Context, request application.InstallRequest) (application.InstallOutcome, error) {
	p.request = request
	return application.InstallOutcome{}, nil
}
