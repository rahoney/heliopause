package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const toolLockSchemaVersion = 1

type toolLock struct {
	SchemaVersion int        `json:"schemaVersion"`
	Tools         []toolSpec `json:"tools"`
}

type toolSpec struct {
	Command         string `json:"command"`
	Package         string `json:"package"`
	Version         string `json:"version"`
	ExpectedVersion string `json:"expectedVersion"`
	SetupGo         string `json:"setupGo"`
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
	if len(lock.Tools) != 1 {
		return &checkFailure{class: unavailable, step: "tool lock", detail: "M0-005 requires exactly one active tool"}
	}
	tool := lock.Tools[0]
	if tool.Command != "staticcheck" || tool.Package != "honnef.co/go/tools/cmd/staticcheck" || tool.Version == "" || tool.ExpectedVersion == "" || tool.SetupGo == "" {
		return &checkFailure{class: unavailable, step: "tool lock", detail: "Staticcheck identity is incomplete or unexpected"}
	}
	if strings.Contains(tool.Version, "@") || strings.EqualFold(tool.Version, "latest") {
		return &checkFailure{class: unavailable, step: "tool lock", detail: "tool version must be an exact release without @"}
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
	tool := c.toolLock.Tools[0]
	if runtime.Version() != "go"+tool.SetupGo {
		return &checkFailure{class: unavailable, step: "bootstrap", detail: fmt.Sprintf("setup Go is %s, require go%s", runtime.Version(), tool.SetupGo)}
	}
	return c.runSequential([]checkStep{
		{name: "prepare tool cache", run: c.prepareToolCache},
		{name: "download product modules", run: c.downloadProductModules},
		{name: "install Staticcheck", run: func() error { return c.installTool(tool) }},
		{name: "verify Staticcheck identity", run: func() error { return c.verifyTool(tool) }},
	})
}

func (c *checker) prepareToolCache() error {
	for _, directory := range []string{
		filepath.Join(c.toolCache, "bin"),
		filepath.Join(c.toolCache, "go-build"),
		filepath.Join(c.toolCache, "go-mod"),
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
	_, err = downloadChecker.runCommandWithTimeout("product module download", 10*time.Minute, c.bootstrapEnvironment(), c.goExecutable, "mod", "download")
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
	identity := tool.Package + "@" + tool.Version
	_, err := c.runCommandWithTimeout("Staticcheck install", 10*time.Minute, c.bootstrapEnvironment(), c.goExecutable, "install", identity)
	return err
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
	output, err := c.runCommandWithTimeout(tool.Command+" version", 10*time.Second, c.offlineEnvironment(), executable, "-version")
	if err != nil {
		return err
	}
	actual := strings.TrimSpace(output)
	return validateToolVersion(tool, actual)
}

func validateToolVersion(tool toolSpec, actual string) error {
	if actual != tool.ExpectedVersion {
		return &checkFailure{class: unavailable, step: tool.Command, detail: fmt.Sprintf("version %q does not match lock %q", actual, tool.ExpectedVersion)}
	}
	return nil
}

func (c *checker) runStaticcheck() error {
	tool := c.toolLock.Tools[0]
	if err := c.verifyTool(tool); err != nil {
		return err
	}
	return c.runAnalysis("Staticcheck", c.toolExecutable(tool), "./...")
}

func (c *checker) runAnalysis(step, executable string, args ...string) error {
	output, err := c.runCommandWithTimeout(step, 5*time.Minute, c.offlineEnvironment(), executable, args...)
	if err == nil {
		return nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 && !strings.Contains(strings.ToLower(output), "invalid configuration") {
		return &checkFailure{class: findingFailure, step: step, detail: strings.TrimSpace(output)}
	}
	return err
}
