// Command check runs Heliopause's canonical local quality profiles.
package main

import (
	"fmt"
	"io"
	"os"
)

const (
	exitSuccess = 0
	exitFailure = 1
	exitUsage   = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "required" {
		if err := validateRequiredResults(args[1:]); err != nil {
			_, _ = fmt.Fprintf(stderr, "check: %v\n", err)
			return exitFailure
		}
		return exitSuccess
	}
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: go run ./scripts/check <bootstrap|bootstrap-modules|foundation|platform|quick|docs|format|required RESULTS...>")
		return exitUsage
	}

	root, err := moduleRoot()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "check: unavailable: %v\n", err)
		return exitFailure
	}

	checker, err := newChecker(root, stdout)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "check: unavailable: %v\n", err)
		return exitFailure
	}
	if err := checker.runProfile(args[0]); err != nil {
		_, _ = fmt.Fprintf(stderr, "check: %v\n", err)
		return exitFailure
	}

	return exitSuccess
}
