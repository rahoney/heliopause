package pypi

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

func TestWheelSourcesPreserveTypedResultAndSingleGraphCharge(t *testing.T) {
	root := t.TempDir()
	subject := prerequisiteFixtureWheel(t, root, "example", "py3-none-any", map[string]string{"example/helper.js": "console.log('DATA');\n", "example/cli": "#!/usr/bin/env node\nconsole.log('DATA');\n"}, "")
	inspector, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
	ctx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), artifactpypi.PublicPyPIProfile())
	if err != nil {
		t.Fatal(err)
	}
	wheel, files, report, err := inspector.InspectWheelSources(ctx, subject)
	if err != nil || len(files) != 3 || wheel.Project != "example" || len(report.Findings()) != 0 {
		t.Fatal("typed source collection unavailable", err, files)
	}
	_, legacy, err := inspector.InspectWheel(context.Background(), subject)
	if err != nil || !reflect.DeepEqual(legacy, report) {
		t.Fatal("source collection changed the static report", err)
	}
	oldProfile, _ := domain.CheckProfileDigest(staticCheckProfileName, fmt.Sprintf("%q|%s", []string{inspector.target.Python, inspector.target.ABI, inspector.target.Platform}, pythonInspectionProfileDescriptor(ctx)))
	if report.Execution().ProfileDigest() != oldProfile {
		t.Fatal("unchanged wheel validation unnecessarily invalidated the old profile")
	}
	var total int64
	measured := map[string]artifactpypi.WheelFile{}
	for _, member := range wheel.Files {
		total += member.Size
		measured[member.Path] = member
	}
	for _, file := range files {
		sum := sha256.Sum256(file.Body)
		member := measured[file.Path]
		if int64(len(file.Body)) != member.Size || hex.EncodeToString(sum[:]) != member.SHA256 {
			t.Fatal("captured source differs from authenticated RECORD", file.Path)
		}
	}
	policy := artifactpypi.ResourcePolicyFromContext(ctx)
	if err := artifactpypi.ChargeUncompressedFromContext(ctx, policy.MaxGraphUncompressed()-total); err != nil {
		t.Fatal("source capture charged the graph twice", err)
	}
	if err := artifactpypi.ChargeUncompressedFromContext(ctx, 1); err == nil {
		t.Fatal("source capture skipped original graph accounting")
	}
	if err := artifactpypi.ChargeFilesFromContext(ctx, policy.MaxGraphFiles()-int64(len(wheel.Files))); err != nil {
		t.Fatal("source capture charged file census twice", err)
	}
	if err := artifactpypi.ChargeFilesFromContext(ctx, 1); err == nil {
		t.Fatal("source capture skipped file census accounting")
	}
}

func TestSourceCaptureRetainsSplitPrefixAndBodyWithoutBufferingOpaqueData(t *testing.T) {
	var capture sourceCapture
	body := []byte("#!/usr/bin/env python3\n" + strings.Repeat("x", 700))
	writer := capture.member("bin/tool", uint64(len(body)))
	for _, character := range body {
		if _, err := writer.Write([]byte{character}); err != nil {
			t.Fatal(err)
		}
	}
	unknown := capture.member("data/blob", 4096)
	if _, err := unknown.Write(bytes.Repeat([]byte{'x'}, 4096)); err != nil {
		t.Fatal(err)
	}
	files, err := capture.files(context.Background())
	if err != nil || len(files) != 1 || files[0].Language != "python" || !bytes.Equal(files[0].Body, body) || capture.bytes != uint64(len(body)) {
		t.Fatal("split source prefix was dropped, duplicated or charged as opaque data", err)
	}
}

func TestWheelSourcesDiscardBytesAfterLateMetadataFailure(t *testing.T) {
	root := t.TempDir()
	subject := prerequisiteFixtureWheel(t, root, "example", "py3-none-any", map[string]string{
		"example-1.0.dist-info/WHEEL": "Wheel-Version: 1.0\nRoot-Is-Purelib: true\nTag: cp313-cp313-manylinux_2_36_x86_64\n",
	}, "")
	inspector, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
	wheel, files, report, err := inspector.InspectWheelSources(context.Background(), subject)
	_, legacy, legacyErr := inspector.InspectWheel(context.Background(), subject)
	if err != nil || legacyErr != nil || wheel.Project != "" || len(files) != 0 || len(report.Findings()) != 1 || !reflect.DeepEqual(legacy, report) {
		t.Fatal("late wheel validation exposed provisional bytes or lost the finding", err, legacyErr)
	}
}

