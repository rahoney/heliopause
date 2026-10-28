package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/spf13/cobra"
)

type presentation struct {
	doctor                           Doctor
	out, diagnostic                  io.Writer
	output, locale, color, progress  string
	language                         string
	languageSource, languageFallback string
	pendingLanguage                  string
	detectLanguage                   func() ([]string, string, error)
	languageDetected                 bool
	detectedTags                     []string
	detectedSource                   string
	detectionError                   error
	tty, diagnosticTTY               bool
	width                            int
	verbose                          bool
	localeExplicit, skipSaved        bool
	warnedSettings                   bool
	writeErr                         error
	reported                         bool
	startedAt                        time.Time
	reportedError                    bool
	selfUpdate                       bool
	interruption                     string
}
type presentationWriter struct {
	io.Writer
	presentation *presentation
}

func attachPresentation(root *cobra.Command, stdout, stderr io.Writer) {
	p := &presentation{out: stdout, diagnostic: stderr, output: "auto", locale: "auto", color: "auto", progress: "auto", language: "en", width: 80}
	if file, ok := stdout.(*os.File); ok {
		_, p.tty = terminalWidth(file)
	}
	if file, ok := stderr.(*os.File); ok {
		p.width, p.diagnosticTTY = terminalWidth(file)
		if p.width == 0 {
			p.width = 80
		}
	}
	root.SetOut(&presentationWriter{Writer: stdout, presentation: p})
	root.SetErr(stderr)
	flags := root.PersistentFlags()
	flags.StringVar(&p.output, "output", "auto", "Output: auto, human or json")
	flags.StringVar(&p.locale, "lang", "auto", "CLI language: auto, en, ko, de, fr, es, pt-BR or ja")
	flags.StringVar(&p.color, "color", "auto", "Color: auto, always or never")
	flags.StringVar(&p.progress, "progress", "auto", "Progress: auto, always or never")
	flags.BoolVar(&p.verbose, "verbose", false, "Show bounded runtime diagnostics")
	root.PersistentPreRunE = func(command *cobra.Command, args []string) error {
		p.localeExplicit = root.PersistentFlags().Changed("lang")
		p.prepareLanguageSetting(command, args)
		return p.configure()
	}
	help := root.HelpFunc()
	addLanguageSettings(root, p)
	root.SetHelpFunc(func(command *cobra.Command, args []string) {
		p.localeExplicit = root.PersistentFlags().Changed("lang")
		if err := p.configure(); err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return
		}
		if p.language == "en" {
			help(command, args)
			return
		}
		localizeHelp(root, p)
		original := command.OutOrStdout()
		var rendered bytes.Buffer
		command.SetOut(&rendered)
		help(command, args)
		command.SetOut(original)
		if rendered.Len() > 64<<10 {
			_, _ = fmt.Fprintln(stderr, "CLI help exceeds display limit")
			return
		}
		_, _ = fmt.Fprint(original, p.localizeHelpText(rendered.String()))
	})
}

