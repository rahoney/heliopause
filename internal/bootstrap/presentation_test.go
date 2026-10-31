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
