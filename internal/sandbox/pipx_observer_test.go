package sandbox

import (
	"context"
	"testing"
)

func TestPipxObserverTopologyAndBudgetKeepFixedDataScope(t *testing.T) {
	for _, phase := range []pythonClosurePhase{pythonClosurePreparation, pythonClosureAnchor, pythonClosureObservation} {
		topology, ok := pipxObservationExpectedTopology(phase)
		if !ok || len(topology) != 4 {
			t.Fatal("fixed topology missing", phase)
		}
		found := false
		for _, mount := range topology {
			if mount.Mountpoint == pythonSitePath {
				t.Fatal("ZIP inherited wheel site")
			}
			if mount.Mountpoint == pipxRuntimeRoot {
				found = true
				if mount.ReadOnly != (phase != pythonClosurePreparation) || mount.Class != "workspace" || mount.FSType != "9p" || mount.Parent != "/" || mount.NoExec {
					t.Fatal("incorrect DATA mount expectation", mount)
				}
			}
		}
		if !found {
			t.Fatal("DATA mount missing")
		}
		topology[0].ReadOnly = false
		fresh, _ := pipxObservationExpectedTopology(phase)
		if !fresh[0].ReadOnly {
			t.Fatal("caller changed shared topology")
		}
		if _, ok := pythonClosureExpectedTopology(pipxObserverProfile, phase); ok {
			t.Fatal("ZIP inherited wheel phase factory")
		}
	}
	if _, ok := pipxObservationExpectedTopology("unknown"); ok {
		t.Fatal("unknown phase admitted")
	}
	if isPythonObserverProfile(pipxObserverProfile) {
		t.Fatal("ZIP gained wheel profile classification")
	}
	budget, err := newObservationTraceLedger(pipxObserverProfile)
	if err != nil || budget.maxEvents != uint64(defaultTraceBudget.events) || budget.maxBytes != defaultTraceBudget.bytes {
		t.Fatal("ZIP budget differs from existing default", err)
	}
	if !validObserverProfile(pipxObserverProfile) {
		t.Fatal("fixed profile is not registered")
	}
}

func TestPipxObserverRequiresExistingSharedBudget(t *testing.T) {
	o := &SharedObserver{}
	for _, budget := range []*observationTraceLedger{nil, {}, {maxEvents: uint64(defaultTraceBudget.events) + 1, maxBytes: defaultTraceBudget.bytes}, {maxEvents: uint64(defaultTraceBudget.events), maxBytes: defaultTraceBudget.bytes + 1}} {
		reader, err := o.StartPipxRuntimeProfileWithBudget(context.Background(), "0123456789abcdef", pythonClosureObservation, budget)
		if err == nil || reader != nil {
			t.Fatal("missing or expanded shared budget admitted")
		}
	}
}
