package domain

import (
	"bytes"
	"errors"
	"testing"
)

func TestGenerateAndParseIdentifiers(t *testing.T) {
	t.Parallel()

	operationText, err := generateID("op_", bytes.NewReader(make([]byte, randomIDBytes)))
	if err != nil {
		t.Fatalf("generateID() error = %v", err)
	}
	if operationText != "op_aaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("operation ID = %q", operationText)
	}
	operationID, err := ParseOperationID(operationText)
	if err != nil || operationID.String() != operationText {
		t.Fatalf("ParseOperationID() = %q, %v", operationID.String(), err)
	}

	runText, err := generateID("run_", bytes.NewReader(bytes.Repeat([]byte{0xff}, randomIDBytes)))
	if err != nil {
		t.Fatalf("generateID() error = %v", err)
	}
	runID, err := ParseRunID(runText)
	if err != nil || runID.String() != runText {
		t.Fatalf("ParseRunID() = %q, %v", runID.String(), err)
	}
}

func TestIdentifierValidationRejectsNonCanonicalValues(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"op_aaaaaaaaaaaaaaaaaaaaaaaaa",
		"op_AAAAAAAAAAAAAAAAAAAAAAAAAA",
		"op_aaaaaaaaaaaaaaaaaaaaaaaaab",
		"run_aaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	for _, value := range tests {
		value := value
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseOperationID(value); err == nil {
				t.Fatal("ParseOperationID() error = nil")
			}
		})
	}
}

func TestGenerateIDPreservesRandomFailure(t *testing.T) {
	t.Parallel()

	want := errors.New("random unavailable")
	_, err := generateID("run_", errorReader{err: want})
	if !errors.Is(err, want) {
		t.Fatalf("generateID() error = %v, want wrapped %v", err, want)
	}
}

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }
