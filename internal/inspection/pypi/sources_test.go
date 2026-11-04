package pypi

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/inspection/content"
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
	total := wheel.ArchiveUncompressedBytes
	measured := map[string]artifactpypi.WheelFile{}
	for _, member := range wheel.Files {
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
	if err := artifactpypi.ChargeFilesFromContext(ctx, policy.MaxGraphFiles()-int64(wheel.ArchiveEntries)); err != nil {
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

// Header-only DATA exercises layout facts; it is never an executable fixture.
func sourceCaptureELF() []byte {
	body := make([]byte, 128)
	copy(body, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	order := binary.LittleEndian
	order.PutUint16(body[16:], 3)
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

func TestContentCaptureStreamsNativeFactsAndReleasesMemberBuffer(t *testing.T) {
	root := t.TempDir()
	subject := prerequisiteFixtureWheel(t, root, "example", "py3-none-any", nil, "")
	native, err := content.NewBinaryInspection(subject)
	if err != nil {
		t.Fatal(err)
	}
	capture := sourceCapture{native: native, ctx: context.Background()}
	body := sourceCaptureELF()
	writer := capture.member("example/disguised.py", uint64(len(body)))
	for _, character := range body {
		if _, err := writer.Write([]byte{character}); err != nil {
			t.Fatal(err)
		}
	}
	if member := writer.(*sourceMember); member.body.Len() != 0 || member.native || len(capture.members) != 0 {
		t.Fatal("native source disguise was retained as source or held after inspection")
	}
	report, err := native.Report(context.Background())
	if err != nil || report.Executions()[0].Status() != domain.ExecutionCompleted || len(report.Findings()) != 1 {
		t.Fatal("incremental native facts were not retained", err)
	}
	oversize := capture.member("oversize.so", ports.SourceAnalysisFileLimit+1)
	if _, err := oversize.Write(bytes.Repeat([]byte{'x'}, 512)); err != nil {
		t.Fatal(err)
	}
	if files, err := capture.files(context.Background()); err == nil || len(files) != 0 || oversize.(*sourceMember).body.Len() != 0 {
		t.Fatal("oversized native body was allocated or partially accepted")
	}
	report, err = native.Report(context.Background())
	if err == nil || report.Executions()[0].Status() == domain.ExecutionCompleted || len(report.Findings()) != 1 {
		t.Fatal("native limit lost earlier signals or invented coverage", err)
	}
}

func TestWheelContentKeepsTypedMetadataAndSingleGraphCharge(t *testing.T) {
	root := t.TempDir()
	subject := prerequisiteFixtureWheel(t, root, "example", "py3-none-any", map[string]string{
		"example/disguised.py": string(sourceCaptureELF()),
		"example/native-data":  string(sourceCaptureELF()),
	}, "")
	inspector, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
	ctx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), artifactpypi.PublicPyPIProfile())
	if err != nil {
		t.Fatal(err)
	}
	wheel, files, report, err := inspector.InspectWheelContent(ctx, subject)
	if err != nil || wheel.Project != "example" || len(files) != 1 || files[0].Path != "example/__init__.py" || len(report.Executions()) != 4 || report.Executions()[1].Status() != domain.ExecutionCompleted || len(report.Findings()) != 2 {
		t.Fatal("wheel content did not preserve typed/source/native facts", err, report.Executions(), files)
	}
	total := wheel.ArchiveUncompressedBytes
	policy := artifactpypi.ResourcePolicyFromContext(ctx)
	if err := artifactpypi.ChargeUncompressedFromContext(ctx, policy.MaxGraphUncompressed()-total); err != nil {
		t.Fatal("native capture charged the graph twice", err)
	}
	if err := artifactpypi.ChargeUncompressedFromContext(ctx, 1); err == nil {
		t.Fatal("native capture skipped graph accounting")
	}
	if err := artifactpypi.ChargeFilesFromContext(ctx, policy.MaxGraphFiles()-int64(wheel.ArchiveEntries)); err != nil {
		t.Fatal("native capture charged file census twice", err)
	}
	if err := artifactpypi.ChargeFilesFromContext(ctx, 1); err == nil {
		t.Fatal("native capture skipped file census accounting")
	}
}

func TestWheelSourcesDiscardBytesAfterLateMetadataFailure(t *testing.T) {
	root := t.TempDir()
	subject := prerequisiteFixtureWheel(t, root, "example", "py3-none-any", map[string]string{
		"example-1.0.dist-info/WHEEL": "Wheel-Version: 1.0\nRoot-Is-Purelib: true\nTag: cp313-cp313-manylinux_2_36_x86_64\n",
		"example/native-data":         string(sourceCaptureELF()),
	}, "")
	inspector, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
	wheel, files, report, err := inspector.InspectWheelSources(context.Background(), subject)
	_, legacy, legacyErr := inspector.InspectWheel(context.Background(), subject)
	if err != nil || legacyErr != nil || wheel.Project != "" || len(files) != 0 || len(report.Findings()) != 1 || !reflect.DeepEqual(legacy, report) {
		t.Fatal("late wheel validation exposed provisional bytes or lost the finding", err, legacyErr)
	}
	contentWheel, contentFiles, contentReport, contentErr := inspector.InspectWheelContent(context.Background(), subject)
	if contentErr != nil || contentWheel.Project != "" || len(contentFiles) != 0 || !reflect.DeepEqual(contentReport, legacy) {
		t.Fatal("late metadata failure published provisional content facts", contentErr)
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
		{"example/native-data", string(sourceCaptureELF())},
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
				contentInfo, contentFiles, contentReport, contentErr := inspector.InspectSdistContent(ctx, subject)
				if contentInfo.Project != "" || len(contentFiles) != 0 || len(contentReport.Executions()) > 1 || contentErr == nil && len(contentReport.Findings()) == 0 {
					t.Fatal("unverified sdist exposed provisional native facts", contentErr)
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
			contentInfo, contentFiles, contentReport, contentErr := inspector.InspectSdistContent(ctx, subject)
			if contentErr != nil || contentInfo.Project != info.Project || !reflect.DeepEqual(contentFiles, files) || len(contentReport.Executions()) != 4 || contentReport.Executions()[1].Status() != domain.ExecutionCompleted || len(contentReport.Findings()) != 1 {
				t.Fatal("sdist content lost typed metadata/source/native facts", contentErr)
			}
		})
	}
}

func TestWheelContentCensusIncludesEmptyOpaqueAndExemptMetadata(t *testing.T) {
	for _, lane := range []string{"public", "cpu"} {
		t.Run(lane, func(t *testing.T) {
			root := t.TempDir()
			subject := prerequisiteFixtureWheel(t, root, "example", "py3-none-any", map[string]string{
				"example/empty.txt": "", "example/opaque.dat": "DATA", "example/helper.sh": "#!/bin/sh\necho DATA\n",
			}, "")
			profile := artifactpypi.PublicPyPIProfile()
			if lane == "cpu" {
				var ok bool
				profile, ok = artifactpypi.PyTorchProfile(lane)
				if !ok {
					t.Fatal("fixed CPU profile absent")
				}
			}
			ctx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), profile)
			if err != nil {
				t.Fatal(err)
			}
			inspector, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
			wheel, files, report, err := inspector.InspectWheelContent(ctx, subject)
			if err != nil || len(files) != 1 || len(report.Executions()) != 4 {
				t.Fatal("complete wheel census unavailable", err)
			}
			var census struct {
				Members     int    `json:"members"`
				Sources     int    `json:"sources"`
				Binaries    int    `json:"binaries"`
				Unsupported int    `json:"unsupported_sources"`
				MaxMembers  int    `json:"max_members"`
				MaxBytes    uint64 `json:"max_bytes"`
			}
			count := 0
			for _, fact := range report.Evidence() {
				if fact.Kind() == "content-inventory" {
					count++
					if err := json.Unmarshal([]byte(fact.Summary()), &census); err != nil {
						t.Fatal(err)
					}
				}
			}
			limits := profile.ResourcePolicy().WheelLimits()
			if count != 1 || census.Members != len(wheel.Files) || census.Sources != 1 || census.Binaries != 0 || census.Unsupported != 1 || census.MaxMembers != int(limits.MaxFiles) || census.MaxBytes != uint64(limits.MaxUncompressed) {
				t.Fatal("census omitted regular DATA/RECORD or changed archive policy", census)
			}
		})
	}
}

