package promotion

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestUVVenvConfigKeepsTargetAndCreationHistorySeparate(t *testing.T) {
	version := sandbox.PinnedPythonRuntime().PythonVersion
	for _, creator := range []string{"0.13.4", "0.12.0", "1.0.0"} {
		body := []byte("home = /usr/local/bin\nimplementation = CPython\nuv = " + creator + "\nversion_info = " + version + "\ninclude-system-site-packages = false\n")
		got, err := uvPythonVenvConfigVersion(body)
		if err != nil || got != version {
			t.Fatal(creator, got, err)
		}
	}
	if got, err := uvPythonVenvConfigVersion([]byte("version = " + version + "\n")); err != nil || got != version {
		t.Fatal(got, err)
	}
	// Existing pip semantics keep their own parser and target validation.
	if got, err := pythonVenvConfigVersion([]byte("version = 3.14.1\nextra = harmless\n")); err != nil || got != "3.14.1" {
		t.Fatal(got, err)
	}
}

func TestUVVenvConfigRejectsConflictingAndUnknownFormats(t *testing.T) {
	v := sandbox.PinnedPythonRuntime().PythonVersion
	valid := "implementation = CPython\nuv = 0.13.4\nversion_info = " + v + "\ninclude-system-site-packages = false\n"
	for _, body := range []string{
		"", "version = 3.13.0\n", "version = " + v + "\nversion = " + v + "\n", valid + "version = " + v + "\n",
		valid + "version_info = " + v + "\n", valid + "unknown = true\n", valid + "malformed\n",
		strings.ReplaceAll(valid, "CPython", "PyPy"), strings.ReplaceAll(valid, "false", "true"),
		strings.ReplaceAll(valid, "0.13.4", ""), strings.ReplaceAll(valid, "0.13.4", "untrusted"),
		strings.ReplaceAll(valid, "0.13.4", "0.13.4.5"), strings.ReplaceAll(valid, "0.13.4", "0.13.+4"),
		strings.ReplaceAll(valid, "0.13.4", "99999999999.1.1"), strings.ReplaceAll(valid, "uv = 0.13.4\n", ""),
		strings.ReplaceAll(valid, "include-system-site-packages = false\n", ""),
		"version = " + v + "\nuv = 0.13.4\n", "version = " + v + "\ninclude-system-site-packages = true\n",
	} {
		if _, err := uvPythonVenvConfigVersion([]byte(body)); err == nil {
			t.Fatalf("unsupported cfg accepted: %q", body)
		}
	}
}
