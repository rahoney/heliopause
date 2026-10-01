//go:build linux

package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLinuxPyPIWheelDynamicIntegration(t *testing.T) {
	if os.Getenv("HELOX_PYPI_DYNAMIC_INTEGRATION") != "1" {
		t.Skip("requires pinned Linux Python/gVisor dynamic integration")
	}
	root := t.TempDir()
	wheel := linuxDynamicWheel(t)
	path := filepath.Join(root, "run_aaaaaaaaaaaaaaaaaaaaaaaaaa", "wheel.whl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, wheel, 0o600); err != nil {
		t.Fatal(err)
	}
	// The dynamic introducer binds the in-container name to the verified intake
	// filename record, exactly as production intake does.
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "filename"), []byte("example-1.0-py3-none-any.whl"), 0o400); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(wheel)
	static, err := artifactpypi.InspectWheel(bytes.NewReader(wheel), int64(len(wheel)), "example-1.0-py3-none-any.whl", hex.EncodeToString(sum[:]), artifactpypi.WheelTarget{Python: "cp314", ABI: "cp314", Platform: "manylinux_2_36_x86_64"}, artifactpypi.DefaultWheelLimits())
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("pypi")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "example", "1.0", "wheel")
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	artifact, err := domain.NewAcquiredArtifact(identity, digest, "intake:run_aaaaaaaaaaaaaaaaaaaaaaaaaa:wheel", uint64(len(wheel)))
	if err != nil {
		t.Fatal(err)
	}
	supervisor := integrationObserverSupervisor(t)
	defer supervisor.Close()
	runner := integrationRunner{t: t}
	introducer, err := NewPythonArtifactIntroducer(root, runner)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := NewPythonDynamicBackend(runner, introducer, supervisor.Observer(), integrationPythonCapabilityProbe(runner))
	if err != nil {
		t.Fatal(err)
	}
	resources, err := hosttool.NewSystemObservationResourceClient()
	if err != nil {
		t.Fatal(err)
	}
	backend.resources = integrationObservationResources{client: resources}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := backend.InspectWheel(ctx, artifact, static.ImportNames)
	if err != nil || result.Status() != domain.SandboxCompleted {
		t.Fatalf("dynamic result = %#v observer_reason=%s, %v", result, integrationObserverFaultReason(supervisor), err)
	}
}

type integrationObservationResources struct {
	client *hosttool.ObservationResourceClient
}

func integrationObservationLease(lease hosttool.ObservationResourceLease) ObservationResourceLease {
	return ObservationResourceLease{CgroupParent: lease.CgroupParent, CPU: lease.CPU, UsageUsec: lease.UsageUsec}
}

func (a integrationObservationResources) Create(ctx context.Context, transaction, profile string) (ObservationResourceLease, error) {
	lease, err := a.client.Create(ctx, transaction, profile)
	return integrationObservationLease(lease), err
}
func (a integrationObservationResources) Register(ctx context.Context, transaction, container, role string) (ObservationResourceLease, error) {
	lease, err := a.client.Register(ctx, transaction, container, role)
	return integrationObservationLease(lease), err
}
func (a integrationObservationResources) Read(ctx context.Context, transaction string) (ObservationResourceLease, error) {
	lease, err := a.client.Read(ctx, transaction)
	return integrationObservationLease(lease), err
}
func (a integrationObservationResources) Terminate(ctx context.Context, transaction, container string) (ObservationResourceLease, error) {
	lease, err := a.client.Terminate(ctx, transaction, container)
	return integrationObservationLease(lease), err
}
func (a integrationObservationResources) Close(ctx context.Context, transaction string) (ObservationResourceLease, error) {
	lease, err := a.client.Close(ctx, transaction)
	return integrationObservationLease(lease), err
}