func TestWheelGraphAccountingIncludesSelfRecordBytes(t *testing.T) {
	root := t.TempDir()
	subject := prerequisiteFixtureWheel(t, root, "example", "py3-none-any", nil, "")
	inspector, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
	name, _, err := inspector.artifactPath(subject)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(name)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	var actual int64
	for _, member := range archive.File {
		actual += int64(member.UncompressedSize64)
	}
	ctx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), artifactpypi.PublicPyPIProfile())
	if err != nil {
		t.Fatal(err)
	}
	policy := artifactpypi.ResourcePolicyFromContext(ctx)
	if err := artifactpypi.ChargeUncompressedFromContext(ctx, policy.MaxGraphUncompressed()-actual); err != nil {
		t.Fatal(err)
	}
	if _, report, err := inspector.InspectWheel(ctx, subject); err != nil || len(report.Findings()) != 0 {
		t.Fatal("exact archive budget rejected", err)
	}
	if err := artifactpypi.ChargeUncompressedFromContext(ctx, 1); err == nil {
		t.Fatal("self RECORD bytes were omitted from graph accounting")
	}
}

func TestWheelContentUsesAuthenticatedInstalledRoles(t *testing.T) {
	root := t.TempDir()
	extras := map[string]string{
		"active.pth":                          "# original hook\nimport os; os.environ.get('CONTENT_DATA')\n",
		"example-1.0.data/purelib/mapped.pth": "import sys\n",
		"declarative.pth":                     "example\n",
		"example/nested.pth":                  "import os\n",
		"example-1.0.data/scripts/unknown":    "opaque executable script\n",
		"example-1.0.data/scripts/python":     "#!/usr/bin/env python3\nprint('DATA')\n",
		"example-1.0.data/scripts/native":     string(sourceCaptureELF()),
	}
	subject := prerequisiteFixtureWheel(t, root, "example", "py3-none-any", extras, "")
	inspector, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
	ctx, err := artifactpypi.ContextWithResourcePolicy(context.Background(), artifactpypi.PublicPyPIProfile())
	if err != nil {
		t.Fatal(err)
	}
	wheel, files, report, err := inspector.InspectWheelContent(ctx, subject)
	if err != nil || len(report.Findings()) != 1 {
		t.Fatal("installed role content unavailable", err, report.Findings())
	}
	want := map[string]string{"example/__init__.py": "pass\n", "active.pth": extras["active.pth"], "example-1.0.data/purelib/mapped.pth": extras["example-1.0.data/purelib/mapped.pth"], "example-1.0.data/scripts/python": extras["example-1.0.data/scripts/python"]}
	if len(files) != len(want) {
		t.Fatalf("active installed source omitted or DATA promoted: got %d want %d", len(files), len(want))
	}
	measured := map[string]artifactpypi.WheelFile{}
	for _, member := range wheel.Files {
		measured[member.Path] = member
	}
	for _, file := range files {
		body, exists := want[file.Path]
		sum := sha256.Sum256(file.Body)
		member := measured[file.Path]
		if !exists || file.Language != "python" || string(file.Body) != body || member.Size != int64(len(file.Body)) || member.SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatal("hook source was transformed, mismapped or unauthenticated", file.Path)
		}
		delete(want, file.Path)
	}
	var census struct {
		Sources     int `json:"sources"`
		Unsupported int `json:"unsupported_sources"`
		Binaries    int `json:"binaries"`
	}
	for _, fact := range report.Evidence() {
		if fact.Kind() == "content-inventory" {
			if err := json.Unmarshal([]byte(fact.Summary()), &census); err != nil {
				t.Fatal(err)
			}
		}
	}
	if census.Sources != 4 || census.Unsupported != 1 || census.Binaries != 1 {
		t.Fatal("installed roles lost source/unknown/native distinction", census)
	}
	policy := artifactpypi.ResourcePolicyFromContext(ctx)
	if err := artifactpypi.ChargeUncompressedFromContext(ctx, policy.MaxGraphUncompressed()-wheel.ArchiveUncompressedBytes); err != nil {
		t.Fatal("hook reread charged bytes twice", err)
	}
	if err := artifactpypi.ChargeUncompressedFromContext(ctx, 1); err == nil {
		t.Fatal("original archive was not charged")
	}
	if err := artifactpypi.ChargeFilesFromContext(ctx, policy.MaxGraphFiles()-int64(wheel.ArchiveEntries)); err != nil {
		t.Fatal("hook reread charged files twice", err)
	}
	if err := artifactpypi.ChargeFilesFromContext(ctx, 1); err == nil {
		t.Fatal("original entry census was not charged")
	}
}

