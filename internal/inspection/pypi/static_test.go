package pypi

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestInspectWheelPreservesBlockingFindingAndCompositeSkipsDynamic(t *testing.T) {
	root := t.TempDir()
	runID, err := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, runID.String())
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	filename := "example-1.0-cp314-cp314-linux_x86_64.whl"
	body := []byte("not a compatible wheel")
	digest := sha256.Sum256(body)
	if err := os.WriteFile(filepath.Join(directory, "filename"), []byte(filename), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "wheel.whl"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "wheel")
	contentDigest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, contentDigest, "intake:"+runID.String()+":wheel", uint64(len(body)), "sha256:"+hexDigest(digest[:]))
	if err != nil {
		t.Fatal(err)
	}
	static, err := NewStaticInspector(root, artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"})
	if err != nil {
		t.Fatal(err)
	}
	wheel, report, err := static.InspectWheel(context.Background(), artifact)
	if err != nil || len(report.Findings()) != 1 || report.Findings()[0].Code() != "M5_WHEEL_STATIC_INVALID" || wheel.Project != "" {
		t.Fatalf("InspectWheel() = wheel=%#v report=%#v err=%v", wheel, report, err)
	}
	dynamic, err := NewDynamicInspector(panicWheelRunner{})
	if err != nil {
		t.Fatal(err)
	}
	composite, err := NewCompositeInspector(static, dynamic)
	if err != nil {
		t.Fatal(err)
	}
	combined, err := composite.Inspect(context.Background(), artifact)
	if err != nil || len(combined.Findings()) != 1 || combined.Findings()[0].Code() != "M5_WHEEL_STATIC_INVALID" {
		t.Fatalf("Composite Inspect() = %#v, %v", combined, err)
	}
}

type panicWheelRunner struct{}

func (panicWheelRunner) InspectWheel(context.Context, domain.AcquiredArtifact, []string) (domain.SandboxResult, error) {
	panic("dynamic inspection must not run after a blocking static finding")
}

func hexDigest(value []byte) string {
	const hex = "0123456789abcdef"
	result := make([]byte, len(value)*2)
	for index, item := range value {
		result[index*2] = hex[item>>4]
		result[index*2+1] = hex[item&0xf]
	}
	return string(result)
}
