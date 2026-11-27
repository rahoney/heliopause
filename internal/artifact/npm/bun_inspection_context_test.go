package npm

import (
	"github.com/rahoney/heliopause/internal/core/domain"
	"strings"
	"testing"
)

func bunPeerInspectionContext(t *testing.T) BunInspectionContext {
	t.Helper()
	manifest, lock, present := bunActualControls(t, "peer")
	bound, controls := bunTestSelection(t, manifest, lock, present)
	frozen, err := bound.InspectionContext(pnpmSnapshotContext(t, "/tmp/haa-bun-inspection", true), controls, "fixture-runtime:bun")
	if err != nil || !frozen.Valid() {
		t.Fatal("Bun context unavailable", err)
	}
	return frozen
}

func TestBunInspectionContextKeepsSnapshotControlsAndExactAcquiredSet(t *testing.T) {
	frozen := bunPeerInspectionContext(t)
	if !frozen.MatchesSnapshot(frozen.Snapshot()) || frozen.RuntimeIdentity() != "fixture-runtime:bun" {
		t.Fatal("own frozen snapshot lost")
	}
	acquired := pnpmAcquiredInspectionFixture(t, frozen)
	input, err := frozen.BindAcquired(frozen.Snapshot(), acquired)
	if err != nil || !input.Valid() || len(input.Artifacts()) != 4 {
		t.Fatalf("complete acquired input lost: %v", err)
	}
	for node := range acquired {
		delete(acquired, node)
	}
	copy := input.Artifacts()
	copy[0] = domain.AcquiredArtifact{}
	if len(input.Artifacts()) != 4 || input.Artifacts()[0].Identity().Name() == "" {
		t.Fatal("caller mutated acquired input")
	}
	controls := frozen.Controls()
	controls[0] = domain.ProjectControlFile{}
	body := frozen.Controls()[0].Body()
	body[0] ^= 1
	if frozen.Controls()[0].Name() == "" || string(frozen.Controls()[0].Body()) == string(body) {
		t.Fatal("caller mutated controls")
	}
	for _, artifact := range input.Artifacts() {
		instances, err := input.InstancesFor(artifact)
		if err != nil || len(instances) != 1 {
			t.Fatalf("peer instance coverage lost: %v", err)
		}
		if artifact.Identity().Name() == "use-sync-external-store" && instances[0].Key != "use-sync-external-store" {
			t.Fatal("native tree key collapsed")
		}
		for key := range instances[0].Dependencies {
			instances[0].Dependencies[key] = "foreign@0.0.0"
		}
		again, err := input.InstancesFor(artifact)
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range again[0].Dependencies {
			if value == "foreign@0.0.0" {
				t.Fatal("caller mutated instance wiring")
			}
		}
	}
	if _, err := input.InstancesFor(domain.AcquiredArtifact{}); err == nil {
		t.Fatal("foreign acquired subject accepted")
	}
	old := input.Artifacts()[0]
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	declared, _ := old.DeclaredIntegrity()
	observed, _ := old.ObservedIntegrity()
	foreign, err := domain.NewAcquiredArtifactWithIntegrity(old.Identity(), old.Digest(), "intake:"+run.String()+":tarball", old.SizeBytes(), declared, observed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := input.InstancesFor(foreign); err == nil {
		t.Fatal("same identity from a different acquired run accepted")
	}
	if _, err := (BunInspectionContext{}).BindAcquired(frozen.Snapshot(), nil); err == nil {
		t.Fatal("zero context accepted")
	}
	if _, err := (BunInspectionInput{}).InstancesFor(input.Artifacts()[0]); err == nil {
		t.Fatal("zero input accepted")
	}
}

func TestBunInspectionContextRejectsForgedSnapshotWithCopiedDigest(t *testing.T) {
	frozen := bunPeerInspectionContext(t)
	original := frozen.Snapshot()
	for _, change := range []string{"target", "mode", "controls", "url", "sri", "missing-dependency", "empty", "source"} {
		t.Run(change, func(t *testing.T) {
			install, source, controls, dependencies := original.Context(), original.Source(), original.ControlDigests(), original.Dependencies()
			var err error
			switch change {
			case "target":
				install = pnpmSnapshotContext(t, "/tmp/haa-bun-other-target", true)
			case "mode":
				install = pnpmSnapshotContext(t, original.Context().Target().String(), false)
			case "controls":
				digest, e := domain.NewSHA256Digest(strings.Repeat("f", 64))
				if e != nil {
					t.Fatal(e)
				}
				controls[0], err = domain.NewProjectControlDigest(controls[0].Name(), digest)
			case "url", "sri":
				old := dependencies[0]
				url, sri := old.AcquisitionLocator(), old.DeclaredIntegrity()
				if change == "url" {
					url = publicRegistryURL + "foreign.tgz"
				} else {
					sri = "sha512-different"
				}
				dependencies[0], err = domain.NewResolvedArtifact(old.Identity(), url, sri)
			case "missing-dependency":
				dependencies = dependencies[1:]
			case "source":
				source, err = domain.NewSourceID("pypi")
				for n, old := range dependencies {
					identity, e := domain.NewResolvedArtifactIdentity(source, old.Identity().Name(), old.Identity().Version(), old.Identity().Variant())
					if e != nil {
						t.Fatal(e)
					}
					dependencies[n], err = domain.NewResolvedArtifact(identity, old.AcquisitionLocator(), old.DeclaredIntegrity())
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			var candidate domain.ProjectDependencySnapshot
			if change == "empty" {
				candidate, err = domain.NewDependencyFreeProjectSnapshot(install, source, controls, original.GraphDigest())
			} else {
				candidate, err = domain.NewProjectDependencySnapshot(install, source, controls, dependencies, original.GraphDigest())
			}
			if err != nil || !candidate.Valid() {
				t.Fatalf("negative fixture failed before the tested boundary: %v", err)
			}
			if frozen.MatchesSnapshot(candidate) {
				t.Fatal("copied digest admitted a different normalized snapshot")
			}
			if _, err := frozen.BindAcquired(candidate, pnpmAcquiredInspectionFixture(t, frozen)); err == nil {
				t.Fatal("forged snapshot bound acquired input")
			}
		})
	}
}

func TestBunInspectionContextRejectsWrongOrPartialAcquiredCoverage(t *testing.T) {
	frozen := bunPeerInspectionContext(t)
	for _, change := range []string{"missing", "extra", "wrong-node", "identity", "variant", "declared", "observed", "missing-observed", "size", "handle", "shared-run"} {
		t.Run(change, func(t *testing.T) {
			acquired := pnpmAcquiredInspectionFixture(t, frozen)
			first, second := frozen.Snapshot().Dependencies()[0], frozen.Snapshot().Dependencies()[1]
			node, err := domain.ProjectDependencyNodeID(first.Identity())
			if err != nil {
				t.Fatal(err)
			}
			secondNode, err := domain.ProjectDependencyNodeID(second.Identity())
			if err != nil {
				t.Fatal(err)
			}
			original := acquired[node]
			identity, digest, handle, size := original.Identity(), original.Digest(), original.ContentHandle(), original.SizeBytes()
			declared, _ := original.DeclaredIntegrity()
			observed, _ := original.ObservedIntegrity()
			switch change {
			case "missing":
				delete(acquired, node)
			case "extra", "wrong-node":
				foreign, e := domain.NewDependencyNodeID("foreign")
				if e != nil {
					t.Fatal(e)
				}
				acquired[foreign] = original
				if change == "wrong-node" {
					delete(acquired, node)
				}
			case "identity", "variant":
				name, variant := identity.Name(), identity.Variant()
				if change == "identity" {
					name = "foreign"
				} else {
					variant = "wheel"
				}
				identity, err = domain.NewResolvedArtifactIdentity(identity.Source(), name, identity.Version(), variant)
			case "declared":
				declared = "sha512-different"
			case "observed":
				observed = "sha512-different"
			case "missing-observed":
				observed = ""
			case "size":
				size = 0
			case "handle":
				handle = "intake:invalid:tarball"
			case "shared-run":
				handle = acquired[secondNode].ContentHandle()
			}
			if err != nil {
				t.Fatal(err)
			}
			if change != "missing" && change != "extra" && change != "wrong-node" {
				acquired[node], err = domain.NewAcquiredArtifactWithIntegrity(identity, digest, handle, size, declared, observed)
				if err != nil {
					t.Fatalf("negative fixture failed before tested boundary: %v", err)
				}
			}
			if input, err := frozen.BindAcquired(frozen.Snapshot(), acquired); err == nil || input.Valid() {
				t.Fatal("wrong or incomplete acquired set admitted")
			}
		})
	}
}

func TestBunInspectionContextRequiresBoundControlsAndExplicitEmpty(t *testing.T) {
	manifest, lock, present := bunActualControls(t, "empty")
	bound, controls := bunTestSelection(t, manifest, lock, present)
	install := pnpmSnapshotContext(t, "/tmp/haa-bun-empty-inspection", true)
	frozen, err := bound.InspectionContext(install, controls, "fixture-runtime:bun")
	if err != nil || !frozen.Valid() || !frozen.Snapshot().DependencyFree() {
		t.Fatalf("explicit empty context lost: %v", err)
	}
	input, err := frozen.BindAcquired(frozen.Snapshot(), nil)
	if err != nil || !input.Valid() || len(input.Artifacts()) != 0 {
		t.Fatalf("explicit empty binding lost: %v", err)
	}
	if _, err := (BunSelection{}).InspectionContext(install, controls, "fixture-runtime:bun"); err == nil {
		t.Fatal("unbound selection accepted")
	}
	if _, err := bound.InspectionContext(install, controls[:1], "fixture-runtime:bun"); err == nil {
		t.Fatal("partial controls accepted")
	}
	if _, err := bound.InspectionContext(install, controls, ""); err == nil {
		t.Fatal("missing runtime accepted")
	}
}

func TestBunInspectionContextPreservesRepeatedScopedArchiveInstances(t *testing.T) {
	roots, rows := bunScopedRows(true)
	manifest, lock := bunSyntheticLock(t, roots, rows)
	bound, controls := bunTestSelection(t, manifest, lock, true)
	frozen, err := bound.InspectionContext(pnpmSnapshotContext(t, "/tmp/haa-bun-repeated-inspection", true), controls, "fixture-runtime:bun")
	if err != nil {
		t.Fatal(err)
	}
	input, err := frozen.BindAcquired(frozen.Snapshot(), pnpmAcquiredInspectionFixture(t, frozen))
	if err != nil {
		t.Fatal(err)
	}
	if len(input.Artifacts()) != 3 {
		t.Fatal("repeated archive was not deduplicated")
	}
	total := 0
	for _, artifact := range input.Artifacts() {
		instances, err := input.InstancesFor(artifact)
		if err != nil {
			t.Fatal(err)
		}
		total += len(instances)
		if artifact.Identity().Name() == "@two/lib" && len(instances) != 2 {
			t.Fatal("distinct scoped tree placements collapsed")
		}
		for n := range instances {
			instances[n].Targets = map[string]string{"foreign": "changed"}
			instances[n].Bins = map[string]string{"foreign": "changed"}
		}
		again, err := input.InstancesFor(artifact)
		if err != nil {
			t.Fatal(err)
		}
		for _, instance := range again {
			if instance.Targets["foreign"] != "" || instance.Bins["foreign"] != "" {
				t.Fatal("caller mutated frozen instance")
			}
		}
	}
	if total != 4 {
		t.Fatal("incomplete instance coverage")
	}
	for _, name := range []string{"single", "transitive"} {
		manifest, lock, present := bunActualControls(t, name)
		bound, controls := bunTestSelection(t, manifest, lock, present)
		frozen, err := bound.InspectionContext(pnpmSnapshotContext(t, "/tmp/haa-bun-fixture", true), controls, "fixture-runtime:bun")
		if err != nil {
			t.Fatal(err)
		}
		if input, err := frozen.BindAcquired(frozen.Snapshot(), pnpmAcquiredInspectionFixture(t, frozen)); err != nil || !input.Valid() {
			t.Fatal(name, err)
		}
	}
}
