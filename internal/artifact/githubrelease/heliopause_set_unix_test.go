//go:build linux || darwin

package githubrelease

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/testutil/releasefixture"
)

type releaseSetFixture struct {
	mu       sync.Mutex
	files    map[string][]byte
	metadata map[string]any
	release  releaseResponse
	byID     map[uint64]string
	calls    map[string]int
	proof    []byte
	client   *Client
	set      *HeliopauseReleaseSet
	install  domain.InstallContext
}

func newReleaseSetFixture(t *testing.T, edit func(map[string]any)) *releaseSetFixture {
	t.Helper()
	base := filepath.Join("..", "..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0")
	read := func(name string) []byte {
		body, err := os.ReadFile(filepath.Join(base, name))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	f := &releaseSetFixture{files: map[string][]byte{}, byID: map[uint64]string{}, calls: map[string]int{}, proof: read("actual-signed-bundle.json")}
	if err := json.Unmarshal(read(heliopauseManifestName), &f.metadata); err != nil {
		t.Fatal(err)
	}
	assets := f.metadata["assets"].([]any)
	assets = append(assets, map[string]any{"name": "haa-release-installer-linux-amd64", "platform": "linux/amd64", "role": "release-installer"})
	assets = append(assets, map[string]any{"name": "pnpm-linux-amd64", "platform": "linux/amd64", "role": "pnpm"}, map[string]any{"name": "pnpm-NOTICE.txt", "platform": "linux/amd64", "role": "release-metadata"})
	noticeBody, noticeErr := os.ReadFile(filepath.Join("..", "..", "..", "tools", "pnpm", "NOTICE"))
	if noticeErr != nil {
		t.Fatal(noticeErr)
	}
	for _, raw := range assets {
		item := raw.(map[string]any)
		name := item["name"].(string)
		if item["platform"] == "linux/amd64" && !strings.HasSuffix(name, ".deb") && !strings.HasSuffix(name, ".rpm") {
			body := make([]byte, 64)
			copy(body, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
			binary.LittleEndian.PutUint16(body[18:20], 62)
			binary.LittleEndian.PutUint64(body[32:40], 64)
			binary.LittleEndian.PutUint16(body[52:54], 64)
			binary.LittleEndian.PutUint16(body[54:56], 56)
			body = append(body, []byte(name)...)
			if name == "pnpm-NOTICE.txt" {
				body = noticeBody
			}
			f.files[name] = body
			item["size"] = len(body)
			item["sha256"] = fmt.Sprintf("%x", sha256.Sum256(body))
		}
	}
	f.metadata["assets"] = assets
	for _, name := range []string{"helox-release-sbom.cdx.json", "helox-runtime-images.json"} {
		f.files[name] = read(name)
	}
	unsigned, err := json.Marshal(f.metadata)
	if err != nil {
		t.Fatal(err)
	}
	materials, err := releasefixture.BindCurrentRuntime(unsigned, f.files["helox-release-sbom.cdx.json"], f.files["helox-runtime-images.json"])
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(materials.Manifest, &f.metadata); err != nil {
		t.Fatal(err)
	}
	f.files["helox-release-sbom.cdx.json"], f.files["helox-runtime-images.json"] = materials.SBOM, materials.Images
	if edit != nil {
		edit(f.metadata)
	}
	manifest, err := json.Marshal(f.metadata)
	if err != nil {
		t.Fatal(err)
	}
	f.files[heliopauseManifestName] = manifest
	f.release = releaseResponse{ID: 41, TagName: "v0.1.0", Immutable: true, PublishedAt: "2026-10-17T01:02:03Z"}
	declarations := map[string]releaseAssetResponse{}
	for _, raw := range f.metadata["assets"].([]any) {
		item := raw.(map[string]any)
		var size int64
		switch number := item["size"].(type) {
		case int:
			size = int64(number)
		case float64:
			size = int64(number)
		case int64:
			size = number
		}
		declarations[item["name"].(string)] = releaseAssetResponse{Name: item["name"].(string), State: "uploaded", Size: size, Digest: "sha256:" + item["sha256"].(string)}
	}
	for name, body := range f.files {
		if _, found := declarations[name]; !found {
			declarations[name] = releaseAssetResponse{Name: name, State: "uploaded", Size: int64(len(body)), Digest: fmt.Sprintf("sha256:%x", sha256.Sum256(body))}
		}
	}
	names := make([]string, 0, len(declarations))
	for name := range declarations {
		names = append(names, name)
	}
	sort.Strings(names)
	for index, name := range names {
		item := declarations[name]
		item.ID = uint64(index + 7)
		f.release.Assets = append(f.release.Assets, item)
		f.byID[item.ID] = name
	}
	root := privateFrozenRoot(t)
	f.client, err = NewPublicClient(root)
	if err != nil {
		t.Fatal(err)
	}
	f.client.httpClient.Transport = attestationRoundTripper(f.request)
	target, err := domain.NewInstallTarget(filepath.Join(privateFrozenRoot(t), "managed"))
	if err != nil {
		t.Fatal(err)
	}
	f.install, err = domain.NewGitHubAssetInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	lock := f.metadata["runtime_lock"].(map[string]any)["sha256"].(string)
	f.set, err = NewHeliopauseReleaseSet(f.client, "v0.1.0", lock, f.install)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.set.Close() })
	return f
}

