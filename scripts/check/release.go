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
	licenseBody, err := readReleasePolicyFile(root, "LICENSE")
	if err != nil {
		findings = append(findings, "LICENSE is missing; release publication needs an explicit license decision")
	} else if !strings.Contains(licenseBody, "Apache License") || !strings.Contains(licenseBody, "Version 2.0") {
		findings = append(findings, "LICENSE does not declare Apache-2.0")
	}
	claBody, err := readReleasePolicyFile(root, "CLA.md")
	if err != nil {
		findings = append(findings, "CLA.md is missing; external contribution rights are not established")
	} else {
		for _, required := range []string{"Harmony", "Copyright License", "Patent License", "Option Five", "Individual contributor", "Legal entity contributor"} {
			if !strings.Contains(claBody, required) {
				findings = append(findings, "CLA.md is missing required policy: "+required)
			}
		}
		if strings.Contains(claBody, "[PROJECT GOVERNING JURISDICTION") {
			findings = append(findings, "CLA governing jurisdiction is not confirmed")
		}
	}
	contributingBody, err := readReleasePolicyFile(root, "CONTRIBUTING.md")
	if err != nil {
		findings = append(findings, "CONTRIBUTING.md is missing; CLA merge policy is not established")
	} else if !strings.Contains(contributingBody, "automated CLA status check") || !strings.Contains(contributingBody, "merge blocker") {
		findings = append(findings, "CONTRIBUTING.md does not require a successful CLA status check before merge")
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
	} else if publishBody, readErr := os.ReadFile(publishPath); readErr != nil {
		findings = append(findings, "public release publication workflow is unreadable")
	} else if workflowFindings := validateReleasePublishWorkflow(string(publishBody)); len(workflowFindings) != 0 {
		findings = append(findings, workflowFindings...)
	}
	if len(findings) == 0 {
		return nil
	}
	return &checkFailure{class: unavailable, step: "release gate", detail: strings.Join(findings, "; "), cause: errors.New("public release authority is not established")}
}

func readReleasePolicyFile(root, name string) (string, error) {
	path := filepath.Join(root, name)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return "", errors.New("policy file is unavailable")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return "", errors.New("policy file is unreadable")
	}
	return string(body), nil
}

func releaseGateSummary(root string) string {
	if err := checkReleaseGate(root); err != nil {
		return fmt.Sprint(err)
	}
	return "release gate is ready"
}
