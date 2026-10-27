package sandbox

import (
	"context"
	"errors"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func pipxObservationArtifact(t *testing.T, mutation string) domain.AcquiredArtifact {
	t.Helper()
	source, name, version, variant, digest, size := "github-release", "rahoney-heliopause", "v9.8.7", runtimeidentity.PipxRuntimeAssetName, runtimeidentity.PipxToolSHA256, uint64(runtimeidentity.PipxToolSize)
	switch mutation {
	case "source":
		source = "pypi"
	case "name":
		name = "pipx"
	case "new-version":
		version = "v9.8.8"
	case "variant":
		variant = "pipx.whl"
	case "digest":
		digest = strings.Repeat("0", 64)
	case "size":
		size--
	}
	src, err := domain.NewSourceID(source)
	if err != nil {
		t.Fatal(err)
	}
	id, err := domain.NewResolvedArtifactIdentity(src, name, version, variant)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := domain.NewSHA256Digest(digest)
	if err != nil {
		t.Fatal(err)
	}
	a, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(id, hash, "intake:pipx-observation-fixture:github-release", size, "sha256:"+digest)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestPipxObservationRequiresBoundPlanAndPreservesCancellation(t *testing.T) {
	a := pipxObservationArtifact(t, "")
	input, err := preparePipxObservationInput(context.Background(), a, artifactpypi.PipxRuntimePlan{})
	if err == nil || input.tool != nil || input.units != nil {
		t.Fatal("zero plan gained observation input")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	input, err = preparePipxObservationInput(ctx, a, artifactpypi.PipxRuntimePlan{})
	if !errors.Is(err, context.Canceled) || input.tool != nil || input.units != nil {
		t.Fatal("cancellation lost", err)
	}
}

func TestPipxObservationUnitsKeepSubjectProgramAndCoverageBindings(t *testing.T) {
	a := pipxObservationArtifact(t, "")
	declared := []artifactpypi.PlannedObservationUnit{{Kind: artifactpypi.DirectImportUnit, Candidate: "pipx"}, {Kind: artifactpypi.DirectImportUnit, Candidate: "userpath.cli", Coverage: artifactpypi.PostInstallCommandObservation}}
	units, err := freezePipxObservationUnits(a, declared)
	if err != nil || len(units) != 2 {
		t.Fatal(err)
	}
	if units[0].coverage != artifactpypi.RequiredObservation || units[1].coverage != artifactpypi.PostInstallCommandObservation || units[0].program != pipxDirectObservation || units[0].ownerDigest != a.Digest().String() {
		t.Fatal("unit owner/program/coverage changed", units)
	}
	changed, err := freezePipxObservationUnits(pipxObservationArtifact(t, "new-version"), declared)
	if err != nil || changed[0].id == units[0].id {
		t.Fatal("HAA version was not bound to the unit", err)
	}
	declared[1].Coverage = artifactpypi.RequiredObservation
	changed, err = freezePipxObservationUnits(a, declared)
	if err != nil || changed[0].id != units[0].id || changed[1].id == units[1].id {
		t.Fatal("coverage was not bound to its independent unit", err)
	}
}

func TestPipxObservationUnitsRejectUnsupportedOrRepeatedUnits(t *testing.T) {
	for _, mode := range []string{"empty", "count", "kind", "candidate", "duplicate", "coverage", "hook", "statement"} {
		t.Run(mode, func(t *testing.T) {
			declared := []artifactpypi.PlannedObservationUnit{{Kind: artifactpypi.DirectImportUnit, Candidate: "pipx"}}
			switch mode {
			case "empty":
				declared = nil
			case "count":
				declared = make([]artifactpypi.PlannedObservationUnit, artifactpypi.PublicPyPIProfile().ResourcePolicy().MaxObservationImportsPerArtifact()+1)
			case "kind":
				declared[0].Kind = artifactpypi.ActivePTHHookUnit
			case "candidate":
				declared[0].Candidate = "pipx/../os"
			case "duplicate":
				declared = append(declared, declared[0])
			case "coverage":
				declared[0].Coverage = "unknown"
			case "hook":
				declared[0].HookFile = "startup.pth"
			case "statement":
				declared[0].Statement = "import os"
			}
			units, err := freezePipxObservationUnits(pipxObservationArtifact(t, ""), declared)
			if err == nil || units != nil {
				t.Fatal("unsupported units were frozen")
			}
		})
	}
}
