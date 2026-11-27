package npm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func npmWholeInspectionFixture(t *testing.T) (NPMProjectInspectionContext, map[domain.DependencyNodeID]domain.AcquiredArtifact, []ArchiveInventory) {
	t.Helper()
	lock, controls := npmWholeBindingFixture(t)
	byPath := map[string]NPMProjectPackage{}
	for _, pkg := range lock.Packages() {
		byPath["/"+pkg.Name+"/"+pkg.Version] = pkg
	}
	resolver := newTestResolver(t, publicRegistryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		pkg, exists := byPath[request.URL.Path]
		if !exists {
			t.Fatal("unexpected fixture query")
		}
		body, _ := json.Marshal(npmWholeSourceMetadata(t, pkg))
		return response(http.StatusOK, "application/json", string(body)), nil
	}))
	selected, err := resolver.BindNPMProjectLock(context.Background(), lock)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget("/tmp/npm-inspection-input")
	install, _ := domain.NewNPMProjectInstallContext(target)
	frozen, err := selected.InspectionContext(install, controls, "pinned-test-runtime")
	if err != nil {
		t.Fatal(err)
	}
	acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	var inventories []ArchiveInventory
	for _, artifact := range selected.Artifacts() {
		run, err := domain.NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		// Synthetic structural subjects only. No archive verification, Policy
		// decision or publication is inferred from these model fixtures.
		digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
		subject, err := domain.NewAcquiredArtifactWithIntegrity(artifact.Identity(), digest, "intake:"+run.String()+":tarball", 64, artifact.DeclaredIntegrity(), artifact.DeclaredIntegrity())
		if err != nil {
			t.Fatal(err)
		}
		node, _ := domain.ProjectDependencyNodeID(artifact.Identity())
		acquired[node] = subject
		pkg := byPath["/"+artifact.Identity().Name()+"/"+artifact.Identity().Version()]
		metadata := npmWholeSourceMetadata(t, pkg)
		delete(metadata, "dist")
		body, _ := json.Marshal(metadata)
		sum := sha256.Sum256(body)
		fileDigest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		entries := []ArchiveEntry{{Path: "package.json", Size: int64(len(body)), Mode: 0o644, Digest: fileDigest}}
		if artifact.Identity().Name() == "alpha" {
			cliSum := sha256.Sum256([]byte(npmWholeCLIContent))
			cliDigest, _ := domain.NewSHA256Digest(hex.EncodeToString(cliSum[:]))
			entries = append(entries, ArchiveEntry{Path: "cli.js", Size: int64(len(npmWholeCLIContent)), Mode: 0o755, Digest: cliDigest})
		}
		inventory, err := NewArchiveInventory(subject, entries, body)
		if err != nil {
			t.Fatal(err)
		}
		inventories = append(inventories, inventory)
	}
	return frozen, acquired, inventories
}

func TestNPMWholeInspectionBindsAllArchivePlacementsAndExactSubject(t *testing.T) {
	frozen, acquired, _ := npmWholeInspectionFixture(t)
	input, err := frozen.BindAcquired(frozen.Snapshot(), acquired)
	if err != nil || !input.Valid() || len(input.Artifacts()) != 3 {
		t.Fatal(input, err)
	}
	placements := 0
	for _, subject := range input.Artifacts() {
		instances, err := input.InstancesFor(subject)
		if err != nil {
			t.Fatal(err)
		}
		placements += len(instances)
		if subject.Identity().Name() == "shared" && len(instances) != 2 {
			t.Fatal("shared archive lost a native instance")
		}
		instances[0].Targets["foreign"] = "outside"
		again, err := input.InstancesFor(subject)
		if err != nil || again[0].Targets["foreign"] != "" {
			t.Fatal("caller mutated placement wiring")
		}
		run, _ := domain.NewRunID()
		declared, hasDeclared := subject.DeclaredIntegrity()
		observed, hasObserved := subject.ObservedIntegrity()
		if !hasDeclared || !hasObserved {
			t.Fatal("fixture lost integrity claims")
		}
		foreign, err := domain.NewAcquiredArtifactWithIntegrity(subject.Identity(), subject.Digest(), "intake:"+run.String()+":tarball", subject.SizeBytes(), declared, observed)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := input.InstancesFor(foreign); err == nil {
			t.Fatal("another intake run borrowed native context")
		}
	}
	if placements != 4 {
		t.Fatal("complete placement count differs", placements)
	}
	for key := range acquired {
		delete(acquired, key)
		break
	}
	if _, err := frozen.BindAcquired(frozen.Snapshot(), acquired); err == nil {
		t.Fatal("incomplete acquired coverage admitted")
	}
	if _, err := (NPMProjectInspectionContext{}).BindAcquired(frozen.Snapshot(), acquired); err == nil {
		t.Fatal("zero-value context admitted")
	}
}

func TestNPMWholeInspectionRejectsCopiedDigestWithDifferentSnapshot(t *testing.T) {
	frozen, acquired, _ := npmWholeInspectionFixture(t)
	original := frozen.Snapshot()
	dependencies := original.Dependencies()
	first := dependencies[0]
	changed, err := domain.NewResolvedArtifact(first.Identity(), publicRegistryURL+"other/changed.tgz", first.DeclaredIntegrity())
	if err != nil {
		t.Fatal(err)
	}
	dependencies[0] = changed
	forged, err := domain.NewProjectDependencySnapshot(original.Context(), original.Source(), original.ControlDigests(), dependencies, original.GraphDigest())
	if err != nil {
		t.Fatal(err)
	}
	if frozen.MatchesSnapshot(forged) {
		t.Fatal("copied graph digest replaced complete source facts")
	}
	if _, err := frozen.BindAcquired(forged, acquired); err == nil {
		t.Fatal("foreign snapshot acquired binding accepted")
	}
}

func TestNPMWholeInspectionBudgetChargesEveryLogicalSharedCopy(t *testing.T) {
	frozen, acquired, inventories := npmWholeInspectionFixture(t)
	input, err := frozen.BindAcquired(frozen.Snapshot(), acquired)
	if err != nil {
		t.Fatal(err)
	}
	if err := input.CheckWorkspace(inventories); err != nil {
		t.Fatal("bounded complete inventory rejected", err)
	}
	budget, err := input.BeginWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	for _, inventory := range inventories {
		if inventory.Artifact().Identity().Name() != "shared" {
			continue
		}
		entries := inventory.Entries()
		for _, name := range []string{"large-a", "large-b", "large-c"} {
			entries = append(entries, ArchiveEntry{Path: name, Size: 17 << 20, Mode: 0o644, Digest: entries[0].Digest})
		}
		large, err := NewArchiveInventory(inventory.Artifact(), entries, inventory.Manifest())
		if err != nil {
			t.Fatal(err)
		}
		// 51MiB with one store + two placements exceeds 128MiB; charging
		// only one logical placement would incorrectly admit this subject.
		if err := budget.Add(large); err == nil {
			t.Fatal("second logical placement was not charged")
		}
		if err := budget.Complete(); err == nil {
			t.Fatal("over-budget collection recovered to success")
		}
		return
	}
	t.Fatal("shared inventory missing")
}

const npmWholeCLIContent = "#!/usr/bin/env node\nconsole.log('fixture');\n"
