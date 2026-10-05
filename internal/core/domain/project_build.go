// Package domain defines the current guarded project-build bindings.
package domain

import (
	"errors"
	"fmt"
	"strings"
)

// ProjectBuildInputs freezes opaque transport subjects. Construction checks
// structure only; the project guard must independently retain approval.
type ProjectBuildInputs struct {
	snapshot      ProjectDependencySnapshot
	source, cache AcquiredArtifact
	run           RunID
	selector      string
}

func NewProjectBuildInputs(snapshot ProjectDependencySnapshot, source, cache AcquiredArtifact, run RunID, selector string) (ProjectBuildInputs, error) {
	if !snapshot.Valid() || snapshot.Source().String() != "go-proxy" || run.String() == "" ||
		source.Identity().Source().String() != "project-local" || source.Identity().Variant() != "go-source" ||
		cache.Identity().Source().String() != "project-local" || cache.Identity().Variant() != "go-cache" || cache.Identity().Version() != snapshot.GraphDigest().String() {
		return ProjectBuildInputs{}, errors.New("project build inputs are invalid")
	}
	for _, artifact := range []AcquiredArtifact{source, cache} {
		declared, ok := artifact.DeclaredIntegrity()
		parts := strings.Split(artifact.ContentHandle(), ":")
		if artifact.Digest().String() == "" || artifact.SizeBytes() == 0 || !ok || declared != "sha256:"+artifact.Digest().String() || len(parts) != 3 || parts[0] != "intake" || parts[2] != artifact.Identity().Variant() {
			return ProjectBuildInputs{}, errors.New("project build transport binding is invalid")
		}
		if _, err := ParseRunID(parts[1]); err != nil {
			return ProjectBuildInputs{}, errors.New("project build transport handle is invalid")
		}
	}
	// Ecosystem grammar is additionally checked before an actual command.
	if validateBoundedText(selector, 256, "package selector") != nil || strings.HasPrefix(selector, "-") || strings.HasPrefix(selector, "/") || strings.ContainsAny(selector, "\\:@") {
		return ProjectBuildInputs{}, errors.New("project build package selector is invalid")
	}
	for _, part := range strings.Split(selector, "/") {
		if part == "" || part == ".." {
			return ProjectBuildInputs{}, errors.New("project build package selector is invalid")
		}
	}
	return ProjectBuildInputs{snapshot, source, cache, run, selector}, nil
}

func (i ProjectBuildInputs) Valid() bool {
	return i.snapshot.Valid() && i.run.String() != "" && i.source.Digest().String() != "" && i.cache.Digest().String() != "" && i.selector != ""
}
func (i ProjectBuildInputs) Snapshot() ProjectDependencySnapshot { return i.snapshot }
func (i ProjectBuildInputs) Source() AcquiredArtifact            { return i.source }
func (i ProjectBuildInputs) Cache() AcquiredArtifact             { return i.cache }
func (i ProjectBuildInputs) RunID() RunID                        { return i.run }
func (i ProjectBuildInputs) Selector() string                    { return i.selector }

// ProjectBuildObservation is an operational result, never publication approval.
// The backend retains output only after complete observation and cleanup.
type ProjectBuildObservation struct {
	inputs  ProjectBuildInputs
	result  SandboxResult
	output  AcquiredArtifact
	binding DerivationBinding
}

func NewProjectBuildObservation(inputs ProjectBuildInputs, result SandboxResult, output AcquiredArtifact, binding DerivationBinding) (ProjectBuildObservation, error) {
	if !inputs.Valid() || result.SessionID().String() == "" {
		return ProjectBuildObservation{}, errors.New("project build observation is invalid")
	}
	if result.Status() != SandboxCompleted {
		if output.Digest().String() != "" || binding.SourceDigest().String() != "" {
			return ProjectBuildObservation{}, errors.New("incomplete project build cannot retain output")
		}
	} else if !validProjectBuildOutput(inputs, output, binding) {
		return ProjectBuildObservation{}, errors.New("completed project build output binding is invalid")
	}
	return ProjectBuildObservation{inputs, result, output, binding}, nil
}
func (o ProjectBuildObservation) Inputs() ProjectBuildInputs   { return o.inputs }
func (o ProjectBuildObservation) SandboxResult() SandboxResult { return o.result }
func (o ProjectBuildObservation) Output() AcquiredArtifact     { return o.output }
func (o ProjectBuildObservation) Binding() DerivationBinding   { return o.binding }

func validProjectBuildOutput(inputs ProjectBuildInputs, output AcquiredArtifact, binding DerivationBinding) bool {
	declared, ok := output.DeclaredIntegrity()
	if output.Identity().Source() != inputs.Source().Identity().Source() || output.Identity().Name() != inputs.Source().Identity().Name() ||
		output.Identity().Variant() != "go-output" || output.Identity().Version() != inputs.RunID().String() || output.Digest().String() == "" || output.SizeBytes() == 0 || !ok ||
		declared != "sha256:"+output.Digest().String() || output.ContentHandle() != "intake:"+inputs.RunID().String()+":go-output" ||
		binding.SourceDigest() != inputs.Source().Digest() || binding.Executor() != "go-build-linux-amd64" || binding.ConfigDigest().String() == "" {
		return false
	}
	seen := map[ContentDigest]bool{}
	for _, digest := range binding.InputDigests() {
		seen[digest] = true
	}
	return len(seen) == 2 && seen[inputs.Cache().Digest()] && seen[inputs.Snapshot().GraphDigest()]
}

