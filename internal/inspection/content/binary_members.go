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
	// Size may declare a checked member that could not be buffered. Zero
	// defaults to len(Body); it never substitutes for the expected digest.
	Size uint64
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

// BinaryInspection retains bounded facts, never member bodies. Archive owners
// may inspect one member at a time and publish Report only after their complete
// integrity/metadata validation succeeds.
type BinaryInspection struct {
	artifact          domain.AcquiredArtifact
	checks            []domain.CheckRequirement
	evidence          [2][]domain.Evidence
	findings          []domain.Finding
	seen              map[string]bool
	total             uint64
	selected          int
	complete, signals bool
	firstErr          error
}

func NewBinaryInspection(artifact domain.AcquiredArtifact) (*BinaryInspection, error) {
	if artifact.Digest().String() == "" {
		return nil, errors.New("binary inspection subject is unavailable")
	}
	checks, err := BinaryChecks()
	if err != nil {
		return nil, err
	}
	return &BinaryInspection{artifact: artifact, checks: checks, seen: make(map[string]bool), complete: true}, nil
}

// Inspect retains the first error and every independently validated signal.
// Unsupported/packed members cannot turn into complete aggregate coverage.
func (b *BinaryInspection) Inspect(ctx context.Context, member BinaryMember) error {
	if b == nil || len(b.checks) != 2 {
		return errors.New("binary inspection request is unavailable")
	}
	if ctx == nil {
		b.complete = false
		b.firstErr = errors.Join(b.firstErr, errors.New("binary inspection context is unavailable"))
		return b.firstErr
	}
	if err := ctx.Err(); err != nil {
		b.complete = false
		if b.firstErr == nil {
			b.firstErr = err
		}
		return err
	}
	index := b.selected
	b.selected++
	size := member.Size
	if size == 0 {
		size = uint64(len(member.Body))
	}
	if b.selected > ports.SourceAnalysisTargetLimit || b.seen[member.Path] || member.Path == "" || size > ports.SourceAnalysisFileLimit || size > ports.SourceAnalysisTotalLimit-b.total || size != uint64(len(member.Body)) {
		b.complete = false
		err := errors.New("binary member identity or size exceeds fixed analysis bounds")
		if b.firstErr == nil {
			b.firstErr = err
		}
		return err
	}
	b.seen[member.Path] = true
	b.total += uint64(len(member.Body))
	report, memberErr := InspectELF(ctx, b.artifact, member.Path, member.Body, member.Mode, member.Digest)
	if b.firstErr == nil {
		b.firstErr = memberErr
	}
	executions := report.Executions()
	if len(executions) != 2 {
		b.complete = false
		return memberErr
	}
	b.complete = b.complete && executions[0].Status() == domain.ExecutionCompleted
	b.signals = b.signals || executions[1].Status() == domain.ExecutionCompleted
	ids := make(map[domain.EvidenceID]domain.EvidenceID)
	for _, fact := range report.Evidence() {
		kind := 0
		if fact.CheckID() == b.checks[1].ID() {
			kind = 1
		}
		id, _ := domain.NewEvidenceID(fmt.Sprintf("binary-member-%05d-%d", index, kind))
		execution := executions[kind]
		limitation, _ := execution.LimitationCode()
		summary := fmt.Sprintf("%s Member capability: %s; status: %s; limitation: %s.", fact.Summary(), execution.Capability(), execution.Status(), limitation)
		bound, err := domain.NewEvidence(id, fact.CheckID(), b.artifact.Identity(), b.artifact.Digest(), fact.Kind(), summary, fact.ProfileDigest())
		if err != nil {
			b.complete = false
			if b.firstErr == nil {
				b.firstErr = err
			}
			return err
		}
		ids[fact.ID()] = id
		b.evidence[kind] = append(b.evidence[kind], bound)
	}
	for _, finding := range report.Findings() {
		var refs []domain.EvidenceID
		for _, id := range finding.EvidenceIDs() {
			refs = append(refs, ids[id])
		}
		bound, err := domain.NewFinding(finding.Code(), refs)
		if err != nil {
			b.complete = false
			if b.firstErr == nil {
				b.firstErr = err
			}
			return err
		}
		b.findings = append(b.findings, bound)
	}
	return memberErr
}

func (b *BinaryInspection) Report(ctx context.Context) (domain.InspectionReport, error) {
	if b == nil || ctx == nil || len(b.checks) != 2 {
		return domain.InspectionReport{}, errors.New("binary inspection request is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	complete := b.complete && b.selected != 0
	var reports []domain.InspectionReport
	for index, check := range b.checks {
		status, limitation := domain.ExecutionCompleted, ""
		if index == 0 && !complete {
			status, limitation = domain.ExecutionIncomplete, "M13_BINARY_ANALYSIS_INCOMPLETE"
		} else if index == 1 && !b.signals {
			status, limitation = domain.ExecutionNotExecuted, "M13_BINARY_SIGNALS_UNAVAILABLE"
		}
		execution, _ := domain.NewCheckExecution(check.ID(), domain.CheckInspection, check.Required(), domain.CapabilitySupported, status, limitation, check.ProfileDigest())
		id, _ := domain.NewEvidenceID(fmt.Sprintf("binary-members-summary-%d", index))
		fact, err := domain.NewEvidence(id, check.ID(), b.artifact.Identity(), b.artifact.Digest(), "binary-content-inventory", fmt.Sprintf("Selected binary members: %d; inspected: %d; bytes: %d; coverage complete: %t. Zero selection alone is not an applicability proof.", b.selected, len(b.seen), b.total, complete), check.ProfileDigest())
		if err != nil {
			return domain.InspectionReport{}, err
		}
		facts := append(append([]domain.Evidence(nil), b.evidence[index]...), fact)
		var findings []domain.Finding
		if index == 1 {
			findings = b.findings
		}
		report, err := domain.NewInspectionReport(execution, findings, facts)
		if err != nil {
			return domain.InspectionReport{}, err
		}
		reports = append(reports, report)
	}
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	report, err := domain.NewCompositeInspectionReport(reports)
	return report, errors.Join(b.firstErr, err)
}

// InspectBinaries uses the same incremental owner for an already bounded map.
// An empty selection is incomplete without a complete applicability census.
func InspectBinaries(ctx context.Context, artifact domain.AcquiredArtifact, members []BinaryMember) (domain.InspectionReport, error) {
	inspection, err := NewBinaryInspection(artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	if len(members) > ports.SourceAnalysisTargetLimit {
		inspection.complete, inspection.selected, inspection.firstErr = false, len(members), errors.New("binary member count exceeds fixed analysis bounds")
	} else {
		for _, member := range members {
			if err := inspection.Inspect(ctx, member); err != nil && ctx != nil && ctx.Err() != nil {
				return domain.InspectionReport{}, err
			}
		}
	}
	return inspection.Report(ctx)
}
