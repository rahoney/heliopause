package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
)

func TestPyPIUpgradeCLIUsesNativeIntentAndReportsUpdatePreparationFailure(t *testing.T) {
	for _, flag := range []string{"--upgrade", "-U"} {
		var out, diag bytes.Buffer
		root, _ := cli.New(&out, &diag)
		base, _ := installCLIFixture(t, policy.M4{}, nil)
		preparation := &cliUpdatePreparation{}
		updater, _ := application.NewInstallUpdateService(base, preparation)
		if err := cli.AddPyPIInstallSourcesAndUpdates(root, map[string]cli.Installer{"pypi": base}, map[string]cli.Installer{"pypi": updater}); err != nil {
			t.Fatal(err)
		}
		root.SetArgs([]string{"--lang", "ko", "--output", "json", "pip", "install", flag, "fixture>=1,<3", "--target", "/tmp/haa-pypi-upgrade-cli"})
		if err := root.ExecuteContext(context.Background()); err == nil {
			t.Fatal("failed upgrade returned success")
		}
		if preparation.calls != 1 || preparation.source.String() != "pypi" || preparation.selector != "fixture>=1,<3" || preparation.adopt || preparation.install.Mode() != domain.InstallPythonVenv {
			t.Fatal("native intent lost", preparation)
		}
		var result struct {
			Operation string `json:"operation"`
			Status    string `json:"operation_status"`
			Error     struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal(out.Bytes(), &result) != nil || result.Operation != "UPDATE" || result.Status != "FAILED" || result.Error.Code != "UPDATE_ADOPTION_REQUIRED" || bytes.Contains(out.Bytes(), []byte("\x1b[")) {
			t.Fatal(out.String())
		}
	}
}

func TestPyPIUpgradeUnsupportedInputRejectsBeforeComposition(t *testing.T) {
	t.Setenv("VIRTUAL_ENV", "/tmp/haa-pypi-upgrade-cli")
	for _, args := range [][]string{
		{"pip", "install", "-U", "a", "b"},
		{"pip", "install", "-U", "a[extra]"},
		{"pip", "install", "-U", "https://example.invalid/a.whl"},
		{"pip", "install", "-U", "a", "--upgrade-strategy", "eager"},
		{"pip", "install", "-U", "a", "--index-url", "https://example.invalid"},
		{"pip", "install", "-U", "a", "--source", "pytorch:cpu"},
		{"pip", "install", "-U", "a", "--target", "relative"},
		{"pip", "install", "-U", "a", "--target", "/tmp/with,comma"},
		{"pip", "install", "a", "--adopt"},
	} {
		var out, diag bytes.Buffer
		root, _ := cli.New(&out, &diag)
		if _, err := cli.PreparePresentation(root, args); err == nil {
			t.Fatal("unsupported upgrade reached runtime composition", args)
		}
	}
	for _, lang := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		var out, diag bytes.Buffer
		root, _ := cli.New(&out, &diag)
		root.SetArgs([]string{"--lang", lang, "pip", "install", "--help"})
		if err := root.ExecuteContext(context.Background()); err != nil || !strings.Contains(out.String(), "--upgrade") || !strings.Contains(out.String(), "only-if-needed") || lang == "ko" && !strings.Contains(out.String(), "업데이트") {
			t.Fatal(lang, out.String(), err)
		}
	}
}
