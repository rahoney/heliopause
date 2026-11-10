package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

func TestGoVersionConsistencyRejectsMixedSDKs(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, module, guest, tool string
		wantError                 bool
	}{
		{"same version", "1.26.9", "1.26.9", "1.26.9", false},
		{"retired 1.26.5 guest", "1.26.9", "1.26.5", "1.26.9", true},
		{"retired 1.26.7 guest", "1.26.9", "1.26.7", "1.26.9", true},
		{"unpatched guest", "1.26.9", "1.26.8", "1.26.9", true},
		{"old product floor", "1.26.7", "1.26.9", "1.26.9", true},
		{"old tool compiler", "1.26.9", "1.26.9", "1.26.7", true},
		{"partial future upgrade", "1.27.2", "1.26.9", "1.27.2", true},
		{"complete future upgrade", "1.27.2", "1.27.2", "1.27.2", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeTestFile(t, filepath.Join(root, "go.mod"), "module example.test/app\n\ngo "+test.module+"\n")
			writeTestFile(t, filepath.Join(root, "README.md"), fmt.Sprintf("# Fixture\n\n## Build\n\nHeliopause는 Go `%s` 이상이 필요하며, 개발·CI toolchain은 Go `%s`으로 고정한다.\n", test.module, test.module))
			writeTestFile(t, filepath.Join(root, "scripts", "runtimes.lock.json"), `{"go_image":{"go_version":"`+test.guest+`"}}`)
			c := checker{root: root, toolLock: toolLock{Tools: []toolSpec{{SetupGo: test.module}, {SetupGo: test.tool}}}}
			if err := c.checkGoVersionConsistency(); (err != nil) != test.wantError {
				t.Fatalf("mixed Go inputs: error = %v, want error %v", err, test.wantError)
			}
		})
	}
}

func TestReadToolLock(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestToolLock(t, root)
	lock, err := readToolLock(filepath.Join(root, "scripts", "tools.lock.json"))
	if err != nil {
		t.Fatalf("readToolLock error: %v", err)
	}
	if len(lock.Tools) != 5 || lock.Tools[0].ExpectedVersion != "staticcheck 2026.1 (v0.7.0)" {
		t.Fatalf("readToolLock = %#v", lock)
	}
}

func TestReadToolLockRejectsUnknownAndFloatingValues(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"unknown field":         strings.Replace(validTestToolLock, `"schemaVersion": 2`, `"schemaVersion": 2, "unexpected": true`, 1),
		"retired schema":        strings.Replace(validTestToolLock, `"schemaVersion": 2`, `"schemaVersion": 1`, 1),
		"archive compiler":      strings.Replace(validTestToolLock, `"command":"gitleaks"`, `"command":"gitleaks","install":"archive"`, 1),
		"wrong source sum":      strings.Replace(validTestToolLock, `h1:mWOfVGO8ksok21iOb7h4DZMcUxyvsol8l6o1uNOQxww=`, `h1:invalid`, 1),
		"unreviewed build flag": strings.Replace(validTestToolLock, `-s -w -X=github.com/zricethezav/gitleaks/v8/cmd.Version=8.18.4`, `-s -w -extldflags=unreviewed`, 1),
		"floating":              strings.Replace(validTestToolLock, `"version":"2026.1"`, `"version":"latest"`, 1),
		"trailing":              validTestToolLock + `{}`,
	}
	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "tools.lock.json")
			writeTestFile(t, path, contents)
			if _, err := readToolLock(path); err == nil {
				t.Fatal("readToolLock error = nil, want non-nil")
			}
		})
	}
}

