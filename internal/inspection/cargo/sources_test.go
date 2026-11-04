package cargo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestCrateSourcesUseCheckedArchiveAndDiscardLateFailure(t *testing.T) {
	original, err := os.ReadFile("../../artifact/cargo/testdata/itoa-1.0.17.crate")
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := domain.NewResolvedArtifactIdentity(artifactcargo.Source(), "itoa", "1.0.17", "crate")
	for _, fault := range []string{"normal", "crc", "second-member", "cancelled"} {
		t.Run(fault, func(t *testing.T) {
			body := bytes.Clone(original)
			if fault == "crc" {
				body[len(body)-8] ^= 1
			} else if fault == "second-member" {
				body = append(body, original...)
			}
			root := t.TempDir()
			if err := os.Chmod(root, 0o700); err != nil {
				t.Fatal(err)
			}
			run, _ := domain.NewRunID()
			directory := filepath.Join(root, run.String())
			if err := os.Mkdir(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "package.crate"), body, 0o600); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(body)
			digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
			subject, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+run.String()+":crate", uint64(len(body)))
			if err != nil {
				t.Fatal(err)
			}
			inspector, _ := NewStaticInspector(root)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fault == "cancelled" {
				cancel()
			}
			sources, report, err := inspector.InspectSources(ctx, subject)
			if fault != "normal" {
				if len(sources) != 0 || (err == nil && len(report.Findings()) == 0) {
					t.Fatal("invalid or cancelled archive leaked source selection", err)
				}
				return
			}
			legacy, legacyErr := inspector.Inspect(ctx, subject)
			if err != nil || legacyErr != nil || !reflect.DeepEqual(legacy, report) || len(sources) == 0 {
				t.Fatal("source capture changed the static report", err, legacyErr)
			}
			files, err := artifactcargo.ArchiveFiles(original, identity)
			if err != nil {
				t.Fatal(err)
			}
			previous := ""
			for _, source := range sources {
				if source.Path <= previous || source.Language != "rust" || !bytes.Equal(source.Body, files[source.Path]) {
					t.Fatal("crate source selection changed measured bytes or order", source.Path)
				}
				previous = source.Path
			}
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 1 || entries[0].Name() != "package.crate" {
				t.Fatal("crate source collection extracted files", err)
			}
		})
	}
}
