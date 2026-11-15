package bootstrap_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/bootstrap"
)

func TestMain(m *testing.M) {
	config, err := os.MkdirTemp("", "helox-bootstrap-test-config-")
	if err != nil {
		os.Exit(1)
	}
	_ = os.Setenv("XDG_CONFIG_HOME", config)
	_ = os.Setenv("HELOX_LANG", "")
	_ = os.Setenv("LC_ALL", "C")
	code := m.Run()
	_ = os.RemoveAll(config)
	os.Exit(code)
}
func TestLanguageSettingsAndHelpNeedNoRuntimeComposition(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	if err := bootstrap.Run(context.Background(), []string{"--lang", "ko", "npm", "--help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "사용 가능한 명령:") {
		t.Fatal(stdout.String())
	}
	stdout.Reset()
	if err := bootstrap.Run(context.Background(), []string{"config", "language", "ja"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := bootstrap.Run(context.Background(), []string{"--help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "利用可能なコマンド:") {
		t.Fatal("saved language not used", stdout.String())
	}
}

func TestInvalidCommandsFailBeforeRuntimePreparation(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("PATH", t.TempDir())
	for _, args := range [][]string{
		{"npm", "ci", "extra"}, {"cargo", "fetch"}, {"go", "mod", "tidy"},
		{"npm", "not-a-command", "--help"}, {"pipx", "not-a-command"},
		{"go", "build"}, {"cargo", "build", "extra"}, {"doctor", "extra"},
	} {
		var stdout, stderr bytes.Buffer
		err := bootstrap.Run(context.Background(), append([]string{"--lang", "en"}, args...), &stdout, &stderr)
		if err == nil || (!strings.Contains(err.Error(), "unknown command") && !strings.Contains(err.Error(), "arg")) {
			t.Fatalf("request %v missed syntax preflight: %v", args, err)
		}
		if stdout.Len() != 0 {
			t.Fatalf("invalid request %v printed successful help: %s", args, stdout.String())
		}
	}
	if entries, err := os.ReadDir(cache); err != nil || len(entries) != 0 {
		t.Fatal("invalid command prepared runtime/cache", entries, err)
	}
}
