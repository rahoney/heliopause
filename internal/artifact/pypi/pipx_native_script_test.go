package pypi

import (
	"bytes"
	"strings"
	"testing"
)

func TestPipxNativeScriptRecordUsesPinnedUVTemplateAndExposureDelta(t *testing.T) {
	for _, s := range uvNativeLauncherFixtures(t) {
		ep := EntryPoint{Group: "console_scripts", Name: s.Name, Module: s.Module, Attr: s.Attr}
		actual := bytes.Clone(s.Body)
		first, rest, _ := bytes.Cut(actual, []byte("\n"))
		changed := bytes.Contains(bytes.ToLower(first[2:]), []byte("python")) && !bytes.ContainsAny(first[2:], " \t") && len(first)+1+3 <= 127
		if changed {
			actual = append(append(append([]byte{}, first...), []byte(" -E\n")...), rest...)
		}
		candidate, err := PipxNativeUVScriptRecordBody(s.Python, actual)
		if err != nil || !bytes.Equal(candidate, s.Body) || ValidateUVNativeConsoleScript(ep, s.Python, candidate) != nil {
			t.Fatal(s.Python, err)
		}
		if changed && ValidateUVNativeConsoleScript(ep, s.Python, actual) == nil {
			t.Fatal("UV matcher accepted pipx-only mutation")
		}
		candidate[0] ^= 1
		if actual[0] != '#' {
			t.Fatal("candidate aliases caller bytes")
		}
	}
}

func TestPipxNativeScriptRecordBoundaryIsInertUntilCompleteEPProof(t *testing.T) {
	for _, length := range []int{121, 122, 124, 125} {
		python := "/" + strings.Repeat("a", length-len("/python")) + "python"
		original := []byte("#!" + python + "\nuntrusted DATA\n")
		actual := bytes.Clone(original)
		if length == 121 {
			actual = bytes.Replace(actual, []byte("\n"), []byte(" -E\n"), 1)
		}
		candidate, err := PipxNativeUVScriptRecordBody(python, actual)
		if err != nil || !bytes.Equal(candidate, original) {
			t.Fatal(length, err)
		}
		ep := EntryPoint{Group: "console_scripts", Name: "demo", Module: "demo", Attr: "main"}
		if ValidateUVNativeConsoleScript(ep, python, candidate) == nil {
			t.Fatal("RECORD candidate became entrypoint authority")
		}
	}
}

func TestPipxNativeScriptRecordRejectsOtherShortFlagsAndInvalidInput(t *testing.T) {
	s := uvNativeLauncherFixtures(t)[0]
	first, rest, _ := bytes.Cut(s.Body, []byte("\n"))
	for _, suffix := range []string{"", " -I", " -E -E", " -EE", "  -E", " -E\t"} {
		actual := append(append(append([]byte{}, first...), []byte(suffix+"\n")...), rest...)
		if _, err := PipxNativeUVScriptRecordBody(s.Python, actual); err == nil {
			t.Fatal("unsupported flags accepted", suffix)
		}
	}
	for _, python := range []string{"", "bin/python", "/tmp/../python", "/tmp/python\n"} {
		if _, err := PipxNativeUVScriptRecordBody(python, s.Body); err == nil {
			t.Fatal("invalid path accepted", python)
		}
	}
	for _, body := range [][]byte{nil, bytes.Repeat([]byte("x"), (32<<10)+1)} {
		if _, err := PipxNativeUVScriptRecordBody(s.Python, body); err == nil {
			t.Fatal("invalid body accepted")
		}
	}
}
