package pypi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
)

type wheelRangeReader struct {
	*bytes.Reader
	read int64
}

func (r *wheelRangeReader) ReadAt(body []byte, offset int64) (int, error) {
	n, err := r.Reader.ReadAt(body, offset)
	r.read += int64(n)
	return n, err
}

func TestWheelInspectionScopeAuthenticatesAndIsolatesResults(t *testing.T) {
	const filename = "example-1.0-py3-none-any.whl"
	archive := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", []wheelTestEntry{
		{name: "example.py", body: []byte("value = 1\n")},
	}, []string{"py3-none-any"}, nil)
	digest := fmt.Sprintf("%x", sha256.Sum256(archive))
	target := WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}
	limits := DefaultWheelLimits()
	ctx := WithWheelInspectionScope(context.Background())
	source := PublicPyPIProfile().Source()
	inspect := func(ctx context.Context, archive []byte, limits WheelLimits) (WheelInspection, int64, error) {
		reader := &wheelRangeReader{Reader: bytes.NewReader(archive)}
		result, err := InspectWheelForSourceInScope(ctx, reader, int64(len(archive)), filename, digest, target, limits, source)
		return result, reader.read, err
	}
	first, coldReads, err := inspect(ctx, archive, limits)
	if err != nil || coldReads <= int64(len(archive)) {
		t.Fatalf("first inspection read %d bytes: %v", coldReads, err)
	}
	first.Files[0].Path = "changed"
	first.Surface.InstalledFiles[0].Destination = "changed"
	second, reads, err := inspect(ctx, archive, limits)
	if err != nil || reads != int64(len(archive)) {
		t.Fatalf("second inspection read %d bytes: %v", reads, err)
	}
	if second.Files[0].Path == "changed" || second.Surface.InstalledFiles[0].Destination == "changed" {
		t.Fatal("caller changed the stored inspection")
	}
	second.Files[0].Path = "changed again"
	third, _, err := inspect(ctx, archive, limits)
	if err != nil || third.Files[0].Path == "changed again" {
		t.Fatalf("reused result shares caller-owned fields: %v", err)
	}

	tampered := bytes.Clone(archive)
	tampered[0] ^= 1
	_, _, err = inspect(ctx, tampered, limits)
	if stage, _ := WheelValidationStageOf(err); stage != WheelValidationDigest {
		t.Fatalf("tampered bytes were not rejected by digest validation: %v", err)
	}
	limits.MaxFiles = 1
	_, _, err = inspect(ctx, archive, limits)
	if stage, _ := WheelValidationStageOf(err); stage != WheelValidationFileCount {
		t.Fatalf("new file count limit was not applied: %v", err)
	}
	_, reads, err = inspect(WithWheelInspectionScope(ctx), archive, DefaultWheelLimits())
	if err != nil || reads <= int64(len(archive)) {
		t.Fatalf("a new scope reused the previous inspection: read %d bytes: %v", reads, err)
	}
}
