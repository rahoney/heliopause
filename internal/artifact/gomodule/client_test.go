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
