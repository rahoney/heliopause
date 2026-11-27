package npm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func bunRunResolutionFixture(t *testing.T, peer bool) (*Resolver, BunRunCache, domain.ArtifactReference, *int) {
	t.Helper()
	entries := bunRunCacheFixture(t)
	locator := "typescript@5.9.3"
	if peer {
		body, err := normalizeBunLockJSON(bunNativeFixture(t, "peer"))
		if err != nil {
			t.Fatal(err)
		}
		var wire map[string]any
		if err := json.Unmarshal(body, &wire); err != nil {
			t.Fatal(err)
		}
		workspace := wire["workspaces"].(map[string]any)[""].(map[string]any)
		workspace["dependencies"] = map[string]string{"tool": "1.0.0"}
		packages := wire["packages"].(map[string]any)
		original := packages["react"].([]any)
		packages["tool"] = []any{"tool@1.0.0", "", map[string]any{"dependencies": map[string]string{"react": "18.3.1", "use-sync-external-store": "1.2.0"}}, original[3]}
		packages["use-sync-external-store/react"] = append([]any(nil), original...)
		body, err = json.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		entries[3] = layoutTestEntry(entries[3].header.Name, entries[3].header.Typeflag, `{"name":"haa-bun-peer","dependencies":{"tool":"1.0.0"}}`, "")
		entries[4] = layoutTestEntry(entries[4].header.Name, entries[4].header.Typeflag, string(body), "")
		locator = "tool@1.0.0"
	}
	reference, _ := ParseReference(locator)
	cache, err := ParseBunRunCache(context.Background(), layoutTarBody(t, entries), reference, runtimeidentity.BunVersion)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]BunLockedPackage{}
	for _, pkg := range cache.lock.packages {
		byPath["/"+pkg.Name+"/"+pkg.Version] = pkg
	}
	calls := new(int)
	source := newTestResolver(t, publicRegistryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		(*calls)++
		pkg, present := byPath[request.URL.Path]
		if !present {
			t.Fatal("foreign metadata request", request.URL.Path)
		}
		return response(http.StatusOK, "application/json", bunMetadataJSON(t, bunMetadata(t, pkg))), nil
	}))
	return source, cache, reference, calls
}

func TestBunRunResolutionPreservesNativeInstancesWithoutInstallSnapshot(t *testing.T) {
	for _, peer := range []bool{false, true} {
		source, cache, reference, calls := bunRunResolutionFixture(t, peer)
		frozen, err := source.BindBunRunCache(context.Background(), cache, reference, "fixture-bun-run-runtime")
		resolution := frozen.ExecutionResolution()
		if err != nil || !frozen.Valid() || frozen.Snapshot().Valid() || len(resolution.Graph().Nodes()) != len(cache.lock.packages) || *calls != len(frozen.selection.artifacts) || resolution.RuntimeIdentity() != "bunx-isolated:fixture-bun-run-runtime" {
			t.Fatal("whole native execution context lost", err)
		}
		if peer && (len(resolution.Graph().Nodes()) != 6 || len(frozen.selection.artifacts) != 5) {
			t.Fatal("duplicate peer placement collapsed")
		}
		acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
		for _, node := range resolution.Graph().Nodes() {
			if !strings.HasPrefix(node.RecordPath(), "node_modules/") {
				t.Fatal("physical native record path lost")
			}
			run, _ := domain.NewRunID()
			digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
			subject, err := domain.NewAcquiredArtifactWithIntegrity(node.Artifact().Identity(), digest, "intake:"+run.String()+":tarball", 1, node.Artifact().DeclaredIntegrity(), node.Artifact().DeclaredIntegrity())
			if err != nil {
				t.Fatal(err)
			}
			acquired[node.Node()] = subject
		}
		input, err := frozen.BindExecutionAcquired(resolution.Graph(), acquired)
		if err != nil || !input.Valid() || len(input.Artifacts()) != len(frozen.selection.artifacts) {
			t.Fatal("whole acquired node coverage lost", err)
		}
		if peer {
			for _, subject := range input.Artifacts() {
				if subject.Identity().Name() == "react" {
					instances, err := input.InstancesFor(subject)
					if err != nil || len(instances) != 2 {
						t.Fatal("shared archive lost native instances", err)
					}
				}
			}
		}
		for node := range acquired {
			delete(acquired, node)
			break
		}
		if input, err := frozen.BindExecutionAcquired(resolution.Graph(), acquired); err == nil || input.Valid() {
			t.Fatal("partial node coverage accepted")
		}
		hybrid := frozen
		hybrid.snapshot = bunPeerInspectionContext(t).Snapshot()
		if hybrid.Valid() {
			t.Fatal("hybrid execution and installation context accepted")
		}
	}
}

func TestBunRunResolutionRejectsWrongRequestCancellationAndMissingSource(t *testing.T) {
	source, cache, reference, calls := bunRunResolutionFixture(t, false)
	for _, locator := range []string{"other", "typescript@5.9.2", "node", "typescript@latest"} {
		wrong, _ := domain.NewArtifactReference(reference.Source(), locator)
		frozen, err := source.BindBunRunCache(context.Background(), cache, wrong, "fixture-runtime")
		if err == nil || frozen.Valid() {
			t.Fatal("foreign or provisioning request survived", locator, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if frozen, err := source.BindBunRunCache(ctx, cache, reference, "fixture-runtime"); !errors.Is(err, context.Canceled) || frozen.Valid() {
		t.Fatal("canceled context survived", err)
	}
	if _, err := source.BindBunRunCache(context.Background(), BunRunCache{}, reference, "fixture-runtime"); err == nil {
		t.Fatal("zero native cache accepted")
	}
	if _, err := (*Resolver)(nil).BindBunRunCache(context.Background(), cache, reference, "fixture-runtime"); err == nil {
		t.Fatal("missing independent source accepted")
	}
	if *calls != 0 {
		t.Fatal("invalid input touched independent source")
	}
}
