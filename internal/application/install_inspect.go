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
	references   []domain.EvidenceReference
	decision     domain.PolicyDecision
	checks       []domain.CheckExecution
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
	for _, dependency := range graph.Nodes() {
		inspection, inspectionErr := s.inspectDependency(ctx, operationID, dependency)
		if inspectionErr != nil {
			return partial, inspectionErr
		}
		inspections = append(inspections, inspection)
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
	decision, err := s.setPolicy.EvaluateSet(set)
	if err != nil {
		return partial, fmt.Errorf("evaluate dependency set policy: %w", err)
	}
	return newInspectedInstall(operationID, request, resolution, set, decision), nil
}

func (s *InstallInspectService) inspectGraphAware(ctx context.Context, operationID domain.OperationID, request InstallRequest, partial InspectedInstall, graph domain.LockedDependencyGraph, inspector dependencyAwareInspection) (InspectedInstall, error) {
	pending := make([]*pendingDependencyInspection, 0, len(graph.Nodes()))
	byNode := make(map[domain.DependencyNodeID]*pendingDependencyInspection, len(graph.Nodes()))
	failPending := func(code string, cause error) (InspectedInstall, error) {
		for _, item := range pending {
			if item.finalized || item.run == nil {
				continue
			}
			_ = item.run.FinalizeFailed(code)
		}
		return partial, cause
	}
	for _, dependency := range graph.Nodes() {
		runID, err := s.newRunID()
		if err != nil {
			return failPending("RUN_ID_GENERATION_FAILED", fmt.Errorf("generate Run ID for dependency %s: %w", dependency.Node().String(), err))
		}
		reference, err := lockedReference(dependency.Artifact())
		if err != nil {
			return failPending("REFERENCE_INVALID", err)
		}
		run, err := domain.NewInspectionRun(runID, operationID, reference, dependency.Artifact().Identity())
		if err != nil {
			return failPending("RUN_CREATION_FAILED", err)
		}
		if err := run.Activate(); err != nil {
			return failPending("RUN_ACTIVATION_FAILED", err)
		}
		item := &pendingDependencyInspection{dependency: dependency, runID: runID, run: run}
		pending = append(pending, item)
		byNode[dependency.Node()] = item
		artifact, err := s.artifact.Acquire(ctx, runID, dependency.Artifact())
		if err != nil {
			return failPending("ARTIFACT_ACQUIRE_FAILED", err)
		}
		if err := run.BindAcquiredArtifact(artifact); err != nil {
			return failPending("ARTIFACT_BINDING_FAILED", err)
		}
		if !acquiredMatchesLocked(artifact, dependency, runID) {
			return failPending("ARTIFACT_BINDING_INVALID", fmt.Errorf("acquired artifact binding is invalid for dependency %s", dependency.Node().String()))
		}
		verification, err := s.verification.Verify(ctx, artifact)
		if err != nil {
			return failPending("VERIFICATION_PROVIDER_FAILED", err)
		}
		item.artifact, item.verification = artifact, verification
	}
	artifacts := make(map[domain.DependencyNodeID]domain.AcquiredArtifact, len(pending))
	for _, dependency := range graph.Nodes() {
		item := byNode[dependency.Node()]
		if item == nil {
			return failPending("DEPENDENCY_ARTIFACTS_INVALID", fmt.Errorf("acquired dependency is absent for node %s", dependency.Node().String()))
		}
		artifacts[dependency.Node()] = item.artifact
	}
	reports, err := inspector.InspectGraph(ctx, graph, artifacts)
	if err != nil {
		return failPending("INSPECTION_PROVIDER_FAILED", err)
	}
	if err := validateGraphReports(graph, reports); err != nil {
		return failPending("INSPECTION_REPORT_INVALID", err)
	}
	for _, item := range pending {
		report := reports[item.dependency.Node()]
		checks := append([]domain.CheckExecution{item.verification.Execution()}, report.Executions()...)
		evidence := append(item.verification.Evidence(), report.Evidence()...)
		references, err := s.evidence.Record(ctx, item.runID, evidence)
		if err != nil {
			return failPending("EVIDENCE_RECORD_FAILED", err)
		}
		input, err := domain.NewPolicyInput(item.runID, item.artifact, item.verification, report, references)
		if err != nil {
			return failPending("POLICY_INPUT_INVALID", err)
		}
		decision, err := s.entryPolicy.Evaluate(input)
		if err != nil {
			return failPending("POLICY_EVALUATION_FAILED", err)
		}
		if err := item.run.FinalizeCompleted(decision); err != nil {
			return failPending("RUN_FINALIZATION_FAILED", err)
		}
		item.finalized = true
		item.references = references
		item.decision = decision
		item.checks = checks
	}
	inspections := make([]domain.DependencyInspection, 0, len(pending))
	for _, item := range pending {
		record, err := domain.NewDependencyInspection(item.dependency.Node(), item.runID, item.artifact, item.checks, item.references, item.decision)
		if err != nil {
			return partial, err
		}
		inspections = append(inspections, record)
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
	return newInspectedInstall(operationID, request, partial.Resolution(), set, decision).withGraphStaticDiagnostics(diagnostics).withGraphDynamicInstallDiagnostics(dynamicDiagnostics), nil
}

func acquiredMatchesLocked(acquired domain.AcquiredArtifact, dependency domain.LockedDependency, runID domain.RunID) bool {
	locked, actual := dependency.Artifact().Identity(), acquired.Identity()
	if locked.Source() != actual.Source() || locked.Name() != actual.Name() || locked.Version() != actual.Version() || locked.Variant() != actual.Variant() || acquired.Digest().String() == "" {
		return false
	}
	declared, ok := acquired.DeclaredIntegrity()
	return ok && declared == dependency.Artifact().DeclaredIntegrity() && acquired.ContentHandle() == "intake:"+runID.String()+":"+actual.Variant()
}

func validateGraphReports(graph domain.LockedDependencyGraph, reports map[domain.DependencyNodeID]domain.InspectionReport) error {
	nodes := graph.Nodes()
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

const dynamicInstallFailurePrefix = "M5_PYPI_DYNAMIC_INSTALL_FAILED_"

func graphDynamicInstallDiagnostics(graph domain.LockedDependencyGraph, reports map[domain.DependencyNodeID]domain.InspectionReport) ([]GraphDynamicInstallDiagnostic, error) {
	diagnostics := make([]GraphDynamicInstallDiagnostic, 0)
	for _, dependency := range graph.Nodes() {
		for _, execution := range reports[dependency.Node()].Executions() {
			if execution.ID().String() != "pypi-dynamic-import" {
				continue
			}
			limitation, limited := execution.LimitationCode()
			failureClass, matched := strings.CutPrefix(limitation, dynamicInstallFailurePrefix)
			if !limited || !matched {
				continue
			}
			diagnostic, err := newGraphDynamicInstallDiagnostic(dependency, failureClass)
			if err != nil {
				return nil, err
			}
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	return diagnostics, nil
}

func validDynamicInstallFailureClass(value string) bool {
	switch value {
	case "PIP_ARGUMENT_ERROR", "WHEEL_PLATFORM_REJECTED", "WHEEL_METADATA_REJECTED", "PACKAGE_CONFLICT", "DUPLICATE_DISTRIBUTION", "ENOSPC", "MEMORY_LIMIT", "TIMEOUT", "PERMISSION", "SANDBOX_RUNTIME", "OTHER":
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
	verification, err := s.verification.Verify(ctx, artifact)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "VERIFICATION_PROVIDER_FAILED", err)
	}
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
	decision, err := s.entryPolicy.Evaluate(input)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "POLICY_EVALUATION_FAILED", err)
	}
	if err := run.FinalizeCompleted(decision); err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "RUN_FINALIZATION_FAILED", err)
	}
	return domain.NewDependencyInspection(dependency.Node(), runID, artifact, checks, references, decision)
}

