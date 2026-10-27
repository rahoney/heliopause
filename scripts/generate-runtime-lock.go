// Command generate-runtime-lock validates the canonical runtime lock and
// generates the product's immutable runtime identity package.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const outputPath = "internal/runtimeidentity/runtime_lock_gen.go"
const observerBuildPath = "tools/gvisor-observer/BUILD"

var hex512 = regexp.MustCompile(`^[a-f0-9]{128}$`)
var hex256 = regexp.MustCompile(`^[a-f0-9]{64}$`)
var release = regexp.MustCompile(`^release-[0-9]{8}\.0$`)
var exactVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
var observerCommitDefinePattern = regexp.MustCompile(`HAA_GVISOR_COMMIT=\\*["']([0-9a-fA-F]{40})\\*["']`)
var targetNamePattern = regexp.MustCompile(`\bname\s*=\s*["']([^"']+)["']`)
var haaTokenPattern = regexp.MustCompile(`\bHAA_GVISOR_COMMIT\b`)

type runtimeLock struct {
	lockSHA256    string
	SchemaVersion int `json:"schema_version"`
	GVisor        struct {
		Release          string `json:"release"`
		Commit           string `json:"commit"`
		SourceRepository string `json:"source_repository"`
		Patch            struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"patch"`
		Build struct {
			BazelModuleLockSHA256 string `json:"bazel_module_lock_sha256"`
			Builder               struct {
				ImageRepository string `json:"image_repository"`
				ImageTag        string `json:"image_tag"`
				ImageDigest     string `json:"image_digest"`
				Architecture    string `json:"architecture"`
			} `json:"builder"`
		} `json:"build"`
		RuntimeBundle struct {
			Architecture string `json:"architecture"`
			Members      []struct {
				Path   string `json:"path"`
				Size   int64  `json:"size"`
				SHA512 string `json:"sha512"`
			} `json:"members"`
		} `json:"runtime_bundle"`
	} `json:"gvisor"`
	Docker struct {
		MinimumEngine string `json:"minimum_engine"`
		CIEngine      string `json:"ci_engine_version"`
		CIUbuntu      struct {
			DockerCE   string `json:"docker_ce_package"`
			DockerCLI  string `json:"docker_ce_cli_package"`
			Containerd string `json:"containerd_package"`
		} `json:"ci_ubuntu_24_04_amd64"`
	} `json:"docker"`
	NodeImage struct {
		Reference  string `json:"reference"`
		NPMVersion string `json:"npm_version"`
	} `json:"node_image"`
	GoImage struct {
		Reference    string `json:"reference"`
		GoVersion    string `json:"go_version"`
		Architecture string `json:"architecture"`
	} `json:"go_image"`
	RustImage struct {
		Reference    string `json:"reference"`
		RustVersion  string `json:"rust_version"`
		Architecture string `json:"architecture"`
		Target       string `json:"target"`
	} `json:"rust_image"`
	PythonImage struct {
		Reference     string `json:"reference"`
		PythonVersion string `json:"python_version"`
		PipVersion    string `json:"pip_version"`
		Target        struct {
			Interpreter string `json:"interpreter"`
			ABI         string `json:"abi"`
			Platform    string `json:"platform"`
		} `json:"target"`
	} `json:"python_image"`
	UVImage struct {
		Reference    string `json:"reference"`
		Version      string `json:"uv_version"`
		Architecture string `json:"architecture"`
		Provenance   struct {
			SourceCommit string `json:"source_commit"`
			WorkflowRun  string `json:"workflow_run"`
			IndexSHA256  string `json:"index_sha256"`
			BundleSHA256 string `json:"bundle_sha256"`
			ConfigSHA256 string `json:"config_sha256"`
		} `json:"provenance"`
		Tool struct {
			Path   string `json:"path"`
			Size   int64  `json:"size"`
			SHA256 string `json:"sha256"`
		} `json:"tool"`
	} `json:"uv_image"`
	PNPMTool struct {
		Version      string `json:"pnpm_version"`
		Architecture string `json:"architecture"`
		Archive      struct {
			URL    string `json:"url"`
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"archive"`
		Provenance struct {
			SourceCommit string `json:"source_commit"`
			WorkflowRun  string `json:"workflow_run"`
			BundlePath   string `json:"bundle_path"`
			BundleSHA256 string `json:"bundle_sha256"`
		} `json:"provenance"`
		Tool struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"tool"`
		Notices struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"notices"`
	} `json:"pnpm_tool"`
	BunTool struct {
		Version      string `json:"bun_version"`
		Architecture string `json:"architecture"`
		RuntimeImage string `json:"runtime_image"`
		Archive      struct {
			URL    string `json:"url"`
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"archive"`
		Publisher struct {
			Fingerprint     string `json:"fingerprint"`
			ChecksumsPath   string `json:"checksums_path"`
			ChecksumsSHA256 string `json:"checksums_sha256"`
		} `json:"publisher"`
		SourceCommitObserved string `json:"source_commit_observed"`
		Tool                 struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"tool"`
		Notices struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"notices"`
	} `json:"bun_tool"`
	PipxTool struct {
		Version      string `json:"pipx_version"`
		Architecture string `json:"architecture"`
		RuntimeImage string `json:"runtime_image"`
		Inputs       []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"inputs"`
		Tool struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"tool"`
		Notices struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		} `json:"notices"`
	} `json:"pipx_tool"`
	PythonSources []struct {
		Name              string   `json:"name"`
		SourceID          string   `json:"source_id"`
		IndexURL          string   `json:"index_url"`
		IndexHost         string   `json:"index_host"`
		DistributionHosts []string `json:"distribution_hosts"`
		OwnedProjects     []string `json:"owned_projects"`
	} `json:"python_sources"`
	Bazel struct {
		Version string `json:"version"`
		URL     string `json:"linux_x86_64_url"`
		SHA512  string `json:"linux_x86_64_sha512"`
	} `json:"bazel"`
}

func main() {
	check := flag.Bool("check", false, "require generated output to be current")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(errors.New("no positional arguments are accepted"))
	}
	lock, err := readLock("scripts/runtimes.lock.json")
	if err != nil {
		fail(err)
	}
	observerBuild, err := os.ReadFile(observerBuildPath)
	if err != nil {
		fail(fmt.Errorf("read observer BUILD: %w", err))
	}
	if err := verifyObserverBuildCommit(observerBuild, lock.GVisor.Commit); err != nil {
		fail(fmt.Errorf("observer BUILD identity differs from canonical lock: %w", err))
	}
	qualification, err := readPyTorchVersions("scripts/version-support.lock.json")
	if err != nil {
		fail(err)
	}
	generated := render(lock, qualification)
	if *check {
		current, err := os.ReadFile(outputPath)
		if err != nil || !bytes.Equal(current, generated) {
			fail(errors.New("generated runtime identity is missing or differs; run go run ./scripts/generate-runtime-lock.go"))
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(outputPath, generated, 0o644); err != nil {
		fail(err)
	}
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func stripStarlarkComments(src []byte) []byte {
	out := make([]byte, 0, len(src))
	n := len(src)
	i := 0
	for i < n {
		b := src[i]
		if (b == '"' || b == '\'') && i+2 < n && src[i+1] == b && src[i+2] == b {
			quote := b
			out = append(out, quote, quote, quote)
			i += 3
			for i < n {
				if src[i] == '\\' {
					out = append(out, src[i])
					i++
					if i < n {
						out = append(out, src[i])
						i++
					}
					continue
				}
				if src[i] == quote && i+2 < n && src[i+1] == quote && src[i+2] == quote {
					out = append(out, quote, quote, quote)
					i += 3
					break
				}
				out = append(out, src[i])
				i++
			}
			continue
		}
		if b == '"' || b == '\'' {
			quote := b
			out = append(out, quote)
			i++
			for i < n {
				if src[i] == '\\' {
					out = append(out, src[i])
					i++
					if i < n {
						out = append(out, src[i])
						i++
					}
					continue
				}
				if src[i] == quote {
					out = append(out, quote)
					i++
					break
				}
				out = append(out, src[i])
				i++
			}
			continue
		}
		if b == '#' {
			for i < n && src[i] != '\n' {
				i++
			}
			if i < n && src[i] == '\n' {
				out = append(out, '\n')
				i++
			}
			continue
		}
		out = append(out, b)
		i++
	}
	return out
}

type parsedTarget struct {
	name string
	body string
}

func extractActiveTargets(cleanSrc []byte, ruleName string) ([]parsedTarget, error) {
	var targets []parsedTarget
	n := len(cleanSrc)
	i := 0
	targetPrefix := []byte(ruleName)

	for i < n {
		b := cleanSrc[i]
		if (b == '"' || b == '\'') && i+2 < n && cleanSrc[i+1] == b && cleanSrc[i+2] == b {
			quote := b
			i += 3
			for i < n {
				if cleanSrc[i] == '\\' {
					i += 2
					continue
				}
				if cleanSrc[i] == quote && i+2 < n && cleanSrc[i+1] == quote && cleanSrc[i+2] == quote {
					i += 3
					break
				}
				i++
			}
			continue
		}
		if b == '"' || b == '\'' {
			quote := b
			i++
			for i < n {
				if cleanSrc[i] == '\\' {
					i += 2
					continue
				}
				if cleanSrc[i] == quote {
					i++
					break
				}
				i++
			}
			continue
		}

		if bytes.HasPrefix(cleanSrc[i:], targetPrefix) {
			if i > 0 && isIdentChar(cleanSrc[i-1]) {
				i++
				continue
			}
			pos := i + len(targetPrefix)
			for pos < n && isWhitespace(cleanSrc[pos]) {
				pos++
			}
			if pos < n && cleanSrc[pos] == '(' {
				openParen := pos
				pos++
				parenDepth := 1
				bodyStart := pos

				for pos < n && parenDepth > 0 {
					cb := cleanSrc[pos]
					if (cb == '"' || cb == '\'') && pos+2 < n && cleanSrc[pos+1] == cb && cleanSrc[pos+2] == cb {
						quote := cb
						pos += 3
						for pos < n {
							if cleanSrc[pos] == '\\' {
								pos += 2
								continue
							}
							if cleanSrc[pos] == quote && pos+2 < n && cleanSrc[pos+1] == quote && cleanSrc[pos+2] == quote {
								pos += 3
								break
							}
							pos++
						}
						continue
					}
					if cb == '"' || cb == '\'' {
						quote := cb
						pos++
						for pos < n {
							if cleanSrc[pos] == '\\' {
								pos += 2
								continue
							}
							if cleanSrc[pos] == quote {
								pos++
								break
							}
							pos++
						}
						continue
					}
					if cb == '(' {
						parenDepth++
					} else if cb == ')' {
						parenDepth--
					}
					if parenDepth == 0 {
						break
					}
					pos++
				}
				if parenDepth != 0 {
					return nil, fmt.Errorf("unclosed %s parenthesis starting at offset %d", ruleName, openParen)
				}
				body := string(cleanSrc[bodyStart:pos])
				nameMatch := targetNamePattern.FindStringSubmatch(body)
				var name string
				if len(nameMatch) > 1 {
					name = nameMatch[1]
				}
				targets = append(targets, parsedTarget{
					name: name,
					body: body,
				})
				i = pos + 1
				continue
			}
		}
		i++
	}
	return targets, nil
}

func verifyObserverBuildCommit(buildContent []byte, expectedCommit string) error {
	if len(expectedCommit) != 40 {
		return fmt.Errorf("expected commit must be 40 hex characters: %q", expectedCommit)
	}

	cleanSrc := stripStarlarkComments(buildContent)

	targets, err := extractActiveTargets(cleanSrc, "cc_binary")
	if err != nil {
		return fmt.Errorf("parse observer BUILD: %w", err)
	}

	requiredTargets := []string{
		"haa_gvisor_observer",
		"haa_gvisor_observer_latch_test",
	}

	targetCounts := make(map[string]int)
	targetBodies := make(map[string]string)
	for _, t := range targets {
		if t.name == "" {
			return fmt.Errorf("active cc_binary target is missing a name attribute")
		}
		targetCounts[t.name]++
		if targetCounts[t.name] > 1 {
			return fmt.Errorf("duplicate active target %q found in observer BUILD", t.name)
		}
		targetBodies[t.name] = t.body
	}

	for _, target := range requiredTargets {
		body, exists := targetBodies[target]
		if !exists {
			return fmt.Errorf("required target %q not found in active observer BUILD", target)
		}
		haaOccurrences := haaTokenPattern.FindAllStringIndex(body, -1)
		if len(haaOccurrences) == 0 {
			return fmt.Errorf("target %q is missing HAA_GVISOR_COMMIT define in observer BUILD", target)
		}
		if len(haaOccurrences) > 1 {
			return fmt.Errorf("target %q contains %d HAA_GVISOR_COMMIT definitions, expected exactly 1", target, len(haaOccurrences))
		}

		commitMatch := observerCommitDefinePattern.FindStringSubmatch(body)
		if commitMatch == nil {
			return fmt.Errorf("target %q has malformed or non-40-hex HAA_GVISOR_COMMIT define", target)
		}
		actualCommit := commitMatch[1]
		if actualCommit != expectedCommit {
			return fmt.Errorf("target %q HAA_GVISOR_COMMIT %q does not match canonical commit %q", target, actualCommit, expectedCommit)
		}
	}

	allActiveHaaOccurrences := haaTokenPattern.FindAllStringIndex(string(cleanSrc), -1)
	if len(allActiveHaaOccurrences) != len(requiredTargets) {
		return fmt.Errorf("expected %d active HAA_GVISOR_COMMIT defines in observer BUILD, found %d", len(requiredTargets), len(allActiveHaaOccurrences))
	}

	return nil
}

func fail(err error) { fmt.Fprintln(os.Stderr, "runtime lock:", err); os.Exit(1) }

func readLock(path string) (runtimeLock, error) {
	file, err := os.Open(path)
	if err != nil {
		return runtimeLock{}, err
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, 64*1024+1))
	if err != nil {
		return runtimeLock{}, err
	}
	if len(body) > 64*1024 {
		return runtimeLock{}, errors.New("canonical lock exceeds size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var lock runtimeLock
	if err := decoder.Decode(&lock); err != nil {
		return runtimeLock{}, fmt.Errorf("decode canonical lock: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return runtimeLock{}, errors.New("canonical lock has trailing content")
	}
	if err := validate(lock); err != nil {
		return runtimeLock{}, err
	}
	sum := sha256.Sum256(body)
	lock.lockSHA256 = hex.EncodeToString(sum[:])
	return lock, nil
}

func validate(lock runtimeLock) error {
	if lock.SchemaVersion != 5 || !release.MatchString(lock.GVisor.Release) || len(lock.GVisor.Commit) != 40 || !hex512.MatchString(lock.Bazel.SHA512) || !exactVersion.MatchString(lock.Bazel.Version) || !strings.HasPrefix(lock.GVisor.SourceRepository, "https://") || !strings.HasPrefix(lock.Bazel.URL, "https://") {
		return errors.New("gVisor or Bazel identity is invalid")
	}
	if lock.GVisor.Patch.Path == "" || !strings.HasPrefix(lock.GVisor.Patch.Path, "tools/gvisor/") || !strings.HasSuffix(lock.GVisor.Patch.Path, ".patch") || strings.Contains(lock.GVisor.Patch.Path, "..") || !hex256.MatchString(lock.GVisor.Patch.SHA256) {
		return errors.New("gVisor patch identity is invalid")
	}
	if !hex256.MatchString(lock.GVisor.Build.BazelModuleLockSHA256) {
		return errors.New("gVisor Bazel module lock identity is invalid")
	}
	builder := lock.GVisor.Build.Builder
	if builder.ImageRepository != "us-central1-docker.pkg.dev/gvisor-presubmit/gvisor-presubmit-images/default_x86_64" ||
		!regexp.MustCompile(`^[a-f0-9]{16}$`).MatchString(builder.ImageTag) ||
		!regexp.MustCompile(`^sha256:[a-f0-9]{64}$`).MatchString(builder.ImageDigest) ||
		builder.Architecture != "amd64" {
		return errors.New("gVisor builder image identity is invalid")
	}
	if !exactVersion.MatchString(lock.Docker.MinimumEngine) || !exactVersion.MatchString(lock.Docker.CIEngine) || lock.Docker.MinimumEngine != lock.Docker.CIEngine || lock.Docker.CIUbuntu.DockerCE == "" || lock.Docker.CIUbuntu.DockerCLI == "" || lock.Docker.CIUbuntu.Containerd == "" {
		return errors.New("docker identity is invalid")
	}
	if !regexp.MustCompile(`^node:[0-9]+\.[0-9]+\.[0-9]+-slim@sha256:[a-f0-9]{64}$`).MatchString(lock.NodeImage.Reference) || !exactVersion.MatchString(lock.NodeImage.NPMVersion) {
		return errors.New("node identity is invalid")
	}
	goReference := regexp.MustCompile(`^golang:[0-9]+\.[0-9]+\.[0-9]+-bookworm@sha256:[a-f0-9]{64}$`)
	if !goReference.MatchString(lock.GoImage.Reference) || !exactVersion.MatchString(lock.GoImage.GoVersion) || !strings.HasPrefix(lock.GoImage.Reference, "golang:"+lock.GoImage.GoVersion+"-bookworm@") || lock.GoImage.Architecture != "amd64" {
		return errors.New("go image identity is invalid")
	}
	rustReference := regexp.MustCompile(`^rust:[0-9]+\.[0-9]+\.[0-9]+-bookworm@sha256:[a-f0-9]{64}$`)
	if !rustReference.MatchString(lock.RustImage.Reference) || !exactVersion.MatchString(lock.RustImage.RustVersion) || !strings.HasPrefix(lock.RustImage.Reference, "rust:"+lock.RustImage.RustVersion+"-bookworm@") || lock.RustImage.Architecture != "amd64" || lock.RustImage.Target != "x86_64-unknown-linux-gnu" {
		return errors.New("rust image identity is invalid")
	}
	pythonReference := regexp.MustCompile(`^python:[0-9]+\.[0-9]+\.[0-9]+-slim-bookworm@sha256:[a-f0-9]{64}$`)
	pythonParts := strings.Split(lock.PythonImage.PythonVersion, ".")
	if !pythonReference.MatchString(lock.PythonImage.Reference) || !exactVersion.MatchString(lock.PythonImage.PythonVersion) || !strings.HasPrefix(lock.PythonImage.Reference, "python:"+lock.PythonImage.PythonVersion+"-slim-bookworm@") || !exactVersion.MatchString(lock.PythonImage.PipVersion) {
		return errors.New("python identity is invalid")
	}
	pythonTag := "cp" + pythonParts[0] + pythonParts[1]
	if lock.PythonImage.Target.Interpreter != pythonTag || lock.PythonImage.Target.ABI != pythonTag || lock.PythonImage.Target.Platform != "manylinux_2_36_x86_64" {
		return errors.New("python interpreter, ABI or platform does not match the image target")
	}
	uv := lock.UVImage
	if !exactVersion.MatchString(uv.Version) || uv.Architecture != "amd64" || !regexp.MustCompile(`^ghcr.io/astral-sh/uv:`+regexp.QuoteMeta(uv.Version)+`@sha256:[a-f0-9]{64}$`).MatchString(uv.Reference) || !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(uv.Provenance.SourceCommit) || !regexp.MustCompile(`^[1-9][0-9]{0,19}$`).MatchString(uv.Provenance.WorkflowRun) || !hex256.MatchString(uv.Provenance.IndexSHA256) || !hex256.MatchString(uv.Provenance.BundleSHA256) || !hex256.MatchString(uv.Provenance.ConfigSHA256) || uv.Tool.Path != "/uv" || uv.Tool.Size <= 0 || uv.Tool.Size > 64<<20 || !hex256.MatchString(uv.Tool.SHA256) {
		return errors.New("uv image, provenance or native tool identity is invalid")
	}
	pnpm := lock.PNPMTool
	if !exactVersion.MatchString(pnpm.Version) || pnpm.Architecture != "amd64" || pnpm.Archive.URL != "https://github.com/pnpm/pnpm/releases/download/v"+pnpm.Version+"/pnpm-linux-x64.tar.gz" || !hex256.MatchString(pnpm.Archive.SHA256) || pnpm.Archive.Size <= 0 || pnpm.Archive.Size > 64<<20 || !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(pnpm.Provenance.SourceCommit) || !regexp.MustCompile(`^[1-9][0-9]{0,19}$`).MatchString(pnpm.Provenance.WorkflowRun) || pnpm.Provenance.BundlePath != "tools/pnpm/release-proof.json" || !hex256.MatchString(pnpm.Provenance.BundleSHA256) || pnpm.Tool.Path != "pnpm" || !hex256.MatchString(pnpm.Tool.SHA256) || pnpm.Tool.Size <= 0 || pnpm.Tool.Size > 64<<20 || pnpm.Notices.Path != "tools/pnpm/NOTICE" || !hex256.MatchString(pnpm.Notices.SHA256) || pnpm.Notices.Size <= 0 || pnpm.Notices.Size > 64<<10 {
		return errors.New("pnpm archive, provenance or native tool identity is invalid")
	}
	bun := lock.BunTool
	if !exactVersion.MatchString(bun.Version) || bun.Architecture != "amd64" || bun.RuntimeImage != "node_image" || bun.Archive.URL != "https://github.com/oven-sh/bun/releases/download/bun-v"+bun.Version+"/bun-linux-x64.zip" || !hex256.MatchString(bun.Archive.SHA256) || bun.Archive.Size <= 0 || bun.Archive.Size > 64<<20 || bun.Publisher.Fingerprint != "F3DCC08A8572C0749B3E18888EAB4D40A7B22B59" || bun.Publisher.ChecksumsPath != "tools/bun/SHASUMS256.txt.asc" || !hex256.MatchString(bun.Publisher.ChecksumsSHA256) || !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(bun.SourceCommitObserved) || bun.Tool.Path != "bun" || !hex256.MatchString(bun.Tool.SHA256) || bun.Tool.Size <= 0 || bun.Tool.Size > 128<<20 || bun.Notices.Path != "tools/bun/NOTICE" || !hex256.MatchString(bun.Notices.SHA256) || bun.Notices.Size <= 0 || bun.Notices.Size > 64<<10 {
		return errors.New("bun publisher/archive/native data owner is invalid")
	}
	pipx := lock.PipxTool
	if !exactVersion.MatchString(pipx.Version) || pipx.Architecture != "amd64" || pipx.RuntimeImage != "python_image" || len(pipx.Inputs) != 27 || pipx.Tool.Path != "pipx-runtime.zip" || !hex256.MatchString(pipx.Tool.SHA256) || pipx.Tool.Size <= 0 || pipx.Tool.Size > 8<<20 || pipx.Notices.Path != "tools/pipx/NOTICE" || !hex256.MatchString(pipx.Notices.SHA256) || pipx.Notices.Size <= 0 || pipx.Notices.Size > 64<<10 {
		return errors.New("pipx immutable supply owner is invalid")
	}
	pipxInputs := map[string]bool{}
	for _, input := range pipx.Inputs {
		if !hex256.MatchString(input.SHA256) || len(input.Path) > 256 || !regexp.MustCompile(`^(argcomplete|click|filelock|packaging|pipx|platformdirs|userpath)/(identity\.json|metadata\.json|provenance\.json|[a-z]+-[0-9]+(?:\.[0-9]+)+-py3-none-any\.whl)$`).MatchString(input.Path) || pipxInputs[input.Path] {
			return errors.New("pipx supply input identity is invalid")
		}
		pipxInputs[input.Path] = true
	}
	if lock.GVisor.RuntimeBundle.Architecture != "amd64" || len(lock.GVisor.RuntimeBundle.Members) != 6 {
		return errors.New("gVisor runtime bundle identity is invalid")
	}
	expectedBundlePaths := []string{
		"containerd-shim-runsc-v1",
		"gvisor-bin/checkpointgofer",
		"gvisor-bin/gvisor-sentry-prewarmer",
		"gvisor-bin/gvisor_sentry",
		"gvisor-bin/runsc-metric-server",
		"runsc",
	}
	for index, member := range lock.GVisor.RuntimeBundle.Members {
		if member.Path != expectedBundlePaths[index] || member.Size <= 0 || !hex512.MatchString(member.SHA512) {
			return errors.New("gVisor runtime bundle member is invalid")
		}
	}
	if len(lock.PythonSources) != 4 {
		return errors.New("require exactly four active python source profiles")
	}
	seenSources := map[string]bool{}
	for _, source := range lock.PythonSources {
		if source.Name == "" || source.SourceID == "" || source.IndexHost == "" || !strings.HasPrefix(source.IndexURL, "https://") || len(source.DistributionHosts) == 0 || len(source.OwnedProjects) == 0 || seenSources[source.SourceID] {
			return errors.New("python source profile is invalid")
		}
		seenSources[source.SourceID] = true
	}
	for _, name := range []string{"pytorch:cpu", "pytorch:cu126", "pytorch:cu130", "pytorch:cu132"} {
		if _, ok := seenSources["pytorch-"+strings.TrimPrefix(name, "pytorch:")]; !ok {
			return errors.New("required python source profile is missing")
		}
	}
	return nil
}

func readPyTorchVersions(path string) (map[string]string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var records struct {
		SchemaVersion int `json:"schema_version"`
		Records       []struct {
			Component string `json:"component"`
			Version   string `json:"pinned_version"`
		} `json:"records"`
	}
	if len(body) > 64*1024 || json.Unmarshal(body, &records) != nil || records.SchemaVersion != 1 {
		return nil, errors.New("invalid PyTorch qualification version owners")
	}
	versions := map[string]string{}
	for _, record := range records.Records {
		if !strings.HasPrefix(record.Component, "PyTorch ") {
			continue
		}
		profile := strings.TrimPrefix(record.Component, "PyTorch ")
		if profile == "CPU" {
			profile = "cpu"
		}
		base, suffix, ok := strings.Cut(record.Version, "+")
		if (profile != "cpu" && profile != "cu126" && profile != "cu130" && profile != "cu132") || !ok || suffix != profile || !exactVersion.MatchString(base) || versions[profile] != "" {
			return nil, errors.New("unknown, duplicate or invalid PyTorch qualification version")
		}
		versions[profile] = record.Version
	}
	if len(versions) != 4 {
		return nil, errors.New("missing PyTorch qualification version owner")
	}
	return versions, nil
}

func render(lock runtimeLock, qualification map[string]string) []byte {
	var output strings.Builder
	output.WriteString("// Code generated by go run ./scripts/generate-runtime-lock.go; DO NOT EDIT.\n\npackage runtimeidentity\n\nconst (\n")
	for _, item := range []struct{ name, value string }{
		{"RuntimeLockSHA256", lock.lockSHA256},
		{"GVisorRelease", lock.GVisor.Release}, {"GVisorCommit", lock.GVisor.Commit}, {"GVisorSourceRepository", lock.GVisor.SourceRepository},
		{"GVisorPatchPath", lock.GVisor.Patch.Path}, {"GVisorPatchSHA256", lock.GVisor.Patch.SHA256}, {"GVisorBazelModuleLockSHA256", lock.GVisor.Build.BazelModuleLockSHA256},
		{"GVisorBuilderImageRepository", lock.GVisor.Build.Builder.ImageRepository}, {"GVisorBuilderImageTag", lock.GVisor.Build.Builder.ImageTag}, {"GVisorBuilderImageDigest", lock.GVisor.Build.Builder.ImageDigest}, {"GVisorBuilderArchitecture", lock.GVisor.Build.Builder.Architecture},
		{"BazelVersion", lock.Bazel.Version}, {"BazelLinuxX8664SHA512", lock.Bazel.SHA512},
		{"DockerMinimumEngine", lock.Docker.MinimumEngine}, {"DockerCIEngine", lock.Docker.CIEngine}, {"DockerCEPackage", lock.Docker.CIUbuntu.DockerCE}, {"DockerCECLIPackage", lock.Docker.CIUbuntu.DockerCLI}, {"ContainerdPackage", lock.Docker.CIUbuntu.Containerd}, {"NodeImageReference", lock.NodeImage.Reference}, {"NodeNPMVersion", lock.NodeImage.NPMVersion},
		{"UVImageReference", lock.UVImage.Reference}, {"UVVersion", lock.UVImage.Version}, {"UVArchitecture", lock.UVImage.Architecture},
		{"UVSourceCommit", lock.UVImage.Provenance.SourceCommit}, {"UVWorkflowRun", lock.UVImage.Provenance.WorkflowRun}, {"UVIndexSHA256", lock.UVImage.Provenance.IndexSHA256}, {"UVBundleSHA256", lock.UVImage.Provenance.BundleSHA256}, {"UVConfigSHA256", lock.UVImage.Provenance.ConfigSHA256}, {"UVToolPath", lock.UVImage.Tool.Path}, {"UVToolSHA256", lock.UVImage.Tool.SHA256},
		{"PNPMVersion", lock.PNPMTool.Version}, {"PNPMArchitecture", lock.PNPMTool.Architecture}, {"PNPMArchiveURL", lock.PNPMTool.Archive.URL}, {"PNPMArchiveSHA256", lock.PNPMTool.Archive.SHA256},
		{"BunVersion", lock.BunTool.Version}, {"BunArchitecture", lock.BunTool.Architecture}, {"BunRuntimeImage", lock.BunTool.RuntimeImage}, {"BunArchiveURL", lock.BunTool.Archive.URL}, {"BunArchiveSHA256", lock.BunTool.Archive.SHA256}, {"BunPublisherFingerprint", lock.BunTool.Publisher.Fingerprint}, {"BunChecksumsPath", lock.BunTool.Publisher.ChecksumsPath}, {"BunChecksumsSHA256", lock.BunTool.Publisher.ChecksumsSHA256}, {"BunSourceCommitObserved", lock.BunTool.SourceCommitObserved}, {"BunToolPath", lock.BunTool.Tool.Path}, {"BunToolSHA256", lock.BunTool.Tool.SHA256}, {"BunNoticesPath", lock.BunTool.Notices.Path}, {"BunNoticesSHA256", lock.BunTool.Notices.SHA256},
		{"PNPMSourceCommit", lock.PNPMTool.Provenance.SourceCommit}, {"PNPMWorkflowRun", lock.PNPMTool.Provenance.WorkflowRun}, {"PNPMBundlePath", lock.PNPMTool.Provenance.BundlePath}, {"PNPMBundleSHA256", lock.PNPMTool.Provenance.BundleSHA256}, {"PNPMToolPath", lock.PNPMTool.Tool.Path}, {"PNPMToolSHA256", lock.PNPMTool.Tool.SHA256}, {"PNPMNoticesPath", lock.PNPMTool.Notices.Path}, {"PNPMNoticesSHA256", lock.PNPMTool.Notices.SHA256},
		{"PipxVersion", lock.PipxTool.Version}, {"PipxArchitecture", lock.PipxTool.Architecture}, {"PipxRuntimeImage", lock.PipxTool.RuntimeImage}, {"PipxToolPath", lock.PipxTool.Tool.Path}, {"PipxToolSHA256", lock.PipxTool.Tool.SHA256}, {"PipxNoticesPath", lock.PipxTool.Notices.Path}, {"PipxNoticesSHA256", lock.PipxTool.Notices.SHA256},
		{"PythonImageReference", lock.PythonImage.Reference}, {"PythonVersion", lock.PythonImage.PythonVersion}, {"PipVersion", lock.PythonImage.PipVersion},
		{"GoImageReference", lock.GoImage.Reference}, {"GoVersion", lock.GoImage.GoVersion}, {"GoArchitecture", lock.GoImage.Architecture},
		{"RustImageReference", lock.RustImage.Reference}, {"RustVersion", lock.RustImage.RustVersion}, {"RustArchitecture", lock.RustImage.Architecture}, {"RustTarget", lock.RustImage.Target},
		{"PythonInterpreterTag", lock.PythonImage.Target.Interpreter}, {"PythonABITag", lock.PythonImage.Target.ABI}, {"PythonPlatformTag", lock.PythonImage.Target.Platform},
	} {
		fmt.Fprintf(&output, "\t%s = %q\n", item.name, item.value)
	}
	for _, item := range []struct{ name, profile string }{
		{"PyTorchCPUVersion", "cpu"}, {"PyTorchCU126Version", "cu126"}, {"PyTorchCU130Version", "cu130"}, {"PyTorchCU132Version", "cu132"},
	} {
		fmt.Fprintf(&output, "\t%s = %q\n", item.name, qualification[item.profile])
	}
	fmt.Fprintf(&output, "\tUVToolSize = %d\n", lock.UVImage.Tool.Size)
	fmt.Fprintf(&output, "\tPNPMArchiveSize = %d\n\tPNPMToolSize = %d\n", lock.PNPMTool.Archive.Size, lock.PNPMTool.Tool.Size)
	fmt.Fprintf(&output, "\tPNPMNoticesSize = %d\n", lock.PNPMTool.Notices.Size)
	fmt.Fprintf(&output, "\tBunArchiveSize = %d\n\tBunToolSize = %d\n\tBunNoticesSize = %d\n", lock.BunTool.Archive.Size, lock.BunTool.Tool.Size, lock.BunTool.Notices.Size)
	fmt.Fprintf(&output, "\tPipxToolSize = %d\n\tPipxNoticesSize = %d\n", lock.PipxTool.Tool.Size, lock.PipxTool.Notices.Size)
	output.WriteString(")\n\ntype PythonSourceProfileLock struct { Name, SourceID, IndexURL, IndexHost string; DistributionHosts, OwnedProjects []string }\n\nvar PythonSourceProfiles = map[string]PythonSourceProfileLock{\n")
	for _, source := range lock.PythonSources {
		fmt.Fprintf(&output, "%q: {Name: %q, SourceID: %q, IndexURL: %q, IndexHost: %q, DistributionHosts: %#v, OwnedProjects: %#v},\n", source.Name, source.Name, source.SourceID, source.IndexURL, source.IndexHost, source.DistributionHosts, source.OwnedProjects)
	}
	output.WriteString("}\n\ntype GVisorBundleMemberLock struct { Path string; Size int64; SHA512 string }\n\nvar GVisorRuntimeBundleMembers = []GVisorBundleMemberLock{\n")
	for _, member := range lock.GVisor.RuntimeBundle.Members {
		fmt.Fprintf(&output, "\t{Path: %q, Size: %d, SHA512: %q},\n", member.Path, member.Size, member.SHA512)
	}
	output.WriteString("}\n")
	formatted, err := format.Source([]byte(output.String()))
	if err != nil {
		panic("internal runtime lock generator error: " + err.Error())
	}
	return formatted
}
