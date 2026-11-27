package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type uvSyncCLIStub struct {
	request cli.UVPipSyncRequest
	outcome cli.UVPipSyncOutcome
	cause   error
	calls   int
}

func (s *uvSyncCLIStub) Sync(_ context.Context, request cli.UVPipSyncRequest) (cli.UVPipSyncOutcome, error) {
	s.calls++
	s.request = request
	return s.outcome, s.cause
}

func uvSyncCLIInput(t *testing.T, name, body string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUVSyncCLIStaticPreflightSeparatesInstallAndSkipsBlank(t *testing.T) {
	for _, language := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		t.Run(language, func(t *testing.T) {
			path := uvSyncCLIInput(t, "requirements.txt", "# no requirements\n")
			for _, output := range []string{"human", "json"} {
				var out, diagnostic bytes.Buffer
				root, err := cli.New(&out, &diagnostic)
				if err != nil {
					t.Fatal(err)
				}
				args := []string{"--lang", language, "--output", output, "uv", "pip", "sync", path, "--target", "/tmp/uv-sync-cli-preserved"}
				source, err := cli.PreparePresentation(root, args)
				if err != nil || source != "" || cli.SelectedPackageClient(root) != cli.UVPipPackageClient || cli.SelectedPackageClientOperation(root) != "sync" {
					t.Fatal("blank source prepared a backend", source, err)
				}
				install, err := cli.PackageClientInstallCommand(root, cli.UVPipPackageClient)
				if err != nil || install.CommandPath() != "helox uv pip install" {
					t.Fatal("install binding selected sync", err)
				}
				root.SetArgs(args)
				if err := root.ExecuteContext(context.Background()); err != nil {
					t.Fatal(err)
				}
				if output == "json" {
					var doc map[string]json.RawMessage
					if json.Unmarshal(out.Bytes(), &doc) != nil || string(doc["operation_status"]) != `"SKIPPED"` || string(doc["promotion_status"]) != `"NOT_PERFORMED"` {
						t.Fatal(out.String())
					}
					for _, field := range []string{"policy", "operation_id", "run", "verified_set", "graph_digest"} {
						if _, exists := doc[field]; exists {
							t.Fatal("skip invented inspection or publication", field)
						}
					}
				} else {
					if !strings.Contains(out.String(), "--allow-empty-requirements") || strings.Contains(out.String(), "Cargo") {
						t.Fatal("incorrect skip explanation", out.String())
					}
				}
			}
		})
	}
}

