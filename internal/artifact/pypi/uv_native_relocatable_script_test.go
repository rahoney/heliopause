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

func uvNativeRelocatableFixture(t *testing.T) (EntryPoint, []byte) {
	t.Helper()
	body, err := os.ReadFile("testdata/persistent-tool/uv-native-relocatable-launcher.json")
	if err != nil {
		t.Fatal(err)
	}
	var row struct {
		Version                    int
		Name, Module, Attr, SHA256 string
		Body                       []byte
	}
	if json.Unmarshal(body, &row) != nil || row.Version != 1 {
		t.Fatal("native relocatable DATA fixture")
	}
	sum := sha256.Sum256(row.Body)
	if hex.EncodeToString(sum[:]) != row.SHA256 || len(row.Body) != 358 {
		t.Fatal("native relocatable bytes changed")
	}
	return EntryPoint{Group: "console_scripts", Name: row.Name, Module: row.Module, Attr: row.Attr}, row.Body
}

func TestUVRelocatableMatcherUsesActualPinnedNativeBytesAndPreservesOtherRecipe(t *testing.T) {
	ep, body := uvNativeRelocatableFixture(t)
	original := bytes.Clone(body)
	if err := ValidateUVRelocatableConsoleScript(ep, body); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, original) {
		t.Fatal("DATA matcher modified source bytes")
	}
	if ValidateUVNativeConsoleScript(ep, "/usr/local/bin/python", body) == nil {
		t.Fatal("nonrelocatable recipe accepted a relative wrapper")
	}
	for _, old := range uvNativeLauncherFixtures(t) {
		if ValidateUVRelocatableConsoleScript(EntryPoint{Group: "console_scripts", Name: old.Name, Module: old.Module, Attr: old.Attr}, old.Body) == nil {
			t.Fatal("relative recipe accepted old absolute wrapper")
		}
	}
	// The caller binds filename/command to the canonical original separately.
	ep.Name = "another-valid-name"
	if err := ValidateUVRelocatableConsoleScript(ep, body); err != nil {
		t.Fatal("recipe invented filename authority", err)
	}
}

func TestUVRelocatableMatcherRejectsChangedHeaderCodeAndUnsupportedDeclarations(t *testing.T) {
	ep, original := uvNativeRelocatableFixture(t)
	changes := map[string]func(*EntryPoint, []byte) []byte{
		"other relative runtime": func(_ *EntryPoint, b []byte) []byte {
			return bytes.Replace(b, []byte("/'python'"), []byte("/'python3'"), 1)
		},
		"absolute runtime": func(_ *EntryPoint, b []byte) []byte {
			return bytes.Replace(b, []byte("/'python'"), []byte("/'/usr/local/bin/python'"), 1)
		},
		"parent runtime": func(_ *EntryPoint, b []byte) []byte {
			return bytes.Replace(b, []byte("/'python'"), []byte("/'../python'"), 1)
		},
		"changed path resolution": func(_ *EntryPoint, b []byte) []byte {
			return bytes.Replace(b, []byte("realpath --"), []byte("readlink --"), 1)
		},
		"changed shell": func(_ *EntryPoint, b []byte) []byte {
			return bytes.Replace(b, []byte("#!/bin/sh"), []byte("#!/bin/bash"), 1)
		},
		"extra program":    func(_ *EntryPoint, b []byte) []byte { return append(b, []byte("print('extra')\n")...) },
		"changed callable": func(e *EntryPoint, b []byte) []byte { e.Attr = "other"; return b },
		"changed module":   func(e *EntryPoint, b []byte) []byte { e.Module = "other"; return b },
		"GUI":              func(e *EntryPoint, b []byte) []byte { e.Group = "gui_scripts"; return b },
		"plugin":           func(e *EntryPoint, b []byte) []byte { e.Group = "plugin"; return b },
		"extras":           func(e *EntryPoint, b []byte) []byte { e.Extras = []string{"extra"}; return b },
		"unsafe name":      func(e *EntryPoint, b []byte) []byte { e.Name = "../other"; return b },
		"bad import":       func(e *EntryPoint, b []byte) []byte { e.Module = "os;other"; return b },
		"bad callable":     func(e *EntryPoint, b []byte) []byte { e.Attr = "cli()"; return b },
		"NUL":              func(_ *EntryPoint, b []byte) []byte { return append(b, 0) },
		"oversize":         func(_ *EntryPoint, b []byte) []byte { return append(b, []byte(strings.Repeat("x", 32<<10))...) },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			selected := ep
			body := change(&selected, bytes.Clone(original))
			if ValidateUVRelocatableConsoleScript(selected, body) == nil {
				t.Fatal("changed native DATA accepted")
			}
		})
	}
	if ValidateUVRelocatableConsoleScript(ep, nil) == nil {
		t.Fatal("empty wrapper accepted")
	}
}
