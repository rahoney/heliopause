package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type packageMetadata struct {
	ImportPath string
	Imports    []string
}

func (c *checker) checkArchitecture() error {
	moduleOutput, err := c.runCommand("architecture module identity", c.goExecutable, "list", "-m", "-f", "{{.Path}}")
	if err != nil {
		return err
	}
	modulePath := strings.TrimSpace(moduleOutput)
	if modulePath == "" {
		return &checkFailure{class: executionFailure, step: "architecture", detail: "go list returned an empty module path"}
	}

	packages, err := c.listPackages()
	if err != nil {
		return err
	}
	findings := validateCurrentImports(modulePath, packages)
	if len(findings) != 0 {
		return &checkFailure{class: findingFailure, step: "architecture", detail: strings.Join(findings, "\n")}
	}
	return nil
}

func (c *checker) listPackages() ([]packageMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, c.goExecutable, "list", "-json", "./...")
	command.Dir = c.root
	command.Env = c.offlineEnvironment()
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, &checkFailure{class: executionFailure, step: "architecture package inventory", cause: err}
	}
	var stderr boundedBuffer
	stderr.limit = outputLimit
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return nil, &checkFailure{class: executionFailure, step: "architecture package inventory", cause: err}
	}

	var packages []packageMetadata
	decoder := json.NewDecoder(stdout)
	for {
		var metadata packageMetadata
		decodeErr := decoder.Decode(&metadata)
		if decodeErr == io.EOF {
			break
		}
		if decodeErr != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
			return nil, &checkFailure{class: executionFailure, step: "architecture package inventory", detail: "decode go list output", cause: decodeErr}
		}
		packages = append(packages, metadata)
	}
	err = command.Wait()
	if ctx.Err() != nil {
		return nil, &checkFailure{class: executionFailure, step: "architecture package inventory", detail: "command timed out", cause: ctx.Err()}
	}
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		return nil, &checkFailure{class: executionFailure, step: "architecture package inventory", detail: detail, cause: err}
	}
	return packages, nil
}

func validateCurrentImports(modulePath string, packages []packageMetadata) []string {
	allowedProductImports := map[string]map[string]bool{
		modulePath + "/cmd/helox": {
			modulePath + "/internal/bootstrap": true,
		},
		modulePath + "/internal/cli":         {},
		modulePath + "/internal/core/domain": {},
		modulePath + "/internal/core/ports": {
			modulePath + "/internal/core/domain": true,
		},
		modulePath + "/internal/application": {
			modulePath + "/internal/core/domain": true,
			modulePath + "/internal/core/ports":  true,
		},
		modulePath + "/internal/policy": {
			modulePath + "/internal/core/domain": true,
		},
		modulePath + "/internal/testutil/fakeworkflow": {
			modulePath + "/internal/core/domain": true,
			modulePath + "/internal/core/ports":  true,
		},
		modulePath + "/scripts/check": {},
	}
	allowedExternalImports := map[string]map[string]bool{
		modulePath + "/cmd/helox": {},
		modulePath + "/internal/cli": {
			"github.com/spf13/cobra": true,
		},
		modulePath + "/internal/core/domain":           {},
		modulePath + "/internal/core/ports":            {},
		modulePath + "/internal/application":           {},
		modulePath + "/internal/policy":                {},
		modulePath + "/internal/testutil/fakeworkflow": {},
		modulePath + "/scripts/check":                  {},
	}
	forbiddenConcreteImports := map[string]map[string]bool{
		modulePath + "/internal/core/domain": {
			"database/sql": true, "net": true, "net/http": true, "os": true, "os/exec": true, "path/filepath": true,
		},
		modulePath + "/internal/testutil/fakeworkflow": {
			"database/sql": true, "net": true, "net/http": true, "os": true, "os/exec": true, "path/filepath": true,
		},
	}

	var findings []string
	for _, metadata := range packages {
		allowedProduct, constrained := allowedProductImports[metadata.ImportPath]
		if !constrained {
			continue
		}
		for _, imported := range metadata.Imports {
			if forbiddenConcreteImports[metadata.ImportPath][imported] {
				findings = append(findings, fmt.Sprintf("%s imports forbidden concrete package %s", metadata.ImportPath, imported))
			}
			if imported == modulePath || strings.HasPrefix(imported, modulePath+"/") {
				if !allowedProduct[imported] {
					findings = append(findings, fmt.Sprintf("%s imports disallowed product package %s", metadata.ImportPath, imported))
				}
				continue
			}
			if isExternalImport(imported) && !allowedExternalImports[metadata.ImportPath][imported] {
				findings = append(findings, fmt.Sprintf("%s imports unreviewed external package %s", metadata.ImportPath, imported))
			}
		}
	}
	sort.Strings(findings)
	return findings
}

func isExternalImport(importPath string) bool {
	first, _, _ := strings.Cut(importPath, "/")
	return strings.Contains(first, ".")
}

func isWithinRoot(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
