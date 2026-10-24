package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// DependencySetPolicy is the deterministic, graph-level decision boundary for
// an M4 Install. It receives only a complete normalized inspected set.
type DependencySetPolicy interface {
	EvaluateSet(domain.InspectedDependencySet) (domain.PolicyDecision, error)
}

// dependencyAwareInspection is an optional graph capability. It exposes only
// normalized Domain values, so Application never depends on an ecosystem
// inspector's static-analysis state.
type dependencyAwareInspection interface {
	InspectGraph(context.Context, domain.LockedDependencyGraph, map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error)
}

type pendingDependencyInspection struct {
	dependency   domain.LockedDependency
	runID        domain.RunID
	run          *domain.InspectionRun
	artifact     domain.AcquiredArtifact
	verification domain.VerificationReport
	decision     domain.PolicyDecision
	inspection   domain.DependencyInspection
	finalized    bool
}

// InstallInspectService resolves the locked dependency graph and independently
// executes the existing acquire, verify, inspect, evidence, and entry-Policy
// sequence for every graph node before evaluating one set-level Policy.
type InstallInspectService struct {
	resolver       ports.DependencyResolver
	artifact       ports.Artifact
	verification   ports.Verification
	inspection     ports.Inspection
	evidence       ports.Evidence
	entryPolicy    Policy
	setPolicy      DependencySetPolicy
	newOperationID func() (domain.OperationID, error)
	newRunID       func() (domain.RunID, error)
	derivation     ports.Derivation
}

// WithDerivation enables a controller-owned derived-Artifact phase after all
// resolver nodes have completed their normal inspection. It is opt-in so npm
// retains its existing workflow unchanged.
func (s *InstallInspectService) WithDerivation(derivation ports.Derivation) *InstallInspectService {
	if s != nil {
		s.derivation = derivation
	}
	return s
}

// NewInstallInspectService constructs the M4 recursive inspection workflow
// from explicit boundary dependencies.
func NewInstallInspectService(resolver ports.DependencyResolver, artifact ports.Artifact, verification ports.Verification, inspection ports.Inspection, evidence ports.Evidence, entryPolicy Policy, setPolicy DependencySetPolicy, newOperationID func() (domain.OperationID, error), newRunID func() (domain.RunID, error)) (*InstallInspectService, error) {
	if resolver == nil || artifact == nil || verification == nil || inspection == nil || evidence == nil || entryPolicy == nil || setPolicy == nil || newOperationID == nil || newRunID == nil {
		return nil, errors.New("install inspection service requires all ports, policies, and ID generators")
	}
	return &InstallInspectService{
		resolver: resolver, artifact: artifact, verification: verification, inspection: inspection, evidence: evidence,
		entryPolicy: entryPolicy, setPolicy: setPolicy, newOperationID: newOperationID, newRunID: newRunID,
	}, nil
}

