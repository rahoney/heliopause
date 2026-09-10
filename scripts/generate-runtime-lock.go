// Command generate-runtime-lock validates the canonical runtime lock and
// generates the product's immutable runtime identity package.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const outputPath = "internal/runtimeidentity/runtime_lock_gen.go"

var hex512 = regexp.MustCompile(`^[a-f0-9]{128}$`)
var hex256 = regexp.MustCompile(`^[a-f0-9]{64}$`)
var release = regexp.MustCompile(`^release-[0-9]{8}\.0$`)
var exactVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

type runtimeLock struct {
	SchemaVersion int `json:"schema_version"`
	GVisor        struct {
		Release          string `json:"release"`
		Commit           string `json:"commit"`
		SourceRepository string `json:"source_repository"`
		Patch            struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"patch"`
		UpstreamBinaries map[string]struct {
			URL    string `json:"url"`
			SHA512 string `json:"sha512"`
		} `json:"upstream_binaries"`
	} `json:"gvisor"`
	Docker struct {
		MinimumEngine string `json:"minimum_engine"`
		CIEngine      string `json:"ci_engine_version"`
		CIUbuntu      struct {
			DockerCE   string `json:"docker_ce_package"`
			Containerd string `json:"containerd_package"`
		} `json:"ci_ubuntu_24_04_amd64"`
	} `json:"docker"`
	NodeImage struct {
		Reference  string `json:"reference"`
		NPMVersion string `json:"npm_version"`
	} `json:"node_image"`
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
	generated := render(lock)
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

func fail(err error) { fmt.Fprintln(os.Stderr, "runtime lock:", err); os.Exit(1) }

func readLock(path string) (runtimeLock, error) {
	file, err := os.Open(path)
	if err != nil {
		return runtimeLock{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64*1024))
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
	return lock, nil
}

func validate(lock runtimeLock) error {
	if lock.SchemaVersion != 1 || !release.MatchString(lock.GVisor.Release) || len(lock.GVisor.Commit) != 40 || !hex512.MatchString(lock.Bazel.SHA512) || !exactVersion.MatchString(lock.Bazel.Version) || !strings.HasPrefix(lock.GVisor.SourceRepository, "https://") || !strings.HasPrefix(lock.Bazel.URL, "https://") {
		return errors.New("gVisor or Bazel identity is invalid")
	}
	if lock.GVisor.Patch.Path == "" || !strings.HasPrefix(lock.GVisor.Patch.Path, "tools/gvisor/") || !strings.HasSuffix(lock.GVisor.Patch.Path, ".patch") || strings.Contains(lock.GVisor.Patch.Path, "..") || !hex256.MatchString(lock.GVisor.Patch.SHA256) {
		return errors.New("gVisor patch identity is invalid")
	}
	if !exactVersion.MatchString(lock.Docker.MinimumEngine) || !exactVersion.MatchString(lock.Docker.CIEngine) || lock.Docker.CIUbuntu.DockerCE == "" || lock.Docker.CIUbuntu.Containerd == "" {
		return errors.New("docker identity is invalid")
	}
	if !strings.Contains(lock.NodeImage.Reference, "@sha256:") || !exactVersion.MatchString(lock.NodeImage.NPMVersion) {
		return errors.New("node identity is invalid")
	}
	if !strings.Contains(lock.PythonImage.Reference, "@sha256:") || !exactVersion.MatchString(lock.PythonImage.PythonVersion) || !exactVersion.MatchString(lock.PythonImage.PipVersion) || lock.PythonImage.Target.Interpreter == "" || lock.PythonImage.Target.ABI == "" || lock.PythonImage.Target.Platform == "" {
		return errors.New("python identity is invalid")
	}
	if len(lock.GVisor.UpstreamBinaries) != 2 {
		return errors.New("require exactly two gVisor architectures")
	}
	if len(lock.PythonSources) == 0 {
		return errors.New("python source profiles are missing")
	}
	seenSources := map[string]bool{}
	for _, source := range lock.PythonSources {
		if source.Name == "" || source.SourceID == "" || source.IndexHost == "" || !strings.HasPrefix(source.IndexURL, "https://") || len(source.DistributionHosts) == 0 || len(source.OwnedProjects) == 0 || seenSources[source.SourceID] {
			return errors.New("python source profile is invalid")
		}
		seenSources[source.SourceID] = true
	}
	for _, architecture := range []string{"x86_64", "aarch64"} {
		binary, ok := lock.GVisor.UpstreamBinaries[architecture]
		if !ok || !strings.HasPrefix(binary.URL, "https://") || !hex512.MatchString(binary.SHA512) {
			return errors.New("gVisor binary identity is invalid")
		}
	}
	return nil
}

func render(lock runtimeLock) []byte {
	architectures := make([]string, 0, len(lock.GVisor.UpstreamBinaries))
	for architecture := range lock.GVisor.UpstreamBinaries {
		architectures = append(architectures, architecture)
	}
	sort.Strings(architectures)
	var output strings.Builder
	output.WriteString("// Code generated by go run ./scripts/generate-runtime-lock.go; DO NOT EDIT.\n\npackage runtimeidentity\n\nconst (\n")
	for _, item := range []struct{ name, value string }{
		{"GVisorRelease", lock.GVisor.Release}, {"GVisorCommit", lock.GVisor.Commit}, {"GVisorSourceRepository", lock.GVisor.SourceRepository},
		{"GVisorPatchPath", lock.GVisor.Patch.Path}, {"GVisorPatchSHA256", lock.GVisor.Patch.SHA256},
		{"BazelVersion", lock.Bazel.Version}, {"BazelLinuxX8664SHA512", lock.Bazel.SHA512},
		{"DockerMinimumEngine", lock.Docker.MinimumEngine}, {"NodeImageReference", lock.NodeImage.Reference}, {"NodeNPMVersion", lock.NodeImage.NPMVersion},
		{"PythonImageReference", lock.PythonImage.Reference}, {"PythonVersion", lock.PythonImage.PythonVersion}, {"PipVersion", lock.PythonImage.PipVersion},
		{"PythonInterpreterTag", lock.PythonImage.Target.Interpreter}, {"PythonABITag", lock.PythonImage.Target.ABI}, {"PythonPlatformTag", lock.PythonImage.Target.Platform},
	} {
		fmt.Fprintf(&output, "\t%s = %q\n", item.name, item.value)
	}
	output.WriteString(")\n\ntype PythonSourceProfileLock struct { Name, SourceID, IndexURL, IndexHost string; DistributionHosts, OwnedProjects []string }\n\nvar PythonSourceProfiles = map[string]PythonSourceProfileLock{\n")
	for _, source := range lock.PythonSources {
		fmt.Fprintf(&output, "%q: {Name: %q, SourceID: %q, IndexURL: %q, IndexHost: %q, DistributionHosts: %#v, OwnedProjects: %#v},\n", source.Name, source.Name, source.SourceID, source.IndexURL, source.IndexHost, source.DistributionHosts, source.OwnedProjects)
	}
	output.WriteString("}\n\nvar upstreamRunscSHA512 = map[string]string{\n")
	for _, architecture := range architectures {
		goarch := map[string]string{"x86_64": "amd64", "aarch64": "arm64"}[architecture]
		fmt.Fprintf(&output, "\t%q: %q,\n", goarch, lock.GVisor.UpstreamBinaries[architecture].SHA512)
	}
	output.WriteString("}\n\n// UpstreamRunscSHA512 returns the official unpatched upstream release digest.\n// It is not an identity for the HAA-patched runtime.\nfunc UpstreamRunscSHA512(goarch string) (string, bool) { value, ok := upstreamRunscSHA512[goarch]; return value, ok }\n")
	formatted, err := format.Source([]byte(output.String()))
	if err != nil {
		panic("internal runtime lock generator error: " + err.Error())
	}
	return formatted
}
