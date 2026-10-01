package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Repository policy and Actions syntax/context validation have separate owners.
// Optional external shell/Python analyzers are not part of this Actions gate.
func (c *checker) lintActionsFiles(paths ...string) error {
	tool, err := c.tool("actionlint")
	if err != nil {
		return err
	}
	if err := c.verifyTool(tool); err != nil {
		return err
	}
	args := append([]string{"-shellcheck=", "-pyflakes="}, paths...)
	return c.runAnalysis("Actions syntax and expression contexts", c.toolExecutable(tool), args...)
}

func (c *checker) checkActionsWorkflows() error {
	if err := checkCIWorkflow(c.root); err != nil {
		return err
	}
	var paths []string
	for _, pattern := range []string{"*.yml", "*.yaml"} {
		matches, err := filepath.Glob(filepath.Join(c.root, ".github", "workflows", pattern))
		if err != nil {
			return err
		}
		paths = append(paths, matches...)
	}
	if len(paths) == 0 {
		return fmt.Errorf("no Actions workflows selected")
	}
	if err := c.lintActionsFiles(paths...); err != nil {
		return err
	}
	// Tool-dependent regressions are explicit, offline, and not a prerequisite for
	// the self-contained default Go tests. Quick runs them after declared bootstrap.
	output, err := c.runCommandWithTimeout("Actions context regressions", commandTimeout, c.offlineEnvironment(), c.goExecutable, "test", "-count=1", "-tags=actionlint", "./scripts/check", "-run", "^TestActionlintContextAvailability$", "-v")
	if err != nil {
		return err
	}
	if !strings.Contains(output, "--- PASS: TestActionlintContextAvailability (") {
		return fmt.Errorf("actions context regression test did not execute successfully")
	}
	_, err = fmt.Fprint(c.stdout, output)
	return err
}
