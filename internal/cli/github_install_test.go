package cli_test

import (
	"bytes"
	"context"
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

type githubInstaller struct{ request application.InstallRequest }

func (i *githubInstaller) Install(_ context.Context, request application.InstallRequest) (application.InstallOutcome, error) {
	i.request = request
	return application.InstallOutcome{}, nil
}
