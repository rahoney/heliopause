//go:build linux || darwin

package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

const launcherPTYHelper = "HELOX_LAUNCHER_PTY_HELPER"

// /usr/bin/script is the explicit native OS PTY test driver (util-linux on
// Linux, BSD on Darwin). No private corpus, network, Python, cgo or skip path.
func TestLauncherNativePTY(t *testing.T) {
	for _, language := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		t.Run(language, func(t *testing.T) { runLauncherPTYFixture(t, language, "normal") })
	}
	for _, mode := range []string{"navigate", "narrow", "resize", "cancel", "interrupt", "doctor-interrupt", "write-failure", "EOF", "CI", "dumb", "json", "help"} {
		t.Run(mode, func(t *testing.T) { runLauncherPTYFixture(t, "en", mode) })
	}
}

type launcherPTYCapture struct {
	mu            sync.Mutex
	buffer        bytes.Buffer
	ready         chan struct{}
	doctorReady   chan struct{}
	once          sync.Once
	doctorOnce    sync.Once
	footer        string
	resizedFooter string
}

func (w *launcherPTYCapture) Write(body []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(body) > (512<<10)-w.buffer.Len() {
		return 0, errors.New("PTY fixture output exceeds bound")
	}
	n, err := w.buffer.Write(body)
	// Input must follow the whole current frame, not the alternate-screen
	// preamble. In particular, VINTR may flush unread PTY output. A previous
	// frame's footer must not release cancellation during a new diagnosis.
	if start := bytes.LastIndex(w.buffer.Bytes(), []byte("\x1b[H\x1b[2J")); start >= 0 {
		frame := w.buffer.Bytes()[start:]
		complete := w.footer != "" && bytes.Contains(frame, []byte(w.footer))
		if w.resizedFooter != "" && bytes.Contains(frame, []byte(w.resizedFooter)) {
			complete = true
		}
		if complete {
			w.once.Do(func() { close(w.ready) })
			if bytes.Contains(frame, []byte("Checking environment")) {
				w.doctorOnce.Do(func() { close(w.doctorReady) })
			}
		}
	}
	return n, err
}

func (w *launcherPTYCapture) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.String()
}

func TestLauncherPTYCaptureWaitsForCompleteFrame(t *testing.T) {
	const footer = "↑↓  Enter  Esc  q"
	w := &launcherPTYCapture{ready: make(chan struct{}), doctorReady: make(chan struct{}), footer: footer}
	write := func(body string) {
		t.Helper()
		if n, err := io.WriteString(w, body); err != nil || n != len(body) {
			t.Fatalf("capture write: %d %v", n, err)
		}
	}
	notReady := func() {
		t.Helper()
		select {
		case <-w.ready:
			t.Fatal("PTY input released before a complete frame was captured")
		default:
		}
	}
	write("\x1b[?1049h\x1b[?25l")
	notReady()
	write("\x1b[H\x1b[2JHeliopause Artifact Airlock\r\n")
	notReady()
	write(footer[:len(footer)-1])
	notReady()
	write(footer[len(footer)-1:])
	select {
	case <-w.ready:
	default:
		t.Fatal("complete split footer did not release input")
	}
}

func TestLauncherPTYCaptureDiagnosisWaitsForOwnFrame(t *testing.T) {
	const footer = "↑↓ move  Enter select  Esc back  q exit"
	w := &launcherPTYCapture{ready: make(chan struct{}), doctorReady: make(chan struct{}), footer: footer}
	for _, body := range []string{
		"\x1b[?1049h\x1b[?25l\x1b[H\x1b[2Jhome\r\n" + footer,
		"\x1b[H\x1b[2JChecking environment…\r\n",
		footer[:len(footer)-1],
	} {
		if _, err := io.WriteString(w, body); err != nil {
			t.Fatal(err)
		}
		select {
		case <-w.doctorReady:
			t.Fatal("diagnosis input borrowed a previous or partial frame footer")
		default:
		}
	}
	if _, err := io.WriteString(w, footer[len(footer)-1:]); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.doctorReady:
	default:
		t.Fatal("complete diagnosis frame did not release cancellation")
	}
}

