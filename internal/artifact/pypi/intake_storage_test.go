package pypi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
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

type intakeStorageTransport func(*http.Request) (*http.Response, error)

func (f intakeStorageTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestIntakeStorageReservesBeforeDownloadAndRetainsUntilLastUse(t *testing.T) {
	for _, scenario := range []string{"complete", "quota", "digest mismatch"} {
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
			intake, err := NewPublicIntake(filepath.Join(root, "intake"))
			if err != nil {
				t.Fatal(err)
			}
			run, _ := domain.NewRunID()
			op, _ := domain.NewOperationID()
			ctx := ports.WithStorageOperation(ports.WithStorage(context.Background(), owner), op)
			body := []byte("small inert distribution fixture")
			filename := "sample-1.0-py3-none-any.whl"
			digest := fmt.Sprintf("%x", sha256.Sum256(body))
			if scenario == "digest mismatch" {
				digest = strings.Repeat("a", 64)
			}
			id, _ := domain.NewResolvedArtifactIdentity(publicPyPIProfile.Source(), "sample", "1.0", "wheel")
			resolved, err := domain.NewResolvedArtifact(id, "https://files.pythonhosted.org/packages/"+filename, "sha256:"+digest)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			intake.client.Transport = intakeStorageTransport(func(*http.Request) (*http.Response, error) {
				calls++
				journal, err := os.ReadFile(filepath.Join(root, ".storage", "resources.jsonl"))
				if err != nil || !bytes.Contains(journal, []byte(`"run":"`+run.String()+`"`)) || !bytes.Contains(journal, []byte(`"phase":"IN_USE"`)) {
					t.Errorf("download preceded durable reservation: %s %v", journal, err)
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), ContentLength: int64(len(body)), Body: io.NopCloser(bytes.NewReader(body))}, nil
			})
			artifact, err := intake.Acquire(ctx, run, resolved)
			if scenario == "quota" {
				if err == nil || calls != 0 {
					t.Fatalf("quota admitted download: calls=%d err=%v", calls, err)
				}
				return
			}
			if scenario == "digest mismatch" {
				if err == nil || calls != 1 {
					t.Fatalf("mismatch accepted: calls=%d err=%v", calls, err)
				}
				journal, readErr := os.ReadFile(filepath.Join(root, ".storage", "resources.jsonl"))
				if readErr != nil || !bytes.Contains(journal, []byte(`"phase":"IN_USE"`)) {
					t.Fatalf("failed intake lost accounting: %s %v", journal, readErr)
				}
				return
			}
			if err != nil || calls != 1 || artifact.SizeBytes() != uint64(len(body)) {
				t.Fatalf("acquire: %v calls=%d", err, calls)
			}
			directory := filepath.Join(root, "intake", run.String())
			info, err := os.Stat(filepath.Join(directory, "filename"))
			if err != nil || info.Mode().Perm() != 0400 {
				t.Fatalf("filename permissions changed: %v %v", info, err)
			}
			journal, err := os.ReadFile(filepath.Join(root, ".storage", "resources.jsonl"))
			if err != nil || !bytes.Contains(journal, []byte(fmt.Sprintf(`"logical_bytes":%d`, len(body)+len(filename)))) || !bytes.Contains(journal, []byte(`"phase":"RETAINED"`)) {
				t.Fatalf("actual bytes not sealed: %s %v", journal, err)
			}
			if err := owner.Finish(ctx, true); err == nil {
				t.Fatal("deleted before inspection completion")
			}
			if _, err := os.Stat(filepath.Join(directory, "wheel.whl")); err != nil {
				t.Fatal(err)
			}
			if err := owner.NoteInspection(ctx, run, true); err != nil {
				t.Fatal(err)
			}
			if err := owner.Finish(ctx, true); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(directory); !os.IsNotExist(err) {
				t.Fatalf("last-use intake remains: %v", err)
			}
		})
	}
}
