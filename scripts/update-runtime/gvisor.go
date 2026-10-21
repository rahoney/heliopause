package main

import (
	"bytes"
	"crypto/sha512"
	"debug/buildinfo"
	"debug/elf"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var bundlePaths = []string{"containerd-shim-runsc-v1", "gvisor-bin/checkpointgofer", "gvisor-bin/gvisor-sentry-prewarmer", "gvisor-bin/gvisor_sentry", "gvisor-bin/runsc-metric-server", "runsc"}

type gvisorPreparation struct {
	Schema   int    `json:"schema_version"`
	Commit   string `json:"gvisor_commit"`
	Patch    string `json:"haa_patch_sha256"`
	Go       string `json:"selected_go"`
	Minimum  string `json:"upstream_go_minimum"`
	SDKPatch string `json:"sdk_patch_sha256"`
	Module   string `json:"prepared_go_mod_sha256"`
}

type gvisorCaptureReceipt struct {
	Schema             int    `json:"schema"`
	Root               string `json:"root"`
	Built              string `json:"built"`
	Module             string `json:"module_lock"`
	Preparation        string `json:"source_manifest"`
	RuntimeInputSHA256 string `json:"runtime_input_sha256"`
	OutputSHA256       string `json:"output_sha256"`
}
type gvisorCaptureInput struct {
	Lock    []byte               `json:"runtime_lock"`
	Receipt gvisorCaptureReceipt `json:"receipt"`
}

// Capture observes new output identities into an external candidate lock. It
// cannot authorize a release, rewrite the checkout or bypass strict verification.
func captureGvisor(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("capture-gvisor", flag.ContinueOnError)
	fs.SetOutput(out)
	rootFlag := fs.String("root", ".", "source repository containing canonical runtime lock")
	built := fs.String("built-dir", "", "absolute six-member release directory")
	module := fs.String("module-lock", "", "absolute produced MODULE.bazel.lock")
	preparation := fs.String("source-manifest", "", "absolute shared source-preparation manifest")
	output := fs.String("output", "", "absolute new external candidate runtime-lock file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected capture arguments")
	}
	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	for _, path := range []string{*built, *module, *preparation, *output} {
		if !filepath.IsAbs(path) {
			return errors.New("capture paths must be explicit absolute paths")
		}
	}
	builtRoot, err := filepath.EvalSymlinks(*built)
	if err != nil {
		return err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(*output))
	if err != nil {
		return err
	}
	destination := filepath.Join(parent, filepath.Base(*output))
	if within(root, destination) {
		return errors.New("capture output must stay outside the source checkout")
	}
	if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		return errors.New("capture destination preexists or is unavailable; preserve it")
	}
	if _, err := os.Lstat(destination + ".receipt.json"); !errors.Is(err, os.ErrNotExist) {
		return errors.New("capture receipt destination preexists; preserve it")
	}
	lockBytes, _, err := readRegular(root, runtimeFile)
	if err != nil {
		return err
	}
	if _, err := jsonSpans(lockBytes); err != nil {
		return err
	}
	var lock struct {
		Schema int `json:"schema_version"`
		GVisor struct {
			Commit string `json:"commit"`
			Patch  struct {
				Path   string `json:"path"`
				SHA256 string `json:"sha256"`
			} `json:"patch"`
			Bundle struct {
				Architecture string `json:"architecture"`
				Members      []struct {
					Path string `json:"path"`
				} `json:"members"`
			} `json:"runtime_bundle"`
		} `json:"gvisor"`
	}
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		return err
	}
	if lock.Schema != 2 || lock.GVisor.Bundle.Architecture != "amd64" || len(lock.GVisor.Bundle.Members) != len(bundlePaths) {
		return errors.New("unknown runtime bundle capture schema")
	}
	selected, err := selectedVersion(root, "go")
	if err != nil {
		return err
	}
	var source gvisorPreparation
	if err := loadJSON(*preparation, &source); err != nil {
		return err
	}
	if source.Schema != 1 || source.Commit != lock.GVisor.Commit || source.Patch != lock.GVisor.Patch.SHA256 || source.Go != selected || !versionPattern.MatchString(source.Minimum) || !digestPattern.MatchString(source.SDKPatch) || !digestPattern.MatchString(source.Module) {
		return errors.New("capture source/SDK/patch provenance does not match the selected owner")
	}
	if !strings.HasSuffix(*preparation, ".haa-source.json") {
		return errors.New("unknown source-preparation layout")
	}
	sourceDir := strings.TrimSuffix(*preparation, ".haa-source.json")
	preparedModule, _, err := readRegular(sourceDir, "go.mod")
	if err != nil {
		return err
	}
	if hash(preparedModule) != source.Module {
		return errors.New("prepared gVisor SDK selector changed")
	}
	if len(regexp.MustCompile(`(?m)^go [0-9]+\.[0-9]+\.[0-9]+$`).FindAll(preparedModule, -1)) != 1 || !regexp.MustCompile(`(?m)^go `+regexp.QuoteMeta(selected)+`$`).Match(preparedModule) || regexp.MustCompile(`(?m)^[\t ]*toolchain[\t ]`).Match(preparedModule) {
		return errors.New("prepared SDK selector does not match the selected Go owner")
	}
	sdkPatch, _, err := readRegular(sourceDir, "tools/go_types_memoize.patch")
	if err != nil {
		return err
	}
	originalSDKPatch, err := command(sourceDir, nil, "git", "show", source.Commit+":tools/go_types_memoize.patch")
	if err != nil {
		return err
	}
	if hash(sdkPatch) != source.SDKPatch || hash(originalSDKPatch) != source.SDKPatch {
		return errors.New("SDK patch differs from exact upstream source")
	}
	head, err := command(sourceDir, nil, "git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(head)) != source.Commit {
		return errors.New("prepared source HEAD changed")
	}
	if err := verifyPreparedGvisor(root, sourceDir, parent, source, lock.GVisor.Patch.Path, preparedModule); err != nil {
		return err
	}
	moduleBytes, _, err := readRegular(filepath.Dir(*module), filepath.Base(*module))
	if err != nil {
		return err
	}
	if len(moduleBytes) == 0 || len(moduleBytes) > 32<<20 || !json.Valid(moduleBytes) {
		return errors.New("produced Bazel module lock is invalid")
	}
	replacements := map[string]any{"/gvisor/build/bazel_module_lock_sha256": hash(moduleBytes)}
	for i, path := range bundlePaths {
		if lock.GVisor.Bundle.Members[i].Path != path {
			return errors.New("runtime bundle member order/path mismatch")
		}
		location := filepath.Join(builtRoot, path)
		resolved, err := filepath.EvalSymlinks(location)
		if err != nil {
			return err
		}
		if resolved != location {
			return errors.New("bundle member alias refused")
		}
		info, err := os.Lstat(location)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 1<<30 {
			return errors.New("invalid bundle member kind/size")
		}
		if path != "gvisor-bin/gvisor-sentry-prewarmer" {
			metadata, err := buildinfo.ReadFile(location)
			if err != nil || metadata.GoVersion != "go"+selected {
				return fmt.Errorf("actual bundle compiler does not match selected Go: %s", path)
			}
			binary, err := elf.Open(location)
			if err != nil {
				return err
			}
			valid := binary.Class == elf.ELFCLASS64 && binary.Machine == elf.EM_X86_64
			closeErr := binary.Close()
			if closeErr != nil {
				return closeErr
			}
			if !valid {
				return errors.New("bundle member architecture does not match amd64")
			}
		}
		file, err := os.Open(location)
		if err != nil {
			return err
		}
		sum := sha512.New()
		size, copyErr := io.Copy(sum, io.LimitReader(file, (1<<30)+1))
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if size != info.Size() {
			return errors.New("bundle member changed during capture")
		}
		prefix := fmt.Sprintf("/gvisor/runtime_bundle/members/%d/", i)
		replacements[prefix+"size"] = size
		replacements[prefix+"sha512"] = hex.EncodeToString(sum.Sum(nil))
	}
	count := 0
	if err := filepath.WalkDir(builtRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(builtRoot, path)
		if err != nil {
			return err
		}
		known := false
		for _, p := range bundlePaths {
			if relative == p {
				known = true
				break
			}
		}
		if !known {
			return errors.New("unexpected bundle member during capture")
		}
		count++
		return nil
	}); err != nil {
		return err
	}
	if count != len(bundlePaths) {
		return errors.New("incomplete bundle capture")
	}
	candidate, err := editJSON(lockBytes, replacements)
	if err != nil {
		return err
	}
	if err := atomicWrite(destination, candidate, 0o600); err != nil {
		return err
	}
	receipt := gvisorCaptureReceipt{1, root, builtRoot, *module, *preparation, hash(lockBytes), hash(candidate)}
	if err := saveJSON(destination+".receipt.json", receipt); err != nil {
		return err
	}
	fmt.Fprintf(out, "captured six-member %s candidate with Go%s and Bazel lock %s; strict verifier/qualification still required\n", destination, selected, hash(moduleBytes))
	return nil
}

