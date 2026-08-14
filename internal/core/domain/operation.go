package domain

import (
	"errors"
	"fmt"
)

// OperationType identifies a user-requested workflow without encoding its result.
type OperationType string

const (
	OperationInspect OperationType = "INSPECT"
	OperationInstall OperationType = "INSTALL"
)

// OperationStatus is independent from Run Outcome and Policy Decision.
type OperationStatus string

const (
	OperationCompleted    OperationStatus = "COMPLETED"
	OperationFailed       OperationStatus = "FAILED"
	OperationPaused       OperationStatus = "PAUSED"
	OperationNotPerformed OperationStatus = "NOT_PERFORMED"
)

// OperationError is a stable sanitized machine error, not the underlying Go error.
type OperationError struct {
	code    string
	message string
}

// NewOperationError constructs a bounded operational error safe for result output.
func NewOperationError(code, message string) (OperationError, error) {
	if !failureCodePattern.MatchString(code) {
		return OperationError{}, errors.New("operation error code must be an uppercase identifier")
	}
	if err := validateBoundedText(message, 256, "operation error message"); err != nil {
		return OperationError{}, err
	}
	if containsSensitiveEvidence(message) {
		return OperationError{}, errors.New("operation error message contains a sensitive pattern")
	}
	return OperationError{code: code, message: message}, nil
}

func (e OperationError) Code() string    { return e.code }
func (e OperationError) Message() string { return e.message }

// OperationResultData carries one immutable Operation Result into its validating constructor.
type OperationResultData struct {
	OperationID      OperationID
	Status           OperationStatus
	Reference        ArtifactReference
	ResolvedIdentity ResolvedArtifactIdentity
	Digest           ContentDigest
	RunID            RunID
	RunOutcome       RunOutcome
	Checks           []CheckExecution
	Evidence         []EvidenceReference
	PolicyDecision   PolicyDecision
	OperationalError OperationError
}

// OperationResult is the single source for human and machine presentation.
type OperationResult struct {
	operationID      OperationID
	status           OperationStatus
	reference        ArtifactReference
	resolvedIdentity ResolvedArtifactIdentity
	digest           ContentDigest
	runID            RunID
	runOutcome       RunOutcome
	checks           []CheckExecution
	evidence         []EvidenceReference
	decision         PolicyDecision
	operationalError OperationError
}

// NewInspectOperationResult validates the independent axes of an Inspect result.
func NewInspectOperationResult(data OperationResultData) (OperationResult, error) {
	if data.OperationID.value == "" || data.Reference.source.value == "" {
		return OperationResult{}, errors.New("operation ID and Artifact Reference are required")
	}
	if data.Status != OperationCompleted && data.Status != OperationFailed {
		return OperationResult{}, fmt.Errorf("invalid M1 Inspect operation status %q", data.Status)
	}
	if data.Status == OperationCompleted {
		if data.RunID.value == "" || data.RunOutcome != RunCompleted || data.ResolvedIdentity.source.value == "" || data.Digest.value == "" || data.PolicyDecision.policyID == "" {
			return OperationResult{}, errors.New("completed Inspect result requires completed Run, exact subject, and Policy Decision")
		}
		if data.OperationalError.code != "" {
			return OperationResult{}, errors.New("completed Inspect result cannot contain an operational error")
		}
	} else {
		if data.OperationalError.code == "" {
			return OperationResult{}, errors.New("failed Inspect result requires an operational error")
		}
		if data.PolicyDecision.policyID != "" {
			return OperationResult{}, errors.New("failed Inspect result cannot contain a Policy Decision")
		}
		if data.RunID.value != "" && data.RunOutcome != RunFailed {
			return OperationResult{}, errors.New("failed Inspect result with a Run requires failed Run Outcome")
		}
		if data.RunID.value == "" && data.RunOutcome != "" {
			return OperationResult{}, errors.New("run outcome cannot exist without a Run ID")
		}
	}
	return OperationResult{
		operationID: data.OperationID, status: data.Status, reference: data.Reference,
		resolvedIdentity: data.ResolvedIdentity, digest: data.Digest,
		runID: data.RunID, runOutcome: data.RunOutcome,
		checks: append([]CheckExecution(nil), data.Checks...), evidence: append([]EvidenceReference(nil), data.Evidence...),
		decision: data.PolicyDecision, operationalError: data.OperationalError,
	}, nil
}

