package advisory

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// report retains normalized response facts for the exact acquired subject.
// Unavailable queries never become completed zero-advisory results. A completed
// assessment with unknown impact/severity is an explicit review finding.
func (i *Inspector) report(artifact domain.AcquiredArtifact, query osvPackage, mapped bool, records []osvRecord, responses []responseFact, lookupErr error) (domain.InspectionReport, error) {
	checkID, _ := domain.NewCheckID("dependency-advisory")
	capability, status, limitation := domain.CapabilitySupported, domain.ExecutionCompleted, ""
	if !mapped {
		capability, status, limitation = domain.CapabilityUnsupported, domain.ExecutionNotExecuted, "M13_ADVISORY_MAPPING_UNSUPPORTED"
	} else if lookupErr != nil {
		status, limitation = domain.ExecutionIncomplete, "M13_ADVISORY_LOOKUP_INCOMPLETE"
		if errors.Is(lookupErr, errCachedResponse) {
			limitation = "M13_ADVISORY_CACHE_INCOMPLETE"
		}
	}
	var evidence []domain.Evidence
	addFact := func(kind, summary string) (domain.EvidenceID, error) {
		id, err := domain.NewEvidenceID(fmt.Sprintf("dependency-advisory-%d", len(evidence)))
		if err != nil {
			return domain.EvidenceID{}, err
		}
		fact, err := domain.NewEvidence(id, checkID, artifact.Identity(), artifact.Digest(), kind, summary, i.profile)
		if err == nil {
			evidence = append(evidence, fact)
		}
		return id, err
	}
	// At most 64 responses exist for a complete graph lookup. These are hashes
	// and times, never provider descriptions, credentials or executable URLs.
	for _, response := range responses {
		body, err := json.Marshal(responseProof{response.requestSHA256, response.responseSHA256, response.fetchedAt.UTC().Format(time.RFC3339Nano), response.responseBytes})
		if err != nil {
			return domain.InspectionReport{}, err
		}
		if _, err := addFact("advisory-response", string(body)); err != nil {
			return domain.InspectionReport{}, err
		}
	}
	var findings []domain.Finding
	var assessments []advisoryFinding
	var assessmentErr error
	if mapped && lookupErr == nil {
		if len(responses) == 0 {
			return domain.InspectionReport{}, errResponse
		}
		for _, record := range records {
			impact, err := assessRecord(query, record)
			if err != nil {
				assessmentErr = errors.Join(assessmentErr, err)
				status, limitation = domain.ExecutionIncomplete, "M13_ADVISORY_ASSESSMENT_INCOMPLETE"
			}
			var score float64
			known := false
			severities := append([]osvSeverity(nil), record.Severity...)
			fixed := ""
			for _, entry := range record.Affected {
				if !samePackage(query, entry.Package) {
					continue
				}
				severities = append(severities, entry.Severity...)
				for _, interval := range entry.Ranges {
					for _, event := range interval.Events {
						if fixed == "" && event.Fixed != "" {
							fixed = event.Fixed
						}
					}
				}
			}
			for _, severity := range severities {
				value, err := baseSeverity(severity)
				if err == nil {
					if !known || value > score {
						score = value
					}
					known = true
				}
			}
			// A bounded example boundary is not a promised upgrade target. The
			// provider may describe multiple disjoint affected/fixed intervals.
			body, err := json.Marshal(struct {
				ID           string        `json:"id"`
				URL          string        `json:"url"`
				Modified     string        `json:"modified"`
				Impact       versionImpact `json:"impact"`
				BaseScore    float64       `json:"base_score"`
				ScoreKnown   bool          `json:"score_known"`
				FixedExample string        `json:"fixed_example,omitempty"`
			}{record.ID, "https://osv.dev/vulnerability/" + record.ID, record.Modified.UTC().Format(time.RFC3339Nano), impact, score, known, fixed})
			if err != nil {
				return domain.InspectionReport{}, err
			}
			id, err := addFact("advisory-record", string(body))
			if err != nil {
				return domain.InspectionReport{}, err
			}
			code := ""
			if impact == impactUnknown || impact == impactAffected {
				code = "M13_ADVISORY_REVIEW"
			}
			if impact == impactAffected && known && score >= 7 {
				code = "M13_ADVISORY_HIGH"
			}
			// MAL is OSV's registered malicious-packages namespace. Provider
			// descriptions, severity labels and arbitrary alias text are not a
			// classification source; exact affected versions still apply.
			if impact == impactAffected && strings.HasPrefix(record.ID, "MAL-") {
				code = "M13_ADVISORY_MALICIOUS"
			}
			assessments = append(assessments, advisoryFinding{record.ID, record.Aliases, code, id})
		}
	}
	if status == domain.ExecutionCompleted {
		var err error
		findings, err = aliasFindings(assessments)
		if err != nil {
			return domain.InspectionReport{}, err
		}
	}
	execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, capability, status, limitation, i.profile)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	report, err := domain.NewInspectionReport(execution, findings, evidence)
	return report, errors.Join(assessmentErr, err)
}

type advisoryFinding struct {
	id       string
	aliases  []string
	code     string
	evidence domain.EvidenceID
}

// OSV aliases are symmetric and transitive. Visit each record/token once, keep
// every original fact, and emit the strongest finding for each component. This
// operates only on one acquired subject's already authenticated record set.
func aliasFindings(facts []advisoryFinding) ([]domain.Finding, error) {
	byAlias := map[string][]*advisoryFinding{}
	for n := range facts {
		fact := &facts[n]
		for _, token := range append([]string{fact.id}, fact.aliases...) {
			byAlias[token] = append(byAlias[token], fact)
		}
	}
	visited := map[*advisoryFinding]bool{}
	tokens := map[string]bool{}
	var findings []domain.Finding
	for n := range facts {
		start := &facts[n]
		if visited[start] {
			continue
		}
		visited[start] = true
		pending := []*advisoryFinding{start}
		code := ""
		var references []domain.EvidenceID
		for len(pending) != 0 {
			fact := pending[0]
			pending = pending[1:]
			references = append(references, fact.evidence)
			switch fact.code {
			case "M13_ADVISORY_MALICIOUS":
				code = fact.code
			case "M13_ADVISORY_HIGH":
				if code != "M13_ADVISORY_MALICIOUS" {
					code = fact.code
				}
			case "M13_ADVISORY_REVIEW":
				if code == "" {
					code = fact.code
				}
			}
			for _, token := range append([]string{fact.id}, fact.aliases...) {
				if tokens[token] {
					continue
				}
				tokens[token] = true
				for _, next := range byAlias[token] {
					if !visited[next] {
						visited[next] = true
						pending = append(pending, next)
					}
				}
			}
		}
		if code != "" {
			finding, err := domain.NewFinding(code, references)
			if err != nil {
				return nil, err
			}
			findings = append(findings, finding)
		}
	}
	return findings, nil
}
