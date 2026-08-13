package domain

import (
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
}

// NewCheckExecution constructs a semantically valid check execution envelope.
func NewCheckExecution(id CheckID, kind CheckKind, required bool, capability Capability, status ExecutionStatus, limitationCode string) (CheckExecution, error) {
	if id.value == "" {
		return CheckExecution{}, errors.New("check ID is required")
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
	return CheckExecution{id: id, kind: kind, required: required, capability: capability, status: status, limitation: limitationCode}, nil
}

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
	execution  CheckExecution
	executions []CheckExecution
	findings   []Finding
	evidence   []Evidence
}

// NewInspectionReport constructs a normalized Inspection report.
func NewInspectionReport(execution CheckExecution, findings []Finding, evidence []Evidence) (InspectionReport, error) {
	return NewCompositeInspectionReport([]InspectionReport{{execution: execution, executions: []CheckExecution{execution}, findings: findings, evidence: evidence}})
}

// NewCompositeInspectionReport combines independently completed or limited Inspection checks for one Artifact.
func NewCompositeInspectionReport(reports []InspectionReport) (InspectionReport, error) {
	if len(reports) == 0 {
		return InspectionReport{}, errors.New("inspection report requires at least one check")
	}
	executions := make([]CheckExecution, 0, len(reports))
	findings := make([]Finding, 0)
	evidence := make([]Evidence, 0)
	for _, report := range reports {
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
		executions = append(executions, execution)
		findings = append(findings, report.findings...)
		evidence = append(evidence, report.evidence...)
	}
	return InspectionReport{execution: executions[0], executions: executions, findings: findings, evidence: evidence}, nil
}

func (r InspectionReport) Execution() CheckExecution { return r.execution }
func (r InspectionReport) Executions() []CheckExecution {
	return append([]CheckExecution(nil), r.executions...)
}
func (r InspectionReport) Findings() []Finding  { return append([]Finding(nil), r.findings...) }
func (r InspectionReport) Evidence() []Evidence { return append([]Evidence(nil), r.evidence...) }

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
