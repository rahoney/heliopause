package pypi

import "testing"

func TestEntryPointExtrasKeepOneExactTarget(t *testing.T) {
	for _, value := range []string{"pkg.ext:extract[i18n]", "pkg.ext : extract  [ i18n ]"} {
		data := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", []wheelTestEntry{
			{name: "pkg/__init__.py", body: []byte("pass\n")},
			{name: "pkg/ext.py", body: []byte("def extract(): pass\n")},
			{name: "example-1.0.dist-info/entry_points.txt", body: []byte("[babel.extractors]\npkg=" + value + "\n")},
		}, []string{"py3-none-any"}, nil)
		inspection := inspectTestWheel(t, data, "example-1.0-py3-none-any.whl")
		plan, err := BuildObservationPlan(inspection, defaultResourcePolicy())
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.EntryPointCoverage) != 1 || len(plan.Units) != 2 || !plan.Admissible() {
			t.Fatalf("duplicate/lost entry point: %+v", plan)
		}
		if err := ValidateTypedObservationPlan(plan, defaultResourcePolicy()); err != nil {
			t.Fatal(err)
		}
		plan.EntryPointDetails[0].Module = "unrelated"
		if err := ValidateTypedObservationPlan(plan, defaultResourcePolicy()); err == nil {
			t.Fatal("substituted target accepted")
		}
	}
}

func TestEntryPointAmbiguityRejects(t *testing.T) {
	for _, body := range []string{
		"x=pkg:main[bad!]", "x=pkg:main[a,]", "x=pkg:main[a,a]",
		"x=pkg:main[a_b,a-b]", "x=pkg:main[[a]]",
		"x=pkg:main\nx=other:main", "x=pkg:main\n[g]\nx=pkg:main",
	} {
		if _, _, err := parseEntryPointsTxt([]byte("[g]\n" + body + "\n")); err == nil {
			t.Fatalf("ambiguous entry point accepted: %q", body)
		}
	}
}
