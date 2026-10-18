package application

import (
	"context"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// NewInstallUpdateRequest preserves an explicit all/one selection until the
// Infrastructure guard validates the current project. It creates no fake graph.
func NewInstallUpdateRequest(source domain.SourceID, selector string, install domain.InstallContext, adopt bool) (InstallRequest, error) {
	if source.String() == "" || !install.Valid() || install.RequiresNewTarget() || len(selector) > 512 || selector != strings.TrimSpace(selector) || strings.ContainsAny(selector, "\x00\r\n") {
		return InstallRequest{}, errors.New("update requires supported existing state and bounded selection")
	}
	locator := selector
	if locator == "" {
		locator = "*"
	} // Unresolved all-package selection, never a graph node.
	reference, err := domain.NewArtifactReference(source, locator)
	if err != nil {
		return InstallRequest{}, err
	}
	return InstallRequest{reference: reference, context: install, updateSource: source, updateSelector: selector, update: true, adopt: adopt}, nil
}

// InstallUpdateService adds the pre-resolution original-state guard to the
// existing InstallService. The inspection/Policy/Manifest/staging order is reused.
type InstallUpdateService struct {
	base        *InstallService
	preparation ports.InstallUpdatePreparation
}

func NewInstallUpdateService(base *InstallService, preparation ports.InstallUpdatePreparation) (*InstallUpdateService, error) {
	if base == nil || preparation == nil {
		return nil, errors.New("update requires existing install pipeline and original-state guard")
	}
	return &InstallUpdateService{base: base, preparation: preparation}, nil
}

func (s *InstallUpdateService) Install(ctx context.Context, request InstallRequest) (outcome InstallOutcome, resultErr error) {
	if s == nil || ctx == nil || !request.update || request.updateSource.String() == "" || !request.context.Valid() {
		return outcome, errors.New("validated update request is required")
	}
	operation, err := s.base.inspection.newOperationID()
	if err != nil {
		return outcome, err
	}
	outcome = InstallOutcome{inspection: newPartialInspectedInstall(operation, request, domain.DependencyResolution{}), failureCode: "UPDATE_PREPARATION_FAILED"}
	if err := ctx.Err(); err != nil {
		return outcome, err
	}
	session, err := s.preparation.BeginInstallUpdate(ctx, request.updateSource, request.updateSelector, request.context, request.adopt)
	if err != nil {
		if code := ports.InstallUpdateFailureCode(err); code != "" {
			outcome.failureCode = code
		}
		return outcome, err
	}
	if session == nil {
		return outcome, errors.New("update preparation returned no original-state guard")
	}
	defer func() {
		closeErr := session.Close()
		if closeErr != nil && outcome.failureCode == "" {
			outcome.failureCode = "UPDATE_GUARD_CLOSE_FAILED"
		}
		resultErr = errors.Join(resultErr, closeErr)
	}()
	request.reference = session.Reference()
	if request.reference.Source() != request.updateSource || request.reference.Locator() == "" {
		return outcome, errors.New("update preparation returned an invalid reference")
	}
	// Rebind only per-request copies. Concurrent installs never share a mutable
	// resolver/promoter or a cached original-state session.
	inspection := *s.base.inspection
	inspection.resolver = session
	inspection.newOperationID = func() (domain.OperationID, error) { return operation, nil }
	pipeline := *s.base
	pipeline.inspection, pipeline.promotion = &inspection, session
	outcome, resultErr = pipeline.Install(ctx, request)
	if code := ports.InstallUpdateFailureCode(resultErr); code != "" {
		outcome.failureCode = code
	}
	return outcome, resultErr
}
