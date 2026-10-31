package cli

import (
	"encoding/json"
	"io"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type uvProjectLockReport struct {
	cargoUpdateReport
	LockUpdate   string `json:"lock_update_status"`
	Installation string `json:"installation_status"`
}

func writeUVProjectLockReport(output io.Writer, verified domain.ProjectVerifiedSet, cause error, records *machineVerifiedSet) error {
	facts := buildProjectMutationReport(verified, cause, "", "pypi", records, false)
	report := uvProjectLockReport{cargoUpdateReport: facts.report, LockUpdate: facts.report.Promotion, Installation: "NOT_REQUESTED"}
	report.Schema = "helox.project-lock-update-result/v1"
	report.Operation = "LOCK_UPDATE"
	p := presentationFor(output)
	if !p.human() {
		return json.NewEncoder(output).Encode(report)
	}
	p.interruption = interruptionCode(cause)
	policy := ""
	if report.Policy != nil {
		policy = report.Policy.Decision
	}
	p.reportHeader("UPDATE", report.Status, policy, "")
	p.line("%s %s", p.text("Lockfile update:", "잠금파일 업데이트:"), p.stateLabel(report.LockUpdate))
	p.line("%s", p.text("Package installation: not requested", "패키지 설치: 요청하지 않음"))
	if facts.inspected.Valid() {
		p.line("%s %d", p.text("Inspected Python distributions:", "검사한 Python 배포 파일 수:"), len(report.Entries))
		for index, entry := range report.Entries {
			if index == 64 {
				p.line("… %s", p.text("See JSON for remaining entries.", "나머지 항목은 JSON으로 확인하세요."))
				break
			}
			p.line("  %s@%s  %s", safeTerminal(entry.Identity.Name), safeTerminal(entry.Identity.Version), safeTerminal(entry.Policy.Decision))
			if p.verbose {
				p.line("    SHA256: %s | Run: %s", entry.Digest.Value, safeTerminal(entry.RunID))
				for _, ref := range entry.Evidence {
					p.line("    Evidence: %s", safeTerminal(ref.Handle))
				}
			}
		}
		p.renderChecks(facts.checks)
		p.renderReasons(facts.decision.Reasons())
	}
	if records != nil {
		p.line("Manifest: sha256:%s", records.ManifestID)
		p.line("SBOM: %s", safeTerminal(records.SBOMHandle))
	}
	if report.Error != nil {
		p.line("%s %s", p.text("Error:", "오류:"), safeTerminal(report.Error.Code))
		p.line("%s", p.text("The lock update is incomplete. Check the original project and retained recovery state before retrying.", "잠금파일 업데이트가 완료되지 않았습니다. 재실행 전에 원본 프로젝트와 보존된 복구 상태를 확인하세요."))
		p.reportedError = true
	}
	return p.writeErr
}