func TestWheelRoleRereadRefusesChangesAndCombinedBounds(t *testing.T) {
	for _, fault := range []string{"archive", "missing", "cancelled", "total", "targets", "member-digest"} {
		t.Run(fault, func(t *testing.T) {
			root := t.TempDir()
			subject := prerequisiteFixtureWheel(t, root, "example", "py3-none-any", map[string]string{"active.pth": "import os\n"}, "")
			inspector, _ := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
			native, _ := content.NewBinaryInspection(subject)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			capture := sourceCapture{native: native, ctx: ctx}
			wheel, report, err := inspector.inspectWheel(ctx, subject, capture.member)
			if err != nil || len(report.Findings()) != 0 {
				t.Fatal("first authenticated read failed", err)
			}
			original := len(capture.members)
			name, _, err := inspector.artifactPath(subject)
			if err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "archive":
				body, err := os.ReadFile(name)
				if err != nil {
					t.Fatal(err)
				}
				body[len(body)-1] ^= 1
				if err := os.WriteFile(name, body, 0600); err != nil {
					t.Fatal(err)
				}
			case "missing":
				if err := os.Remove(name); err != nil {
					t.Fatal(err)
				}
			case "cancelled":
				cancel(fmt.Errorf("role capture cancelled"))
			case "total":
				capture.bytes = ports.SourceAnalysisTotalLimit
			case "targets":
				capture.targets = ports.SourceAnalysisTargetLimit
			case "member-digest":
				wrong, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
				for index := range capture.facts {
					if capture.facts[index].Path == "active.pth" {
						capture.facts[index].Digest = wrong
					}
				}
			}
			if err := capture.wheelRoles(ctx, inspector, subject, wheel); err == nil {
				t.Fatal("unauthenticated/partial/beyond-budget source was accepted", fault)
			}
			if len(capture.members) != original {
				t.Fatal("failed second read exposed provisional source bodies")
			}
			for _, fact := range capture.facts {
				if fact.Path == "active.pth" && fact.Language != "" {
					t.Fatal("failed reread published hook coverage")
				}
			}
		})
	}
}
