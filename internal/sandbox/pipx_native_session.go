package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// This phase returns unapproved native metadata, never an inspection outcome.
// The existing zero-unit ledger is used only for operational CPU accounting.
type pipxNativeSession struct {
	runner                CommandRunner
	observer              TraceObserver
	resources             ObservationResourceClient
	transaction           string
	lease                 ObservationResourceLease
	ledger                *observationTransaction
	watch                 *observationCPUWatch
	data                  *pipxDataVolume
	wheels                *pipxWheelVolume
	runtime               nativeRuntimeVolume
	output                closureVolume
	keepers               []*pipxNativeKeeper
	reader                *pipxNativeKeeper
	registered            bool
	registrationAttempted bool
	resourceCreated       bool
	trace                 TraceReader
}

type pipxNativeKeeper struct {
	name, owner, id    string
	attempted, removed bool
}

func pipxNativeRequirementMatches(requirement artifactpypi.UpgradeRequirement, frozen *pipxNativeInput) bool {
	if frozen == nil || !frozen.valid() {
		return false
	}
	for _, node := range frozen.graph.Nodes() {
		if node.Node() != frozen.graph.Primary() {
			continue
		}
		identity := node.Artifact().Identity()
		return identity.Name() == requirement.Project() && identity.Source() == artifactpypi.PublicPyPIProfile().Source() && requirement.AcceptsVersion(identity.Version())
	}
	return false
}

func pipxNativeAccountedArguments(data *pipxDataVolume, wheels *pipxWheelVolume, runtime nativeRuntimeVolume, output closureVolume, transaction string, lease ObservationResourceLease) ([]string, error) {
	if _, err := domain.ParseSandboxSessionID(transaction); err != nil || lease.CgroupParent == "" || lease.CPU < 0 {
		return nil, errors.New("pipx native resource identity unavailable")
	}
	args, err := pipxNativeCreateArguments(data, wheels, runtime, output)
	if err != nil {
		return nil, err
	}
	// All fields come from the existing authenticated typed resource lease.
	bound := []string{"create", "--cpuset-cpus", strconv.Itoa(lease.CPU), "--cgroup-parent", lease.CgroupParent,
		"--label", closureVolumeLabel + "=" + transaction, "--label", closureManifestLabel + "=" + output.manifestID,
		"--label", "io.heliopause.observation-phase=" + string(phasePreparation)}
	return append(bound, args[1:]...), nil
}

func (s *pipxNativeSession) newKeeper() (*pipxNativeKeeper, error) {
	identity, err := domain.NewSandboxSessionID()
	if err != nil {
		return nil, err
	}
	owner := identity.String()
	k := &pipxNativeKeeper{name: "haa-uv-resolver-" + owner, owner: owner}
	s.keepers = append(s.keepers, k)
	return k, nil
}

func (s *pipxNativeSession) createKeeper(ctx context.Context, k *pipxNativeKeeper, args []string) error {
	if k == nil || len(args) < 2 || args[0] != "create" {
		return errors.New("pipx native keeper arguments unavailable")
	}
	listed, err := s.runner.Output(ctx, "docker", "ps", "--all", "--quiet", "--no-trunc", "--filter", "name=^/"+k.name+"$")
	if err != nil || len(listed) > 1024 || strings.TrimSpace(string(listed)) != "" {
		return errors.New("pipx native keeper name preflight unavailable or nonempty")
	}
	args = append([]string{"create", "--name", k.name, "--label", uvResolverOwnerLabel + "=" + k.owner}, args[1:]...)
	k.attempted = true
	created, err := s.runner.Output(ctx, "docker", args...)
	if err != nil || !exactObservationContainerID(strings.TrimSpace(string(created))) {
		return errors.New("pipx native keeper CREATE identity unavailable")
	}
	k.id = strings.TrimSpace(string(created))
	return verifyOwnedUVResolverContainer(ctx, s.runner, k.name, k.owner, k.id)
}

