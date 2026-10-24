package hosttool

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const displayReadLimit = 4 << 10
const displayReadTimeout = 2 * time.Second

var errDisplayLanguage = errors.New("OS display language unavailable")

type displayCommand struct {
	path, directory, source string
	arguments               []string
}

// PreferredDisplayLanguages reads display-only OS preferences. It grants no
// executable role, accepts no request data and does not compose a runtime/helper.
func PreferredDisplayLanguages(ctx context.Context) ([]string, string, error) {
	plan, ok := displayLanguageCommand(runtime.GOOS, runtime.GOOS == "linux" && displayWSL())
	if !ok || ctx == nil {
		return nil, "", errDisplayLanguage
	}
	if plan.source == "MACOS_UI" {
		if _, err := verifySystemExecutable(plan.path); err != nil {
			return nil, plan.source, errDisplayLanguage
		}
	} else {
		// DrvFS maps Windows ACLs to Unix modes/UIDs. This fixed Windows system
		// executable is only an optional UI read, never trusted artifact execution.
		info, err := os.Lstat(plan.path)
		if err != nil || !info.Mode().IsRegular() {
			return nil, plan.source, errDisplayLanguage
		}
	}
	env := []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	if plan.source == "MACOS_UI" {
		home, err := os.UserHomeDir()
		if err != nil || !filepath.IsAbs(home) {
			return nil, plan.source, errDisplayLanguage
		}
		env = append(env, "HOME="+home)
	} else {
		for _, key := range []string{"WSL_INTEROP", "WSL_DISTRO_NAME"} {
			if value := os.Getenv(key); value != "" {
				env = append(env, key+"="+value)
			}
		}
	}
	readContext, cancel := context.WithTimeout(ctx, displayReadTimeout)
	defer cancel()
	body, err := runDisplayCommand(readContext, plan, env)
	if err != nil {
		return nil, plan.source, err
	}
	tags, err := parseDisplayLanguages(body, plan.source)
	return tags, plan.source, err
}

func displayLanguageCommand(goos string, wsl bool) (displayCommand, bool) {
	if goos == "darwin" {
		return displayCommand{"/usr/bin/defaults", "/", "MACOS_UI", []string{"read", "-g", "AppleLanguages"}}, true
	}
	if goos == "linux" && wsl {
		return displayCommand{"/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe", "/mnt/c/Windows", "WINDOWS_UI", []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "$PSModuleAutoLoadingPreference='None'; [System.Globalization.CultureInfo]::CurrentUICulture.Name"}}, true
	}
	return displayCommand{}, false
}
func displayWSL() bool {
	file, err := os.Open("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, 513))
	return err == nil && len(body) <= 512 && strings.Contains(strings.ToLower(string(body)), "microsoft")
}

type displayOutput struct {
	buffer   bytes.Buffer
	overflow bool
	cancel   context.CancelFunc
}

func (b *displayOutput) Write(data []byte) (int, error) {
	n := len(data)
	remaining := displayReadLimit - b.buffer.Len()
	if n > remaining {
		data = data[:remaining]
		b.overflow = true
		b.cancel()
	}
	_, _ = b.buffer.Write(data)
	return n, nil
}
func runDisplayCommand(ctx context.Context, plan displayCommand, environment []string) ([]byte, error) {
	childContext, cancel := context.WithCancel(ctx)
	defer cancel()
	stdout, stderr := &displayOutput{cancel: cancel}, &displayOutput{cancel: cancel}
	command := exec.CommandContext(childContext, plan.path, plan.arguments...)
	command.Dir, command.Env = plan.directory, environment
	command.Stdin, command.Stdout, command.Stderr = nil, stdout, stderr
	command.WaitDelay = 100 * time.Millisecond
	err := command.Run()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil || stdout.overflow || stderr.overflow {
		return nil, errDisplayLanguage
	}
	return stdout.buffer.Bytes(), nil
}

func parseDisplayLanguages(body []byte, source string) ([]string, error) {
	if len(body) == 0 || len(body) > displayReadLimit {
		return nil, errDisplayLanguage
	}
	text := strings.TrimSpace(strings.TrimPrefix(string(body), "\ufeff"))
	if source == "WINDOWS_UI" {
		if !validDisplayTag(text) {
			return nil, errDisplayLanguage
		}
		return []string{text}, nil
	}
	if source != "MACOS_UI" || !strings.HasPrefix(text, "(") || !strings.HasSuffix(text, ")") {
		return nil, errDisplayLanguage
	}
	members := strings.Split(text[1:len(text)-1], ",")
	if len(members) > 33 {
		return nil, errDisplayLanguage
	}
	tags := make([]string, 0, len(members))
	for index, member := range members {
		member = strings.TrimSpace(member)
		if member == "" && index == len(members)-1 {
			continue
		}
		if strings.HasPrefix(member, "\"") && strings.HasSuffix(member, "\"") && len(member) >= 2 {
			member = member[1 : len(member)-1]
		}
		if !validDisplayTag(member) {
			return nil, errDisplayLanguage
		}
		tags = append(tags, member)
	}
	if len(tags) == 0 || len(tags) > 32 {
		return nil, errDisplayLanguage
	}
	return tags, nil
}
func validDisplayTag(value string) bool {
	if len(value) < 2 || len(value) > 64 {
		return false
	}
	parts := strings.Split(value, "-")
	for i, part := range parts {
		if i == 0 && (len(part) < 2 || len(part) > 3) || i > 0 && (len(part) < 2 || len(part) > 8) {
			return false
		}
		for _, r := range part {
			if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9' {
				continue
			}
			return false
		}
	}
	return true
}
