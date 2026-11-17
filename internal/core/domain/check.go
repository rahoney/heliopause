package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// CheckKind separates Verification from Inspection meaning.
type CheckKind string

const (
	CheckVerification CheckKind = "VERIFICATION"
	CheckInspection   CheckKind = "INSPECTION"
)

// Capability describes whether a check is supported, not whether it succeeded.
type Capability string

const (
	CapabilitySupported   Capability = "SUPPORTED"
	CapabilityUnsupported Capability = "UNSUPPORTED"
)

// ExecutionStatus describes how a check execution ended, not its security meaning.
type ExecutionStatus string

const (
	ExecutionCompleted   ExecutionStatus = "COMPLETED"
	ExecutionFailed      ExecutionStatus = "FAILED"
	ExecutionIncomplete  ExecutionStatus = "INCOMPLETE"
	ExecutionNotExecuted ExecutionStatus = "NOT_EXECUTED"
	ExecutionUnavailable ExecutionStatus = "UNAVAILABLE"
)

// InspectionDiagnosticCause identifies a bounded reason why a graph static
// inspection cannot enter its dynamic phase. It is diagnostic metadata only;
// Policy decisions continue to use Findings and CheckExecution values.
type InspectionDiagnosticCause string

const (
	InspectionDiagnosticStaticFinding           InspectionDiagnosticCause = "STATIC_FINDING"
	InspectionDiagnosticStaticIncomplete        InspectionDiagnosticCause = "STATIC_INCOMPLETE"
	InspectionDiagnosticSdistDynamicUnavailable InspectionDiagnosticCause = "SDIST_DYNAMIC_UNAVAILABLE"
	InspectionDiagnosticResourceExceeded        InspectionDiagnosticCause = "RESOURCE_EXCEEDED"
	InspectionDiagnosticInvalidWheel            InspectionDiagnosticCause = "INVALID_WHEEL"
)

// InspectionDiagnosticStage is a bounded parser or resource accounting
// stage. Empty is valid only for causes that have no stage-specific detail.
type InspectionDiagnosticStage string

const (
	InspectionDiagnosticStageFilename                InspectionDiagnosticStage = "FILENAME"
	InspectionDiagnosticStageCompatibility           InspectionDiagnosticStage = "COMPATIBILITY"
	InspectionDiagnosticStageZIP                     InspectionDiagnosticStage = "ZIP"
	InspectionDiagnosticStageDigest                  InspectionDiagnosticStage = "DIGEST"
	InspectionDiagnosticStageMetadataIdentity        InspectionDiagnosticStage = "METADATA_IDENTITY"
	InspectionDiagnosticStageDistInfoIdentity        InspectionDiagnosticStage = "DIST_INFO_IDENTITY"
	InspectionDiagnosticStageWheelTag                InspectionDiagnosticStage = "WHEEL_TAG"
	InspectionDiagnosticStageRecord                  InspectionDiagnosticStage = "RECORD"
	InspectionDiagnosticStageFileType                InspectionDiagnosticStage = "FILE_TYPE"
	InspectionDiagnosticStageOther                   InspectionDiagnosticStage = "OTHER"
	InspectionDiagnosticStagePerArtifactCompressed   InspectionDiagnosticStage = "PER_ARTIFACT_COMPRESSED"
	InspectionDiagnosticStagePerArtifactUncompressed InspectionDiagnosticStage = "PER_ARTIFACT_UNCOMPRESSED"
	InspectionDiagnosticStageFileCount               InspectionDiagnosticStage = "FILE_COUNT"
	InspectionDiagnosticStageMetadata                InspectionDiagnosticStage = "METADATA"
	InspectionDiagnosticStageGraphUncompressed       InspectionDiagnosticStage = "GRAPH_UNCOMPRESSED"
)

// InspectionDiagnostic is bounded metadata for qualification diagnostics.
// It intentionally carries no parser error, path, environment, URL, or
// archive content.
type InspectionDiagnostic struct {
	cause InspectionDiagnosticCause
	stage InspectionDiagnosticStage
}