// PreparePresentation parses the already-built static command tree before any
// runtime composition, so leading global flags never select the wrong backend.
func PreparePresentation(root *cobra.Command, args []string) (string, error) {
	args, err := normalizeTerraformInitArgs(root, args)
	if err != nil {
		return "", err
	}
	if err := validateGitHubReplacementIntent(root, args); err != nil {
		return "", err
	}
	if err := validateSelfUpdateIntent(root, args); err != nil {
		return "", err
	}
	if err := validateBootstrapPreparationIntent(root, args); err != nil {
		return "", err
	}
	command, remaining, err := root.Find(args)
	if err != nil {
		return "", err
	}
	descriptor, packageClient := packageClientForCommand(command)
	if packageClient && descriptor.validateRaw != nil {
		if err := descriptor.validateRaw(args); err != nil {
			return "", err
		}
	}
	command.InitDefaultHelpFlag()
	if err = command.ParseFlags(remaining); err != nil {
		return "", err
	}
	if !packageClient && relativeCommandPath(command) == "pipx" && len(command.Flags().Args()) != 0 {
		return "", errors.New("pipx supports install and managed upgrade only")
	}
	// Preflight and Execute parse the same argv. Slice flags must not accumulate
	// one accepted occurrence twice; explicit repeats were validated before reset.
	if command.CommandPath() == "helox cargo update" {
		flag := command.Flags().Lookup("package")
		defer func() { _ = flag.Value.(interface{ Replace([]string) error }).Replace(nil); flag.Changed = false }()
	}
	p := presentationFor(root.OutOrStdout())
	top := command
	for top.Parent() != nil && top.Parent() != root {
		top = top.Parent()
	}
	source := ""
	if top != root {
		source = top.Name()
	}
	if root.Annotations == nil {
		root.Annotations = map[string]string{}
	}
	delete(root.Annotations, selectedPackageClientAnnotation)
	delete(root.Annotations, selectedPackageClientOperationAnnotation)
	if packageClient {
		source = descriptor.route
		root.Annotations[selectedPackageClientAnnotation] = string(descriptor.id)
		root.Annotations[selectedPackageClientOperationAnnotation] = descriptor.path[len(descriptor.path)-1]
	}
	p.selfUpdate = source == "self-update"
	p.localeExplicit = root.PersistentFlags().Changed("lang")
	p.prepareLanguageSetting(command, command.Flags().Args())
	if err = p.configure(); err != nil {
		return "", err
	}
	if packageClient || command.CommandPath() == "helox bootstrap prepare" || command.CommandPath() == "helox self-update" || command.CommandPath() == "helox npm update" || command.CommandPath() == "helox pip install" || command.CommandPath() == "helox cargo update" || command.CommandPath() == "helox go get" || command.CommandPath() == "helox terraform init" || command.CommandPath() == "helox github install" {
		help, _ := command.Flags().GetBool("help")
		if !help {
			if err := command.ValidateArgs(command.Flags().Args()); err != nil {
				return "", err
			}
		}
	}
	if p.language != "en" {
		localizeHelp(root, p)
	}
	goUpgrade, _ := command.Flags().GetBool("upgrade")
	if p.output == "json" && ((source == "go" && !(command.CommandPath() == "helox go get" && goUpgrade)) || (source == "cargo" && command.CommandPath() != "helox cargo update") || (source == "terraform" && !goUpgrade) || source == "doctor") {
		return "", errors.New("explicit JSON output is not available for this command")
	}
	if packageClient && descriptor.id == UVPipPackageClient && SelectedPackageClientOperation(root) == "sync" {
		help, _ := command.Flags().GetBool("help")
		if !help {
			skip, err := prepareUVSyncSkip(command, command.Flags().Args())
			if err != nil {
				return "", err
			}
			if skip {
				source = ""
			}
		}
	}
	return source, nil
}

func presentationFor(output io.Writer) *presentation {
	if writer, ok := output.(*presentationWriter); ok {
		return writer.presentation
	}
	return &presentation{out: output, diagnostic: io.Discard, language: "en", output: "human", color: "never", progress: "never", width: 80}
}
func machineOutput(command *cobra.Command) bool {
	return !presentationFor(command.OutOrStdout()).human()
}
func (p *presentation) human() bool { return p.output == "human" || p.output == "auto" && p.tty }

