package npm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func pnpmKeyDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// A completed fact check means accepted observations were interpreted. Required
// lifecycle coverage is separate and cannot be completed by these facts.
func pnpmInstanceFacts(subject domain.AcquiredArtifact, instance string, layout artifactnpm.PNPMInspectionLayout, result domain.SandboxResult) (domain.InspectionReport, error) {
	return npmProjectInstanceFacts(subject, instance, layout, result, projectInspectionPNPM)
}

type projectInspectionClient uint8

const (
	projectInspectionPNPM projectInspectionClient = iota + 1
	projectInspectionBun
)

func projectInspectionPrefix(client projectInspectionClient) (string, error) {
	switch client {
	case projectInspectionPNPM:
		return "pnpm", nil
	case projectInspectionBun:
		return "bun", nil
	default:
		return "", errors.New("unknown project inspection client")
	}
}

type projectInstanceFactLayout interface {
	Digest() domain.ContentDigest
	RuntimeIdentity() string
}

func npmProjectInstanceFacts(subject domain.AcquiredArtifact, instance string, layout projectInstanceFactLayout, result domain.SandboxResult, client projectInspectionClient) (domain.InspectionReport, error) {
	prefix, err := projectInspectionPrefix(client)
	if err != nil {
		return domain.InspectionReport{}, err
	}

	if result.SessionID().String() == "" {
		return domain.InspectionReport{}, errors.New(prefix + " instance has no actual session")
	}
	summary, summaryErr := npmProjectInstanceSummary(result, client)
	if summary == "" {
		return domain.InspectionReport{}, summaryErr
	}
	key := pnpmKeyDigest(instance)
	check, err := domain.NewCheckID(prefix + "-instance-facts-" + key[:24])
	if err != nil {
		return domain.InspectionReport{}, err
	}
	execution, err := domain.NewCheckExecution(check, domain.CheckInspection, false, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	observedID, err := domain.NewEvidenceID(prefix + "-observations-" + key[:24])
	if err != nil {
		return domain.InspectionReport{}, err
	}
	observed, err := domain.NewEvidence(observedID, check, subject.Identity(), subject.Digest(), prefix+"-instance-observations", summary)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	binding, err := json.Marshal(struct{ Instance, Session, Layout, Runtime string }{key, result.SessionID().String(), layout.Digest().String(), pnpmKeyDigest(layout.RuntimeIdentity())})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	bindingID, err := domain.NewEvidenceID(prefix + "-binding-" + key[:24])
	if err != nil {
		return domain.InspectionReport{}, err
	}
	bound, err := domain.NewEvidence(bindingID, check, subject.Identity(), subject.Digest(), prefix+"-instance-binding", string(binding))
	if err != nil {
		return domain.InspectionReport{}, err
	}
	findings := []domain.Finding{}
	for _, code := range dynamicFindingCodes(result.Observations()) {
		finding, err := domain.NewFinding(code, []domain.EvidenceID{observedID, bindingID})
		if err != nil {
			return domain.InspectionReport{}, err
		}
		findings = append(findings, finding)
	}
	report, err := domain.NewInspectionReport(execution, findings, []domain.Evidence{observed, bound})
	return report, errors.Join(summaryErr, err)
}

func pnpmCoverageReport(subject domain.AcquiredArtifact, expected, attempted, completed int, failed bool) (domain.InspectionReport, error) {
	return npmProjectCoverageReport(subject, expected, attempted, completed, failed, projectInspectionPNPM)
}

func npmProjectCoverageReport(subject domain.AcquiredArtifact, expected, attempted, completed int, failed bool, client projectInspectionClient) (domain.InspectionReport, error) {
	prefix, err := projectInspectionPrefix(client)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	incomplete, notExecuted := "M13_PNPM_DYNAMIC_INCOMPLETE", "M13_PNPM_DYNAMIC_NOT_EXECUTED"
	if client == projectInspectionBun {
		incomplete, notExecuted = "M13_BUN_DYNAMIC_INCOMPLETE", "M13_BUN_DYNAMIC_NOT_EXECUTED"
	}

	check, err := domain.NewCheckID(prefix + "-dynamic-lifecycle")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	status, code := domain.ExecutionCompleted, ""
	if failed || completed != expected {
		status, code = domain.ExecutionIncomplete, incomplete
		if attempted == 0 {
			status, code = domain.ExecutionNotExecuted, notExecuted
		}
	}
	execution, err := domain.NewCheckExecution(check, domain.CheckInspection, true, domain.CapabilitySupported, status, code)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	id, err := domain.NewEvidenceID(prefix + "-instance-coverage")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	evidence, err := domain.NewEvidence(id, check, subject.Identity(), subject.Digest(), prefix+"-instance-coverage", fmt.Sprintf("expected=%d attempted=%d completed=%d", expected, attempted, completed))
	if err != nil {
		return domain.InspectionReport{}, err
	}
	// The aggregate is a coverage count, never a fabricated Sandbox result.
	return domain.NewInspectionReport(execution, nil, []domain.Evidence{evidence})
}

// An unrepresentable full summary is still an operational failure. Retain only
// the existing four suspicious interpretations in a bounded fallback, tied to
// this actual Session; never drop accepted security facts or widen the ceiling.
func npmProjectInstanceSummary(result domain.SandboxResult, client projectInspectionClient) (string, error) {
	prefix, err := projectInspectionPrefix(client)
	if err != nil {
		return "", err
	}

	summary, original := result.ObservationSummary()
	if original == nil {
		return summary, nil
	}
	counts := map[string]uint64{}
	for _, observation := range result.Observations() {
		for _, code := range dynamicFindingCodes([]domain.SandboxObservation{observation}) {
			counts[code] += min(observation.Count(), domain.MaximumObservationSummaryCount-counts[code])
		}
	}
	body, err := json.Marshal(struct {
		Schema            string            `json:"schema"`
		Session           string            `json:"session"`
		SummaryIncomplete bool              `json:"summary_incomplete"`
		Suspicious        map[string]uint64 `json:"suspicious"`
	}{prefix + "-partial-observations/v1", result.SessionID().String(), true, counts})
	if err != nil {
		return "", errors.Join(original, err)
	}
	return string(body), original
}
