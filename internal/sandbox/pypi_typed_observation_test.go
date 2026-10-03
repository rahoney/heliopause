package sandbox

import (
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

func TestTypedPythonObservationFreezesIndependentUnits(t *testing.T) {
	plan := artifactpypi.ObservationPlan{
		Project: "fixture", Version: "1", ImportCandidates: []string{"fixture"},
		SiteStartupHooks: []string{"a.pth", "b.pth"},
		SiteHookLines: []artifactpypi.SiteHookLine{
			{File: "a.pth", Line: 1, Statement: "import fixture"},
			{File: "b.pth", Line: 1, Statement: "import sys; sys.exit(0)"},
		},
		Units: []artifactpypi.PlannedObservationUnit{
			{Kind: artifactpypi.DirectImportUnit, Candidate: "fixture"},
			{Kind: artifactpypi.ActivePTHHookUnit, HookFile: "a.pth", HookLine: 1, Statement: "import fixture"},
			{Kind: artifactpypi.ActivePTHHookUnit, HookFile: "b.pth", HookLine: 1, Statement: "import sys; sys.exit(0)"},
			{Kind: artifactpypi.InstalledStartupUnit},
		},
	}
	units, err := freezePythonObservationUnits(plan, strings.Repeat("a", 64))
	if err != nil || len(units) != 4 {
		t.Fatalf("frozen units = %#v, %v", units, err)
	}
	if units[0].program != pythonDirectObservation || strings.Contains(units[0].program, "a.pth") {
		t.Fatal("direct import implicitly executes site startup")
	}
	if strings.Contains(units[1].program, "sys.exit(0)") || !strings.Contains(units[2].program, "sys.exit(0)") ||
		!strings.Contains(units[3].program, "sys.exit(0)") || !strings.Contains(units[3].program, "import fixture") {
		t.Fatalf("hook and combined startup programs are not independent: %#v", units)
	}
	seen := map[string]bool{}
	for _, unit := range units {
		if seen[unit.id] || unit.id == "" {
			t.Fatalf("repeated or empty unit identity: %#v", units)
		}
		seen[unit.id] = true
	}
	other, err := freezePythonObservationUnits(plan, strings.Repeat("b", 64))
	if err != nil || other[0].id == units[0].id {
		t.Fatal("unit identity was not bound to exact closure")
	}
}

func TestPTHDeclarativePathUsesAuthenticatedClosure(t *testing.T) {
	plan := artifactpypi.ObservationPlan{SiteHookLines: []artifactpypi.SiteHookLine{{File: "site.pth", Line: 1, Path: "extra"}}}
	manifest := closureManifest{files: map[string]closureFile{
		"extra/module.py": {scheme: artifactpypi.SchemeSite},
	}}
	if err := validatePTHPathsInClosure(plan, manifest); err != nil {
		t.Fatalf("path installed by another authenticated wheel rejected: %v", err)
	}
	manifest.files["extra/module.py"] = closureFile{scheme: artifactpypi.SchemeData}
	if err := validatePTHPathsInClosure(plan, manifest); err == nil {
		t.Fatal("non-site file satisfied active site path")
	}
	delete(manifest.files, "extra/module.py")
	if err := validatePTHPathsInClosure(plan, manifest); err == nil {
		t.Fatal("missing closure path was accepted")
	}
}
