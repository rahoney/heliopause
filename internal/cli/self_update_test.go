package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/testutil/fakeworkflow"
)

type selfUpdateCapture struct {
	request application.InstallRequest
	calls   int
}

func (s *selfUpdateCapture) Install(_ context.Context, request application.InstallRequest) (application.InstallOutcome, error) {
	s.calls++
	s.request = request
	return application.InstallOutcome{}, nil
}

func TestSelfUpdateAdmitsOnlyExactNativeIntentBeforeComposition(t *testing.T) {
	for _, args := range [][]string{{"self-update"}, {"self-update", "--version", "latest"}, {"self-update", "--version", "v1.2"}, {"self-update", "--version", "v1.2.3", "other"}, {"self-update", "--version", "v1.2.3", "--target", "/tmp/target"}, {"self-update", "--version", "v1.2.3", "--adopt"}, {"self-update", "--version", "v1.2.3", "--version=v2.0.0"}, {"self-update", "--version=v1.2.3", "--version", "v2.0.0"}} {
		root, _ := New(io.Discard, io.Discard)
		capture := &selfUpdateCapture{}
		if err := AddSelfUpdate(root, capture); err != nil {
			t.Fatal(err)
		}
		if _, err := PreparePresentation(root, args); err == nil || capture.calls != 0 {
			t.Fatal("invalid self-update admitted", args, capture.calls, err)
		}
	}
	root, _ := New(io.Discard, io.Discard)
	capture := &selfUpdateCapture{}
	if err := AddSelfUpdate(root, capture); err != nil {
		t.Fatal(err)
	}
	args := []string{"--output", "json", "self-update", "--version", "v1.2.3"}
	if source, err := PreparePresentation(root, args); err != nil || source != "self-update" {
		t.Fatal(source, err)
	}
	root.SetArgs(args)
	_ = root.ExecuteContext(context.Background())
	if capture.calls != 1 || capture.request.Operation() != domain.OperationUpdate || capture.request.Source().String() != "github-release" || capture.request.Reference().Locator() != "v1.2.3" || capture.request.Context().Target().String() != "/usr/lib/heliopause" || capture.request.Context().RequiresNewTarget() {
		t.Fatal(capture)
	}
}

func TestSelfUpdateHelpAndProgressSupportAllSevenLanguagesWithoutCallingInstaller(t *testing.T) {
	for _, language := range []string{"en", "ko", "de", "fr", "es", "pt-BR", "ja"} {
		var output bytes.Buffer
		root, _ := New(&output, io.Discard)
		capture := &selfUpdateCapture{}
		_ = AddSelfUpdate(root, capture)
		root.SetArgs([]string{"self-update", "--help", "--lang", language})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatal(language, err)
		}
		p := presentationFor(root.OutOrStdout())
		if !strings.Contains(output.String(), "--version") || capture.calls != 0 || !strings.Contains(output.String(), p.text(selfUpdateDescriptionEnglish, selfUpdateDescriptionKorean)) {
			t.Fatal(language, output.String(), capture.calls)
		}
		for _, stage := range []string{"authorize", "drain", "activate"} {
			if !knownStage(stage) || p.stage(stage) == stage || language != "en" && p.stage(stage) == (&presentation{language: "en"}).stage(stage) {
				t.Fatal("missing progress translation", language, stage)
			}
		}
		if language != "en" && strings.Contains(output.String(), selfUpdateDescriptionEnglish) {
			t.Fatal("missing core help translation", language)
		}
	}
}

type selfUpdateNoPublish struct{ calls int }

func (p *selfUpdateNoPublish) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	p.calls++
	return domain.DependencyResolution{}, errors.New("unexpected resolve")
}
func (p *selfUpdateNoPublish) Stage(context.Context, domain.VerifiedBundle) (domain.StagedSet, error) {
	p.calls++
	return domain.StagedSet{}, errors.New("unexpected staging")
}
func (p *selfUpdateNoPublish) Promote(context.Context, domain.StagedSet, domain.VerifiedBundle, domain.InstallContext) (domain.PromotedInstall, error) {
	p.calls++
	return domain.PromotedInstall{}, errors.New("unexpected promotion")
}

type selfUpdateFailedPreparation struct{ err error }

func (p selfUpdateFailedPreparation) BeginInstallUpdate(context.Context, domain.SourceID, string, domain.InstallContext, bool) (ports.InstallUpdateSession, error) {
	return nil, p.err
}

func TestSelfUpdateReportPreservesRecoveryCauseAndDoesNotClaimSingleAssetScope(t *testing.T) {
	for _, machine := range []bool{false, true} {
		fake, _ := fakeworkflow.New(fakeworkflow.Safe)
		unused := &selfUpdateNoPublish{}
		inspection, err := application.NewInstallInspectService(unused, fake, fake, fake, fake, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
		if err != nil {
			t.Fatal(err)
		}
		base, _ := application.NewInstallService(inspection, evidence.Generator{}, unused, unused)
		cause := errors.New("private raw artifact output")
		updater, _ := application.NewInstallUpdateService(base, selfUpdateFailedPreparation{ports.NewInstallUpdateFailure("SELF_UPDATE_TRANSITION_OR_RECOVERY_REQUIRED", errors.Join(cause, context.Canceled))})
		var out bytes.Buffer
		root, _ := New(&out, io.Discard)
		if err := AddSelfUpdate(root, updater); err != nil {
			t.Fatal(err)
		}
		format := "human"
		if machine {
			format = "json"
		}
		root.SetArgs([]string{"self-update", "--version", "v1.2.3", "--output", format, "--lang", "ko"})
		err = root.ExecuteContext(context.Background())
		if !errors.Is(err, cause) || unused.calls != 0 {
			t.Fatal(err, unused.calls)
		}
		if strings.Contains(out.String(), cause.Error()) || strings.Contains(out.String(), "one exact GitHub asset") || !strings.Contains(out.String(), "SELF_UPDATE_TRANSITION_OR_RECOVERY_REQUIRED") {
			t.Fatal(out.String())
		}
		if machine {
			var doc map[string]any
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Fatal("JSON stdout polluted", out.String(), err)
			}
			if doc["operation"] != "UPDATE" || doc["operation_status"] != "FAILED" {
				t.Fatal(doc)
			}
		} else if !strings.Contains(out.String(), "이미 교체됐을 수도") || !strings.Contains(out.String(), "native 실행 파일 네 개") {
			t.Fatal(out.String())
		}
	}
}
