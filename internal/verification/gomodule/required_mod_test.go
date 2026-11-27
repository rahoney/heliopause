package gomodule

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/sumdb"
	"golang.org/x/mod/sumdb/dirhash"
	"golang.org/x/mod/sumdb/note"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestRequiredGoControlHasIndependentAuthenticatedVersionProof(t *testing.T) {
	for _, fault := range []string{"", "same-content", "wrong-old-hash", "current-proof-only", "missing-proof", "body-tamper"} {
		t.Run(fault, func(t *testing.T) {
			root, original, z, m := testModule(t, "package module\n")
			bundle, err := artifactgo.ReadIntake(root, original)
			if err != nil {
				t.Fatal(err)
			}
			old := []byte("module example.com/module\ngo 1.25\n")
			if fault == "same-content" || fault == "current-proof-only" {
				old = bundle.Mod()
			}
			oldSum, err := dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(old)), nil })
			if err != nil {
				t.Fatal(err)
			}
			if fault == "body-tamper" {
				old[0] = '!'
			}
			control, err := json.Marshal(struct {
				Mod      []byte                          `json:"mod"`
				Required []artifactgo.RequiredModControl `json:"required_mods"`
			}{bundle.Mod(), []artifactgo.RequiredModControl{{Version: "v0.9.0", Body: old}}})
			if err != nil {
				t.Fatal(err)
			}
			header := "HAA-GO-MODULE-2\n"
			body := append([]byte(header), make([]byte, 8)...)
			binary.BigEndian.PutUint64(body[len(header):], uint64(len(control)))
			body = append(append(body, control...), bundle.Zip()...)
			parts := strings.Split(original.ContentHandle(), ":")
			if err := os.WriteFile(filepath.Join(root, parts[1], "module.bundle"), body, 0o600); err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(body)
			digest, _ := domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
			a, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(original.Identity(), digest, original.ContentHandle(), uint64(len(body)), "h1="+z+";go.mod="+m+";required-mods=v0.9.0:"+oldSum)
			if err != nil {
				t.Fatal(err)
			}
			private, key, err := note.GenerateKey(rand.Reader, "sum.golang.org")
			if err != nil {
				t.Fatal(err)
			}
			oldLookups := 0
			server := sumdb.NewServer(sumdb.NewTestServer(private, func(path, version string) ([]byte, error) {
				if path != "example.com/module" {
					return nil, errors.New("unexpected module")
				}
				if version == "v1.0.0" {
					return []byte(path + " " + version + " " + z + "\n" + path + " " + version + "/go.mod " + m + "\n"), nil
				}
				if version != "v0.9.0" {
					return nil, errors.New("unexpected version")
				}
				oldLookups++
				if fault == "missing-proof" {
					return nil, errors.New("no old record")
				}
				if fault == "current-proof-only" {
					version = "v1.0.0"
				}
				sum := oldSum
				if fault == "wrong-old-hash" || fault == "current-proof-only" {
					sum = "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
				}
				return []byte(path + " " + version + " " + z + "\n" + path + " " + version + "/go.mod " + sum + "\n"), nil
			}))
			client := artifactgo.PublicHTTPClient()
			client.Transport = databaseTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Scheme != "https" || r.URL.Host != "sum.golang.org" {
					t.Fatal("noncanonical proof source")
				}
				recorder := httptest.NewRecorder()
				server.ServeHTTP(recorder, r)
				return recorder.Result(), nil
			})
			v := &IntegrityVerifier{intakeRoot: root, http: client, key: key}
			result, err := v.Verify(context.Background(), a)
			valid := fault == "" || fault == "same-content"
			if valid {
				if err != nil || result.Outcome() != domain.VerificationVerified || oldLookups != 1 {
					t.Fatalf("independent old proof=%v result=%v lookups=%d", err, result.Outcome(), oldLookups)
				}
			} else if err == nil && result.Outcome() == domain.VerificationVerified {
				t.Fatal("missing/mismatched old-version proof approved")
			}
			if fault == "current-proof-only" && oldLookups != 1 {
				t.Fatal("equal current bytes replaced independent old-version lookup")
			}
		})
	}
}
