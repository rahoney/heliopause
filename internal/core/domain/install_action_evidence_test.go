package domain_test

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestDependencyInspectionBindsRecordedInstallActionEvidence(t *testing.T) {
	_, first, _ := inspectedGraph(t)
	base := dependencyInspection(t, first, "a")
	makeFact := func(id domain.EvidenceID, check domain.CheckID, identity domain.ResolvedArtifactIdentity, digest domain.ContentDigest, kind, summary string) domain.Evidence {
		t.Helper()
		fact, err := domain.NewEvidence(id, check, identity, digest, kind, summary)
		if err != nil {
			t.Fatal(err)
		}
		return fact
	}
	for _, kind := range []string{domain.EvidenceHostInstallActionRequired, domain.EvidenceHostInstallActionNotRequired} {
		fact := makeFact(base.Evidence()[0].ID(), base.Checks()[0].ID(), base.Artifact().Identity(), base.Artifact().Digest(), kind, "arbitrary sanitized display text")
		got, err := base.WithHostInstallActionEvidence([]domain.Evidence{fact})
		if err != nil {
			t.Fatal(err)
		}
		retained, known := got.HostInstallActionEvidence()
		if !known || retained != fact {
			t.Fatal("recorded exact fact was lost")
		}
		if _, known := base.HostInstallActionEvidence(); known {
			t.Fatal("binding changed the original value")
		}
		if _, err := got.WithHostInstallActionEvidence([]domain.Evidence{fact}); err == nil {
			t.Fatal("repeated binding accepted")
		}
	}
	id, _ := domain.NewEvidenceID("unrecorded-fact")
	check, _ := domain.NewCheckID("other-check")
	identity, _ := domain.NewResolvedArtifactIdentity(base.Artifact().Identity().Source(), "other", "1.0.0", "tarball")
	digest, _ := domain.NewSHA256Digest(strings.Repeat("c", 64))
	valid := makeFact(base.Evidence()[0].ID(), base.Checks()[0].ID(), base.Artifact().Identity(), base.Artifact().Digest(), domain.EvidenceHostInstallActionNotRequired, "no installation action")
	conflict := makeFact(valid.ID(), valid.CheckID(), valid.Identity(), valid.Digest(), domain.EvidenceHostInstallActionRequired, "installation action declared")
	for _, test := range []struct {
		name  string
		facts []domain.Evidence
	}{
		{"unrecorded", []domain.Evidence{makeFact(id, valid.CheckID(), valid.Identity(), valid.Digest(), valid.Kind(), "fact")}},
		{"unknown-check", []domain.Evidence{makeFact(valid.ID(), check, valid.Identity(), valid.Digest(), valid.Kind(), "fact")}},
		{"other-identity", []domain.Evidence{makeFact(valid.ID(), valid.CheckID(), identity, valid.Digest(), valid.Kind(), "fact")}},
		{"other-digest", []domain.Evidence{makeFact(valid.ID(), valid.CheckID(), valid.Identity(), digest, valid.Kind(), "fact")}},
		{"duplicate", []domain.Evidence{valid, valid}},
		{"conflict", []domain.Evidence{valid, conflict}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := base.WithHostInstallActionEvidence(test.facts); err == nil {
				t.Fatal("invalid installation fact accepted")
			}
		})
	}
	for _, test := range []struct {
		name       string
		kind       domain.CheckKind
		required   bool
		capability domain.Capability
		status     domain.ExecutionStatus
		limit      string
		duplicate  bool
	}{
		{"optional", domain.CheckInspection, false, domain.CapabilitySupported, domain.ExecutionCompleted, "", false},
		{"verification", domain.CheckVerification, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", false},
		{"incomplete", domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, "FIXTURE_INCOMPLETE", false},
		{"unavailable", domain.CheckInspection, true, domain.CapabilityUnsupported, domain.ExecutionNotExecuted, "FIXTURE_UNAVAILABLE", false},
		{"duplicate-check", domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			check, err := domain.NewCheckExecution(valid.CheckID(), test.kind, test.required, test.capability, test.status, test.limit)
			if err != nil {
				t.Fatal(err)
			}
			checks := []domain.CheckExecution{check}
			if test.duplicate {
				checks = append(checks, check)
			}
			record, err := domain.NewDependencyInspection(base.Node(), base.RunID(), base.Artifact(), checks, base.Evidence(), base.PolicyDecision())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := record.WithHostInstallActionEvidence([]domain.Evidence{valid}); err == nil {
				t.Fatal("unqualified check accepted as installation fact")
			}
		})
	}
	other := makeFact(valid.ID(), valid.CheckID(), valid.Identity(), valid.Digest(), "inspection", "host-install-action-not-required")
	got, err := base.WithHostInstallActionEvidence([]domain.Evidence{other})
	if err != nil {
		t.Fatal(err)
	}
	if _, known := got.HostInstallActionEvidence(); known {
		t.Fatal("display text was interpreted as a fact")
	}
	if _, err := (domain.DependencyInspection{}).WithHostInstallActionEvidence(nil); err == nil {
		t.Fatal("zero inspection accepted")
	}
}
