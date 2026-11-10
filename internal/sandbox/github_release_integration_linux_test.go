//go:build linux

package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	evidencelocal "github.com/rahoney/heliopause/internal/evidence/local"
	inspectiongithub "github.com/rahoney/heliopause/internal/inspection/githubrelease"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestLinuxGitHubReleaseELFDynamicIntegration(t *testing.T) {
	if os.Getenv("HELOX_GITHUB_RELEASE_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor runtime for GitHub Release ELF")
	}
	runLinuxGitHubReleaseELFDynamicIntegration(t, "/bin/true")
}

func TestLinuxResponderReleaseProbeIntegration(t *testing.T) {
	if os.Getenv("HELOX_GITHUB_RELEASE_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux gVisor and protected responder source")
	}
	if err := CheckResponderSource(context.Background()); err != nil {
		t.Fatal("fixed responder readiness", err)
	}
	runLinuxGitHubReleaseELFDynamicIntegration(t, filepath.Join(runtimeidentity.NativeInstallRoot, "current", runtimeidentity.SandboxResponderAssetName))
}

func runLinuxGitHubReleaseELFDynamicIntegration(t *testing.T, program string) {
	t.Helper()
	root := t.TempDir()
	runID, _ := domain.NewRunID()
	directory := filepath.Join(root, runID.String())
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(program)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "asset"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	source, _ := domain.NewSourceID("github-release")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "owner-repo", "v1", "tool")
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:"+runID.String()+":github-release", uint64(len(content)))
	if err != nil {
		t.Fatal(err)
	}
	request, err := domain.NewSandboxRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	runner := integrationRunner{t: t}
	backend, err := NewGitHubELFBackend(runner, root, supervisor.Observer(), integrationCapabilityProbe(runner))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	backend.resources = integrationObservationResourceAdapter(t)
	inspector, err := inspectiongithub.NewDynamicInspector(backend)
	if err != nil {
		t.Fatal(err)
	}
	report, err := inspector.Inspect(ctx, request.Artifact())
	if err != nil || report.Execution().Status() != domain.ExecutionCompleted || len(report.Evidence()) != 3 {
		t.Fatalf("GitHub ELF current dynamic report incomplete: status=%s observer_reason=%s err=%v", report.Execution().Status(), integrationObserverFaultReason(supervisor), err)
	}
	store, err := evidencelocal.NewStore(filepath.Join(root, "evidence"))
	if err != nil {
		t.Fatal(err)
	}
	refs, err := store.Record(ctx, runID, report.Evidence(), report.Executions()...)
	if err != nil {
		t.Fatal(err)
	}
	profiles := map[domain.SandboxEnvironmentProfile]bool{}
	for _, ref := range refs {
		fact, _, err := store.ReadReference(ctx, runID, ref, artifact.Identity(), artifact.Digest())
		if err != nil || fact.CheckID() != report.Execution().ID() || fact.ProfileDigest() != report.Execution().ProfileDigest() {
			t.Fatal("GitHub ELF stored Evidence binding changed", err)
		}
		if fact.Kind() != "workspace-state" {
			continue
		}
		var state struct {
			Schema     string                           `json:"schema"`
			Profile    domain.SandboxEnvironmentProfile `json:"profile"`
			Before     string                           `json:"baseline_digest"`
			After      string                           `json:"final_digest"`
			Operations string                           `json:"operation_scope"`
		}
		if err := json.Unmarshal([]byte(fact.Summary()), &state); err != nil {
			t.Fatal(err)
		}
		if state.Schema != domain.SandboxEnvironmentStateSchema || len(state.Before) != 64 || len(state.After) != 64 || state.Operations != "CREATE_WRITE_PWRITE_RENAME_UNLINK_MODE_OWNER_SIZE" || profiles[state.Profile] {
			t.Fatal("GitHub ELF stored state missing, duplicated or uncovered")
		}
		profiles[state.Profile] = true
	}
	if !profiles[domain.SandboxEnvironmentMinimal] || !profiles[domain.SandboxEnvironmentDevelopment] || len(profiles) != 2 {
		t.Fatal("GitHub ELF lost stored environment pair")
	}
	t.Log("actual bounded GitHub ELF probe: two environments and current same-check/profile Record+ReadReference complete; synthetic source identity, no publisher attestation")

}
