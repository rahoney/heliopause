package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

// AdvisoryEvidenceReader is the existing exact-subject recorded-fact read. It
// performs no query, inspection, Policy evaluation or installation.
type AdvisoryEvidenceReader interface {
	ReadReference(context.Context, domain.RunID, domain.EvidenceReference, domain.ResolvedArtifactIdentity, domain.ContentDigest) (domain.Evidence, domain.ContentDigest, error)
}

func SetAdvisoryEvidenceReader(root *cobra.Command, reader AdvisoryEvidenceReader) error {
	if root == nil || reader == nil {
		return errors.New("advisory display requires its recorded Evidence reader")
	}
	presentationFor(root.OutOrStdout()).readAdvisoryEvidence = func(run domain.RunID, ref domain.EvidenceReference, identity domain.ResolvedArtifactIdentity, digest domain.ContentDigest) (domain.Evidence, domain.ContentDigest, error) {
		return reader.ReadReference(contextOrBackground(root.Context()), run, ref, identity, digest)
	}
	return nil
}

// This is the producer's normalized record wire shape, not the provider's raw
// response. The UI displays its assessed impact/base score without recalculating.
type advisoryRecordView struct {
	ID           string  `json:"id"`
	URL          string  `json:"url"`
	Modified     string  `json:"modified"`
	Impact       string  `json:"impact"`
	BaseScore    float64 `json:"base_score"`
	ScoreKnown   bool    `json:"score_known"`
	FixedExample string  `json:"fixed_example,omitempty"`
}

type advisoryDetails struct {
	RunID   string               `json:"run_id"`
	Subject machineIdentity      `json:"subject"`
	SHA256  string               `json:"sha256"`
	Status  string               `json:"status"`
	Issue   string               `json:"issue,omitempty"`
	Records []advisoryRecordView `json:"records"`
}

// Only a non-ALLOW completed advisory check needs default record detail. Normal
// success adds no record I/O. Missing/corrupt detail remains a separate view
// state and cannot replace the original operation, Policy or exit semantics.
func (p *presentation) advisoryDetails(plan domain.CheckPlan, checks []domain.CheckExecution, refs []domain.EvidenceReference, decision domain.PolicyDecision) *advisoryDetails {
	if p.writeErr != nil || decision.PolicyID() == "" || decision.Decision() == domain.DecisionAllow {
		return nil
	}
	var profile domain.ContentDigest
	for _, check := range checks {
		if check.ID().String() == "dependency-advisory" && check.Status() == domain.ExecutionCompleted && check.Capability() == domain.CapabilitySupported {
			profile = check.ProfileDigest()
		}
	}
	if profile.String() == "" {
		return nil
	}
	identity := plan.Identity()
	view := &advisoryDetails{RunID: plan.RunID().String(), Subject: machineIdentity{identity.Source().String(), identity.Name(), identity.Version(), identity.Variant()}, SHA256: plan.Digest().String(), Status: "UNAVAILABLE", Issue: "EVIDENCE_READER_UNAVAILABLE", Records: []advisoryRecordView{}}
	if !plan.Valid() {
		view.Issue = "CHECK_PLAN_UNAVAILABLE"
		return view
	}
	if p.readAdvisoryEvidence == nil {
		return view
	}
	declared := false
	for _, requirement := range plan.Requirements() {
		declared = declared || requirement.ID().String() == "dependency-advisory" && requirement.ProfileDigest() == profile
	}
	view.Status, view.Issue = "INCOMPLETE", "EVIDENCE_BINDING_INVALID"
	if !declared {
		return view
	}
	seen := map[domain.EvidenceID]bool{}
	for _, ref := range refs {
		if seen[ref.ID()] {
			continue
		}
		seen[ref.ID()] = true
		fact, _, err := p.readAdvisoryEvidence(plan.RunID(), ref, identity, plan.Digest())
		if err != nil {
			view.Issue = "EVIDENCE_READ_FAILED"
			return view
		}
		if fact.ID() != ref.ID() || fact.Identity() != identity || fact.Digest() != plan.Digest() {
			return view
		}
		if fact.CheckID().String() != "dependency-advisory" || fact.Kind() != "advisory-record" {
			continue
		}
		if fact.ProfileDigest() != profile {
			return view
		}
		var record advisoryRecordView
		decoder := json.NewDecoder(bytes.NewBufferString(fact.Summary()))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&record) != nil || decoder.Decode(new(any)) != io.EOF || record.ID == "" || len(record.ID) > 192 || record.URL != "https://osv.dev/vulnerability/"+record.ID || record.ScoreKnown && (record.BaseScore < 0 || record.BaseScore > 10) {
			view.Issue = "ADVISORY_RECORD_UNSUPPORTED"
			return view
		}
		switch record.Impact {
		case "UNKNOWN", "UNAFFECTED", "AFFECTED", "WITHDRAWN":
		default:
			view.Issue = "ADVISORY_RECORD_UNSUPPORTED"
			return view
		}
		view.Records = append(view.Records, record)
	}
	for _, reason := range decision.Reasons() {
		if strings.HasPrefix(reason, "M13_ADVISORY_") && len(view.Records) == 0 {
			view.Issue = "ADVISORY_RECORDS_MISSING"
			return view
		}
	}
	view.Status, view.Issue = "AVAILABLE", ""
	return view
}

func (p *presentation) renderAdvisoryDetails(view *advisoryDetails) {
	if view == nil {
		return
	}
	if view.Subject.Name != "" {
		p.line("%s %s@%s", p.text("Advisory records:", "취약점 정보:"), safeTerminal(view.Subject.Name), safeTerminal(view.Subject.Version))
	} else {
		p.line("%s", p.text("Advisory records:", "취약점 정보:"))
	}
	if view.Status != "AVAILABLE" {
		p.line("  %s (%s)", p.text("Details incomplete", "상세 정보 미확인"), safeTerminal(view.Issue))
	}
	for n, record := range view.Records {
		if n == 5 {
			p.line("  %s", p.text("See JSON for remaining advisory records.", "나머지 취약점 정보는 JSON으로 확인하세요."))
			break
		}
		score := p.text("unknown severity", "심각도 미확인")
		if record.ScoreKnown {
			score = fmt.Sprintf("CVSS base %.1f", record.BaseScore)
		}
		p.line("  %s · %s · %s", safeTerminal(record.ID), safeTerminal(record.Impact), score)
		if record.FixedExample != "" {
			p.line("    %s %s", p.text("Example fixed boundary:", "수정 경계 예:"), safeTerminal(record.FixedExample))
		}
		p.line("    %s", safeTerminal(record.URL))
	}
}
