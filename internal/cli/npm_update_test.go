package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/policy"
)

type cliUpdatePreparation struct {
	source   domain.SourceID
	selector string
	install  domain.InstallContext
	adopt    bool
	calls    int
}

func (p *cliUpdatePreparation) BeginInstallUpdate(_ context.Context, source domain.SourceID, selector string, install domain.InstallContext, adopt bool) (ports.InstallUpdateSession, error) {
	p.calls++
	p.source, p.selector, p.install, p.adopt = source, selector, install, adopt
	return nil, ports.NewInstallUpdateFailure("UPDATE_ADOPTION_REQUIRED", errors.New("no prior ownership"))
}

func TestNPMUpdateCLIReportsTypedPreparationFailureAsJSONAndPreservesSelection(t *testing.T) {
	for _, selector := range []string{"", "safe"} {
		var out, diag bytes.Buffer
		root, err := cli.New(&out, &diag)
		if err != nil {
			t.Fatal(err)
		}
		base, _ := installCLIFixture(t, policy.M4{}, nil)
		preparation := &cliUpdatePreparation{}
		updater, err := application.NewInstallUpdateService(base, preparation)
		if err != nil {
			t.Fatal(err)
		}
		if err := cli.AddNPMUpdate(root, updater); err != nil {
			t.Fatal(err)
		}
		args := []string{"--lang", "ko", "--output", "json", "npm", "update", "--adopt", "--target", "/tmp/haa-update-cli-project"}
		if selector != "" {
			args = append(args, selector)
		}
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); err == nil {
			t.Fatal("failed update returned success")
		}
		if preparation.calls != 1 || preparation.source.String() != "npm" || preparation.selector != selector || !preparation.adopt || preparation.install.Mode() != domain.InstallNPMProject || preparation.install.Target().String() != "/tmp/haa-update-cli-project" {
			t.Fatal(preparation)
		}
		var doc struct {
			Operation, OperationID, Status, Target string
			Artifact                               struct {
				Reference struct{ Source, Locator string }
			}
			Error struct{ Code, Message string }
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
			t.Fatal(out.String(), err)
		}
		var operation, status string
		_ = json.Unmarshal(raw["operation"], &operation)
		_ = json.Unmarshal(raw["operation_status"], &status)
		if operation != "UPDATE" || status != "FAILED" || len(raw["operation_id"]) < 28 || bytes.Contains(out.Bytes(), []byte("\x1b[")) {
			t.Fatal(out.String())
		}
		if err := json.Unmarshal(raw["error"], &doc.Error); err != nil || doc.Error.Code != "UPDATE_ADOPTION_REQUIRED" || doc.Error.Message != "Update operation failed." {
			t.Fatal(out.String(), err)
		}
		var artifact struct {
			Reference struct {
				Source  string `json:"source_id"`
				Locator string `json:"locator"`
			} `json:"reference"`
		}
		if json.Unmarshal(raw["artifact"], &artifact) != nil || artifact.Reference.Source != "npm" || artifact.Reference.Locator == "" {
			t.Fatal(out.String())
		}
	}
}

func TestNPMUpdateCLIRejectsUnsupportedInputBeforeUseCaseAndDefaultsToCurrentProject(t *testing.T) {
	for _, args := range [][]string{{"npm", "update", "a", "b"}, {"npm", "update", "a@2.0.0"}, {"npm", "update", "--force"}, {"npm", "update", "--target", "relative"}} {
		var out, diag bytes.Buffer
		root, _ := cli.New(&out, &diag)
		capture := &npmCLIInstaller{}
		_ = cli.AddNPMUpdate(root, capture)
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); err == nil || capture.request.Operation() == domain.OperationUpdate {
			t.Fatal("unsupported update reached workflow", args, err)
		}
	}
	var out, diag bytes.Buffer
	root, _ := cli.New(&out, &diag)
	capture := &npmCLIInstaller{}
	_ = cli.AddNPMUpdate(root, capture)
	root.SetArgs([]string{"npm", "update"})
	_ = root.ExecuteContext(context.Background())
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if capture.request.Operation() != domain.OperationUpdate || capture.request.Context().Mode() != domain.InstallNPMProject || capture.request.Context().Target().String() != cwd {
		t.Fatal(capture.request)
	}
	for _, language := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		var out, diag bytes.Buffer
		root, _ := cli.New(&out, &diag)
		root.SetArgs([]string{"--lang", language, "npm", "update", "--help"})
		if err := root.ExecuteContext(context.Background()); err != nil || !strings.Contains(out.String(), "--adopt") || !strings.Contains(out.String(), "helox npm update [package]") {
			t.Fatal(language, out.String(), err)
		}
		if language == "ko" && !strings.Contains(out.String(), "업데이트") {
			t.Fatal("Korean update help missing")
		}
	}
}

func TestNPMUpdateArgumentsValidatedBeforeRuntimeComposition(t *testing.T) {
	for _, args := range [][]string{{"npm", "update", "a", "b"}, {"npm", "update", "a@1.0.0"}, {"npm", "update", "--target", "relative"}, {"npm", "update", "--force"}} {
		var out, diag bytes.Buffer
		root, _ := cli.New(&out, &diag)
		if _, err := cli.PreparePresentation(root, args); err == nil {
			t.Fatal("unsupported update reached composition", args)
		}
	}
	var out, diag bytes.Buffer
	root, _ := cli.New(&out, &diag)
	if _, err := cli.PreparePresentation(root, []string{"npm", "update", "a", "b", "--help"}); err != nil {
		t.Fatal("static help unexpectedly requires valid operation arguments", err)
	}
}
