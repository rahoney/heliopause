package sandbox

import (
	"reflect"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestPipxNativeArgumentsFixFrontendBackendAndInterpreter(t *testing.T) {
	version, err := pipxNativeArguments(pipxNativeVersion, artifactpypi.UpgradeRequirement{})
	if err != nil || !reflect.DeepEqual(version, []string{"/usr/local/bin/python", "-I", "-S", "-B", "-c", pipxNativeProgram, "--version"}) {
		t.Fatal("version program differs", err)
	}
	for _, operation := range []pipxNativeOperation{pipxNativeInstall, pipxNativeUpgrade} {
		name := "cowsay==6.1"
		if operation == pipxNativeUpgrade {
			name = "cowsay"
		}
		request, err := artifactpypi.ParseUpgradeRequirement(name)
		if err != nil {
			t.Fatal(err)
		}
		args, err := pipxNativeArguments(operation, request)
		if err != nil || len(args) < 11 || !reflect.DeepEqual(args[:6], version[:6]) {
			t.Fatal("native prefix changed", err)
		}
		expected := []string{"install", "cowsay==6.1", "--backend", "uv", "--python", "/usr/local/bin/python"}
		if operation == pipxNativeUpgrade {
			expected = []string{"upgrade", "cowsay", "--backend", "uv"}
		}
		if !reflect.DeepEqual(args[6:len(args)-1], expected) {
			t.Fatal("native operation/constraint/backend changed", args[6:])
		}
		flags := strings.Fields(strings.TrimPrefix(args[len(args)-1], "--pip-args="))
		wanted := map[string]bool{"--no-build": true, "--no-sources": true, "--no-config": true, "--no-cache": true, "--no-python-downloads": true, "--no-managed-python": true, "--link-mode=copy": true, "--index-url=https://pypi.org/simple/": true}
		for _, flag := range flags {
			if !wanted[flag] {
				t.Fatal("duplicate/unknown native pip option or duplicate --python", flag)
			}
			delete(wanted, flag)
		}
		if len(wanted) != 0 {
			t.Fatal("fixed native safety/source options absent", wanted)
		}
	}
}

func TestPipxNativeArgumentsRejectUnknownOperationConstraintAndSource(t *testing.T) {
	bare, _ := artifactpypi.ParseUpgradeRequirement("cowsay")
	exact, _ := artifactpypi.ParseUpgradeRequirement("cowsay==6.1")
	bounded, _ := artifactpypi.ParseUpgradeRequirement("cowsay>=6,<7")
	profile, ok := artifactpypi.PyTorchProfile("cpu")
	if !ok {
		t.Fatal("named source fixture unavailable")
	}
	foreign, err := artifactpypi.ParseUpgradeRequirementForSource("torch", profile.Source())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		operation pipxNativeOperation
		request   artifactpypi.UpgradeRequirement
	}{
		{0, bare}, {255, bare}, {pipxNativeVersion, bare}, {pipxNativeInstall, artifactpypi.UpgradeRequirement{}}, {pipxNativeUpgrade, artifactpypi.UpgradeRequirement{}},
		{pipxNativeUpgrade, exact}, {pipxNativeUpgrade, bounded}, {pipxNativeInstall, foreign}, {pipxNativeUpgrade, foreign},
	} {
		if args, err := pipxNativeArguments(test.operation, test.request); err == nil || len(args) != 0 {
			t.Fatal("unsupported native input produced argv")
		}
	}
	args, err := pipxNativeArguments(pipxNativeInstall, bounded)
	if err != nil || args[7] != bounded.Requirement() {
		t.Fatal("supported install constraint lost", err)
	}
}

func TestPipxNativeEnvironmentUsesOnlyFixedOfflineGuestInputs(t *testing.T) {
	before := pipxNativeEnvironmentArguments()
	for _, name := range []string{"PATH", "HOME", "PIPX_HOME", "PIPX_DEFAULT_BACKEND", "UV_FIND_LINKS", "UV_OFFLINE", "UV_INDEX_URL", "PYTHONPATH", "VIRTUAL_ENV", "ARBITRARY_TOKEN"} {
		t.Setenv(name, "caller-controlled-sentinel")
	}
	after := pipxNativeEnvironmentArguments()
	if !reflect.DeepEqual(before, after) || len(after)%2 != 0 {
		t.Fatal("caller environment reached native guest")
	}
	values := map[string]string{}
	for i := 0; i < len(after); i += 2 {
		if after[i] != "--env" {
			t.Fatal("environment builder produced non-environment argument")
		}
		key, value, ok := strings.Cut(after[i+1], "=")
		if !ok || values[key] != "" || strings.Contains(value, "caller-controlled-sentinel") {
			t.Fatal("duplicate or caller environment")
		}
		values[key] = value
	}
	for key, value := range map[string]string{
		"HOME": "/tmp", "PATH": "/haa-runtime:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"PIPX_HOME": "/haa-site/pipx", "PIPX_BIN_DIR": "/haa-site/tool-bin", "PIPX_MAN_DIR": "/haa-site/tool-man", "PIPX_COMPLETION_DIR": "/haa-site/tool-completions",
		"PIPX_DEFAULT_PYTHON": "/usr/local/bin/python", "PIPX_DEFAULT_BACKEND": "uv", "PYTHONDONTWRITEBYTECODE": "1",
		"UV_OFFLINE": "true", "UV_NO_INDEX": "true", "UV_FIND_LINKS": "/haa-wheels", "UV_NO_CONFIG": "true", "UV_PYTHON_DOWNLOADS": "never", "UV_NO_CACHE": "true",
	} {
		if values[key] != value {
			t.Fatal("fixed guest environment differs", key, values[key])
		}
	}
	if values["PYTHONPATH"] != "" || values["VIRTUAL_ENV"] != "" || values["ARBITRARY_TOKEN"] != "" {
		t.Fatal("ambient capability reached guest")
	}
	after[1] = "caller-mutated"
	if !reflect.DeepEqual(before, pipxNativeEnvironmentArguments()) {
		t.Fatal("environment result aliases later caller")
	}
}