// NewInspectionDiagnostic validates the small diagnostic vocabulary.
func NewInspectionDiagnostic(cause InspectionDiagnosticCause, stage InspectionDiagnosticStage) (InspectionDiagnostic, error) {
	switch cause {
	case InspectionDiagnosticStaticFinding, InspectionDiagnosticStaticIncomplete, InspectionDiagnosticSdistDynamicUnavailable, InspectionDiagnosticResourceExceeded, InspectionDiagnosticInvalidWheel:
	default:
		return InspectionDiagnostic{}, errors.New("invalid inspection diagnostic cause")
	}
	if cause == InspectionDiagnosticInvalidWheel {
		switch stage {
		case InspectionDiagnosticStageFilename, InspectionDiagnosticStageCompatibility, InspectionDiagnosticStageZIP, InspectionDiagnosticStageDigest, InspectionDiagnosticStageMetadataIdentity, InspectionDiagnosticStageDistInfoIdentity, InspectionDiagnosticStageWheelTag, InspectionDiagnosticStageRecord, InspectionDiagnosticStageFileType, InspectionDiagnosticStageOther:
		default:
			return InspectionDiagnostic{}, errors.New("invalid wheel diagnostic stage")
		}
	} else if cause == InspectionDiagnosticResourceExceeded {
		switch stage {
		case InspectionDiagnosticStagePerArtifactCompressed, InspectionDiagnosticStagePerArtifactUncompressed, InspectionDiagnosticStageFileCount, InspectionDiagnosticStageMetadata, InspectionDiagnosticStageGraphUncompressed, InspectionDiagnosticStageOther:
		default:
			return InspectionDiagnostic{}, errors.New("invalid resource diagnostic stage")
		}
	} else if stage != "" {
		return InspectionDiagnostic{}, errors.New("diagnostic stage is not applicable")
	}
	return InspectionDiagnostic{cause: cause, stage: stage}, nil
}

func (d InspectionDiagnostic) Cause() InspectionDiagnosticCause { return d.cause }
func (d InspectionDiagnostic) Stage() InspectionDiagnosticStage { return d.stage }

// CheckID identifies one normalized check within a Run. Its zero value is invalid.
type CheckID struct{ value string }

// NewCheckID validates a normalized check identifier.
func NewCheckID(value string) (CheckID, error) {
	if err := validateNormalizedIdentifier(value, 64, "check ID"); err != nil {
		return CheckID{}, err
	}
	return CheckID{value: value}, nil
}

func (id CheckID) String() string { return id.value }

// CheckExecution records independent requirement, capability, and execution axes.
type CheckExecution struct {
	id         CheckID
	kind       CheckKind
	required   bool
	capability Capability
	status     ExecutionStatus
	limitation string
	profile    ContentDigest
}

// NewCheckExecution constructs a semantically valid check execution envelope.
func NewCheckExecution(id CheckID, kind CheckKind, required bool, capability Capability, status ExecutionStatus, limitationCode string, profiles ...ContentDigest) (CheckExecution, error) {
	if id.value == "" {
		return CheckExecution{}, errors.New("check ID is required")
	}
	if len(profiles) > 1 || len(profiles) == 1 && profiles[0].value == "" {
		return CheckExecution{}, errors.New("check execution profile is invalid")
	}
	var profile ContentDigest
	if len(profiles) == 1 {
		profile = profiles[0]
	}

	if kind != CheckVerification && kind != CheckInspection {
		return CheckExecution{}, fmt.Errorf("invalid check kind %q", kind)
	}
	if capability != CapabilitySupported && capability != CapabilityUnsupported {
		return CheckExecution{}, fmt.Errorf("invalid capability %q", capability)
	}
	switch status {
	case ExecutionCompleted, ExecutionFailed, ExecutionIncomplete, ExecutionNotExecuted, ExecutionUnavailable:
	default:
		return CheckExecution{}, fmt.Errorf("invalid execution status %q", status)
	}
	if capability == CapabilityUnsupported && status != ExecutionNotExecuted {
		return CheckExecution{}, errors.New("unsupported check must be NOT_EXECUTED")
	}
	if status == ExecutionCompleted && limitationCode != "" {
		return CheckExecution{}, errors.New("completed check cannot have a limitation")
	}
	if status != ExecutionCompleted {
		if !failureCodePattern.MatchString(limitationCode) {
			return CheckExecution{}, errors.New("non-completed check requires an uppercase limitation code")
		}
	}
	return CheckExecution{id: id, kind: kind, required: required, capability: capability, status: status, limitation: limitationCode, profile: profile}, nil
}

