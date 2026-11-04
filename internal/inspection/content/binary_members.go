package content

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// BinaryMember refers to DATA already validated by its archive owner. Inspecting
// it neither copies the body nor opens a path or executes artifact code.
type BinaryMember struct {
	Path   string
	Body   []byte
	Mode   uint32
	Digest domain.ContentDigest
}

// IsBinary selects native/container formats by bytes first, including formats
// outside the ELF capability. Names are additional hints, never format proof.
func IsBinary(name string, prefix []byte) bool {
	for _, magic := range [][]byte{
		{0x7f, 'E', 'L', 'F'}, {'M', 'Z'}, {0, 'a', 's', 'm'},
		{0xfe, 0xed, 0xfa, 0xce}, {0xce, 0xfa, 0xed, 0xfe},
		{0xfe, 0xed, 0xfa, 0xcf}, {0xcf, 0xfa, 0xed, 0xfe},
		{0xca, 0xfe, 0xba, 0xbe}, {0xbe, 0xba, 0xfe, 0xca},
		{0xca, 0xfe, 0xba, 0xbf}, {0xbf, 0xba, 0xfe, 0xca},
		[]byte("!<arch>\n"),
	} {
		if bytes.HasPrefix(prefix, magic) {
			return true
		}
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".so", ".node", ".dll", ".exe", ".dylib", ".wasm", ".a", ".o":
		return true
	}
	return strings.Contains(strings.ToLower(path.Base(name)), ".so.")
}

// InspectBinaries reuses the single-member decoder, preserving its per-member
// facts under two aggregate checks. Any unsupported, packed or invalid member
// prevents complete coverage, while valid signals from other members survive.
// An empty selection is incomplete until the caller supplies a complete census.
func InspectBinaries(ctx context.Context, artifact domain.AcquiredArtifact, members []BinaryMember) (domain.InspectionReport, error) {
	if ctx == nil || artifact.Digest().String() == "" {
		return domain.InspectionReport{}, errors.New("binary inspection subject is unavailable")
	}
	checks, err := BinaryChecks()
	if err != nil {
		return domain.InspectionReport{}, err
	}
	var evidence [2][]domain.Evidence
	var findings []domain.Finding
	seen := make(map[string]bool)
	complete, signals := len(members) != 0, false
	selectedCount := len(members)
	var total uint64
	var firstErr error
	if len(members) > ports.SourceAnalysisTargetLimit {
		complete, firstErr = false, errors.New("binary member count exceeds fixed analysis bounds")
		members = nil
	}
	for index, member := range members {
		if err := ctx.Err(); err != nil {
			return domain.InspectionReport{}, err
		}
		if seen[member.Path] || member.Path == "" || len(member.Body) > ports.SourceAnalysisFileLimit || uint64(len(member.Body)) > ports.SourceAnalysisTotalLimit-total {
			complete = false
			if firstErr == nil {
				firstErr = errors.New("binary member identity or size exceeds fixed analysis bounds")
			}
			break
		}
		seen[member.Path] = true
		total += uint64(len(member.Body))
		report, memberErr := InspectELF(ctx, artifact, member.Path, member.Body, member.Mode, member.Digest)
		if firstErr == nil {
			firstErr = memberErr
		}
		if len(report.Executions()) != 2 {
			complete = false
			continue
		}
		complete = complete && report.Executions()[0].Status() == domain.ExecutionCompleted
		signals = signals || report.Executions()[1].Status() == domain.ExecutionCompleted
		ids := make(map[domain.EvidenceID]domain.EvidenceID)
		for _, fact := range report.Evidence() {
			kind := 0
			if fact.CheckID() == checks[1].ID() {
				kind = 1
			}
			id, _ := domain.NewEvidenceID(fmt.Sprintf("binary-member-%05d-%d", index, kind))
			execution := report.Executions()[kind]
			limitation, _ := execution.LimitationCode()
			summary := fmt.Sprintf("%s Member capability: %s; status: %s; limitation: %s.", fact.Summary(), execution.Capability(), execution.Status(), limitation)
			bound, err := domain.NewEvidence(id, fact.CheckID(), artifact.Identity(), artifact.Digest(), fact.Kind(), summary, fact.ProfileDigest())
			if err != nil {
				return domain.InspectionReport{}, err
			}
			ids[fact.ID()] = id
			evidence[kind] = append(evidence[kind], bound)
		}
		for _, finding := range report.Findings() {
			var refs []domain.EvidenceID
			for _, id := range finding.EvidenceIDs() {
				refs = append(refs, ids[id])
			}
			bound, err := domain.NewFinding(finding.Code(), refs)
			if err != nil {
				return domain.InspectionReport{}, err
			}
			findings = append(findings, bound)
		}
	}
	var reports []domain.InspectionReport
	for index, check := range checks {
		status, limitation := domain.ExecutionCompleted, ""
		if index == 0 && !complete {
			status, limitation = domain.ExecutionIncomplete, "M13_BINARY_ANALYSIS_INCOMPLETE"
		} else if index == 1 && !signals {
			status, limitation = domain.ExecutionNotExecuted, "M13_BINARY_SIGNALS_UNAVAILABLE"
		}
		execution, _ := domain.NewCheckExecution(check.ID(), domain.CheckInspection, check.Required(), domain.CapabilitySupported, status, limitation, check.ProfileDigest())
		id, _ := domain.NewEvidenceID(fmt.Sprintf("binary-members-summary-%d", index))
		fact, err := domain.NewEvidence(id, check.ID(), artifact.Identity(), artifact.Digest(), "binary-content-inventory", fmt.Sprintf("Selected binary members: %d; inspected: %d; bytes: %d; coverage complete: %t. Zero selection alone is not an applicability proof.", selectedCount, len(seen), total, complete), check.ProfileDigest())
		if err != nil {
			return domain.InspectionReport{}, err
		}
		evidence[index] = append(evidence[index], fact)
		var selected []domain.Finding
		if index == 1 {
			selected = findings
		}
		report, err := domain.NewInspectionReport(execution, selected, evidence[index])
		if err != nil {
			return domain.InspectionReport{}, err
		}
		reports = append(reports, report)
	}
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	report, err := domain.NewCompositeInspectionReport(reports)
	return report, errors.Join(firstErr, err)
}
