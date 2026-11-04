package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

func sourceAnalysisInputFixture(t *testing.T) ports.SourceAnalysisInput {
	t.Helper()
	source, _ := domain.NewSourceID("npm")
	identity, err := domain.NewResolvedArtifactIdentity(source, "source-fixture", "1.0.0", "tarball")
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:source-fixture:tarball", 123, "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	return ports.SourceAnalysisInput{Artifact: artifact, Files: []ports.SourceAnalysisFile{{Path: "lib/main.py", Language: "python", Body: []byte("print('DATA only')\n")}}}
}

func TestSourceAnalysisInputBindsAllSixLanguagesAndOpaqueTransportPaths(t *testing.T) {
	input := sourceAnalysisInputFixture(t)
	input.Files = nil
	for _, item := range []struct{ path, language string }{{"lib/entry.jsx", "javascript"}, {"ui/view.tsx", "typescript"}, {"lib/main.py", "python"}, {"pkg/main.go", "go"}, {"src/main.rs", "rust"}, {"main.tf", "hcl"}, {"literal ;$(not-a-command) 'quoted'.py", "python"}} {
		input.Files = append(input.Files, ports.SourceAnalysisFile{Path: item.path, Language: item.language, Body: []byte("source DATA: " + item.path)})
	}
	prepared, err := prepareSourceAnalysisInput(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.targets) != len(input.Files) || len(prepared.manifestID) != 64 {
		t.Fatal("source inventory lost")
	}
	var targetList []json.RawMessage
	if err := json.Unmarshal(prepared.metadata, &targetList); err != nil || string(targetList[0]) != `"Targets"` {
		t.Fatal("wrong explicit target list", err)
	}
	var stream bytes.Buffer
	if err := prepared.writeTar(context.Background(), &stream); err != nil {
		t.Fatal(err)
	}
	if err := prepared.scanTar(bytes.NewReader(stream.Bytes())); err != nil {
		t.Fatal("original readonly source rejected", err)
	}
	archive := tar.NewReader(&stream)
	for index, file := range input.Files {
		header, err := archive.Next()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(archive)
		if err != nil || header.Name != prepared.targets[index].name || header.Name == file.Path || strings.ContainsAny(header.Name, "/;$'") || header.Mode != 0444 || header.Uid != 0 || header.Gid != 0 || !bytes.Equal(body, file.Body) {
			t.Fatal("source bytes/role or opaque path changed", header, err)
		}
	}
	last, err := archive.Next()
	if err != nil || last.Name != sourceAnalysisTargetsName || last.Mode != 0444 {
		t.Fatal("target metadata missing", err)
	}
	body, err := io.ReadAll(archive)
	if err != nil || !bytes.Equal(body, prepared.metadata) {
		t.Fatal("target metadata changed", err)
	}
	if _, err := archive.Next(); err != io.EOF {
		t.Fatal("unexpected extra source input", err)
	}
}

func TestSourceAnalysisInputRejectsUnsafeAmbiguousAndUnboundTargets(t *testing.T) {
	for _, fault := range []string{"unbound", "empty", "excessive-count", "duplicate", "parent", "absolute", "backslash", "unclean", "dot", "control", "invalid-utf8", "unknown-language"} {
		t.Run(fault, func(t *testing.T) {
			input := sourceAnalysisInputFixture(t)
			switch fault {
			case "unbound":
				input.Artifact = domain.AcquiredArtifact{}
			case "empty":
				input.Files = nil
			case "excessive-count":
				input.Files = make([]ports.SourceAnalysisFile, sourceAnalysisTargetLimit+1)
			case "duplicate":
				input.Files = append(input.Files, input.Files[0])
			case "parent":
				input.Files[0].Path = "../file.py"
			case "absolute":
				input.Files[0].Path = "/file.py"
			case "backslash":
				input.Files[0].Path = "lib\\file.py"
			case "unclean":
				input.Files[0].Path = "lib/../file.py"
			case "dot":
				input.Files[0].Path = "."
			case "control":
				input.Files[0].Path = "file\n.py"
			case "invalid-utf8":
				input.Files[0].Path = "\xff.py"
			case "unknown-language":
				input.Files[0].Language = "artifact-selected-parser"
			}
			if _, err := prepareSourceAnalysisInput(context.Background(), input); err == nil {
				t.Fatal("invalid source input admitted")
			}
		})
	}
}