// Reconstruct the selected patch in a private Git index. Object writes stay in
// the private repository; the prepared source and its original index are read-only.
func verifyPreparedGvisor(root, sourceDir, parent string, source gvisorPreparation, patchPath string, preparedModule []byte) error {
	patch, _, err := readRegular(root, patchPath)
	if err != nil {
		return err
	}
	if hash(patch) != source.Patch {
		return errors.New("selected HAA patch bytes changed")
	}
	probe, err := os.MkdirTemp(parent, "source-check-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(probe)
	if _, err := command(probe, nil, "git", "init", "--quiet", "--bare"); err != nil {
		return err
	}
	env := append(os.Environ(), "GIT_DIR="+probe, "GIT_WORK_TREE="+sourceDir, "GIT_ALTERNATE_OBJECT_DIRECTORIES="+filepath.Join(sourceDir, ".git", "objects"), "GIT_INDEX_FILE="+filepath.Join(probe, "index"))
	if _, err := command(probe, env, "git", "read-tree", source.Commit); err != nil {
		return err
	}
	if _, err := command(probe, env, "git", "apply", "--cached", filepath.Join(root, patchPath)); err != nil {
		return err
	}
	expectedModule, err := command(probe, env, "git", "show", ":go.mod")
	if err != nil {
		return err
	}
	goLine := regexp.MustCompile(`(?m)^go [0-9]+\.[0-9]+\.[0-9]+$`)
	if len(goLine.FindAll(expectedModule, -1)) != 1 || !bytes.Equal(goLine.ReplaceAll(expectedModule, []byte("go "+source.Go)), preparedModule) {
		return errors.New("prepared Go module differs from exact patched upstream plus selected SDK")
	}
	args := []string{"diff", "--no-ext-diff", "--no-textconv", "--binary", "--full-index", "--no-renames", "--no-color", "--diff-algorithm=myers", "--unified=3", "--src-prefix=a/", "--dst-prefix=b/"}
	paths := []string{"--", ".", ":(exclude)go.mod", ":(exclude)MODULE.bazel.lock", ":(exclude)tools/haa_gvisor_observer"}
	expectedArgs := append(append(append([]string{}, args...), "--cached", source.Commit), paths...)
	expected, err := command(probe, env, "git", expectedArgs...)
	if err != nil {
		return err
	}
	actualEnv := append(append([]string{}, env...), "GIT_INDEX_FILE="+filepath.Join(probe, "actual-index"))
	if _, err := command(sourceDir, actualEnv, "git", "read-tree", source.Commit); err != nil {
		return err
	}
	// Include new patch-created files as well as tracked modifications. Bazel's
	// produced module lock is validated separately; the HAA observer is its own target.
	if _, err := command(sourceDir, actualEnv, "git", "add", "--all", "--", "."); err != nil {
		return err
	}
	actual, err := command(sourceDir, actualEnv, "git", expectedArgs...)
	if err != nil {
		return err
	}
	if len(expected) >= 2<<20 || len(actual) >= 2<<20 || !bytes.Equal(expected, actual) {
		return errors.New("prepared tracked source differs from exact selected HAA patch")
	}
	return nil
}

func readGvisorCapture(path string) (gvisorCaptureInput, error) {
	if !filepath.IsAbs(path) {
		return gvisorCaptureInput{}, errors.New("gVisor capture input must be absolute")
	}
	body, _, err := readRegular(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		return gvisorCaptureInput{}, err
	}
	var receipt gvisorCaptureReceipt
	if err := loadJSON(path+".receipt.json", &receipt); err != nil {
		return gvisorCaptureInput{}, err
	}
	if receipt.Schema != 1 || receipt.OutputSHA256 != hash(body) {
		return gvisorCaptureInput{}, errors.New("capture receipt/output identity mismatch")
	}
	return gvisorCaptureInput{body, receipt}, nil
}

func gvisorCaptureChanges(p plan, state string, runtimeBody []byte) ([]byte, error) {
	capture := p.GvisorCapture
	if capture == nil {
		return runtimeBody, nil
	}
	if p.Component != "go" || capture.Receipt.Schema != 1 || capture.Receipt.OutputSHA256 != hash(capture.Lock) {
		return nil, errors.New("capture cannot change an unrelated component")
	}
	if _, err := jsonSpans(capture.Lock); err != nil {
		return nil, err
	}
	var proposed, current map[string]any
	if err := json.Unmarshal(capture.Lock, &proposed); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(runtimeBody, &current); err != nil {
		return nil, err
	}
	proposedGo, ok := proposed["go_image"].(map[string]any)
	if !ok || proposedGo["go_version"] != p.Version {
		return nil, errors.New("captured SDK version does not match requested Go")
	}
	proposedGvisor, ok := proposed["gvisor"].(map[string]any)
	if !ok {
		return nil, errors.New("captured gVisor owner missing")
	}
	currentGvisor, ok := current["gvisor"].(map[string]any)
	if !ok {
		return nil, errors.New("current gVisor owner missing")
	}
	proposedBuild, ok := proposedGvisor["build"].(map[string]any)
	if !ok {
		return nil, errors.New("captured build owner missing")
	}
	currentBuild, ok := currentGvisor["build"].(map[string]any)
	if !ok {
		return nil, errors.New("current build owner missing")
	}
	proposedBundle, ok := proposedGvisor["runtime_bundle"].(map[string]any)
	if !ok {
		return nil, errors.New("captured bundle owner missing")
	}
	currentBundle, ok := currentGvisor["runtime_bundle"].(map[string]any)
	if !ok {
		return nil, errors.New("current bundle owner missing")
	}
	// Source/patch/builder/architecture remain the selected owner. Only observed
	// member identities and the produced Bazel lock may be imported.
	currentBuild["bazel_module_lock_sha256"] = proposedBuild["bazel_module_lock_sha256"]
	currentBundle["members"] = proposedBundle["members"]
	afterGvisor, _ := json.Marshal(currentGvisor)
	capturedGvisor, _ := json.Marshal(proposedGvisor)
	if !bytes.Equal(afterGvisor, capturedGvisor) {
		return nil, errors.New("capture attempts to change source/patch/builder/architecture")
	}
	input, _, err := readRegular(capture.Receipt.Root, runtimeFile)
	if err != nil {
		return nil, err
	}
	if hash(input) != capture.Receipt.RuntimeInputSHA256 {
		return nil, errors.New("captured build input changed")
	}
	probe, err := os.MkdirTemp(state, "capture-check-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(probe)
	recomputed := filepath.Join(probe, "runtime.json")
	if err := captureGvisor([]string{"--root", capture.Receipt.Root, "--built-dir", capture.Receipt.Built, "--module-lock", capture.Receipt.Module, "--source-manifest", capture.Receipt.Preparation, "--output", recomputed}, io.Discard); err != nil {
		return nil, err
	}
	actual, err := os.ReadFile(recomputed)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(actual, capture.Lock) {
		return nil, errors.New("capture metadata differs from actual current build outputs")
	}
	members, ok := proposedBundle["members"].([]any)
	if !ok || len(members) != len(bundlePaths) {
		return nil, errors.New("captured member inventory mismatch")
	}
	replacements := map[string]any{"/gvisor/build/bazel_module_lock_sha256": proposedBuild["bazel_module_lock_sha256"]}
	for i, raw := range members {
		m, ok := raw.(map[string]any)
		if !ok || m["path"] != bundlePaths[i] {
			return nil, errors.New("captured member path mismatch")
		}
		prefix := fmt.Sprintf("/gvisor/runtime_bundle/members/%d/", i)
		replacements[prefix+"size"] = m["size"]
		replacements[prefix+"sha512"] = m["sha512"]
	}
	return editJSON(runtimeBody, replacements)
}
