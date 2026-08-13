package main

import (
	"bytes"
	"context"
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

const (
	commandTimeout = 2 * time.Minute
	outputLimit    = 1 << 20
)

type failureClass string

const (
	findingFailure   failureClass = "finding"
	executionFailure failureClass = "execution failure"
	unavailable      failureClass = "unavailable"
)

type checkFailure struct {
	class  failureClass
	step   string
	detail string
	cause  error
}

func (f *checkFailure) Error() string {
	message := fmt.Sprintf("%s: %s", f.class, f.step)
	if f.detail != "" {
		message += ": " + f.detail
	}
	if f.cause != nil {
		message += ": " + f.cause.Error()
	}
	return message
}

func (f *checkFailure) Unwrap() error { return f.cause }

type checker struct {
	root         string
	stdout       io.Writer
	goExecutable string
	gofmt        string
}

type checkStep struct {
	name string
	run  func() error
}

func moduleRoot() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	return validateModuleRoot(workingDirectory)
}

func validateModuleRoot(workingDirectory string) (string, error) {
	root, err := filepath.EvalSymlinks(workingDirectory)
	if err != nil {
		return "", fmt.Errorf("resolve module root: %w", err)
	}

	moduleFile := filepath.Join(root, "go.mod")
	info, err := os.Lstat(moduleFile)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("run from the module root containing a regular go.mod")
	}
	contents, err := os.ReadFile(moduleFile)
	if err != nil {
		return "", errors.New("run from the module root containing go.mod")
	}
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" && fields[1] != "" {
			return root, nil
		}
	}

	return "", errors.New("go.mod has no module directive")
}

func newChecker(root string, stdout io.Writer) (*checker, error) {
	goExecutable, err := exec.LookPath(executableName("go"))
	if err != nil {
		return nil, errors.New("go executable is unavailable")
	}
	goExecutable, err = filepath.Abs(goExecutable)
	if err != nil {
		return nil, fmt.Errorf("resolve Go executable: %w", err)
	}
	goExecutable, err = filepath.EvalSymlinks(goExecutable)
	if err != nil {
		return nil, fmt.Errorf("resolve Go executable symlinks: %w", err)
	}
	if err := requireRegularExecutable(goExecutable); err != nil {
		return nil, err
	}

	versionOutput, err := runToolDiscovery(root, goExecutable, "version")
	if err != nil {
		return nil, fmt.Errorf("verify Go executable: %w", err)
	}
	versionFields := strings.Fields(versionOutput)
	if len(versionFields) < 3 || versionFields[0] != "go" || versionFields[1] != "version" || versionFields[2] != runtime.Version() {
		return nil, fmt.Errorf("go executable version %q does not match runner build version %q", strings.TrimSpace(versionOutput), runtime.Version())
	}

	gorootOutput, err := runToolDiscovery(root, goExecutable, "env", "GOROOT")
	if err != nil {
		return nil, fmt.Errorf("resolve GOROOT: %w", err)
	}
	goroot := strings.TrimSpace(gorootOutput)
	if !filepath.IsAbs(goroot) {
		return nil, fmt.Errorf("go executable returned a non-absolute GOROOT %q", goroot)
	}
	gofmt := filepath.Join(goroot, "bin", executableName("gofmt"))
	if err := requireRegularExecutable(gofmt); err != nil {
		return nil, err
	}

	return &checker{root: root, stdout: stdout, goExecutable: goExecutable, gofmt: gofmt}, nil
}

func requireRegularExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return fmt.Errorf("required Go tool is not a regular executable: %s", path)
	}
	return nil
}