func TestSourceCaptureBoundsDiscardTheWholeSelection(t *testing.T) {
	for _, limit := range []string{"file", "total", "count"} {
		t.Run(limit, func(t *testing.T) {
			var capture sourceCapture
			if _, err := capture.member("first.py", 1).Write([]byte("x")); err != nil {
				t.Fatal(err)
			}
			switch limit {
			case "file":
				capture.member("large.py", ports.SourceAnalysisFileLimit+1)
			case "total":
				for n := 0; n < ports.SourceAnalysisTotalLimit/ports.SourceAnalysisFileLimit; n++ {
					capture.member("large.py", ports.SourceAnalysisFileLimit)
				}
			case "count":
				for n := 0; n < ports.SourceAnalysisTargetLimit; n++ {
					capture.member("empty.py", 0)
				}
			}
			if files, err := capture.files(context.Background()); err == nil || len(files) != 0 || capture.member("later.py", 1) != nil {
				t.Fatal("source bound silently produced a partial selection")
			}
		})
	}
}

func TestSdistSourcesReturnOnlyAfterWholeArchiveValidation(t *testing.T) {
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	for _, member := range []struct{ name, body string }{
		{"PKG-INFO", "Name: example\nVersion: 1.0\n"},
		{"pyproject.toml", "[build-system]\nrequires = [\"wheel\"]\nbuild-backend = \"setuptools.build_meta\"\n"},
		{"example/__init__.py", "raise RuntimeError('must never execute')\n"},
	} {
		if err := tw.WriteHeader(&tar.Header{Name: "example-1.0/" + member.name, Mode: 0o644, Size: int64(len(member.body)), Typeflag: tar.TypeReg, Format: tar.FormatPAX, PAXRecords: map[string]string{"comment": "source DATA"}}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(member.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"normal", "crc", "second-member", "cancelled"} {
		t.Run(fault, func(t *testing.T) {
			body := bytes.Clone(archive.Bytes())
			if fault == "crc" {
				body[len(body)-8] ^= 1
			} else if fault == "second-member" {
				body = append(body, archive.Bytes()...)
			}
			root := t.TempDir()
			run, _ := domain.NewRunID()
			directory := filepath.Join(root, run.String())
			if err := os.Mkdir(directory, 0o700); err != nil {
				t.Fatal(err)
			}
			for name, value := range map[string][]byte{"sdist.tar.gz": body, "filename": []byte("example-1.0.tar.gz")} {
				if err := os.WriteFile(filepath.Join(directory, name), value, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			sum := sha256.Sum256(body)
			digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
			id, _ := domain.NewResolvedArtifactIdentity(artifactpypi.PublicPyPIProfile().Source(), "example", "1.0", "sdist")
			subject, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(id, digest, "intake:"+run.String()+":sdist", uint64(len(body)), "sha256:"+digest.String())
			inspector, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if fault == "cancelled" {
				cancel()
			}
			info, files, report, err := inspector.InspectSdistSources(ctx, subject)
			if fault != "normal" {
				if info.Project != "" || len(files) != 0 || (err == nil && len(report.Findings()) == 0) {
					t.Fatal("unverified sdist exposed source selection", err)
				}
				return
			}
			if err != nil || info.Project != "example" || len(files) != 1 || files[0].Path != "example/__init__.py" || files[0].Language != "python" || !bytes.Contains(files[0].Body, []byte("must never execute")) {
				t.Fatal("validated source distribution lost metadata or bytes", err, files)
			}
			oldProfile, _ := domain.CheckProfileDigest(staticCheckProfileName, fmt.Sprintf("%q|%s", []string{inspector.target.Python, inspector.target.ABI, inspector.target.Platform}, pythonInspectionProfileDescriptor(ctx)))
			planned, err := inspector.ExpectedChecks(ctx, subject, domain.CheckInspection)
			if err != nil || len(planned) != 1 || planned[0].ProfileDigest() != report.Execution().ProfileDigest() || report.Execution().ProfileDigest() == oldProfile {
				t.Fatal("source plan accepted a result without the new trailer validation", err)
			}
		})
	}
}
