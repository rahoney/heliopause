package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var exactRuntimeVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
var nodeRuntimeReference = regexp.MustCompile(`^node:([0-9]+\.[0-9]+\.[0-9]+)-slim@sha256:[a-f0-9]{64}$`)

// These are input owners, not a second set of selected versions. Generated
// product identities and executable image contents have separate checks.
func runtimeVersionOwners(root string, tools toolLock) (map[string]string, error) {
	body, err := os.ReadFile(filepath.Join(root, "scripts", "runtimes.lock.json"))
	if err != nil {
		return nil, err
	}
	var lock map[string]any
	if err := json.Unmarshal(body, &lock); err != nil {
		return nil, err
	}
	wantKeys := map[string]bool{"schema_version": true, "gvisor": true, "docker": true, "go_image": true, "rust_image": true, "node_image": true, "python_image": true, "python_sources": true, "bazel": true, "uv_image": true, "pnpm_tool": true}
	if len(lock) != len(wantKeys) || lock["schema_version"] != float64(3) {
		return nil, errors.New("runtime version owners: missing or unsupported lock component")
	}
	for key := range lock {
		if !wantKeys[key] {
			return nil, fmt.Errorf("runtime version owners: unowned lock component %s", key)
		}
	}
	owners := map[string]string{}
	for component, path := range map[string]string{
		"Go": "go_image.go_version", "Docker": "docker.ci_engine_version",
		"gVisor": "gvisor.release", "Bazel": "bazel.version", "Rust/Cargo": "rust_image.rust_version",
		"uv": "uv_image.uv_version", "npm": "node_image.npm_version", "Python": "python_image.python_version", "pip": "python_image.pip_version",
		"pnpm": "pnpm_tool.pnpm_version",
	} {
		value, err := runtimeOwnerString(lock, path)
		if err != nil {
			return nil, err
		}
		if component != "gVisor" && !exactRuntimeVersion.MatchString(value) {
			return nil, fmt.Errorf("runtime version owners: invalid selected version for %s", component)
		}
		owners[component] = value
	}
	if err := checkPNPMProofFile(root, lock); err != nil {
		return nil, err
	}
	node, err := runtimeOwnerString(lock, "node_image.reference")
	if err != nil {
		return nil, err
	}
	match := nodeRuntimeReference.FindStringSubmatch(node)
	if len(match) != 2 {
		return nil, errors.New("runtime version owners: invalid Node image")
	}
	owners["Node"] = match[1]
	containerd, err := runtimeOwnerString(lock, "docker.ci_ubuntu_24_04_amd64.containerd_package")
	if err != nil {
		return nil, err
	}
	owners["containerd"], _, _ = strings.Cut(containerd, "-")
	if !exactRuntimeVersion.MatchString(owners["containerd"]) {
		return nil, errors.New("runtime version owners: invalid containerd package")
	}
	for _, tool := range tools.Tools {
		component := map[string]string{"staticcheck": "Staticcheck", "gosec": "gosec", "govulncheck": "govulncheck", "gitleaks": "Gitleaks", "actionlint": "Actionlint"}[tool.Command]
		if component == "" || owners[component] != "" || tool.SetupGo != owners["Go"] {
			return nil, errors.New("runtime version owners: unknown/duplicate tool or mixed Go compiler")
		}
		owners[component] = tool.Version
	}
	return owners, nil
}