// Inspect resolves one exact graph and returns only after every locked entry
// has completed its own inspection. Any resolver or entry failure stops the
// workflow before a set-level decision can be produced.
func (s *InstallInspectService) Inspect(ctx context.Context, request InstallRequest) (InspectedInstall, error) {
	if ctx == nil {
		return InspectedInstall{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return InspectedInstall{}, err
	}
	if request.reference.Source().String() == "" || !request.context.Valid() {
		return InspectedInstall{}, errors.New("validated Install request is required")
	}
	operationID, err := s.newOperationID()
	if err != nil {
		return InspectedInstall{}, fmt.Errorf("generate operation ID: %w", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "resolve"})
	resolution, err := s.resolver.ResolveDependencies(ctx, request.reference, request.context)
	if err != nil {
		return newPartialInspectedInstall(operationID, request, domain.DependencyResolution{}), fmt.Errorf("resolve locked dependency graph: %w", err)
	}
	partial := newPartialInspectedInstall(operationID, request, resolution)
	graph := resolution.Graph()
	if graphInspector, ok := s.inspection.(dependencyAwareInspection); ok && len(graph.Edges()) != 0 {
		return s.inspectGraphAware(ctx, operationID, request, partial, graph, graphInspector)
	}
	inspections := make([]domain.DependencyInspection, 0, len(graph.Nodes()))
	policyDiagnostics := make([]DependencyPolicyDiagnostic, 0, len(graph.Nodes()))
	for _, dependency := range graph.Nodes() {
		inspection, diagnostic, inspectionErr := s.inspectDependency(ctx, operationID, dependency)
		if inspectionErr != nil {
			return partial, inspectionErr
		}
		inspections = append(inspections, inspection)
		policyDiagnostics = append(policyDiagnostics, diagnostic)
	}
	if s.derivation != nil {
		derived, deriveErr := s.derivation.Derive(ctx, inspections)
		if deriveErr != nil {
			return partial, fmt.Errorf("derive verified Artifact: %w", deriveErr)
		}
		graph, err = domain.ExtendLockedDependencyGraph(graph, derived)
		if err != nil {
			return partial, fmt.Errorf("extend derived dependency graph: %w", err)
		}
		for _, item := range derived {
			inspection, inspectErr := s.inspectAcquiredDependency(ctx, operationID, item.Node(), item.Artifact(), []domain.CheckExecution{item.Check()}, []domain.Evidence{item.Evidence()})
			if inspectErr != nil {
				return partial, inspectErr
			}
			inspections = append(inspections, inspection)
		}
	}
	set, err := domain.NewInspectedDependencySet(graph, inspections)
	if err != nil {
		return partial, fmt.Errorf("construct inspected dependency set: %w", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "policy"})
	decision, err := s.setPolicy.EvaluateSet(set)
	if err != nil {
		return partial, fmt.Errorf("evaluate dependency set policy: %w", err)
	}
	return newInspectedInstall(operationID, request, resolution, set, decision).withDependencyPolicyDiagnostics(policyDiagnostics), nil
}

func (s *InstallInspectService) inspectGraphAware(ctx context.Context, operationID domain.OperationID, request InstallRequest, partial InspectedInstall, graph domain.LockedDependencyGraph, inspector dependencyAwareInspection) (InspectedInstall, error) {
	inspections, reports, policyDiagnostics, err := s.inspectDependencyBatch(ctx, operationID, graph.Nodes(), func(artifacts map[domain.DependencyNodeID]domain.AcquiredArtifact) (map[domain.DependencyNodeID]domain.InspectionReport, error) {
		return inspector.InspectGraph(ctx, graph, artifacts)
	})
	if err != nil {
		return partial, err
	}
	if s.derivation != nil {
		derived, deriveErr := s.derivation.Derive(ctx, inspections)
		if deriveErr != nil {
			return partial, fmt.Errorf("derive verified Artifact: %w", deriveErr)
		}
		graph, err = domain.ExtendLockedDependencyGraph(graph, derived)
		if err != nil {
			return partial, fmt.Errorf("extend derived dependency graph: %w", err)
		}
		for _, item := range derived {
			inspection, inspectErr := s.inspectAcquiredDependency(ctx, operationID, item.Node(), item.Artifact(), []domain.CheckExecution{item.Check()}, []domain.Evidence{item.Evidence()})
			if inspectErr != nil {
				return partial, inspectErr
			}
			inspections = append(inspections, inspection)
		}
	}
	set, err := domain.NewInspectedDependencySet(graph, inspections)
	if err != nil {
		return partial, fmt.Errorf("construct inspected dependency set: %w", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "policy"})
	decision, err := s.setPolicy.EvaluateSet(set)
	if err != nil {
		return partial, fmt.Errorf("evaluate dependency set policy: %w", err)
	}
	diagnostics, err := graphStaticDiagnostics(graph, reports)
	if err != nil {
		return partial, fmt.Errorf("construct graph static diagnostics: %w", err)
	}
	dynamicDiagnostics, err := graphDynamicInstallDiagnostics(graph, reports)
	if err != nil {
		return partial, fmt.Errorf("construct graph dynamic install diagnostics: %w", err)
	}
	return newInspectedInstall(operationID, request, partial.Resolution(), set, decision).withGraphStaticDiagnostics(diagnostics).withGraphDynamicInstallDiagnostics(dynamicDiagnostics).withDependencyPolicyDiagnostics(policyDiagnostics), nil
}

func acquiredMatchesLocked(acquired domain.AcquiredArtifact, dependency domain.LockedDependency, runID domain.RunID) bool {
	locked, actual := dependency.Artifact().Identity(), acquired.Identity()
	if locked.Source() != actual.Source() || locked.Name() != actual.Name() || locked.Version() != actual.Version() || locked.Variant() != actual.Variant() || acquired.Digest().String() == "" {
		return false
	}
	declared, ok := acquired.DeclaredIntegrity()
	return ok && declared == dependency.Artifact().DeclaredIntegrity() && acquired.ContentHandle() == "intake:"+runID.String()+":"+actual.Variant()
}

func validateDependencyReports(nodes []domain.LockedDependency, reports map[domain.DependencyNodeID]domain.InspectionReport) error {
	if len(reports) != len(nodes) {
		return errors.New("graph inspection reports do not cover every dependency")
	}
	for _, dependency := range nodes {
		report, ok := reports[dependency.Node()]
		if !ok || len(report.Executions()) == 0 {
			return fmt.Errorf("graph inspection report is invalid for node %s", dependency.Node().String())
		}
	}
	return nil
}

func graphStaticDiagnostics(graph domain.LockedDependencyGraph, reports map[domain.DependencyNodeID]domain.InspectionReport) ([]GraphStaticDiagnostic, error) {
	diagnostics := make([]GraphStaticDiagnostic, 0)
	for _, dependency := range graph.Nodes() {
		report := reports[dependency.Node()]
		reportDiagnostics := report.Diagnostics()
		if len(reportDiagnostics) == 0 {
			continue
		}
		diagnostic, err := newGraphStaticDiagnostic(dependency, reportDiagnostics[0])
		if err != nil {
			return nil, err
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics, nil
}

const (
	dynamicInstallFailurePrefix  = "M5_PYPI_DYNAMIC_INSTALL_FAILED_"
	dynamicImportFailurePrefix   = "M5_PYPI_DYNAMIC_IMPORT_FAILED_"
	dynamicObserverFailurePrefix = "M5_PYPI_DYNAMIC_OBSERVER_FAILED_"
	dynamicReasonInstall         = "PYPI_DYNAMIC_INSTALL_FAILED"
	dynamicReasonImport          = "PYPI_DYNAMIC_IMPORT_FAILED"
	dynamicReasonObserver        = "PYPI_DYNAMIC_OBSERVER_FAILED"
)

func graphDynamicInstallDiagnostics(graph domain.LockedDependencyGraph, reports map[domain.DependencyNodeID]domain.InspectionReport) ([]GraphDynamicInstallDiagnostic, error) {
	diagnostics := make([]GraphDynamicInstallDiagnostic, 0)
	for _, dependency := range graph.Nodes() {
		for _, execution := range reports[dependency.Node()].Executions() {
			if execution.ID().String() != "pypi-dynamic-import" && execution.ID().String() != "pypi-dynamic-import-not-applicable" && execution.ID().String() != "pypi-dynamic-import-supplemented" {
				continue
			}
			limitation, limited := execution.LimitationCode()
			reason, phase, failureClass, matched := dynamicFailureParts(limitation)
			if !limited || !matched {
				continue
			}
			diagnostic, err := newGraphDynamicInstallDiagnostic(dependency, reason, phase, failureClass)
			if err != nil {
				return nil, err
			}
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	return diagnostics, nil
}

func dynamicFailureParts(limitation string) (reason, phase, failureClass string, matched bool) {
	if failureClass, matched = strings.CutPrefix(limitation, dynamicInstallFailurePrefix); matched {
		return dynamicReasonInstall, "", failureClass, true
	}
	if failureClass, matched = strings.CutPrefix(limitation, dynamicImportFailurePrefix); matched {
		return dynamicReasonImport, "", failureClass, true
	}
	remaining, matched := strings.CutPrefix(limitation, dynamicObserverFailurePrefix)
	if !matched {
		return "", "", "", false
	}
	for _, candidate := range []string{"START", "FINALIZE", "COLLECT", "NEXT_SESSION"} {
		if failureClass, matched = strings.CutPrefix(remaining, candidate+"_"); matched {
			return dynamicReasonObserver, candidate, failureClass, true
		}
	}
	return "", "", "", false
}

func validDynamicFailure(reason, phase, failureClass string) bool {
	switch reason {
	case dynamicReasonInstall:
		return phase == "" && validDynamicInstallFailureClass(failureClass)
	case dynamicReasonImport:
		return phase == "" && validDynamicImportFailureClass(failureClass)
	case dynamicReasonObserver:
		return validDynamicObserverPhase(phase) && validDynamicObserverFailureClass(failureClass)
	default:
		return false
	}
}

func validDynamicInstallFailureClass(value string) bool {
	switch value {
	case "PIP_ARGUMENT_ERROR", "WHEEL_PLATFORM_REJECTED", "WHEEL_METADATA_REJECTED", "PACKAGE_CONFLICT", "DUPLICATE_DISTRIBUTION", "ENOSPC", "MEMORY_LIMIT", "TIMEOUT", "PERMISSION", "SANDBOX_RUNTIME", "OTHER":
		return true
	default:
		return false
	}
}

func validDynamicImportFailureClass(value string) bool {
	switch value {
	case "MISSING_SHARED_LIBRARY", "DLOPEN_PERMISSION", "ELF_LOADER", "SYMBOL_VERSION", "PYTHON_ABI", "IMPORT_EXCEPTION", "MEMORY_LIMIT", "CPU_LIMIT", "PID_LIMIT", "TIMEOUT", "SANDBOX_RUNTIME", "OTHER":
		return true
	default:
		return false
	}
}

func validDynamicObserverPhase(value string) bool {
	return value == "START" || value == "FINALIZE" || value == "COLLECT" || value == "NEXT_SESSION"
}

func validDynamicObserverFailureClass(value string) bool {
	switch value {
	case "PREVIOUS_TRACE_NOT_FINALIZED", "HELPER_UNAVAILABLE", "HELPER_CRASHED", "EVENT_LIMIT", "BYTE_LIMIT", "SESSION_LIMIT", "TRACE_COLLECTION_FAILED", "CLEANUP_ORDERING", "LIFECYCLE_ERROR", "OTHER":
		return true
	default:
		return false
	}
}

func (s *InstallInspectService) inspectAcquiredDependency(ctx context.Context, operationID domain.OperationID, dependency domain.LockedDependency, artifact domain.AcquiredArtifact, additionalChecks []domain.CheckExecution, additionalEvidence []domain.Evidence) (domain.DependencyInspection, error) {
	runID, err := s.newRunID()
	if err != nil {
		return domain.DependencyInspection{}, err
	}
	reference, err := lockedReference(dependency.Artifact())
	if err != nil {
		return domain.DependencyInspection{}, err
	}
	run, err := domain.NewInspectionRun(runID, operationID, reference, dependency.Artifact().Identity())
	if err != nil {
		return domain.DependencyInspection{}, err
	}
	if err := run.Activate(); err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "RUN_ACTIVATION_FAILED", err)
	}
	if err := run.BindAcquiredArtifact(artifact); err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "ARTIFACT_BINDING_FAILED", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "verify"})
	verification, err := s.verification.Verify(ctx, artifact)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "VERIFICATION_PROVIDER_FAILED", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "inspect"})
	inspection, err := s.inspection.Inspect(ctx, artifact)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "INSPECTION_PROVIDER_FAILED", err)
	}
	checks := append([]domain.CheckExecution{verification.Execution()}, inspection.Executions()...)
	checks = append(checks, additionalChecks...)
	evidence := append(verification.Evidence(), inspection.Evidence()...)
	evidence = append(evidence, additionalEvidence...)
	references, err := s.evidence.Record(ctx, runID, evidence)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "EVIDENCE_RECORD_FAILED", err)
	}
	input, err := domain.NewPolicyInput(runID, artifact, verification, inspection, references)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "POLICY_INPUT_INVALID", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "policy"})
	decision, err := s.entryPolicy.Evaluate(input)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "POLICY_EVALUATION_FAILED", err)
	}
	record, err := newRecordedDependencyInspection(dependency.Node(), runID, artifact, checks, references, decision, evidence)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "DEPENDENCY_INSPECTION_INVALID", err)
	}
	if err := run.FinalizeCompleted(decision); err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "RUN_FINALIZATION_FAILED", err)
	}
	return record, nil
}

