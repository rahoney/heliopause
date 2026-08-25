package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// checkReleaseGate is intentionally separate from quick/PR quality. It is the
// final publication gate and must remain unavailable until repository policy
// explicitly authorizes public release assets.
func checkReleaseGate(root string) error {
	var findings []string
	license, err := os.Lstat(filepath.Join(root, "LICENSE"))
	if err != nil || !license.Mode().IsRegular() || license.Size() == 0 {
		findings = append(findings, "LICENSE is missing; release publication needs an explicit license decision")
	}
	workflowPath := filepath.Join(root, filepath.FromSlash(".github/workflows/heliopause-release-build.yml"))
	workflowBody, err := os.ReadFile(workflowPath)
	if err != nil {
		findings = append(findings, "release build workflow is unavailable")
	} else if workflowFindings := validateReleaseWorkflow(string(workflowBody)); len(workflowFindings) != 0 {
		findings = append(findings, workflowFindings...)
	}
	publishPath := filepath.Join(root, filepath.FromSlash(".github/workflows/heliopause-release-publish.yml"))
	publishInfo, err := os.Lstat(publishPath)
	if err != nil || !publishInfo.Mode().IsRegular() || publishInfo.Size() == 0 {
		findings = append(findings, "public release publication workflow is not configured")
	}
	if len(findings) == 0 {
		return nil
	}
	return &checkFailure{class: unavailable, step: "release gate", detail: strings.Join(findings, "; "), cause: errors.New("public release authority is not established")}
}

func releaseGateSummary(root string) string {
	if err := checkReleaseGate(root); err != nil {
		return fmt.Sprint(err)
	}
	return "release gate is ready"
}
