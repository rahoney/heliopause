package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/testutil/fakeworkflow"
)

func TestMain(m *testing.M) {
	config, err := os.MkdirTemp("", "helox-cli-test-config-")
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
func TestLanguageSelectionSettingsAndEnglishFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("HELOX_LANG", "")
	for _, test := range []struct{ locale, want string }{{"ko_KR.UTF-8", "ko"}, {"de_DE.UTF-8", "de"}, {"fr_FR.UTF-8", "fr"}, {"es_ES.UTF-8", "es"}, {"pt_BR.UTF-8", "pt-BR"}, {"ja_JP.UTF-8", "ja"}, {"zz_ZZ", "en"}} {
		t.Setenv("LANG", test.locale)
		p := &presentation{output: "auto", locale: "auto", color: "auto", progress: "auto"}
		if err := p.configure(); err != nil || p.language != test.want {
			t.Fatalf("locale=%s => %s, %v", test.locale, p.language, err)
		}
	}
	if err := writeLanguageSettings("ko"); err != nil {
		t.Fatal(err)
	}
	p := &presentation{output: "auto", locale: "auto", color: "never", progress: "never"}
	if err := p.configure(); err != nil || p.language != "ko" {
		t.Fatal("saved setting not selected", err, p.language)
	}
	p.locale = "en"
	p.localeExplicit = true
	if err := p.configure(); err != nil || p.language != "en" {
		t.Fatal("explicit override not selected")
	}
	p.locale = "ja"
	if err := p.configure(); err != nil {
		t.Fatal(err)
	}
	if p.text("untranslated explanation", "unused") != "untranslated explanation" {
		t.Fatal("missing secondary translation did not use English")
	}
	path, _ := languageSettingsPath()
	if body, _ := os.ReadFile(path); !bytes.Contains(body, []byte("ko")) {
		t.Fatal("one-shot flag rewrote setting")
	}
	if err := writeLanguageSettings("auto"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"language":"ko","unexpected":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readLanguageSettings(); err == nil {
		t.Fatal("unknown config field accepted")
	}
	if err := writeLanguageSettings("en"); err != nil {
		t.Fatal("cannot repair malformed language preference", err)
	}
}
func TestSettingsRejectSymlinkAndUnsupportedLanguage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := writeLanguageSettings("unsupported"); err == nil {
		t.Fatal("unsupported language saved")
	}
	path, _ := languageSettingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "other")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err := writeLanguageSettings("ko"); err == nil {
		t.Fatal("settings symlink overwritten")
	}
	if body, _ := os.ReadFile(target); string(body) != "original" {
		t.Fatal("unrelated symlink target changed")
	}
}
func TestSevenLanguageHelpAndLeadingFlags(t *testing.T) {
	for _, test := range []struct{ lang, label string }{{"en", "Available Commands:"}, {"ko", "사용 가능한 명령:"}, {"de", "Verfügbare Befehle:"}, {"fr", "Commandes disponibles"}, {"es", "Comandos disponibles:"}, {"pt-BR", "Comandos disponíveis:"}, {"ja", "利用可能なコマンド:"}} {
		t.Run(test.lang, func(t *testing.T) {
			var out bytes.Buffer
			root, err := New(&out, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"--lang", test.lang, "npm", "--help"}
			source, err := PreparePresentation(root, args)
			if err != nil || source != "npm" {
				t.Fatal(source, err)
			}
			root.SetArgs(args)
			if err = root.ExecuteContext(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), test.label) || strings.Contains(out.String(), "\x1b") {
				t.Fatalf("bad localized non-TTY help: %q", out.String())
			}
		})
	}
}
func TestHumanAndJSONKeepSamePolicyAndErrorAxes(t *testing.T) {
	for _, test := range []struct {
		scenario fakeworkflow.Scenario
		policy   string
		code     int
	}{{fakeworkflow.Safe, "ALLOW", 0}, {fakeworkflow.Review, "MANUAL_REVIEW", 10}, {fakeworkflow.Blocked, "BLOCK", 20}, {fakeworkflow.ResolveError, "", 1}} {
		fake, err := fakeworkflow.New(test.scenario)
		if err != nil {
			t.Fatal(err)
		}
		reference, _ := fake.Reference()
		request, _ := application.NewInspectRequest(reference)
		operationID, _ := domain.ParseOperationID("op_aaaaaaaaaaaaaaaaaaaaaaaaaa")
		runID, _ := domain.ParseRunID("run_aaaaaaaaaaaaaaaaaaaaaaaaaa")
		service, err := application.NewInspectService(fake, fake, fake, fake, policy.M1{}, func() (domain.OperationID, error) { return operationID, nil }, func() (domain.RunID, error) { return runID, nil })
		if err != nil {
			t.Fatal(err)
		}
		var human bytes.Buffer
		p := &presentation{out: &human, diagnostic: io.Discard, language: "ko", output: "human", color: "never", progress: "never"}
		writer := &presentationWriter{Writer: &human, presentation: p}
		code, operationErr := ExecuteInspect(context.Background(), service, request, false, writer)
		if code != test.code {
			t.Fatal("human exit changed", code)
		}
		if test.policy != "" && !strings.Contains(human.String(), test.policy) {
			t.Fatal("policy omitted", human.String())
		}
		if test.policy == "" && strings.Contains(human.String(), "ALLOW") {
			t.Fatal("error shown as allowed", human.String())
		}
		if strings.Contains(human.String(), "synthetic resolve failure") || strings.Contains(human.String(), "\x1b") {
			t.Fatal("raw error or ANSI leaked", human.String())
		}
		var machine bytes.Buffer
		machineCode, machineErr := ExecuteInspect(context.Background(), service, request, true, &machine)
		if machineCode != code || (machineErr == nil) != (operationErr == nil) {
			t.Fatal("presentation changed outcome")
		}
		if operationErr != nil && (!errors.Is(operationErr, fakeworkflow.ErrResolve) || !errors.Is(machineErr, fakeworkflow.ErrResolve)) {
			t.Fatal("original error identity lost")
		}
		var doc map[string]any
		if err := json.Unmarshal(machine.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		if doc["schema_version"] != "helox.operation-result/v1" || strings.Contains(machine.String(), "수행한 검사") {
			t.Fatal("machine contract changed")
		}
	}
}
func TestProgressStopsAndNeverContaminatesMachineOutput(t *testing.T) {
	var out, diagnostic bytes.Buffer
	p := &presentation{out: &out, diagnostic: &diagnostic, language: "ko", output: "human", color: "never", progress: "always", width: 30}
	writer := &presentationWriter{Writer: &out, presentation: p}
	ctx, finish := beginProgress(context.Background(), writer, false)
	ports.ReportProgress(ctx, ports.Progress{Stage: "download", Current: 20, Total: 0, Unit: "bytes"})
	if err := finish(); err != nil {
		t.Fatal(err)
	}
	before := diagnostic.String()
	ports.ReportProgress(ctx, ports.Progress{Stage: "install"})
	if diagnostic.String() != before {
		t.Fatal("progress continued after finish")
	}
	if strings.Contains(before, "%") || strings.Contains(before, "\x1b") {
		t.Fatal("unknown total or non-TTY output fabricated", before)
	}
	machineCtx, machineFinish := beginProgress(context.Background(), writer, true)
	ports.ReportProgress(machineCtx, ports.Progress{Stage: "install"})
	_ = machineFinish()
	if diagnostic.String() != before {
		t.Fatal("machine progress contaminated output")
	}
	if safeTerminal("x\x1b[31m\n\u202eevil") == "x\x1b[31m\n\u202eevil" {
		t.Fatal("terminal controls not escaped")
	}
}

func TestMalformedPreferenceFallsBackAndExplicitAutoUsesSystem(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LANG", "ja_JP.UTF-8")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("HELOX_LANG", "")
	if err := writeLanguageSettings("ko"); err != nil {
		t.Fatal(err)
	}
	p := &presentation{output: "human", locale: "auto", localeExplicit: true, color: "never", progress: "never", diagnostic: io.Discard}
	if err := p.configure(); err != nil || p.language != "ja" {
		t.Fatal("explicit auto did not follow system", err, p.language)
	}
	path, _ := languageSettingsPath()
	if err := os.WriteFile(path, []byte(`{"language":"bad"}`), 0600); err != nil {
		t.Fatal(err)
	}
	p.localeExplicit = false
	if err := p.configure(); err != nil || p.language != "en" {
		t.Fatal("malformed preference broke operation instead of using English", err, p.language)
	}
}
func TestUnknownReasonDoesNotInventExplanation(t *testing.T) {
	p := &presentation{language: "ko"}
	if got := p.reason("UNKNOWN_NETWORK_ATTEMPT"); strings.Contains(got, "네트워크 연결") {
		t.Fatal("unknown code given a guessed explanation", got)
	}
}

func TestAnimatedProgressStopsOnCancellation(t *testing.T) {
	for _, term := range []string{"xterm-256color", "dumb"} {
		t.Run(term, func(t *testing.T) {
			t.Setenv("TERM", term)
			var output, diagnostic bytes.Buffer
			p := &presentation{out: &output, diagnostic: &diagnostic, output: "human", language: "ko", color: "never", progress: "always", diagnosticTTY: true, width: 24}
			writer := &presentationWriter{Writer: &output, presentation: p}
			original, cancel := context.WithCancel(context.Background())
			ctx, finish := beginProgress(original, writer, false)
			ports.ReportProgress(ctx, ports.Progress{Stage: "download", Current: 3, Total: 6, Unit: "bytes"})
			cancel()
			if err := finish(); err != nil {
				t.Fatal(err)
			}
			before := diagnostic.String()
			ports.ReportProgress(ctx, ports.Progress{Stage: "install"})
			if diagnostic.String() != before {
				t.Fatal("cancelled progress kept writing")
			}
			if got := strings.Contains(before, "\x1b[2K"); got != (term != "dumb") {
				t.Fatalf("terminal clear differs from TERM capability: %s %q", term, before)
			}
			if got := fitTerminal("한국어表示", 6); got != "한국어" {
				t.Fatalf("wide terminal text was not bounded: %q", got)
			}
		})
	}
}

func TestFinalizationFailureIsVisibleAfterAllowedOperation(t *testing.T) {
	var out bytes.Buffer
	root, err := New(&out, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	p := presentationFor(root.OutOrStdout())
	p.output = "human"
	p.language = "ko"
	p.color = "never"
	p.reportHeader("INSTALL", "COMPLETED", "ALLOW", "COMPLETED")
	PresentFinalizationError(root, errors.New("private raw cleanup cause"))
	if !strings.Contains(out.String(), "PROCESS_FINALIZATION_FAILED") || !strings.Contains(out.String(), "정상 종료·정리") || strings.Contains(out.String(), "private raw") {
		t.Fatal("cleanup uncertainty hidden or raw cause leaked", out.String())
	}
}

func TestUserCancellationIsNotReportedAsBrokenObserver(t *testing.T) {
	var out bytes.Buffer
	root, err := New(&out, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	p := presentationFor(root.OutOrStdout())
	p.output = "human"
	p.language = "ko"
	p.color = "never"
	cause := errors.Join(errors.New("observer startup failed"), context.Canceled)
	reported := PresentRuntimeError(root, cause)
	if !errors.Is(reported, context.Canceled) || !strings.Contains(out.String(), "OPERATION_CANCELLED") || !strings.Contains(out.String(), "사용자 중단") || strings.Contains(out.String(), "OBSERVER_HELPER_UNAVAILABLE") {
		t.Fatal("user cancellation misclassified", out.String())
	}
}