func TestLauncherPTYCaptureAcceptsCompletedResizedFrame(t *testing.T) {
	const narrow = "↑↓  Enter  Esc  q"
	w := &launcherPTYCapture{
		ready: make(chan struct{}), doctorReady: make(chan struct{}),
		footer: "↑↓ move  Enter select  Esc back  q exit", resizedFooter: narrow,
	}
	// A redraw can replace the first frame before the driver delivers it.
	if _, err := io.WriteString(w, "\x1b[?1049h\x1b[?25l\x1b[H\x1b[2Jwide\x1b[H\x1b[2Jnarrow\r\n"+narrow); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.ready:
	default:
		t.Fatal("complete resized frame was ignored")
	}
}

func runLauncherPTYFixture(t *testing.T, language, mode string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	var args []string
	if runtime.GOOS == "darwin" {
		args = []string{"-q", "/dev/null", executable, "-test.run=^TestLauncherPTYProcess$", "-test.count=1"}
	} else {
		// The sole shell word is our exact generated test-binary path. No user
		// input or environment value is interpolated into a shell program.
		quoted := "'" + strings.ReplaceAll(executable, "'", "'\\''") + "'"
		args = []string{"-q", "-e", "-c", quoted + " -test.run=^TestLauncherPTYProcess$ -test.count=1", "/dev/null"}
	}
	command := exec.CommandContext(ctx, "/usr/bin/script", args...)
	command.Dir = t.TempDir()
	command.WaitDelay = time.Second
	for _, entry := range os.Environ() {
		name := strings.SplitN(entry, "=", 2)[0]
		switch name {
		case "CI", "TERM", "NO_COLOR", "HELOX_LANG", launcherPTYHelper,
			"HELOX_LAUNCHER_PTY_CONFIG", "HELOX_LAUNCHER_PTY_LANGUAGE":
			continue
		}
		command.Env = append(command.Env, entry)
	}
	privateConfig := t.TempDir()
	command.Env = append(command.Env, launcherPTYHelper+"="+mode,
		"HELOX_LAUNCHER_PTY_CONFIG="+privateConfig,
		"HELOX_LAUNCHER_PTY_LANGUAGE="+language, "TERM=xterm-256color", "NO_COLOR=1")
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	footer := (&presentation{language: language}).text("↑↓ move  Enter select  Esc back  q exit", "↑↓ 이동  Enter 선택  Esc 뒤로  q 종료")
	if mode == "narrow" {
		footer = "↑↓  Enter  Esc  q"
	}
	output := &launcherPTYCapture{ready: make(chan struct{}), doctorReady: make(chan struct{}), footer: footer}
	if mode == "resize" {
		output.resizedFooter = "↑↓  Enter  Esc  q"
	}
	command.Stdout, command.Stderr = output, output
	if err := command.Start(); err != nil {
		t.Fatal("native PTY driver unavailable:", err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	interactive := mode == "normal" || mode == "navigate" || mode == "narrow" || mode == "resize" || mode == "interrupt" || mode == "doctor-interrupt"
	if interactive {
		select {
		case <-output.ready:
			keys := "q"
			if mode == "navigate" || mode == "narrow" {
				keys = "\r\r\x1b[6~\x1b[D\x1b[D\x1b[B\r\x1b[H\x1b[B\x1b[B\r\x1b[D\x1b[B\r\x1b[D\x1b[F\r"
			}
			if mode == "interrupt" {
				keys = "\x03"
			}
			if mode == "doctor-interrupt" {
				keys = "\x1b[B\x1b[B\r"
			}
			if _, err := io.WriteString(input, keys); err != nil {
				cancel()
				<-done
				t.Fatal("write PTY keys:", err)
			}
			if mode == "doctor-interrupt" {
				select {
				case <-output.doctorReady:
					if _, err := io.WriteString(input, "\x03"); err != nil {
						cancel()
						<-done
						t.Fatal(err)
					}
				case err := <-done:
					t.Fatalf("diagnosis exited before cancellation: %v\n%s", err, output.String())
				case <-ctx.Done():
					cancel()
					<-done
					t.Fatal("explicit diagnosis did not become ready")
				}
			}
		case err := <-done:
			t.Fatalf("PTY process exited before menu: %v\n%s", err, output.String())
		case <-ctx.Done():
			cancel()
			<-done
			t.Fatal("PTY menu did not become ready")
		}
	}
	err = <-done
	if err != nil || ctx.Err() != nil {
		t.Fatalf("PTY %s/%s failed: %v context=%v\n%s", language, mode, err, ctx.Err(), output.String())
	}
	text := output.String()
	if !strings.Contains(text, "LAUNCHER_PTY_RESTORED") {
		// This private witness is diagnostic only: never substitute it for
		// captured output, termios, cursor/screen or process-success assertions.
		completed, readErr := os.ReadFile(filepath.Join(privateConfig, "pty-completed"))
		t.Fatalf("PTY helper did not validate original terminal state: child_completion=%q error=%v\n%s", completed, readErr, text)
	}
	excluded := mode == "CI" || mode == "dumb" || mode == "json" || mode == "help"
	if excluded {
		if strings.Contains(text, "\x1b[?1049h") || !strings.Contains(text, "helox [") {
			t.Fatal("noninteractive invocation entered launcher", text)
		}
		return
	}
	if !strings.Contains(text, "\x1b[?25h\x1b[?1049l") {
		t.Fatal("PTY cursor/screen was not restored", text)
	}
	if regexp.MustCompile("\x1b\\[[0-9;]*m").MatchString(text) {
		t.Fatal("NO_COLOR PTY emitted SGR", text)
	}
	if mode == "normal" {
		p := &presentation{language: language}
		if !strings.Contains(text, p.text("Usage and examples", "사용법과 예시")) {
			t.Fatal("localized core menu absent", language, text)
		}
	}
	if mode == "navigate" && (!strings.Contains(text, "사용법과 예시") || !strings.Contains(text, "준비 미완료")) {
		t.Fatal("PTY language save/explicit unhealthy diagnosis not rendered", text)
	}
	if mode == "resize" && strings.Count(text, "\x1b[H\x1b[2J") < 2 {
		t.Fatal("live PTY window change did not redraw", text)
	}
}

type launcherEOFPTY struct{ launcherTerminal }

func (t launcherEOFPTY) Next(context.Context) (launcherKey, error) { return launcherIdle, io.EOF }

type launcherResizePTY struct {
	launcherTerminal
	resized bool
}

func (t *launcherResizePTY) Next(ctx context.Context) (launcherKey, error) {
	if !t.resized {
		t.resized = true
		err := unix.IoctlSetWinsize(int(os.Stdout.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Col: 20, Row: 8})
		return launcherIdle, err
	}
	return t.launcherTerminal.Next(ctx)
}

func TestLauncherPTYProcess(t *testing.T) {
	mode := os.Getenv(launcherPTYHelper)
	if mode == "" {
		return
	}
	// TestMain intentionally resets display inputs. Restore only this child
	// fixture's private config; never share the parent's real user settings.
	t.Setenv("XDG_CONFIG_HOME", os.Getenv("HELOX_LAUNCHER_PTY_CONFIG"))
	t.Setenv("HELOX_LANG", "")
	t.Setenv("CI", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "1")
	size := &unix.Winsize{Col: 80, Row: 24}
	if mode == "narrow" {
		size.Col, size.Row = 20, 8
	}
	if err := unix.IoctlSetWinsize(int(os.Stdout.Fd()), unix.TIOCSWINSZ, size); err != nil {
		t.Fatal(err)
	}
	original, err := launcherGetTermios(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	root, err := New(os.Stdout, os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	p := presentationFor(root.OutOrStdout())
	language := os.Getenv("HELOX_LAUNCHER_PTY_LANGUAGE")
	if err := writeLanguageSettings(language); err != nil {
		t.Fatal(err)
	}
	calls := 0
	if err := AddDoctor(root, launcherDoctorFunc(func(ctx context.Context) DoctorReport {
		calls++
		if mode == "doctor-interrupt" {
			<-ctx.Done()
			return DoctorReport{Healthy: true}
		}
		return DoctorReport{Checks: []DoctorCheck{{Name: "release-installation", Detail: "RELEASE_INSTALLATION_UNAVAILABLE"}}}
	})); err != nil {
		t.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	root.SetArgs([]string{})
	var want error
	switch mode {
	case "CI":
		t.Setenv("CI", "1")
	case "dumb":
		t.Setenv("TERM", "dumb")
	case "json":
		root.SetArgs([]string{"--output", "json"})
	case "help":
		root.SetArgs([]string{"--help"})
	}
	if mode == "cancel" || mode == "write-failure" || mode == "EOF" || mode == "resize" {
		p.output = "human"
		if err := p.configure(); err != nil {
			t.Fatal(err)
		}
		terminal, err := openLauncherTerminal(os.Stdin, os.Stdout)
		if err != nil {
			t.Fatal(err)
		}
		if mode == "cancel" {
			cancelCtx, cancel := context.WithCancel(ctx)
			timer := time.AfterFunc(100*time.Millisecond, cancel)
			defer timer.Stop()
			defer cancel()
			err, want = runLauncher(cancelCtx, root, p, terminal), context.Canceled
		} else if mode == "write-failure" {
			cause := errors.New("PTY output fixture")
			// Fail one frame; allow the following restoration write through to
			// the real PTY so the parent can independently observe cleanup.
			writer := &launcherPTYFailWriter{output: os.Stdout, cause: cause}
			p.out = writer
			err, want = runLauncher(ctx, root, p, terminal), cause
		} else if mode == "EOF" {
			err, want = runLauncher(ctx, root, p, launcherEOFPTY{terminal}), io.EOF
		} else {
			err = runLauncher(ctx, root, p, &launcherResizePTY{launcherTerminal: terminal})
		}
		if (want == nil && err != nil) || (want != nil && !errors.Is(err, want)) {
			t.Fatal("PTY primary cause lost:", err, want)
		}
	} else {
		err := root.ExecuteContext(ctx)
		if mode == "interrupt" || mode == "doctor-interrupt" {
			if !errors.Is(err, context.Canceled) {
				t.Fatal("keyboard interruption lost:", err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
	}
	restored, err := launcherGetTermios(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal("read restored PTY termios:", err)
	}
	if !launcherTermiosRestored(*original, *restored, runtime.GOOS == "darwin") {
		t.Fatalf("PTY original termios not restored: before=%+v after=%+v", *original, *restored)
	}
	expectedCalls := 0
	if mode == "navigate" || mode == "narrow" {
		expectedCalls = 1
		if saved, err := readLanguageSettings(); err != nil || saved != "ko" {
			t.Fatal("PTY language action did not persist Korean", saved, err)
		}
	}
	if mode == "doctor-interrupt" {
		expectedCalls = 1
	}
	if calls != expectedCalls {
		t.Fatal("unexpected doctor activity", mode, calls)
	}
	if _, err := fmt.Fprintln(os.Stdout, "\nLAUNCHER_PTY_RESTORED"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(os.Getenv("HELOX_LAUNCHER_PTY_CONFIG"), "pty-completed"), []byte("terminal and doctor assertions completed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

type launcherPTYFailWriter struct {
	output io.Writer
	cause  error
	calls  int
}

func (w *launcherPTYFailWriter) Write(body []byte) (int, error) {
	w.calls++
	if w.calls == 2 {
		return 0, w.cause
	}
	return w.output.Write(body)
}
