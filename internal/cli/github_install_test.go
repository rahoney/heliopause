package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/cli"
)

func TestGitHubReleaseInstallParsesExactReferenceAndTarget(t *testing.T) {
	var out, errOut bytes.Buffer
	root, err := cli.New(&out, &errOut)
	if err != nil {
		t.Fatal(err)
	}
	installer := &githubInstaller{}
	if err := cli.AddGitHubReleaseInstall(root, githubrelease.ParseReference, installer); err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"github", "install", "Owner/Repo@v1.2.3#tool.zip", "--target", "/tmp/haa-github-target"})
	_ = root.ExecuteContext(context.Background())
	if installer.request.Reference().Locator() != "owner/repo@v1.2.3#tool.zip" || installer.request.Context().Target().String() != "/tmp/haa-github-target" {
		t.Fatalf("request=%#v", installer.request)
	}
}

func TestGitHubReleaseInstallDefaultsToScopedNonExistingTarget(t *testing.T) {
	var out, errOut bytes.Buffer
	root, err := cli.New(&out, &errOut)
	if err != nil {
		t.Fatal(err)
	}
	installer := &githubInstaller{}
	if err := cli.AddGitHubReleaseInstall(root, githubrelease.ParseReference, installer); err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"github", "install", "Owner/Repo@v1.2.3#tool.zip"})
	_ = root.ExecuteContext(context.Background())

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	target := installer.request.Context().Target().String()
	if filepath.Dir(target) != workingDirectory {
		t.Fatalf("default target parent = %q, want %q", filepath.Dir(target), workingDirectory)
	}
	if !strings.HasPrefix(filepath.Base(target), ".helox-github-owner-repo-v1.2.3-tool.zip-") {
		t.Fatalf("default target = %q", target)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("default target exists or could not be checked: %v", err)
	}
}

type githubInstaller struct{ request application.InstallRequest }

func (i *githubInstaller) Install(_ context.Context, request application.InstallRequest) (application.InstallOutcome, error) {
	i.request = request
	return application.InstallOutcome{}, nil
}
