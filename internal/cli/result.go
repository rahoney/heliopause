package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const operationResultSchema = "helox.operation-result/v1"

// Inspector is the use-case boundary consumed by the Inspect CLI adapter.
type Inspector interface {
	Inspect(context.Context, application.InspectRequest) (domain.OperationResult, error)
}

// ExecuteInspect invokes an injected use case and always presents a returned partial result.
func ExecuteInspect(ctx context.Context, inspector Inspector, request application.InspectRequest, machine bool, output io.Writer) (int, error) {
	if ctx == nil || inspector == nil || output == nil {
		return 2, errors.New("inspect CLI adapter requires context, use case, and output")
	}
	result, operationErr := inspector.Inspect(ctx, request)
	var presentationErr error
	if machine {
		presentationErr = WriteJSONResult(output, result)
	} else {
		presentationErr = WriteHumanResult(output, result)
	}
	if presentationErr != nil {
		return 1, errors.Join(operationErr, presentationErr)
	}
	return ExitCode(result), operationErr
}

// ExitCode maps independent operation and Policy axes to the stable M1 process contract.
func ExitCode(result domain.OperationResult) int {
	if result.Status() != domain.OperationCompleted {
		return 1
	}
	decision, ok := result.PolicyDecision()
	if !ok {
		return 1
	}
	switch decision.Decision() {
	case domain.DecisionAllow:
		return 0
	case domain.DecisionManualReview:
		return 10
	case domain.DecisionBlock:
		return 20
	default:
		return 1
	}
}

// WriteHumanResult renders a bounded summary from the immutable Operation Result.
func WriteHumanResult(output io.Writer, result domain.OperationResult) error {
	if output == nil || result.OperationID().String() == "" {
		return errors.New("human presenter requires output and a valid Operation Result")
	}
	if _, err := fmt.Fprintf(output, "Operation: %s\nOperation ID: %s\nStatus: %s\n", result.Operation(), result.OperationID(), result.Status()); err != nil {
		return fmt.Errorf("write human Operation Result: %w", err)
	}
	if runID, ok := result.RunID(); ok {
		outcome, _ := result.RunOutcome()
		if _, err := fmt.Fprintf(output, "Run ID: %s\nRun outcome: %s\n", runID, outcome); err != nil {
			return fmt.Errorf("write human Run result: %w", err)
		}
	}
	if identity, ok := result.ResolvedIdentity(); ok {
		if _, err := fmt.Fprintf(output, "Artifact: %s/%s@%s (%s)\n", identity.Source(), identity.Name(), identity.Version(), identity.Variant()); err != nil {
			return fmt.Errorf("write human Artifact identity: %w", err)
		}
	}
	if digest, ok := result.Digest(); ok {
		if _, err := fmt.Fprintf(output, "Digest: %s:%s\n", digest.Algorithm(), digest); err != nil {
			return fmt.Errorf("write human Artifact digest: %w", err)
		}
	}
	if decision, ok := result.PolicyDecision(); ok {
		if _, err := fmt.Fprintf(output, "Policy: %s (%s v%d)\nReasons: %v\n", decision.Decision(), decision.PolicyID(), decision.Version(), decision.Reasons()); err != nil {
			return fmt.Errorf("write human Policy Decision: %w", err)
		}
	}
	if operationError, ok := result.Error(); ok {
		if _, err := fmt.Fprintf(output, "Error: %s: %s\n", operationError.Code(), operationError.Message()); err != nil {
			return fmt.Errorf("write human operational error: %w", err)
		}
	}
	return nil
}

// WriteJSONResult emits one stable schema document and never serializes raw Go errors.
func WriteJSONResult(output io.Writer, result domain.OperationResult) error {
	if output == nil || result.OperationID().String() == "" {
		return errors.New("JSON presenter requires output and a valid Operation Result")
	}
	document := machineResult{
		SchemaVersion: operationResultSchema,
		OperationID:   result.OperationID().String(), Operation: string(result.Operation()), OperationStatus: string(result.Status()),
		Artifact: machineArtifact{Reference: machineReference{SourceID: result.Reference().Source().String(), Locator: result.Reference().Locator()}},
		Checks:   []machineCheck{}, EvidenceReferences: []machineEvidenceReference{},
	}
	if runID, ok := result.RunID(); ok {
		outcome, _ := result.RunOutcome()
		document.Run = &machineRun{ID: runID.String(), Outcome: string(outcome)}
	}
	if identity, ok := result.ResolvedIdentity(); ok {
		document.Artifact.ResolvedIdentity = &machineIdentity{SourceID: identity.Source().String(), Name: identity.Name(), Version: identity.Version(), Variant: identity.Variant()}
	}
	if digest, ok := result.Digest(); ok {
		document.Artifact.Digest = &machineDigest{Algorithm: digest.Algorithm(), Value: digest.String()}
	}
	for _, check := range result.Checks() {
		item := machineCheck{ID: check.ID().String(), Kind: string(check.Kind()), Required: check.Required(), Capability: string(check.Capability()), ExecutionStatus: string(check.Status())}
		item.LimitationCode, _ = check.LimitationCode()
		document.Checks = append(document.Checks, item)
	}
	for _, reference := range result.Evidence() {
		document.EvidenceReferences = append(document.EvidenceReferences, machineEvidenceReference{ID: reference.ID().String(), Handle: reference.Handle()})
	}
	if decision, ok := result.PolicyDecision(); ok {
		document.Policy = &machinePolicy{Decision: string(decision.Decision()), ID: decision.PolicyID(), Version: decision.Version(), Reasons: decision.Reasons()}
	}
	if operationError, ok := result.Error(); ok {
		document.Error = &machineError{Code: operationError.Code(), Message: operationError.Message()}
	}
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(true)
	if err := encoder.Encode(document); err != nil {
		return fmt.Errorf("encode machine Operation Result: %w", err)
	}
	return nil
}

type machineResult struct {
	SchemaVersion      string                     `json:"schema_version"`
	OperationID        string                     `json:"operation_id"`
	Operation          string                     `json:"operation"`
	OperationStatus    string                     `json:"operation_status"`
	Run                *machineRun                `json:"run,omitempty"`
	Artifact           machineArtifact            `json:"artifact"`
	Checks             []machineCheck             `json:"checks"`
	EvidenceReferences []machineEvidenceReference `json:"evidence_references"`
	Policy             *machinePolicy             `json:"policy,omitempty"`
	Error              *machineError              `json:"error,omitempty"`
}

type machineRun struct {
	ID      string `json:"id"`
	Outcome string `json:"outcome"`
}
type machineReference struct {
	SourceID string `json:"source_id"`
	Locator  string `json:"locator"`
}
type machineIdentity struct {
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Variant  string `json:"variant"`
}
type machineDigest struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}
type machineArtifact struct {
	Reference        machineReference `json:"reference"`
	ResolvedIdentity *machineIdentity `json:"resolved_identity,omitempty"`
	Digest           *machineDigest   `json:"digest,omitempty"`
}
type machineCheck struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	Required        bool   `json:"required"`
	Capability      string `json:"capability"`
	ExecutionStatus string `json:"execution_status"`
	LimitationCode  string `json:"limitation_code,omitempty"`
}
type machineEvidenceReference struct {
	ID     string `json:"id"`
	Handle string `json:"handle"`
}
type machinePolicy struct {
	Decision string   `json:"decision"`
	ID       string   `json:"id"`
	Version  uint64   `json:"version"`
	Reasons  []string `json:"reasons"`
}
type machineError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