func (s *pipxNativeSession) dataKeeper(ctx context.Context, user string, mounts ...string) (*pipxNativeKeeper, error) {
	k, err := s.newKeeper()
	if err != nil {
		return nil, err
	}
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	args := []string{"create", "--pull", "never", "--runtime", "runc", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", strconv.FormatInt(policy.RuntimeMemory(), 10), "--cpus", "1", "--ulimit", "cpu=" + strconv.Itoa(policy.RuntimeCPUSecs()) + ":" + strconv.Itoa(policy.RuntimeCPUSecs()), "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=128m,uid=1000,gid=1000,mode=0700"}
	args = append(args, mounts...)
	args = append(args, "--user", user)
	args = append(args, isolatedContainerEnvironmentArguments()...)
	args = append(args, "--entrypoint", "/bin/sleep", pythonImageReference, strconv.FormatInt(int64(policy.Duration()/time.Second), 10))
	if err = s.createKeeper(ctx, k, args); err != nil {
		return nil, err
	}
	if _, err = s.runner.Output(ctx, "docker", "start", k.id); err != nil {
		return nil, err
	}
	if err = verifyObservationRuntimeAlive(ctx, s.runner, k.id); err != nil {
		return nil, err
	}
	return k, nil
}

func (s *pipxNativeSession) removeKeeper(ctx context.Context, k *pipxNativeKeeper) error {
	if k == nil || !k.attempted || k.removed {
		return nil
	}
	if err := removeOwnedUVResolverContainer(ctx, s.runner, k.name, k.owner, k.id); err != nil {
		return err
	}
	k.removed = true
	return nil
}

func (s *pipxNativeSession) prepare(ctx context.Context, tool *pipxPreparedTool, frozen *pipxNativeInput) error {
	var err error
	s.data, err = createPipxDataVolume(ctx, s.runner, s.transaction)
	if err != nil {
		return err
	}
	s.wheels, err = createPipxWheelVolume(ctx, s.runner, s.transaction, frozen)
	if err != nil {
		return err
	}
	zipRW, err := s.data.preparationMount()
	if err != nil {
		return err
	}
	wheelsRW, err := s.wheels.preparationMount()
	if err != nil {
		return err
	}
	writer, err := s.dataKeeper(ctx, "0:0", "--mount", zipRW, "--mount", wheelsRW)
	if err != nil {
		return err
	}
	if err = s.data.seed(ctx, s.runner, tool, writer.id); err != nil {
		return err
	}
	if err = s.wheels.seed(ctx, s.runner, writer.id); err != nil {
		return err
	}
	zipRO, err := s.data.readerMount()
	if err != nil {
		return err
	}
	wheelsRO, err := s.wheels.readerMount()
	if err != nil {
		return err
	}
	anchor, err := s.dataKeeper(ctx, "1000:1000", "--mount", zipRO, "--mount", wheelsRO)
	if err != nil {
		return err
	}
	if err = s.data.bindAnchor(ctx, s.runner, anchor.id); err != nil {
		return err
	}
	if err = s.wheels.bindAnchor(ctx, s.runner, anchor.id); err != nil {
		return err
	}
	if err = s.removeKeeper(ctx, writer); err != nil {
		return err
	}
	if err = s.data.seal(ctx, s.runner); err != nil {
		return err
	}
	if err = s.wheels.seal(ctx, s.runner); err != nil {
		return err
	}
	s.runtime, err = createUVRuntimeVolume(ctx, s.runner)
	if err != nil {
		return err
	}
	s.output, err = createClosureVolume(ctx, s.runner, s.transaction+"-native-output", frozen.id, artifactpypi.PublicPyPIProfile().ResourcePolicy().PromotionTmpfs())
	return err
}

func (s *pipxNativeSession) start(ctx context.Context, cancel context.CancelFunc) error {
	args, err := pipxNativeAccountedArguments(s.data, s.wheels, s.runtime, s.output, s.transaction, s.lease)
	if err != nil {
		return err
	}
	s.reader, err = s.newKeeper()
	if err != nil {
		return err
	}
	if err = s.createKeeper(ctx, s.reader, args); err != nil {
		return err
	}
	s.trace, err = s.observer.(profiledTraceObserver).StartProfile(ctx, s.reader.id, pipxNativeResolverProfile)
	if err != nil {
		return err
	}
	if _, err = s.runner.Output(ctx, "docker", "start", s.reader.id); err != nil {
		return err
	}
	s.registrationAttempted = true
	registered, err := s.resources.Register(ctx, s.transaction, s.reader.id, string(phasePreparation))
	if err != nil {
		return err
	}
	if registered.CgroupParent != s.lease.CgroupParent || registered.CPU != s.lease.CPU {
		return errors.New("pipx native cgroup registration differs from lease")
	}
	s.registered = true
	// Capture immutable IDs; the emergency callback never changes keeper state.
	id := s.reader.id
	s.watch, err = startObservationCPUWatch(ctx, s.resources, s.transaction, s.ledger, cancel, func(stop context.Context) error { return s.abortRuntime(stop, id) })
	if err != nil {
		return err
	}
	if err = awaitMountAnchors(ctx, s.observer, id); err != nil {
		return err
	}
	if err = awaitBoundaryHelper(ctx, s.runner, id); err != nil {
		return err
	}
	if err = s.data.verifyReader(ctx, s.runner, id); err != nil {
		return err
	}
	if err = s.wheels.verifyReader(ctx, s.runner, id); err != nil {
		return err
	}
	if err = s.runtime.verifyTarget(ctx, s.runner, id); err != nil {
		return err
	}
	if err = s.output.verifyContainerMount(ctx, s.runner, id, false); err != nil {
		return err
	}
	return s.output.verifyAttachments(ctx, s.runner, map[string]bool{id: false})
}

// The existing typed terminate and exact-ID Docker fallback are independent.
// A lost accounting transport must not delay draining this owned runtime.
func (s *pipxNativeSession) abortRuntime(ctx context.Context, id string) error {
	done := make(chan error, 1)
	go func() { _, err := s.resources.Terminate(ctx, s.transaction, id); done <- err }()
	removal := discardCommand(ctx, s.runner, "docker", "rm", "--force", id)
	remaining, query := s.runner.Output(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "id="+id, "--format", "{{.ID}}")
	helper := <-done
	if helper != nil || (removal != nil && (query != nil || strings.TrimSpace(string(remaining)) != "")) || query != nil || len(remaining) > 1024 || strings.TrimSpace(string(remaining)) != "" {
		return errors.Join(helper, errors.New("pipx native runtime drainage is unproven"))
	}
	return nil
}

// runInstalledPipxNativeInstall binds the fixed protected current supply to
// already frozen original bytes. It returns unapproved metadata only and accepts
// no caller-selected supply path, digest, ZIP bytes or runtime override.
func runInstalledPipxNativeInstall(ctx context.Context, runner CommandRunner, observer TraceObserver, resources ObservationResourceClient, frozen *pipxNativeInput, requirement artifactpypi.UpgradeRequirement) (pipxNativeOutput, error) {
	if ctx == nil || runner == nil || observer == nil || resources == nil || !pipxNativeRequirementMatches(requirement, frozen) {
		return pipxNativeOutput{}, errors.New("pipx installed native input unavailable")
	}
	if err := ctx.Err(); err != nil {
		return pipxNativeOutput{}, err
	}
	tool, err := preparePipxTool(ctx)
	if err != nil {
		return pipxNativeOutput{}, err
	}
	return runPipxNativeInstall(ctx, runner, observer, resources, tool, frozen, requirement)
}

// runPipxNativeInstall consumes a previously prepared fixed ZIP and a complete
// frozen selection. Installed-source composition and upgrades are separate.
func runPipxNativeInstall(ctx context.Context, runner CommandRunner, observer TraceObserver, resources ObservationResourceClient, tool *pipxPreparedTool, frozen *pipxNativeInput, requirement artifactpypi.UpgradeRequirement) (result pipxNativeOutput, resultErr error) {
	if ctx == nil || runner == nil || observer == nil || resources == nil || tool == nil || !pipxNativeRequirementMatches(requirement, frozen) {
		return result, errors.New("pipx native session input unavailable")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if _, ok := observer.(profiledTraceObserver); !ok {
		return result, errors.New("pipx native profile observer unavailable")
	}
	if _, ok := observer.(mountAnchorReadyObserver); !ok {
		return result, errors.New("pipx native mount observer unavailable")
	}
	raw := runner
	if admitted, ok := raw.(*admissionAwareCommandRunner); ok {
		raw = admitted.runner
	}
	if _, ok := raw.(inputCommandRunner); !ok {
		return result, errors.New("pipx native input transport unavailable")
	}
	if _, ok := raw.(streamingDockerOutput); !ok {
		return result, errors.New("pipx native archive transport unavailable")
	}
	if _, ok := raw.(discardCommandRunner); !ok {
		return result, errors.New("pipx native discard command transport unavailable")
	}
	if _, ok := raw.(boundedCommandRunner); !ok {
		return result, errors.New("pipx native bounded command transport unavailable")
	}
	spec, ok := nativeRuntimeToolIdentity(nativeRuntimePipx)
	digest := sha256.Sum256(tool.body)
	if !ok || int64(len(tool.body)) != spec.size || hex.EncodeToString(digest[:]) != spec.sha256 {
		return result, errPipxToolSupply
	}
	command, err := pipxNativeArguments(pipxNativeInstall, requirement)
	if err != nil {
		return result, err
	}
	now := time.Now()
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	accounting, err := observationPolicyForPython(policy, pipxNativeResolverProfile, 0, now)
	if err != nil {
		return result, err
	}
	ledger, err := newObservationTransaction(accounting, nil, now)
	if err != nil {
		return result, err
	}
	runCtx, cancel := context.WithDeadline(ctx, accounting.wallDeadline)
	defer cancel()
	preparedRunner := admissionAwareRunner(runner)
	uv, err := prepareUVTool(runCtx, preparedRunner)
	if err != nil {
		return result, err
	}
	identity, err := domain.NewSandboxSessionID()
	if err != nil {
		return result, err
	}
	s := &pipxNativeSession{runner: preparedRunner, observer: observer, resources: resources, transaction: identity.String(), ledger: ledger}
	defer func() {
		cleanupErr := s.cleanup()
		if err := ctx.Err(); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
		if cleanupErr != nil {
			result = pipxNativeOutput{}
			resultErr = errors.Join(resultErr, fmt.Errorf("pipx native cleanup or qualification failed: %w", cleanupErr))
		}
	}()
	s.lease, err = resources.Create(runCtx, s.transaction, "pypi")
	if err != nil {
		return result, uvReaderFailure{"pipx native resource creation failed", err}
	}
	if s.lease.CgroupParent == "" || s.lease.CPU < 0 {
		return result, errors.New("pipx native resource lease is invalid")
	}
	s.resourceCreated = true
	if err = s.prepare(runCtx, tool, frozen); err != nil {
		return result, uvReaderFailure{"pipx native DATA preparation failed", err}
	}
	if err = s.start(runCtx, cancel); err != nil {
		return result, uvReaderFailure{"pipx native observed startup failed", err}
	}
	if !s.registered || s.watch == nil {
		return result, errors.New("pipx native runtime lacks resource admission")
	}
	if err = uv.introduce(runCtx, preparedRunner, s.reader.id); err != nil {
		return result, err
	}
	if err = s.runtime.verifyTarget(runCtx, preparedRunner, s.reader.id); err != nil {
		return result, err
	}
	// Successful external admission and target exit are necessary, but the
	// complete trace/accounting/cleanup reconciliation in defer is mandatory.
	if _, err = preparedRunner.(boundedCommandRunner).RunBounded(runCtx, "docker", boundaryExecArguments(s.reader.id, boundaryPythonHandoffMode, command...)...); err != nil {
		return result, uvReaderFailure{"pipx native install failed", err}
	}
	if err = s.data.verifyReader(runCtx, preparedRunner, s.reader.id); err != nil {
		return result, err
	}
	if err = s.wheels.verifyReader(runCtx, preparedRunner, s.reader.id); err != nil {
		return result, err
	}
	if err = s.output.verifyAttachments(runCtx, preparedRunner, map[string]bool{s.reader.id: false}); err != nil {
		return result, err
	}
	result, err = readPipxNativeOutput(runCtx, preparedRunner, s.reader.id, requirement.Project(), frozen)
	if err != nil {
		return pipxNativeOutput{}, err
	}
	if !result.receipt.Requirement().SameConstraints(requirement) {
		return pipxNativeOutput{}, errors.New("pipx native request changed")
	}
	return result, nil
}

func (s *pipxNativeSession) collectTrace(ctx context.Context) error {
	_, limitation, diagnostic := collectTraceDiagnostic(ctx, s.trace)
	if limitation != "" || !diagnostic.SessionComplete {
		return fmt.Errorf("pipx native observation incomplete: %s", diagnostic.String())
	}
	if diagnostic.KindCounts["process-exec-expected"] == 0 {
		return errors.New("pipx native observation lacks expected execution")
	}
	for kind, count := range diagnostic.KindCounts {
		if count > 0 && kind != "process-exec-expected" && kind != "filesystem-workspace-access" {
			return fmt.Errorf("pipx native observation contains actionable or unsupported kind: %s", kind)
		}
	}
	return nil
}

// Match the existing Python transaction's trusted cleanup bound. Guest CPU,
// wall authorization and each typed stop's seven-second bound remain separate.
func (s *pipxNativeSession) cleanup() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.close(ctx)
}

func (s *pipxNativeSession) close(ctx context.Context) error {
	var joined error
	var traceDone chan error
	if s.trace != nil {
		traceDone = make(chan error, 1)
		go func() {
			collect, done := context.WithTimeout(ctx, 7*time.Second)
			defer done()
			traceDone <- s.collectTrace(collect)
		}()
	}
	transition := false
	if s.watch != nil {
		if err := s.watch.beginTermination(ctx); err != nil {
			joined = errors.Join(joined, err)
		} else {
			transition = true
		}
	}
	if s.reader != nil && s.registrationAttempted {
		stop, done := context.WithTimeout(ctx, 7*time.Second)
		lease, err := s.resources.Terminate(stop, s.transaction, s.reader.id)
		done()
		if err == nil && lease.CgroupParent == s.lease.CgroupParent {
			s.reader.removed = true
		} else {
			joined = errors.Join(joined, err, errors.New("pipx native typed runtime termination unqualified"))
			// Keep exact owned-ID drainage independent of accounting availability.
			_ = discardCommand(ctx, s.runner, "docker", "rm", "--force", s.reader.id)
			remaining, query := s.runner.Output(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "id="+s.reader.id, "--format", "{{.ID}}")
			if query == nil && len(remaining) <= 1024 && strings.TrimSpace(string(remaining)) == "" {
				s.reader.removed = true
			} else {
				joined = errors.Join(joined, errors.New("pipx native exact runtime absence unproven"))
			}
		}
	}
	// Remove unregistered/unknown-CREATE keepers only through their existing
	// authenticated name/image/owner inventory protocol.
	for i := len(s.keepers) - 1; i >= 0; i-- {
		joined = errors.Join(joined, s.removeKeeper(ctx, s.keepers[i]))
	}
	if traceDone != nil {
		joined = errors.Join(joined, <-traceDone)
	}
	if transition {
		joined = errors.Join(joined, s.watch.endTermination(ctx))
	}
	if s.output.name != "" {
		joined = errors.Join(joined, closeUVOutputVolume(ctx, s.runner, &s.output))
	}
	if s.runtime.name != "" {
		joined = errors.Join(joined, s.runtime.close(ctx, s.runner))
	}
	if s.wheels != nil {
		joined = errors.Join(joined, s.wheels.close(ctx, s.runner))
	}
	if s.data != nil {
		joined = errors.Join(joined, s.data.close(ctx, s.runner))
	}
	if s.watch != nil {
		joined = errors.Join(joined, s.watch.stop())
	}
	if s.resourceCreated {
		started := time.Now()
		lease, err := s.resources.Close(ctx, s.transaction)
		if err != nil || lease.CgroupParent != s.lease.CgroupParent || lease.CPU != s.lease.CPU {
			joined = errors.Join(joined, err, errors.New("pipx native final resource accounting unqualified"))
		} else if s.watch != nil {
			// No unit is completed here. Reuse only the existing bounded lifecycle
			// counter validation after the watcher has stopped and the keeper drained.
			joined = errors.Join(joined, s.ledger.accountCPUAfterTransition(lease.UsageUsec, started, time.Now(), true))
		}
	}
	if s.ledger != nil && !time.Now().Before(s.ledger.policy.wallDeadline) {
		joined = errors.Join(joined, errors.New("pipx native wall deadline exceeded"))
	}
	return joined
}
