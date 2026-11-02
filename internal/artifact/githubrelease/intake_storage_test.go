package githubrelease

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/storage"
)

type assetStorageTransport func(*http.Request) (*http.Response, error)

func (f assetStorageTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestReleaseIntakeStorageAdmissionAndLastUse(t *testing.T) {
	for _, scenario := range []string{"complete", "quota", "download failure", "digest mismatch"} {
		t.Run(scenario, func(t *testing.T) {
			parent, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			root := filepath.Join(parent, "haa")
			spec := domain.DefaultStoragePolicy().Spec()
			spec.MinimumFreeBytes = 1
			if scenario == "quota" {
				spec.TotalBytes, spec.CacheBytes, spec.EvidenceBytes = 64<<10, 16<<10, 32<<10
			}
			policy, err := domain.NewStoragePolicy(spec)
			if err != nil {
				t.Fatal(err)
			}
			owner, err := storage.NewLocal(context.Background(), root, policy)
			if err != nil {
				t.Fatal(err)
			}
			defer owner.Close()
			body := []byte("small inert release fixture")
			digest, _ := domain.NewSHA256Digest(fmt.Sprintf("%x", sha256.Sum256(body)))
			selector, _ := ParseSelector("owner/repo@v1#asset")
			asset := ResolvedAsset{releaseID: 1, assetID: 2, selector: selector, digest: digest, size: uint64(len(body))}
			if scenario == "quota" {
				asset.size = 128 << 10
			}
			id, _ := domain.NewResolvedArtifactIdentity(mustSource(t), "owner-repo", "v1", "asset")
			resolved, err := domain.NewResolvedArtifact(id, assetLocator(asset), "sha256:"+digest.String())
			if err != nil {
				t.Fatal(err)
			}
			run, _ := domain.NewRunID()
			op, _ := domain.NewOperationID()
			ctx := ports.WithStorageOperation(ports.WithStorage(context.Background(), owner), op)
			calls := 0
			api, _ := url.Parse("https://api.github.com")
			client, err := newClient(api, &http.Client{Transport: assetStorageTransport(func(*http.Request) (*http.Response, error) {
				calls++
				journal, err := os.ReadFile(filepath.Join(root, ".storage", "resources.jsonl"))
				if err != nil || !bytes.Contains(journal, []byte(`"phase":"IN_USE"`)) {
					t.Errorf("download preceded reservation: %s %v", journal, err)
				}
				if scenario == "download failure" {
					return nil, errors.New("controlled source failure")
				}
				payload := bytes.Clone(body)
				if scenario == "digest mismatch" {
					payload[0] ^= 1
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), ContentLength: int64(len(payload)), Body: io.NopCloser(bytes.NewReader(payload))}, nil
			})}, filepath.Join(root, "intake"), true)
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := client.Acquire(ctx, run, resolved)
			if scenario == "quota" {
				if err == nil || calls != 0 {
					t.Fatalf("quota admitted download: %d %v", calls, err)
				}
				return
			}
			if scenario == "download failure" {
				if err == nil || calls != 1 {
					t.Fatalf("source failure lost: %d %v", calls, err)
				}
				journal, readErr := os.ReadFile(filepath.Join(root, ".storage", "resources.jsonl"))
				if readErr != nil || !bytes.Contains(journal, []byte(`"action":"abort"`)) {
					t.Fatalf("empty failed allocation not released: %s %v", journal, readErr)
				}
				return
			}
			if scenario == "digest mismatch" {
				if err == nil {
					t.Fatal("accepted changed payload")
				}
				journal, readErr := os.ReadFile(filepath.Join(root, ".storage", "resources.jsonl"))
				if readErr != nil || !bytes.Contains(journal, []byte(`"action":"abort"`)) {
					t.Fatalf("owned failed asset was not finalized: %s %v", journal, readErr)
				}
				return
			}
			if err != nil || artifact.SizeBytes() != uint64(len(body)) {
				t.Fatal(err)
			}
			if err := owner.Finish(ctx, true); err == nil {
				t.Fatal("deleted before inspection completion")
			}
			if err := owner.NoteInspection(ctx, run, true); err != nil {
				t.Fatal(err)
			}
			if err := owner.Finish(ctx, true); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(root, "intake", run.String())); !os.IsNotExist(err) {
				t.Fatalf("intake remains: %v", err)
			}
		})
	}
}
