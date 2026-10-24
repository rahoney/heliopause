package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/spf13/cobra"
)

func TestLauncherKeysAndBoundedEscapeInput(t *testing.T) {
	for input, want := range map[string]launcherKey{
		"\x1b[A": launcherUp, "\x1b[B": launcherDown, "\x1bOC": launcherSelect,
		"\x1b[D": launcherBack, "\x1b[5~": launcherPageUp, "\x1b[6~": launcherPageDown,
		"\x1b[H": launcherFirst, "\x1b[4~": launcherLast,
		"\r": launcherSelect, "q": launcherQuit, "\x03": launcherInterrupt, "\x04": launcherQuit,
	} {
		t.Run(input, func(t *testing.T) {
			var decoder launcherKeyDecoder
			got := launcherIdle
			for _, value := range []byte(input) {
				if key, ready := decoder.feed(value); ready {
					got = key
				}
			}
			if got != want || decoder.pending() {
				t.Fatalf("input %q: key=%d pending=%v", input, got, decoder.pending())
			}
		})
	}
	var decoder launcherKeyDecoder
	decoder.feed(27)
	if decoder.expire() != launcherBack {
		t.Fatal("standalone Esc did not return")
	}
	for _, input := range []string{"\x1b[999q", "\x1b[" + strings.Repeat("1", 200) + "q"} {
		for _, value := range []byte(input) {
			if key, ready := decoder.feed(value); ready {
				t.Fatalf("unknown/overlong sequence became action %d", key)
			}
			if decoder.length > len(decoder.sequence) {
				t.Fatal("escape buffer grew")
			}
		}
		if decoder.pending() {
			t.Fatal("terminated sequence stayed pending")
		}
	}
	decoder.feed(27)
	decoder.feed('[')
	if decoder.expire() != launcherIdle {
		t.Fatal("incomplete CSI became a navigation action")
	}
}

type launcherDoctorFunc func(context.Context) DoctorReport

func (f launcherDoctorFunc) Diagnose(ctx context.Context) DoctorReport { return f(ctx) }

