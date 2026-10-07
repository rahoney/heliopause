package promotion

import (
	"context"
	"errors"
	"reflect"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func (g *approvedCargoProjectGuard) verifyBuildApproval(ctx context.Context, build domain.ApprovedProjectBuild) ([]goBuildEvidenceRecord, error) {
	if g == nil || g.buildInputs == nil || !build.Valid() || !reflect.DeepEqual(build.Report().Inputs(), *g.buildInputs) {
		return nil, errors.New("cargo build approval differs from guarded inputs")
	}
	if _, err := domain.NewApprovedProjectBuild(build.Report(), build.Verification(), build.Result()); err != nil {
		return nil, err
	}
	if err := g.VerifyBuildSource(ctx); err != nil {
		return nil, err
	}
	snapshot, _, err := g.OpenBuildInputs(ctx)
	if err != nil || !reflect.DeepEqual(snapshot, build.Report().Inputs().Snapshot()) {
		return nil, errors.New("cargo build retained approval changed")
	}
	expected := map[domain.EvidenceID]domain.Evidence{}
	for _, fact := range append(build.Verification().Evidence(), build.Report().Inspection().Evidence()...) {
		expected[fact.ID()] = fact
	}
	bindingID, _ := domain.NewEvidenceID("project-build-output-binding")
	if fact, ok := expected[bindingID]; !ok || fact.Kind() != "cargo-build-output-binding" || fact.Summary() != domain.BuildBindingSummary(*g.buildInputs, build.Report().Output(), build.Report().Binding()) {
		return nil, errors.New("cargo build output lacks exact recorded binding")
	}
	refs := build.Result().Evidence()
	if len(refs) != len(expected) {
		return nil, errors.New("cargo build Evidence coverage differs")
	}
	records := make([]goBuildEvidenceRecord, 0, len(refs))
	seen := map[domain.EvidenceID]bool{}
	for _, ref := range refs {
		fact, ok := expected[ref.ID()]
		if !ok || seen[ref.ID()] {
			return nil, errors.New("cargo build Evidence reference is substituted")
		}
		actual, digest, err := g.owner.cache.evidence.ReadReference(ctx, g.buildInputs.RunID(), ref, g.buildInputs.Source().Identity(), g.buildInputs.Source().Digest())
		if err != nil || actual != fact || digest.String() == "" {
			return nil, errors.New("cargo build required Evidence is unavailable or changed")
		}
		records = append(records, goBuildEvidenceRecord{ref.ID().String(), ref.Handle(), digest.String()})
		seen[ref.ID()] = true
	}
	return records, nil
}

func (g *approvedCargoProjectGuard) PublishBuild(ctx context.Context, build domain.ApprovedProjectBuild) (domain.PublishedProjectBuild, error) {
	if g == nil || g.buildInputs == nil {
		return domain.PublishedProjectBuild{}, errors.New("cargo build inputs were not frozen")
	}
	return publishProjectBuildOutput(ctx, g.guard.root, g.owner.cache.intakeRoot, *g.buildInputs, build, g.verifyBuildApproval, func(step string) error {
		if g.owner.checkpoint != nil {
			return g.owner.checkpoint("BUILD_" + step)
		}
		return nil
	})
}
