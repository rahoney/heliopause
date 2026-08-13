package main

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestValidateRequiredResults(t *testing.T) {
	t.Parallel()

	if err := validateRequiredResults([]string{"success", "success"}); err != nil {
		t.Fatalf("validateRequiredResults success error: %v", err)
	}

	for _, result := range []string{"failure", "cancelled", "skipped", ""} {
		result := result
		t.Run(result, func(t *testing.T) {
			t.Parallel()
			var failure *checkFailure
			err := validateRequiredResults([]string{"success", result})
			if !errors.As(err, &failure) || failure.class != findingFailure {
				t.Fatalf("validateRequiredResults(%q) error = %v, want finding", result, err)
			}
			if !strings.Contains(err.Error(), result) && result != "" {
				t.Fatalf("error %q does not preserve result %q", err, result)
			}
		})
	}
}

func TestValidateRequiredResultsRejectsMissingChild(t *testing.T) {
	t.Parallel()

	var failure *checkFailure
	err := validateRequiredResults([]string{"success"})
	if !errors.As(err, &failure) || failure.class != executionFailure {
		t.Fatalf("validateRequiredResults error = %v, want execution failure", err)
	}
}

func TestRunRequiredExit(t *testing.T) {
	t.Parallel()

	if code := run([]string{"required", "success", "success"}, io.Discard, io.Discard); code != exitSuccess {
		t.Fatalf("run success code = %d", code)
	}
	var stderr strings.Builder
	if code := run([]string{"required", "success", "cancelled"}, io.Discard, &stderr); code != exitFailure {
		t.Fatalf("run cancelled code = %d", code)
	}
	if !strings.Contains(stderr.String(), "cancelled") {
		t.Fatalf("stderr = %q, want cancelled", stderr.String())
	}
}
