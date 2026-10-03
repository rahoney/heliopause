package pypi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// The trusted backend publishes these only after full ledger reconciliation and
// cleanup. They supplement the still-required transaction check; they cannot
// turn an incomplete transaction into a completed report.
func commandObservationEvidence(checkID domain.CheckID, artifact domain.AcquiredArtifact, plan artifactpypi.ObservationPlan, inputs []domain.AcquiredArtifact, outcomes []sandbox.PythonCommandObservation, noFindings bool) ([]domain.Evidence, error) {
	expected := map[string]bool{}
	for _, unit := range plan.Units {
		if unit.Coverage == artifactpypi.PostInstallCommandObservation {
			expected[unit.Candidate] = false
		}
	}
	owners := map[string]bool{artifact.Digest().String(): true}
	for _, input := range inputs {
		owners[input.Digest().String()] = true
	}
	seen := map[string]bool{}
	var evidenceItems []domain.Evidence
	for _, outcome := range outcomes {
		id := strings.TrimPrefix(outcome.UnitID, "DIRECT_IMPORT:")
		decoded, err := hex.DecodeString(id)
		if !owners[outcome.OwnerSHA256] || outcome.Module == "" || len(outcome.Module) > 256 || len(decoded) != 32 || err != nil || id == outcome.UnitID || seen[outcome.UnitID] {
			return nil, errors.New("invalid command observation identity")
		}
		seen[outcome.UnitID] = true
		if outcome.OwnerSHA256 == artifact.Digest().String() {
			done, ok := expected[outcome.Module]
			if !ok || done {
				return nil, errors.New("unexpected command observation")
			}
			expected[outcome.Module] = true
		}
		digest := sha256.Sum256([]byte(outcome.OwnerSHA256 + "\x00" + outcome.Module))
		name := "pypi-command-not-attested-" + hex.EncodeToString(digest[:16])
		disposition := "NOT_ATTESTED"
		if outcome.ZeroExit && noFindings {
			name = "pypi-command-attested-" + hex.EncodeToString(digest[:16])
			disposition = "ATTESTED"
		}
		summary, err := json.Marshal(struct {
			Scope                  string `json:"scope"`
			Module                 string `json:"module"`
			Owner                  string `json:"owner_sha256"`
			Unit                   string `json:"unit"`
			Disposition            string `json:"disposition"`
			FunctionalityAttested  bool   `json:"functionality_attested"`
			LaterExecutionEnforced bool   `json:"later_execution_enforced"`
		}{"post-install-command-target-import", outcome.Module, outcome.OwnerSHA256, outcome.UnitID, disposition, false, false})
		if err != nil {
			return nil, err
		}
		evidenceID, err := domain.NewEvidenceID(name)
		if err != nil {
			return nil, err
		}
		evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "pypi-command-observation", string(summary))
		if err != nil {
			return nil, err
		}
		evidenceItems = append(evidenceItems, evidence)
	}
	for _, done := range expected {
		if !done {
			return nil, errors.New("missing command observation")
		}
	}
	return evidenceItems, nil
}
