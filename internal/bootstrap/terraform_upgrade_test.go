package bootstrap

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestTerraformUpgradeHelpIsStaticBeforeRuntimeComposition(t *testing.T) {
	for _, lang := range []string{"en", "ko"} {
		var out, diag bytes.Buffer
		if err := Run(context.Background(), []string{"--lang", lang, "terraform", "init", "-upgrade", "--help"}, &out, &diag); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "helox terraform init -upgrade") || strings.Contains(out.String(), "only-if-needed") {
			t.Fatal("native argv/help lost before runtime dispatch", out.String())
		}
	}
}
