package cargo

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
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
				contentSources, contentReport, contentErr := inspector.InspectContent(ctx, subject)
				if len(contentSources) != 0 || len(contentReport.Executions()) > 1 || contentErr == nil && len(contentReport.Findings()) == 0 {
					t.Fatal("invalid archive leaked native content facts", contentErr)
				}
				return
			}
			legacy, legacyErr := inspector.Inspect(ctx, subject)
			if err != nil || legacyErr != nil || !reflect.DeepEqual(legacy, report) || len(sources) == 0 {
				t.Fatal("source capture changed the static report", err, legacyErr)
			}
			contentSources, contentReport, contentErr := inspector.InspectContent(ctx, subject)
			if contentErr != nil || !reflect.DeepEqual(contentSources, sources) || len(contentReport.Executions()) != 3 || contentReport.Executions()[1].Status() != domain.ExecutionIncomplete {
				t.Fatal("current content owner changed the committed public crate's sources", contentErr)
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

func TestCrateContentUsesVerifiedBinaryBytesBeforeSourceNames(t *testing.T) {
	identity, _ := domain.NewResolvedArtifactIdentity(artifactcargo.Source(), "fixture", "1.2.3", "crate")
	for _, withBinaries := range []bool{false, true} {
		files := map[string][]byte{
			"Cargo.toml": []byte("[package]\nname=\"fixture\"\nversion=\"1.2.3\"\nbuild='custom-build'\n[lib]\npath='custom-lib'\n[[bin]]\npath='custom-bin'\n[[example]]\npath='custom-example'\n[[test]]\npath='custom-test'\n[[bench]]\npath='custom-bench'\n"),
			"src/lib.rs": []byte("pub fn example() {}\n"),
		}
		for _, name := range []string{"custom-build", "custom-lib", "custom-bin", "custom-example", "custom-test", "custom-bench"} {
			files[name] = []byte("fn main() {}\n")
		}
		if withBinaries {
			files["hidden.rs"] = []byte("MZ")
			files["native"] = []byte("\x7fELF")
		}
		var encoded bytes.Buffer
		gz := gzip.NewWriter(&encoded)
		tarWriter := tar.NewWriter(gz)
		var names []string
		for name := range files {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if err := tarWriter.WriteHeader(&tar.Header{Name: "fixture-1.2.3/" + name, Mode: 0o644, Size: int64(len(files[name])), Typeflag: tar.TypeReg}); err != nil {
				t.Fatal(err)
			}
			if _, err := tarWriter.Write(files[name]); err != nil {
				t.Fatal(err)
			}
		}
		if err := tarWriter.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
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
		body := encoded.Bytes()
		if err := os.WriteFile(filepath.Join(directory, "package.crate"), body, 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		subject, _ := domain.NewAcquiredArtifact(identity, digest, "intake:"+run.String()+":crate", uint64(len(body)))
		inspector, _ := NewStaticInspector(root)
		sources, report, err := inspector.InspectContent(context.Background(), subject)
		if (err != nil) != withBinaries || len(sources) != 7 || len(report.Executions()) != 3 || report.Executions()[0].Status() != domain.ExecutionCompleted || report.Executions()[1].Status() != domain.ExecutionIncomplete {
			t.Fatal("crate content changed static/source facts or invented coverage", err, report.Executions())
		}
		for _, source := range sources {
			if source.Language != "rust" || !bytes.Equal(source.Body, files[source.Path]) {
				t.Fatal("metadata-selected Rust source was dropped, remapped or altered")
			}
		}
		binaryFacts := 0
		for _, fact := range report.Evidence() {
			if fact.Kind() == "binary-content-coverage" {
				binaryFacts++
				if !strings.Contains(fact.Summary(), "member_sha256") {
					t.Fatal("member digest was lost")
				}
			}
		}
		if withBinaries && binaryFacts != 2 || !withBinaries && binaryFacts != 0 {
			t.Fatal("crate native member census differs from verified map", binaryFacts)
		}
		entries, err := os.ReadDir(directory)
		if err != nil || len(entries) != 1 {
			t.Fatal("content inspection extracted a crate", err)
		}
	}
}
