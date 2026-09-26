package runtimeidentity

import "testing"

func TestValidateGVisorRunscVersionOutput(t *testing.T) {
	t.Parallel()
	stamp := ExpectedGVisorRunscVersion()
	if stamp != "7c6199801fd2-dirty" {
		t.Fatalf("source-built stamp = %q", stamp)
	}
	for _, test := range []struct {
		name   string
		output string
		valid  bool
	}{
		{"canonical", "runsc version " + stamp + "\nspec: 1.0.2\n", true},
		{"wrong commit", "runsc version 000000000000-dirty\nspec: 1.0.2\n", false},
		{"unpatched-looking", "runsc version 7c6199801fd2\nspec: 1.0.2\n", false},
		{"release label", "runsc version " + GVisorRelease + "\nspec: 1.0.2\n", false},
		{"embedded stamp", "runsc version other-" + stamp + "\nspec: 1.0.2\n", false},
		{"extra line", "runsc version " + stamp + "\nspec: 1.0.2\nextra\n", false},
		{"missing spec", "runsc version " + stamp + "\n", false},
		{"malformed spec", "runsc version " + stamp + "\nspec: unknown\n", false},
		{"missing newline", "runsc version " + stamp + "\nspec: 1.0.2", false},
		{"empty", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidateGVisorRunscVersionOutput([]byte(test.output)); got != test.valid {
				t.Fatalf("validation of %q = %t, want %t", test.output, got, test.valid)
			}
		})
	}
}