func (c CheckExecution) ProfileDigest() ContentDigest { return c.profile }

func (c CheckExecution) ID() CheckID                    { return c.id }
func (c CheckExecution) Kind() CheckKind                { return c.kind }
func (c CheckExecution) Required() bool                 { return c.required }
func (c CheckExecution) Capability() Capability         { return c.capability }
func (c CheckExecution) Status() ExecutionStatus        { return c.status }
func (c CheckExecution) LimitationCode() (string, bool) { return c.limitation, c.limitation != "" }

// VerificationOutcome is the normalized result of a completed verification.
type VerificationOutcome string

const (
	VerificationVerified VerificationOutcome = "VERIFIED"
	VerificationMismatch VerificationOutcome = "MISMATCH"
)

// VerificationReport keeps execution status separate from a completed verification result.
type VerificationReport struct {
	execution CheckExecution
	outcome   VerificationOutcome
	findings  []Finding
	evidence  []Evidence
}

// NewVerificationReport constructs a normalized Verification report.
func NewVerificationReport(execution CheckExecution, outcome VerificationOutcome, evidence []Evidence) (VerificationReport, error) {
	return NewVerificationReportWithFindings(execution, outcome, nil, evidence)
}

// NewVerificationReportWithFindings constructs a normalized Verification report with supporting findings.
func NewVerificationReportWithFindings(execution CheckExecution, outcome VerificationOutcome, findings []Finding, evidence []Evidence) (VerificationReport, error) {
	if execution.kind != CheckVerification {
		return VerificationReport{}, errors.New("verification report requires a Verification check")
	}
	if execution.status == ExecutionCompleted {
		if outcome != VerificationVerified && outcome != VerificationMismatch {
			return VerificationReport{}, errors.New("completed verification requires an interpretable outcome")
		}
		if len(evidence) == 0 {
			return VerificationReport{}, errors.New("completed verification requires Evidence")
		}
		if outcome == VerificationVerified && len(findings) != 0 {
			return VerificationReport{}, errors.New("verified verification cannot claim Findings")
		}
	} else if outcome != "" {
		return VerificationReport{}, errors.New("non-completed verification cannot claim an outcome")
	} else if len(findings) != 0 {
		return VerificationReport{}, errors.New("non-completed verification cannot claim final Findings")
	}
	if err := validateEvidenceForCheck(execution.id, evidence); err != nil {
		return VerificationReport{}, err
	}
	if err := validateFindingsForEvidence(findings, evidence); err != nil {
		return VerificationReport{}, err
	}
	return VerificationReport{execution: execution, outcome: outcome, findings: append([]Finding(nil), findings...), evidence: append([]Evidence(nil), evidence...)}, nil
}

func (r VerificationReport) Execution() CheckExecution    { return r.execution }
func (r VerificationReport) Outcome() VerificationOutcome { return r.outcome }
func (r VerificationReport) Findings() []Finding          { return append([]Finding(nil), r.findings...) }
func (r VerificationReport) Evidence() []Evidence         { return append([]Evidence(nil), r.evidence...) }

// InspectionReport records a completed or limited inspection and its Findings.
type InspectionReport struct {
	execution   CheckExecution
	executions  []CheckExecution
	findings    []Finding
	evidence    []Evidence
	diagnostics []InspectionDiagnostic
}

// NewInspectionReport constructs a normalized Inspection report.
func NewInspectionReport(execution CheckExecution, findings []Finding, evidence []Evidence) (InspectionReport, error) {
	return NewCompositeInspectionReport([]InspectionReport{{execution: execution, executions: []CheckExecution{execution}, findings: findings, evidence: evidence}})
}

// NewInspectionReportWithDiagnostic preserves the existing report semantics
// while attaching bounded qualification metadata.
func NewInspectionReportWithDiagnostic(execution CheckExecution, findings []Finding, evidence []Evidence, diagnostic InspectionDiagnostic) (InspectionReport, error) {
	return NewCompositeInspectionReport([]InspectionReport{{execution: execution, executions: []CheckExecution{execution}, findings: findings, evidence: evidence, diagnostics: []InspectionDiagnostic{diagnostic}}})
}