// ProjectBuildReport carries only normalized inspection/verification facts.
// Local source integrity is distinct from public dependency SumDB provenance.
type ProjectBuildReport struct {
	inputs     ProjectBuildInputs
	output     AcquiredArtifact
	binding    DerivationBinding
	inspection InspectionReport
}

func NewProjectBuildReport(observation ProjectBuildObservation, inspection InspectionReport) (ProjectBuildReport, error) {
	if !observation.inputs.Valid() || observation.result.Status() != SandboxCompleted || !validProjectBuildOutput(observation.inputs, observation.output, observation.binding) ||
		len(inspection.Executions()) != 1 || inspection.Execution().ID().String() != "project-build-observation" || !inspection.Execution().Required() {
		return ProjectBuildReport{}, errors.New("project build report is invalid")
	}
	for _, evidence := range inspection.Evidence() {
		if evidence.Identity() != observation.inputs.Source().Identity() || evidence.Digest() != observation.inputs.Source().Digest() {
			return ProjectBuildReport{}, errors.New("project build evidence subject differs")
		}
	}
	return ProjectBuildReport{observation.inputs, observation.output, observation.binding, inspection}, nil
}
func (r ProjectBuildReport) Inputs() ProjectBuildInputs   { return r.inputs }
func (r ProjectBuildReport) Output() AcquiredArtifact     { return r.output }
func (r ProjectBuildReport) Binding() DerivationBinding   { return r.binding }
func (r ProjectBuildReport) Inspection() InspectionReport { return r.inspection }

// BuildBindingSummary contains digests only, never selected paths/raw output.
func BuildBindingSummary(inputs ProjectBuildInputs, output AcquiredArtifact, binding DerivationBinding) string {
	return fmt.Sprintf("graph_sha256=%s cache_sha256=%s output_sha256=%s recipe_sha256=%s", inputs.Snapshot().GraphDigest(), inputs.Cache().Digest(), output.Digest(), binding.ConfigDigest())
}

// ApprovedProjectBuild preserves one actual completed Run and its recorded
// Evidence/ALLOW. Structural construction does not replace evidence re-reading.
type ApprovedProjectBuild struct {
	report       ProjectBuildReport
	verification VerificationReport
	result       OperationResult
}

func NewApprovedProjectBuild(report ProjectBuildReport, verification VerificationReport, result OperationResult) (ApprovedProjectBuild, error) {
	run, hasRun := result.RunID()
	outcome, hasOutcome := result.RunOutcome()
	identity, hasIdentity := result.ResolvedIdentity()
	digest, hasDigest := result.Digest()
	decision, hasDecision := result.PolicyDecision()
	if !report.inputs.Valid() || !validProjectBuildOutput(report.inputs, report.output, report.binding) || result.Status() != OperationCompleted || !hasRun || run != report.inputs.RunID() ||
		!hasOutcome || outcome != RunCompleted || !hasIdentity || identity != report.inputs.Source().Identity() || !hasDigest || digest != report.inputs.Source().Digest() || !hasDecision || decision.Decision() != DecisionAllow ||
		verification.Execution().ID().String() != "project-build-input-binding" || verification.Outcome() != VerificationVerified || len(verification.Findings()) != 0 || len(report.inspection.Findings()) != 0 {
		return ApprovedProjectBuild{}, errors.New("project build requires completed exact Run and ALLOW")
	}
	if _, err := NewPolicyInput(report.inputs.RunID(), report.inputs.Source(), verification, report.inspection, result.Evidence()); err != nil {
		return ApprovedProjectBuild{}, err
	}
	checks := append([]CheckExecution{verification.Execution()}, report.inspection.Executions()...)
	actual := result.Checks()
	if len(checks) != len(actual) {
		return ApprovedProjectBuild{}, errors.New("project build check coverage differs")
	}
	for i, check := range checks {
		if check != actual[i] || !check.Required() || check.Capability() != CapabilitySupported || check.Status() != ExecutionCompleted {
			return ApprovedProjectBuild{}, errors.New("project build has incomplete or substituted checks")
		}
	}
	return ApprovedProjectBuild{report, verification, result}, nil
}
func (b ApprovedProjectBuild) Valid() bool {
	decision, ok := b.result.PolicyDecision()
	return b.report.inputs.Valid() && b.result.Status() == OperationCompleted && ok && decision.Decision() == DecisionAllow
}
func (b ApprovedProjectBuild) Report() ProjectBuildReport       { return b.report }
func (b ApprovedProjectBuild) Verification() VerificationReport { return b.verification }
func (b ApprovedProjectBuild) Result() OperationResult          { return b.result }

// PublishedProjectBuild identifies one new output directory under the guarded
// project. Output names do not confer future inspection/approval authority.
type PublishedProjectBuild struct{ build ApprovedProjectBuild }

func NewPublishedProjectBuild(build ApprovedProjectBuild) (PublishedProjectBuild, error) {
	if !build.Valid() {
		return PublishedProjectBuild{}, errors.New("published project build requires approval")
	}
	return PublishedProjectBuild{build}, nil
}
func (b PublishedProjectBuild) Valid() bool                 { return b.build.Valid() }
func (b PublishedProjectBuild) Build() ApprovedProjectBuild { return b.build }
func (b PublishedProjectBuild) RelativeDirectory() string {
	return ".heliopause/builds/" + b.build.Report().Inputs().RunID().String()
}
