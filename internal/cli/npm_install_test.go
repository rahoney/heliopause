package cli_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestNPMInstallDefaultsToCurrentProject(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	root, err := cli.New(&stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	installer := &npmCLIInstaller{}
	if err := cli.AddNPMInstall(root, installer); err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"npm", "install", "demo-package@1.0.0"})
	_ = root.ExecuteContext(context.Background())
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if installer.request.Context().Mode() != domain.InstallNPMProject {
		t.Fatalf("install mode = %s", installer.request.Context().Mode())
	}
	if installer.request.Context().Target().String() != workingDirectory {
		t.Fatalf("install target = %q, want %q", installer.request.Context().Target(), workingDirectory)
	}
	if installer.request.Reference().Locator() != "demo-package@1.0.0" {
		t.Fatalf("reference = %q", installer.request.Reference().Locator())
	}
}

type npmCLIInstaller struct{ request application.InstallRequest }

func (i *npmCLIInstaller) Install(_ context.Context, request application.InstallRequest) (application.InstallOutcome, error) {
	i.request = request
	return application.InstallOutcome{}, nil
}
