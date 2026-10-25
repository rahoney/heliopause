package cli_test

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

	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/spf13/cobra"
)

func bunCLIProject(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"dependencies":{"is-number":"6.0.0"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

// Renderer contract fixture only; this does not claim actual native inspection.
type bunCLIRecorder struct {
	t        *testing.T
	requests []cli.BunProjectRequest
	missing  bool
	cause    error
}

func (r *bunCLIRecorder) ApplyBun(_ context.Context, request cli.BunProjectRequest) (cli.BunProjectOutcome, error) {
	r.requests = append(r.requests, request)
	source, _ := domain.NewSourceID("npm")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
	control, _ := domain.NewProjectControlDigest("package.json", digest)
	snapshot, err := domain.NewDependencyFreeProjectSnapshot(request.Install, source, []domain.ProjectControlDigest{control}, digest)
	if err != nil {
		r.t.Fatal(err)
	}
	inspected, err := domain.NewInspectedProjectSet(snapshot, nil)
	if err != nil {
		r.t.Fatal(err)
	}
	decision, err := domain.NewPolicyDecision(domain.DecisionAllow, "mechanical-renderer", 1, []string{"ALLOW"})
	if err != nil {
		r.t.Fatal(err)
	}
	set, err := domain.NewProjectVerifiedSet(inspected, decision)
	if err != nil {
		r.t.Fatal(err)
	}
	if r.cause != nil {
		return cli.BunProjectOutcome{}, cargoFailedUpdateFacts{inspected, decision, r.cause}
	}
	run, _ := domain.NewRunID()
	handle := "project-cache:" + run.String()
	out := cli.BunProjectOutcome{Verified: set, ManifestID: digest, ManifestHandle: handle + ":manifest.json", SBOMHandle: handle + ":sbom.cdx.json"}
	if r.missing {
		out.ManifestID = domain.ContentDigest{}
	}
	return out, nil
}

func TestBunCLIRoutesCompleteProjectAndRendersCorrectOperation(t *testing.T) {
	for _, operation := range []string{"add", "install", "update"} {
		for _, lang := range []string{"en", "ko"} {
			for _, output := range []string{"json", "human"} {
				t.Run(operation+lang+output, func(t *testing.T) {
					target := bunCLIProject(t)
					args := []string{"--lang", lang, "--output", output, "bun", operation}
					selector := ""
					if operation == "add" {
						selector = "@scope/pkg@^1.0.0"
					}
					if operation == "update" {
						selector = "is-number"
					}
					if selector != "" {
						args = append(args, "--target", target, "--", selector)
					} else {
						args = append(args, "--target", target)
					}
					var out bytes.Buffer
					root, err := bunCLICommand(&out, io.Discard)
					if err != nil {
						t.Fatal(err)
					}
					if source, err := cli.PreparePresentation(root, args); err != nil || source != "npm" || cli.SelectedPackageClient(root) != cli.BunPackageClient || cli.SelectedPackageClientOperation(root) != operation {
						t.Fatal("client/source/operation route", source, err)
					}
					recorder := &bunCLIRecorder{t: t}
					if err := cli.AddBunProject(root, recorder); err != nil {
						t.Fatal(err)
					}
					root.SetArgs(args)
					if err := root.ExecuteContext(context.Background()); err != nil {
						t.Fatal(err, out.String())
					}
					if len(recorder.requests) != 1 {
						t.Fatal("request not invoked exactly once")
					}
					request := recorder.requests[0]
					if request.Selector != selector || string(request.Intent) != operation || request.Install.Target().String() != target || request.Install.Mode() != domain.InstallNPMProject {
						t.Fatal("wrong typed request", request)
					}
					want := "INSTALL"
					schema := "helox.project-install-result/v1"
					if operation == "update" {
						want = "UPDATE"
						schema = "helox.project-update-result/v1"
					}
					if output == "json" {
						var report struct {
							Schema    string           `json:"schema_version"`
							Operation string           `json:"operation"`
							Status    string           `json:"operation_status"`
							Promotion string           `json:"promotion_status"`
							Source    string           `json:"source_id"`
							Records   *json.RawMessage `json:"verified_set"`
						}
						if err := json.Unmarshal(out.Bytes(), &report); err != nil {
							t.Fatal(err, out.String())
						}
						if report.Schema != schema || report.Operation != want || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || report.Source != "npm" || report.Records == nil {
							t.Fatal("incorrect completed report", out.String())
						}
					} else {
						label := "Project installation:"
						if operation == "update" {
							label = "Project update:"
						}
						if lang == "ko" {
							label = "프로젝트 설치:"
							if operation == "update" {
								label = "프로젝트 업데이트:"
							}
						}
						if !strings.Contains(out.String(), label) || strings.Contains(out.String(), "crates") || strings.Contains(out.String(), "crate 수") || strings.Contains(out.String(), "cargo build") || !strings.Contains(out.String(), "Manifest:") {
							t.Fatal("incorrect npm human report", out.String())
						}
					}
				})
			}
		}
	}
}

func TestBunCLIRejectsInvalidInputBeforeComposition(t *testing.T) {
	target := bunCLIProject(t)
	for _, tail := range [][]string{
		{"add"}, {"add", ""}, {"add", "file:../outside"}, {"add", "https://example.invalid/a.tgz"}, {"add", "a", "b"}, {"install", "a"}, {"update", ""}, {"update", "foreign"}, {"update", "is-number@7.0.0"},
		{"install", "--target="}, {"install", "--target", target, "--target", target}, {"install", "--adopt", "--adopt"}, {"install", "--registry=https://example.invalid"}, {"install", "--global"},
	} {
		root, err := bunCLICommand(io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		args := append([]string{"bun"}, tail...)
		if !strings.Contains(strings.Join(tail, " "), "--target") {
			args = append(args, "--target", target)
		}
		if _, err := cli.PreparePresentation(root, args); err == nil {
			t.Fatal("unsupported input reached composition", args)
		}
	}
	for _, kind := range []string{"missing-manifest", "symlink-manifest", "hardlink-manifest", "oversize", "target-symlink", "target-absent"} {
		t.Run(kind, func(t *testing.T) {
			target := bunCLIProject(t)
			manifest := filepath.Join(target, "package.json")
			switch kind {
			case "missing-manifest":
				if err := os.Remove(manifest); err != nil {
					t.Fatal(err)
				}
			case "symlink-manifest":
				if err := os.Rename(manifest, manifest+".actual"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("package.json.actual", manifest); err != nil {
					t.Fatal(err)
				}
			case "hardlink-manifest":
				if err := os.Link(manifest, manifest+".alias"); err != nil {
					t.Fatal(err)
				}
			case "oversize":
				if err := os.WriteFile(manifest, bytes.Repeat([]byte("x"), (4<<20)+1), 0o600); err != nil {
					t.Fatal(err)
				}
			case "target-symlink":
				link := target + "-link"
				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Remove(link) })
				target = link
			case "target-absent":
				target = filepath.Join(target, "missing")
			}
			root, err := bunCLICommand(io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := cli.PreparePresentation(root, []string{"bun", "install", "--target", target}); err == nil {
				t.Fatal("unbounded/aliased project reached composition")
			}
		})
	}
}

func TestBunCLIHelpAndFailedPublication(t *testing.T) {
	for _, lang := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		var out bytes.Buffer
		root, err := bunCLICommand(&out, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		args := []string{"--lang", lang, "bun", "install", "--help"}
		if _, err := cli.PreparePresentation(root, args); err != nil {
			t.Fatal(err)
		}
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "bun.lock") || !strings.Contains(out.String(), "--target") || !strings.Contains(out.String(), "--adopt") {
			t.Fatal("missing truthful project help", lang, out.String())
		}
	}
	for _, kind := range []string{"missing-records", "publication-failure"} {
		t.Run(kind, func(t *testing.T) {
			target := bunCLIProject(t)
			var out bytes.Buffer
			root, _ := bunCLICommand(&out, io.Discard)
			cause := errors.New("controlled publication failure")
			recorder := &bunCLIRecorder{t: t, missing: kind == "missing-records"}
			if kind == "publication-failure" {
				recorder.cause = cause
			}
			if err := cli.AddBunProject(root, recorder); err != nil {
				t.Fatal(err)
			}
			args := []string{"--output", "json", "bun", "install", "--target", target}
			if _, err := cli.PreparePresentation(root, args); err != nil {
				t.Fatal(err)
			}
			root.SetArgs(args)
			err := root.ExecuteContext(context.Background())
			if err == nil || recorder.cause != nil && !errors.Is(err, cause) {
				t.Fatal("failed publication claimed completion", err)
			}
			if strings.Contains(out.String(), `"promotion_status":"COMPLETED"`) || !strings.Contains(out.String(), `"operation_status":"FAILED"`) {
				t.Fatal("incorrect failure report", out.String())
			}
		})
	}
}

func bunCLICommand(out, diagnostic io.Writer) (*cobra.Command, error) {
	root, err := cli.New(out, diagnostic)
	if err != nil {
		return nil, err
	}
	if err := cli.SetBunClientVersion(root, runtimeidentity.BunVersion); err != nil {
		return nil, err
	}
	return root, nil
}

func TestBunPreflightRequiresComposedReviewedIdentity(t *testing.T) {
	target := bunCLIProject(t)
	root, err := cli.New(io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.PreparePresentation(root, []string{"bun", "install", "--target", target}); err == nil {
		t.Fatal("missing static identity accepted")
	}
}
