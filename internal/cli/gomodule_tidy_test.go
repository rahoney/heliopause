package cli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func (f *goUpgradeCLI) Tidy(ctx context.Context, install domain.InstallContext, adopt bool) (domain.ProjectVerifiedSet, error) {
	return f.Update(ctx, install, "", adopt)
}

func TestGoTidyPreflightAndFailureReport(t *testing.T) {
	for _, lang := range []string{"en", "ko"} {
		var out, diag bytes.Buffer
		root, err := cli.New(&out, &diag)
		if err != nil {
			t.Fatal(err)
		}
		primary := errors.New("trusted source refusal")
		f := &goUpgradeCLI{}
		f.failure = primary
		if err := cli.AddGoModuleTidy(root, f); err != nil {
			t.Fatal(err)
		}
		args := []string{"--lang", lang, "--output", "json", "go", "mod", "tidy", "--adopt"}
		if source, err := cli.PreparePresentation(root, args); err != nil || source != "go" {
			t.Fatalf("preflight: %s %v", source, err)
		}
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); !errors.Is(err, primary) {
			t.Fatal(err)
		}
		if f.calls != 1 || !f.adopt || !f.install.Valid() || f.selector != "" {
			t.Fatal("source/adoption request changed")
		}
		for _, field := range []string{`"schema_version":"helox.project-tidy-result/v1"`, `"operation":"TIDY"`, `"inspection_scope":"DEPENDENCY_CONTROLS"`, `"promotion_status":"NOT_COMPLETED"`} {
			if !strings.Contains(out.String(), field) {
				t.Fatal(out.String())
			}
		}
	}
	for _, tail := range [][]string{{"extra"}, {"-e"}, {"-go", "1.27"}, {"--target", "/tmp/foreign"}, {"--modfile", "foreign"}} {
		var out, diag bytes.Buffer
		root, _ := cli.New(&out, &diag)
		f := &goUpgradeCLI{}
		if err := cli.AddGoModuleTidy(root, f); err != nil {
			t.Fatal(err)
		}
		args := append([]string{"go", "mod", "tidy"}, tail...)
		if _, err := cli.PreparePresentation(root, args); err == nil || f.calls != 0 {
			t.Fatalf("unsupported input reached composition: %v", args)
		}
	}
}