func (f *releaseSetFixture) request(request *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if request.URL.Host != "api.github.com" || request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" {
		return nil, errors.New("unexpected origin or credentials")
	}
	f.calls[request.URL.Path]++
	if request.URL.Path == "/repos/rahoney/heliopause/releases/tags/v0.1.0" {
		body, err := json.Marshal(f.release)
		if err != nil {
			return nil, err
		}
		return attestationHTTPResponse(body, "application/json", 200), nil
	}
	if suffix, found := strings.CutPrefix(request.URL.Path, "/repos/rahoney/heliopause/releases/assets/"); found {
		id, err := strconv.ParseUint(suffix, 10, 64)
		if err != nil {
			return nil, err
		}
		body, found := f.files[f.byID[id]]
		if !found {
			return nil, errors.New("fixture payload is not selected")
		}
		return attestationHTTPResponse(body, "application/octet-stream", 200), nil
	}
	if strings.HasPrefix(request.URL.Path, "/repos/rahoney/heliopause/attestations/sha256:") {
		body, err := json.Marshal(map[string]any{"attestations": []any{map[string]any{"repository_id": 1331904030, "bundle": json.RawMessage(f.proof)}}})
		if err != nil {
			return nil, err
		}
		return attestationHTTPResponse(body, "application/json", 200), nil
	}
	return nil, errors.New("unexpected release fixture endpoint")
}

func TestHeliopauseSetConcurrentCopiesCloseAndUncertainOwnership(t *testing.T) {
	f := newReleaseSetFixture(t, nil)
	resolution, err := f.set.ResolveDependencies(context.Background(), f.set.Reference(), f.install)
	if err != nil {
		t.Fatal(err)
	}
	selected := resolution.Graph().Nodes()[0].Artifact()
	var workers sync.WaitGroup
	firstCopy := make(chan struct{}, 1)
	for index := 0; index < 8; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for attempt := 0; attempt < 8; attempt++ {
				run, err := domain.NewRunID()
				if err != nil {
					t.Error(err)
					return
				}
				artifact, err := f.set.Acquire(context.Background(), run, selected)
				if err != nil {
					return
				}
				select {
				case firstCopy <- struct{}{}:
				default:
				}
				_, _, _ = f.set.ReleaseProof(artifact)
			}
		}()
	}
	finished := make(chan struct{})
	go func() { workers.Wait(); close(finished) }()
	select {
	case <-firstCopy:
	case <-finished:
		t.Fatal("no source copy reached completion")
	}
	if err := f.set.Close(); err != nil {
		t.Fatal(err)
	}
	<-finished

	other := newReleaseSetFixture(t, nil)
	if _, err := other.set.ResolveDependencies(context.Background(), other.set.Reference(), other.install); err != nil {
		t.Fatal(err)
	}
	var owned string
	for name := range other.set.owned {
		owned = name
		break
	}
	path := filepath.Join(other.client.intakeRoot, owned)
	if err := os.Rename(path, path+"-original"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := other.set.Close(); err == nil {
		t.Fatal("unknown cleanup ownership was treated as absence")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("replacement directory was deleted")
	}
}

