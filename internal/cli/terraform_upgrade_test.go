package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestTerraformUpgradeStaticAdmission(t *testing.T) {
	for _, args := range [][]string{
		{"terraform", "init", "-upgrade"},
		{"--output", "json", "terraform", "init", "--upgrade", "--adopt"},
	} {
		var out, diagnostic bytes.Buffer
		root, err := cli.New(&out, &diagnostic)
		if err != nil {
			t.Fatal(err)
		}
		source, err := cli.PreparePresentation(root, args)
		if err != nil || source != "terraform" {
			t.Fatalf("supported current-project upgrade was not admitted before composition: %q %v", source, err)
		}
	}
}

func TestTerraformUpgradeHelpDescribesItsOwnSelectionInEveryLanguage(t *testing.T) {
	for _, lang := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		var out, diag bytes.Buffer
		root, _ := cli.New(&out, &diag)
		args := []string{"--lang", lang, "terraform", "init", "-upgrade", "--help"}
		root.SetArgs(args)
		if _, err := cli.PreparePresentation(root, args); err != nil {
			t.Fatal(err)
		}
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "-upgrade") || strings.Contains(out.String(), "only-if-needed") || lang == "ko" && !strings.Contains(out.String(), "최신 안정 버전") {
			t.Fatal("wrong ecosystem help", lang, out.String())
		}
	}
}

type terraformUpgradeCLI struct{ cargoUpdateCLI }

func (f *terraformUpgradeCLI) Upgrade(ctx context.Context, install domain.InstallContext, adopt bool) (domain.ProjectVerifiedSet, error) {
	return f.Update(ctx, install, "", adopt)
}

func TestTerraformUpgradeRejectsUnsupportedArgsBeforeComposition(t *testing.T) {
	for _, tail := range [][]string{
		{"-upgrade=false"}, {"--upgrade=true"}, {"-upgrade", "--upgrade"},
		{"-upgrade", "hashicorp/random@3.7.2"}, {"--adopt", "hashicorp/random@3.7.2"},
		{"-upgrade", "-plugin-dir=/tmp/foreign"}, {"-upgrade", "--lockfile=readonly"},
		{"-upgrade", "--backend=false"}, {"-upgrade", "--target", "/tmp/foreign"},
		{"-upgrade", "--", "--upgrade"}, {"-upgrade", "--source", "mirror"},
	} {
		var out, diag bytes.Buffer
		root, _ := cli.New(&out, &diag)
		f := &terraformUpgradeCLI{}
		if err := cli.AddTerraformUpgrade(root, f); err != nil {
			t.Fatal(err)
		}
		args := append([]string{"terraform", "init"}, tail...)
		if _, err := cli.PreparePresentation(root, args); err == nil || f.calls != 0 {
			t.Fatal("unsupported argv reached upgrade", args, err)
		}
	}
}

func TestTerraformUpgradeKeepsNativeFlagEnrollmentAndExactInit(t *testing.T) {
	for _, lang := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		var out, diag bytes.Buffer
		root, _ := cli.New(&out, &diag)
		f := &terraformUpgradeCLI{}
		primary := errors.New("trusted refusal")
		f.failure = primary
		if err := cli.AddTerraformUpgrade(root, f); err != nil {
			t.Fatal(err)
		}
		args := []string{"--lang", lang, "--output", "json", "terraform", "init", "-upgrade", "--adopt"}
		root.SetArgs(args)
		if source, err := cli.PreparePresentation(root, args); err != nil || source != "terraform" {
			t.Fatal(source, err)
		}
		if err := root.ExecuteContext(context.Background()); !errors.Is(err, primary) {
			t.Fatal(err)
		}
		if f.calls != 1 || !f.adopt || f.selector != "" || !f.install.Valid() {
			t.Fatal("current-project enrollment intent lost")
		}
		if !strings.Contains(out.String(), `"source_id":"terraform-registry"`) || !strings.Contains(out.String(), `"promotion_status":"NOT_COMPLETED"`) {
			t.Fatal(out.String())
		}
	}
	var out, diag bytes.Buffer
	root, _ := cli.New(&out, &diag)
	exact := &terraformResolverFixture{}
	if err := cli.AddTerraformInit(root, exact); err != nil {
		t.Fatal(err)
	}
	f := &terraformUpgradeCLI{}
	if err := cli.AddTerraformUpgrade(root, f); err != nil {
		t.Fatal(err)
	}
	args := []string{"terraform", "init", "hashicorp/random@3.7.2"}
	root.SetArgs(args)
	if _, err := cli.PreparePresentation(root, args); err != nil {
		t.Fatal(err)
	}
	if err := root.ExecuteContext(context.Background()); err != nil || !exact.called || f.calls != 0 {
		t.Fatal("exact init changed", err)
	}
	var output, diagnostic bytes.Buffer
	other, _ := cli.New(&output, &diagnostic)
	if _, err := cli.PreparePresentation(other, []string{"--output", "json", "terraform", "init", "hashicorp/random@3.7.2"}); err == nil {
		t.Fatal("old exact-init output contract changed")
	}
}