func newTestLauncher(t *testing.T) (*launcher, *bytes.Buffer) {
	t.Helper()
	var output bytes.Buffer
	root, err := New(&output, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	p := presentationFor(root.OutOrStdout())
	p.output, p.color, p.progress, p.language = "human", "never", "never", "en"
	return &launcher{root: root, p: p, commands: launcherCommands(root), bodyHeight: 4, lastWidth: 80, lastHeight: 24}, &output
}

func TestLauncherSettingsNavigationAndLazyDiagnosis(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HELOX_LANG", "")
	l, output := newTestLauncher(t)
	calls := 0
	if err := AddDoctor(l.root, launcherDoctorFunc(func(context.Context) DoctorReport {
		calls++
		return DoctorReport{Checks: []DoctorCheck{{Name: "release-installation", Detail: "RELEASE_INSTALLATION_UNAVAILABLE"}}}
	})); err != nil {
		t.Fatal(err)
	}
	for _, key := range []launcherKey{launcherSelect, launcherSelect, launcherBack, launcherBack, launcherDown, launcherSelect} {
		if done, err := l.handle(context.Background(), key); done || err != nil {
			t.Fatal(done, err)
		}
	}
	if calls != 0 || l.page != launcherLanguage {
		t.Fatal("opening/help/settings invoked doctor", calls, l.page)
	}
	l.selected[launcherLanguage] = 2 // Korean preference.
	if _, err := l.handle(context.Background(), launcherSelect); err != nil {
		t.Fatal(err)
	}
	if saved, err := readLanguageSettings(); err != nil || saved != "ko" || l.p.language != "ko" {
		t.Fatal("menu failed to persist/select Korean", saved, l.p.language, err)
	}
	l.handle(context.Background(), launcherBack)
	l.handle(context.Background(), launcherDown)
	if _, err := l.handle(context.Background(), launcherSelect); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || l.page != launcherDetail || !strings.Contains(l.detail, "준비 미완료") || strings.Contains(l.detail, "종료 코드는 1") {
		t.Fatal("diagnosis was not explicit/truthful/menu-specific", calls, l.detail)
	}
	if err := l.draw(20, 8); err != nil {
		t.Fatal(err)
	}
	if output.Len() == 0 {
		t.Fatal("no frame was rendered")
	}
}

func TestLauncherLanguageOverrideAutoAndWriteRefusal(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HELOX_LANG", "en")
	l, _ := newTestLauncher(t)
	l.page, l.selected[launcherLanguage] = launcherLanguage, 2
	if _, err := l.choose(context.Background()); err != nil {
		t.Fatal(err)
	}
	if saved, _ := readLanguageSettings(); saved != "ko" || l.p.language != "en" || !strings.Contains(l.notice, "override") {
		t.Fatal("saved preference bypassed environment override", saved, l.p.language, l.notice)
	}
	l.selected[launcherLanguage] = 0
	l.choose(context.Background())
	if saved, _ := readLanguageSettings(); saved != "auto" {
		t.Fatal("auto was persisted as a detected language")
	}
	path, err := languageSettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(t.TempDir(), "foreign")
	if err := os.WriteFile(foreign, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(foreign, path); err != nil {
		t.Fatal(err)
	}
	l.selected[launcherLanguage] = 2
	if _, err := l.choose(context.Background()); err != nil || !strings.Contains(l.notice, "LANGUAGE_SETTINGS_WRITE_FAILED") {
		t.Fatal("failed write was not reported", err, l.notice)
	}
	if body, _ := os.ReadFile(foreign); string(body) != "preserve" {
		t.Fatal("settings action changed a foreign target")
	}
}

func TestLauncherCancelledDiagnosisNeverDisplaysHealthy(t *testing.T) {
	l, _ := newTestLauncher(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l.p.doctor = launcherDoctorFunc(func(context.Context) DoctorReport {
		cancel()
		return DoctorReport{Healthy: true}
	})
	if err := l.diagnose(ctx); !errors.Is(err, context.Canceled) || l.page == launcherDetail {
		t.Fatal("cancelled diagnosis displayed a result", err, l.detail)
	}
}

func TestLauncherExamplesUseRegisteredSyntax(t *testing.T) {
	// Sync examples declare source files. Supply inert complete input fixtures,
	// while still validating the actual displayed syntax without execution.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	if err := os.WriteFile("package.json", []byte(`{"dependencies":{"is-number":"6.0.0"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"requirements.txt": "packaging==25.0\n", "empty.txt": "# explicit empty\n", "pylock.toml": "lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=3.14'\npackages=[]\n"} {
		if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// The displayed pip examples explicitly require an active virtual
	// environment. Supply that declared prerequisite without running Python.
	t.Setenv("VIRTUAL_ENV", t.TempDir())
	l, _ := newTestLauncher(t)
	executions := 0
	for _, command := range l.commands {
		command.RunE = func(*cobra.Command, []string) error { executions++; return nil }
	}
	for i, command := range l.commands {
		l.page, l.selected[launcherUsage] = launcherUsage, i
		if _, err := l.choose(context.Background()); err != nil {
			t.Fatal(command.CommandPath(), err)
		}
		for _, example := range launcherExamples(command.CommandPath()) {
			args := strings.Fields(example)
			// Bind the example's declared existing project to this inert fixture.
			for index, arg := range args {
				if arg == "/absolute/path/project" {
					args[index] = root
				}
			}
			fresh, err := New(io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if err := SetPNPMClientVersion(fresh, runtimeidentity.PNPMVersion); err != nil {
				t.Fatal(err)
			}
			normalized, err := normalizeTerraformInitArgs(fresh, args[1:])
			if err != nil {
				t.Fatal(example, err)
			}
			found, remaining, err := fresh.Find(normalized)
			if err != nil {
				t.Fatal(example, err)
			}
			found.InitDefaultHelpFlag()
			if err := found.ParseFlags(remaining); err != nil {
				t.Fatal(example, err)
			}
			if help, _ := found.Flags().GetBool("help"); !help {
				if err := found.ValidateArgs(found.Flags().Args()); err != nil {
					t.Fatal(example, err)
				}
			}
		}
	}
	if executions != 0 {
		t.Fatal("examples executed a command")
	}
}

type launcherNeverReader struct{}

func (launcherNeverReader) Read([]byte) (int, error) {
	panic("noninteractive launcher must not read input")
}

func TestLauncherRedirectedInputNeverWaits(t *testing.T) {
	l, output := newTestLauncher(t)
	l.root.SetIn(launcherNeverReader{})
	l.root.SetArgs([]string{})
	if err := l.root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "helox [") || strings.Contains(output.String(), "\x1b") {
		t.Fatal("redirected input changed static help", output.String())
	}
}

func TestLauncherMalformedSettingsResetAndFlagOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HELOX_LANG", "")
	if err := writeLanguageSettings("en"); err != nil {
		t.Fatal(err)
	}
	path, err := languageSettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	l, _ := newTestLauncher(t)
	l.selected[launcherHome] = 1
	if _, err := l.choose(context.Background()); err != nil || !strings.Contains(l.notice, "reset") {
		t.Fatal("malformed settings did not expose reset", err, l.notice)
	}
	l.p.locale, l.p.localeExplicit = "en", true
	l.selected[launcherLanguage] = 2
	if _, err := l.choose(context.Background()); err != nil {
		t.Fatal(err)
	}
	if saved, err := readLanguageSettings(); err != nil || saved != "ko" || l.p.language != "en" || l.p.languageSource != "FLAG" {
		t.Fatal("reset or one-shot flag precedence lost", saved, l.p.language, l.p.languageSource, err)
	}
}

func TestLauncherBoundsTranslationsAndNoColor(t *testing.T) {
	l, output := newTestLauncher(t)
	for _, language := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		l.p.language = language
		for _, item := range l.items() {
			if item == "" {
				t.Fatal("empty core translation", language)
			}
		}
		for _, size := range [][2]int{{80, 24}, {43, 30}, {44, 27}, {44, 28}, {44, 30}, {67, 24}, {68, 23}, {68, 24}, {111, 24}, {112, 18}, {160, 80}, {20, 8}, {2, 2}} {
			output.Reset()
			if err := l.draw(size[0], size[1]); err != nil {
				t.Fatal(err)
			}
			frame := strings.TrimPrefix(output.String(), "\x1b[H\x1b[2J")
			lines := strings.Split(frame, "\r\n")
			if len(lines) > size[1] {
				t.Fatal("frame exceeds terminal height", size)
			}
			for _, line := range lines {
				cells := 0
				for _, r := range line {
					cells += displayWidth(r)
				}
				if cells > size[0]-1 || strings.Contains(line, "\x1b") {
					t.Fatal("frame exceeds width or emits SGR in never mode", language, size, line)
				}
			}
		}
	}
	body := "한글 日本語 e\u0301 " + strings.Repeat("long ", 100) + "\x1b[2J"
	for _, line := range launcherWrap(body, 19) {
		cells := 0
		for _, r := range line {
			cells += displayWidth(r)
		}
		if cells > 19 || strings.Contains(line, "\x1b") {
			t.Fatal("unsafe or overflowing wrapped help", line)
		}
	}
	var bounded launcherBuffer
	if _, err := io.Copy(&bounded, strings.NewReader(strings.Repeat("x", (64<<10)+1))); err == nil || bounded.buffer.Len() > 64<<10 {
		t.Fatal("help bound bypassed via io.Copy")
	}
}

func TestLauncherBrandColorPolicyAndMenuSpace(t *testing.T) {
	l, output := newTestLauncher(t)
	l.p.tty = true
	sgr := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	for _, size := range [][2]int{{80, 24}, {44, 30}, {112, 18}} {
		for _, notice := range []string{"", "A notice that wraps across several lines when the terminal is narrow; navigation must remain available."} {
			l.notice = notice
			l.p.color = "never"
			output.Reset()
			if err := l.draw(size[0], size[1]); err != nil {
				t.Fatal(err)
			}
			plain := output.String()
			if l.bodyHeight < 4 || !strings.Contains(plain, "Heliopause Artifact Airlock") || !strings.Contains(plain, "helox") {
				t.Fatal("brand obscures product, command or menu", size, l.bodyHeight)
			}
			for _, mode := range []struct {
				color, term, noColor string
				colored              bool
			}{
				{"auto", "xterm-256color", "", true},
				{"auto", "xterm-256color", "1", false},
				{"auto", "dumb", "", false},
				{"never", "xterm-256color", "", false},
				{"always", "dumb", "1", true},
			} {
				t.Setenv("TERM", mode.term)
				t.Setenv("NO_COLOR", mode.noColor)
				l.p.color = mode.color
				output.Reset()
				if err := l.draw(size[0], size[1]); err != nil {
					t.Fatal(err)
				}
				frame := output.String()
				if sgr.MatchString(frame) != mode.colored || sgr.ReplaceAllString(frame, "") != plain {
					t.Fatal("color changes geometry or ignores terminal preference", size, mode)
				}
				for _, line := range strings.Split(frame, "\r\n") {
					if sgr.MatchString(line) && !strings.HasSuffix(line, "\x1b[0m") {
						t.Fatal("background can bleed beyond a painted row")
					}
				}
			}
		}
	}
	l.notice, l.p.color = "", "never"
	for _, page := range []launcherPage{launcherLanguage, launcherUsage, launcherDetail} {
		l.page = page
		output.Reset()
		if err := l.draw(112, 30); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(output.String(), "█") {
			t.Fatal("large home brand leaked into a working page", page)
		}
	}
}

type launcherFaultTerminal struct {
	keys       []launcherKey
	inputErr   error
	closeErr   error
	sizeErr    error
	closeCalls int
}

func (t *launcherFaultTerminal) Next(context.Context) (launcherKey, error) {
	if len(t.keys) == 0 {
		return launcherIdle, t.inputErr
	}
	key := t.keys[0]
	t.keys = t.keys[1:]
	return key, nil
}
func (t *launcherFaultTerminal) Size() (int, int, error) { return 80, 24, t.sizeErr }
func (t *launcherFaultTerminal) Close() error            { t.closeCalls++; return t.closeErr }

type launcherFaultWriter struct {
	buffer bytes.Buffer
	failOn int
	calls  int
	cause  error
}

func (w *launcherFaultWriter) Write(body []byte) (int, error) {
	w.calls++
	if w.calls == w.failOn {
		return 0, w.cause
	}
	return w.buffer.Write(body)
}

func TestLauncherAlwaysRestoresAndPreservesPrimaryError(t *testing.T) {
	primary, cleanup := errors.New("input fixture"), errors.New("restore fixture")
	for _, mode := range []string{"normal", "read", "EOF", "size", "first-write", "frame-write", "cancel", "cleanup"} {
		t.Run(mode, func(t *testing.T) {
			l, _ := newTestLauncher(t)
			terminal := &launcherFaultTerminal{keys: []launcherKey{launcherQuit}}
			writer := &launcherFaultWriter{cause: primary}
			l.p.out = writer
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var want error
			switch mode {
			case "read", "EOF":
				terminal.keys, terminal.inputErr = nil, primary
				if mode == "EOF" {
					terminal.inputErr = io.EOF
				}
				want = terminal.inputErr
			case "size":
				terminal.sizeErr, want = primary, primary
			case "first-write":
				writer.failOn, want = 1, primary
			case "frame-write":
				writer.failOn, want = 2, primary
			case "cancel":
				cancel()
				want = context.Canceled
			case "cleanup":
				terminal.closeErr, want = cleanup, cleanup
			}
			err := runLauncher(ctx, l.root, l.p, terminal)
			if (want == nil && err != nil) || (want != nil && !errors.Is(err, want)) {
				t.Fatal("primary cause lost", mode, err)
			}
			if terminal.closeCalls != 1 || !strings.Contains(writer.buffer.String(), "\x1b[?25h\x1b[?1049l") {
				t.Fatal("terminal/cursor restoration omitted", mode, terminal.closeCalls, writer.buffer.String())
			}
		})
	}
}
