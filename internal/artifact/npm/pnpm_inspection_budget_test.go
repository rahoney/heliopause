package npm

import (
	"errors"
	"fmt"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
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

func TestPNPMInspectionCollectionStopsAtFirstExcessAndStaysFailed(t *testing.T) {
	f := fullInspectionLayoutFixture(t, true)
	budget, err := f.input.BeginWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	failed := false
	for _, inventory := range f.inventories {
		if inventory.Artifact().Identity().Name() == "use-sync-external-store" {
			entries := inventory.Entries()
			for part, size := range []int64{20 << 20, 20 << 20, 8 << 20} {
				entries = append(entries, ArchiveEntry{Path: fmt.Sprintf("incremental-%d", part), Size: size, Mode: 0o644, Digest: layoutDigestFixture(t, []byte("synthetic size only"))})
			}
			inventory, err = NewArchiveInventory(inventory.Artifact(), entries, inventory.Manifest())
			if err != nil {
				t.Fatal(err)
			}
			if err := budget.Add(inventory); !errors.Is(err, errPNPMInspectionBudget) {
				t.Fatal("peer expansion not rejected during collection", err)
			}
			failed = true
			break
		}
		if err := budget.Add(inventory); err != nil {
			t.Fatal(err)
		}
	}
	if !failed {
		t.Fatal("excessive subject missing")
	}
	if err := budget.Add(f.inventories[0]); !errors.Is(err, errPNPMInspectionBudget) {
		t.Fatal("failed collector reused", err)
	}
	if err := budget.Complete(); !errors.Is(err, errPNPMInspectionBudget) {
		t.Fatal("failure lost", err)
	}
}

func TestPNPMInspectionCollectionRequiresExactOnceOnlySubjects(t *testing.T) {
	f := fullInspectionLayoutFixture(t, true)
	budget, err := f.input.BeginWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	if budget.Complete() == nil {
		t.Fatal("empty collection complete")
	}
	for _, inventory := range f.inventories {
		if err := budget.Add(inventory); err != nil {
			t.Fatal(err)
		}
	}
	if err := budget.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := budget.Add(f.inventories[0]); !errors.Is(err, errPNPMInspectionLayout) {
		t.Fatal("duplicate accepted", err)
	}
	if budget.Complete() == nil {
		t.Fatal("duplicate failure not sticky")
	}
	foreign := fullInspectionLayoutFixture(t, true)
	budget, err = f.input.BeginWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	if budget.Add(foreign.inventories[0]) == nil {
		t.Fatal("foreign run accepted")
	}
	var zero PNPMInspectionBudget
	if zero.Add(f.inventories[0]) == nil || zero.Complete() == nil {
		t.Fatal("zero collector accepted")
	}
}

func TestPNPMInspectionCollectionChargesCompressedInputsBeforeStatic(t *testing.T) {
	f := fullInspectionLayoutFixture(t, false)
	original := f.input.artifacts[0]
	// Explicit structural size fixture; no archive authenticity is asserted.
	oversized, err := domain.NewAcquiredArtifact(original.Identity(), original.Digest(), original.ContentHandle(), PNPMInspectionWorkspaceLimit+1)
	if err != nil {
		t.Fatal(err)
	}
	f.input.artifacts[0] = oversized
	if _, err := f.input.BeginWorkspace(); !errors.Is(err, errPNPMInspectionBudget) {
		t.Fatal("compressed total was deferred until after collection", err)
	}
}
