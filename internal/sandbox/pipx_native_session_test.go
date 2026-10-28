package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type pipxNativeNoCommands struct{ calls int }

func (r *pipxNativeNoCommands) Output(context.Context, string, ...string) ([]byte, error) {
	r.calls++
	return nil, errors.New("unexpected command")
}
func (r *pipxNativeNoCommands) RunBounded(context.Context, string, ...string) ([]byte, error) {
	r.calls++
	return nil, errors.New("unexpected command")
}
func (r *pipxNativeNoCommands) RunInput(context.Context, io.Reader, string, ...string) error {
	r.calls++
	return errors.New("unexpected input")
}
func (r *pipxNativeNoCommands) RunOutput(context.Context, io.Writer, string, ...string) error {
	r.calls++
	return errors.New("unexpected stream")
}

type pipxNativeUnusedObserver struct{ TraceObserver }

func (pipxNativeUnusedObserver) StartProfile(context.Context, string, string) (TraceReader, error) {
	panic("unexpected profile start")
}
func (pipxNativeUnusedObserver) AwaitMountAnchors(context.Context, string) error {
	panic("unexpected mount wait")
}

type pipxNativeUnusedResources struct{ ObservationResourceClient }

func TestPipxNativeSessionRefusesBeforeAnyCommands(t *testing.T) {
	for _, scenario := range []string{"nil-runner", "nil-observer", "nil-resources", "nil-tool", "nil-input", "changed-input", "foreign-root", "wrong-version", "cancelled", "unsupported-profile", "unsupported-discard", "wrong-ZIP"} {
		t.Run(scenario, func(t *testing.T) {
			frozen, _, _, _, _ := pipxFrozenInputFixture(t, nil)
			requirement, _ := artifactpypi.ParseUpgradeRequirement("probe==1.0")
			runner := &pipxNativeNoCommands{}
			var selected CommandRunner = &pipxNativeDiscardCommands{runner}
			var observer TraceObserver = pipxNativeUnusedObserver{}
			var resources ObservationResourceClient = pipxNativeUnusedResources{}
			tool := &pipxPreparedTool{body: []byte("not the pinned ZIP")}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch scenario {
			case "nil-runner":
				selected = nil
			case "nil-observer":
				observer = nil
			case "nil-resources":
				resources = nil
			case "nil-tool":
				tool = nil
			case "nil-input":
				frozen = nil
			case "changed-input":
				frozen.files[0].body[0] ^= 1
			case "foreign-root":
				requirement, _ = artifactpypi.ParseUpgradeRequirement("other==1.0")
			case "wrong-version":
				requirement, _ = artifactpypi.ParseUpgradeRequirement("probe==2.0")
			case "cancelled":
				cancel()
			case "unsupported-discard":
				selected = runner
			case "unsupported-profile":
				observer = &pipxNativeUnprofiledObserver{}
			}
			result, err := runPipxNativeInstall(ctx, selected, observer, resources, tool, frozen, requirement)
			if scenario == "unsupported-discard" && (err == nil || !strings.Contains(err.Error(), "discard command transport unavailable")) {
				t.Fatal("discard-less runner reached later validation", err)
			}
			if err == nil || runner.calls != 0 || len(result.controls) != 0 || result.receipt.Client() != "" {
				t.Fatal("invalid native input reached commands or returned selection", err, runner.calls)
			}
		})
	}
}

type pipxNativeUnprofiledObserver struct{}

func (*pipxNativeUnprofiledObserver) Start(context.Context, string) (TraceReader, error) {
	panic("unexpected fallback")
}

func TestPipxNativeAccountedRuntimeBindsOnlyTrustedLease(t *testing.T) {
	data, wheels, runtime, output := pipxNativeRuntimeFixture(t)
	transaction, err := domain.NewSandboxSessionID()
	if err != nil {
		t.Fatal(err)
	}
	lease := ObservationResourceLease{CgroupParent: "haaobsfixture.slice", CPU: 3}
	args, err := pipxNativeAccountedArguments(data, wheels, runtime, output, transaction.String(), lease)
	if err != nil {
		t.Fatal(err)
	}
	for flag, want := range map[string]string{"--network": "none", "--runtime": gVisorRuntimeName, "--cpuset-cpus": "3", "--cgroup-parent": lease.CgroupParent} {
		count := 0
		for i := 0; i < len(args)-1; i++ {
			if args[i] == flag {
				count++
				if args[i+1] != want {
					t.Fatal("unbound runtime field", flag)
				}
			}
		}
		if count != 1 {
			t.Fatal("missing or duplicate runtime field", flag)
		}
	}
	for _, want := range []string{closureVolumeLabel + "=" + transaction.String(), closureManifestLabel + "=" + output.manifestID, "io.heliopause.observation-phase=preparation"} {
		found := false
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "--label" && args[i+1] == want {
				found = true
			}
		}
		if !found {
			t.Fatal("resource label missing", want)
		}
	}
	for _, arg := range args {
		if arg == "--user" || arg == "--env-file" || strings.HasPrefix(arg, "DOCKER_HOST=") {
			t.Fatal("runtime override", arg)
		}
	}
	for _, bad := range []ObservationResourceLease{{CgroupParent: lease.CgroupParent, CPU: -1}, {CPU: 3}} {
		if _, err := pipxNativeAccountedArguments(data, wheels, runtime, output, transaction.String(), bad); err == nil {
			t.Fatal("invalid lease accepted")
		}
	}
	if _, err := pipxNativeAccountedArguments(data, wheels, runtime, output, "../foreign", lease); err == nil {
		t.Fatal("invalid transaction accepted")
	}
}