func (s *InstallInspectService) inspectDependency(ctx context.Context, operationID domain.OperationID, dependency domain.LockedDependency) (domain.DependencyInspection, error) {
	runID, err := s.newRunID()
	if err != nil {
		return domain.DependencyInspection{}, fmt.Errorf("generate Run ID for dependency %s: %w", dependency.Node().String(), err)
	}
	reference, err := lockedReference(dependency.Artifact())
	if err != nil {
		return domain.DependencyInspection{}, fmt.Errorf("construct dependency reference: %w", err)
	}
	run, err := domain.NewInspectionRun(runID, operationID, reference, dependency.Artifact().Identity())
	if err != nil {
		return domain.DependencyInspection{}, fmt.Errorf("create dependency Inspection Run: %w", err)
	}
	if err := run.Activate(); err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "RUN_ACTIVATION_FAILED", err)
	}
	artifact, err := s.artifact.Acquire(ctx, runID, dependency.Artifact())
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "ARTIFACT_ACQUIRE_FAILED", err)
	}
	if err := run.BindAcquiredArtifact(artifact); err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "ARTIFACT_BINDING_FAILED", err)
	}
	verification, err := s.verification.Verify(ctx, artifact)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "VERIFICATION_PROVIDER_FAILED", err)
	}
	checks := []domain.CheckExecution{verification.Execution()}
	inspection, err := s.inspection.Inspect(ctx, artifact)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "INSPECTION_PROVIDER_FAILED", err)
	}
	checks = append(checks, inspection.Executions()...)
	evidence := append(verification.Evidence(), inspection.Evidence()...)
	references, err := s.evidence.Record(ctx, runID, evidence)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "EVIDENCE_RECORD_FAILED", err)
	}
	input, err := domain.NewPolicyInput(runID, artifact, verification, inspection, references)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "POLICY_INPUT_INVALID", err)
	}
	decision, err := s.entryPolicy.Evaluate(input)
	if err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "POLICY_EVALUATION_FAILED", err)
	}
	if err := run.FinalizeCompleted(decision); err != nil {
		return domain.DependencyInspection{}, failDependencyRun(run, "RUN_FINALIZATION_FAILED", err)
	}
	record, err := domain.NewDependencyInspection(dependency.Node(), runID, artifact, checks, references, decision)
	if err != nil {
		return domain.DependencyInspection{}, fmt.Errorf("construct dependency inspection record: %w", err)
	}
	return record, nil
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