func TestActualToolCompilerAndSourceIdentity(t *testing.T) {
	t.Parallel()
	const source = "github.com/zricethezav/gitleaks/v8"
	const sum = "h1:mWOfVGO8ksok21iOb7h4DZMcUxyvsol8l6o1uNOQxww="
	for _, scenario := range []string{"matching", "future owner", "old compiler", "wrong package", "wrong OS", "wrong architecture", "missing settings", "duplicate setting", "wrong module version", "wrong checksum", "replacement", "untrimmed path", "CGO build", "missing metadata"} {
		t.Run(scenario, func(t *testing.T) {
			tool := toolSpec{Command: "gitleaks", Package: source, Version: "v8.18.4", SetupGo: "1.26.9", ModuleSum: sum}
			info := &debug.BuildInfo{GoVersion: "go1.26.9", Path: source, Main: debug.Module{Path: source, Version: "v8.18.4", Sum: sum}, Settings: []debug.BuildSetting{{Key: "GOOS", Value: runtime.GOOS}, {Key: "GOARCH", Value: runtime.GOARCH}, {Key: "-trimpath", Value: "true"}, {Key: "CGO_ENABLED", Value: "0"}}}
			wantError := true
			switch scenario {
			case "matching":
				wantError = false
			case "future owner":
				tool.SetupGo = "1.99.1"
				info.GoVersion = "go1.99.1"
				wantError = false
			case "old compiler":
				info.GoVersion = "go1.22.1"
			case "wrong package":
				info.Path = "example.invalid/other"
			case "wrong OS":
				info.Settings[0].Value = "unknown"
			case "wrong architecture":
				info.Settings[1].Value = "unknown"
			case "missing settings":
				info.Settings = nil
			case "duplicate setting":
				info.Settings = append(info.Settings, info.Settings[0])
			case "wrong module version":
				info.Main.Version = "v8.18.3"
			case "wrong checksum":
				info.Main.Sum = "h1:other"
			case "replacement":
				info.Main.Replace = &debug.Module{Path: "example.invalid/replacement"}
			case "untrimmed path":
				info.Settings[2].Value = "false"
			case "CGO build":
				info.Settings[3].Value = "1"
			case "missing metadata":
				info = nil
			}
			if err := validateToolBuildInfo(tool, info); (err != nil) != wantError {
				t.Fatalf("actual build identity error=%v, want=%v", err, wantError)
			}
		})
	}
}

func TestResolveToolCache(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "quality-cache")
	resolved, err := resolveToolCache(root, outside)
	if err != nil {
		t.Fatalf("resolveToolCache outside error: %v", err)
	}
	if resolved == "" || !filepath.IsAbs(resolved) {
		t.Fatalf("resolveToolCache = %q", resolved)
	}
	if _, err := resolveToolCache(root, filepath.Join(root, "cache")); err == nil {
		t.Fatal("resolveToolCache inside source error = nil, want non-nil")
	}
	if _, err := resolveToolCache(root, "relative/cache"); err == nil {
		t.Fatal("resolveToolCache relative error = nil, want non-nil")
	}
}

func TestResolveToolCacheRejectsSymlinkIntoSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic link permissions vary on Windows")
	}
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(outside, "cache-link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := resolveToolCache(root, filepath.Join(link, "tools")); err == nil {
		t.Fatal("resolveToolCache symlink into source error = nil, want non-nil")
	}
}

func TestValidateToolVersion(t *testing.T) {
	t.Parallel()

	tool := toolSpec{Command: "staticcheck", ExpectedVersion: "staticcheck 2026.1 (v0.7.0)"}
	if err := validateToolVersion(tool, tool.ExpectedVersion); err != nil {
		t.Fatalf("validateToolVersion matching error: %v", err)
	}
	var failure *checkFailure
	err := validateToolVersion(tool, "staticcheck 2025.1.1 (v0.6.1)")
	if !errors.As(err, &failure) || failure.class != unavailable {
		t.Fatalf("validateToolVersion mismatch error = %v, want unavailable", err)
	}
}

func TestBootstrapRejectsSetupGoMismatchBeforeMutation(t *testing.T) {
	t.Parallel()

	cache := filepath.Join(t.TempDir(), "missing-cache")
	checker := checker{
		stdout:    io.Discard,
		toolCache: cache,
		toolLock: toolLock{Tools: []toolSpec{{
			SetupGo: "0.0.0",
		}}},
	}
	var failure *checkFailure
	err := checker.bootstrap()
	if !errors.As(err, &failure) || failure.class != unavailable {
		t.Fatalf("bootstrap error = %v, want unavailable", err)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("bootstrap created cache before rejecting setup Go: %v", err)
	}
}

