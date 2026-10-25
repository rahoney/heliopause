package pypi

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func nativeToolFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/persistent-tool/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestNativeToolReceiptActualInstallAndUpgrade(t *testing.T) {
	for _, client := range []string{"uv", "pipx"} {
		t.Run(client, func(t *testing.T) {
			parse, extension, environment := ParseUVToolReceipt, ".toml", "/work/uv-tools/haa-tool-fixture"
			if client == "pipx" {
				parse, extension, environment = ParsePipxToolReceipt, ".json", "/work/pipx/venvs/haa-tool-fixture"
			}
			for _, step := range []string{"installed", "upgraded"} {
				body := nativeToolFixture(t, client+"-"+step+extension)
				r, err := parse(body)
				if err != nil {
					t.Fatal(err)
				}
				if !r.Valid() || r.Client() != client || r.Requirement().Project() != "haa-tool-fixture" || !bytes.Equal(r.Body(), body) {
					t.Fatalf("native receipt identity lost: %+v", r)
				}
				if err := r.ValidateLayout(environment, "/work/"+client+"-bin", "/usr/local/bin/python"); err != nil {
					t.Fatal(err)
				}
				var names []string
				for _, command := range r.Commands() {
					names = append(names, command.Name)
				}
				expected := []string{"haa-alt", "haa-demo"}
				if step == "upgraded" {
					expected = []string{"haa-demo", "haa-new"}
				}
				if !reflect.DeepEqual(names, expected) {
					t.Fatalf("command retirement/addition lost: %v", names)
				}
				links := map[string]string{}
				for _, name := range names {
					links["/work/"+client+"-bin/"+name] = environment + "/bin/" + name
				}
				if err := r.ValidateExposedCommands(environment, "/work/"+client+"-bin", "/usr/local/bin/python", links); err != nil {
					t.Fatal(err)
				}
				links["/work/"+client+"-bin/"+names[0]] = "/foreign/target"
				if r.ValidateExposedCommands(environment, "/work/"+client+"-bin", "/usr/local/bin/python", links) == nil {
					t.Fatal("foreign exposed link accepted")
				}
				delete(links, "/work/"+client+"-bin/"+names[0])
				if r.ValidateExposedCommands(environment, "/work/"+client+"-bin", "/usr/local/bin/python", links) == nil {
					t.Fatal("missing exposed link accepted")
				}
				upgrade, err := r.UpgradeRequirement()
				if err != nil {
					t.Fatal(err)
				}
				if client == "uv" {
					if r.Version() != "" || r.Requirement().Requirement() != "haa-tool-fixture>=1,<2" || upgrade.AcceptsVersion("2.0.0") {
						t.Fatal("UV version was invented or constraint lost")
					}
				} else {
					version := "1.0.0"
					if step == "upgraded" {
						version = "1.1.0"
					}
					if r.Version() != version || upgrade.Requirement() != "haa-tool-fixture" || !upgrade.AcceptsVersion("2.0.0") {
						t.Fatal("pipx native unconstrained upgrade semantics lost")
					}
					if step == "installed" && r.Requirement().AcceptsVersion("2.0.0") {
						t.Fatal("original pipx install constraint lost")
					}
				}
				commands := r.Commands()
				commands[0].Name = "foreign"
				copyBody := r.Body()
				copyBody[0] = '!'
				body[0] = '!'
				if r.Commands()[0].Name == "foreign" || r.Body()[0] == '!' {
					t.Fatal("receipt data is aliased")
				}
				if client == "uv" && r.ValidateLayout(environment, "/elsewhere/bin", "/usr/local/bin/python") == nil {
					t.Fatal("foreign UV exposure directory accepted")
				}
				for _, layout := range [][3]string{{environment, "/work/" + client + "-bin", "/evil/python"}, {environment, "/work/" + client + "-bin/../bin", "/usr/local/bin/python"}, {environment, environment + "/bin", "/usr/local/bin/python"}, {"/work/foreign", "/work/" + client + "-bin", "/usr/local/bin/python"}} {
					if r.ValidateLayout(layout[0], layout[1], layout[2]) == nil {
						t.Fatalf("foreign layout accepted: %v", layout)
					}
				}
			}
		})
	}
	if (ToolReceipt{}).ValidateLayout("/a/pkg", "/b", "/c") == nil {
		t.Fatal("zero receipt accepted")
	}
	if _, err := (ToolReceipt{}).UpgradeRequirement(); err == nil {
		t.Fatal("zero receipt has upgrade intent")
	}
}

