package npm

import (
	"os"
	"strings"
	"testing"
)

func nodeShimVector(t *testing.T, target string) string {
	t.Helper()
	body, err := os.ReadFile("testdata/pnpm-node-bin-shim.normalized")
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(body), "{{HAA_BIN_TARGET}}", target)
}

func TestPNPMNodeBinShimCompleteCodeAndModulePaths(t *testing.T) {
	shim := "node_modules/.pnpm/node_modules/.bin/tool"
	target := "node_modules/.pnpm/tool@1.0.0/node_modules/tool/cli.js"
	relative := "../../tool@1.0.0/node_modules/tool/cli.js"
	code := nodeShimVector(t, relative)
	if paths, err := ParsePNPMNodeBinShim([]byte(code), shim, target); err != nil || len(paths) != 0 {
		t.Fatal("known complete Node template rejected", paths, err)
	}
	value := "$basedir_abs/../../tool@1.0.0/node_modules/tool/node_modules:$basedir_abs/../../tool@1.0.0/node_modules:$basedir_abs/.."
	block := "if [ -z \"$NODE_PATH\" ]; then\n  export NODE_PATH=\"" + value + "\"\nelse\n  export NODE_PATH=\"" + value + ":$NODE_PATH\"\nfi\n"
	code = strings.Replace(code, "if [ -n \"$exe\" ]", block+"if [ -n \"$exe\" ]", 1)
	paths, err := ParsePNPMNodeBinShim([]byte(code), shim, target)
	if err != nil || len(paths) != 3 || paths[0] != "node_modules/.pnpm/tool@1.0.0/node_modules/tool/node_modules" || paths[2] != "node_modules/.pnpm/node_modules" {
		t.Fatal("module path block differs", paths, err)
	}
	for _, changed := range []string{
		strings.Replace(code, "# Resolve $0", "# Changed $0", 1),
		strings.Replace(code, "  exec node ", "  exec sh ", 1),
		strings.Replace(code, "  exec node ", "  exec node --eval attacker ", 1),
		"printf attacker\n" + code,
		strings.Replace(code, relative, "../../foreign/cli.js", 1),
		code + "# " + relative + "\n",
		strings.Replace(code, ":$NODE_PATH\"", ":other:$NODE_PATH\"", 1),
		strings.ReplaceAll(code, "$basedir_abs/..\"", "$basedir_abs/../..\""),
		strings.ReplaceAll(code, "$basedir_abs/..:", "$basedir_abs/../../../../outside/node_modules:"),
		strings.ReplaceAll(code, "$basedir_abs/..", "$(attacker)/.."),
		strings.ReplaceAll(code, value, "$basedir_abs/..:$basedir_abs/.."),
	} {
		if _, err := ParsePNPMNodeBinShim([]byte(changed), shim, target); err == nil {
			t.Fatal("altered code or module path data accepted")
		}
	}
	if _, err := ParsePNPMNodeBinShim([]byte(code), shim, strings.Replace(target, "cli.js", "other.js", 1)); err == nil {
		t.Fatal("shim authorized a different expected target")
	}
	for _, bad := range []string{"$basedir_abs/../../../../outside/node_modules", "$basedir_abs/../../tool@1.0.0", "relative/node_modules", "$basedir_abs/$(attacker)/node_modules", "$basedir_abs/..:$basedir_abs/.."} {
		changed := strings.ReplaceAll(code, value, bad)
		if _, err := ParsePNPMNodeBinShim([]byte(changed), shim, target); err == nil {
			t.Fatal("invalid module data accepted despite a matching export pair", bad)
		}
	}
}

func TestPNPMNodeBinShimPathSyntaxAndSlashSemantics(t *testing.T) {
	for _, path := range []string{"../escape", "/absolute", "node_modules/a/../b", "node_modules/$(code)", "node_modules/a\\b", "node_modules/a b", "node_modules/a\nfile"} {
		code := nodeShimVector(t, pnpmBinRelativePath("node_modules/.bin", path))
		if _, err := ParsePNPMNodeBinShim([]byte(code), "node_modules/.bin/tool", path); err == nil {
			t.Fatal("unsafe expected target accepted", path)
		}
	}
	if pnpmBinRelativePath("node_modules/Case/.bin", "node_modules/case/cli.js") != "../../case/cli.js" {
		t.Fatal("guest path case comparison depends on Host rules")
	}
}