func TestPrepareModuleCacheExcludesQualityTools(t *testing.T) {
	t.Parallel()

	cache := filepath.Join(t.TempDir(), "cache")
	checker := checker{toolCache: cache}
	if err := checker.prepareModuleCache(); err != nil {
		t.Fatalf("prepareModuleCache error: %v", err)
	}
	for _, name := range []string{"go-build", "go-mod"} {
		if info, err := os.Stat(filepath.Join(cache, name)); err != nil || !info.IsDir() {
			t.Fatalf("module cache directory %q unavailable: %v", name, err)
		}
	}
	for _, name := range []string{"bin", "staticcheck"} {
		if _, err := os.Stat(filepath.Join(cache, name)); !os.IsNotExist(err) {
			t.Fatalf("quality tool directory %q created by module bootstrap: %v", name, err)
		}
	}
}

func TestVerifyToolRejectsMissingPinnedExecutable(t *testing.T) {
	t.Parallel()

	checker := checker{toolCache: t.TempDir()}
	var failure *checkFailure
	err := checker.verifyTool(toolSpec{Command: "staticcheck"})
	if !errors.As(err, &failure) || failure.class != unavailable {
		t.Fatalf("verifyTool error = %v, want unavailable", err)
	}
}

func TestBootstrapEnvironmentKeepsVerificationEnabled(t *testing.T) {
	t.Setenv("GOINSECURE", "*")
	t.Setenv("GONOSUMDB", "*")
	t.Setenv("GOPRIVATE", "*")

	checker := checker{toolCache: filepath.Join(t.TempDir(), "cache")}
	environment := strings.Join(checker.bootstrapEnvironment(), "\n")
	for _, expected := range []string{
		"GOENV=off",
		"GOINSECURE=",
		"GONOSUMDB=",
		"GOPRIVATE=",
		"GOPROXY=https://proxy.golang.org",
		"GOSUMDB=sum.golang.org",
		"GOTOOLCHAIN=local",
	} {
		if !strings.Contains(environment, expected) {
			t.Errorf("bootstrap environment missing %q", expected)
		}
	}
	for _, forbidden := range []string{"GOINSECURE=*", "GONOSUMDB=*", "GOPRIVATE=*"} {
		if strings.Contains(environment, forbidden) {
			t.Errorf("bootstrap environment retained %q", forbidden)
		}
	}
}

const validTestToolLock = `{
  "schemaVersion": 2,
  "tools": [
    {"command":"staticcheck","package":"honnef.co/go/tools/cmd/staticcheck","version":"2026.1","expectedVersion":"staticcheck 2026.1 (v0.7.0)","setupGo":"1.26.9"},
    {"command":"actionlint","package":"github.com/rhysd/actionlint/cmd/actionlint","version":"v1.7.12","expectedVersion":"v1.7.12","versionMatch":"contains","setupGo":"1.26.9"},
    {"command":"gosec","package":"github.com/securego/gosec/v2/cmd/gosec","version":"v2.28.0","expectedVersion":"Version: dev","setupGo":"1.26.9"},
    {"command":"govulncheck","package":"golang.org/x/vuln/cmd/govulncheck","version":"v1.7.0","expectedVersion":"govulncheck@v1.7.0","versionMatch":"contains","setupGo":"1.26.9"},
    {"command":"gitleaks","package":"github.com/zricethezav/gitleaks/v8","version":"v8.18.4","expectedVersion":"8.18.4","versionArguments":["version"],"setupGo":"1.26.9","moduleSum":"h1:mWOfVGO8ksok21iOb7h4DZMcUxyvsol8l6o1uNOQxww=","ldflags":"-s -w -X=github.com/zricethezav/gitleaks/v8/cmd.Version=8.18.4"}
  ]
}`
