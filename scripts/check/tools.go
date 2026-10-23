package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

const toolLockSchemaVersion = 2

// checkGoVersionConsistency keeps the product floor, quality compiler and
// isolated Go SDK on the same explicitly selected version.
func (c *checker) checkGoVersionConsistency() error {
	body, err := os.ReadFile(filepath.Join(c.root, "go.mod"))
	if err != nil {
		return err
	}
	moduleVersion := ""
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "go" {
			continue
		}
		if moduleVersion != "" || len(fields) < 2 || (len(fields) > 2 && !strings.HasPrefix(fields[2], "//")) {
			return errors.New("go version consistency: invalid or duplicate go directive")
		}
		moduleVersion = fields[1]
	}
	if moduleVersion == "" {
		return errors.New("go version consistency: invalid or missing go directive")
	}
	body, err = os.ReadFile(filepath.Join(c.root, "scripts", "runtimes.lock.json"))
	if err != nil {
		return err
	}
	var lock struct {
		GoImage struct {
			Version string `json:"go_version"`
		} `json:"go_image"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		return err
	}
	if len(c.toolLock.Tools) == 0 || lock.GoImage.Version != moduleVersion {
		return errors.New("go version consistency: guest SDK must match go.mod")
	}
	for _, tool := range c.toolLock.Tools {
		if tool.SetupGo != moduleVersion {
			return errors.New("go version consistency: quality compiler must match go.mod")
		}
	}
	body, err = os.ReadFile(filepath.Join(c.root, "README.md"))
	if err != nil {
		return fmt.Errorf("go version consistency: read README: %w", err)
	}
	return checkReadmeGoBuild(body, moduleVersion)
}

func checkReadmeGoBuild(body []byte, selected string) error {
	const heading = "\n## Build\n"
	text := string(body)
	if strings.Count(text, heading) != 1 {
		return errors.New("go version consistency: README must contain one Build section")
	}
	_, section, _ := strings.Cut(text, heading)
	section, _, _ = strings.Cut(section, "\n## ")
	expected := fmt.Sprintf("Heliopause는 Go `%s` 이상이 필요하며, 개발·CI toolchain은 Go `%s`으로 고정한다.", selected, selected)
	fields := 0
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "Heliopause는 Go ") {
			if line != expected {
				return errors.New("go version consistency: README minimum/toolchain must match go.mod")
			}
			fields++
		}
	}
	if fields != 1 {
		return errors.New("go version consistency: README requires one known minimum/toolchain paragraph")
	}
	return nil
}

type toolLock struct {
	SchemaVersion int        `json:"schemaVersion"`
	Tools         []toolSpec `json:"tools"`
}

type toolSpec struct {
	Command          string      `json:"command"`
	Package          string      `json:"package"`
	Version          string      `json:"version"`
	ExpectedVersion  string      `json:"expectedVersion"`
	VersionArguments []string    `json:"versionArguments"`
	VersionMatch     string      `json:"versionMatch"`
	Install          string      `json:"install"`
	SetupGo          string      `json:"setupGo"`
	Assets           []toolAsset `json:"assets"`
	ModuleSum        string      `json:"moduleSum,omitempty"`
	LDFlags          string      `json:"ldflags,omitempty"`
}

type toolAsset struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

func readToolLock(path string) (toolLock, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return toolLock{}, &checkFailure{class: unavailable, step: "tool lock", detail: "scripts/tools.lock.json is not a regular file"}
	}
	file, err := os.Open(path)
	if err != nil {
		return toolLock{}, &checkFailure{class: executionFailure, step: "tool lock", cause: err}
	}
	decoder := json.NewDecoder(io.LimitReader(file, outputLimit+1))
	decoder.DisallowUnknownFields()
	var lock toolLock
	decodeErr := decoder.Decode(&lock)
	if decodeErr != nil {
		_ = file.Close()
		return toolLock{}, &checkFailure{class: executionFailure, step: "tool lock", detail: "decode lock manifest", cause: decodeErr}
	}
	if err := ensureJSONEOF(decoder); err != nil {
		_ = file.Close()
		return toolLock{}, err
	}
	if err := file.Close(); err != nil {
		return toolLock{}, &checkFailure{class: executionFailure, step: "tool lock", detail: "close lock manifest", cause: err}
	}
	if err := validateToolLock(lock); err != nil {
		return toolLock{}, err
	}
	return lock, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return &checkFailure{class: executionFailure, step: "tool lock", detail: "trailing content", cause: err}
	}
	return nil
}

func validateToolLock(lock toolLock) error {
	if lock.SchemaVersion != toolLockSchemaVersion {
		return &checkFailure{class: unavailable, step: "tool lock", detail: fmt.Sprintf("unsupported schemaVersion %d", lock.SchemaVersion)}
	}
	if len(lock.Tools) != 5 {
		return &checkFailure{class: unavailable, step: "tool lock", detail: "require exactly five active tools"}
	}
	want := map[string]string{
		"staticcheck": "honnef.co/go/tools/cmd/staticcheck",
		"actionlint":  "github.com/rhysd/actionlint/cmd/actionlint",
		"gosec":       "github.com/securego/gosec/v2/cmd/gosec",
		"govulncheck": "golang.org/x/vuln/cmd/govulncheck",
		"gitleaks":    "github.com/zricethezav/gitleaks/v8",
	}
	seen := make(map[string]bool, len(want))
	for _, tool := range lock.Tools {
		packagePath, known := want[tool.Command]
		if !known || seen[tool.Command] || tool.Version == "" || tool.ExpectedVersion == "" || !exactRuntimeVersion.MatchString(tool.SetupGo) || tool.SetupGo != lock.Tools[0].SetupGo {
			return &checkFailure{class: unavailable, step: "tool lock", detail: "tool identity is incomplete, duplicated or unexpected"}
		}
		seen[tool.Command] = true
		if strings.Contains(tool.Version, "@") || strings.EqualFold(tool.Version, "latest") {
			return &checkFailure{class: unavailable, step: "tool lock", detail: "tool version must be an exact release without @"}
		}
		if tool.Install != "" || len(tool.Assets) != 0 || tool.Package != packagePath {
			return &checkFailure{class: unavailable, step: "tool lock", detail: "Go-installed tool identity is unexpected"}
		}
		if tool.Command == "gitleaks" {
			sum, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(tool.ModuleSum, "h1:"))
			if !strings.HasPrefix(tool.ModuleSum, "h1:") || err != nil || len(sum) != sha256.Size || tool.LDFlags != "-s -w -X="+tool.Package+"/cmd.Version="+strings.TrimPrefix(tool.Version, "v") {
				return &checkFailure{class: unavailable, step: "tool lock", detail: "gitleaks source checksum or version flags are incomplete"}
			}
		} else if tool.ModuleSum != "" || tool.LDFlags != "" {
			return &checkFailure{class: unavailable, step: "tool lock", detail: "unexpected source build override"}
		}
	}
	if len(seen) != len(want) {
		return &checkFailure{class: unavailable, step: "tool lock", detail: "required tool identity is missing"}
	}
	return nil
}

func resolveToolCache(root, configured string) (string, error) {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", &checkFailure{class: unavailable, step: "tool cache", detail: "resolve source root", cause: err}
	}
	root = resolvedRoot

	cache := configured
	if cache == "" {
		userCache, cacheErr := os.UserCacheDir()
		if cacheErr != nil {
			return "", &checkFailure{class: unavailable, step: "tool cache", detail: "resolve user cache", cause: cacheErr}
		}
		cache = filepath.Join(userCache, "heliopause", "quality-tools")
	}
	if !filepath.IsAbs(cache) {
		return "", &checkFailure{class: unavailable, step: "tool cache", detail: "HELOX_TOOL_CACHE must be an absolute path"}
	}
	cache, err = resolvePotentialPath(cache)
	if err != nil {
		return "", &checkFailure{class: unavailable, step: "tool cache", detail: "resolve cache path", cause: err}
	}
	if isWithinRoot(root, cache) {
		return "", &checkFailure{class: unavailable, step: "tool cache", detail: "tool cache must be outside the source tree"}
	}
	return cache, nil
}

func resolvePotentialPath(path string) (string, error) {
	path = filepath.Clean(path)
	current := path
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for index := len(missing) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, missing[index])
			}
			return resolved, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func (c *checker) bootstrap() error {
	if len(c.toolLock.Tools) == 0 {
		return &checkFailure{class: unavailable, step: "bootstrap", detail: "tool lock has no active tools"}
	}
	setupGo := c.toolLock.Tools[0].SetupGo
	for _, tool := range c.toolLock.Tools {
		if tool.SetupGo != setupGo {
			return &checkFailure{class: unavailable, step: "bootstrap", detail: "active tools require different setup Go versions"}
		}
	}
	if runtime.Version() != "go"+setupGo {
		return &checkFailure{class: unavailable, step: "bootstrap", detail: fmt.Sprintf("setup Go is %s, require go%s", runtime.Version(), setupGo)}
	}
	steps := []checkStep{
		{name: "prepare module cache", run: c.prepareModuleCache},
		{name: "download product modules", run: c.downloadProductModules},
		{name: "prepare quality tool cache", run: c.prepareQualityToolCache},
	}
	for _, tool := range c.toolLock.Tools {
		tool := tool
		steps = append(steps,
			checkStep{name: "install " + tool.Command, run: func() error { return c.installTool(tool) }},
			checkStep{name: "verify " + tool.Command + " identity", run: func() error { return c.verifyTool(tool) }},
		)
	}
	return c.runSequential(steps)
}

func (c *checker) bootstrapModules() error {
	return c.runSequential([]checkStep{
		{name: "prepare module cache", run: c.prepareModuleCache},
		{name: "download product modules", run: c.downloadProductModules},
	})
}

func (c *checker) prepareModuleCache() error {
	for _, directory := range []string{
		filepath.Join(c.toolCache, "go-build"),
		filepath.Join(c.toolCache, "go-mod"),
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return &checkFailure{class: executionFailure, step: "bootstrap", detail: "create project tool cache", cause: err}
		}
	}
	return nil
}

func (c *checker) prepareQualityToolCache() error {
	for _, directory := range []string{
		filepath.Join(c.toolCache, "bin"),
		filepath.Join(c.toolCache, "staticcheck"),
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return &checkFailure{class: executionFailure, step: "bootstrap", detail: "create project tool cache", cause: err}
		}
	}
	return nil
}

func (c *checker) downloadProductModules() (resultErr error) {
	temporaryRoot, err := os.MkdirTemp(c.toolCache, "module-download-")
	if err != nil {
		return &checkFailure{class: executionFailure, step: "product module download", detail: "create isolated module root", cause: err}
	}
	defer func() {
		if cleanupErr := os.RemoveAll(temporaryRoot); cleanupErr != nil {
			cleanupFailure := &checkFailure{class: executionFailure, step: "product module download", detail: "remove isolated module root", cause: cleanupErr}
			resultErr = errors.Join(resultErr, cleanupFailure)
		}
	}()
	for _, name := range []string{"go.mod", "go.sum"} {
		if err := copyRegularFile(filepath.Join(c.root, name), filepath.Join(temporaryRoot, name)); err != nil {
			return err
		}
	}

	downloadChecker := *c
	downloadChecker.root = temporaryRoot
	_, err = downloadChecker.runCommandWithTimeout("product module download", 10*time.Minute, c.bootstrapEnvironment(), c.goExecutable, "mod", "download", "all")
	return err
}

func copyRegularFile(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil || !info.Mode().IsRegular() {
		return &checkFailure{class: unavailable, step: "product module download", detail: fmt.Sprintf("%s is not a regular file", filepath.Base(source))}
	}
	contents, err := os.ReadFile(source)
	if err != nil {
		return &checkFailure{class: executionFailure, step: "product module download", detail: fmt.Sprintf("read %s", filepath.Base(source)), cause: err}
	}
	if err := os.WriteFile(destination, contents, 0o600); err != nil {
		return &checkFailure{class: executionFailure, step: "product module download", detail: fmt.Sprintf("stage %s", filepath.Base(source)), cause: err}
	}
	return nil
}

func (c *checker) installTool(tool toolSpec) error {
	if tool.Install == "archive" {
		return c.installArchiveTool(tool)
	}
	identity := tool.Package + "@" + tool.Version
	args := []string{"install"}
	env := c.bootstrapEnvironment()
	if tool.LDFlags != "" {
		args = append(args, "-trimpath", "-ldflags="+tool.LDFlags)
		env = overrideEnvironment(env, map[string]string{"CGO_ENABLED": "0", "GOEXPERIMENT": "", "GOAMD64": "v1", "GOARM64": "v8.0"})
	}
	args = append(args, identity)
	_, err := c.runCommandWithTimeout(tool.Command+" install", 10*time.Minute, env, c.goExecutable, args...)
	return err
}

func (c *checker) installArchiveTool(tool toolSpec) (resultErr error) {
	var asset *toolAsset
	for index := range tool.Assets {
		candidate := &tool.Assets[index]
		if candidate.GOOS == runtime.GOOS && candidate.GOARCH == runtime.GOARCH {
			asset = candidate
			break
		}
	}
	if asset == nil {
		return &checkFailure{class: unavailable, step: tool.Command + " install", detail: "no pinned release asset for this platform"}
	}
	temporaryRoot, err := os.MkdirTemp(c.toolCache, tool.Command+"-download-")
	if err != nil {
		return &checkFailure{class: executionFailure, step: tool.Command + " install", detail: "create private download directory", cause: err}
	}
	defer func() {
		if cleanupErr := makeWritableRemoveAll(temporaryRoot); cleanupErr != nil {
			resultErr = errors.Join(resultErr, &checkFailure{class: executionFailure, step: tool.Command + " install", detail: "remove private download directory", cause: cleanupErr})
		}
	}()
	archivePath := filepath.Join(temporaryRoot, "release.tar.gz")
	if err := downloadPinnedAsset(asset.URL, asset.SHA256, archivePath); err != nil {
		return &checkFailure{class: executionFailure, step: tool.Command + " install", cause: err}
	}
	stagedExecutable := filepath.Join(temporaryRoot, executableName(tool.Command))
	if err := extractPinnedExecutable(archivePath, executableName(tool.Command), stagedExecutable); err != nil {
		return &checkFailure{class: executionFailure, step: tool.Command + " install", cause: err}
	}
	if err := os.Chmod(stagedExecutable, 0o700); err != nil {
		return &checkFailure{class: executionFailure, step: tool.Command + " install", detail: "set executable mode", cause: err}
	}
	if err := os.Rename(stagedExecutable, c.toolExecutable(tool)); err != nil {
		return &checkFailure{class: executionFailure, step: tool.Command + " install", detail: "publish verified executable", cause: err}
	}
	return nil
}

func downloadPinnedAsset(url, expectedSHA256, destination string) error {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Request.URL.Scheme != "https" {
		return fmt.Errorf("unexpected release download response %s", response.Status)
	}
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(file, io.TeeReader(io.LimitReader(response.Body, 64<<20), hash))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != expectedSHA256 {
		return errors.New("release asset SHA-256 does not match lock")
	}
	return nil
}

func extractPinnedExecutable(archivePath, command, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)
	found := false
	for {
		header, readErr := reader.Next()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != command {
			continue
		}
		if found || header.Size <= 0 || header.Size > 64<<20 {
			return errors.New("release archive has an invalid executable entry")
		}
		output, createErr := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
		if createErr != nil {
			return createErr
		}
		_, copyErr := io.Copy(output, io.LimitReader(reader, header.Size))
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		found = true
	}
	if !found {
		return errors.New("release archive does not contain the expected executable")
	}
	return nil
}

func makeWritableRemoveAll(path string) error {
	if err := filepath.Walk(path, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := os.Chmod(current, info.Mode()|0o700); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func (c *checker) bootstrapEnvironment() []string {
	return overrideEnvironment(os.Environ(), map[string]string{
		"GOBIN":       filepath.Join(c.toolCache, "bin"),
		"GOCACHE":     filepath.Join(c.toolCache, "go-build"),
		"GOENV":       "off",
		"GOFLAGS":     "",
		"GOINSECURE":  "",
		"GOMODCACHE":  filepath.Join(c.toolCache, "go-mod"),
		"GONOPROXY":   "",
		"GONOSUMDB":   "",
		"GOPRIVATE":   "",
		"GOPROXY":     "https://proxy.golang.org",
		"GOSUMDB":     "sum.golang.org",
		"GOTOOLCHAIN": "local",
		"GOWORK":      "off",
	})
}

func (c *checker) toolExecutable(tool toolSpec) string {
	return filepath.Join(c.toolCache, "bin", executableName(tool.Command))
}

func (c *checker) verifyTool(tool toolSpec) error {
	executable := c.toolExecutable(tool)
	info, err := os.Lstat(executable)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return &checkFailure{class: unavailable, step: tool.Command, detail: "pinned executable is missing from the project tool cache"}
	}
	metadata, err := buildinfo.ReadFile(executable)
	if err != nil {
		return &checkFailure{class: unavailable, step: tool.Command, detail: "managed Go tool has no readable compiler metadata", cause: err}
	}
	if err := validateToolBuildInfo(tool, metadata); err != nil {
		return err
	}
	arguments := tool.VersionArguments
	if len(arguments) == 0 {
		arguments = []string{"-version"}
	}
	output, err := c.runCommandWithTimeout(tool.Command+" version", 10*time.Second, c.offlineEnvironment(), executable, arguments...)
	if err != nil {
		return err
	}
	actual := strings.TrimSpace(output)
	return validateToolVersion(tool, actual)
}

func validateToolBuildInfo(tool toolSpec, info *debug.BuildInfo) error {
	if info == nil || info.GoVersion != "go"+tool.SetupGo || info.Path != tool.Package {
		return &checkFailure{class: unavailable, step: tool.Command, detail: "actual compiler/package does not match the selected tool owner; run canonical bootstrap"}
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		if _, exists := settings[setting.Key]; exists {
			return &checkFailure{class: unavailable, step: tool.Command, detail: "duplicate executable build setting"}
		}
		settings[setting.Key] = setting.Value
	}
	if settings["GOOS"] != runtime.GOOS || settings["GOARCH"] != runtime.GOARCH {
		return &checkFailure{class: unavailable, step: tool.Command, detail: "executable build platform does not match the host"}
	}
	if tool.ModuleSum != "" && (info.Main.Path != tool.Package || info.Main.Version != tool.Version || info.Main.Sum != tool.ModuleSum || info.Main.Replace != nil || settings["-trimpath"] != "true" || settings["CGO_ENABLED"] != "0") {
		return &checkFailure{class: unavailable, step: tool.Command, detail: "source module checksum/version or deterministic build settings do not match lock"}
	}
	return nil
}

func validateToolVersion(tool toolSpec, actual string) error {
	if tool.VersionMatch == "contains" {
		if strings.Contains(actual, tool.ExpectedVersion) {
			return nil
		}
	} else if tool.VersionMatch == "" && actual == tool.ExpectedVersion {
		return nil
	}
	if tool.VersionMatch != "" && tool.VersionMatch != "contains" {
		return &checkFailure{class: unavailable, step: tool.Command, detail: "unsupported version match mode"}
	}
	if tool.VersionMatch == "" || !strings.Contains(actual, tool.ExpectedVersion) {
		return &checkFailure{class: unavailable, step: tool.Command, detail: fmt.Sprintf("version %q does not match lock %q", actual, tool.ExpectedVersion)}
	}
	return nil
}

func (c *checker) runStaticcheck() error {
	tool, err := c.tool("staticcheck")
	if err != nil {
		return err
	}
	if err := c.verifyTool(tool); err != nil {
		return err
	}
	return c.runAnalysis("Staticcheck", c.toolExecutable(tool), "./...")
}

func (c *checker) tool(command string) (toolSpec, error) {
	for _, tool := range c.toolLock.Tools {
		if tool.Command == command {
			return tool, nil
		}
	}
	return toolSpec{}, &checkFailure{class: unavailable, step: "tool lock", detail: command + " is not configured"}
}

func (c *checker) runAnalysis(step, executable string, args ...string) error {
	return c.runAnalysisWithEnvironment(step, c.offlineEnvironment(), executable, args...)
}

func (c *checker) runAnalysisWithEnvironment(step string, environment []string, executable string, args ...string) error {
	output, err := c.runCommandWithTimeout(step, 5*time.Minute, environment, executable, args...)
	if err == nil {
		return nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 && !strings.Contains(strings.ToLower(output), "invalid configuration") {
		return &checkFailure{class: findingFailure, step: step, detail: strings.TrimSpace(output)}
	}
	return err
}
