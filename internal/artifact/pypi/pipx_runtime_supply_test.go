package pypi

import "testing"

func TestPipxRuntimeSupplyKeepsTwoOriginalGraphsAndNamespaces(t *testing.T) {
	for _, mode := range []string{"valid", "missing-pytest", "missing-shared", "unreachable", "unsatisfied", "cross-owner", "stdlib-shadow", "py-package", "root-cycle"} {
		t.Run(mode, func(t *testing.T) {
			requirement := "click>=1.0.0"
			extras := []wheelTestEntry{
				{name: "pytest-1.0.0.dist-info/WHEEL", body: []byte("Wheel-Version: 1.0\nRoot-Is-Purelib: true\nTag: py3-none-any\n")},
				{name: "_pytest/__init__.py", body: []byte("pass\n")},
				{name: "py.py", body: []byte("pass\n")},
			}
			switch mode {
			case "unsatisfied":
				requirement = "click>=2.0.0"
			case "cross-owner":
				extras = append(extras, wheelTestEntry{name: "pipx/foreign.py", body: []byte("pass\n")})
			case "stdlib-shadow":
				extras = append(extras, wheelTestEntry{name: "os.py", body: []byte("pass\n")})
			case "py-package":
				extras = append(extras, wheelTestEntry{name: "py/__init__.py", body: []byte("pass\n")})
			case "root-cycle":
				requirement = "pipx>=1.0.0"
			}
			pipxRequirement := "click>=1.0.0"
			if mode == "root-cycle" {
				pipxRequirement = "pytest>=1.0.0"
			}
			wheels := map[string][]byte{
				"pipx":   toolTestWheel(t, "pipx", pipxRequirement, pipxSupplyEntry, wheelTestEntry{name: "pipx-1.0.0.dist-info/WHEEL", body: []byte("Wheel-Version: 1.0\nRoot-Is-Purelib: true\nTag: py3-none-any\n")}, wheelTestEntry{name: "pipx/__main__.py", body: []byte("def cli(): return 0\n")}),
				"pytest": toolTestWheel(t, "pytest", requirement, "", extras...),
				"click":  toolTestWheel(t, "click", "", "", wheelTestEntry{name: "click-1.0.0.dist-info/WHEEL", body: []byte("Wheel-Version: 1.0\nRoot-Is-Purelib: true\nTag: py3-none-any\n")}),
			}
			switch mode {
			case "missing-pytest":
				delete(wheels, "pytest")
			case "missing-shared":
				delete(wheels, "click")
			case "unreachable":
				wheels["pluggy"] = toolTestWheel(t, "pluggy", "", "")
			}
			packages, inputs, pages := pipxSupplySelections(t, wheels)
			closure, err := BindPipxRuntimeSupply(packages, inputs, pages)
			if mode != "valid" {
				if err == nil || closure.Valid() {
					t.Fatal("invalid runtime supply accepted")
				}
				return
			}
			if err != nil || len(closure.Wheels()) != 3 {
				t.Fatal("two-root shared dependency failed", err)
			}
			if old, err := BindPipxSupplyClosure(packages, inputs, pages); err == nil || old.Valid() {
				t.Fatal("single-root constructor accepted an unrelated second root")
			}
			body, err := closure.ZIP(inputs)
			if err != nil {
				t.Fatal(err)
			}
			distributions, units, err := pipxRuntimeZIPPlan(body)
			if err != nil || len(distributions) != 3 || len(units) != 6 {
				t.Fatal("original pytest namespaces were not planned", distributions, units, err)
			}
			facts, err := pipxRuntimeZIPFiles(body)
			if err != nil || len(facts) != len(closure.Files()) {
				t.Fatal("compressed original member set changed", err)
			}
		})
	}
}
