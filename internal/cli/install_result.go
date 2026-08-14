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

// Installer is the M4 use-case boundary consumed by the CLI adapter.
type Installer interface {
	Install(context.Context, application.InstallRequest) (application.InstallOutcome, error)
}

func ExecuteInstall(ctx context.Context, installer Installer, request application.InstallRequest, machine bool, output io.Writer) (int, error) {
	if ctx == nil || installer == nil || output == nil {
		return 2, errors.New("install CLI adapter requires context, use case, and output")
	}
	outcome, operationErr := installer.Install(ctx, request)
	var presentationErr error
	if machine {
		presentationErr = WriteJSONInstallResult(output, outcome, operationErr)
	} else {
		presentationErr = WriteHumanInstallResult(output, outcome, operationErr)
	}
	if presentationErr != nil {
		return 1, errors.Join(operationErr, presentationErr)
	}
	return InstallExitCode(outcome, operationErr), operationErr
}

func InstallExitCode(outcome application.InstallOutcome, operationErr error) int {
	if operationErr != nil {
		return 1
	}
	switch outcome.Inspection().Decision().Decision() {
	case domain.DecisionAllow:
		if outcome.Promoted().Target().String() != "" {
			return 0
		}
		return 1
	case domain.DecisionManualReview:
		return 10
	case domain.DecisionBlock:
		return 20
	default:
		return 1
	}
}

func WriteHumanInstallResult(output io.Writer, outcome application.InstallOutcome, operationErr error) error {
	inspection := outcome.Inspection()
	if output == nil || inspection.OperationID().String() == "" {
		return errors.New("human presenter requires a started Install operation")
	}
	status, promotion := installStatuses(outcome, operationErr)
	if _, err := fmt.Fprintf(output, "Operation: %s\nOperation ID: %s\nStatus: %s\nTarget: %s\nPromotion: %s\n", domain.OperationInstall, inspection.OperationID(), status, inspection.Request().Context().Target(), promotion); err != nil {
		return err
	}
	if decision := inspection.Decision(); decision.PolicyID() != "" {
		if _, err := fmt.Fprintf(output, "Policy: %s (%s v%d)\nReasons: %v\n", decision.Decision(), decision.PolicyID(), decision.Version(), decision.Reasons()); err != nil {
			return err
		}
	}
	if outcome.Bundle().ManifestID().String() != "" {
		if _, err := fmt.Fprintf(output, "Manifest: sha256:%s\nVerified entries: %d\n", outcome.Bundle().ManifestID(), len(outcome.Bundle().Set().Inspected().Inspections())); err != nil {
			return err
		}
	}
	if operationErr != nil {
		_, err := fmt.Fprintf(output, "Error: %s: Install operation failed.\n", outcome.FailureCode())
		return err
	}
	return nil
}

func WriteJSONInstallResult(output io.Writer, outcome application.InstallOutcome, operationErr error) error {
	inspection := outcome.Inspection()
	if output == nil || inspection.OperationID().String() == "" {
		return errors.New("JSON presenter requires a started Install operation")
	}
	status, promotion := installStatuses(outcome, operationErr)
	document := installMachineResult{
		SchemaVersion: operationResultSchema, OperationID: inspection.OperationID().String(), Operation: string(domain.OperationInstall),
		OperationStatus: status, Artifact: machineArtifact{Reference: machineReference{SourceID: inspection.Request().Reference().Source().String(), Locator: inspection.Request().Reference().Locator()}},
		Checks: []machineCheck{}, EvidenceReferences: []machineEvidenceReference{}, Target: inspection.Request().Context().Target().String(), PromotionStatus: promotion,
	}
	if inspection.Set().Valid() {
		for _, item := range inspection.Set().Inspections() {
			for _, check := range item.Checks() {
				machine := machineCheck{ID: check.ID().String(), Kind: string(check.Kind()), Required: check.Required(), Capability: string(check.Capability()), ExecutionStatus: string(check.Status())}
				machine.LimitationCode, _ = check.LimitationCode()
				document.Checks = append(document.Checks, machine)
			}
			for _, reference := range item.Evidence() {
				document.EvidenceReferences = append(document.EvidenceReferences, machineEvidenceReference{ID: reference.ID().String(), Handle: reference.Handle()})
			}
		}
	}
	if decision := inspection.Decision(); decision.PolicyID() != "" {
		document.Policy = &machinePolicy{Decision: string(decision.Decision()), ID: decision.PolicyID(), Version: decision.Version(), Reasons: decision.Reasons()}
	}
	if outcome.Bundle().ManifestID().String() != "" {
		document.VerifiedSet = &machineVerifiedSet{ManifestID: outcome.Bundle().ManifestID().String(), SBOMHandle: "staging:" + outcome.Bundle().ManifestID().String() + ":sbom.cdx.json", EntryCount: len(outcome.Bundle().Set().Inspected().Inspections())}
		if outcome.Staged().Handle() != "" {
			document.VerifiedSet.StagingHandle = outcome.Staged().Handle()
		}
	}
	if operationErr != nil {
		document.Error = &machineError{Code: outcome.FailureCode(), Message: "Install operation failed."}
	}
	return json.NewEncoder(output).Encode(document)
}

func installStatuses(outcome application.InstallOutcome, operationErr error) (string, string) {
	if operationErr != nil {
		return string(domain.OperationFailed), "FAILED"
	}
	if outcome.Promoted().Target().String() != "" {
		return string(domain.OperationCompleted), "COMPLETED"
	}
	return string(domain.OperationNotPerformed), "NOT_PERFORMED"
}

type installMachineResult struct {
	SchemaVersion      string                     `json:"schema_version"`
	OperationID        string                     `json:"operation_id"`
	Operation          string                     `json:"operation"`
	OperationStatus    string                     `json:"operation_status"`
	Artifact           machineArtifact            `json:"artifact"`
	Checks             []machineCheck             `json:"checks"`
	EvidenceReferences []machineEvidenceReference `json:"evidence_references"`
	Policy             *machinePolicy             `json:"policy,omitempty"`
	Target             string                     `json:"target"`
	VerifiedSet        *machineVerifiedSet        `json:"verified_set,omitempty"`
	PromotionStatus    string                     `json:"promotion_status"`
	Error              *machineError              `json:"error,omitempty"`
}

type machineVerifiedSet struct {
	ManifestID    string `json:"manifest_id"`
	SBOMHandle    string `json:"sbom_handle"`
	StagingHandle string `json:"staging_handle,omitempty"`
	EntryCount    int    `json:"entry_count"`
}
