package pypi

import (
	"slices"
	"strings"
	"testing"
)

func TestUVRunRootExtrasPreserveBareExactAndOwnedRequest(t *testing.T) {
	for input, want := range map[string]string{
		"python-slugify":                   "python-slugify",
		"python-slugify@8.0.4":             "python-slugify==8.0.4",
		"Python_Slugify[unidecode]==8.0.4": "python-slugify[unidecode]==8.0.4",
		"python-slugify[unidecode]@8.0.4":  "python-slugify[unidecode]==8.0.4",
		"example[b,a]":                     "example[a,b]",
	} {
		r, err := ParseUVRunRequirement(input)
		if err != nil || !r.Valid() || r.Requirement() != want || strings.Contains(r.Reference().Locator(), "[") {
			t.Fatal("root selection was not retained separately from artifact identity", input, r, err)
		}
		if len(r.Extras()) != 0 {
			extras := r.Extras()
			expected := slices.Clone(extras)
			extras[0] = "changed"
			if !slices.Equal(r.Extras(), expected) {
				t.Fatal("request extras are mutable")
			}
		}
	}
	var zero UVRunRequirement
	if zero.Valid() {
		t.Fatal("zero request is valid")
	}
}

func TestUVRunExtrasDoNotWidenInstalledOrNumericRequirementGrammar(t *testing.T) {
	for _, input := range []string{"", "example@6.2rc1", "example[a]==6.2rc1", "example[]", "example[a,a]", "example[a_b,a-b]", "example[a][b]", "example[[a]]", "example[a", "example]", "example[a]>=1", "example>=1", "example[a]==1,<2", "example[a]; python_version > '3'", "example[a] @ https://example.com/file.whl", "example[../x]", " example[a]", "example[a]\n"} {
		if r, err := ParseUVRunRequirement(input); err == nil || r.Valid() {
			t.Fatal("unsupported or ambiguous request accepted", input)
		}
	}
	if _, err := ParseUpgradeRequirement("example[feature]==1.0"); err == nil {
		t.Fatal("installed upgrade grammar widened")
	}
	if _, err := ParseBoundedRequirement("example[feature]==1.0"); err == nil {
		t.Fatal("CUDA/ordinary bounded grammar widened")
	}
}
