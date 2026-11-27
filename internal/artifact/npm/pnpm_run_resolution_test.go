package npm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"go.yaml.in/yaml/v3"
)

func pnpmRunResolutionFixture(t *testing.T, peer bool) (*Resolver, PNPMRunCache, domain.ArtifactReference, *int) {
	t.Helper()
	manifest, body, locator := pnpmSingleManifest, pnpmFixture(t, "single"), "is-number@6.0.0"
	if peer {
		var wire pnpmLockWire
		if err := yaml.Unmarshal(pnpmFixture(t, "peer"), &wire); err != nil {
			t.Fatal(err)
		}
		wire.Importers["."] = pnpmImporterWire{Dependencies: map[string]pnpmRootWire{"tool": {"1.0.0", "1.0.0"}}}
		wire.Packages["react@17.0.2"] = wire.Packages["react@18.3.1"]
		wire.Packages["bridge@1.0.0"] = wire.Packages["react@18.3.1"]
		wire.Packages["tool@1.0.0"] = wire.Packages["react@18.3.1"]
		wire.Snapshots["react@17.0.2"] = pnpmSnapshotWire{}
		wire.Snapshots["bridge@1.0.0"] = pnpmSnapshotWire{Dependencies: map[string]string{"use-sync-external-store": "1.2.0(react@17.0.2)"}}
		wire.Snapshots["tool@1.0.0"] = pnpmSnapshotWire{Dependencies: map[string]string{"use-sync-external-store": "1.2.0(react@18.3.1)", "bridge": "1.0.0"}}
		wire.Snapshots["use-sync-external-store@1.2.0(react@17.0.2)"] = pnpmSnapshotWire{Dependencies: map[string]string{"react": "17.0.2"}}
		var err error
		body, err = yaml.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		manifest, locator = `{"dependencies":{"tool":"1.0.0"}}`, "tool@1.0.0"
	}
	entries := pnpmRunCacheFixture(t)
	entries[4] = layoutTestEntry(entries[4].header.Name, entries[4].header.Typeflag, manifest, "")
	entries[5] = layoutTestEntry(entries[5].header.Name, entries[5].header.Typeflag, string(body), "")
	reference, err := ParseReference(locator)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := ParsePNPMRunCache(context.Background(), layoutTarBody(t, entries), reference, runtimeidentity.PNPMVersion)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]PNPMLockedPackage{}
	for _, pkg := range cache.lock.packages {
		byPath["/"+pkg.Name+"/"+pkg.Version] = pkg
	}
	calls := new(int)
	source := newTestResolver(t, publicRegistryURL, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		(*calls)++
		pkg, present := byPath[request.URL.Path]
		if !present {
			t.Fatal("foreign exact metadata request", request.URL.Path)
		}
		return response(http.StatusOK, "application/json", pnpmMetadataBody(t, pkg, publicRegistryURL+pkg.Name+"/-/"+pkg.Name+"-"+pkg.Version+".tgz")), nil
	}))
	return source, cache, reference, calls
}

