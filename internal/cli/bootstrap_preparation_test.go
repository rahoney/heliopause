package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/ports"
)

type bootstrapPreparationFixture struct {
	calls     int
	version   string
	err       error
	directory string
}

func (f *bootstrapPreparationFixture) Prepare(ctx context.Context, version string) (ports.BootstrapPreparationResult, error) {
	f.calls++
	f.version = version
	ports.ReportProgress(ctx, ports.Progress{Stage: "verify"})
	return ports.BootstrapPreparationResult{Version: version, ManifestSHA256: strings.Repeat("a", 64), Directory: f.directory, Files: []string{"1", "2", "3", "4", "5", "6", "7"}, Proofs: []string{"proof"}}, f.err
}

func TestBootstrapHelpSupportsSevenLanguagesWithoutPreparing(t *testing.T) {
	for _, language := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		t.Run(language, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			var out, diagnostic bytes.Buffer
			root, err := New(&out, &diagnostic)
			if err != nil {
				t.Fatal(err)
			}
			service := &bootstrapPreparationFixture{}
			if err := AddBootstrapPreparation(root, service); err != nil {
				t.Fatal(err)
			}
			args := []string{"--lang", language, "bootstrap", "prepare", "--help"}
			root.SetArgs(args)
			if _, err := PreparePresentation(root, args); err != nil {
				t.Fatal(err)
			}
			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatal(err)
			}
			if service.calls != 0 || !strings.Contains(out.String(), "NOT_PERFORMED") || !strings.Contains(out.String(), "PENDING") {
				t.Fatal("help claimed installation or executed preparation", out.String())
			}
			if language == "ko" && !strings.Contains(out.String(), "최초 설치") {
				t.Fatal("Korean help missing")
			}
		})
	}
}

func TestBootstrapPreflightRejectsInvalidOrDuplicateVersionBeforeService(t *testing.T) {
	for _, args := range [][]string{
		{"bootstrap", "prepare"}, {"bootstrap", "prepare", "--version", "latest"}, {"bootstrap", "prepare", "--version", "v1.2.3", "--version", "v1.2.4"}, {"bootstrap", "prepare", "--version", "v1.2.3", "extra"}, {"bootstrap", "prepare", "--version", "v1.2.3", "--target", "/etc"},
	} {
		var out, diagnostic bytes.Buffer
		root, err := New(&out, &diagnostic)
		if err != nil {
			t.Fatal(err)
		}
		service := &bootstrapPreparationFixture{}
		if err := AddBootstrapPreparation(root, service); err != nil {
			t.Fatal(err)
		}
		root.SetArgs(args)
		if _, err := PreparePresentation(root, args); err == nil {
			t.Fatal("invalid preparation intent accepted", args)
		}
		if service.calls != 0 {
			t.Fatal("invalid request reached service")
		}
	}
}

func TestBootstrapReportDistinguishesFirstPreparationFromQuarantineAndInstall(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		for _, format := range []string{"json", "human"} {
			t.Run(language+"/"+format, func(t *testing.T) {
				t.Setenv("XDG_CONFIG_HOME", t.TempDir())
				var out, diagnostic bytes.Buffer
				root, err := New(&out, &diagnostic)
				if err != nil {
					t.Fatal(err)
				}
				service := &bootstrapPreparationFixture{directory: "/tmp/prepared\x1b[31m\nroot"}
				if err := AddBootstrapPreparation(root, service); err != nil {
					t.Fatal(err)
				}
				args := []string{"--lang", language, "--output", format, "bootstrap", "prepare", "--version", "v1.2.3"}
				root.SetArgs(args)
				if _, err := PreparePresentation(root, args); err != nil {
					t.Fatal(err)
				}
				if err := root.ExecuteContext(context.Background()); err != nil {
					t.Fatal(err)
				}
				if service.calls != 1 || service.version != "v1.2.3" || strings.Contains(out.String(), "ALLOW") || strings.Contains(out.String(), "\x1b[") {
					t.Fatal("incorrect bootstrap display", out.String())
				}
				if format == "json" {
					var result struct{ Status, Provenance, Quarantine, Installation string }
					if err := json.Unmarshal(out.Bytes(), &result); err != nil {
						t.Fatal(err)
					}
					if result.Status != "PREPARED" || result.Provenance != "VERIFIED" || result.Quarantine != "NOT_PERFORMED" || result.Installation != "PENDING" {
						t.Fatal(result)
					}
				} else if !strings.Contains(out.String(), "NOT_PERFORMED") || !strings.Contains(out.String(), "PENDING") {
					t.Fatal("first-install scope missing")
				}
			})
		}
	}
}

func TestBootstrapFailurePreservesStableCodeAndDoesNotRenderCause(t *testing.T) {
	var out, diagnostic bytes.Buffer
	root, err := New(&out, &diagnostic)
	if err != nil {
		t.Fatal(err)
	}
	sourceErr := errors.New("untrusted transport detail")
	service := &bootstrapPreparationFixture{err: ports.NewBootstrapPreparationFailure(ports.BootstrapProvenanceFailed, sourceErr)}
	if err := AddBootstrapPreparation(root, service); err != nil {
		t.Fatal(err)
	}
	args := []string{"--output", "json", "bootstrap", "prepare", "--version", "v1.2.3"}
	root.SetArgs(args)
	if _, err := PreparePresentation(root, args); err != nil {
		t.Fatal(err)
	}
	err = root.ExecuteContext(context.Background())
	if !errors.Is(err, sourceErr) || !strings.Contains(out.String(), string(ports.BootstrapProvenanceFailed)) || strings.Contains(out.String(), sourceErr.Error()) {
		t.Fatal("bootstrap cause leaked or disappeared", out.String(), err)
	}
}
