package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"strings"
	"testing"
)

type goUpgradeCLI struct{ cargoUpdateCLI }

func (f *goUpgradeCLI) Upgrade(ctx context.Context, install domain.InstallContext, selector string, adopt bool) (domain.ProjectVerifiedSet, error) {
	return f.Update(ctx, install, selector, adopt)
}

func TestGoUpgradeRejectsUnsupportedInputBeforeComposition(t *testing.T) {
	for _, tail := range [][]string{{"-u"}, {"-u", "example.com/module", "example.com/other"}, {"-u=patch", "example.com/module"}, {"-u", "example.com/module@v1.2.3"}, {"-u", "./..."}, {"-u", "example.com/module/..."}, {"-u", "go"}, {"-u", "toolchain"}, {"-u", "https://example.com/module"}, {"-u", "-modfile=foreign", "example.com/module"}, {"--adopt", "example.com/module@v1.2.3"}, {"-u", "--target", "/tmp/foreign", "example.com/module"}, {"-u", "-t", "example.com/module"}} {
		for _, preflight := range []bool{false, true} {
			var out, diag bytes.Buffer
			root, _ := cli.New(&out, &diag)
			f := &goUpgradeCLI{}
			if err := cli.AddGoModuleUpgrade(root, f); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"go", "get"}, tail...)
			var err error
			if preflight {
				_, err = cli.PreparePresentation(root, args)
			} else {
				root.SetArgs(args)
				err = root.ExecuteContext(context.Background())
			}
			if err == nil || f.calls != 0 {
				t.Fatalf("unsupported input reached upgrade: %v %v", args, err)
			}
		}
	}
}

func TestGoUpgradePreflightKeepsOneNamedEnrollmentIntent(t *testing.T) {
	for _, lang := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		var out, diag bytes.Buffer
		root, _ := cli.New(&out, &diag)
		f := &goUpgradeCLI{}
		primary := errors.New("trusted refusal")
		f.failure = primary
		if err := cli.AddGoModuleUpgrade(root, f); err != nil {
			t.Fatal(err)
		}
		args := []string{"--lang", lang, "--output", "json", "go", "get", "-u", "example.com/module", "--adopt"}
		if source, err := cli.PreparePresentation(root, args); err != nil || source != "go" {
			t.Fatal(source, err)
		}
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); !errors.Is(err, primary) {
			t.Fatal(err)
		}
		if f.calls != 1 || !f.adopt || f.selector != "example.com/module" || !f.install.Valid() {
			t.Fatal("intent lost")
		}
		if !strings.Contains(out.String(), `"source_id":"go-proxy"`) || !strings.Contains(out.String(), `"promotion_status":"NOT_COMPLETED"`) {
			t.Fatal(out.String())
		}
	}
	var out, diag bytes.Buffer
	root, _ := cli.New(&out, &diag)
	if _, err := cli.PreparePresentation(root, []string{"--output", "json", "go", "get", "example.com/module@v1.2.3"}); err == nil {
		t.Fatal("old exact get JSON contract changed")
	}
}

func TestGoUpgradeReportSeparatesInspectionAndCompletedPublication(t *testing.T) {
	for _, kind := range []string{"completed", "policy-block", "publication-failed"} {
		for _, output := range []string{"json", "human"} {
			for _, lang := range []string{"en", "ko"} {
				t.Run(kind+"/"+output+"/"+lang, func(t *testing.T) {
					decision := domain.DecisionAllow
					if kind == "policy-block" {
						decision = domain.DecisionBlock
					}
					inspected, policy := projectUpdateFactsForSource(t, decision, "go-proxy")
					f := &goUpgradeCLI{}
					primary := errors.New("trusted publication or policy failure")
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
					if err := cli.AddGoModuleUpgrade(root, f); err != nil {
						t.Fatal(err)
					}
					root.SetArgs([]string{"--lang", lang, "--output", output, "--color", "never", "go", "get", "-u", "example.com/module"})
					err := root.ExecuteContext(context.Background())
					if (err == nil) != (kind == "completed") || kind != "completed" && !errors.Is(err, primary) {
						t.Fatal(err)
					}
					if output == "json" {
						var doc struct {
							Schema    string `json:"schema_version"`
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
						if err := json.Unmarshal(out.Bytes(), &doc); err != nil || doc.Schema != "helox.project-update-result/v1" || len(doc.Entries) != 1 || len(doc.Entries[0].Checks) != 2 || len(doc.Entries[0].Evidence) != 2 || doc.Policy.Decision != string(decision) || bytes.Contains(out.Bytes(), []byte("\x1b[")) {
							t.Fatal(out.String(), err)
						}
						if (doc.Promotion == "COMPLETED") != (kind == "completed") || (doc.Status == "COMPLETED") != (kind == "completed") {
							t.Fatal("inspection ALLOW became publication success", out.String())
						}
					} else {
						if !strings.Contains(out.String(), "example.com/module@v1.2.3") || !strings.Contains(out.String(), "helox go build") || lang == "ko" && (!strings.Contains(out.String(), "의존성 설정 업데이트:") || !strings.Contains(out.String(), "정적 검사")) {
							t.Fatal(out.String())
						}
					}
				})
			}
		}
	}
}