func TestUVSyncCLIRejectsUnsupportedInputsBeforeComposition(t *testing.T) {
	for _, mode := range []string{"missing", "range", "include", "duplicate-target", "duplicate-adopt", "empty-target", "unknown", "multiple-source", "symlink", "hardlink", "oversize", "pylock-allow"} {
		t.Run(mode, func(t *testing.T) {
			body := "packaging==25.0\n"
			name := "requirements.txt"
			switch mode {
			case "range":
				body = "packaging>=25\n"
			case "include":
				body = "-r other.txt\n"
			case "pylock-allow":
				name = "pylock.toml"
				body = "lock-version='1.0'\ncreated-by='uv'\nrequires-python='>=3.14.7'\npackages=[]\n"
			}
			path := uvSyncCLIInput(t, name, body)
			if mode == "missing" {
				path += ".absent"
			}
			if mode == "symlink" {
				link := path + ".link"
				if err := os.Symlink(path, link); err != nil {
					t.Fatal(err)
				}
				path = link
			}
			if mode == "hardlink" {
				if err := os.Link(path, path+".alias"); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "oversize" {
				if err := os.WriteFile(path, bytes.Repeat([]byte("x"), (4<<20)+1), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			args := []string{"--lang", "en", "uv", "pip", "sync", path, "--target", "/tmp/uv-sync-cli"}
			switch mode {
			case "duplicate-target":
				args = append(args, "--target=/tmp/another")
			case "duplicate-adopt":
				args = append(args, "--adopt", "--adopt")
			case "empty-target":
				args[len(args)-1] = ""
			case "unknown":
				args = append(args, "--system")
			case "multiple-source":
				args = append(args, path)
			case "pylock-allow":
				args = append(args, "--allow-empty-requirements")
			}
			var out, diagnostic bytes.Buffer
			root, err := cli.New(&out, &diagnostic)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := cli.PreparePresentation(root, args); err == nil {
				t.Fatal("unsupported sync input reached composition", mode)
			}
		})
	}
}

func TestUVSyncCLIReportsActualProjectFactsAndRecordReferences(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		for _, output := range []string{"human", "json"} {
			t.Run(language+output, func(t *testing.T) {
				inspected, decision := projectUpdateFactsForSource(t, domain.DecisionAllow, "pypi")
				set, err := domain.NewProjectVerifiedSet(inspected, decision)
				if err != nil {
					t.Fatal(err)
				}
				digest, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
				run, _ := domain.NewRunID()
				handle := "project-cache:" + run.String()
				stub := &uvSyncCLIStub{outcome: cli.UVPipSyncOutcome{Verified: set, ManifestID: digest, ManifestHandle: handle + ":manifest.json", SBOMHandle: handle + ":sbom.cdx.json"}}
				path := uvSyncCLIInput(t, "requirements.txt", "packaging==25.0\n")
				args := []string{"--lang", language, "--output", output, "uv", "pip", "sync", path, "--target", set.Inspected().Snapshot().Context().Target().String(), "--adopt"}
				var out, diagnostic bytes.Buffer
				root, err := cli.New(&out, &diagnostic)
				if err != nil {
					t.Fatal(err)
				}
				if source, err := cli.PreparePresentation(root, args); err != nil || source != "pypi" {
					t.Fatal(source, err)
				}
				if err := cli.AddUVPipSync(root, stub); err != nil {
					t.Fatal(err)
				}
				root.SetArgs(args)
				if err := root.ExecuteContext(context.Background()); err != nil {
					t.Fatal(err)
				}
				if stub.calls != 1 || !stub.request.Adopt || stub.request.SourceFile != path || stub.request.Install != set.Inspected().Snapshot().Context() {
					t.Fatal("sync intent or target lost", stub.request)
				}
				if output == "json" {
					var doc struct {
						Status    string `json:"operation_status"`
						Promotion string `json:"promotion_status"`
						Source    string `json:"source_id"`
						Entries   []struct {
							Evidence []json.RawMessage `json:"evidence_references"`
						}
						VerifiedSet *struct {
							ManifestID string `json:"manifest_id"`
						} `json:"verified_set"`
					}
					if json.Unmarshal(out.Bytes(), &doc) != nil || doc.Status != "COMPLETED" || doc.Promotion != "COMPLETED" || doc.Source != "pypi" || len(doc.Entries) != 1 || len(doc.Entries[0].Evidence) != 2 || doc.VerifiedSet == nil || doc.VerifiedSet.ManifestID != digest.String() {
						t.Fatal("incomplete project sync report", out.String())
					}
				} else {
					if !strings.Contains(out.String(), "Python") || !strings.Contains(out.String(), digest.String()) || strings.Contains(out.String(), "crates") || strings.Contains(out.String(), "cargo build") {
						t.Fatal("wrong ecosystem report", out.String())
					}
				}
			})
		}
	}
}

func TestUVSyncCLIHelpDoesNotReadSourceOrBindRuntime(t *testing.T) {
	var out, diagnostic bytes.Buffer
	root, err := cli.New(&out, &diagnostic)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"--lang", "ko", "uv", "pip", "sync", "--help"}
	if _, err := cli.PreparePresentation(root, args); err != nil {
		t.Fatal(err)
	}
	root.SetArgs(args)
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "동기화") || !strings.Contains(out.String(), "--allow-empty-requirements") || !strings.Contains(out.String(), "helox uv pip sync requirements.txt") {
		t.Fatal(out.String())
	}
	stub := &uvSyncCLIStub{cause: errors.New("must not run")}
	if err := cli.AddUVPipSync(nil, stub); err == nil {
		t.Fatal("nil command accepted")
	}
}
