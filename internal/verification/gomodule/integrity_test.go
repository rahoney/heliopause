package gomodule

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/sumdb/dirhash"
	"golang.org/x/mod/sumdb/note"
	"golang.org/x/mod/sumdb/tlog"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type databaseTransport func(*http.Request) (*http.Response, error)

func (f databaseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testModule(t *testing.T, contents string) (root string, artifact domain.AcquiredArtifact, zipSum, modSum string) {
	t.Helper()
	mod := []byte("module example.com/module\ngo 1.26.0\n")
	var z bytes.Buffer
	writer := zip.NewWriter(&z)
	for _, item := range []struct{ name, body string }{{"go.mod", string(mod)}, {"module.go", contents}} {
		f, err := writer.Create("example.com/module@v1.0.0/" + item.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(f, item.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "module.zip")
	if err := os.WriteFile(zipPath, z.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	zipSum, err := dirhash.HashZip(zipPath, dirhash.Hash1)
	if err != nil {
		t.Fatal(err)
	}
	modSum, err = dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(mod)), nil })
	if err != nil {
		t.Fatal(err)
	}
	body := append([]byte("HAA-GO-MODULE-1\n"), make([]byte, 8)...)
	binary.BigEndian.PutUint64(body[len(body)-8:], uint64(len(mod)))
	body = append(body, mod...)
	body = append(body, z.Bytes()...)
	root = t.TempDir()
	run, _ := domain.NewRunID()
	if err := os.Mkdir(filepath.Join(root, run.String()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, run.String(), "module.bundle"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	d, _ := domain.NewSHA256Digest(hex.EncodeToString(digest[:]))
	identity, _ := domain.NewResolvedArtifactIdentity(artifactgo.Source(), "example.com/module", "v1.0.0", "module")
	artifact, err = domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, d, "intake:"+run.String()+":module", uint64(len(body)), "h1="+zipSum+";go.mod="+modSum)
	if err != nil {
		t.Fatal(err)
	}
	return
}

func signedDatabase(t *testing.T, zipSum, modSum, fault string) (string, *http.Client, *int) {
	t.Helper()
	private, key, err := note.GenerateKey(rand.Reader, "sum.golang.org")
	if err != nil {
		t.Fatal(err)
	}
	signer, err := note.NewSigner(private)
	if err != nil {
		t.Fatal(err)
	}
	record := []byte("example.com/module v1.0.0 " + zipSum + "\nexample.com/module v1.0.0/go.mod " + modSum + "\n")
	treeTextExtra := ""
	if fault == "unlogged-hash" {
		treeTextExtra = string(record)
		record = bytes.ReplaceAll(record, []byte("example.com/module"), []byte("example.com/other"))
	}
	hash := tlog.RecordHash(record)
	signed, err := note.Sign(&note.Note{Text: string(tlog.FormatTree(tlog.Tree{N: 1, Hash: hash})) + treeTextExtra}, signer)
	if err != nil {
		t.Fatal(err)
	}
	formatted, err := tlog.FormatRecord(0, record)
	if err != nil {
		t.Fatal(err)
	}
	lookup := append(formatted, signed...)
	if fault == "signature" {
		at := bytes.Index(lookup, []byte("— sum.golang.org ")) + len("— sum.golang.org ") + 12
		if lookup[at] == 'A' {
			lookup[at] = 'B'
		} else {
			lookup[at] = 'A'
		}
	}
	if fault == "lookup" {
		lookup[bytes.Index(lookup, []byte("h1:"))+3] ^= 1
	}
	calls := new(int)
	client := artifactgo.PublicHTTPClient()
	client.Transport = databaseTransport(func(r *http.Request) (*http.Response, error) {
		*calls++
		if r.URL.Host != "sum.golang.org" || r.URL.Scheme != "https" {
			t.Fatal("noncanonical SumDB request")
		}
		status := 200
		var body []byte
		if strings.HasPrefix(r.URL.Path, "/lookup/") {
			body = bytes.Clone(lookup)
		} else {
			tile, err := tlog.ParseTilePath(strings.TrimPrefix(r.URL.Path, "/"))
			if err != nil || tile.L != 0 || tile.N != 0 || tile.W != 1 {
				status = 404
				body = []byte("missing")
			} else {
				body = bytes.Clone(hash[:])
				if fault == "tile" {
					body[0] ^= 1
				}
			}
		}
		if fault == "redirect" {
			status = 302
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Location": []string{"https://evil.invalid/proof"}}, Body: io.NopCloser(bytes.NewReader(body)), Request: r}, nil
	})
	return key, client, calls
}

func TestSumDBRequiresAuthenticatedArchiveAndModProof(t *testing.T) {
	for _, fault := range []string{"", "signature", "lookup", "tile", "redirect", "checksum", "mod-checksum", "unlogged-hash"} {
		t.Run(fault, func(t *testing.T) {
			root, artifact, z, m := testModule(t, "package module\nconst Value = 1\n")
			if fault == "checksum" {
				z = "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
			}
			if fault == "mod-checksum" {
				m = "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
			}
			key, client, calls := signedDatabase(t, z, m, fault)
			v := &IntegrityVerifier{intakeRoot: root, http: client, key: key}
			result, err := v.Verify(context.Background(), artifact)
			if fault == "" {
				if err != nil || result.Outcome() != domain.VerificationVerified || *calls < 2 {
					t.Fatalf("authentication=%v result=%v calls=%d", err, result.Outcome(), *calls)
				}
			} else if err == nil && result.Outcome() == domain.VerificationVerified {
				t.Fatal("unauthenticated/tampered subject verified")
			}
		})
	}
}

func TestSumDBCannotTrustAProjectDeclaredChecksum(t *testing.T) {
	root, artifact, z, m := testModule(t, "package module\nconst Forged = true\n")
	key, client, _ := signedDatabase(t, "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", m, "")
	v := &IntegrityVerifier{intakeRoot: root, http: client, key: key}
	result, err := v.Verify(context.Background(), artifact)
	if err != nil || result.Outcome() != domain.VerificationMismatch {
		t.Fatalf("forged declaration %s accepted: %v %v", z, err, result.Outcome())
	}
}

func TestSumDBUnavailableHasNoSuccessfulReport(t *testing.T) {
	root, artifact, _, _ := testModule(t, "package module\n")
	v, err := NewIntegrityVerifier(root)
	if err != nil {
		t.Fatal(err)
	}
	v.http.Transport = databaseTransport(func(*http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF })
	if report, err := v.Verify(context.Background(), artifact); err == nil || report.Outcome() == domain.VerificationVerified {
		t.Fatal("unavailable SumDB was accepted")
	}
}