func TestTerraformUpgradeReportSeparatesPolicyAndPublication(t *testing.T) {
	for _, kind := range []string{"completed", "policy-block", "publication-failed"} {
		for _, output := range []string{"json", "human"} {
			for _, lang := range []string{"en", "ko"} {
				t.Run(kind+"/"+output+"/"+lang, func(t *testing.T) {
					decision := domain.DecisionAllow
					if kind == "policy-block" {
						decision = domain.DecisionBlock
					}
					inspected, policy := projectUpdateFactsForSource(t, decision, "terraform-registry")
					f := &terraformUpgradeCLI{}
					primary := errors.New("trusted policy/publication refusal")
					if kind == "completed" {
						var err error
						f.set, err = domain.NewProjectVerifiedSet(inspected, policy)
						if err != nil {
							t.Fatal(err)
						}
					} else {
						f.failure = cargoFailedUpdateFacts{inspected, policy, primary}
					}
					var out, diag bytes.Buffer
					root, _ := cli.New(&out, &diag)
					if err := cli.AddTerraformUpgrade(root, f); err != nil {
						t.Fatal(err)
					}
					args := []string{"--lang", lang, "--output", output, "--color", "never", "terraform", "init", "-upgrade"}
					root.SetArgs(args)
					if _, err := cli.PreparePresentation(root, args); err != nil {
						t.Fatal(err)
					}
					err := root.ExecuteContext(context.Background())
					if (err == nil) != (kind == "completed") || kind != "completed" && !errors.Is(err, primary) {
						t.Fatal(err)
					}
					if output == "json" {
						var doc struct {
							Source    string `json:"source_id"`
							Status    string `json:"operation_status"`
							Promotion string `json:"promotion_status"`
							Entries   []struct {
								Checks   []json.RawMessage `json:"checks"`
								Evidence []json.RawMessage `json:"evidence_references"`
							} `json:"entries"`
							Policy struct {
								Decision string `json:"decision"`
							} `json:"policy"`
						}
						if err := json.Unmarshal(out.Bytes(), &doc); err != nil || doc.Source != "terraform-registry" || len(doc.Entries) != 1 || len(doc.Entries[0].Checks) != 3 || len(doc.Entries[0].Evidence) != 3 || doc.Policy.Decision != string(decision) {
							t.Fatal(out.String(), err)
						}
						if (doc.Status == "COMPLETED") != (kind == "completed") || (doc.Promotion == "COMPLETED") != (kind == "completed") {
							t.Fatal("inspection became false publication success", out.String())
						}
					} else {
						if !strings.Contains(out.String(), "hashicorp/random@3.7.2") || strings.Contains(out.String(), "cargo build") || strings.Contains(out.String(), "go build") || lang == "ko" && !strings.Contains(out.String(), "Provider 업데이트:") {
							t.Fatal(out.String())
						}
					}
				})
			}
		}
	}
}
