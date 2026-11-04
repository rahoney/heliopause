package githubrelease

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func releaseSourceArchives(t *testing.T, native bool) ([]byte, []byte) {
	t.Helper()
	var zipped bytes.Buffer
	zw := zip.NewWriter(&zipped)
	member, err := zw.CreateHeader(&zip.FileHeader{Name: "source.py", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := member.Write([]byte("raise RuntimeError('DATA only')")); err != nil {
		t.Fatal(err)
	}
	member, err = zw.Create("bin/tool")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := member.Write([]byte("#!/usr/bin/env node\nconsole.log('DATA');\n")); err != nil {
		t.Fatal(err)
	}
	if native {
		for _, name := range []string{"native", "disguised.py"} {
			entry, err := zw.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := entry.Write(releaseContentELF()); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "source.py", Mode: 0o644, Size: 4, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("pass")); err != nil {
		t.Fatal(err)
	}
	script := "#!/usr/bin/env node\nconsole.log('DATA');\n"
	if err := tw.WriteHeader(&tar.Header{Name: "bin/tool", Mode: 0o755, Size: int64(len(script)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(script)); err != nil {
		t.Fatal(err)
	}
	if native {
		body := releaseContentELF()
		for _, name := range []string{"native", "disguised.py"} {
			if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
				t.Fatal(err)
			}
			if _, err := tw.Write(body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return zipped.Bytes(), compressed.Bytes()
}

func TestReleaseArchiveRejectsUnverifiedPayload(t *testing.T) {
	zipped, compressed := releaseSourceArchives(t, false)
	for _, fault := range []string{"zip-crc", "gzip-crc", "gzip-second-member", "gzip-compressed-tail"} {
		t.Run(fault, func(t *testing.T) {
			body := bytes.Clone(compressed)
			switch fault {
			case "zip-crc":
				body = bytes.Clone(zipped)
				position := bytes.Index(body, []byte("raise RuntimeError"))
				if position < 0 {
					t.Fatal("fixture payload missing")
				}
				body[position] ^= 1
			case "gzip-crc":
				body[len(body)-8] ^= 1
			case "gzip-second-member":
				body = append(body, body...)
			case "gzip-compressed-tail":
				body = append(body, 1)
			}
			name := filepath.Join(t.TempDir(), "asset")
			if err := os.WriteFile(name, body, 0o600); err != nil {
				t.Fatal(err)
			}
			file, err := os.Open(name)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			if fault == "zip-crc" {
				err = inspectZIP(context.Background(), file, int64(len(body)), nil, nil)
			} else {
				err = inspectTarGZ(context.Background(), file, nil, nil)
			}
			if err == nil {
				t.Fatal("unverified archive payload accepted")
			}
		})
	}
}

func TestReleaseSourcesBindOriginalAndDiscardInvalidArchive(t *testing.T) {
	zipped, compressed := releaseSourceArchives(t, false)
	for _, fault := range []string{"zip", "targz", "digest", "late-crc", "cancelled", "run-symlink"} {
		t.Run(fault, func(t *testing.T) {
			body, filename := zipped, "tool.zip"
			if fault == "targz" || fault == "late-crc" {
				body, filename = bytes.Clone(compressed), "tool.tar.gz"
			}
			if fault == "late-crc" {
				body[len(body)-8] ^= 1
			}
			root := t.TempDir()
			run, _ := domain.NewRunID()
			directory := filepath.Join(root, run.String())
			if fault == "run-symlink" {
				directory = t.TempDir()
				if err := os.Symlink(directory, filepath.Join(root, run.String())); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Mkdir(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "asset"), body, 0o600); err != nil {
				t.Fatal(err)
			}
			subject := testArtifact(t, run, filename, uint64(len(body)))
			if fault != "digest" {
				sum := sha256.Sum256(body)
				digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
				subject, _ = domain.NewAcquiredArtifact(subject.Identity(), digest, subject.ContentHandle(), subject.SizeBytes())
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fault == "cancelled" {
				cancel()
			}
			inspector, _ := NewStaticInspector(root)
			files, report, err := inspector.InspectSources(ctx, subject)
			if fault != "zip" && fault != "targz" {
				if len(files) != 0 || (err == nil && len(report.Findings()) == 0) {
					t.Fatal("unverified source bytes escaped", err)
				}
				contentFiles, contentReport, contentErr := inspector.InspectContent(ctx, subject)
				if len(contentFiles) != 0 || len(contentReport.Executions()) > 1 || contentErr == nil && len(contentReport.Findings()) == 0 {
					t.Fatal("unverified archive exposed native content", contentErr)
				}
				return
			}
			legacy, legacyErr := inspector.Inspect(ctx, subject)
			if err != nil || legacyErr != nil || !reflect.DeepEqual(legacy, report) || len(files) != 2 || files[0].Path != "source.py" || files[0].Language != "python" || files[1].Path != "bin/tool" || files[1].Language != "javascript" {
				t.Fatal("source selection changed static facts or parser DATA", err, legacyErr, files)
			}
			want := "raise RuntimeError('DATA only')"
			if fault == "targz" {
				want = "pass"
			}
			if string(files[0].Body) != want {
				t.Fatal("source payload differs from archive bytes")
			}
			contentFiles, contentReport, contentErr := inspector.InspectContent(ctx, subject)
			if contentErr != nil || !reflect.DeepEqual(contentFiles, files) || len(contentReport.Executions()) != 3 || contentReport.Executions()[1].Status() != domain.ExecutionIncomplete {
				t.Fatal("release content changed source DATA or invented zero-binary applicability", contentErr)
			}
			old, _ := domain.CheckProfileDigest("github-release-static-v1", "")
			checks, err := inspector.ExpectedChecks(ctx, subject, domain.CheckInspection)
			if err != nil || len(checks) != 1 || checks[0].ProfileDigest() != report.Execution().ProfileDigest() || checks[0].ProfileDigest() == old {
				t.Fatal("old header-only profile satisfies the new content validation", err)
			}
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 1 || entries[0].Name() != "asset" {
				t.Fatal("source collection extracted files", err)
			}
		})
	}
}

// Bounded header DATA only; none of these bytes is ever executed.
func releaseContentELF() []byte {
	body := make([]byte, 128)
	copy(body, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	order := binary.LittleEndian
	order.PutUint16(body[16:], 2)
	order.PutUint16(body[18:], 62)
	order.PutUint32(body[20:], 1)
	order.PutUint64(body[32:], 64)
	order.PutUint16(body[52:], 64)
	order.PutUint16(body[54:], 56)
	order.PutUint16(body[56:], 1)
	order.PutUint32(body[64:], 1)
	order.PutUint32(body[68:], 7)
	order.PutUint64(body[96:], uint64(len(body)))
	order.PutUint64(body[104:], uint64(len(body)))
	return body
}

func TestReleaseContentUsesHeldRawAndArchiveNativeBytes(t *testing.T) {
	zipped, compressed := releaseSourceArchives(t, true)
	for _, scenario := range []string{"zip", "targz", "raw", "late-crc"} {
		t.Run(scenario, func(t *testing.T) {
			body, filename, expectedMembers := zipped, "tool.zip", 2
			if scenario == "targz" || scenario == "late-crc" {
				body, filename = bytes.Clone(compressed), "tool.tar.gz"
			} else if scenario == "raw" {
				body, filename, expectedMembers = releaseContentELF(), "tool", 1
			}
			if scenario == "late-crc" {
				body[len(body)-8] ^= 1
			}
			root := t.TempDir()
			run, _ := domain.NewRunID()
			directory := filepath.Join(root, run.String())
			if err := os.Mkdir(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "asset"), body, 0o600); err != nil {
				t.Fatal(err)
			}
			subject := testArtifact(t, run, filename, uint64(len(body)))
			sum := sha256.Sum256(body)
			digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
			subject, _ = domain.NewAcquiredArtifact(subject.Identity(), digest, subject.ContentHandle(), subject.SizeBytes())
			inspector, _ := NewStaticInspector(root)
			files, report, err := inspector.InspectContent(context.Background(), subject)
			if scenario == "late-crc" {
				if len(files) != 0 || len(report.Executions()) != 1 || len(report.Findings()) != 1 {
					t.Fatal("failed whole archive published provisional native signals", err)
				}
				return
			}
			if err != nil || len(report.Executions()) != 3 || report.Executions()[1].Status() != domain.ExecutionCompleted || len(report.Findings()) != expectedMembers {
				t.Fatal("native release coverage or signals were lost", err, report.Executions())
			}
			if scenario == "raw" && len(files) != 0 || scenario != "raw" && len(files) != 2 {
				t.Fatal("binary DATA was duplicated into source targets")
			}
			facts := 0
			for _, fact := range report.Evidence() {
				if fact.Kind() == "binary-content-coverage" {
					facts++
					if !strings.Contains(fact.Summary(), "member_sha256") {
						t.Fatal("native member digest disappeared")
					}
				}
			}
			if facts != expectedMembers {
				t.Fatal("native census differs from held bytes")
			}
		})
	}
}