// DiagnosticWriter selects display-only diagnostics; inspection and Evidence are
// independent of this writer and remain enabled in every output mode.
func DiagnosticWriter(command *cobra.Command) io.Writer {
	p := presentationFor(command.OutOrStdout())
	if p.human() && !p.verbose {
		return io.Discard
	}
	return p.diagnostic
}
func (p *presentation) configure() error {
	for _, item := range []struct {
		value   string
		allowed []string
	}{{p.output, []string{"auto", "human", "json"}}, {p.color, []string{"auto", "always", "never"}}, {p.progress, []string{"auto", "always", "never"}}} {
		valid := false
		for _, s := range item.allowed {
			if item.value == s {
				valid = true
			}
		}
		if !valid {
			return errors.New("unsupported CLI presentation option; use --help")
		}
	}
	p.languageSource, p.languageFallback = "FLAG", ""
	language := normalizeLanguage(p.locale)
	if language == "" {
		return errors.New("unsupported language; choose auto, en, ko, de, fr, es, pt-BR or ja")
	}
	if !p.localeExplicit {
		if override := os.Getenv("HELOX_LANG"); override != "" {
			p.languageSource = "ENV_HELOX_LANG"
			language = normalizeLanguage(override)
			if language == "" {
				language = "en"
			}
		} else if p.skipSaved && p.pendingLanguage != "" {
			language, p.languageSource = p.pendingLanguage, "SAVED"
		} else if !p.skipSaved {
			saved, err := readLanguageSettings()
			if err != nil {
				language = "en"
				p.languageSource = "SETTINGS_FALLBACK"
				p.languageFallback = "LANGUAGE_SETTINGS_UNAVAILABLE"
				if !p.warnedSettings {
					_, _ = fmt.Fprintln(p.diagnostic, "Language settings unavailable; using English. Use helox config language auto to reset.")
					p.warnedSettings = true
				}
			} else {
				p.languageSource = "SAVED"
				language = saved
			}
		}
	}
	if language == "auto" {
		language = p.automaticLanguage()
	}
	p.language = language
	return nil
}
func (p *presentation) text(en, ko string) string {
	if p.language == "ko" {
		return ko
	}
	if text, ok := coreTranslations[en]; ok {
		if index := secondaryLanguageIndex(p.language); index >= 0 && text[index] != "" {
			return text[index]
		}
	}
	return en
}
func (p *presentation) line(format string, args ...any) {
	if p.writeErr != nil {
		return
	}
	_, p.writeErr = fmt.Fprintf(p.out, format+"\n", args...)
}
func (p *presentation) paint(text, color string) string {
	enabled := p.color == "always" || p.color != "never" && p.tty && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	if enabled {
		return "\x1b[" + color + "m" + text + "\x1b[0m"
	}
	return text
}