// A lost CREATE response does not authorize adopting a foreign container. A
// failed observer stream must not prevent independent owned-container cleanup.
func TestPipxNativePartialCleanupRetainsUncertainOwnerAndClosesResources(t *testing.T) {
	for _, scenario := range []string{"owned-unknown-create", "foreign-owner", "inventory-failed"} {
		t.Run(scenario, func(t *testing.T) {
			id := strings.Repeat("a", 64)
			owner := "sbx_partial"
			keeper := &pipxNativeKeeper{name: "haa-uv-resolver-" + owner, owner: owner, attempted: true}
			actualOwner := owner
			if scenario == "foreign-owner" {
				actualOwner = "foreign"
			}
			inspection, err := json.Marshal(map[string]any{"Id": id, "Name": "/" + keeper.name, "Config": map[string]any{"Image": pythonImageReference, "Labels": map[string]string{uvResolverOwnerLabel: actualOwner}}})
			if err != nil {
				t.Fatal(err)
			}
			runner := &uvOwnedContainerRunner{inventory: []byte(id + "\n"), inspection: inspection}
			if scenario == "inventory-failed" {
				runner.queryErr = errors.New("test inventory unavailable")
			}
			resources := &pipxNativeCleanupResources{closeErr: errors.New("test final counter unavailable")}
			session := pipxNativeSession{runner: runner, resources: resources, resourceCreated: true, keepers: []*pipxNativeKeeper{keeper}, trace: terminationDiagnosticTrace{err: errors.New("test observer unavailable")}}
			err = session.close(context.Background())
			if err == nil || !strings.Contains(err.Error(), "observation incomplete") || !errors.Is(err, resources.closeErr) || resources.closes != 1 {
				t.Fatal("partial cleanup hid observation/accounting failure or skipped close", err)
			}
			if scenario == "owned-unknown-create" {
				if runner.removed != id || !keeper.removed {
					t.Fatal("attested unknown CREATE not drained")
				}
			} else if runner.removed != "" || keeper.removed {
				t.Fatal("uncertain or foreign owner removed")
			}
		})
	}
}

type pipxNativeCleanupResources struct {
	ObservationResourceClient
	terminateErr, closeErr error
	terminated, closes     int
}

func (r *pipxNativeCleanupResources) Terminate(context.Context, string, string) (ObservationResourceLease, error) {
	r.terminated++
	return ObservationResourceLease{}, r.terminateErr
}
func (r *pipxNativeCleanupResources) Close(context.Context, string) (ObservationResourceLease, error) {
	r.closes++
	return ObservationResourceLease{}, r.closeErr
}

func TestPipxNativeLostAccountingStillDrainsExactOwnedRuntime(t *testing.T) {
	id := strings.Repeat("b", 64)
	keeper := &pipxNativeKeeper{id: id, attempted: true}
	runner := &pipxNativeDiscardCleanupRunner{}
	resources := &pipxNativeCleanupResources{terminateErr: errors.New("test terminate unavailable"), closeErr: errors.New("test close unavailable")}
	session := pipxNativeSession{runner: runner, resources: resources, reader: keeper, keepers: []*pipxNativeKeeper{keeper}, registrationAttempted: true, resourceCreated: true}
	err := session.close(context.Background())
	if !errors.Is(err, resources.terminateErr) || !errors.Is(err, resources.closeErr) || resources.terminated != 1 || resources.closes != 1 || runner.removed != id || !keeper.removed {
		t.Fatal("lost accounting hid failure or skipped exact drainage", err, resources)
	}
}

type pipxNativeDiscardCommands struct{ *pipxNativeNoCommands }

func (r *pipxNativeDiscardCommands) RunDiscard(context.Context, string, ...string) error {
	r.calls++
	return errors.New("unexpected discard")
}

type pipxNativeDiscardCleanupRunner struct{ uvOwnedContainerRunner }

func (r *pipxNativeDiscardCleanupRunner) RunDiscard(ctx context.Context, name string, args ...string) error {
	_, err := r.Output(ctx, name, args...)
	return err
}

type pipxNativeDelayedCleanupResources struct {
	ObservationResourceClient
	lease      ObservationResourceLease
	closes     int
	stopBudget time.Duration
}

func (r *pipxNativeDelayedCleanupResources) Terminate(ctx context.Context, _ string, _ string) (ObservationResourceLease, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return ObservationResourceLease{}, errors.New("unbounded stop")
	}
	r.stopBudget = time.Until(deadline)
	select {
	case <-time.After(4 * time.Second):
		return r.lease, nil
	case <-ctx.Done():
		return ObservationResourceLease{}, ctx.Err()
	}
}
func (r *pipxNativeDelayedCleanupResources) Close(ctx context.Context, _ string) (ObservationResourceLease, error) {
	r.closes++
	return r.lease, ctx.Err()
}
func TestPipxNativeTrustedCleanupDoesNotTruncateLegitimateStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		resources := &pipxNativeDelayedCleanupResources{lease: ObservationResourceLease{CgroupParent: "owned", CPU: 2}}
		runner := &pipxNativeNoCommands{}
		keeper := &pipxNativeKeeper{id: strings.Repeat("c", 64), attempted: true}
		session := pipxNativeSession{runner: runner, resources: resources, lease: resources.lease, reader: keeper, keepers: []*pipxNativeKeeper{keeper}, registrationAttempted: true, resourceCreated: true}
		started := time.Now()
		err := session.cleanup()
		if err != nil || time.Since(started) != 4*time.Second || resources.stopBudget != 7*time.Second || resources.closes != 1 || !keeper.removed || runner.calls != 0 {
			t.Fatal("trusted teardown truncated or stop became unbounded", err, resources, runner.calls)
		}
	})
}
