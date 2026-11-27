package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/testutil/fakeworkflow"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSystemLanguageSelectionPrecedenceAndSingleProbe(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HELOX_LANG", "")
	t.Setenv("LANGUAGE", "")
	t.Setenv("LC_ALL", "C.UTF-8")
	for _, language := range []string{"en-US", "ko-KR", "de-DE", "fr-FR", "es-ES", "pt-BR", "ja-JP"} {
		p := &presentation{output: "auto", locale: "auto", color: "never", progress: "never"}
		calls := 0
		p.detectLanguage = func() ([]string, string, error) { calls++; return []string{"zh-CN", language}, "WINDOWS_UI", nil }
		for range 3 {
			if err := p.configure(); err != nil {
				t.Fatal(err)
			}
		}
		if p.language != normalizeLanguage(language) || p.languageSource != "WINDOWS_UI" || calls != 1 {
			t.Fatal(p.language, p.languageSource, calls)
		}
	}
	p := &presentation{output: "auto", locale: "en", localeExplicit: true, color: "never", progress: "never", detectLanguage: func() ([]string, string, error) { t.Fatal("manual language probed OS"); return nil, "", nil }}
	if err := p.configure(); err != nil || p.languageSource != "FLAG" {
		t.Fatal(err, p.languageSource)
	}
	p.localeExplicit = false
	t.Setenv("HELOX_LANG", "ja")
	if err := p.configure(); err != nil || p.language != "ja" || p.languageSource != "ENV_HELOX_LANG" {
		t.Fatal(err)
	}
	t.Setenv("HELOX_LANG", "")
	if err := writeLanguageSettings("ko"); err != nil {
		t.Fatal(err)
	}
	if err := p.configure(); err != nil || p.language != "ko" || p.languageSource != "SAVED" {
		t.Fatal(err)
	}
}
func TestAutomaticLanguageFallbackSourceAndCache(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HELOX_LANG", "")
	t.Setenv("LANGUAGE", "")
	t.Setenv("LC_ALL", "C.UTF-8")
	for _, test := range []struct {
		err    error
		tags   []string
		reason string
	}{{context.DeadlineExceeded, nil, "OS_LANGUAGE_TIMEOUT"}, {errors.New("unavailable"), nil, "OS_LANGUAGE_UNAVAILABLE"}, {nil, []string{"zh-CN"}, "OS_LANGUAGE_UNSUPPORTED"}} {
		calls := 0
		p := &presentation{output: "auto", locale: "auto", color: "never", progress: "never", detectLanguage: func() ([]string, string, error) { calls++; return test.tags, "WINDOWS_UI", test.err }}
		for range 2 {
			if err := p.configure(); err != nil {
				t.Fatal(err)
			}
		}
		if p.language != "en" || p.languageSource != "ENGLISH_FALLBACK" || p.languageFallback != test.reason || calls != 1 {
			t.Fatal(p.language, p.languageSource, p.languageFallback, calls)
		}
	}
	t.Setenv("LC_ALL", "ko_KR.UTF-8")
	p := &presentation{output: "auto", locale: "auto", color: "never", progress: "never"}
	if err := p.configure(); err != nil || p.language != "ko" || p.languageSource != "LOCALE_LC_ALL" {
		t.Fatal(err, p.languageSource)
	}
}
func TestLanguageSourceAfterSaveHonorsFlagAndEnvironment(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HELOX_LANG", "")
	t.Setenv("LANGUAGE", "")
	t.Setenv("LC_ALL", "C.UTF-8")
	run := func(args []string, detect func() ([]string, string, error)) string {
		var output bytes.Buffer
		root, err := New(&output, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		if err := SetLanguageDetector(root, detect); err != nil {
			t.Fatal(err)
		}
		if _, err := PreparePresentation(root, args); err != nil {
			t.Fatal(err)
		}
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatal(err)
		}
		return output.String()
	}
	noProbe := func() ([]string, string, error) { t.Fatal("manual save probed OS"); return nil, "", nil }
	body := run([]string{"config", "language", "ko"}, noProbe)
	if !strings.Contains(body, "현재 언어: ko") || !strings.Contains(body, "SAVED") {
		t.Fatal(body)
	}
	body = run([]string{"--lang", "en", "config", "language", "ja"}, noProbe)
	if !strings.Contains(body, "Current language: en") || !strings.Contains(body, "FLAG") {
		t.Fatal(body)
	}
	t.Setenv("HELOX_LANG", "en")
	body = run([]string{"config", "language", "ko"}, noProbe)
	if !strings.Contains(body, "Current language: en") || !strings.Contains(body, "ENV_HELOX_LANG") {
		t.Fatal(body)
	}
	t.Setenv("HELOX_LANG", "")
	calls := 0
	body = run([]string{"config", "language", "auto"}, func() ([]string, string, error) { calls++; return []string{"ko-KR"}, "WINDOWS_UI", nil })
	if calls != 1 || !strings.Contains(body, "현재 언어: ko") || !strings.Contains(body, "WINDOWS_UI") {
		t.Fatal(calls, body)
	}
	path, _ := languageSettingsPath()
	saved, _ := os.ReadFile(path)
	if !bytes.Contains(saved, []byte(`"auto"`)) {
		t.Fatal("detected language persisted", string(saved))
	}
}

func TestSystemLanguageHelpAndJSONKeepOutputContract(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HELOX_LANG", "")
	t.Setenv("LANGUAGE", "")
	t.Setenv("LC_ALL", "C.UTF-8")
	var output bytes.Buffer
	root, err := New(&output, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	detect := func() ([]string, string, error) { calls++; return []string{"ko-KR"}, "WINDOWS_UI", nil }
	if err := SetLanguageDetector(root, detect); err != nil {
		t.Fatal(err)
	}
	args := []string{"npm", "--help"}
	if _, err := PreparePresentation(root, args); err != nil {
		t.Fatal(err)
	}
	root.SetArgs(args)
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !strings.Contains(output.String(), "사용 가능한 명령:") {
		t.Fatal(calls, output.String())
	}
	output.Reset()
	calls = 0
	root, err = New(&output, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if err := SetLanguageDetector(root, detect); err != nil {
		t.Fatal(err)
	}
	if _, err := PreparePresentation(root, []string{"--output", "json", "npm", "inspect", "sample@1.0.0"}); err != nil {
		t.Fatal(err)
	}
	fake, err := fakeworkflow.New(fakeworkflow.Safe)
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := fake.Reference()
	request, _ := application.NewInspectRequest(ref)
	service, err := application.NewInspectService(fake, fake, fake, fake, policy.M1{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	code, err := ExecuteInspect(context.Background(), service, request, machineOutput(root), root.OutOrStdout())
	if err != nil || code != 0 {
		t.Fatal(code, err)
	}
	var result map[string]any
	if err := json.Unmarshal(output.Bytes(), &result); err != nil || result["schema_version"] != "helox.operation-result/v1" {
		t.Fatal(output.String(), err)
	}
	if calls != 1 || strings.Contains(output.String(), "WINDOWS_UI") || strings.Contains(output.String(), "현재 언어") || strings.Contains(output.String(), "\x1b") {
		t.Fatal(calls, output.String())
	}
}