func (r OperationResult) OperationID() OperationID     { return r.operationID }
func (r OperationResult) Operation() OperationType     { return OperationInspect }
func (r OperationResult) Status() OperationStatus      { return r.status }
func (r OperationResult) Reference() ArtifactReference { return r.reference }
func (r OperationResult) Checks() []CheckExecution     { return append([]CheckExecution(nil), r.checks...) }
func (r OperationResult) Evidence() []EvidenceReference {
	return append([]EvidenceReference(nil), r.evidence...)
}

func (r OperationResult) ResolvedIdentity() (ResolvedArtifactIdentity, bool) {
	return r.resolvedIdentity, r.resolvedIdentity.source.value != ""
}
func (r OperationResult) Digest() (ContentDigest, bool)  { return r.digest, r.digest.value != "" }
func (r OperationResult) RunID() (RunID, bool)           { return r.runID, r.runID.value != "" }
func (r OperationResult) RunOutcome() (RunOutcome, bool) { return r.runOutcome, r.runOutcome != "" }
func (r OperationResult) PolicyDecision() (PolicyDecision, bool) {
	return r.decision, r.decision.policyID != ""
}
func (r OperationResult) Error() (OperationError, bool) {
	return r.operationalError, r.operationalError.code != ""
}

// PolicyInput is the normalized, I/O-free input to M1 Policy evaluation.
type PolicyInput struct {
	runID        RunID
	artifact     AcquiredArtifact
	verification VerificationReport
	inspection   InspectionReport
	evidence     []EvidenceReference
}

// NewPolicyInput validates a complete normalized check set for one exact subject.
func NewPolicyInput(runID RunID, artifact AcquiredArtifact, verification VerificationReport, inspection InspectionReport, evidence []EvidenceReference) (PolicyInput, error) {
	if runID.value == "" || artifact.identity.source.value == "" {
		return PolicyInput{}, errors.New("policy input requires Run ID and acquired Artifact")
	}
	if verification.execution.id.value == "" || inspection.execution.id.value == "" {
		return PolicyInput{}, errors.New("policy input requires Verification and Inspection reports")
	}
	if len(evidence) == 0 {
		return PolicyInput{}, errors.New("policy input requires recorded Evidence references")
	}
	reportEvidence := append(verification.Evidence(), inspection.Evidence()...)
	wantEvidence := make(map[EvidenceID]bool, len(reportEvidence))
	for _, item := range reportEvidence {
		if item.identity != artifact.identity || item.digest != artifact.digest {
			return PolicyInput{}, errors.New("policy input Evidence subject does not match acquired Artifact")
		}
		wantEvidence[item.id] = true
	}
	if len(evidence) != len(wantEvidence) {
		return PolicyInput{}, errors.New("policy input Evidence references do not cover normalized Evidence")
	}
	seenReferences := make(map[EvidenceID]bool, len(evidence))
	for _, reference := range evidence {
		if !wantEvidence[reference.id] || seenReferences[reference.id] {
			return PolicyInput{}, errors.New("policy input contains unknown or duplicate Evidence references")
		}
		seenReferences[reference.id] = true
	}
	return PolicyInput{runID: runID, artifact: artifact, verification: verification, inspection: inspection, evidence: append([]EvidenceReference(nil), evidence...)}, nil
}

func (i PolicyInput) RunID() RunID                     { return i.runID }
func (i PolicyInput) Artifact() AcquiredArtifact       { return i.artifact }
func (i PolicyInput) Verification() VerificationReport { return i.verification }
func (i PolicyInput) Inspection() InspectionReport     { return i.inspection }
func (i PolicyInput) Evidence() []EvidenceReference {
	return append([]EvidenceReference(nil), i.evidence...)
}

// Valid reports whether the input was created through NewPolicyInput.
func (i PolicyInput) Valid() bool {
	return i.runID.value != "" && i.artifact.identity.source.value != "" && i.verification.execution.id.value != "" && i.inspection.execution.id.value != "" && len(i.evidence) != 0
}
