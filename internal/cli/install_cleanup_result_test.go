package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/policy"
)

type completedCLIUpdateSession struct {
	*installCLIPorts
	closeErr error
}

func (s *completedCLIUpdateSession) Reference() domain.ArtifactReference { return s.reference }
func (s *completedCLIUpdateSession) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (domain.PromotedInstall, error) {
	return (installCLIPromotion{}).Promote(ctx, staged, bundle, install)
}
func (s *completedCLIUpdateSession) Close() error { return s.closeErr }

type completedCLIUpdatePreparation struct{ session ports.InstallUpdateSession }

func (p completedCLIUpdatePreparation) BeginInstallUpdate(context.Context, domain.SourceID, string, domain.InstallContext, bool) (ports.InstallUpdateSession, error) {
	return p.session, nil
}

func TestUpdateReportRetainsConfirmedPromotionWhenGuardCleanupFails(t *testing.T) {
	base, old := installCLIFixture(t, policy.M4{}, nil)
	cause := errors.New("guard release uncertain")
	session := &completedCLIUpdateSession{newInstallCLIPorts(t), cause}
	updater, err := application.NewInstallUpdateService(base, completedCLIUpdatePreparation{session})
	if err != nil {
		t.Fatal(err)
	}
	install, _ := domain.NewNPMProjectInstallContext(old.Context().Target())
	request, _ := application.NewInstallUpdateRequest(old.Source(), "safe", install, false)
	var out bytes.Buffer
	code, err := cli.ExecuteInstall(context.Background(), updater, request, true, &out)
	if code != 1 || !errors.Is(err, cause) {
		t.Fatal(code, err)
	}
	var doc struct {
		Status    string `json:"operation_status"`
		Promotion string `json:"promotion_status"`
		Error     struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err, out.String())
	}
	if doc.Status != "FAILED" || doc.Promotion != "COMPLETED" || doc.Error.Code != "UPDATE_GUARD_CLOSE_FAILED" {
		t.Fatal("confirmed installation hidden by cleanup failure", out.String())
	}
}
