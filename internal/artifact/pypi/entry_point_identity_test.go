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

func TestEntryPointNamespaceTargetUsesExactOwnedDirectory(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
		shadow  bool
		covered bool
	}{
		{"owned namespace", "pkg/lib/catalog/example.pc", false, true},
		{"renamed resource", "pkg/lib/catalog/opaque.payload", false, true},
		{"neighbor directory", "pkg/lib/catalogue/example.pc", false, false},
		{"data scheme", "example-1.0.data/data/share/pkg/lib/catalog/example.pc", false, false},
		{"nonpackage parent", "pkg/lib/catalog/example.pc", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entries := []wheelTestEntry{
				{name: "pkg/__init__.py", body: []byte("pass\n")},
				{name: tc.payload, body: []byte("untrusted resource\n")},
				{name: "example-1.0.dist-info/entry_points.txt", body: []byte("[example.locations]\nlocation = pkg.lib.catalog\n")},
			}
			if tc.shadow {
				entries = append(entries, wheelTestEntry{name: "pkg/lib.py", body: []byte("pass\n")})
			}
			archive := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
			inspection := inspectTestWheel(t, archive, "example-1.0-py3-none-any.whl")
			plan, err := BuildObservationPlan(inspection, defaultResourcePolicy())
			if err != nil {
				t.Fatal(err)
			}
			if plan.Admissible() != tc.covered || containsCandidate(plan.ImportCandidates, "pkg.lib.catalog") != tc.covered {
				t.Fatalf("exact namespace coverage=%v: %+v", tc.covered, plan)
			}
			if err := ValidateTypedObservationPlan(plan, defaultResourcePolicy()); err != nil {
				t.Fatal(err)
			}
			if tc.covered && (len(plan.Units) != 2 || plan.Units[1].Candidate != "pkg.lib.catalog") {
				t.Fatalf("namespace lacks its independent exact unit: %+v", plan.Units)
			}
		})
	}
}

func TestEntryPointCoverageRequiredRoleWins(t *testing.T) {
	for _, tc := range []struct {
		name, group, target, hook string
		required                  bool
	}{
		{"console only", "console_scripts", "pkg.cli", "", false},
		{"gui only", "gui_scripts", "pkg.cli", "", false},
		{"normal import", "console_scripts", "pkg", "", true},
		{"plugin", "plugins", "pkg.cli", "", true},
		{"startup overlap", "console_scripts", "pkg.cli", "import pkg.cli", true},
		{"declarative path", "console_scripts", "pkg.cli", "pkg", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entries := []wheelTestEntry{
				{name: "pkg/__init__.py", body: []byte("pass\n")},
				{name: "pkg/cli.py", body: []byte("raise ModuleNotFoundError('numpy')\n")},
				{name: "example-1.0.dist-info/entry_points.txt", body: []byte("[" + tc.group + "]\ncommand=" + tc.target + ":main\n")},
			}
			if tc.hook != "" {
				entries = append(entries, wheelTestEntry{name: "startup.pth", body: []byte(tc.hook + "\n")})
			}
			inspection := inspectTestWheel(t, recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", entries, []string{"py3-none-any"}, nil), "example-1.0-py3-none-any.whl")
			plan, err := BuildObservationPlan(inspection, defaultResourcePolicy())
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateTypedObservationPlan(plan, defaultResourcePolicy()); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, unit := range plan.Units {
				if unit.Candidate == tc.target {
					found = true
					if (unit.Coverage == RequiredObservation) != tc.required {
						t.Fatalf("required=%v: %+v", tc.required, unit)
					}
				}
			}
			if !found {
				t.Fatal("missing exact target")
			}
			for i := range plan.Units {
				if plan.Units[i].Candidate == tc.target {
					if tc.required {
						plan.Units[i].Coverage = PostInstallCommandObservation
					} else {
						plan.Units[i].Coverage = RequiredObservation
					}
				}
			}
			if ValidateTypedObservationPlan(plan, defaultResourcePolicy()) == nil {
				t.Fatal("substituted classification accepted")
			}
		})
	}
}
