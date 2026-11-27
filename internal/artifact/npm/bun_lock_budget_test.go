package npm

import (
	"fmt"
	"strings"
	"testing"
)

func TestBunTreeDepthAndCompleteReachability(t *testing.T) {
	for _, depth := range []int{32, 33} {
		rows := map[string][]any{}
		parts := []string{}
		for i := 0; i < depth; i++ {
			name := fmt.Sprintf("p%d", i)
			parts = append(parts, name)
			info := map[string]any{}
			if i+1 < depth {
				info["dependencies"] = map[string]string{fmt.Sprintf("p%d", i+1): "1.0.0"}
			}
			rows[strings.Join(parts, "/")] = bunSyntheticRow(name, "1.0.0", info)
		}
		manifest, body := bunSyntheticLock(t, map[string]string{"p0": "1.0.0"}, rows)
		lock, err := parseBunTestLock(t, manifest, body, true)
		if (err == nil) != (depth == 32) || depth == 32 && len(lock.Packages()) != depth || depth == 33 && lock.valid {
			t.Fatal("package-unit depth boundary differs", depth, err)
		}
	}
	for _, fault := range []string{"unreachable", "missing-parent", "conflicting-archive"} {
		t.Run(fault, func(t *testing.T) {
			roots, rows := bunScopedRows(true)
			switch fault {
			case "unreachable":
				rows["extra"] = bunSyntheticRow("extra", "1.0.0", map[string]any{})
			case "missing-parent":
				rows["missing/extra"] = bunSyntheticRow("extra", "1.0.0", map[string]any{})
			case "conflicting-archive":
				rows["@one/root/@two/lib"][2] = map[string]any{"hasInstallScript": true}
			}
			manifest, body := bunSyntheticLock(t, roots, rows)
			if selected, err := parseBunTestLock(t, manifest, body, true); err == nil || selected.valid {
				t.Fatal("incomplete or contradictory census accepted")
			}
		})
	}
}

func TestBunTreeEdgeBudget(t *testing.T) {
	for _, count := range []int{126, 129} {
		rows := map[string][]any{}
		dependencies, roots := map[string]string{}, map[string]string{"root": "1.0.0"}
		for i := 0; i < count; i++ {
			targets := map[string]string{}
			for j := 0; j < count; j++ {
				name := fmt.Sprintf("target%d", j)
				targets[name] = "1.0.0"
				rows[name] = bunSyntheticRow(name, "1.0.0", map[string]any{})
			}
			name := fmt.Sprintf("source%d", i)
			rows[name] = bunSyntheticRow(name, "1.0.0", map[string]any{"dependencies": targets})
			dependencies[name] = "1.0.0"
		}
		rows["root"] = bunSyntheticRow("root", "1.0.0", map[string]any{"dependencies": dependencies})
		manifest, body := bunSyntheticLock(t, roots, rows)
		selected, err := parseBunTestLock(t, manifest, body, true)
		if (err == nil) != (count == 126) || count == 129 && selected.valid {
			t.Fatal("whole-edge budget differs", count, err)
		}
	}
}

func TestBunPeerComparisonBudget(t *testing.T) {
	for _, count := range []int{128, 129} {
		rows := map[string][]any{"shared": bunSyntheticRow("shared", "1.0.0", map[string]any{})}
		dependencies := map[string]string{"shared": "1.0.0"}
		for i := 0; i < count; i++ {
			consumer := fmt.Sprintf("consumer%d", i)
			// All ranges match only1.0.0 among the retained versions, but
			// each is distinct so the candidate-comparison budget is real.
			spec := fmt.Sprintf("1.0.0 || >%d.0.0 <%d.0.1", 1000+i, 1000+i)
			rows[consumer] = bunSyntheticRow(consumer, "1.0.0", map[string]any{"peerDependencies": map[string]string{"shared": spec}})
			dependencies[consumer] = "1.0.0"
			if i > 0 {
				holder := fmt.Sprintf("holder%d", i)
				version := fmt.Sprintf("%d.0.0", i+1)
				rows[holder] = bunSyntheticRow(holder, "1.0.0", map[string]any{"dependencies": map[string]string{"shared": version}})
				rows[holder+"/shared"] = bunSyntheticRow("shared", version, map[string]any{})
				dependencies[holder] = "1.0.0"
			}
		}
		rows["root"] = bunSyntheticRow("root", "1.0.0", map[string]any{"dependencies": dependencies})
		manifest, body := bunSyntheticLock(t, map[string]string{"root": "1.0.0"}, rows)
		selected, err := parseBunTestLock(t, manifest, body, true)
		if (err == nil) != (count == 128) || count == 129 && selected.valid {
			t.Fatal("peer cross-product budget differs", count, err)
		}
	}
}

func TestBunConstraintsUseReviewedStableSubset(t *testing.T) {
	for _, test := range []struct {
		spec, version string
		want          bool
	}{
		{"^0.0.3", "0.0.3", true}, {"^0.0.3", "0.0.4", false},
		{"~1.2.0", "1.2.99", true}, {"~1.2.0", "1.3.0", false},
		{"^16.8.0 || ^17.0.0 || ^18.0.0", "18.3.1", true},
		{"^16.8.0 || ^17.0.0 || ^18.0.0", "19.0.0", false},
		{"1.0.0-beta.1", "1.0.0-beta.1", true}, {">=1.0.0-0", "1.0.0-beta.1", false},
		{"latest", "1.0.0", false}, {"file:../escape", "1.0.0", false},
		{"1.0.0", "01.0.0", false}, {"1.0.0", "v1.0.0", false},
	} {
		if bunConstraintMatches(test.spec, test.version) != test.want {
			t.Fatal("constraint subset differs", test.spec, test.version)
		}
	}
}