func runToolDiscovery(root, executable string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = root
	command.Env = deterministicGoEnvironment(os.Environ())
	var output boundedBuffer
	output.limit = 64 * 1024
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if ctx.Err() != nil {
		return output.String(), ctx.Err()
	}
	return output.String(), err
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func (c *checker) runProfile(profile string) error {
	switch profile {
	case "foundation":
		return c.runSequential([]checkStep{
			{"format check", c.checkFormat},
			{"module drift", c.checkModuleDrift},
			{"module integrity", func() error { return c.runGo("module integrity", "mod", "verify") }},
			{"production build", func() error { return c.runGo("production build", "build", "./...") }},
			{"test build validity", func() error { return c.runGo("test build validity", "test", "-run", "^$", "./...") }},
			{"architecture", c.checkArchitecture},
			{"documentation", func() error { return checkMarkdownTree(c.root) }},
		})
	case "docs":
		return c.runStep("documentation", func() error { return checkMarkdownTree(c.root) })
	case "format":
		return c.runStep("format", c.applyFormat)
	default:
		return &checkFailure{class: unavailable, step: "profile", detail: fmt.Sprintf("unknown profile %q", profile)}
	}
}

func (c *checker) runSequential(steps []checkStep) error {
	for _, step := range steps {
		if err := c.runStep(step.name, step.run); err != nil {
			return err
		}
	}
	return nil
}

func (c *checker) runStep(name string, action func() error) error {
	if _, err := fmt.Fprintf(c.stdout, "[check] %s\n", name); err != nil {
		return &checkFailure{class: executionFailure, step: name, detail: "write progress", cause: err}
	}
	return action()
}

func (c *checker) runGo(step string, args ...string) error {
	output, err := c.runCommand(step, c.goExecutable, args...)
	if err != nil && strings.Contains(output, "GOPROXY=off") {
		return &checkFailure{class: unavailable, step: step, detail: "offline module cache prerequisite is missing", cause: err}
	}
	return err
}

func (c *checker) checkModuleDrift() error {
	output, err := c.runCommand("module drift", c.goExecutable, "mod", "tidy", "-diff")
	trimmed := strings.TrimSpace(output)
	if trimmed != "" && (strings.HasPrefix(trimmed, "diff ") || strings.Contains(trimmed, "\ndiff ")) {
		return &checkFailure{class: findingFailure, step: "module drift", detail: trimmed}
	}
	if err != nil && strings.Contains(output, "GOPROXY=off") {
		return &checkFailure{class: unavailable, step: "module drift", detail: "offline module cache prerequisite is missing", cause: err}
	}
	return err
}

func (c *checker) runCommand(step, executable string, args ...string) (string, error) {
	if !filepath.IsAbs(executable) {
		return "", &checkFailure{class: unavailable, step: step, detail: "executable path is not absolute"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = c.root
	command.Env = deterministicGoEnvironment(os.Environ())
	var output boundedBuffer
	output.limit = outputLimit
	command.Stdout = &output
	command.Stderr = &output

	err := command.Run()
	if ctx.Err() != nil {
		return output.String(), &checkFailure{class: executionFailure, step: step, detail: "command timed out", cause: ctx.Err()}
	}
	if err != nil {
		detail := fmt.Sprintf("%s %s", executable, strings.Join(args, " "))
		if text := strings.TrimSpace(output.String()); text != "" {
			detail += "\n" + text
		}
		return output.String(), &checkFailure{class: executionFailure, step: step, detail: detail, cause: err}
	}

	return output.String(), nil
}

func deterministicGoEnvironment(current []string) []string {
	overrides := map[string]string{
		"GOFLAGS":     "",
		"GOPROXY":     "off",
		"GOTOOLCHAIN": "local",
		"GOWORK":      "off",
	}
	result := make([]string, 0, len(current)+len(overrides))
	for _, entry := range current {
		key, _, found := strings.Cut(entry, "=")
		if _, replace := overrides[key]; found && replace {
			continue
		}
		result = append(result, entry)
	}
	for _, key := range []string{"GOFLAGS", "GOPROXY", "GOTOOLCHAIN", "GOWORK"} {
		result = append(result, key+"="+overrides[key])
	}
	return result
}

type boundedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	originalLength := len(data)
	remaining := b.limit - b.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}
		_, _ = b.buffer.Write(data)
	}
	if originalLength > remaining {
		b.truncated = true
	}
	return originalLength, nil
}

func (b *boundedBuffer) String() string {
	if b.truncated {
		return b.buffer.String() + "\n[output truncated]"
	}
	return b.buffer.String()
}
