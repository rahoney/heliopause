package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
)

type launcherKey uint8

const (
	launcherIdle launcherKey = iota
	launcherUp
	launcherDown
	launcherSelect
	launcherBack
	launcherQuit
	launcherInterrupt
	launcherFirst
	launcherLast
	launcherPageUp
	launcherPageDown
)

// The terminal owns input state; the view owns output state. Neither can
// execute an artifact command. The small interface also permits I/O faults.
type launcherTerminal interface {
	Next(context.Context) (launcherKey, error)
	Size() (int, int, error)
	Close() error
}

func runLauncherOrHelp(command *cobra.Command) error {
	p := presentationFor(command.OutOrStdout())
	input, inputOK := command.InOrStdin().(*os.File)
	output, outputOK := p.out.(*os.File)
	if !inputOK || !outputOK || !p.human() || os.Getenv("CI") != "" ||
		os.Getenv("TERM") == "" || os.Getenv("TERM") == "dumb" {
		return command.Help()
	}
	width, height, usable := launcherGeometry(input, output)
	if !usable || width < 20 || height < 8 {
		return command.Help()
	}
	terminal, err := openLauncherTerminal(input, output)
	if err != nil {
		return err
	}
	return runLauncher(command.Context(), command, p, terminal)
}

type launcherPage uint8

const (
	launcherHome launcherPage = iota
	launcherUsage
	launcherLanguage
	launcherDetail
)

type launcher struct {
	root       *cobra.Command
	p          *presentation
	page       launcherPage
	parent     launcherPage
	selected   [3]int
	offset     int
	commands   []*cobra.Command
	detail     string
	notice     string
	saved      string
	bodyHeight int
	lastWidth  int
	lastHeight int
}

var launcherLanguages = [...]string{"auto", "en", "ko", "de", "fr", "es", "pt-BR", "ja"}

func runLauncher(ctx context.Context, root *cobra.Command, p *presentation, terminal launcherTerminal) (resultErr error) {
	reset := ""
	if p.paint("x", "36") != "x" {
		reset = "\x1b[0m"
	}
	// Register restoration before any output can fail, including a partial
	// alternate-screen/cursor write. Preserve the primary input/output cause.
	defer func() {
		_, outputErr := io.WriteString(p.out, reset+"\x1b[?25h\x1b[?1049l")
		resultErr = errors.Join(resultErr, terminal.Close(), outputErr)
	}()
	if _, err := io.WriteString(p.out, "\x1b[?1049h\x1b[?25l"); err != nil {
		return err
	}
	localizeHelp(root, p)
	view := &launcher{root: root, p: p, bodyHeight: 1}
	view.commands = launcherCommands(root)
	width, height := 0, 0
	dirty := true
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		nextWidth, nextHeight, err := terminal.Size()
		if err != nil {
			return err
		}
		if dirty || width != nextWidth || height != nextHeight {
			width, height = nextWidth, nextHeight
			if err := view.draw(width, height); err != nil {
				return err
			}
			dirty = false
		}
		key, err := terminal.Next(ctx)
		if err != nil {
			return err
		}
		if key == launcherIdle {
			continue
		}
		done, err := view.handle(ctx, key)
		if err != nil || done {
			return err
		}
		dirty = true
	}
}

func launcherCommands(root *cobra.Command) []*cobra.Command {
	commands := []*cobra.Command{root}
	var visit func(*cobra.Command)
	visit = func(parent *cobra.Command) {
		for _, command := range parent.Commands() {
			if command.Hidden {
				continue
			}
			commands = append(commands, command)
			visit(command)
		}
	}
	visit(root)
	return commands
}

func (l *launcher) items() []string {
	switch l.page {
	case launcherHome:
		return []string{
			l.p.text("Usage and examples", "사용법과 예시"),
			l.p.text("Language settings", "언어 설정"),
			l.p.text("Environment diagnostics", "환경 진단"),
			l.p.text("Exit", "종료"),
		}
	case launcherUsage:
		items := make([]string, len(l.commands))
		for i, command := range l.commands {
			items[i] = command.CommandPath()
			switch items[i] {
			case "helox pip install":
				items[i] += "  (--upgrade / -U)"
			case "helox github install":
				items[i] += "  (--replace)"
			}
		}
		return items
	case launcherLanguage:
		return []string{
			l.p.text("Automatic (system language)", "자동 (시스템 언어)"),
			"English · en", "한국어 · ko", "Deutsch · de", "Français · fr",
			"Español · es", "Português (Brasil) · pt-BR", "日本語 · ja",
		}
	}
	return nil
}