// NewCompositeInspectionReport combines independently completed or limited Inspection checks for one Artifact.
func NewCompositeInspectionReport(reports []InspectionReport) (InspectionReport, error) {
	if len(reports) == 0 {
		return InspectionReport{}, errors.New("inspection report requires at least one check")
	}
	executions := make([]CheckExecution, 0, len(reports))
	findings := make([]Finding, 0)
	evidence := make([]Evidence, 0)
	diagnostics := make([]InspectionDiagnostic, 0)
	seenChecks := make(map[CheckID]bool)
	seenEvidence := make(map[EvidenceID]bool)
	for _, report := range reports {
		if len(report.executions) == 0 {
			return InspectionReport{}, errors.New("inspection report has no declared execution")
		}
		// Nested reports are immutable values already validated here. Validate
		// primitive constructor inputs once, then retain every child check.
		if len(report.executions) == 1 {
			execution := report.execution
			if execution.kind != CheckInspection {
				return InspectionReport{}, errors.New("inspection report requires an Inspection check")
			}
			if execution.status != ExecutionCompleted && len(report.findings) != 0 {
				return InspectionReport{}, errors.New("non-completed inspection cannot claim final Findings")
			}
			if execution.status == ExecutionCompleted && len(report.evidence) == 0 {
				return InspectionReport{}, errors.New("completed inspection requires Evidence")
			}
			if err := validateEvidenceForCheck(execution.id, report.evidence); err != nil {
				return InspectionReport{}, err
			}
			if err := validateFindingsForEvidence(report.findings, report.evidence); err != nil {
				return InspectionReport{}, err
			}
			for _, diagnostic := range report.diagnostics {
				if _, err := NewInspectionDiagnostic(diagnostic.cause, diagnostic.stage); err != nil {
					return InspectionReport{}, err
				}
			}
		}
		for _, execution := range report.executions {
			if seenChecks[execution.id] {
				return InspectionReport{}, errors.New("inspection report contains duplicate checks")
			}
			seenChecks[execution.id] = true
			executions = append(executions, execution)
		}
		for _, item := range report.evidence {
			if seenEvidence[item.id] {
				return InspectionReport{}, errors.New("inspection report contains duplicate Evidence")
			}
			seenEvidence[item.id] = true
		}
		findings = append(findings, report.findings...)
		evidence = append(evidence, report.evidence...)
		diagnostics = append(diagnostics, report.diagnostics...)
	}
	return InspectionReport{execution: executions[0], executions: executions, findings: findings, evidence: evidence, diagnostics: diagnostics}, nil
}

func (r InspectionReport) Execution() CheckExecution { return r.execution }
func (r InspectionReport) Executions() []CheckExecution {
	return append([]CheckExecution(nil), r.executions...)
}
func (r InspectionReport) Findings() []Finding  { return append([]Finding(nil), r.findings...) }
func (r InspectionReport) Evidence() []Evidence { return append([]Evidence(nil), r.evidence...) }
func (r InspectionReport) Diagnostics() []InspectionDiagnostic {
	return append([]InspectionDiagnostic(nil), r.diagnostics...)
}

func validateEvidenceForCheck(id CheckID, evidence []Evidence) error {
	for index, item := range evidence {
		if item.id.value == "" || item.checkID != id {
			return fmt.Errorf("evidence %d does not belong to check %q", index, id.value)
		}
	}
	return nil
}

func validateFindingsForEvidence(findings []Finding, evidence []Evidence) error {
	evidenceIDs := make(map[EvidenceID]bool, len(evidence))
	for _, item := range evidence {
		evidenceIDs[item.id] = true
	}
	for findingIndex, finding := range findings {
		if finding.code == "" || len(finding.evidenceID) == 0 {
			return fmt.Errorf("finding %d is invalid", findingIndex)
		}
		for _, evidenceID := range finding.evidenceID {
			if !evidenceIDs[evidenceID] {
				return fmt.Errorf("finding %d references Evidence outside its report", findingIndex)
			}
		}
	}
	return nil
}

