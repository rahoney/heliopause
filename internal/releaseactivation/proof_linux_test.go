//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRecordedReleaseRejectsInvalidFactsBeforeReadingFiles(t *testing.T) {
	if _, err := readRecordedRelease(context.Background(), registration{}); err == nil {
		t.Fatal("empty registration admitted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readRecordedRelease(ctx, registration{}); !errors.Is(err, context.Canceled) {
		t.Fatal("recorded read cancellation lost")
	}
	verifier := protectedBundleVerifier{}
	if err := verifier.Verify(context.Background(), "/tmp/caller-selected-bundle"); err == nil {
		t.Fatal("foreign bundle accepted")
	}
	journal := journalFixture()
	record := candidateRegistration(journal)
	if record.Version != journal.Version || record.ManifestSHA256 != strings.Repeat("b", 64) || record.ClientUID != journal.Old.ClientUID || record.ClientGID != journal.Old.ClientGID {
		t.Fatal("candidate registration changed caller identity")
	}
}
