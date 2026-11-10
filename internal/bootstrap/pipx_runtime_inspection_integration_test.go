//go:build linux && realcorpus

package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	inspectiongithub "github.com/rahoney/heliopause/internal/inspection/githubrelease"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func TestPipxAcquiredRuntimeRequiredInspectionIntegration(t *testing.T) {
	if os.Getenv("HELOX_PIPX_INSPECTION_INTEGRATION") != "1" {
		t.Skip("requires selected acquired ZIP and isolated pinned observer")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user protected test client")
	}
	body, err := os.ReadFile(os.Getenv("HELOX_PIPX_RUNTIME_ZIP"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	launcher, err := hosttool.NewSystemObserverLauncher()
	if err != nil {
		t.Fatal(err)
	}
	observer, err := sandbox.NewObserverSupervisor(ctx, func(start context.Context, remote, output string) (sandbox.ObserverProcess, error) {
		return launcher.StartObserver(start, remote, output)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := observer.Close(); err != nil {
			t.Error(err)
		}
	}()
	root := t.TempDir()
	run, err := domain.NewRunID()
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, run.String())
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(directory, "asset")
	if err := os.WriteFile(filename, body, 0600); err != nil {
		t.Fatal(err)
	}
	source, err := domain.NewSourceID("github-release")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := domain.NewResolvedArtifactIdentity(source, "rahoney-heliopause", "v9.8.7", runtimeidentity.PipxRuntimeAssetName)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.NewSHA256Digest(runtimeidentity.PipxToolSHA256)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":github-release", uint64(len(body)), "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	backend, err := sandbox.NewLinuxPipxRuntimeBackendWithExecutor(executor, observer.Observer(), newObservationResourceAdapter())
	if err != nil {
		t.Fatal(err)
	}
	inspector, err := inspectiongithub.NewPipxRuntimeInspector(root, pipxReleaseObservationRunner{backend: backend})
	if err != nil {
		t.Fatal(err)
	}
	report, err := inspector.Inspect(ctx, artifact)
	if err != nil || report.Execution().Status() != domain.ExecutionCompleted || !report.Execution().Required() || report.Execution().ID().String() != "github-release-pipx-runtime" || len(report.Findings()) != 0 || len(report.Evidence()) != 6 {
		t.Fatalf("required acquired inspection failed: report=%+v error=%v", report, err)
	}
	for _, evidence := range report.Evidence() {
		if evidence.Identity() != artifact.Identity() || evidence.Digest() != artifact.Digest() {
			t.Fatal("HAA identity/digest substituted")
		}
	}

	for _, evidence := range report.Evidence()[1:] {
		var command struct {
			Module, Disposition string
			Owner               string `json:"owner_sha256"`
			Unit                string `json:"unit"`
			Functionality       bool   `json:"functionality_attested"`
			Enforced            bool   `json:"later_execution_enforced"`
		}
		if err := json.Unmarshal([]byte(evidence.Summary()), &command); err != nil {
			t.Fatal(err)
		}
		if command.Disposition != "ATTESTED" || command.Owner != artifact.Digest().String() || command.Unit == "" || command.Functionality || command.Enforced {
			t.Fatal("typed bounded outcome changed", command)
		}
	}
	if err := os.WriteFile(filename, []byte("tampered after successful acquired inspection"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := inspector.Inspect(ctx, artifact); err == nil {
		t.Fatal("tampered intake reused prior completion")
	}
	t.Logf("required=%s status=COMPLETED findings=0 Evidence=6 commands=5 original_HAA=%s source=%s; tampered acquired bytes refused", report.Execution().ID(), artifact.Identity().Version(), artifact.Identity().Source())
}
