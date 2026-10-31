//go:build linux && realcorpus

package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	evidencerecord "github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

type uvToolUnavailableEvidence struct{ cause error }

func (e uvToolUnavailableEvidence) Record(context.Context, domain.RunID, []domain.Evidence) ([]domain.EvidenceReference, error) {
	return nil, e.cause
}

func TestUVPublicToolUpgradeEvidenceFailureIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_UPDATE_EVIDENCE_FAILURE") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user managed UV tool current Evidence failure qualification")
	}
	ctx, bindings, root, interpreter := uvToolApprovedIntegrationFixture(t)
	venv, bin := filepath.Join(root, "cowsay"), filepath.Join(root, "commands")
	install := prepareUVToolTargetFixture(t, venv, bin, interpreter)
	fresh, err := newUVToolInstallService(bindings, bin)
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := domain.NewArtifactReference(artifactpypi.PublicPyPIProfile().Source(), "cowsay==6.1")
	seed, err := application.NewInstallRequest(reference, install)
	if err != nil {
		t.Fatal(err)
	}
	previous, err := fresh.Install(ctx, seed)
	if err != nil || !previous.Bundle().Valid() {
		t.Fatal("managed old tool preparation", err)
	}
	before := uvToolTargetState(t, venv, bin)
	primary := errors.New("current Evidence storage unavailable")
	inspection, err := application.NewInstallInspectService(bindings.resolver, bindings.intake, verificationpypi.IntegrityVerifier{}, bindings.inspector, uvToolUnavailableEvidence{primary}, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	staging, err := promotion.NewLocalStaging(bindings.intakeRoot, bindings.evidenceRoot, bindings.stagingRoot)
	if err != nil {
		t.Fatal(err)
	}
	base, err := application.NewInstallService(inspection, evidencerecord.Generator{}, staging, bindings.promoter)
	if err != nil {
		t.Fatal(err)
	}
	bindings.installer = base
	service, err := newUVToolUpdateService(bindings, bin)
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewInstallUpdateRequest(artifactpypi.PublicPyPIProfile().Source(), "cowsay", install, false)
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := service.Install(ctx, request)
	if !errors.Is(err, primary) || rejected.Bundle().Valid() || rejected.Promoted().ManifestID().String() != "" {
		t.Fatal("old approval bypassed failed current Evidence", err)
	}
	if !reflect.DeepEqual(before, uvToolTargetState(t, venv, bin)) {
		t.Fatal("failed current approval changed old tool")
	}
	t.Log("uv_tool_noop_current_evidence_failure refused=true original_inodes_bytes_modes_links_unchanged=true")
}
