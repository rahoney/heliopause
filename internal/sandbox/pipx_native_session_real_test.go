//go:build linux && realcorpus

package sandbox

import (
	"context"
	"errors"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/hosttool"
)

type pipxNativeSessionActualRunner struct {
	CommandRunner
	targets atomic.Int64
}

func (*pipxNativeSessionActualRunner) RequiresDirectExecAdmission() {}
func (r *pipxNativeSessionActualRunner) RunInput(ctx context.Context, input io.Reader, name string, args ...string) error {
	return r.CommandRunner.(inputCommandRunner).RunInput(ctx, input, name, args...)
}
func (r *pipxNativeSessionActualRunner) RunOutput(ctx context.Context, output io.Writer, name string, args ...string) error {
	return r.CommandRunner.(streamingDockerOutput).RunOutput(ctx, output, name, args...)
}
func (r *pipxNativeSessionActualRunner) RunDiscard(ctx context.Context, name string, args ...string) error {
	return r.CommandRunner.(discardCommandRunner).RunDiscard(ctx, name, args...)
}
func (r *pipxNativeSessionActualRunner) RunBounded(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, mode, _, ok := directExecOrigin(args, name); ok && mode == boundaryPythonHandoffMode {
		r.targets.Add(1)
	}
	return r.CommandRunner.(boundedCommandRunner).RunBounded(ctx, name, args...)
}

type pipxNativeSessionActualResources struct {
	*pipxActualResources
	fail     bool
	injected atomic.Bool
}

func (r *pipxNativeSessionActualResources) Read(ctx context.Context, transaction string) (ObservationResourceLease, error) {
	lease, err := r.pipxActualResources.Read(ctx, transaction)
	if err == nil && r.fail && r.injected.CompareAndSwap(false, true) {
		return lease, errors.New("test-only native accounting failure after real read")
	}
	return lease, err
}

func TestLinuxPipxObservedNativeInstallIntegration(t *testing.T) {
	if os.Getenv("HELOX_PIPX_NATIVE_SESSION_INTEGRATION") != "1" {
		t.Skip("requires explicit fixed native input and isolated observed runtime")
	}
	if os.Geteuid() == 0 {
		t.Fatal("ordinary caller required")
	}
	input := pipxDiagnosticFrozenInput(t)
	frozen, err := freezePipxNativeInput(context.Background(), input.lock, input.inputs, input.pages)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(os.Getenv("HELOX_PIPX_RUNTIME_ZIP"))
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement("userpath==" + input.version)
	if err != nil {
		t.Fatal(err)
	}
	for _, failure := range []bool{false, true} {
		name := "fresh"
		if failure {
			name = "accounting-refusal"
		}
		if !t.Run(name, func(t *testing.T) {
			supervisor := integrationObserverSupervisor(t)
			defer func() {
				if err := supervisor.Close(); err != nil {
					t.Error(err)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			system, err := hosttool.NewSystem(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := system.Close(); err != nil {
					t.Error(err)
				}
			}()
			runner := &pipxNativeSessionActualRunner{CommandRunner: system}
			client, err := hosttool.NewSystemObservationResourceClient()
			if err != nil {
				t.Fatal(err)
			}
			resources := &pipxNativeSessionActualResources{pipxActualResources: &pipxActualResources{ObservationResourceClient: integrationObservationResources{client: client}}, fail: failure}
			result, runErr := runPipxNativeInstall(ctx, runner, supervisor.Observer(), resources, &pipxPreparedTool{body: body}, frozen, requirement)
			resources.mu.Lock()
			creates, registers, closes, cpu, last, reads := resources.creates, resources.registers, resources.closes, resources.cpu, resources.lastReadCPU, resources.readCount
			resources.mu.Unlock()
			t.Logf("native_profile=%s target_attempts=%d creates=%d registers=%d closes=%d cpu_usec=%d last_read_cpu_usec=%d reads=%d injected=%t controls=%d graph=%d observer_reason=%s error=%v", pipxNativeResolverProfile, runner.targets.Load(), creates, registers, closes, cpu, last, reads, resources.injected.Load(), len(result.controls), len(result.graph.Nodes()), integrationObserverFaultReason(supervisor), runErr)
			if creates != 1 || closes != 1 {
				t.Fatal("native resource owner not closed exactly once")
			}
			if failure {
				if runErr == nil || !resources.injected.Load() || runner.targets.Load() != 0 || len(result.controls) != 0 || result.receipt.Client() != "" {
					t.Fatal("native accounting failure executed a target or returned metadata")
				}
				return
			}
			if runErr != nil {
				t.Fatal("dedicated observed native session failed", runErr)
			}
			if runner.targets.Load() != 1 || registers != 1 || cpu == 0 || cpu >= 13_000_000 || result.receipt.Client() != "pipx" || len(result.controls) != 2 || len(result.graph.Nodes()) != 2 || !result.receipt.Requirement().SameConstraints(requirement) {
				t.Fatal("native lifecycle/complete original selection differs")
			}
			t.Log("actual original42 fixture with synthetic lock binding only; no live native public resolution, installed supply authentication, Policy or Host publication")
		}) {
			return
		}
	}
}
