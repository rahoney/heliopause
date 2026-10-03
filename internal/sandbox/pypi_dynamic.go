package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	pythonDynamicTimeout = 45 * time.Second
	pythonSitePath       = "/haa-site"
)

var pythonImportName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*$`)

// PythonWheelRunner is the narrow adapter boundary consumed by PyPI dynamic
// inspection. Import names are normalized static-inspection output, not caller
// supplied module names.
type PythonWheelRunner interface {
	InspectWheel(context.Context, domain.AcquiredArtifact, []string) (domain.SandboxResult, error)
}

// PythonCommandObservation records an externally reconciled target import.
// ZeroExit does not assert normal import return or later command functionality.
type PythonCommandObservation struct {
	Module      string `json:"module"`
	UnitID      string `json:"unit_id"`
	OwnerSHA256 string `json:"owner_sha256"`
	ZeroExit    bool   `json:"zero_exit"`
}

// SandboxCompleted still requires complete trusted lifecycle evidence. Commands
// are externally reconciled bounded outcomes, never interpreter receipts.
type PythonObservationResult struct {
	domain.SandboxResult
	Commands []PythonCommandObservation
}

// PlanAwarePythonWheelRunner accepts the canonical authenticated observation plan.
type PlanAwarePythonWheelRunner interface {
	PythonWheelRunner
	InspectWheelWithPlan(context.Context, domain.AcquiredArtifact, artifactpypi.ObservationPlan, []domain.AcquiredArtifact) (PythonObservationResult, error)
}

// InspectionPrerequisitePythonWheelRunner observes explicitly selected leaf
// inputs and the target together under one transaction authorization. These
// inputs belong to inspection, never to the requested dependency graph.
type InspectionPrerequisitePythonWheelRunner interface {
	PlanAwarePythonWheelRunner
	InspectWheelWithPrerequisites(context.Context, domain.AcquiredArtifact, artifactpypi.ObservationPlan, []domain.AcquiredArtifact, []domain.AcquiredArtifact) (PythonObservationResult, error)
}

// DependencyAwarePythonWheelRunner is the optional graph-install capability
// used when a Python node must import with its already-acquired dependency
// closure present. The legacy single-wheel method remains unchanged.
type DependencyAwarePythonWheelRunner interface {
	PythonWheelRunner
	InspectWheelWithClosure(context.Context, domain.AcquiredArtifact, []string, []domain.AcquiredArtifact) (domain.SandboxResult, error)
}

// NoImportSurfacePythonWheelRunner still performs observed offline installation
// for a statically proven metadata-only wheel. It does not claim an import ran.
type NoImportSurfacePythonWheelRunner interface {
	InspectWheelWithoutImportSurface(context.Context, domain.AcquiredArtifact, []domain.AcquiredArtifact) (domain.SandboxResult, error)
}

type discardCommandRunner interface {
	RunDiscard(context.Context, string, ...string) error
}

// boundedCommandRunner retains a short process-local diagnostic window. The
// Dynamic backend classifies it before returning, and never exposes it in a
// Sandbox Result, Evidence, or CLI output.
type boundedCommandRunner interface {
	RunBounded(context.Context, string, ...string) ([]byte, error)
}

// PythonArtifactIntroducer streams one verified wheel from the HAA intake root
// into a running container. It never puts the Host path into a Docker command.
type PythonArtifactIntroducer struct {
	intakeRoot string
	runner     CommandRunner
}

func NewPythonArtifactIntroducer(intakeRoot string, runner CommandRunner) (*PythonArtifactIntroducer, error) {
	if !filepath.IsAbs(intakeRoot) || runner == nil {
		return nil, errors.New("python wheel introducer is not configured")
	}
	return &PythonArtifactIntroducer{intakeRoot: filepath.Clean(intakeRoot), runner: admissionAwareRunner(runner)}, nil
}

func (i *PythonArtifactIntroducer) IntroduceWheel(ctx context.Context, containerID string, artifact domain.AcquiredArtifact) error {
	destination, err := i.validatedWheelDestination(artifact)
	if err != nil {
		return err
	}
	return i.introduceWheelAt(ctx, containerID, artifact, destination)
}

func pythonWheelVariant(variant string) bool { return variant == "wheel" || variant == "derived-wheel" }

func (i *PythonArtifactIntroducer) validatedWheelFilename(artifact domain.AcquiredArtifact) (string, error) {
	if i == nil || !pythonWheelVariant(artifact.Identity().Variant()) {
		return "", errors.New("python wheel filename request is invalid")
	}
	source, err := i.artifactPath(artifact.ContentHandle(), artifact.Identity().Variant())
	if err != nil {
		return "", err
	}
	recordName := "filename"
	if artifact.Identity().Variant() == "derived-wheel" {
		recordName = "derived-filename"
	}
	recordPath := filepath.Join(filepath.Dir(source), recordName)
	info, err := os.Lstat(recordPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("verified Python intake filename is unavailable")
	}
	filenameBytes, err := os.ReadFile(recordPath)
	if err != nil {
		return "", errors.New("verified Python intake filename is unavailable")
	}
	filename := string(filenameBytes)
	if filename == "" || filename == "." || filename == ".." || filepath.IsAbs(filename) || filepath.Base(filename) != filename || strings.ContainsAny(filename, `/\\`) {
		return "", errors.New("verified Python intake filename is invalid")
	}
	project, version, _, _, _, err := artifactpypi.ParseWheelFilenameForSource(filename, artifact.Identity().Source())
	if err != nil || project != artifact.Identity().Name() || version != artifact.Identity().Version() {
		return "", errors.New("verified Python intake filename does not match Artifact")
	}
	return filename, nil
}

func (i *PythonArtifactIntroducer) validatedWheelDestination(artifact domain.AcquiredArtifact) (string, error) {
	filename, err := i.validatedWheelFilename(artifact)
	if err != nil {
		return "", err
	}
	return "/tmp/" + filename, nil
}

type validatedWheel struct {
	artifact    domain.AcquiredArtifact
	destination string
}

func (i *PythonArtifactIntroducer) validatedWheelDestinations(target domain.AcquiredArtifact, closure []domain.AcquiredArtifact) ([]validatedWheel, error) {
	if len(closure) == 0 {
		return nil, errors.New("python dynamic dependency closure is empty")
	}
	targetSeen := false
	validated := make([]validatedWheel, 0, len(closure))
	seen := make(map[string]bool, len(closure))
	for _, item := range closure {
		if !pythonWheelVariant(item.Identity().Variant()) || item.ContentHandle() == "" || item.Digest().String() == "" {
			return nil, errors.New("python dynamic dependency closure contains unsupported artifact")
		}
		destination, err := i.validatedWheelDestination(item)
		if err != nil {
			return nil, err
		}
		if seen[destination] {
			return nil, errors.New("python dynamic dependency closure contains duplicate artifact path")
		}
		seen[destination] = true
		validated = append(validated, validatedWheel{artifact: item, destination: destination})
		if sameArtifactIdentity(item, target) {
			targetSeen = true
		}
	}
	if !targetSeen {
		return nil, errors.New("python dynamic dependency closure omits target artifact")
	}
	return validated, nil
}

// PythonDynamicBackend owns bounded, independent runsc-trace observations for
// a statically declared wheel surface. It returns observations only;
// it never creates Findings, Evidence, Policy or Promotion state.
type PythonDynamicBackend struct {
	runner       CommandRunner
	introducer   *PythonArtifactIntroducer
	observer     TraceObserver
	probe        func(context.Context) (PythonCapability, error)
	newSessionID func() (domain.SandboxSessionID, error)
	resources    ObservationResourceClient
}

func NewPythonDynamicBackend(runner CommandRunner, introducer *PythonArtifactIntroducer, observer TraceObserver, probe func(context.Context) (PythonCapability, error)) (*PythonDynamicBackend, error) {
	if runner == nil || introducer == nil || observer == nil || probe == nil {
		return nil, errors.New("python dynamic backend is not configured")
	}
	if _, ok := runner.(discardCommandRunner); !ok {
		return nil, errors.New("python dynamic runner must discard command output")
	}
	return &PythonDynamicBackend{runner: admissionAwareRunner(runner), introducer: introducer, observer: observer, probe: probe, newSessionID: domain.NewSandboxSessionID}, nil
}

func (b *PythonDynamicBackend) InspectWheel(ctx context.Context, artifact domain.AcquiredArtifact, imports []string) (domain.SandboxResult, error) {
	return b.InspectWheelWithClosure(ctx, artifact, imports, []domain.AcquiredArtifact{artifact})
}

func (b *PythonDynamicBackend) InspectWheelWithClosure(ctx context.Context, artifact domain.AcquiredArtifact, imports []string, closure []domain.AcquiredArtifact) (domain.SandboxResult, error) {
	return b.inspectWheelWithClosure(ctx, artifact, imports, closure, false)
}

func (b *PythonDynamicBackend) InspectWheelWithoutImportSurface(ctx context.Context, artifact domain.AcquiredArtifact, closure []domain.AcquiredArtifact) (domain.SandboxResult, error) {
	return b.inspectWheelWithClosure(ctx, artifact, nil, closure, true)
}

func (b *PythonDynamicBackend) InspectWheelWithPlan(ctx context.Context, artifact domain.AcquiredArtifact, plan artifactpypi.ObservationPlan, closure []domain.AcquiredArtifact) (PythonObservationResult, error) {
	var commands []PythonCommandObservation
	result, err := b.inspectWheelWithPlanAndPrerequisites(ctx, artifact, plan, closure, nil, &commands)
	return PythonObservationResult{result, commands}, err
}

func (b *PythonDynamicBackend) InspectWheelWithPrerequisites(ctx context.Context, artifact domain.AcquiredArtifact, plan artifactpypi.ObservationPlan, closure, prerequisites []domain.AcquiredArtifact) (PythonObservationResult, error) {
	if len(prerequisites) == 0 {
		return PythonObservationResult{}, errors.New("inspection prerequisites must be explicit")
	}
	var commands []PythonCommandObservation
	result, err := b.inspectWheelWithPlanAndPrerequisites(ctx, artifact, plan, closure, prerequisites, &commands)
	return PythonObservationResult{result, commands}, err
}

func (b *PythonDynamicBackend) inspectWheelWithPlanAndPrerequisites(ctx context.Context, artifact domain.AcquiredArtifact, plan artifactpypi.ObservationPlan, closure, prerequisites []domain.AcquiredArtifact, commands *[]PythonCommandObservation) (domain.SandboxResult, error) {
	if b == nil || b.runner == nil || b.introducer == nil || b.observer == nil || b.probe == nil || b.newSessionID == nil || ctx == nil {
		return domain.SandboxResult{}, errors.New("python dynamic inspection request is invalid")
	}
	resourcePolicy := artifactpypi.ResourcePolicyFromContext(ctx)
	if plan.Project != artifact.Identity().Name() || plan.Version != artifact.Identity().Version() ||
		artifactpypi.ValidateTypedObservationPlan(plan, resourcePolicy) != nil {
		return domain.SandboxResult{}, errors.New("python dynamic observation plan does not reconcile")
	}
	if _, err := b.introducer.validatedWheelDestinations(artifact, closure); err != nil {
		return domain.SandboxResult{}, err
	}
	sessionID, err := b.newSessionID()
	if err != nil {
		return domain.SandboxResult{}, err
	}
	if !plan.Admissible() {
		return pythonIncomplete(sessionID, "M5_PYPI_DYNAMIC_SURFACE_UNSUPPORTED")
	}
	// Only the controller-owned transaction ledger can qualify a plan. Process
	// output, Python-side markers and successful exit alone have no authority.
	return b.executeTrustedTransaction(ctx, sessionID, artifact, plan, closure, commands, prerequisites...)
}

func (b *PythonDynamicBackend) inspectWheelWithClosure(ctx context.Context, artifact domain.AcquiredArtifact, imports []string, closure []domain.AcquiredArtifact, noImportSurface bool) (domain.SandboxResult, error) {
	if b == nil || ctx == nil {
		return domain.SandboxResult{}, errors.New("python dynamic inspection request is invalid")
	}
	resourcePolicy := artifactpypi.ResourcePolicyFromContext(ctx)
	if noImportSurface {
		if len(imports) != 0 {
			return domain.SandboxResult{}, errors.New("python dynamic inspection request is invalid")
		}
		plan := artifactpypi.ObservationPlan{
			Project:         artifact.Identity().Name(),
			Version:         artifact.Identity().Version(),
			NoImportSurface: true,
		}
		result, err := b.InspectWheelWithPlan(ctx, artifact, plan, closure)
		return result.SandboxResult, err
	}

	if len(imports) == 0 || len(imports) > resourcePolicy.MaxObservationImportsPerArtifact() {
		return domain.SandboxResult{}, errors.New("python dynamic inspection request is invalid")
	}
	seen := make(map[string]bool, len(imports))
	for _, name := range imports {
		if !pythonImportName.MatchString(name) || seen[name] {
			return domain.SandboxResult{}, errors.New("python dynamic inspection request is invalid")
		}
		seen[name] = true
	}
	imports = append([]string(nil), imports...)
	sort.Strings(imports)
	plan := artifactpypi.ObservationPlan{
		Project:                  artifact.Identity().Name(),
		Version:                  artifact.Identity().Version(),
		ImportCandidates:         imports,
		RequiredImportCandidates: append([]string(nil), imports...),
		TotalImportCount:         len(imports),
	}
	for _, name := range imports {
		plan.Units = append(plan.Units, artifactpypi.PlannedObservationUnit{Kind: artifactpypi.DirectImportUnit, Candidate: name})
	}
	result, err := b.InspectWheelWithPlan(ctx, artifact, plan, closure)
	return result.SandboxResult, err
}

func (i *PythonArtifactIntroducer) introduceWheelAt(ctx context.Context, containerID string, artifact domain.AcquiredArtifact, destination string) error {
	return i.introduce(ctx, containerID, artifact, destination, artifact.Identity().Variant())
}

func sameArtifactIdentity(left, right domain.AcquiredArtifact) bool {
	leftIdentity, rightIdentity := left.Identity(), right.Identity()
	return leftIdentity.Source() == rightIdentity.Source() && leftIdentity.Name() == rightIdentity.Name() && leftIdentity.Version() == rightIdentity.Version() && leftIdentity.Variant() == rightIdentity.Variant() && left.Digest() == right.Digest()
}

func discardCommand(ctx context.Context, runner CommandRunner, binary string, arguments ...string) error {
	discarder, ok := runner.(discardCommandRunner)
	if !ok {
		return errors.New("sandbox command runner must discard command output")
	}
	return discarder.RunDiscard(ctx, binary, arguments...)
}

func pythonIncomplete(sessionID domain.SandboxSessionID, code string) (domain.SandboxResult, error) {
	return domain.NewSandboxResult(sessionID, domain.SandboxIncomplete, code, nil)
}
func pythonDynamicCreateArguments(sessionID domain.SandboxSessionID, resourcePolicy artifactpypi.ResourcePolicy) []string {
	size := strconv.FormatInt(resourcePolicy.RuntimeTmpfs(), 10)
	cpuLimit := strconv.Itoa(resourcePolicy.RuntimeCPUSecs())
	arguments := []string{"create", "--pull", "never", "--runtime", gVisorRuntimeName, "--network", "none", "--read-only", "--cap-drop", "ALL", "--cap-add", "SETUID", "--cap-add", "SETGID", "--cap-add", "SETPCAP", "--security-opt", "no-new-privileges", "--pids-limit", "64", "--memory", strconv.FormatInt(resourcePolicy.RuntimeMemory(), 10), "--cpus", "1", "--ulimit", "cpu=" + cpuLimit + ":" + cpuLimit, "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=" + size + ",uid=1000,gid=1000,mode=0700", "--tmpfs", pythonSitePath + ":rw,exec,nosuid,nodev,size=" + size + ",uid=1000,gid=1000,mode=0700", "--tmpfs", boundaryHelperMount}
	arguments = append(arguments, isolatedContainerEnvironmentArguments()...)
	return append(arguments, "--name", "heliopause-pypi-"+sessionID.String(), pythonImageReference, "/bin/sh", "-ceu", boundaryContainerCommand())
}

func pythonDynamicObserverProfile(rootProfileName string) (string, error) {
	switch rootProfileName {
	case "pypi":
		return "pypi-wheel", nil
	case "pytorch:cpu":
		return "pypi-wheel-pytorch-cpu", nil
	case "pytorch:cu126":
		return "pypi-wheel-pytorch-cu126", nil
	case "pytorch:cu130":
		return "pypi-wheel-pytorch-cu130", nil
	case "pytorch:cu132":
		return "pypi-wheel-pytorch-cu132", nil
	default:
		return "", errors.New("unsupported Python root source profile for dynamic observer")
	}
}
