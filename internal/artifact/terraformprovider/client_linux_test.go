package terraformprovider

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type responseTransport struct {
	bodies map[string][]byte
	calls  map[string]int
	fault  string
}

func (r *responseTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	u := request.URL.String()
	r.calls[u]++
	body, ok := r.bodies[u]
	if !ok {
		return nil, errors.New("unregistered fixture request")
	}
	response := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)), Request: request}
	if strings.HasSuffix(u, "/terraform.json") && r.fault == "redirect" {
		response.StatusCode = http.StatusFound
		response.Header.Set("Location", "https://mirror.example/v1/providers/")
	}
	if r.fault == "truncated" && strings.HasSuffix(u, ".zip") {
		response.ContentLength++
	}
	if r.fault == "excessive" && strings.HasSuffix(u, ".zip") {
		response.ContentLength = MaxProviderArchiveBytes + 1
	}
	return response, nil
}

func clientFixture(t *testing.T) (*Client, *responseTransport, domain.ArtifactReference) {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	reference, _ := ParseReference("hashicorp/random@3.7.2")
	packageBody, err := os.ReadFile("testdata/hashicorp-random-3.7.2.json")
	if err != nil {
		t.Fatal(err)
	}
	archive := []byte("synthetic acquisition bytes; not a verified provider archive")
	packageBody = bytes.ReplaceAll(packageBody, []byte("7b8434212eef0f8c83f5a90c6d76feaf850f6502b61b53c329e85b3b281cba34"), []byte(sha256Hex(archive)))
	transport := &responseTransport{calls: map[string]int{}, bodies: map[string][]byte{
		registryEndpoint + "/.well-known/terraform.json":                                                                         []byte(`{"providers.v1":"/v1/providers/"}`),
		registryEndpoint + "/v1/providers/hashicorp/random/versions":                                                             []byte(`{"versions":[{"version":"3.7.2","protocols":["5.0"],"platforms":[{"os":"linux","arch":"amd64"}]}]}`),
		registryEndpoint + "/v1/providers/hashicorp/random/3.7.2/download/linux/amd64":                                           packageBody,
		"https://releases.hashicorp.com/terraform-provider-random/3.7.2/terraform-provider-random_3.7.2_SHA256SUMS":              []byte("synthetic checksums"),
		"https://releases.hashicorp.com/terraform-provider-random/3.7.2/terraform-provider-random_3.7.2_SHA256SUMS.72D7468F.sig": []byte("synthetic signature"),
		"https://releases.hashicorp.com/terraform-provider-random/3.7.2/terraform-provider-random_3.7.2_linux_amd64.zip":         archive,
	}}
	endpoint, _ := url.Parse(registryEndpoint)
	resolver, err := newResolver(endpoint, &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	releases, err := githubrelease.NewPublicClient(root)
	if err != nil {
		t.Fatal(err)
	}
	return &Client{intakeRoot: root, resolver: resolver, releases: releases, frozen: map[string]RegistrySnapshot{}}, transport, reference
}

func TestAcquisitionConsumesSingleFrozenRegistrySelection(t *testing.T) {
	client, transport, reference := clientFixture(t)
	resolved, err := client.Resolve(context.Background(), reference)
	if err != nil {
		t.Fatal(err)
	}
	// A changed live package response cannot silently select another artifact
	// during Acquire. Verification still has the original metadata and bytes.
	transport.bodies[registryEndpoint+"/v1/providers/hashicorp/random/3.7.2/download/linux/amd64"] = []byte(`{"malicious":"live drift"}`)
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := client.Acquire(context.Background(), run, resolved)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := ReadIntake(client.intakeRoot, artifact)
	if err != nil {
		t.Fatal(err)
	}
	registry, archive, err := ParseIntegrity(resolved.DeclaredIntegrity())
	if err != nil {
		t.Fatal(err)
	}
	if bundle.RegistryDigest() != registry || bundle.ArchiveDigest() != archive || artifact.ContentHandle() != "intake:"+run.String()+":linux/amd64" {
		t.Fatal("frozen content/Run binding differs")
	}
	for endpoint, count := range transport.calls {
		if count != 1 {
			t.Fatalf("selected endpoint repeated %s: %d", endpoint, count)
		}
	}
	if len(transport.calls) != 6 {
		t.Fatalf("requests=%d", len(transport.calls))
	}
	if _, err := client.Acquire(context.Background(), run, resolved); err == nil {
		t.Fatal("duplicate Run overwrote prior intake")
	}
	if _, err := ReadIntake(client.intakeRoot, artifact); err != nil {
		t.Fatalf("duplicate request damaged original intake: %v", err)
	}
}

func TestRegistryRedirectAndIncompleteDownloadsFailClosed(t *testing.T) {
	for _, fault := range []string{"redirect", "truncated", "excessive", "cancelled"} {
		t.Run(fault, func(t *testing.T) {
			client, transport, reference := clientFixture(t)
			transport.fault = fault
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fault == "cancelled" {
				cancel()
			}
			resolved, err := client.Resolve(ctx, reference)
			if fault == "redirect" || fault == "cancelled" {
				if err == nil {
					t.Fatal("selection unexpectedly succeeded")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			run, err := domain.NewRunID()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Acquire(ctx, run, resolved); err == nil {
				t.Fatal("incomplete download was acquired")
			}
			if _, err := os.Lstat(filepath.Join(client.intakeRoot, run.String())); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("partial intake remains")
			}
		})
	}
}

func TestPrivateIntakeRejectsSubstitutionAndAliases(t *testing.T) {
	for _, fault := range []string{"tamper", "hardlink", "symlink", "mode", "Run alias", "wrong handle"} {
		t.Run(fault, func(t *testing.T) {
			client, _, reference := clientFixture(t)
			resolved, err := client.Resolve(context.Background(), reference)
			if err != nil {
				t.Fatal(err)
			}
			run, err := domain.NewRunID()
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := client.Acquire(context.Background(), run, resolved)
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(client.intakeRoot, run.String(), "provider.bundle")
			switch fault {
			case "tamper":
				body, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				body[len(body)-1] ^= 1
				if err := os.WriteFile(file, body, 0o600); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(file, file+".alias"); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(file, file+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("provider.bundle.original", file); err != nil {
					t.Fatal(err)
				}
			case "mode":
				if err := os.Chmod(file, 0o644); err != nil {
					t.Fatal(err)
				}
			case "Run alias":
				directory := filepath.Dir(file)
				if err := os.Rename(directory, directory+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Base(directory)+".original", directory); err != nil {
					t.Fatal(err)
				}
			case "wrong handle":
				artifact, err = domain.NewAcquiredArtifactWithDeclaredIntegrity(artifact.Identity(), artifact.Digest(), "intake:"+run.String()+":other", artifact.SizeBytes(), resolved.DeclaredIntegrity())
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := ReadIntake(client.intakeRoot, artifact); err == nil {
				t.Fatal("substituted/aliased intake accepted")
			}
		})
	}
}