func (l *launcher) handle(ctx context.Context, key launcherKey) (bool, error) {
	switch key {
	case launcherQuit:
		return true, nil
	case launcherInterrupt:
		return false, context.Canceled
	case launcherBack:
		if l.page == launcherDetail {
			l.page = l.parent
		} else {
			l.page = launcherHome
		}
		l.offset, l.notice = 0, ""
		return false, nil
	}
	if l.page == launcherDetail {
		switch key {
		case launcherUp:
			l.offset--
		case launcherDown:
			l.offset++
		case launcherPageUp:
			l.offset -= l.bodyHeight
		case launcherPageDown:
			l.offset += l.bodyHeight
		case launcherFirst:
			l.offset = 0
		case launcherLast:
			l.offset = 1 << 20 // draw clamps against the bounded wrapped help.
		}
		return false, nil
	}
	items := l.items()
	selection := &l.selected[l.page]
	switch key {
	case launcherUp:
		*selection--
	case launcherDown:
		*selection++
	case launcherFirst:
		*selection = 0
	case launcherLast:
		*selection = len(items) - 1
	case launcherPageUp:
		*selection -= l.bodyHeight
	case launcherPageDown:
		*selection += l.bodyHeight
	case launcherSelect:
		return l.choose(ctx)
	}
	*selection = max(0, min(*selection, len(items)-1))
	return false, nil
}

func (l *launcher) choose(ctx context.Context) (bool, error) {
	switch l.page {
	case launcherHome:
		switch l.selected[launcherHome] {
		case 0:
			l.page = launcherUsage
		case 1:
			l.page = launcherLanguage
			saved, err := readLanguageSettings()
			if err != nil {
				l.notice = l.p.text("Language settings unavailable. Select a language to reset.", "언어 설정을 읽지 못했습니다. 언어를 선택하면 재설정합니다.")
				saved = "auto"
			}
			l.saved = saved
			for i, language := range launcherLanguages {
				if language == saved {
					l.selected[launcherLanguage] = i
				}
			}
		case 2:
			return false, l.diagnose(ctx)
		case 3:
			return true, nil
		}
	case launcherUsage:
		command := l.commands[l.selected[launcherUsage]]
		body, err := launcherHelp(command)
		if err != nil {
			return false, err
		}
		if examples := launcherExamples(command.CommandPath()); len(examples) > 0 {
			body = l.p.text("Examples", "사용 예시") + "\n\n" + strings.Join(examples, "\n") +
				"\n\n" + l.p.text("Examples are displayed only. Run commands in your shell with the documented project, target and runtime prerequisites.",
				"예시는 표시만 합니다. 문서의 프로젝트·설치 대상·실행 환경 조건을 갖춘 뒤 셸에서 명령을 실행하세요.") + "\n\n" + body
		}
		l.showDetail(body)
	case launcherLanguage:
		language := launcherLanguages[l.selected[launcherLanguage]]
		if err := writeLanguageSettings(language); err != nil {
			l.notice = l.p.text("Language was not saved. Use helox config language to check settings.", "언어를 저장하지 못했습니다. helox config language로 설정을 확인하세요.") + " [LANGUAGE_SETTINGS_WRITE_FAILED]"
			return false, nil
		}
		l.p.skipSaved, l.p.pendingLanguage = false, ""
		if err := l.p.configure(); err != nil {
			return false, err
		}
		l.saved = language
		localizeHelp(l.root, l.p)
		l.notice = l.p.text("Language saved.", "언어를 저장했습니다.")
		if l.p.localeExplicit || os.Getenv("HELOX_LANG") != "" {
			l.notice += " " + l.p.text("The command-line or environment language override still takes precedence.", "명령행 또는 환경 변수의 언어 지정이 우선 적용됩니다.")
		}
	}
	l.offset = 0
	return false, nil
}

func (l *launcher) showDetail(body string) {
	l.parent, l.page, l.detail, l.offset, l.notice = l.page, launcherDetail, body, 0, ""
}

func (l *launcher) diagnose(ctx context.Context) error {
	if l.p.doctor == nil {
		l.showDetail(l.p.text("Environment diagnostics are unavailable.", "환경 진단을 사용할 수 없습니다.") + "\nDOCTOR_UNAVAILABLE")
		return nil
	}
	// Paint the real action before the bounded synchronous service call. No
	// inspection percentage, successful check or background activity is invented.
	l.notice = l.p.text("Checking environment…", "실행 환경을 확인하고 있습니다…")
	if err := l.drawCurrentSize(); err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	report := l.p.doctor.Diagnose(callCtx)
	if err := ctx.Err(); err != nil {
		return err
	}
	if callCtx.Err() != nil {
		l.showDetail(l.p.text("Environment diagnosis did not complete.", "환경 진단을 완료하지 못했습니다.") + "\nDOCTOR_DEADLINE_EXCEEDED")
		return nil
	}
	var buffer launcherBuffer
	p := *l.p
	p.out, p.color, p.writeErr = &buffer, "never", nil
	if err := p.renderDoctorNotice(report, false); err != nil {
		return err
	}
	if buffer.err != nil {
		return buffer.err
	}
	l.showDetail(buffer.buffer.String())
	return nil
}

func (l *launcher) drawCurrentSize() error {
	// The last frame's dimensions are bounded by draw. The next input-loop
	// iteration reads the real terminal geometry again.
	return l.draw(l.lastWidth, l.lastHeight)
}