func TestPNPMRunResolutionPreservesEveryPeerInstanceAndAuthenticatedOriginal(t *testing.T) {
	for _, peer := range []bool{false, true} {
		source, cache, reference, calls := pnpmRunResolutionFixture(t, peer)
		frozen, err := source.BindPNPMRunCache(context.Background(), cache, reference, "fixture-pnpm-run-runtime")
		resolution := frozen.ExecutionResolution()
		if err != nil || !frozen.Valid() || frozen.Snapshot().Valid() || len(resolution.Graph().Nodes()) != len(cache.lock.snapshots) || *calls != len(cache.lock.packages) || resolution.RuntimeIdentity() != "pnpm-dlx-isolated:fixture-pnpm-run-runtime" {
			t.Fatal("instance coverage or unique source binding lost", err)
		}
		byID := map[domain.DependencyNodeID]string{}
		shared := 0
		for _, node := range resolution.Graph().Nodes() {
			if !strings.HasPrefix(node.RecordPath(), "node_modules/.pnpm/") || !strings.HasSuffix(node.RecordPath(), "/node_modules/"+node.Artifact().Identity().Name()) {
				t.Fatal("native instance path lost")
			}
			byID[node.Node()] = node.RecordPath()
			if node.Artifact().Identity().Name() == "use-sync-external-store" {
				shared++
			}
		}
		if peer && (shared != 2 || len(byID) != 8 || *calls != 7 || len(resolution.Graph().Edges()) != 7) {
			t.Fatal("shared tarball collapsed distinct peer wiring", shared, len(byID), *calls, len(resolution.Graph().Edges()))
		}
		again, err := source.BindPNPMRunCache(context.Background(), cache, reference, "fixture-pnpm-run-runtime")
		if err != nil || !reflect.DeepEqual(resolution, again.ExecutionResolution()) {
			t.Fatal("same frozen inputs changed resolution", err)
		}
		acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
		for _, node := range resolution.Graph().Nodes() {
			run, runErr := domain.NewRunID()
			sum := sha256.Sum256([]byte(node.Artifact().Identity().Name()))
			digest, digestErr := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
			subject, subjectErr := domain.NewAcquiredArtifactWithIntegrity(node.Artifact().Identity(), digest, "intake:"+run.String()+":tarball", 1, node.Artifact().DeclaredIntegrity(), node.Artifact().DeclaredIntegrity())
			if runErr != nil || digestErr != nil || subjectErr != nil {
				t.Fatal(runErr, digestErr, subjectErr)
			}
			acquired[node.Node()] = subject
		}
		input, err := frozen.BindExecutionAcquired(resolution.Graph(), acquired)
		if err != nil || !input.Valid() || len(input.Artifacts()) != len(cache.lock.packages) {
			t.Fatal("execution acquired input lost unique original coverage", err)
		}
		if _, err := frozen.BindAcquired(frozen.Snapshot(), acquired); err == nil {
			t.Fatal("execution became an installation snapshot")
		}
		if peer {
			var first domain.AcquiredArtifact
			for node, subject := range acquired {
				if first.ContentHandle() == "" {
					first = subject
					continue
				}
				if subject.Identity() == first.Identity() {
					continue
				}
				declared, _ := subject.DeclaredIntegrity()
				reused, err := domain.NewAcquiredArtifactWithIntegrity(subject.Identity(), subject.Digest(), first.ContentHandle(), subject.SizeBytes(), declared, declared)
				if err != nil {
					t.Fatal(err)
				}
				acquired[node] = reused
				if _, err := frozen.BindExecutionAcquired(resolution.Graph(), acquired); err == nil {
					t.Fatal("distinct originals shared one intake run")
				}
				acquired[node] = subject
				break
			}
		}
		for node := range acquired {
			delete(acquired, node)
			break
		}
		if _, err := frozen.BindExecutionAcquired(resolution.Graph(), acquired); err == nil {
			t.Fatal("partial instance set accepted")
		}
	}
}

func TestPNPMRunResolutionRejectsWrongRequestCancellationAndMissingSource(t *testing.T) {
	source, cache, reference, calls := pnpmRunResolutionFixture(t, false)
	for _, locator := range []string{"other", "is-number@7.0.0", "node", "is-number@latest"} {
		wrong, _ := domain.NewArtifactReference(reference.Source(), locator)
		resolution, err := source.BindPNPMRunCache(context.Background(), cache, wrong, "fixture-runtime")
		if err == nil || resolution.Valid() {
			t.Fatal("foreign or provisioning request survived", locator, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resolution, err := source.BindPNPMRunCache(ctx, cache, reference, "fixture-runtime")
	if !errors.Is(err, context.Canceled) || resolution.Valid() {
		t.Fatal("canceled resolution survived", err)
	}
	if _, err := source.BindPNPMRunCache(context.Background(), PNPMRunCache{}, reference, "fixture-runtime"); err == nil {
		t.Fatal("zero cache accepted")
	}
	if _, err := (*Resolver)(nil).BindPNPMRunCache(context.Background(), cache, reference, "fixture-runtime"); err == nil {
		t.Fatal("missing public source accepted")
	}
	if *calls != 0 {
		t.Fatal("preflight-invalid request performed source I/O")
	}
}