func TestHeliopauseSetFreezesNineMembersAndCleansOnlyOwnedAcquisitions(t *testing.T) {
	f := newReleaseSetFixture(t, nil)
	foreign := filepath.Join(f.client.intakeRoot, "unrelated")
	if err := os.Mkdir(foreign, 0700); err != nil {
		t.Fatal(err)
	}
	resolution, err := f.set.ResolveDependencies(context.Background(), f.set.Reference(), f.install)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Graph().Nodes()) != 9 || len(resolution.Graph().Edges()) != 8 {
		t.Fatal("incomplete native/metadata set")
	}
	manifest, err := f.set.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.release.Assets[0].Digest = "sha256:" + strings.Repeat("f", 64)
	f.mu.Unlock()
	if _, err := f.set.ResolveDependencies(context.Background(), f.set.Reference(), f.install); err != nil {
		t.Fatal(err)
	}
	var copied string
	for _, node := range resolution.Graph().Nodes() {
		run, err := domain.NewRunID()
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := f.set.Acquire(context.Background(), run, node.Artifact())
		if err != nil {
			t.Fatal(err)
		}
		parsed, proof, err := f.set.ReleaseProof(artifact)
		if err != nil || parsed.SHA256() != manifest.SHA256() || len(proof) != 1 {
			t.Fatalf("proof error=%v", err)
		}
		proof[0][0] = '!'
		_, proof, err = f.set.ReleaseProof(artifact)
		if err != nil || proof[0][0] != '{' {
			t.Fatal("proof bytes aliased")
		}
		copied = filepath.Join(f.client.intakeRoot, run.String(), "asset")
	}
	f.mu.Lock()
	gets := f.calls["/repos/rahoney/heliopause/releases/tags/v0.1.0"]
	f.mu.Unlock()
	if gets != 1 {
		t.Fatal("live discovery repeated")
	}
	owned := make([]string, 0, len(f.set.owned))
	for name := range f.set.owned {
		owned = append(owned, name)
	}
	if err := f.set.Close(); err != nil {
		t.Fatal(err)
	}
	for _, name := range owned {
		if _, err := os.Lstat(filepath.Join(f.client.intakeRoot, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("original acquisition retained")
		}
	}
	if _, err := os.Stat(copied); err != nil {
		t.Fatal("source Close removed another inspection Run")
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatal("source Close removed unrelated content")
	}
	if err := f.set.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := f.set.Manifest(); err == nil {
		t.Fatal("closed metadata returned")
	}
}

func TestHeliopauseSetRejectsSelectionAndBindingGapsWithoutPayloadPromotion(t *testing.T) {
	for _, name := range []string{"missing-installer", "runtime-lock", "manifest-api-digest", "asset-api-size", "mutable-release", "set-byte-limit", "cancelled"} {
		t.Run(name, func(t *testing.T) {
			var edit func(map[string]any)
			if name == "missing-installer" {
				edit = func(m map[string]any) {
					assets := m["assets"].([]any)
					kept := make([]any, 0, len(assets))
					for _, raw := range assets {
						if raw.(map[string]any)["name"] != "haa-release-installer-linux-amd64" {
							kept = append(kept, raw)
						}
					}
					m["assets"] = kept
				}
			}
			if name == "set-byte-limit" {
				edit = func(m map[string]any) {
					for _, raw := range m["assets"].([]any) {
						item := raw.(map[string]any)
						if item["role"] == "helox" || item["role"] == "release-installer" || item["role"] == "network-policy-helper" || item["role"] == "gvisor-observer" {
							item["size"] = int64(1 << 29)
						}
					}
				}
			}
			f := newReleaseSetFixture(t, edit)
			if name == "runtime-lock" {
				f.set.lockSHA256 = strings.Repeat("a", 64)
			}
			if name == "mutable-release" {
				f.release.Immutable = false
			}
			if name == "manifest-api-digest" || name == "asset-api-size" {
				for index := range f.release.Assets {
					item := &f.release.Assets[index]
					if name == "manifest-api-digest" && item.Name == heliopauseManifestName {
						item.Digest = "sha256:" + strings.Repeat("b", 64)
					}
					if name == "asset-api-size" && item.Name == "helox-linux-amd64" {
						item.Size++
					}
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if name == "cancelled" {
				cancel()
			}
			if _, err := f.set.ResolveDependencies(ctx, f.set.Reference(), f.install); err == nil {
				t.Fatal("invalid selection accepted")
			} else if name == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation cause lost")
			}
			if err := f.set.Close(); err != nil {
				t.Fatal(err)
			}
			remaining, err := os.ReadDir(f.client.intakeRoot)
			if err != nil || len(remaining) != 0 {
				t.Fatalf("failed source retained completed inputs: %v %v", remaining, err)
			}
		})
	}
	f := newReleaseSetFixture(t, nil)
	for _, version := range []string{"latest", "v1", "v01.0.0", "https://other.invalid"} {
		if _, err := NewHeliopauseReleaseSet(f.client, version, f.set.lockSHA256, f.install); err == nil {
			t.Fatal("non-exact version accepted")
		}
	}
	other, _ := ParseReference("other/heliopause@v0.1.0#" + heliopauseManifestName)
	if _, err := f.set.ResolveDependencies(context.Background(), other, f.install); err == nil {
		t.Fatal("other repository accepted")
	}
}

// ReleaseSetPipelineFixture exposes only public operation dependencies to the
// external integration test, avoiding a test-only import cycle with promotion.
type ReleaseSetPipelineFixture struct {
	Source     *HeliopauseReleaseSet
	IntakeRoot string
	Install    domain.InstallContext
}

func NewReleaseSetPipelineFixture(t *testing.T) ReleaseSetPipelineFixture {
	f := newReleaseSetFixture(t, nil)
	return ReleaseSetPipelineFixture{f.set, f.client.intakeRoot, f.install}
}
