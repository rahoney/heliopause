package pypi

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type corpusEntry struct {
	FileOnDisk             string
	Canonical              string
	Profile                string
	Project                string
	Version                string
	Source                 string
	ExpectedSHA256         string
	WantImports            []string
	NoImport               bool
	ExpectedCandidateCount int
}

// Both the opt-in corpus and small negative fixtures use the production digest,
// archive, RECORD, installed-role and identity validator. Nothing is executed.
func inspectCorpusWheel(root string, entry corpusEntry) (WheelInspection, error) {
	f, err := os.Open(filepath.Join(root, entry.FileOnDisk))
	if err != nil {
		return WheelInspection{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return WheelInspection{}, err
	}
	limits := DefaultWheelLimits()
	if profile, ok := PyTorchProfile(strings.TrimPrefix(entry.Profile, "pytorch:")); ok {
		limits = profile.ResourcePolicy().WheelLimits()
	}
	source, err := domain.NewSourceID(entry.Source)
	if err != nil {
		return WheelInspection{}, err
	}
	return InspectWheelForSource(f, info.Size(), entry.Canonical, entry.ExpectedSHA256,
		WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, limits, source)
}

func TestModelC_MissingCorpusDirectoryFails(t *testing.T) {
	_, err := inspectCorpusWheel(t.TempDir(), corpusEntry{FileOnDisk: "missing.whl"})
	if !os.IsNotExist(err) {
		t.Fatalf("missing required bytes: %v", err)
	}
}

func TestModelC_CorpusHashMismatchFails(t *testing.T) {
	data := recordedWheelArchive(t, "example", "1.0", "example-1.0.dist-info", nil, []string{"py3-none-any"}, nil)
	_, err := InspectWheel(bytes.NewReader(data), int64(len(data)), "example-1.0-py3-none-any.whl", strings.Repeat("0", 64), WheelTarget{"cp314", "cp314", "manylinux_2_36_x86_64"}, DefaultWheelLimits())
	stage, _ := WheelValidationStageOf(err)
	if err == nil || stage != WheelValidationDigest {
		t.Fatalf("digest mismatch must fail at digest validation: %v (%s)", err, stage)
	}
}
