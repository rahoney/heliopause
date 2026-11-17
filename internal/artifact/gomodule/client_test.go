package gomodule

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/storage"
)

type proxyTransport func(*http.Request) (*http.Response, error)

func (f proxyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPublicAcquisitionUsesExactProxyAndBoundIntake(t *testing.T) {
	root := filepath.Join(t.TempDir(), "intake")
	client, err := NewPublicClient(root)
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := ParseReference("github.com/Azure/sdk@v1.2.3")
	resolved, err := client.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	client.http.Transport = proxyTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Scheme != "https" || r.URL.Host != "proxy.golang.org" || r.Header.Get("Authorization") != "" {
			t.Fatal("noncanonical public request")
		}
		seen = append(seen, r.URL.Path)
		body := []byte("module github.com/Azure/sdk\n")
		if strings.HasSuffix(r.URL.Path, ".zip") {
			body = []byte("archive-fixture")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body))}, nil
	})
	run, _ := domain.NewRunID()
	acquired, err := client.Acquire(context.Background(), run, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 || seen[0] != "/github.com/!azure/sdk/@v/v1.2.3.mod" || seen[1] != "/github.com/!azure/sdk/@v/v1.2.3.zip" {
		t.Fatalf("requests=%v", seen)
	}
	bundle, err := ReadIntake(root, acquired)
	if err != nil || string(bundle.Zip()) != "archive-fixture" {
		t.Fatalf("read=%v", err)
	}
	copyMod := bundle.Mod()
	copyMod[0] = '!'
	if bundle.Mod()[0] == '!' {
		t.Fatal("mutable intake exposure")
	}
	if _, err := client.Acquire(context.Background(), run, resolved); err == nil {
		t.Fatal("duplicate Run overwrote intake")
	}
	name := filepath.Join(root, run.String(), "module.bundle")
	body, _ := os.ReadFile(name)
	body[len(body)-1] ^= 1
	if err := os.WriteFile(name, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIntake(root, acquired); err == nil {
		t.Fatal("tampered intake accepted")
	}
}

func TestPublicAcquisitionRejectsSourceSubstitutionAndRedirect(t *testing.T) {
	client, _ := NewPublicClient(filepath.Join(t.TempDir(), "intake"))
	ref, _ := ParseReference("example.com/module@v1.0.0")
	resolved, _ := client.Resolve(context.Background(), ref)
	run, _ := domain.NewRunID()
	substituted, _ := domain.NewResolvedArtifact(resolved.Identity(), "https://evil.invalid/module.zip", "")
	client.http.Transport = proxyTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("substituted source reached HTTP")
		return nil, nil
	})
	if _, err := client.Acquire(context.Background(), run, substituted); err == nil {
		t.Fatal("substituted source accepted")
	}
	calls := 0
	client.http.Transport = proxyTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://evil.invalid/zip"}}, Body: io.NopCloser(strings.NewReader("redirect")), Request: r}, nil
	})
	if _, err := client.Acquire(context.Background(), run, resolved); err == nil || calls != 1 {
		t.Fatalf("redirect accepted/followed: %v calls=%d", err, calls)
	}
	if _, err := os.Stat(filepath.Join(client.intakeRoot, run.String())); !os.IsNotExist(err) {
		t.Fatal("failed download produced intake")
	}
}

func TestGoIntakeRejectsLinksAndMalformedEnvelope(t *testing.T) {
	root := filepath.Join(t.TempDir(), "intake")
	client, _ := NewPublicClient(root)
	ref, _ := ParseReference("example.com/module@v1.0.0")
	resolved, _ := client.Resolve(context.Background(), ref)
	client.http.Transport = proxyTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("fixture")), Header: make(http.Header)}, nil
	})
	run, _ := domain.NewRunID()
	acquired, err := client.Acquire(context.Background(), run, resolved)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, run.String(), "module.bundle")
	backup := file + ".original"
	if err := os.Rename(file, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(backup, file); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIntake(root, acquired); err == nil {
		t.Fatal("symlink accepted")
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Dir(file), filepath.Dir(file)+".original"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(file)+".original", filepath.Dir(file)); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIntake(root, acquired); err == nil {
		t.Fatal("symlink Run accepted")
	}
}

func TestAcquisitionStorageAdmissionAndLastUse(t *testing.T) {
	for _, denied := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "quota"}[denied], func(t *testing.T) {
			parent, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			root := filepath.Join(parent, "haa")
			spec := domain.DefaultStoragePolicy().Spec()
			spec.MinimumFreeBytes = 1
			if denied {
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
			run, _ := domain.NewRunID()
			operation, _ := domain.NewOperationID()
			ctx := ports.WithStorageOperation(ports.WithStorage(context.Background(), owner), operation)

			client, err := NewPublicClient(filepath.Join(root, "intake"))
			if err != nil {
				t.Fatal(err)
			}
			ref, _ := ParseReference("example.org/sample@v1.0.0")
			resolved, err := client.Resolve(ctx, ref)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			client.http.Transport = proxyTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				body := []byte("module example.org/sample\n")
				if strings.HasSuffix(r.URL.Path, ".zip") {
					body = []byte("small archive fixture")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), ContentLength: int64(len(body)), Body: io.NopCloser(bytes.NewReader(body))}, nil
			})

			artifact, err := client.Acquire(ctx, run, resolved)
			if denied {
				if err == nil || calls != 0 {
					t.Fatalf("quota admitted download: calls=%d err=%v", calls, err)
				}
				return
			}
			if err != nil || calls == 0 {
				t.Fatalf("acquire: %v calls=%d", err, calls)
			}
			journal, err := os.ReadFile(filepath.Join(root, ".storage", "resources.jsonl"))
			if err != nil || !bytes.Contains(journal, []byte(`"phase":"RETAINED"`)) {
				t.Fatalf("unsealed intake: %s %v", journal, err)
			}
			if _, err := ReadIntake(client.intakeRoot, artifact); err != nil {
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
			if _, err := os.Stat(filepath.Join(client.intakeRoot, run.String())); !os.IsNotExist(err) {
				t.Fatalf("completed intake remains: %v", err)
			}
		})
	}
}
