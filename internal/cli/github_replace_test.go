package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/policy"
)

func TestGitHubReplacementStaticAdmission(t *testing.T) {
	var out, diagnostic bytes.Buffer
	root, err := cli.New(&out, &diagnostic)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"github", "install", "owner/repo@v2#tool.tar.gz", "--target", "/tmp/haa-managed-github", "--replace"}
	if source, err := cli.PreparePresentation(root, args); err != nil || source != "github" {
		t.Fatalf("explicit managed replacement not admitted before composition: %q %v", source, err)
	}
}

func TestGitHubReplacementRejectsUnsupportedIntentBeforeRuntime(t *testing.T) {
	for _, args := range [][]string{
		{"owner/repo@v2#tool", "--replace"},
		{"owner/repo@v2#tool", "--replace", "--target", "relative"},
		{"owner/repo@v2#tool", "--replace=false", "--target", "/tmp/target"},
		{"owner/repo@v2#tool", "--replace=true", "--target", "/tmp/target"},
		{"owner/repo@v2#tool", "--replace", "--replace", "--target", "/tmp/target"},
		{"owner/repo@v2#tool", "--adopt", "--target", "/tmp/target"},
		{"owner/repo@latest#tool", "--replace", "--target", "/tmp/target"},
	} {
		var out, diagnostic bytes.Buffer
		root, _ := cli.New(&out, &diagnostic)
		if _, err := cli.PreparePresentation(root, append([]string{"github", "install"}, args...)); err == nil {
			t.Fatal("unsupported replacement admitted", args)
		}
	}
}

type githubReplacementInstaller struct {
	request application.InstallRequest
	calls   int
}

func (f *githubReplacementInstaller) Install(_ context.Context, request application.InstallRequest) (application.InstallOutcome, error) {
	f.request = request
	f.calls++
	return application.InstallOutcome{}, nil
}

func TestGitHubReplacementBindsSeparateModeAndKeepsOldInstall(t *testing.T) {
	for _, replace := range []bool{false, true} {
		var out, diagnostic bytes.Buffer
		root, _ := cli.New(&out, &diagnostic)
		initial, update := &githubReplacementInstaller{}, &githubReplacementInstaller{}
		if err := cli.AddGitHubReleaseInstall(root, githubrelease.ParseReference, initial); err != nil {
			t.Fatal(err)
		}
		if err := cli.AddGitHubReleaseReplacement(root, update); err != nil {
			t.Fatal(err)
		}
		args := []string{"github", "install", "Owner/Repo@v2#tool.tar.gz", "--target", "/tmp/github-managed"}
		if replace {
			args = append(args, "--replace")
		}
		root.SetArgs(args)
		if _, err := cli.PreparePresentation(root, args); err != nil {
			t.Fatal(err)
		}
		_ = root.ExecuteContext(context.Background())
		selected := initial
		if replace {
			selected = update
			if initial.calls != 0 || selected.request.Context().Mode() != domain.InstallGitHubAsset || selected.request.Operation() != domain.OperationUpdate {
				t.Fatal(initial.calls, selected.request)
			}
		} else if update.calls != 0 || !selected.request.Context().RequiresNewTarget() {
			t.Fatal("initial route changed")
		}
		if selected.calls != 1 || selected.request.Reference().Locator() != "owner/repo@v2#tool.tar.gz" || selected.request.Context().Target().String() != "/tmp/github-managed" {
			t.Fatal(selected.request)
		}
	}
}

func TestGitHubReplacementHelpExposesExactScope(t *testing.T) {
	for _, language := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		var out, diagnostic bytes.Buffer
		root, _ := cli.New(&out, &diagnostic)
		args := []string{"--lang", language, "github", "install", "--help"}
		root.SetArgs(args)
		if _, err := cli.PreparePresentation(root, args); err != nil {
			t.Fatal(err)
		}
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "--replace") || language == "en" && !strings.Contains(out.String(), "independent HAA approval") || language == "ko" && !strings.Contains(out.String(), "독립 HAA 승인") {
			t.Fatal(language, out.String())
		}
	}
}

type githubOwnershipRefusal struct{}

func (githubOwnershipRefusal) BeginInstallUpdate(context.Context, domain.SourceID, string, domain.InstallContext, bool) (ports.InstallUpdateSession, error) {
	return nil, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", errors.New("controlled refusal"))
}

func TestGitHubReplacementReportsOwnershipFailureWithoutAdoptionAdvice(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		for _, output := range []string{"human", "json"} {
			var out, diagnostic bytes.Buffer
			root, _ := cli.New(&out, &diagnostic)
			base, _ := installCLIFixture(t, policy.M4{}, nil)
			updater, err := application.NewInstallUpdateService(base, githubOwnershipRefusal{})
			if err != nil {
				t.Fatal(err)
			}
			if err := cli.AddGitHubReleaseInstall(root, githubrelease.ParseReference, base); err != nil {
				t.Fatal(err)
			}
			if err := cli.AddGitHubReleaseReplacement(root, updater); err != nil {
				t.Fatal(err)
			}
			args := []string{"--lang", language, "--output", output, "--color", "never", "--progress", "never", "github", "install", "owner/repo@v2#tool.zip", "--target", "/tmp/github-owned-report", "--replace"}
			root.SetArgs(args)
			if _, err := cli.PreparePresentation(root, args); err != nil {
				t.Fatal(err)
			}
			if err := root.ExecuteContext(context.Background()); err == nil {
				t.Fatal("ownership refusal reported as success")
			}
			if strings.Contains(out.String(), "--adopt") || strings.Contains(out.String(), "\x1b") {
				t.Fatal(out.String())
			}
			if output == "json" {
				var doc struct {
					Operation string `json:"operation"`
					Status    string `json:"operation_status"`
					Promotion string `json:"promotion_status"`
					Error     struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if json.Unmarshal(out.Bytes(), &doc) != nil || doc.Operation != "UPDATE" || doc.Status != "FAILED" || doc.Promotion != "FAILED" || doc.Error.Code != "UPDATE_OWNERSHIP_INVALID" {
					t.Fatal(out.String())
				}
			} else if !strings.Contains(out.String(), "UPDATE_OWNERSHIP_INVALID") || language == "ko" && !strings.Contains(out.String(), "새 관리 대상") || language == "en" && !strings.Contains(out.String(), "new managed target") {
				t.Fatal(out.String())
			}
		}
	}
}
