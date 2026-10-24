package npm

import (
	"errors"
	"fmt"
	"testing"
)

func TestPNPMInspectionWorkspaceChargesEveryPeerInstance(t *testing.T) {
	for _, multiple := range []bool{false, true} {
		f := fullInspectionLayoutFixture(t, multiple)
		if err := f.input.CheckWorkspace(f.inventories); err != nil {
			t.Fatal("small complete input rejected", err)
		}
		// Structural, explicitly synthetic footprint data: no large archive is
		// allocated and these facts do not assert measured source authenticity.
		for index, inventory := range f.inventories {
			if inventory.Artifact().Identity().Name() != "use-sync-external-store" {
				continue
			}
			entries := inventory.Entries()
			for part, size := range []int64{20 << 20, 20 << 20, 8 << 20} {
				entries = append(entries, ArchiveEntry{Path: fmt.Sprintf("footprint-%d", part), Size: size, Mode: 0o644, Digest: layoutDigestFixture(t, []byte("synthetic footprint"))})
			}
			var err error
			f.inventories[index], err = NewArchiveInventory(inventory.Artifact(), entries, inventory.Manifest())
			if err != nil {
				t.Fatal("invalid budget fixture", err)
			}
		}
		err := f.input.CheckWorkspace(f.inventories)
		if multiple {
			// Store48MiB + two logical instances96MiB exceeds128MiB.
			if !errors.Is(err, errPNPMInspectionBudget) {
				t.Fatal("shared archive peer cost collapsed", err)
			}
		} else if err != nil {
			// Store48MiB + one instance48MiB plus small other inputs fits.
			t.Fatal("single context was charged extra peer instances", err)
		}
	}
}

func TestPNPMInspectionWorkspaceRequiresExactCompleteCoverage(t *testing.T) {
	f := fullInspectionLayoutFixture(t, false)
	for _, inventory := range [][]ArchiveInventory{nil, f.inventories[:3], {f.inventories[0], f.inventories[0], f.inventories[2], f.inventories[3]}} {
		if err := f.input.CheckWorkspace(inventory); !errors.Is(err, errPNPMInspectionLayout) {
			t.Fatal("bad coverage admitted before resource work", err)
		}
	}
	foreign := fullInspectionLayoutFixture(t, false)
	if err := f.input.CheckWorkspace(foreign.inventories); !errors.Is(err, errPNPMInspectionLayout) {
		t.Fatal("different intake runs admitted", err)
	}
	if err := (PNPMInspectionInput{}).CheckWorkspace(f.inventories); err == nil {
		t.Fatal("zero input admitted")
	}
}
