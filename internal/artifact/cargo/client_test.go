package cargo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type publicTransport func(*http.Request) (*http.Response, error)

func (f publicTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCargoPublicAcquisitionFrozenSubjectAndIntake(t *testing.T) {
	root := filepath.Join(t.TempDir(), "intake")
	client, err := NewPublicClient(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile("testdata/itoa-index.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	crate, err := os.ReadFile("testdata/itoa-1.0.17.crate")
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	client.http.Transport = publicTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Scheme != "https" || r.Header.Get("Authorization") != "" || r.URL.User != nil || r.Header.Get("Cookie") != "" {
			t.Fatal("noncanonical credential-bearing request")
		}
		seen = append(seen, r.URL.String())
		var body []byte
		switch r.URL.String() {
		case "https://index.crates.io/config.json":
			body = []byte(`{"dl":"https://static.crates.io/crates","api":"https://crates.io"}`)
		case "https://index.crates.io/it/oa/itoa":
			body = index
		case "https://static.crates.io/crates/itoa/itoa-1.0.17.crate":
			body = crate
		default:
			t.Fatalf("unexpected source %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body))}, nil
	})
	ref, _ := ParseReference("itoa@1.0.17")
	resolved, err := client.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	run, _ := domain.NewRunID()
	acquired, err := client.Acquire(context.Background(), run, resolved)
	if err != nil || len(seen) != 3 {
		t.Fatalf("acquire %v requests=%d", err, len(seen))
	}
	body, err := ReadIntake(root, acquired)
	if err != nil || !bytes.Equal(body, crate) {
		t.Fatalf("exact intake: %v", err)
	}
	if _, err := client.Acquire(context.Background(), run, resolved); err == nil {
		t.Fatal("duplicate Run overwrote intake")
	}
	name := filepath.Join(root, run.String(), "package.crate")
	body[0] ^= 1
	if err := os.WriteFile(name, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIntake(root, acquired); err == nil {
		t.Fatal("tampered intake accepted")
	}
	if err := os.WriteFile(name, crate, 0o600); err != nil {
		t.Fatal(err)
	}
	backup := name + ".original"
	if err := os.Rename(name, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(backup, name); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIntake(root, acquired); err == nil {
		t.Fatal("linked intake accepted")
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, name); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Dir(name)
	if err := os.Rename(directory, directory+".original"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(directory+".original", directory); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIntake(root, acquired); err == nil {
		t.Fatal("linked Run directory accepted")
	}
}

func TestCargoPublicAcquisitionRejectsSubstitutionRedirectAndIncomplete(t *testing.T) {
	client, _ := NewPublicClient(filepath.Join(t.TempDir(), "intake"))
	identity, _ := domain.NewResolvedArtifactIdentity(Source(), "fixture", "1.2.3", "crate")
	endpoint, _ := DownloadURL("fixture", "1.2.3")
	resolved, _ := domain.NewResolvedArtifact(identity, endpoint, "sha256="+strings.Repeat("a", 64))
	run, _ := domain.NewRunID()
	substituted, _ := domain.NewResolvedArtifact(identity, "https://evil.invalid/archive", resolved.DeclaredIntegrity())
	client.http.Transport = publicTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("substituted source reached HTTP")
		return nil, nil
	})
	if _, err := client.Acquire(context.Background(), run, substituted); err == nil {
		t.Fatal("substituted source accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.LookupChecksum(ctx, "fixture", "1.2.3"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled lookup %v", err)
	}
	ref, _ := ParseReference("fixture@1.2.3")
	if _, err := client.Resolve(ctx, ref); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled resolution %v", err)
	}
	if _, err := client.Acquire(ctx, run, resolved); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled acquisition %v", err)
	}
	for _, fault := range []string{"redirect", "length cap", "truncated", "empty"} {
		t.Run(fault, func(t *testing.T) {
			calls := 0
			client.http.Transport = publicTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				response := &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("fixture")), ContentLength: 7, Request: r}
				switch fault {
				case "redirect":
					response.StatusCode = 302
					response.Header.Set("Location", "https://evil.invalid/crate")
				case "length cap":
					response.ContentLength = MaxCrateBytes + 1
				case "truncated":
					response.ContentLength = 8
				case "empty":
					response.Body = io.NopCloser(strings.NewReader(""))
					response.ContentLength = 0
				}
				return response, nil
			})
			if _, err := client.Acquire(context.Background(), run, resolved); err == nil || calls != 1 {
				t.Fatalf("failed response accepted/retried: %v calls=%d", err, calls)
			}
			if _, err := os.Stat(filepath.Join(client.intakeRoot, run.String())); !os.IsNotExist(err) {
				t.Fatal("failed download produced intake")
			}
		})
	}
	transport := PublicHTTPClient().Transport.(*http.Transport)
	if transport.Proxy != nil || PublicHTTPClient().Jar != nil {
		t.Fatal("ambient proxy or cookie jar enabled")
	}
}
