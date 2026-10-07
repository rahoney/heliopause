package gomodule

import (
	"strings"
	"testing"
)

func TestProjectModRejectsReplacementBeforeResolverExecution(t *testing.T) {
	valid := "module example.com/app\n\ngo 1.26.8\nrequire example.com/dep v1.2.3\n"
	if err := ValidateProjectMod([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateProjectMod([]byte(strings.Replace(valid, "example.com/app", "local-app", 1))); err != nil {
		t.Fatalf("local main module identity was treated as an acquired dependency: %v", err)
	}
	for name, body := range map[string]string{
		"relative":             valid + "replace example.com/dep => ../host-secret\n",
		"absolute quoted":      valid + "replace example.com/dep => \"/host/private directory\"\n",
		"block":                valid + "replace (\n example.com/dep => /host/private\n)\n",
		"other public source":  valid + "replace example.com/dep => other.example/dep v1.2.3\n",
		"noncanonical version": "module example.com/app\nrequire example.com/dep v2.0.0\n",
		"invalid syntax":       valid + "replace (",
		"empty":                "",
		"oversize":             strings.Repeat(" ", MaxProjectControlBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateProjectMod([]byte(body)); err == nil {
				t.Fatal("accepted unsafe project module controls")
			}
		})
	}
}
