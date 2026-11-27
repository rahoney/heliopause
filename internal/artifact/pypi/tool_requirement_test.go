package pypi

import "testing"

func TestToolRequirementSameConjunctionPreservesAllClauses(t *testing.T) {
	for _, item := range []struct {
		left, right string
		want        bool
	}{
		{"tool>=1,<2", "tool<2,>=1", true},
		{"tool>=1.0,<=1.0", "tool<=1.0,>=1.0", true},
		{"tool>=1,>=1,<2", "tool<2,>=1", true},
		{"tool", "tool", true},
		{"tool==1.0", "tool==1.0", true},
		{"tool>=1,<2", "tool>=1,<3", false},
		{"tool>=1,<2", "tool>1,<2", false},
		{"tool>=1,<2", "tool>=1,<=2", false},
		{"tool>=1,<2", "tool>=1,==1.5", false},
		{"tool>=1,<2", "other>=1,<2", false},
		{"tool", "tool>=1", false},
		{"tool==1.0.0.*", "tool==1.0.0.0.*", false},
		{"tool~=1.0", "tool~=1.0.0", false},
		{"tool>=1,<=1", "tool==1", false},
	} {
		left, err := ParseUpgradeRequirement(item.left)
		if err != nil {
			t.Fatal(item.left, item.right, err)
		}
		right, err := ParseUpgradeRequirement(item.right)
		if err != nil {
			t.Fatal(item.left, item.right, err)
		}
		if left.SameConstraints(right) != item.want || right.SameConstraints(left) != item.want {
			t.Fatalf("constraint set comparison changed: %s / %s", item.left, item.right)
		}
	}
	public, err := ParseUpgradeRequirement("torch>=1,<2")
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := PyTorchProfile("cpu")
	if !ok {
		t.Fatal("pinned named source missing")
	}
	other, err := ParseUpgradeRequirementForSource("torch>=1,<2", profile.Source())
	if err != nil {
		t.Fatal(err)
	}
	if public.SameConstraints(other) || public.SameConstraints(UpgradeRequirement{}) {
		t.Fatal("different source or empty requirement matched")
	}
}
