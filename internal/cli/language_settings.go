package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type languageSettings struct {
	Language string `json:"language"`
}

func normalizeLanguage(value string) string {
	value = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", "-"))
	if i := strings.IndexAny(value, ".@"); i >= 0 {
		value = value[:i]
	}
	switch value {
	case "auto":
		return "auto"
	case "pt", "pt-br":
		return "pt-BR"
	}
	for _, lang := range []string{"en", "ko", "de", "fr", "es", "ja"} {
		if value == lang || strings.HasPrefix(value, lang+"-") {
			return lang
		}
	}
	return ""
}
func languageSettingsPath() (string, error) {
	root := os.Getenv("XDG_CONFIG_HOME")
	var err error
	if root == "" {
		root, err = os.UserConfigDir()
	}
	if err != nil || !filepath.IsAbs(root) {
		return "", errors.New("user language settings directory is unavailable")
	}
	return filepath.Join(root, "heliopause", "cli.json"), nil
}
func readLanguageSettings() (string, error) {
	path, err := languageSettingsPath()
	if err != nil {
		// A missing user config location (for example a minimal CI environment)
		// means no saved display preference, not unavailable CLI execution.
		return "auto", nil
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "auto", nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 1024 {
		return "", errors.New("invalid CLI language settings; use helox config language auto to reset")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", errors.New("cannot read CLI language settings")
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 1025))
	decoder.DisallowUnknownFields()
	var settings languageSettings
	if decoder.Decode(&settings) != nil || decoder.Decode(&struct{}{}) != io.EOF || normalizeLanguage(settings.Language) == "" {
		return "", errors.New("invalid CLI language settings; use helox config language auto to reset")
	}
	return normalizeLanguage(settings.Language), nil
}
func writeLanguageSettings(language string) error {
	normalized := normalizeLanguage(language)
	if normalized == "" {
		return errors.New("unsupported language; choose auto, en, ko, de, fr, es, pt-BR or ja")
	}
	path, err := languageSettingsPath()
	if err != nil {
		return err
	}
	directory := filepath.Dir(path)
	if err = os.MkdirAll(directory, 0700); err != nil {
		return errors.New("cannot create CLI settings directory")
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0022 != 0 {
		return errors.New("CLI settings directory must not be a link or writable by others")
	}
	if existing, statErr := os.Lstat(path); statErr == nil {
		if !existing.Mode().IsRegular() || existing.Mode()&os.ModeSymlink != 0 {
			return errors.New("CLI settings must be a regular file")
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	body, err := json.MarshalIndent(languageSettings{Language: normalized}, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	temporary, err := os.CreateTemp(directory, ".cli-language-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	_, writeErr := io.Copy(temporary, bytes.NewReader(body))
	syncErr := temporary.Sync()
	closeErr := temporary.Close()
	if err = errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func addLanguageSettings(root *cobra.Command, p *presentation) {
	config := &cobra.Command{Use: "config", Short: "Manage CLI display settings"}
	language := &cobra.Command{Use: "language [auto|en|ko|de|fr|es|pt-BR|ja]", Short: "Show or save CLI language", Args: cobra.MaximumNArgs(1)}
	language.RunE = func(command *cobra.Command, args []string) error {
		if len(args) == 1 {
			if err := writeLanguageSettings(args[0]); err != nil {
				return err
			}
			p.skipSaved, p.pendingLanguage = false, ""
			if err := p.configure(); err != nil {
				return err
			}
			p.line("%s: %s", p.text("Saved language", "저장한 언어"), normalizeLanguage(args[0]))
			p.languageSelectionOutput()
			return p.writeErr
		}
		saved, err := readLanguageSettings()
		if err != nil {
			return err
		}
		p.line("%s: %s", p.text("Saved language", "저장한 언어"), saved)
		p.languageSelectionOutput()
		return p.writeErr
	}
	config.AddCommand(language)
	root.AddCommand(config)
}
