package npm

import (
	"errors"
	"fmt"
	"testing"
)

func TestBunWorkspaceChargesUniqueCacheAndEveryRepeatedInstance(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		f := bunLayoutFixture(t, repeated)
		for n, inventory := range f.inventories {
			name := inventory.Artifact().Identity().Name()
			if name != "react" && name != "@two/lib" {
				continue
			}
			entries := inventory.Entries()
			for part, size := range []int64{20 << 20, 20 << 20, 8 << 20} {
				entries = append(entries, ArchiveEntry{Path: fmt.Sprintf("large-%d", part), Size: size, Mode: 0o644, Digest: layoutDigestFixture(t, []byte("synthetic budget only"))})
			}
			var err error
			f.inventories[n], err = NewArchiveInventory(inventory.Artifact(), entries, inventory.Manifest())
			if err != nil {
				t.Fatal(err)
			}
		}
		err := f.input.CheckWorkspace(f.inventories)
		if repeated && !errors.Is(err, errPNPMInspectionBudget) || !repeated && err != nil {
			t.Fatal("logical copies charged incorrectly", repeated, err)
		}
	}
}

func TestBunWorkspaceCollectionFailureStaysFailed(t *testing.T) {
	f := bunLayoutFixture(t, false)
	budget, err := f.input.BeginWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	if err := budget.Add(f.inventories[0]); err != nil {
		t.Fatal(err)
	}
	if err := budget.Add(f.inventories[0]); err == nil {
		t.Fatal("duplicate collection admitted")
	}
	for _, inventory := range f.inventories[1:] {
		if err := budget.Add(inventory); err == nil {
			t.Fatal("failed collection recovered")
		}
	}
	if err := budget.Complete(); err == nil {
		t.Fatal("failed collection completed")
	}
	if _, err := (BunInspectionInput{}).BeginWorkspace(); err == nil {
		t.Fatal("zero workspace")
	}
}
