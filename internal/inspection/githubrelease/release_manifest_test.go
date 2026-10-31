package githubrelease

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestReleaseMetadataInspectionBindsReadBytesAndSchema(t *testing.T) {
	published, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0", "helox-release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name                             string
		invalidSchema, tampered, symlink bool
	}{
		{name: "actual-manifest"},
		{name: "invalid-schema", invalidSchema: true},
		{name: "tampered-intake", tampered: true},
		{name: "symlink-intake", symlink: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			body := append([]byte(nil), published...)
			if test.invalidSchema {
				body = bytes.Replace(body, []byte(`"schema":`), []byte(`"unsupported":true,"schema":`), 1)
			}
			source, _ := domain.NewSourceID("github-release")
			id, _ := domain.NewResolvedArtifactIdentity(source, "rahoney-heliopause", "v0.1.0", "helox-release-manifest.json")
			sum := sha256.Sum256(body)
			digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
			run, err := domain.NewRunID()
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, run.String())
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "asset")
			if err := os.WriteFile(path, body, 0600); err != nil {
				t.Fatal(err)
			}
			artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(id, digest, "intake:"+run.String()+":github-release", uint64(len(body)), "sha256:"+digest.String())
			if err != nil {
				t.Fatal(err)
			}
			if test.tampered {
				body[len(body)-2] = '!'
				if err := os.WriteFile(path, body, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if test.symlink {
				if err := os.Rename(path, path+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".original", path); err != nil {
					t.Fatal(err)
				}
			}
			inspector, err := NewReleaseManifestInspector(root)
			if err != nil {
				t.Fatal(err)
			}
			report, err := inspector.Inspect(context.Background(), artifact)
			if test.symlink {
				if err == nil {
					t.Fatal("symlink intake accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if test.invalidSchema || test.tampered {
				if len(report.Findings()) != 1 || report.Findings()[0].Code() != "SELF_UPDATE_MANIFEST_INVALID" {
					t.Fatal("invalid metadata not reported")
				}
			} else if len(report.Findings()) != 0 || len(report.Executions()) != 1 || report.Evidence()[0].Kind() != "release-manifest-schema" {
				t.Fatal("valid passive metadata not inspected")
			}
			// The ordinary GitHub inspector still refuses JSON as unsupported.
			ordinary, err := NewStaticInspector(root)
			if err != nil {
				t.Fatal(err)
			}
			rejected, err := ordinary.Inspect(context.Background(), artifact)
			if err != nil || len(rejected.Findings()) != 1 || rejected.Findings()[0].Code() != "M6_FORMAT_UNSUPPORTED" {
				t.Fatal("generic JSON format support broadened")
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := inspector.Inspect(ctx, artifact); !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation cause lost")
			}
			badID, _ := domain.NewResolvedArtifactIdentity(source, "rahoney-heliopause", "v0.1.0", "other.json")
			other, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(badID, digest, artifact.ContentHandle(), artifact.SizeBytes(), "sha256:"+strings.Repeat("a", 64))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := inspector.Inspect(context.Background(), other); err == nil {
				t.Fatal("arbitrary JSON name accepted")
			}
		})
	}
}
