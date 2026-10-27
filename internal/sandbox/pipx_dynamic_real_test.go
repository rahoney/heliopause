//go:build linux && realcorpus

package sandbox

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

type pipxActualResources struct {
	ObservationResourceClient
	mu                         sync.Mutex
	creates, registers, closes int
	cpu                        uint64
	inject, injected           bool
}

func (r *pipxActualResources) Create(ctx context.Context, transaction, profile string) (ObservationResourceLease, error) {
	lease, err := r.ObservationResourceClient.Create(ctx, transaction, profile)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creates++
	return lease, err
}
func (r *pipxActualResources) Register(ctx context.Context, transaction, container, role string) (ObservationResourceLease, error) {
	lease, err := r.ObservationResourceClient.Register(ctx, transaction, container, role)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registers++
	return lease, err
}
func (r *pipxActualResources) Read(ctx context.Context, transaction string) (ObservationResourceLease, error) {
	lease, err := r.ObservationResourceClient.Read(ctx, transaction)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err == nil && r.inject && !r.injected && r.registers >= 3 {
		r.injected = true
		return lease, errors.New("test-only accounting transport failure after real read")
	}
	return lease, err
}
func (r *pipxActualResources) Close(ctx context.Context, transaction string) (ObservationResourceLease, error) {
	lease, err := r.ObservationResourceClient.Close(ctx, transaction)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closes++
	r.cpu = lease.UsageUsec
	return lease, err
}

func TestLinuxPipxAcquiredRuntimeTransactionIntegration(t *testing.T) {
	if os.Getenv("HELOX_PIPX_DYNAMIC_INTEGRATION") != "1" {
		t.Skip("requires explicit acquired ZIP and isolated observer runtime")
	}
	body, err := os.ReadFile(os.Getenv("HELOX_PIPX_RUNTIME_ZIP"))
	if err != nil {
		t.Fatal(err)
	}
	zip, err := artifactpypi.ParsePipxRuntimeZIP(body)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := zip.ObservationPlan()
	if err != nil {
		t.Fatal(err)
	}
	artifact := pipxObservationArtifact(t, "")
	for _, name := range []string{"complete-before", "accounting-failure", "complete-after"} {
		t.Run(name, func(t *testing.T) {
			supervisor := integrationObserverSupervisor(t)
			defer supervisor.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			executor, err := hosttool.NewSystem(ctx)
			if err != nil {
				t.Fatal(err)
			}
			client, err := hosttool.NewSystemObservationResourceClient()
			if err != nil {
				t.Fatal(err)
			}
			resources := &pipxActualResources{ObservationResourceClient: integrationObservationResources{client: client}, inject: name == "accounting-failure"}
			backend, err := NewLinuxPipxRuntimeBackendWithExecutor(executor, supervisor.Observer(), resources)
			if err != nil {
				t.Fatal(err)
			}
			result, runErr := backend.InspectPipxRuntime(ctx, artifact, plan)
			code, _ := result.LimitationCode()
			resources.mu.Lock()
			defer resources.mu.Unlock()
			t.Logf("status=%s code=%s commands=%d creates=%d registers=%d closes=%d cpu_usec=%d injected=%t observer_reason=%s error=%v", result.Status(), code, len(result.Commands), resources.creates, resources.registers, resources.closes, resources.cpu, resources.injected, integrationObserverFaultReason(supervisor), runErr)
			if resources.creates != 1 || resources.closes != 1 {
				t.Fatal("transaction resource owner not closed exactly once")
			}
			if resources.inject {
				if !resources.injected || result.Status() != domain.SandboxIncomplete || len(result.Commands) != 0 || resources.registers < 3 {
					t.Fatal("accounting failure produced completion")
				}
				return
			}
			if runErr != nil || result.Status() != domain.SandboxCompleted || resources.registers != 22 || len(result.Commands) != 5 || resources.cpu > 30_000_000 {
				t.Fatal("acquired transaction did not reconcile all20 units", runErr)
			}
			for _, command := range result.Commands {
				if !command.ZeroExit || command.OwnerSHA256 != artifact.Digest().String() || command.UnitID == "" {
					t.Fatal("command outcome lost original subject")
				}
			}
		})
	}
}