func checkPNPMProofFile(root string, lock map[string]any) error {
	name, err := runtimeOwnerString(lock, "pnpm_tool.provenance.bundle_path")
	if err != nil || name != "tools/pnpm/release-proof.json" {
		return errors.New("pnpm production proof owner is missing or changed")
	}
	expected, err := runtimeOwnerString(lock, "pnpm_tool.provenance.bundle_sha256")
	if err != nil {
		return err
	}
	path := filepath.Join(root, filepath.FromSlash(name))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 1<<20 {
		return errors.New("pnpm production proof is unavailable or excessive")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	body, readErr := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	closeErr := file.Close()
	sum := sha256.Sum256(body)
	if readErr != nil || closeErr != nil || int64(len(body)) != info.Size() || hex.EncodeToString(sum[:]) != expected {
		return errors.New("pnpm production proof differs from its runtime owner")
	}
	return nil
}

func runtimeOwnerString(lock map[string]any, path string) (string, error) {
	var current any = lock
	for _, key := range strings.Split(path, ".") {
		object, _ := current.(map[string]any)
		current = object[key]
	}
	value, ok := current.(string)
	if !ok || value == "" {
		return "", fmt.Errorf("runtime version owner %s is missing", path)
	}
	return value, nil
}

func checkRuntimeVersionOwners(root string, tools toolLock) error {
	owners, err := runtimeVersionOwners(root, tools)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(filepath.Join(root, "scripts", "version-support.lock.json"))
	if err != nil {
		return err
	}
	var reviewed supportRecordFile
	if err := json.Unmarshal(body, &reviewed); err != nil || reviewed.SchemaVersion != 1 {
		return errors.New("runtime version owners: invalid support record")
	}
	seen := map[string]bool{}
	for _, record := range reviewed.Records {
		version, exists := owners[record.Component]
		if strings.HasPrefix(record.Component, "PyTorch ") {
			profile := strings.TrimPrefix(record.Component, "PyTorch ")
			if profile == "CPU" {
				profile = "cpu"
			}
			base, suffix, ok := strings.Cut(record.PinnedVersion, "+")
			exists = (profile == "cpu" || profile == "cu126" || profile == "cu130" || profile == "cu132") && ok && suffix == profile && exactRuntimeVersion.MatchString(base)
			version = record.PinnedVersion
		}
		if !exists || seen[record.Component] || version != record.PinnedVersion {
			return fmt.Errorf("runtime version owners: unknown, duplicate or mismatched support component %s", record.Component)
		}
		seen[record.Component] = true
	}
	for component := range owners {
		if !seen[component] {
			return fmt.Errorf("runtime version owners: missing support record for %s", component)
		}
	}
	for _, profile := range []string{"CPU", "cu126", "cu130", "cu132"} {
		if !seen["PyTorch "+profile] {
			return fmt.Errorf("runtime version owners: missing PyTorch %s record", profile)
		}
	}
	return checkRuntimeLiteralOwners(root)
}

// Inspect active Go source, including newly added consumers. Historical test
// fixtures and the generated identity owner are deliberately distinct roles.
func checkRuntimeLiteralOwners(root string) error {
	image := regexp.MustCompile(`(?:^|[^A-Za-z0-9_])(node|python|golang|rust|ghcr\.io/astral-sh/uv):(?:[0-9]|latest)`)
	pnpmArchive := regexp.MustCompile(`https://github\.com/pnpm/pnpm/releases/(?:download/(?:v[0-9]|latest)|latest/)`)
	for _, directory := range []string{"cmd", "internal", "scripts"} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if relative == "internal/runtimeidentity/runtime_lock_gen.go" {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			var finding error
			ast.Inspect(file, func(node ast.Node) bool {
				literal, ok := node.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return true
				}
				value, err := strconv.Unquote(literal.Value)
				if err != nil {
					finding = err
					return false
				}
				if relative == "scripts/check/workflow.go" && retiredRuntimeImagePolicyToken(value) {
					return true
				}
				if image.MatchString(value) || pnpmArchive.MatchString(value) {
					finding = fmt.Errorf("runtime version owners: unowned image literal in %s", relative)
					return false
				}
				return true
			})
			return finding
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Parse actual Actions steps: comments and another correct setup step cannot
// conceal an old compiler in one job. No ambient or floating selector is allowed.
func validateGoWorkflowVersions(contents, selected string) []string {
	doc, err := parseWorkflow(contents)
	if err != nil {
		return []string{"cannot parse Go workflow bindings: " + err.Error()}
	}
	var findings []string
	count := 0
	for jobID, raw := range workflowMap(doc["jobs"]) {
		steps, _ := workflowMap(raw)["steps"].([]any)
		for _, rawStep := range steps {
			step := workflowMap(rawStep)
			uses, _ := step["uses"].(string)
			if !strings.HasPrefix(uses, "actions/setup-go@") {
				continue
			}
			count++
			with := workflowMap(step["with"])
			if !exactRuntimeVersion.MatchString(selected) || with["go-version"] != selected || with["check-latest"] != false || with["cache"] != false || len(with) != 3 {
				findings = append(findings, fmt.Sprintf("job %s Go setup differs from selected compiler %s", jobID, selected))
			}
		}
	}
	if count == 0 {
		findings = append(findings, "workflow has no owned Go setup")
	}
	return findings
}
