package main

import (
	"fmt"
	"strings"
)

const requiredResultCount = 6

func validateRequiredResults(results []string) error {
	if len(results) != requiredResultCount {
		return &checkFailure{
			class:  executionFailure,
			step:   "required aggregate",
			detail: fmt.Sprintf("received %d child results, require %d", len(results), requiredResultCount),
		}
	}

	var failed []string
	for index, result := range results {
		if result != "success" {
			failed = append(failed, fmt.Sprintf("child %d=%q", index+1, result))
		}
	}
	if len(failed) != 0 {
		return &checkFailure{class: findingFailure, step: "required aggregate", detail: strings.Join(failed, ", ")}
	}
	return nil
}