// launcherBuffer deliberately does not embed bytes.Buffer: io.Copy must not
// acquire a ReaderFrom path that can bypass the display bound.
type launcherBuffer struct {
	buffer bytes.Buffer
	err    error
}

func (b *launcherBuffer) Write(body []byte) (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	if len(body) > (64<<10)-b.buffer.Len() {
		b.err = errors.New("launcher help exceeds display limit")
		return 0, b.err
	}
	return b.buffer.Write(body)
}

func launcherHelp(command *cobra.Command) (string, error) {
	var buffer launcherBuffer
	original := command.OutOrStdout()
	command.SetOut(&buffer)
	defer command.SetOut(original)
	err := command.Help()
	if err = errors.Join(err, buffer.err); err != nil {
		return "", err
	}
	return buffer.buffer.String(), nil
}

func launcherWrap(body string, width int) []string {
	width = max(1, width)
	var lines []string
	var line strings.Builder
	cells := 0
	appendRune := func(r rune) {
		size := displayWidth(r)
		if cells+size > width {
			lines = append(lines, line.String())
			line.Reset()
			cells = 0
		}
		if size <= width {
			line.WriteRune(r)
			cells += size
		}
	}
	for _, r := range body {
		switch {
		case r == '\n':
			lines = append(lines, line.String())
			line.Reset()
			cells = 0
		case r == '\t':
			for range 4 {
				appendRune(' ')
			}
		case unicode.IsControl(r) || unicode.Is(unicode.Cf, r):
			for _, escaped := range fmt.Sprintf("\\u%04x", r) {
				appendRune(escaped)
			}
		default:
			appendRune(r)
		}
	}
	return append(lines, line.String())
}

func (l *launcher) draw(width, height int) error {
	width, height = max(2, min(width, 160)), max(2, min(height, 80))
	l.lastWidth, l.lastHeight = width, height
	cells := width - 1 // Leave the physical last column unused.
	var header []string
	brandRows := 1
	if l.page == launcherHome && height >= 22 && width >= 60 {
		header = append(header,
			"██  ██ ██████ ██      ████  ██  ██",
			"██  ██ ██     ██     ██  ██  ████ ",
			"██████ █████  ██     ██  ██   ██  ",
			"██  ██ ██     ██     ██  ██  ████ ",
			"██  ██ ██████ ██████  ████  ██  ██",
			"")
		brandRows = len(header) + 1 // Include the full product name.
	} else if height >= 18 && width >= 40 {
		header = append(header, "H E L O X", "")
	}
	header = append(header, launcherWrap("Heliopause Artifact Airlock", cells)...)
	if height >= 18 {
		header = append(header, launcherWrap(l.root.Short, cells)...)
	}
	header = append(header, strings.Repeat("─", cells))
	if l.page == launcherLanguage {
		header = append(header, launcherWrap(fmt.Sprintf("%s: %s → %s (%s)",
			l.p.text("Language", "언어"), l.saved, l.p.language, l.p.languageSource), cells)...)
	}
	if l.notice != "" {
		notice := launcherWrap(l.notice, cells)
		header = append(header, notice[:min(2, len(notice))]...)
	}
	if len(header) > height-2 {
		header = header[:height-2]
	}
	l.bodyHeight = max(1, height-len(header)-1)
	var body []string
	selected := -1
	if l.page == launcherDetail {
		body = launcherWrap(l.detail, cells)
		l.offset = max(0, min(l.offset, len(body)-l.bodyHeight))
	} else {
		items := l.items()
		selected = l.selected[l.page]
		for i, item := range items {
			prefix := "  "
			if i == selected {
				prefix = "› "
			}
			body = append(body, fitTerminal(prefix+item, cells))
		}
		if selected < l.offset {
			l.offset = selected
		}
		if selected >= l.offset+l.bodyHeight {
			l.offset = selected - l.bodyHeight + 1
		}
		l.offset = max(0, min(l.offset, max(0, len(body)-l.bodyHeight)))
	}
	lines := make([]string, 0, height)
	for i, line := range header {
		if i < brandRows {
			line = l.p.paint(line, "1;36")
		}
		lines = append(lines, line)
	}
	for i := 0; i < l.bodyHeight; i++ {
		line := ""
		index := l.offset + i
		if index < len(body) {
			line = body[index]
		}
		if index == selected {
			line = l.p.paint(line, "1;36")
		}
		lines = append(lines, line)
	}
	hint := "↑↓  Enter  Esc  q"
	if cells >= 60 {
		hint = l.p.text("↑↓ move  Enter select  Esc back  q exit", "↑↓ 이동  Enter 선택  Esc 뒤로  q 종료")
	}
	lines = append(lines, fitTerminal(hint, cells))
	_, err := io.WriteString(l.p.out, "\x1b[H\x1b[2J"+strings.Join(lines, "\r\n"))
	return err
}
