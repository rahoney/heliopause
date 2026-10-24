//go:build realcorpus

package npm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

// Public acquisition/static facts only. No lifecycle script is executed and
// no dynamic completion, PNPM selection or installation approval is claimed.
func TestPublicArchiveInstallActionFacts(t *testing.T) {
	if os.Getenv("HELOX_NPM_INSTALL_ACTION_PUBLIC") != "1" {
		t.Fatal("declare HELOX_NPM_INSTALL_ACTION_PUBLIC=1 for public archive qualification")
	}
	for _, test := range []struct{ reference, kind string }{
		{"is-number@7.0.0", domain.EvidenceHostInstallActionNotRequired},
		{"node-sass@9.0.0", domain.EvidenceHostInstallActionRequired},
	} {
		t.Run(test.reference, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			intake, records := t.TempDir(), t.TempDir()
			resolver, err := artifactnpm.NewPublicResolver(intake)
			if err != nil {
				t.Fatal(err)
			}
			reference, err := artifactnpm.ParseReference(test.reference)
			if err != nil {
				t.Fatal(err)
			}
			resolved, err := resolver.Resolve(ctx, reference)
			if err != nil {
				t.Fatal(err)
			}
			run, err := domain.NewRunID()
			if err != nil {
				t.Fatal(err)
			}
			artifact, err := resolver.Acquire(ctx, run, resolved)
			if err != nil {
				t.Fatal(err)
			}
			verified, err := (verificationnpm.IntegrityVerifier{}).Verify(ctx, artifact)
			if err != nil || verified.Outcome() != domain.VerificationVerified {
				t.Fatalf("archive integrity: %v %v", verified, err)
			}
			inspector, err := NewInspector(intake)
			if err != nil {
				t.Fatal(err)
			}
			report, err := inspector.Inspect(ctx, artifact)
			if err != nil || len(report.Findings()) != 0 {
				t.Fatalf("static archive inspection: %v %v", report, err)
			}
			store, err := local.NewStore(records)
			if err != nil {
				t.Fatal(err)
			}
			facts := append(verified.Evidence(), report.Evidence()...)
			refs, err := store.Record(ctx, run, facts)
			if err != nil {
				t.Fatal(err)
			}
			node, err := domain.ProjectDependencyNodeID(artifact.Identity())
			if err != nil {
				t.Fatal(err)
			}
			decision, err := domain.NewPolicyDecision(domain.DecisionManualReview, "fixture-static-only", 1, []string{"DYNAMIC_NOT_EXECUTED"})
			if err != nil {
				t.Fatal(err)
			}
			checks := append([]domain.CheckExecution{verified.Execution()}, report.Executions()...)
			entry, err := domain.NewDependencyInspection(node, run, artifact, checks, refs, decision)
			if err != nil {
				t.Fatal(err)
			}
			entry, err = entry.WithHostInstallActionEvidence(facts)
			if err != nil {
				t.Fatal(err)
			}
			fact, known := entry.HostInstallActionEvidence()
			if !known || fact.Kind() != test.kind || fact.Identity() != artifact.Identity() || fact.Digest() != artifact.Digest() {
				t.Fatal("public archive action fact lost")
			}
			body, err := os.ReadFile(filepath.Join(records, run.String(), fact.ID().String()+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var saved struct {
				Kind         string `json:"kind"`
				SHA256       string `json:"sha256"`
				RecordSHA256 string `json:"record_sha256"`
			}
			if err := json.Unmarshal(body, &saved); err != nil || saved.Kind != test.kind || saved.SHA256 != artifact.Digest().String() || saved.RecordSHA256 == "" {
				t.Fatalf("stored fact lost: %v %v", saved, err)
			}
			t.Logf("public_archive identity=%s@%s bytes=%d sha256=%s kind=%s references=%d; dynamic/install not executed", artifact.Identity().Name(), artifact.Identity().Version(), artifact.SizeBytes(), artifact.Digest().String(), fact.Kind(), len(refs))
		})
	}
}