// safeTerminal keeps untrusted references/paths/codes from controlling a terminal.
func safeTerminal(value string) string {
	var b strings.Builder
	for i, r := range []rune(value) {
		if i >= 240 {
			b.WriteString("…")
			break
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			fmt.Fprintf(&b, "\\u%04x", r)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
func displayWidth(r rune) int {
	if unicode.Is(unicode.Mn, r) {
		return 0
	}
	if r >= 0x1100 && (r <= 0x115f || r >= 0x2e80 && r <= 0xa4cf || r >= 0xac00 && r <= 0xd7af || r >= 0xf900 && r <= 0xfaff || r >= 0xfe10 && r <= 0xfe6f || r >= 0xff00 && r <= 0xff60 || r >= 0x1f300) {
		return 2
	}
	return 1
}
func fitTerminal(value string, width int) string {
	var b strings.Builder
	used := 0
	for _, r := range value {
		w := displayWidth(r)
		if used+w > width {
			break
		}
		b.WriteRune(r)
		used += w
	}
	return b.String()
}

func beginProgress(ctx context.Context, output io.Writer, machine bool) (context.Context, func() error) {
	if machine {
		return ctx, func() error { return nil }
	}
	p := presentationFor(output)
	p.startedAt = time.Now()
	animate := p.diagnosticTTY && os.Getenv("TERM") != "dumb"
	if p.progress == "never" || p.progress == "auto" && !animate {
		return ctx, func() error { return nil }
	}
	var mu sync.Mutex
	event := ports.Progress{Stage: "prepare"}
	started := time.Now()
	frame := 0
	var writeErr error
	closed := false
	draw := func() {
		if writeErr != nil {
			return
		}
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}[frame%10]
		frame++
		line := fmt.Sprintf("%s %s  %.1fs", spinner, p.stage(event.Stage), time.Since(started).Seconds())
		if event.Unit == "bytes" {
			if event.Total > 0 {
				ratio := float64(event.Current) / float64(event.Total)
				if ratio < 0 {
					ratio = 0
				}
				if ratio > 1 {
					ratio = 1
				}
				filled := int(ratio * 16)
				line = fmt.Sprintf("%s %s [%s%s] %3.0f%%  %s / %s  %.1fs", spinner, p.stage(event.Stage), strings.Repeat("█", filled), strings.Repeat("░", 16-filled), ratio*100, byteSize(event.Current), byteSize(event.Total), time.Since(started).Seconds())
			} else {
				line += fmt.Sprintf("  %s", byteSize(event.Current))
			}
		} else if event.Total > 0 {
			line += fmt.Sprintf("  %d / %d", event.Current, event.Total)
		}
		if animate {
			_, writeErr = fmt.Fprintf(p.diagnostic, "\r\x1b[2K%s", p.paint(fitTerminal(line, p.width-1), "36"))
		} else {
			_, writeErr = fmt.Fprintln(p.diagnostic, line)
		}
	}
	observe := func(next ports.Progress) {
		if !knownStage(next.Stage) {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return
		}
		changed := next.Stage != event.Stage
		event = next
		if changed {
			started = time.Now()
		}
		if changed || animate && next.Total > 0 && next.Current == next.Total {
			draw()
		}
	}
	stop, done := make(chan struct{}), make(chan struct{})
	if animate {
		go func() {
			defer close(done)
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					mu.Lock()
					draw()
					mu.Unlock()
				}
			}
		}()
	} else {
		close(done)
	}
	var once sync.Once
	finish := func() error {
		once.Do(func() {
			close(stop)
			<-done
			mu.Lock()
			defer mu.Unlock()
			closed = true
			if animate {
				_, err := fmt.Fprint(p.diagnostic, "\r\x1b[2K")
				writeErr = errors.Join(writeErr, err)
			}
		})
		return writeErr
	}
	return ports.WithProgress(ctx, observe), finish
}
func knownStage(stage string) bool {
	switch stage {
	case "prepare", "resolve", "download", "verify", "inspect", "policy", "manifest", "stage", "install", "build", "evidence", "authorize", "drain", "activate":
		return true
	}
	return false
}
func (p *presentation) stage(stage string) string {
	switch stage {
	case "authorize":
		return p.text("Preparing protected installer", "관리자 설치 도우미 준비")
	case "drain":
		return p.text("Finishing runtime cleanup", "검사 실행 환경 종료·정리")
	case "activate":
		return p.text("Activating release and checking runtime health", "릴리스 전환·실행 환경 정상 확인")
	}
	names := map[string][2]string{"prepare": {"Preparing operation", "작업 준비"}, "resolve": {"Resolving exact dependencies", "정확한 버전·의존성 확인"}, "download": {"Downloading artifact", "패키지 다운로드"}, "verify": {"Verifying integrity", "출처·무결성 검증"}, "inspect": {"Static / sandbox inspection", "정적·격리 환경 검사"}, "policy": {"Evaluating policy", "정책 판정"}, "manifest": {"Recording manifest / SBOM", "검증 목록·SBOM 생성"}, "stage": {"Staging verified files", "검증 파일 반입 준비"}, "install": {"Installing verified files", "검증 파일 설치"}, "build": {"Building in sandbox", "격리 환경 빌드"}, "evidence": {"Recording evidence", "검사 근거 기록"}}
	value, ok := names[stage]
	if !ok {
		return safeTerminal(stage)
	}
	return p.text(value[0], value[1])
}
func byteSize(value int64) string {
	if value < 0 {
		value = 0
	}
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	if value < 1024*1024 {
		return fmt.Sprintf("%.1f KiB", float64(value)/1024)
	}
	return fmt.Sprintf("%.1f MiB", float64(value)/(1024*1024))
}
