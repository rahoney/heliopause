package pypi

import (
	"strings"
	"testing"
)

// Opaque bytes remain untrusted. The same bytes have different execution
// applicability when placed in site startup, import, or installer script roles.
func TestResourceInstalledRoleAdmission(t *testing.T) {
	for _, tc := range []struct {
		path               string
		unresolved, manual bool
		imports, units     int
	}{
		{"pkg/algorithms/flow/tests/gl1.gpickle.bz2", false, false, 1, 1},
		{"pkg/algorithms/isomorphism/tests/iso_r01_s80.A99", false, false, 1, 1},
		{"pkg/generators/atlas.dat.gz", false, false, 1, 1},
		{"pkg/renamed.opaque", false, false, 1, 1},
		{"pkg/LICENSE.opaque", false, false, 1, 1},
		{"pkg/data.json", false, false, 1, 1},
		{"example-1.0.data/purelib/pkg/data.opaque", false, false, 1, 1},
		{"example-1.0.data/platlib/pkg/data.opaque", false, false, 1, 1},
		{"example-1.0.data/data/share/hook.py", false, false, 1, 1},
		{"example-1.0.data/headers/hook.pth", false, false, 1, 1},
		{"example-1.0.data/scripts/data.json", false, true, 1, 1},
		{"active.pth", false, false, 1, 3},
		{"example-1.0.data/purelib/active.pth", false, false, 1, 3},
		{"loader.py", false, false, 2, 2},
		{"pkg/loader.pyc", true, false, 0, 0},
		{"pkg/LICENSE.pyc", true, false, 0, 0},
		{"pkg/loader.so", true, false, 0, 0},
		{"pkg/loader.cpython-314-aarch64-linux-gnu.so", true, false, 0, 0},
	} {
		t.Run(tc.path, func(t *testing.T) {
			entries := []wheelTestEntry{{name: "pkg/__init__.py", body: []byte("pass\n")}, {name: tc.path, body: []byte("import pkg\n")}}
			data := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", entries, []string{"py3-none-any"}, nil)
			inspection := inspectTestWheel(t, data, "example-1.0-py3-none-any.whl")
			plan, err := BuildObservationPlan(inspection, defaultResourcePolicy())
			if tc.unresolved {
				if err == nil || !strings.Contains(err.Error(), "unresolved") {
					t.Fatalf("required execution must fail closed: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if plan.Admissible() == tc.manual {
				t.Fatalf("unexpected admission: %+v", plan)
			}
			if !tc.manual {
				if err := ValidateTypedObservationPlan(plan, defaultResourcePolicy()); err != nil {
					t.Fatal(err)
				}
			}
			if plan.TotalImportCount != tc.imports || len(plan.Units) != tc.units {
				t.Fatalf("unexpected coverage: %+v", plan)
			}
		})
	}
}

func TestOpaqueResourceRECORDTamperingRejects(t *testing.T) {
	entries := []wheelTestEntry{{name: "pkg/__init__.py", body: []byte("pass\n")}, {name: "pkg/data.opaque", body: []byte("untrusted")}}
	data := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", entries, []string{"py3-none-any"}, func(rows []string) []string {
		for i, row := range rows {
			if strings.HasPrefix(row, "pkg/data.opaque,") {
				rows[i] = "pkg/data.opaque,sha256=" + strings.Repeat("A", 43) + ",9"
			}
		}
		return rows
	})
	if _, err := inspectTestWheelResult(data, "example-1.0-py3-none-any.whl"); err == nil {
		t.Fatal("resource classification must not bypass RECORD")
	}
}

func TestDataSchemeUsesFinalPipDestinations(t *testing.T) {
	for _, tc := range []struct {
		archive, installed string
		scheme             InstallationScheme
	}{
		{"example-1.0.data/data/share/man/man1/tool.1", "share/man/man1/tool.1", SchemeData},
		{"example-1.0.data/headers/example.h", "include/python/example/example.h", SchemeHeaders},
	} {
		data := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", []wheelTestEntry{{name: tc.archive, body: []byte("opaque")}}, []string{"py3-none-any"}, nil)
		inspection := inspectTestWheel(t, data, "example-1.0-py3-none-any.whl")
		found := false
		for _, f := range inspection.Surface.InstalledFiles {
			if f.ArchivePath == tc.archive {
				found = f.Destination == tc.installed && f.Scheme == tc.scheme
			}
		}
		if !found {
			t.Fatalf("wrong final destination: %+v", inspection.Surface.InstalledFiles)
		}
	}
	for _, name := range []string{"hook.py", "active.pth", "bin/tool", "lib/python/active.pth"} {
		data := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", []wheelTestEntry{{name: "example-1.0.data/data/" + name, body: []byte("import os")}}, []string{"py3-none-any"}, nil)
		if _, err := inspectTestWheelResult(data, "example-1.0-py3-none-any.whl"); err == nil {
			t.Fatalf("unsupported final destination accepted: %s", name)
		}
	}
	data := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", []wheelTestEntry{
		{name: "share/data.opaque", body: []byte("same")},
		{name: "example-1.0.data/data/share/data.opaque", body: []byte("same")},
	}, []string{"py3-none-any"}, nil)
	if _, err := inspectTestWheelResult(data, "example-1.0-py3-none-any.whl"); err == nil {
		t.Fatal("convergent final destinations accepted")
	}
}