func (s *InstallInspectService) inspectDependency(ctx context.Context, operationID domain.OperationID, dependency domain.LockedDependency) (domain.DependencyInspection, DependencyPolicyDiagnostic, error) {
	runID, err := s.newRunID()
	if err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, fmt.Errorf("generate Run ID for dependency %s: %w", dependency.Node().String(), err)
	}
	reference, err := lockedReference(dependency.Artifact())
	if err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, fmt.Errorf("construct dependency reference: %w", err)
	}
	run, err := domain.NewInspectionRun(runID, operationID, reference, dependency.Artifact().Identity())
	if err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, fmt.Errorf("create dependency Inspection Run: %w", err)
	}
	if err := run.Activate(); err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, failDependencyRun(run, "RUN_ACTIVATION_FAILED", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "download"})
	artifact, err := s.artifact.Acquire(ctx, runID, dependency.Artifact())
	if err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, failDependencyRun(run, "ARTIFACT_ACQUIRE_FAILED", err)
	}
	if err := run.BindAcquiredArtifact(artifact); err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, failDependencyRun(run, "ARTIFACT_BINDING_FAILED", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "verify"})
	verification, err := s.verification.Verify(ctx, artifact)
	if err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, failDependencyRun(run, "VERIFICATION_PROVIDER_FAILED", err)
	}
	checks := []domain.CheckExecution{verification.Execution()}
	ports.ReportProgress(ctx, ports.Progress{Stage: "inspect"})
	inspection, err := s.inspection.Inspect(ctx, artifact)
	if err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, failDependencyRun(run, "INSPECTION_PROVIDER_FAILED", err)
	}
	checks = append(checks, inspection.Executions()...)
	evidence := append(verification.Evidence(), inspection.Evidence()...)
	references, err := s.evidence.Record(ctx, runID, evidence)
	if err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, failDependencyRun(run, "EVIDENCE_RECORD_FAILED", err)
	}
	input, err := domain.NewPolicyInput(runID, artifact, verification, inspection, references)
	if err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, failDependencyRun(run, "POLICY_INPUT_INVALID", err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "policy"})
	decision, err := s.entryPolicy.Evaluate(input)
	if err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, failDependencyRun(run, "POLICY_EVALUATION_FAILED", err)
	}
	record, err := newRecordedDependencyInspection(dependency.Node(), runID, artifact, checks, references, decision, evidence)
	if err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, failDependencyRun(run, "DEPENDENCY_INSPECTION_INVALID", err)
	}
	if err := run.FinalizeCompleted(decision); err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, failDependencyRun(run, "RUN_FINALIZATION_FAILED", err)
	}
	diagnostic, err := newDependencyPolicyDiagnostic(dependency, decision, inspection.Findings())
	if err != nil {
		return domain.DependencyInspection{}, DependencyPolicyDiagnostic{}, err
	}
	return record, diagnostic, nil
}

func newRecordedDependencyInspection(node domain.DependencyNodeID, run domain.RunID, artifact domain.AcquiredArtifact, checks []domain.CheckExecution, references []domain.EvidenceReference, decision domain.PolicyDecision, evidence []domain.Evidence) (domain.DependencyInspection, error) {
	record, err := domain.NewDependencyInspection(node, run, artifact, checks, references, decision)
	if err != nil {
		return domain.DependencyInspection{}, err
	}
	return record.WithHostInstallActionEvidence(evidence)
}

func lockedReference(artifact domain.ResolvedArtifact) (domain.ArtifactReference, error) {
	identity := artifact.Identity()
	return domain.NewArtifactReference(identity.Source(), identity.Name()+"@"+identity.Version())
}

func failDependencyRun(run *domain.InspectionRun, code string, cause error) error {
	if err := run.FinalizeFailed(code); err != nil {
		return errors.Join(cause, fmt.Errorf("finalize failed dependency Inspection Run: %w", err))
	}
	return cause
}
