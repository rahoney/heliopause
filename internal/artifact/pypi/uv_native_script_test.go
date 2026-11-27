package pypi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type uvNativeLauncherFixture struct {
	Name, Module, Attr, Python string
	Body                       []byte
	SHA256                     string
}

func uvNativeLauncherFixtures(t *testing.T) []uvNativeLauncherFixture {
	t.Helper()
	body, err := os.ReadFile("testdata/persistent-tool/uv-native-launchers.json")
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Version int
		Cases   []struct {
			Name    string
			Scripts []uvNativeLauncherFixture
		}
	}
	if json.Unmarshal(body, &wire) != nil || wire.Version != 1 || len(wire.Cases) != 5 {
		t.Fatal("native fixture provenance differs")
	}
	var scripts []uvNativeLauncherFixture
	for _, c := range wire.Cases {
		if len(c.Scripts) != 2 {
			t.Fatal("native command census differs")
		}
		for _, s := range c.Scripts {
			sum := sha256.Sum256(s.Body)
			if hex.EncodeToString(sum[:]) != s.SHA256 {
				t.Fatal("native bytes changed")
			}
			scripts = append(scripts, s)
		}
	}
	return scripts
}

func TestUVNativeConsoleMatcherUsesActualPinnedProducerBytes(t *testing.T) {
	for _, s := range uvNativeLauncherFixtures(t) {
		ep := EntryPoint{Group: "console_scripts", Name: s.Name, Module: s.Module, Attr: s.Attr}
		if err := ValidateUVNativeConsoleScript(ep, s.Python, s.Body); err != nil {
			t.Fatal(s.Python, err)
		}
		// Name is deliberately absent from the program body. The later authentic
		// wheel/destination proof must bind it; this matcher grants no ownership.
		ep.Name = "another-valid-name"
		if err := ValidateUVNativeConsoleScript(ep, s.Python, s.Body); err != nil {
			t.Fatal("inert recipe inferred filename authority", err)
		}
	}
}

func TestUVNativeConsoleMatcherRejectsChangedCodeRuntimeAndUnsupportedRoles(t *testing.T) {
	s := uvNativeLauncherFixtures(t)[0]
	original := EntryPoint{Group: "console_scripts", Name: s.Name, Module: s.Module, Attr: s.Attr}
	for name, mutate := range map[string]func(*EntryPoint, *string, *[]byte){
		"trailing program": func(_ *EntryPoint, _ *string, b *[]byte) { *b = append(*b, []byte("print('unexpected')\n")...) },
		"changed template": func(_ *EntryPoint, _ *string, b *[]byte) {
			*b = bytes.ReplaceAll(*b, []byte("sys.exit"), []byte("sys.argv"))
		},
		"changed interpreter": func(_ *EntryPoint, p *string, _ *[]byte) { *p = "/different/bin/python" },
		"changed module":      func(e *EntryPoint, _ *string, _ *[]byte) { e.Module = "other" },
		"changed callable":    func(e *EntryPoint, _ *string, _ *[]byte) { e.Attr = "other" },
		"GUI":                 func(e *EntryPoint, _ *string, _ *[]byte) { e.Group = "gui_scripts" },
		"plugin":              func(e *EntryPoint, _ *string, _ *[]byte) { e.Group = "other_plugin" },
		"extras":              func(e *EntryPoint, _ *string, _ *[]byte) { e.Extras = []string{"extra"} },
		"path name":           func(e *EntryPoint, _ *string, _ *[]byte) { e.Name = "../other" },
		"callable injection":  func(e *EntryPoint, _ *string, _ *[]byte) { e.Attr = "main);print(1)" },
		"module injection":    func(e *EntryPoint, _ *string, _ *[]byte) { e.Module = "x\nimport other" },
		"no callable":         func(e *EntryPoint, _ *string, _ *[]byte) { e.Attr = "" },
		"path traversal":      func(_ *EntryPoint, p *string, _ *[]byte) { *p = "/root/../other/bin/python" },
		"path newline":        func(_ *EntryPoint, p *string, _ *[]byte) { *p += "\n" },
		"relative path":       func(_ *EntryPoint, p *string, _ *[]byte) { *p = "bin/python" },
		"oversize path":       func(_ *EntryPoint, p *string, _ *[]byte) { *p = "/" + strings.Repeat("a", 4096) },
		"oversize module":     func(e *EntryPoint, _ *string, _ *[]byte) { e.Module = strings.Repeat("a", 513) },
		"oversize body":       func(_ *EntryPoint, _ *string, b *[]byte) { *b = bytes.Repeat([]byte("x"), (32<<10)+1) },
	} {
		t.Run(name, func(t *testing.T) {
			ep, python, body := original, s.Python, bytes.Clone(s.Body)
			mutate(&ep, &python, &body)
			if ValidateUVNativeConsoleScript(ep, python, body) == nil {
				t.Fatal("changed/unsupported native script accepted")
			}
		})
	}
}

func TestUVNativeConsoleMatcherImportsOnlyTheFirstDottedAttributeComponent(t *testing.T) {
	s := uvNativeLauncherFixtures(t)[0]
	ep := EntryPoint{Group: "console_scripts", Name: s.Name, Module: s.Module, Attr: "nested.run"}
	body := bytes.ReplaceAll(s.Body, []byte(" import "+s.Attr+"\n"), []byte(" import nested\n"))
	body = bytes.ReplaceAll(body, []byte("sys.exit("+s.Attr+"())"), []byte("sys.exit(nested.run())"))
	if err := ValidateUVNativeConsoleScript(ep, s.Python, body); err != nil {
		t.Fatal(err)
	}
	wrong := bytes.ReplaceAll(body, []byte(" import nested\n"), []byte(" import nested.run\n"))
	if ValidateUVNativeConsoleScript(ep, s.Python, wrong) == nil {
		t.Fatal("incorrect dotted import accepted")
	}
}
