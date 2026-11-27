package cli

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestUVToolUpgradeRoutesCurrentUpdateAndPreservesCondition(t *testing.T) {
	for _, input := range []string{"Demo_Package", "Demo_Package@1.0", "Demo_Package>=1,<2"} {
		for _, adopt := range []bool{false, true} {
			target, bin := uvToolCLIPaths(t)
			root, err := New(&bytes.Buffer{}, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			recorder := &uvClientRequestRecorder{}
			if err := AddUVToolUpgrade(root, recorder); err != nil {
				t.Fatal(err)
			}
			args := []string{"--output", "json", "uv", "tool", "upgrade", input, "--target", target, "--bin-dir", bin}
			if adopt {
				args = append(args, "--adopt")
			}
			if source, err := PreparePresentation(root, args); err != nil || source != "pypi" || SelectedPackageClient(root) != UVToolPackageClient || SelectedPackageClientOperation(root) != "upgrade" {
				t.Fatal("wrong upgrade route", err)
			}
			root.SetArgs(args)
			_ = root.ExecuteContext(context.Background())
			if len(recorder.requests) != 1 {
				t.Fatal("upgrade request not routed once")
			}
			request := recorder.requests[0]
			got, err := artifactpypi.ParseUpgradeRequirement(request.Reference().Locator())
			want, _ := artifactpypi.ParseUpgradeRequirement(input)
			if err != nil || !got.SameConstraints(want) || request.Operation() != domain.OperationUpdate || request.Context().Target().String() != target || request.Source() != artifactpypi.PublicPyPIProfile().Source() {
				t.Fatal("upgrade condition/context lost", err)
			}
			expected, err := application.NewInstallUpdateRequest(request.Source(), input, request.Context(), adopt)
			if err != nil || expected != request {
				t.Fatal("explicit native adoption intent lost", err)
			}
		}
	}
}

func TestUVToolUpgradeRejectsImplicitAndNativeOptionsBeforeBinding(t *testing.T) {
	target, bin := uvToolCLIPaths(t)
	for _, tail := range [][]string{
		{}, {"cowsay"}, {"demo-package", "--target", target},
		{"other", "--target", target, "--bin-dir", bin},
		{"demo-package", "--target", target, "--bin-dir", bin, "--adopt", "--adopt=false"},
		{"demo-package", "--target", target, "--bin-dir", bin, "--all"},
		{"demo-package", "--target", target, "--bin-dir", bin, "--with", "other"},
		{"demo-package", "--target", target, "--target", target, "--bin-dir", bin},
	} {
		root, err := New(io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := PreparePresentation(root, append([]string{"uv", "tool", "upgrade"}, tail...)); err == nil {
			t.Fatal("unsupported upgrade input passed", tail)
		}
	}
}
