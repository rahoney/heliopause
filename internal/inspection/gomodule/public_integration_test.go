//go:build integration

package gomodule_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectiongo "github.com/rahoney/heliopause/internal/inspection/gomodule"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	verificationgo "github.com/rahoney/heliopause/internal/verification/gomodule"
)

// This uses an empty operation-owned intake and real public source, signed
// SumDB/proofs, normalized static inspection, recorded Evidence and entry Policy.
// It does not qualify project mutation, verified cache or sandbox builds.
func TestGoPublicModuleAcquisitionIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	root := t.TempDir()
	intake := filepath.Join(root, "intake")
	client, err := artifactgo.NewPublicClient(intake)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := local.NewStore(filepath.Join(root, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := verificationgo.NewIntegrityVerifier(intake)
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := inspectiongo.NewStaticInspector(intake)
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewInspectService(client, verifier, inspector, evidence, policy.M3{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := artifactgo.ParseReference("github.com/spf13/pflag@v1.0.9")
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewInspectRequest(ref)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Inspect(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	decision, ok := result.PolicyDecision()
	if !ok || decision.Decision() != domain.DecisionAllow || len(result.Checks()) != 2 || len(result.Evidence()) != 2 {
		t.Fatalf("incomplete acquisition/verification/inspection/evidence/policy: status=%s", result.Status())
	}
	for _, check := range result.Checks() {
		if check.Status() != domain.ExecutionCompleted || !check.Required() {
			t.Fatal("incomplete required check")
		}
	}
	identity, _ := result.ResolvedIdentity()
	digest, _ := result.Digest()
	t.Logf("source=%s version=%s digest=%s checks=%d evidence=%d policy=%s", ref.Source().String(), identity.Version(), digest.String(), len(result.Checks()), len(result.Evidence()), decision.Decision())
}

// The snapshot below is an explicit qualification input, not a claim that the
// isolated project resolver is wired. All subsequent source, proof, per-entry
// inspection, set Policy, Evidence and cache boundaries are real consumers.
func TestGoPublicProjectVerifiedCacheIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	root := t.TempDir()
	intake := filepath.Join(root, "intake")
	evidenceRoot := filepath.Join(root, "evidence")
	poison := filepath.Join(root, "ambient-cache")
	if err := os.Mkdir(poison, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(poison, "poison.go"), []byte("untrusted caller cache"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMODCACHE", poison)
	client, err := artifactgo.NewPublicClient(intake)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := local.NewStore(evidenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := verificationgo.NewIntegrityVerifier(intake)
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := inspectiongo.NewStaticInspector(intake)
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewProjectInspectService(client, verifier, inspector, evidence, policy.M3{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := promotion.NewGoVerifiedCache(intake, evidenceRoot, filepath.Join(root, "verified-cache"))
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(filepath.Join(root, "project"))
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	const zSum = "h1:9exaQaMOCwffKiiiYk6/BndUBv+iRViNW+4lEMi0PvY="
	// This exact public pin is also used by the acquisition baseline.
	const modSum = "h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg="
	mod := []byte("module example.com/project\ngo 1.26.0\nrequire github.com/spf13/pflag v1.0.9\n")
	sum := []byte("github.com/spf13/pflag v1.0.9 " + zSum + "\ngithub.com/spf13/pflag v1.0.9/go.mod " + modSum + "\n")
	records := []artifactgo.DownloadRecord{{Path: "github.com/spf13/pflag", Version: "v1.0.9", GoMod: "/fixture/pflag.mod", Zip: "/fixture/pflag.zip", Sum: zSum, GoModSum: modSum}}
	snapshot, err := artifactgo.BuildProjectSnapshot(install, records, []byte("example.com/project github.com/spf13/pflag@v1.0.9\nexample.com/project go@1.26.0\n"), mod, sum)
	if err != nil {
		t.Fatal(err)
	}
	approved, err := service.InspectProject(ctx, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !approved.Valid() || len(approved.Inspected().Inspections()) != 1 {
		t.Fatal("project did not receive complete independent approval")
	}
	staged, err := cache.StageProject(ctx, approved)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := cache.OpenProjectCache(ctx, staged)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "github.com/spf13/pflag@v1.0.9", "flag.go")); err != nil {
		t.Fatal("authenticated Go source was not materialized")
	}
	if _, err := os.Stat(filepath.Join(dir, "poison.go")); !os.IsNotExist(err) {
		t.Fatal("ambient cache entered approved tree")
	}
	i := approved.Inspected().Inspections()[0]
	if len(i.Evidence()) != 2 || len(i.Checks()) != 2 || i.PolicyDecision().Decision() != domain.DecisionAllow {
		t.Fatal("project lost entry inspection/Evidence/Policy binding")
	}
	t.Logf("graph=%s subject=%s receipt=%s entries=%d checks=%d evidence=%d policy=%s", snapshot.GraphDigest().String(), i.Artifact().Digest().String(), staged.Digest().String(), len(approved.Inspected().Inspections()), len(i.Checks()), len(i.Evidence()), approved.Decision().Decision())
}
