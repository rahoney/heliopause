package pypi

import "testing"

func TestInspectionPrerequisiteUsesInstalledRolesAndRejectsExpansion(t *testing.T) {
	parse := func(project string, entries ...wheelTestEntry) WheelInspection {
		archive := recordedWheelArchive(t, project, "1.0", project+"-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
		return inspectTestWheel(t, archive, project+"-1.0-py3-none-any.whl")
	}
	original := parse("target", wheelTestEntry{name: "target/__init__.py", body: []byte("pass\n")})
	for _, tc := range []struct {
		name     string
		project  string
		entries  []wheelTestEntry
		requires bool
		want     bool
	}{
		{"leaf", "support", []wheelTestEntry{{name: "support/__init__.py", body: []byte("pass\n")}}, false, true},
		{"opaque data", "support", []wheelTestEntry{{name: "support/__init__.py", body: []byte("pass\n")}, {name: "support/data.unknown", body: []byte("untrusted")}}, false, true},
		{"replacement", "target", []wheelTestEntry{{name: "different/__init__.py", body: []byte("pass\n")}}, false, false},
		{"exact collision", "support", []wheelTestEntry{{name: "target/__init__.py", body: []byte("pass\n")}}, false, false},
		{"import shadow", "support", []wheelTestEntry{{name: "target/other.py", body: []byte("pass\n")}}, false, false},
		{"file directory collision", "support", []wheelTestEntry{{name: "target/__init__.py/child", body: []byte("opaque")}}, false, false},
		{"startup", "support", []wheelTestEntry{{name: "support/__init__.py", body: []byte("pass\n")}, {name: "startup.pth", body: []byte("import support\n")}}, false, false},
		{"unsupported script", "support", []wheelTestEntry{{name: "support-1.0.data/scripts/tool", body: []byte("#!/bin/sh\nexit 0\n")}}, false, false},
		{"transitive or self cycle", "support", []wheelTestEntry{{name: "support/__init__.py", body: []byte("pass\n")}}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wheel := parse(tc.project, tc.entries...)
			if tc.requires {
				wheel.RequiresDist = []string{"support>=1"}
			}
			plan, err := ValidateInspectionPrerequisite(wheel, []WheelInspection{original}, defaultResourcePolicy())
			if (err == nil) != tc.want || tc.want && (!plan.Admissible() || len(plan.Units) == 0) {
				t.Fatalf("plan=%+v error=%v", plan, err)
			}
		})
	}
}
