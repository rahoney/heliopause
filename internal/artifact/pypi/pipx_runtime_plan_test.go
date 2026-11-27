package pypi

import (
	"archive/zip"
	"bytes"
	"io"
	"sort"
	"strings"
	"testing"
)

func runtimePlanFixture(t *testing.T, mutation string) []byte {
	t.Helper()
	wheels := [][]byte{
		toolTestWheel(t, "pipx", "click>=1.0.0", pipxSupplyEntry, wheelTestEntry{name: "pipx-1.0.0.dist-info/WHEEL", body: []byte("Wheel-Version: 1.0\nRoot-Is-Purelib: true\nTag: py3-none-any\n")}, wheelTestEntry{name: "pipx/__main__.py", body: []byte("def cli(): return 0\n")}),
		toolTestWheel(t, "click", "", "[console_scripts]\nclick-tool = click.command:run\n[pytest11]\nclick-plugin = click.plugin\n", wheelTestEntry{name: "click-1.0.0.dist-info/WHEEL", body: []byte("Wheel-Version: 1.0\nRoot-Is-Purelib: true\nTag: py3-none-any\n")}, wheelTestEntry{name: "click/command.py", body: []byte("def run(): return 0\n")}, wheelTestEntry{name: "click/plugin.py", body: []byte("import pytest\n")}),
	}
	members := map[string][]byte{}
	for _, wheel := range wheels {
		archive, err := zip.NewReader(bytes.NewReader(wheel), int64(len(wheel)))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range archive.File {
			if file.FileInfo().IsDir() {
				continue
			}
			r, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(r)
			closeErr := r.Close()
			if err != nil || closeErr != nil {
				t.Fatal(err, closeErr)
			}
			members[file.Name] = body
		}
	}
	switch mutation {
	case "metadata":
		members["click-1.0.0.dist-info/METADATA"] = []byte("Name: other\nVersion: 1.0.0\n")
	case "missing-record":
		delete(members, "click-1.0.0.dist-info/RECORD")
	case "changed-module":
		members["click/plugin.py"] = []byte("changed\n")
	case "changed-record":
		members["click-1.0.0.dist-info/RECORD"] = []byte("unknown,,\n")
	case "no-main":
		for name := range members {
			if strings.HasPrefix(name, "pipx") {
				delete(members, name)
			}
		}
	case "unknown":
		members["foreign/module.py"] = []byte("unknown\n")
	case "missing-loader":
		delete(members, "pipx/__main__.py")
	case "wheel":
		members["click-1.0.0.dist-info/WHEEL"] = []byte("Wheel-Version: 1.0\nRoot-Is-Purelib: false\nTag: cp314-cp314-linux_x86_64\n")
	}
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	for _, name := range names {
		header := runtimeZIPHeader(name)
		w, err := writer.CreateHeader(&header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(members[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func TestPipxRuntimePlanPreservesOriginalRecordAndEntryPointCoverage(t *testing.T) {
	distributions, units, err := pipxRuntimeZIPPlan(runtimePlanFixture(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(distributions) != 2 || len(units) != 5 {
		t.Fatal("incomplete plan", distributions, units)
	}
	expected := map[string]ObservationCoverage{"click": RequiredObservation, "click.plugin": RequiredObservation, "click.command": PostInstallCommandObservation, "pipx": RequiredObservation, "pipx.__main__": RequiredObservation}
	for _, unit := range units {
		coverage, ok := expected[unit.Candidate]
		if !ok || unit.Kind != DirectImportUnit || unit.Coverage != coverage {
			t.Fatal("entrypoint coverage changed", unit)
		}
		delete(expected, unit.Candidate)
	}
	if len(expected) != 0 {
		t.Fatal("missing units", expected)
	}
}

func TestPipxRuntimePlanRejectsUnownedOrChangedEmbeddedInputs(t *testing.T) {
	for _, mutation := range []string{"metadata", "missing-record", "changed-module", "changed-record", "no-main", "unknown", "missing-loader", "wheel"} {
		t.Run(mutation, func(t *testing.T) {
			distributions, units, err := pipxRuntimeZIPPlan(runtimePlanFixture(t, mutation))
			if err == nil || distributions != nil || units != nil {
				t.Fatal("invalid plan accepted")
			}
		})
	}
	if plan, err := (PipxRuntimeZIP{}).ObservationPlan(); err == nil || plan.Valid() {
		t.Fatal("zero ZIP planned")
	}
	if (PipxRuntimePlan{}).Valid() {
		t.Fatal("zero plan accepted")
	}
}