func TestUVToolReceiptRejectsUnsupportedNativeClaims(t *testing.T) {
	body := string(nativeToolFixture(t, "uv-installed.toml"))
	cases := map[string]string{
		"empty": "", "unknown": body + "force = true\n", "duplicate": body + "no-build = false\n",
		"unknown source":       strings.Replace(body, "from = \"haa-tool-fixture\"", "from = \"foreign\"", 1),
		"direct root":          strings.Replace(body, "specifier = \">=1,<2\"", "specifier = \" @ https://evil.test/a.whl\"", 1),
		"dependency source":    strings.Replace(body, "requirements = [", "constraints = []\nrequirements = [", 1),
		"multi root":           strings.Replace(body, "specifier = \">=1,<2\" }", "specifier = \">=1,<2\" }, {name = \"other\"}", 1),
		"implicit interpreter": strings.Replace(body, "python = \"/usr/local/bin/python\"\n", "", 1),
		"interpreter escape":   strings.ReplaceAll(body, "/usr/local/bin/python", "/usr/local/bin/../python"),
		"empty commands":       strings.Replace(body, "entrypoints = [", "other = [", 1),
		"command path":         strings.Replace(body, "/work/uv-bin/haa-alt", "/work/uv-bin/other", 1),
		"command escape":       strings.ReplaceAll(body, "haa-alt", "../haa-alt"),
		"case collision":       strings.ReplaceAll(body, "haa-demo", "HAA-ALT"),
		"nul":                  body + "\x00", "oversize": strings.Repeat("x", upgradeInputLimit+1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseUVToolReceipt([]byte(input)); err == nil {
				t.Fatal("unsupported native claim accepted")
			}
		})
	}
}

func TestPipxToolReceiptRejectsUnsupportedNativeClaims(t *testing.T) {
	body := nativeToolFixture(t, "pipx-installed.json")
	mutations := map[string]func(map[string]any){
		"schema":           func(w map[string]any) { w["pipx_metadata_version"] = "0.13" },
		"implicit backend": func(w map[string]any) { delete(w, "backend") },
		"pip backend":      func(w map[string]any) { w["backend"] = "pip" },
		"unknown":          func(w map[string]any) { w["other"] = true },
		"field case":       func(w map[string]any) { w["Backend"] = "pip" },
		"hidden exposure":  func(w map[string]any) { w["exposure_enabled"] = false },
		"injected":         func(w map[string]any) { w["injected_packages"] = map[string]any{"other": map[string]any{}} },
		"environment":      func(w map[string]any) { w["environment"] = "other" },
		"Python":           func(w map[string]any) { w["python_version"] = "Python 3.13.0" },
		"venv args":        func(w map[string]any) { w["venv_args"] = []string{"--system-site-packages"} },
		"interpreter type": func(w map[string]any) { w["source_interpreter"].(map[string]any)["__type__"] = "Foreign" },
	}
	mainMutations := map[string]func(map[string]any){
		"package identity":     func(m map[string]any) { m["package"] = "other" },
		"version constraint":   func(m map[string]any) { m["package_version"] = "2.0.0" },
		"direct source":        func(m map[string]any) { m["package_or_url"] = "haa-tool-fixture @ https://evil.test/a.whl" },
		"pinned":               func(m map[string]any) { m["pinned"] = true },
		"suffix":               func(m map[string]any) { m["suffix"] = "-evil" },
		"include dependencies": func(m map[string]any) { m["include_dependencies"] = true },
		"resource exposure":    func(m map[string]any) { m["man_pages"] = []string{"tool.1"} },
		"missing include apps": func(m map[string]any) { delete(m, "include_apps") },
		"empty apps":           func(m map[string]any) { m["apps"] = []string{} },
		"foreign path":         func(m map[string]any) { m["app_paths"].([]any)[0].(map[string]any)["__Path__"] = "/work/bin/foreign" },
		"path type":            func(m map[string]any) { m["app_paths"].([]any)[0].(map[string]any)["__type__"] = "str" },
		"nested unknown":       func(m map[string]any) { m["app_paths"].([]any)[0].(map[string]any)["extra"] = true },
		"cooldown":             func(m map[string]any) { m["cooldown_days"] = 7 },
		"lock file":            func(m map[string]any) { m["lock_file"] = "foreign.lock" },
	}
	for name, mutate := range mainMutations {
		mutations[name] = func(w map[string]any) { mutate(w["main_package"].(map[string]any)) }
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			var w map[string]any
			if err := json.Unmarshal(body, &w); err != nil {
				t.Fatal(err)
			}
			mutate(w)
			input, err := json.Marshal(w)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParsePipxToolReceipt(input); err == nil {
				t.Fatal("unsupported native claim accepted")
			}
		})
	}
	for name, input := range map[string][]byte{
		"duplicate":        append([]byte(`{"backend":"pip",`), body[1:]...),
		"nested duplicate": []byte(strings.Replace(string(body), `"__type__": "Path"`, `"__type__": "Foreign", "__type__": "Path"`, 1)),
		"trailing":         append(bytes.Clone(body), []byte(` {}`)...),
		"deep":             []byte(`{"injected_packages":{"a":{"b":{"c":{"d":{"e":{"f":{"g":0}}}}}}}}`),
		"null":             []byte("null"), "oversize": bytes.Repeat([]byte("x"), upgradeInputLimit+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParsePipxToolReceipt(input); err == nil {
				t.Fatal("ambiguous native JSON accepted")
			}
		})
	}
}