func TestSourceAnalysisInputRejectsMutationAndPreservesCancellation(t *testing.T) {
	input := sourceAnalysisInputFixture(t)
	prepared, err := prepareSourceAnalysisInput(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	input.Files[0].Body[0] ^= 1
	if err := prepared.writeTar(context.Background(), io.Discard); err == nil {
		t.Fatal("source bytes changed after preparation")
	}
	input = sourceAnalysisInputFixture(t)
	prepared, err = prepareSourceAnalysisInput(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	prepared.metadata[0] ^= 1
	if err := prepared.writeTar(context.Background(), io.Discard); err == nil {
		t.Fatal("target metadata changed after preparation")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := prepareSourceAnalysisInput(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatal("source preparation lost cancellation", err)
	}
	if err := prepared.writeTar(ctx, io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatal("source copy lost cancellation", err)
	}
}

func TestSourceAnalysisReadbackRejectsMissingForeignOrChangedMembers(t *testing.T) {
	input := sourceAnalysisInputFixture(t)
	prepared, err := prepareSourceAnalysisInput(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"missing", "duplicate", "name", "mode", "owner", "symlink", "changed", "truncated", "trailing"} {
		t.Run(fault, func(t *testing.T) {
			var stream bytes.Buffer
			archive := tar.NewWriter(&stream)
			for index, item := range []struct {
				name string
				body []byte
			}{{prepared.targets[0].name, input.Files[0].Body}, {sourceAnalysisTargetsName, prepared.metadata}} {
				if index == 1 && fault == "missing" {
					continue
				}
				header := &tar.Header{Name: item.name, Typeflag: tar.TypeReg, Mode: 0444, Size: int64(len(item.body))}
				data := bytes.Clone(item.body)
				if index == 0 {
					switch fault {
					case "name":
						header.Name = "../" + item.name
					case "mode":
						header.Mode = 0555
					case "owner":
						header.Uid = 1000
					case "symlink":
						header.Typeflag = tar.TypeSymlink
						header.Linkname = "/foreign"
						header.Size = 0
					case "changed":
						data[0] ^= 1
					}
				}
				times := 1
				if index == 0 && fault == "duplicate" {
					times = 2
				}
				for count := 0; count < times; count++ {
					if err := archive.WriteHeader(header); err != nil {
						t.Fatal(err)
					}
					if header.Typeflag == tar.TypeReg {
						if _, err := archive.Write(data); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			data := stream.Bytes()
			if fault == "truncated" {
				data = data[:512+len(input.Files[0].Body)-1]
			}
			if fault == "trailing" {
				data = append(data, '!')
			}
			if err := prepared.scanTar(bytes.NewReader(data)); err == nil {
				t.Fatal("untrusted source readback accepted")
			}
		})
	}
}

func TestSourceAnalysisReadonlyVolumeSealsWithoutExecutingTargets(t *testing.T) {
	ctx := context.Background()
	input, err := prepareSourceAnalysisInput(ctx, sourceAnalysisInputFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	transaction := "source-test-source-analysis"
	name, _ := closureVolumeName(transaction)
	volume := closureVolume{name: name, transaction: transaction, manifestID: input.manifestID, createdAt: "2026-11-04T00:00:00Z", mountpoint: "/var/lib/docker/volumes/" + name + "/_data", capacity: sourceAnalysisVolumeCapacity, projectBuildDestination: sourceAnalysisRoot}
	runner := &pipxWheelVolumeRunner{pipxVolumeRunner: &pipxVolumeRunner{volume: volume, mounts: map[string]dockerContainerMountInspection{}}}
	v, err := createSourceAnalysisVolume(ctx, runner, "source-test", input)
	if err != nil || !v.valid() {
		t.Fatal("source volume rejected", err)
	}
	if v.volume.tmpfsOptions() != "size=545259520,nosuid,nodev,uid=0,gid=0,mode=0755,noexec" {
		t.Fatal("source target became executable", v.volume.tmpfsOptions())
	}
	writer, anchor := strings.Repeat("a", 64), strings.Repeat("b", 64)
	runner.attach(writer, false)
	if err := v.seed(ctx, runner, input, writer); err != nil {
		t.Fatal(err)
	}
	runner.attach(anchor, true)
	if err := v.bindAnchor(ctx, runner, anchor); err != nil {
		t.Fatal(err)
	}
	if err := v.seal(ctx, runner); err == nil {
		t.Fatal("live source writer was ignored")
	}
	runner.inventory = []string{anchor}
	if err := v.seal(ctx, runner); err != nil {
		t.Fatal(err)
	}
	if err := v.verifyReader(ctx, runner, anchor); err != nil {
		t.Fatal(err)
	}
	runner.archive[512] ^= 1
	if err := v.verifyReader(ctx, runner, anchor); err == nil {
		t.Fatal("source bytes changed after seal")
	}
	runner.inventory = nil
	if err := v.close(ctx, runner); err != nil || !runner.removed {
		t.Fatal("source input cleanup missing", err)
	}
}

func TestSourceAnalysisReadonlyVolumePreservesPartialCopyCause(t *testing.T) {
	input, err := prepareSourceAnalysisInput(context.Background(), sourceAnalysisInputFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"copy", "readback"} {
		t.Run(phase, func(t *testing.T) {
			transaction := "partial-source-source-analysis"
			name, _ := closureVolumeName(transaction)
			volume := closureVolume{name: name, transaction: transaction, manifestID: input.manifestID, createdAt: "fixed", mountpoint: "/owned", capacity: sourceAnalysisVolumeCapacity, projectBuildDestination: sourceAnalysisRoot}
			runner := &pipxWheelVolumeRunner{pipxVolumeRunner: &pipxVolumeRunner{volume: volume, mounts: map[string]dockerContainerMountInspection{}}}
			v := &readonlyToolVolume{volume: volume, source: input}
			writer := strings.Repeat("c", 64)
			runner.attach(writer, false)
			cause := errors.New("primary source transport failure")
			if phase == "copy" {
				runner.copyError = cause
			} else {
				runner.readError = cause
			}
			if err := v.seed(context.Background(), runner, input, writer); !errors.Is(err, cause) || v.seeded || v.sealed {
				t.Fatal("partial input lost cause or was adopted", err)
			}
			if err := v.seed(context.Background(), runner, input, writer); err == nil {
				t.Fatal("partial source input retried")
			}
		})
	}
}
