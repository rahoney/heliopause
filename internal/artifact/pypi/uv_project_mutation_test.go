package pypi

import (
	"bytes"
	"strings"
	"testing"
)

func uvAddInput(t *testing.T, dependencies, selector string) UVProjectInput {
	t.Helper()
	input, err := ParseUVProjectInput(uvVirtualProjectTestInput(dependencies), nil, false, UVProjectAdd, selector)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestUVNativeAddConservesMetadataAndOtherRequirements(t *testing.T) {
	input := uvAddInput(t, `["idna==3.11"]`, "packaging>=24,<26")
	body := uvVirtualProjectTestInput(`["packaging<26,>=24", "idna==3.11"]`)
	after, err := input.SelectManifest(body)
	if err != nil {
		t.Fatal("requested native dependency edit rejected", err)
	}
	if err := input.ValidateAddSelection(after, []UVLockedPackage{{Name: "idna", Version: "3.11"}, {Name: "packaging", Version: "25.0"}}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(input.Body(), uvVirtualProjectTestInput(`["idna==3.11"]`)) || !bytes.Equal(after.Body(), body) {
		t.Fatal("native edit mutated original bytes or lost selected bytes")
	}
	for name, mutated := range map[string][]byte{
		"unrelated removal":            uvVirtualProjectTestInput(`["packaging>=24,<26"]`),
		"unrelated weakening":          uvVirtualProjectTestInput(`["idna>=3", "packaging>=24,<26"]`),
		"unrelated addition":           uvVirtualProjectTestInput(`["idna==3.11", "packaging>=24,<26", "certifi"]`),
		"changed requested constraint": uvVirtualProjectTestInput(`["idna==3.11", "packaging>=24"]`),
		"wrong added name":             uvVirtualProjectTestInput(`["idna==3.11", "certifi"]`),
		"project identity":             bytes.ReplaceAll(body, []byte("haa-example"), []byte("foreign-project")),
		"project version":              bytes.ReplaceAll(body, []byte("0.1.0"), []byte("0.2.0")),
		"compatible Python mutation":   bytes.ReplaceAll(body, []byte("==3.14.7"), []byte(">=3.14.7")),
		"new tool config":              append(append([]byte(nil), body...), []byte("[tool.uv]\npackage=false\n")...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := input.SelectManifest(mutated); err == nil {
				t.Fatal("unrequested native manifest mutation accepted")
			}
		})
	}
	if err := input.ValidateAddSelection(after, []UVLockedPackage{{Name: "packaging", Version: "26.0"}}); err == nil {
		t.Fatal("selected version escaped user constraint")
	}
	if err := input.ValidateAddSelection(after, []UVLockedPackage{{Name: "packaging", Version: "25.0"}, {Name: "packaging", Version: "25.0"}}); err == nil {
		t.Fatal("ambiguous selected root accepted")
	}
}

func TestUVNativeAddUpdatesOnlyRequestedExistingConstraint(t *testing.T) {
	input := uvAddInput(t, `["packaging==24.2", "idna==3.11"]`, "packaging==25.0")
	after, err := input.SelectManifest(uvVirtualProjectTestInput(`["idna==3.11", "packaging==25.0"]`))
	if err != nil {
		t.Fatal(err)
	}
	if err := input.ValidateAddSelection(after, []UVLockedPackage{{Name: "packaging", Version: "25.0"}}); err != nil {
		t.Fatal(err)
	}
	if err := input.ValidateAddSelection(after, []UVLockedPackage{{Name: "packaging", Version: "24.2"}}); err == nil {
		t.Fatal("old selected root survived explicit version change")
	}
}

func TestUVNativeBareAddBindsItsActualLowerBound(t *testing.T) {
	input := uvAddInput(t, `[]`, "packaging")
	for _, test := range []struct {
		constraint string
		valid      bool
	}{{"packaging>=25.0", true}, {"packaging>=24.2", false}, {"packaging", false}, {"packaging==25.0", false}} {
		after, err := input.SelectManifest(uvVirtualProjectTestInput(`["` + test.constraint + `"]`))
		if err != nil {
			t.Fatal(err)
		}
		err = input.ValidateAddSelection(after, []UVLockedPackage{{Name: "packaging", Version: "25.0"}})
		if (err == nil) != test.valid {
			t.Fatal("native bare lower bound differs", test.constraint, err)
		}
	}
	existing := uvAddInput(t, `["packaging==25.0"]`, "packaging")
	after, err := existing.SelectManifest(existing.Body())
	if err != nil {
		t.Fatal(err)
	}
	if err := existing.ValidateAddSelection(after, []UVLockedPackage{{Name: "packaging", Version: "25.0"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := existing.SelectManifest(uvVirtualProjectTestInput(`["packaging>=25.0"]`)); err == nil {
		t.Fatal("bare add weakened existing constraint")
	}
}

func TestUVSyncSelectionCannotRewriteEvenEquivalentManifest(t *testing.T) {
	body := uvVirtualProjectTestInput(`[]`)
	input, err := ParseUVProjectInput(body, nil, false, UVProjectSync, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := input.SelectManifest(append([]byte("# formatting only\n"), body...)); err == nil {
		t.Fatal("sync rewrote original manifest bytes")
	}
	if _, err := (UVProjectInput{}).SelectManifest(body); err == nil {
		t.Fatal("zero original accepted manifest")
	}
	if err := input.ValidateAddSelection(input, nil); err == nil {
		t.Fatal("sync accepted add-only version validation")
	}
	if strings.Contains(string(input.Body()), "formatting only") {
		t.Fatal("original buffer mutated")
	}
}
