package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type uvClientRequestRecorder struct{ requests []application.InstallRequest }

func (r *uvClientRequestRecorder) Install(_ context.Context, request application.InstallRequest) (application.InstallOutcome, error) {
	r.requests = append(r.requests, request)
	return application.InstallOutcome{}, nil
}

func TestUVPackageClientRoutingAndCanonicalInstallRequest(t *testing.T) {
	for _, spec := range []string{"Demo_Package", "Demo_Package==1.0", "Demo_Package@1.0"} {
		for _, leading := range []bool{false, true} {
			var out bytes.Buffer
			root, err := New(&out, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			recorder := &uvClientRequestRecorder{}
			if err := AddUVPipInstall(root, recorder); err != nil {
				t.Fatal(err)
			}
			args := []string{"uv", "pip", "install", "--target", "/tmp/haa-uv-cli-target", "--", spec}
			if leading {
				args = append([]string{"--lang", "ko", "--output", "json"}, args...)
			}
			source, err := PreparePresentation(root, args)
			if err != nil || source != "pypi" || SelectedPackageClient(root) != UVPipPackageClient {
				t.Fatal(source, err, SelectedPackageClient(root))
			}
			// Display writers are replaceable and cannot own backend selection.
			root.SetOut(&out)
			if SelectedPackageClient(root) != UVPipPackageClient {
				t.Fatal("client selection depends on display writer")
			}
			root.SetArgs(args)
			_ = root.ExecuteContext(context.Background()) // Recorder has no result to render.
			if len(recorder.requests) != 1 {
				t.Fatal("install was not routed exactly once", out.String())
			}
			request := recorder.requests[0]
			want := "demo-package"
			if spec != "Demo_Package" {
				want += "@1.0"
			}
			if request.Reference().Source().String() != "pypi" || request.Reference().Locator() != want || request.Context().Mode() != domain.InstallPythonVenv || request.Context().Target().String() != "/tmp/haa-uv-cli-target" {
				t.Fatal("wrong install request", request)
			}
		}
	}
}

func TestUVPackageClientRejectsUnsupportedIntentBeforeComposition(t *testing.T) {
	t.Setenv("VIRTUAL_ENV", "")
	cases := [][]string{
		{}, {"idna"}, {"idna", "--target", "relative"}, {"idna", "--target", "/tmp/a,b"},
		{"idna>=3", "--target", "/tmp/venv"}, {"idna==3.11rc1", "--target", "/tmp/venv"},
		{"idna==3.11==4", "--target", "/tmp/venv"}, {"https://example.invalid/a.whl", "--target", "/tmp/venv"},
		{"idna", "requests", "--target", "/tmp/venv"}, {"idna", "--target", "/tmp/venv", "--target=/tmp/other"},
		{"idna", "--target", "/tmp/venv", "--adopt"},
		{"idna", "--target", "/tmp/venv", "--upgrade", "--upgrade-package", "idna"},
		{"idna", "--target", "/tmp/venv", "--upgrade-package", "requests"},
		{"idna", "--target", "/tmp/venv", "--upgrade-package", "idna==3.11"},
		{"idna", "--target", "/tmp/venv", "--upgrade-package="},
		{"idna", "--target", "/tmp/venv", "--upgrade-package", "idna", "--upgrade-package=idna"},
		{"idna", "--target", "/tmp/venv", "--source", "pytorch:cpu"}, {"idna", "--target", "/tmp/venv", "--index-url", "https://example.invalid"},
	}
	for _, tail := range cases {
		root, err := New(io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := PreparePresentation(root, append([]string{"uv", "pip", "install"}, tail...)); err == nil {
			t.Fatalf("accepted unsupported intent: %q", tail)
		}
	}
}

func TestPackageClientHelpAndLegacySelection(t *testing.T) {
	for _, lang := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		var out bytes.Buffer
		root, err := New(&out, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		args := []string{"--lang", lang, "uv", "pip", "install", "--help"}
		if source, err := PreparePresentation(root, args); err != nil || source != "pypi" {
			t.Fatal(source, err)
		}
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		want := uvPipInstallEnglish
		if lang == "ko" {
			want = uvPipInstallKorean
		}
		if !strings.Contains(out.String(), want) || !strings.Contains(out.String(), "--target") || !strings.Contains(out.String(), "--upgrade-package") || strings.Contains(out.String(), "only-if-needed") || !strings.Contains(out.String(), "--adopt") || strings.Contains(out.String(), "--source") {
			t.Fatal("incorrect client help", lang, out.String())
		}
	}
	root, err := New(io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if source, err := PreparePresentation(root, []string{"pip", "install", "idna", "--target", "/tmp/venv"}); err != nil || source != "pip" || SelectedPackageClient(root) != PipPackageClient {
		t.Fatal("legacy lane changed", source, err)
	}
	if source, err := PreparePresentation(root, []string{"npm", "--help"}); err != nil || source != "npm" || SelectedPackageClient(root) != "" {
		t.Fatal("client selection leaked", source, err)
	}
}

func TestUVPackageClientPreservesUpgradeModeAndOperation(t *testing.T) {
	for _, spec := range []struct {
		flags []string
		mode  packageUpgradeMode
	}{
		{nil, packageNoUpgrade}, {[]string{"--upgrade"}, packageUpgradeClosure}, {[]string{"--upgrade", "--adopt"}, packageUpgradeClosure}, {[]string{"-U"}, packageUpgradeClosure}, {[]string{"--upgrade-package", "Demo_Package"}, packageUpgradeRoot}, {[]string{"--upgrade-package", "demo-package", "--adopt"}, packageUpgradeRoot},
	} {
		t.Run(strings.Join(spec.flags, "/"), func(t *testing.T) {
			root, err := New(io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			ordinary, all, one := &uvClientRequestRecorder{}, &uvClientRequestRecorder{}, &uvClientRequestRecorder{}
			if err := AddUVPipInstallAndUpdates(root, ordinary, all, one); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"--output", "json", "uv", "pip", "install", "Demo_Package==1.0", "--target", "/tmp/haa-uv-cli-target"}, spec.flags...)
			if _, err := PreparePresentation(root, args); err != nil {
				t.Fatal(err)
			}
			root.SetArgs(args)
			_ = root.ExecuteContext(context.Background())
			wanted := map[packageUpgradeMode]*uvClientRequestRecorder{packageNoUpgrade: ordinary, packageUpgradeClosure: all, packageUpgradeRoot: one}
			for mode, recorder := range wanted {
				count := 0
				if mode == spec.mode {
					count = 1
				}
				if len(recorder.requests) != count {
					t.Fatal("wrong service selected", mode, len(recorder.requests))
				}
			}
			request := wanted[spec.mode].requests[0]
			operation := domain.OperationInstall
			if spec.mode != packageNoUpgrade {
				operation = domain.OperationUpdate
			}
			if spec.mode != packageNoUpgrade {
				adopt := false
				for _, flag := range spec.flags {
					if flag == "--adopt" {
						adopt = true
					}
				}
				expected, err := application.NewInstallUpdateRequest(request.Source(), "Demo_Package==1.0", request.Context(), adopt)
				if err != nil || expected != request {
					t.Fatal("native enrollment intent changed", request, err)
				}
			}
			if request.Operation() != operation || request.Source().String() != "pypi" || request.Context().Target().String() != "/tmp/haa-uv-cli-target" {
				t.Fatal("wrong operation/source/target", request)
			}
		})
	}
}
