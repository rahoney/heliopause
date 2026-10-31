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

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func uvCLIProjectFixture(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile("../artifact/pypi/testdata/uv-0.13.4-empty-project.pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}
	// Project the historical format onto the current runtime; keep captured testdata unchanged.
	body = bytes.ReplaceAll(body, []byte("3.14.7"), []byte(runtimeidentity.PythonVersion))
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".venv"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

// Renderer-only facts: actual approval/publication is separately qualified.
type uvProjectCLIRecorder struct {
	t        *testing.T
	requests []UVProjectRequest
	missing  bool
	cause    error
}

func (r *uvProjectCLIRecorder) ApplyUVProject(_ context.Context, request UVProjectRequest) (UVProjectOutcome, error) {
	r.requests = append(r.requests, request)
	if r.cause != nil {
		return UVProjectOutcome{}, r.cause
	}
	digest, err := domain.NewSHA256Digest(strings.Repeat("e", 64))
	if err != nil {
		r.t.Fatal(err)
	}
	c, err := domain.NewProjectControlDigest("pyproject.toml", digest)
	if err != nil {
		r.t.Fatal(err)
	}
	snapshot, err := domain.NewDependencyFreeProjectSnapshot(request.Install, artifactpypi.PublicPyPIProfile().Source(), []domain.ProjectControlDigest{c}, digest)
	if err != nil {
		r.t.Fatal(err)
	}
	inspected, err := domain.NewInspectedProjectSet(snapshot, nil)
	if err != nil {
		r.t.Fatal(err)
	}
	decision, err := domain.NewPolicyDecision(domain.DecisionAllow, "synthetic-renderer", 1, []string{"SYNTHETIC_ALLOW"})
	if err != nil {
		r.t.Fatal(err)
	}
	set, err := domain.NewProjectVerifiedSet(inspected, decision)
	if err != nil {
		r.t.Fatal(err)
	}
	run, err := domain.NewRunID()
	if err != nil {
		r.t.Fatal(err)
	}
	prefix := "project-cache:" + run.String()
	out := UVProjectOutcome{set, digest, prefix + ":manifest.json", prefix + ":sbom.cdx.json"}
	if r.missing {
		out.ManifestID = domain.ContentDigest{}
	}
	return out, nil
}

func TestUVProjectCLIRoutesAddSyncAndRendersActualRecordReferences(t *testing.T) {
	for _, operation := range []string{"add", "sync"} {
		for _, lang := range []string{"en", "ko"} {
			for _, output := range []string{"json", "human"} {
				t.Run(operation+lang+output, func(t *testing.T) {
					project := uvCLIProjectFixture(t)
					args := []string{"--lang", lang, "--output", output, "uv", operation, "--project", project}
					adopt := lang == "ko"
					if adopt {
						args = append(args, "--adopt")
					}
					selector := ""
					if operation == "add" {
						selector = "packaging==25.0"
						args = append(args, selector)
					}
					var out bytes.Buffer
					root, err := New(&out, io.Discard)
					if err != nil {
						t.Fatal(err)
					}
					if source, err := PreparePresentation(root, args); err != nil || source != "pypi" || SelectedPackageClient(root) != UVProjectPackageClient || SelectedPackageClientOperation(root) != operation {
						t.Fatal("wrong project route", source, err)
					}
					recorder := &uvProjectCLIRecorder{t: t}
					if err := AddUVProject(root, recorder); err != nil {
						t.Fatal(err)
					}
					root.SetArgs(args)
					if err := root.ExecuteContext(context.Background()); err != nil {
						t.Fatal(err, out.String())
					}
					if len(recorder.requests) != 1 {
						t.Fatal("project request repeated or omitted")
					}
					request := recorder.requests[0]
					if request.ProjectRoot != project || request.Selector != selector || request.Install.Target().String() != filepath.Join(project, ".venv") || request.Install.Mode() != domain.InstallPythonVenv || request.Adopt != adopt {
						t.Fatal("project/venv request changed", request)
					}
					if output == "json" {
						var doc struct {
							Schema, Operation string
							Status            string              `json:"operation_status"`
							Promotion         string              `json:"promotion_status"`
							Records           *machineVerifiedSet `json:"verified_set"`
						}
						if json.Unmarshal(out.Bytes(), &doc) != nil || doc.Status != "COMPLETED" || doc.Promotion != "COMPLETED" || doc.Records == nil || doc.Records.ManifestID == "" {
							t.Fatal("completed record report missing", out.String())
						}
						if (doc.Operation == "INSTALL") != (operation == "add") {
							t.Fatal("native intent report changed")
						}
					} else if json.Valid(out.Bytes()) || !strings.Contains(out.String(), "Python") {
						t.Fatal("human Python result missing", out.String())
					}
				})
			}
		}
	}
}

func TestUVProjectCLIRejectsUnsupportedInputBeforeComposition(t *testing.T) {
	for _, name := range []string{"duplicate project", "duplicate target", "empty project", "empty target", "unknown option", "duplicate adopt", "sync argument", "missing add", "direct selector", "extra selector", "source overlap", "missing venv", "manifest symlink", "manifest hardlink", "manifest oversize", "unsupported manifest", "malformed lock"} {
		t.Run(name, func(t *testing.T) {
			project := uvCLIProjectFixture(t)
			args := []string{"uv", "sync", "--project", project}
			path := filepath.Join(project, "pyproject.toml")
			var err error
			switch name {
			case "duplicate project":
				args = append(args, "--project="+project)
			case "duplicate target":
				args = append(args, "--target", filepath.Join(project, ".venv"), "--target="+filepath.Join(project, ".venv"))
			case "empty project":
				args = []string{"uv", "sync", "--project="}
			case "empty target":
				args = append(args, "--target=")
			case "unknown option":
				args = append(args, "--index-url", "https://example.invalid")
			case "duplicate adopt":
				args = append(args, "--adopt", "--adopt=false")
			case "sync argument":
				args = append(args, "packaging")
			case "missing add":
				args = []string{"uv", "add", "--project", project}
			case "direct selector":
				args = []string{"uv", "add", "--project", project, "https://example.invalid/a.whl"}
			case "extra selector":
				args = []string{"uv", "add", "--project", project, "packaging[extra]"}
			case "source overlap":
				args = append(args, "--target", project)
			case "missing venv":
				err = os.Remove(filepath.Join(project, ".venv"))
			case "manifest symlink":
				err = os.Rename(path, path+".original")
				if err == nil {
					err = os.Symlink(path+".original", path)
				}
			case "manifest hardlink":
				err = os.Link(path, path+".alias")
			case "manifest oversize":
				err = os.WriteFile(path, bytes.Repeat([]byte(" "), 4<<20+1), 0o600)
			case "unsupported manifest":
				err = os.WriteFile(path, []byte("[build-system]\nrequires=['x']\n"), 0o600)
			case "malformed lock":
				err = os.WriteFile(filepath.Join(project, "uv.lock"), []byte("version=99"), 0o600)
			}
			if err != nil {
				t.Fatal(err)
			}
			root, err := New(io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			recorder := &uvProjectCLIRecorder{t: t}
			if err := AddUVProject(root, recorder); err != nil {
				t.Fatal(err)
			}
			if _, err := PreparePresentation(root, args); err == nil {
				t.Fatal("unsupported project input accepted")
			}
			if len(recorder.requests) != 0 {
				t.Fatal("invalid preflight reached backend")
			}
		})
	}
}

func TestUVProjectCLIHelpAndFailureNeverClaimCompletion(t *testing.T) {
	for _, lang := range []string{"en", "ko"} {
		var out bytes.Buffer
		root, err := New(&out, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		args := []string{"--lang", lang, "uv", "add", "--help"}
		if _, err := PreparePresentation(root, args); err != nil {
			t.Fatal("help required project/runtime", err)
		}
		root.SetArgs(args)
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatal(err)
		}
		for _, text := range []string{"--project", "--target", ".venv"} {
			if !strings.Contains(out.String(), text) {
				t.Fatal("help scope missing", text, out.String())
			}
		}
	}
	for _, missing := range []bool{false, true} {
		project := uvCLIProjectFixture(t)
		var out bytes.Buffer
		root, err := New(&out, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		cause := errors.New("synthetic backend failure")
		recorder := &uvProjectCLIRecorder{t: t, missing: missing}
		if !missing {
			recorder.cause = cause
		}
		if err := AddUVProject(root, recorder); err != nil {
			t.Fatal(err)
		}
		args := []string{"--output", "json", "uv", "sync", "--project", project}
		if _, err := PreparePresentation(root, args); err != nil {
			t.Fatal(err)
		}
		root.SetArgs(args)
		err = root.ExecuteContext(context.Background())
		if err == nil || !missing && !errors.Is(err, cause) {
			t.Fatal("failed/missing records completed", err)
		}
		var doc struct {
			Status    string `json:"operation_status"`
			Promotion string `json:"promotion_status"`
		}
		if json.Unmarshal(out.Bytes(), &doc) != nil || doc.Status == "COMPLETED" || doc.Promotion == "COMPLETED" {
			t.Fatal("failure rendered completion", out.String())
		}
	}
}
