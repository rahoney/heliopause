package cli

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// SetLanguageDetector injects an optional display-only OS preference read. It
// runs at most once per command and never participates in policy or execution.
func SetLanguageDetector(root *cobra.Command, detect func() ([]string, string, error)) error {
	if root == nil || detect == nil {
		return errors.New("language detector requires command and callback")
	}
	writer, ok := root.OutOrStdout().(*presentationWriter)
	if !ok {
		return errors.New("command presentation is unavailable")
	}
	writer.presentation.detectLanguage = detect
	return nil
}
func (p *presentation) automaticLanguage() string {
	if p.detectLanguage != nil {
		if !p.languageDetected {
			p.detectedTags, p.detectedSource, p.detectionError = p.detectLanguage()
			p.languageDetected = true
		}
		if p.detectionError == nil {
			for _, tag := range p.detectedTags {
				if language := normalizeLanguage(tag); language != "" && language != "auto" {
					p.languageSource = p.detectedSource
					return language
				}
			}
			p.languageFallback = "OS_LANGUAGE_UNSUPPORTED"
		} else if errors.Is(p.detectionError, context.DeadlineExceeded) {
			p.languageFallback = "OS_LANGUAGE_TIMEOUT"
		} else {
			p.languageFallback = "OS_LANGUAGE_UNAVAILABLE"
		}
	}
	if hints := os.Getenv("LANGUAGE"); len(hints) <= 1024 {
		for _, hint := range strings.Split(hints, ":") {
			if language := normalizeLanguage(hint); language != "" && language != "auto" {
				p.languageSource = "LOCALE_LANGUAGE"
				return language
			}
		}
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(key); value != "" {
			if language := normalizeLanguage(value); language != "" && language != "auto" {
				p.languageSource = "LOCALE_" + key
				return language
			}
			break
		}
	}
	p.languageSource = "ENGLISH_FALLBACK"
	return "en"
}
func (p *presentation) languageSelectionOutput() {
	p.line("%s: %s", p.text("Current language", "현재 언어"), p.language)
	p.line("%s: %s", p.text("Language source", "언어 선택 기준"), safeTerminal(p.languageSource))
	if p.languageFallback != "" {
		p.line("%s: %s", p.text("Fallback reason", "대체 언어 사용 이유"), p.languageFallback)
	}
}

func (p *presentation) prepareLanguageSetting(command *cobra.Command, args []string) {
	p.skipSaved = command.CommandPath() == "helox config language" && len(args) == 1
	p.pendingLanguage = ""
	if p.skipSaved {
		p.pendingLanguage = normalizeLanguage(args[0])
	}
}
