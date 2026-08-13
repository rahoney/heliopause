package domain

import (
	"errors"
	"fmt"
	"regexp"
)

var failureCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)

// RunLifecycle describes only the lifecycle of an Inspection Run.
type RunLifecycle string

const (
	RunCreated   RunLifecycle = "CREATED"
	RunActive    RunLifecycle = "ACTIVE"
	RunFinalized RunLifecycle = "FINALIZED"
)

// RunOutcome describes whether a finalized workflow completed or failed operationally.
type RunOutcome string

const (
	RunCompleted RunOutcome = "COMPLETED"
	RunFailed    RunOutcome = "FAILED"
)

// Decision is a Policy security decision, separate from operational success.
type Decision string

const (
	DecisionAllow        Decision = "ALLOW"
	DecisionManualReview Decision = "MANUAL_REVIEW"
	DecisionBlock        Decision = "BLOCK"
)

// PolicyDecision is a traceable terminal security decision. Its zero value is invalid.
type PolicyDecision struct {
	decision Decision
	policyID string
	version  uint64
	reasons  []string
}

// NewPolicyDecision constructs a decision with an exact Policy identity and ordered reasons.
func NewPolicyDecision(decision Decision, policyID string, version uint64, reasons []string) (PolicyDecision, error) {
	if err := validateDecision(decision); err != nil {
		return PolicyDecision{}, err
	}
	if err := validateNormalizedIdentifier(policyID, 64, "policy ID"); err != nil {
		return PolicyDecision{}, err
	}
	if version == 0 {
		return PolicyDecision{}, errors.New("policy version must be greater than zero")
	}
	if len(reasons) == 0 {
		return PolicyDecision{}, errors.New("policy decision requires at least one reason")
	}
	ownedReasons := make([]string, len(reasons))
	for index, reason := range reasons {
		if !failureCodePattern.MatchString(reason) {
			return PolicyDecision{}, fmt.Errorf("policy reason %d must be 1 to 64 uppercase identifier characters", index)
		}
		ownedReasons[index] = reason
	}
	return PolicyDecision{decision: decision, policyID: policyID, version: version, reasons: ownedReasons}, nil
}

func (d PolicyDecision) Decision() Decision { return d.decision }
func (d PolicyDecision) PolicyID() string   { return d.policyID }
func (d PolicyDecision) Version() uint64    { return d.version }

// Reasons returns an owned copy so callers cannot mutate the finalized decision.
func (d PolicyDecision) Reasons() []string { return append([]string(nil), d.reasons...) }

// InspectionRun connects an operation and exact resolved identity to one immutable terminal outcome.
type InspectionRun struct {
	id          RunID
	operationID OperationID
	reference   ArtifactReference
	identity    ResolvedArtifactIdentity
	lifecycle   RunLifecycle
	artifact    AcquiredArtifact
	hasArtifact bool
	outcome     RunOutcome
	decision    PolicyDecision
	failureCode string
}

// NewInspectionRun creates a Run after exact identity resolution and before acquisition.
func NewInspectionRun(id RunID, operationID OperationID, reference ArtifactReference, identity ResolvedArtifactIdentity) (*InspectionRun, error) {
	if id.value == "" {
		return nil, errors.New("run ID is required")
	}
	if operationID.value == "" {
		return nil, errors.New("operation ID is required")
	}
	if reference.source.value == "" {
		return nil, errors.New("artifact reference is required")
	}
	if identity.source.value == "" {
		return nil, errors.New("resolved artifact identity is required")
	}
	if reference.source != identity.source {
		return nil, errors.New("reference and resolved identity source mismatch")
	}
	return &InspectionRun{id: id, operationID: operationID, reference: reference, identity: identity, lifecycle: RunCreated}, nil
}

func (r *InspectionRun) ID() RunID                                  { return r.id }
func (r *InspectionRun) OperationID() OperationID                   { return r.operationID }
func (r *InspectionRun) Reference() ArtifactReference               { return r.reference }
func (r *InspectionRun) Identity() ResolvedArtifactIdentity         { return r.identity }
func (r *InspectionRun) Lifecycle() RunLifecycle                    { return r.lifecycle }
func (r *InspectionRun) AcquiredArtifact() (AcquiredArtifact, bool) { return r.artifact, r.hasArtifact }

// Activate begins acquisition and refuses every non-CREATED transition.
func (r *InspectionRun) Activate() error {
	if r == nil || r.lifecycle != RunCreated {
		return errors.New("inspection run can only activate from CREATED")
	}
	r.lifecycle = RunActive
	return nil
}

// BindAcquiredArtifact fixes the actual content subject for this Run.
func (r *InspectionRun) BindAcquiredArtifact(artifact AcquiredArtifact) error {
	if r == nil || r.lifecycle != RunActive {
		return errors.New("acquired artifact can only bind to an ACTIVE run")
	}
	if r.hasArtifact {
		return errors.New("acquired artifact is already bound")
	}
	if artifact.identity != r.identity {
		return errors.New("acquired artifact identity does not match the run")
	}
	if artifact.digest.value == "" {
		return errors.New("acquired artifact digest is required")
	}
	r.artifact, r.hasArtifact = artifact, true
	return nil
}

// FinalizeCompleted records exactly one valid Policy Decision on a successful workflow.
func (r *InspectionRun) FinalizeCompleted(decision PolicyDecision) error {
	if r == nil || r.lifecycle != RunActive {
		return errors.New("inspection run can only finalize from ACTIVE")
	}
	if !r.hasArtifact {
		return errors.New("completed inspection run requires an acquired artifact")
	}
	if decision.policyID == "" || len(decision.reasons) == 0 {
		return errors.New("completed inspection run requires a valid Policy Decision")
	}
	r.lifecycle, r.outcome, r.decision = RunFinalized, RunCompleted, decision
	return nil
}

// FinalizeFailed records an operational failure without creating a Policy Decision.
func (r *InspectionRun) FinalizeFailed(failureCode string) error {
	if r == nil || r.lifecycle != RunActive {
		return errors.New("inspection run can only finalize from ACTIVE")
	}
	if !failureCodePattern.MatchString(failureCode) {
		return errors.New("operational failure code must be 1 to 64 uppercase identifier characters")
	}
	r.lifecycle, r.outcome, r.failureCode = RunFinalized, RunFailed, failureCode
	return nil
}

func (r *InspectionRun) Outcome() (RunOutcome, bool) {
	if r == nil || r.lifecycle != RunFinalized {
		return "", false
	}
	return r.outcome, true
}

func (r *InspectionRun) PolicyDecision() (PolicyDecision, bool) {
	if r == nil || r.outcome != RunCompleted {
		return PolicyDecision{}, false
	}
	return r.decision, true
}

func (r *InspectionRun) FailureCode() (string, bool) {
	if r == nil || r.outcome != RunFailed {
		return "", false
	}
	return r.failureCode, true
}

func validateDecision(decision Decision) error {
	switch decision {
	case DecisionAllow, DecisionManualReview, DecisionBlock:
		return nil
	default:
		return fmt.Errorf("invalid Policy Decision %q", decision)
	}
}