// CheckRequirement is declared by a controller-owned operation/profile before
// execution. It carries no actual status or approval. Experiment identity is
// part of the unique CheckID; profile binds its fixed runtime/rules context.
type CheckRequirement struct {
	id       CheckID
	kind     CheckKind
	required bool
	profile  ContentDigest
}

func NewCheckRequirement(id CheckID, kind CheckKind, required bool, profile ContentDigest) (CheckRequirement, error) {
	if id.value == "" || (kind != CheckVerification && kind != CheckInspection) || profile.value == "" {
		return CheckRequirement{}, errors.New("check requirement needs a valid identifier, kind and profile")
	}
	return CheckRequirement{id, kind, required, profile}, nil
}

func (r CheckRequirement) ID() CheckID                  { return r.id }
func (r CheckRequirement) Kind() CheckKind              { return r.kind }
func (r CheckRequirement) Required() bool               { return r.required }
func (r CheckRequirement) ProfileDigest() ContentDigest { return r.profile }

// CheckPlan is immutable expected membership for one actual Run/content subject.
// The caller must freeze it before invoking its verification/inspection ports.
type CheckPlan struct {
	run          RunID
	identity     ResolvedArtifactIdentity
	digest       ContentDigest
	requirements []CheckRequirement
}

func NewCheckPlan(run RunID, artifact AcquiredArtifact, requirements []CheckRequirement) (CheckPlan, error) {
	if run.value == "" || artifact.identity.source.value == "" || artifact.digest.value == "" || len(requirements) == 0 {
		return CheckPlan{}, errors.New("check plan requires a Run, exact subject and declared checks")
	}
	seen := make(map[CheckID]bool, len(requirements))
	for _, r := range requirements {
		if _, err := NewCheckRequirement(r.id, r.kind, r.required, r.profile); err != nil {
			return CheckPlan{}, err
		}
		if seen[r.id] {
			return CheckPlan{}, errors.New("check plan contains duplicate identities")
		}
		seen[r.id] = true
	}
	return CheckPlan{run, artifact.identity, artifact.digest, append([]CheckRequirement(nil), requirements...)}, nil
}

func (p CheckPlan) Valid() bool {
	return p.run.value != "" && p.identity.source.value != "" && p.digest.value != "" && len(p.requirements) != 0
}
func (p CheckPlan) RunID() RunID                       { return p.run }
func (p CheckPlan) Identity() ResolvedArtifactIdentity { return p.identity }
func (p CheckPlan) Digest() ContentDigest              { return p.digest }

func (p CheckPlan) Requirements() []CheckRequirement {
	return append([]CheckRequirement(nil), p.requirements...)
}

// Validate compares exact membership in O(n). A failed/unavailable required
// check can match its declaration; Policy still refuses automatic ALLOW.
func (p CheckPlan) Validate(run RunID, artifact AcquiredArtifact, checks []CheckExecution) error {
	if !p.Valid() || p.run != run || p.identity != artifact.identity || p.digest != artifact.digest {
		return errors.New("check plan belongs to a different Run or subject")
	}
	expected := make(map[CheckID]CheckRequirement, len(p.requirements))
	for _, r := range p.requirements {
		expected[r.id] = r
	}
	seen := make(map[CheckID]bool, len(checks))
	for _, c := range checks {
		r, ok := expected[c.id]
		if !ok || seen[c.id] || c.kind != r.kind || c.required != r.required || c.profile != r.profile {
			return errors.New("actual check identity, kind, requirement or profile differs from its declaration")
		}
		seen[c.id] = true
	}
	for _, r := range p.requirements {
		if r.required && !seen[r.id] {
			return errors.New("actual checks do not cover the declared required plan")
		}
	}
	return nil
}

// CheckProfileDigest binds a named rule/experiment schema to its opaque fixed
// runtime descriptor. Domain does not interpret provider or ecosystem names.
func CheckProfileDigest(name, runtimeDescriptor string) (ContentDigest, error) {
	if _, err := NewCheckID(name); err != nil {
		return ContentDigest{}, err
	}
	body, err := json.Marshal([]string{name, runtimeDescriptor})
	if err != nil {
		return ContentDigest{}, err
	}
	digest := sha256.Sum256(body)
	return NewSHA256Digest(hex.EncodeToString(digest[:]))
}
