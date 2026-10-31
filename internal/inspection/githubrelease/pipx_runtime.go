package githubrelease

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// PipxRuntimeObservation is the consumer-owned result boundary. Only the
// composition root converts the existing external transaction's typed result.
type PipxRuntimeObservation struct {
	domain.SandboxResult
	Commands []PipxCommandObservation
}

type PipxCommandObservation struct {
	Module, UnitID, OwnerSHA256 string
	ZeroExit                    bool
}

// PipxRuntimeRunner accepts only an acquired HAA ZIP and its immutable static plan.
type PipxRuntimeRunner interface {
	InspectPipxRuntime(context.Context, domain.AcquiredArtifact, artifactpypi.PipxRuntimePlan) (PipxRuntimeObservation, error)
}

type PipxRuntimeInspector struct {
	static *StaticInspector
	runner PipxRuntimeRunner
}

func NewPipxRuntimeInspector(intakeRoot string, runner PipxRuntimeRunner) (*PipxRuntimeInspector, error) {
	if runner == nil {
		return nil, errors.New("acquired pipx runtime runner is required")
	}
	static, err := NewStaticInspector(intakeRoot)
	if err != nil {
		return nil, err
	}
	return &PipxRuntimeInspector{static: static, runner: runner}, nil
}

const pipxRuntimeCheck = "github-release-pipx-runtime"

func (i *PipxRuntimeInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i == nil || ctx == nil || i.static == nil || i.runner == nil {
		return domain.InspectionReport{}, errors.New("pipx runtime inspection is unavailable")
	}
	input, err := i.static.readPipxRuntimeInput(ctx, artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	observed, err := i.runner.InspectPipxRuntime(ctx, input.artifact, input.plan)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	return pipxRuntimeReport(input.artifact, input.plan.Units(), observed)
}

func pipxRuntimeIncomplete(code string) (domain.InspectionReport, error) {
	id, err := domain.NewCheckID(pipxRuntimeCheck)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	check, err := domain.NewCheckExecution(id, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionIncomplete, code)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	return domain.NewInspectionReport(check, nil, nil)
}

func pipxRuntimeReport(artifact domain.AcquiredArtifact, units []artifactpypi.PlannedObservationUnit, observed PipxRuntimeObservation) (domain.InspectionReport, error) {
	if observed.Status() != domain.SandboxCompleted && observed.Status() != domain.SandboxIncomplete {
		return pipxRuntimeIncomplete("P4_PIPX_RUNTIME_RESULT_INVALID")
	}
	report, err := githubDynamicReport(artifact, observed.SandboxResult, pipxRuntimeCheck)
	if err != nil || report.Execution().Status() != domain.ExecutionCompleted {
		return report, err
	}
	expected := map[string]bool{}
	for _, unit := range units {
		if unit.Coverage == artifactpypi.PostInstallCommandObservation {
			expected[unit.Candidate] = false
		}
	}
	type outcome struct {
		Module                 string `json:"module"`
		Owner                  string `json:"owner_sha256"`
		Unit                   string `json:"unit"`
		Disposition            string `json:"disposition"`
		FunctionalityAttested  bool   `json:"functionality_attested"`
		LaterExecutionEnforced bool   `json:"later_execution_enforced"`
	}
	commands := make([]outcome, 0, len(expected))
	seen := map[string]bool{}
	for _, command := range observed.Commands {
		done, exists := expected[command.Module]
		id := strings.TrimPrefix(command.UnitID, "DIRECT_IMPORT:")
		decoded, err := hex.DecodeString(id)
		if !exists || done || command.OwnerSHA256 != artifact.Digest().String() || id == command.UnitID || err != nil || len(decoded) != 32 || hex.EncodeToString(decoded) != id || seen[command.UnitID] {
			return pipxRuntimeIncomplete("P4_PIPX_COMMAND_EVIDENCE_INCOMPLETE")
		}
		expected[command.Module] = true
		seen[command.UnitID] = true
		disposition := "NOT_ATTESTED"
		if command.ZeroExit && len(report.Findings()) == 0 {
			disposition = "ATTESTED"
		}
		commands = append(commands, outcome{Module: command.Module, Owner: command.OwnerSHA256, Unit: command.UnitID, Disposition: disposition})
	}
	for _, done := range expected {
		if !done {
			return pipxRuntimeIncomplete("P4_PIPX_COMMAND_EVIDENCE_INCOMPLETE")
		}
	}

	evidenceItems := report.Evidence()
	for _, command := range commands {
		summary, err := json.Marshal(struct {
			Schema string `json:"schema"`
			Scope  string `json:"scope"`
			outcome
		}{"pipx-command-observation/v1", "post-install-command-target-import", command})
		if err != nil {
			return domain.InspectionReport{}, err
		}
		id := strings.TrimPrefix(command.Unit, "DIRECT_IMPORT:")
		evidenceID, err := domain.NewEvidenceID("pipx-command-" + strings.ToLower(command.Disposition) + "-" + id[:32])
		if err != nil {
			return domain.InspectionReport{}, err
		}
		evidence, err := domain.NewEvidence(evidenceID, report.Execution().ID(), artifact.Identity(), artifact.Digest(), "github-release-pipx-command-observation", string(summary))
		if err != nil {
			return domain.InspectionReport{}, err
		}
		evidenceItems = append(evidenceItems, evidence)
	}
	return domain.NewInspectionReport(report.Execution(), report.Findings(), evidenceItems)
}
