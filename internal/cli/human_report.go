package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func (p *presentation) renderResult(result domain.OperationResult) error {
	p.reportedError = result.Status() == domain.OperationFailed
	decision, hasPolicy := result.PolicyDecision()
	policy := ""
	if hasPolicy {
		policy = string(decision.Decision())
	}
	p.reportHeader(string(result.Operation()), string(result.Status()), policy, "")
	reference := result.Reference()
	p.line("%s  %s / %s", p.text("Artifact:", "대상:"), safeTerminal(reference.Source().String()), safeTerminal(reference.Locator()))
	if identity, ok := result.ResolvedIdentity(); ok {
		p.line("%s  %s@%s (%s)", p.text("Resolved:", "확정 버전:"), safeTerminal(identity.Name()), safeTerminal(identity.Version()), safeTerminal(identity.Variant()))
	}
	p.renderChecks(result.Checks())
	if hasPolicy {
		p.renderReasons(decision.Reasons())
	}
	if opErr, ok := result.Error(); ok {
		p.line("%s %s — %s", p.text("Error:", "오류:"), safeTerminal(opErr.Code()), p.errorExplanation(opErr.Code()))
	}
	if runID, ok := result.RunID(); ok {
		p.line("Run: %s", runID.String())
	}
	p.line("Operation: %s", result.OperationID().String())
	if p.verbose {
		if digest, ok := result.Digest(); ok {
			p.line("SHA-256: %s", digest.String())
		}
		for _, ref := range result.Evidence() {
			p.line("Evidence: %s", safeTerminal(ref.Handle()))
		}
	}
	p.nextAction(string(result.Status()), policy, "")
	return p.writeErr
}

func (p *presentation) renderInstall(outcome application.InstallOutcome, operationErr error) error {
	p.reportedError = operationErr != nil
	inspection := outcome.Inspection()
	status, promotion := installStatuses(outcome, operationErr)
	decision := inspection.Decision()
	policy := string(decision.Decision())
	p.reportHeader(string(inspection.Request().Operation()), status, policy, promotion)
	reference := inspection.Request().Reference()
	p.line("%s %s / %s", p.text("Artifact:", "대상:"), safeTerminal(reference.Source().String()), safeTerminal(reference.Locator()))
	p.line("%s %s", p.text("Target:", "설치 위치:"), safeTerminal(inspection.Request().Context().Target().String()))
	var checks []domain.CheckExecution
	if inspection.Set().Valid() {
		items := inspection.Set().Inspections()
		p.line("%s %d", p.text("Inspected artifacts:", "검사한 패키지 수:"), len(items))
		for index, item := range items {
			checks = append(checks, item.Checks()...)
			if index < 8 {
				p.line("Run: %s", item.RunID().String())
			}
		}
	}
	p.renderChecks(checks)
	p.renderReasons(decision.Reasons())
	for index, item := range inspection.DependencyPolicyDiagnostics() {
		if index >= 16 {
			break
		}
		if item.Decision() != domain.DecisionAllow {
			p.line("%s %s@%s — %s", p.text("Dependency:", "의존 패키지:"), safeTerminal(item.Package()), safeTerminal(item.Version()), safeTerminal(string(item.Decision())))
			p.renderReasons(item.Reasons())
		}
	}
	for index, item := range inspection.GraphStaticDiagnostics() {
		if index >= 16 {
			break
		}
		p.line("%s %s@%s — %s / %s", p.text("Static detail:", "정적 검사 상세:"), safeTerminal(item.Package()), safeTerminal(item.Version()), safeTerminal(string(item.Cause())), safeTerminal(string(item.Stage())))
	}
	for index, item := range inspection.GraphDynamicInstallDiagnostics() {
		if index >= 16 {
			break
		}
		p.line("%s %s@%s — %s / %s", p.text("Sandbox detail:", "격리 검사 상세:"), safeTerminal(item.Package()), safeTerminal(item.Version()), safeTerminal(item.Reason()), safeTerminal(item.FailureClass()))
	}
	if operationErr != nil {
		explanation := p.errorExplanation(outcome.FailureCode())
		if inspection.Request().Source().String() == "github-release" {
			switch outcome.FailureCode() {
			case "UPDATE_OWNERSHIP_INVALID":
				explanation = p.text("Matching independent HAA approval, recorded Evidence and installed bytes are required. Choose a new managed target for unproven files; --replace cannot adopt them.", "독립 HAA 승인·Evidence·설치 파일이 일치해야 합니다. 소유 기록이 없는 파일은 새 관리 대상에 설치하세요. --replace로 기존 파일을 인수할 수 없습니다.")
			case "UPDATE_SOURCE_CHANGED":
				explanation = p.text("The repository selection or original target state changed. Replacement was stopped.", "저장소 선택 또는 원래 대상 상태가 달라져 교체를 중단했습니다.")
			}
		}
		p.line("%s %s — %s", p.text("Error:", "오류:"), safeTerminal(outcome.FailureCode()), explanation)
	}
	if inspection.Request().Source().String() == "github-release" {
		p.line("%s", p.text("Scope: one exact GitHub asset, declared digest and format checks. Archives stay intact; supported ELF files alone enter isolated dynamic inspection.", "검사 범위: 정확한 GitHub 파일 하나와 출처 선언 해시·파일 형식입니다. Archive는 압축을 풀지 않으며 지원하는 ELF만 격리된 동작 검사를 수행합니다."))
	}
	p.line("Operation: %s", inspection.OperationID().String())
	if p.verbose && outcome.Bundle().ManifestID().String() != "" {
		p.line("Manifest: %s", outcome.Bundle().ManifestID().String())
	}
	p.nextAction(status, policy, promotion)
	return p.writeErr
}
func (p *presentation) reportHeader(operation, status, policy, promotion string) {
	p.reported = true
	label := p.text("Inspection", "검사")
	if operation == "UPDATE" {
		label = p.text("Update", "업데이트")
	} else if operation == "INSTALL" {
		label = p.text("Installation", "설치")
	} else if strings.Contains(operation, "BUILD") {
		label = p.text("Build", "빌드")
	}
	title, color := p.text("INCOMPLETE", "미완료"), "33"
	switch {
	case status == "FAILED" || promotion == "FAILED":
		title, color = p.text("FAILED", "실패"), "31"
	case policy == "BLOCK":
		title, color = p.text("BLOCKED", "차단"), "31"
	case policy == "MANUAL_REVIEW":
		title, color = p.text("REVIEW REQUIRED", "검토 필요"), "33"
	case status == "COMPLETED" && policy == "ALLOW":
		title, color = p.text("COMPLETED", "완료"), "32"
	}
	if operation != "FINALIZATION" {
		if p.interruption == "OPERATION_CANCELLED" {
			title, color = p.text("Cancelled", "사용자 중단"), "33"
		}
		if p.interruption == "OPERATION_TIMED_OUT" {
			title, color = p.text("Time limit reached", "시간 제한 도달"), "33"
		}
	}
	p.line("")
	p.line("╭─ Heliopause · %s %s", label, p.paint(title, color))
	if p.interruption != "" {
		p.line("│ %s", p.interruption)
	}
	policyText := p.stateLabel(policy)
	if policy == "" {
		policyText = p.text("Not evaluated", "미판정")
	}
	p.line("│ %s: %s  |  %s: %s", p.text("Operation", "작업"), p.stateLabel(status), p.text("Policy", "정책"), policyText)
	if promotion != "" {
		p.line("│ %s: %s", p.text("Actual installation", "실제 설치"), p.stateLabel(promotion))
	}
	if !p.startedAt.IsZero() {
		p.line("│ %s: %.2fs", p.text("Elapsed", "소요 시간"), time.Since(p.startedAt).Seconds())
	}
	p.line("╰────────────────────────────────────────")
}
func (p *presentation) renderChecks(checks []domain.CheckExecution) {
	p.line("\n%s", p.text("Checks performed", "수행한 검사"))
	if len(checks) == 0 {
		p.line("  ! %s", p.text("No completed check results are available.", "완료된 검사 결과가 없습니다."))
		return
	}
	type row struct {
		check domain.CheckExecution
		count int
	}
	grouped := map[string]row{}
	for _, check := range checks {
		limitation, _ := check.LimitationCode()
		key := fmt.Sprintf("%s\x00%s\x00%s\x00%t\x00%s", check.ID(), check.Status(), check.Capability(), check.Required(), limitation)
		item := grouped[key]
		item.check = check
		item.count++
		grouped[key] = item
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for index, key := range keys {
		if index >= 64 {
			p.line("  … %s: %d", p.text("Further result rows", "추가 결과 항목"), len(keys)-64)
			break
		}
		item := grouped[key]
		check := item.check
		marker, color := "!", "33"
		if string(check.Status()) == "COMPLETED" && string(check.Capability()) == "SUPPORTED" {
			marker, color = "✓", "32"
		}
		if string(check.Status()) == "FAILED" {
			marker, color = "✗", "31"
		}
		required := p.text("optional", "선택")
		if check.Required() {
			required = p.text("required", "필수")
		}
		p.line("  %s %s — %s / %s [%s] ×%d", p.paint(marker, color), p.checkName(check.ID().String()), p.stateLabel(string(check.Status())), p.stateLabel(string(check.Capability())), required, item.count)
		if limitation, ok := check.LimitationCode(); ok {
			p.line("    %s %s", p.text("Limitation:", "검사 한계:"), safeTerminal(limitation))
		}
	}
}
func (p *presentation) checkName(id string) string {
	switch id {
	case "go-module-sumdb":
		return p.text("Independent SumDB integrity verification", "독립 SumDB 무결성 검증")
	case "npm-declared-integrity", "pypi-declared-sha256", "github-release-declared-sha256", "cargo-registry-integrity":
		return p.text("Declared integrity / digest", "출처 선언 무결성·해시 검증")
	case "npm-static-archive", "pypi-static-archive", "github-release-static", "cargo-crate-static", "go-module-static":
		return p.text("Archive / metadata static inspection", "압축파일·메타데이터 정적 검사")
	case "npm-dynamic-lifecycle", "pypi-dynamic-import", "github-release-elf-dynamic":
		return p.text("Isolated behavior inspection", "격리 환경 동작 검사")
	default:
		return safeTerminal(id)
	}
}
func (p *presentation) renderReasons(reasons []string) {
	for index, code := range reasons {
		if index >= 16 {
			break
		}
		p.line("%s %s (%s)", p.text("Reason:", "판정 근거:"), p.reason(code), safeTerminal(code))
	}
}
func (p *presentation) reason(code string) string {
	switch code {
	case "M1_REQUIRED_CHECKS_COMPLETED", "M3_REQUIRED_CHECKS_COMPLETED", "M6_REQUIRED_CHECKS_COMPLETED", "M4_VERIFIED_SET_COMPLETED", "M5_VERIFIED_SET_COMPLETED", "M6_VERIFIED_SET_COMPLETED":
		return p.text("Required checks completed and policy allowed this operation.", "필수 검사가 완료되어 정책이 이번 작업을 허용했습니다.")
	case "M1_REQUIRED_CHECK_INCOMPLETE", "M2_REQUIRED_CHECK_INCOMPLETE", "M3_REQUIRED_CHECK_INCOMPLETE", "M4_REQUIRED_CHECK_INCOMPLETE", "M5_REQUIRED_CHECK_INCOMPLETE", "M6_REQUIRED_CHECK_INCOMPLETE":
		return p.text("A required check was not completed; do not treat this as passed.", "필수 검사가 완료되지 않아 통과로 판단할 수 없습니다.")
	case "M2_INTEGRITY_MISMATCH", "M3_INTEGRITY_MISMATCH", "M6_INTEGRITY_MISMATCH":
		return p.text("Downloaded bytes do not match the expected integrity.", "다운로드한 파일이 기대한 무결성과 일치하지 않습니다.")
	case "M3_NETWORK_ATTEMPT":
		return p.text("The inspected code attempted a policy-relevant network connection.", "검사 대상 코드에서 정책상 주의가 필요한 네트워크 연결 시도가 관찰됐습니다.")
	case "M3_FILESYSTEM_VIOLATION":
		return p.text("A disallowed filesystem action was observed.", "허용되지 않은 파일 접근이 관찰됐습니다.")
	case "M3_HONEYTOKEN_ACCESS":
		return p.text("Access to a monitored decoy was observed.", "감시용 미끼 정보에 대한 접근이 관찰됐습니다.")
	case "M3_UNEXPECTED_PROCESS":
		return p.text("An unexpected process was observed.", "예상하지 않은 프로세스 실행이 관찰됐습니다.")
	case "M1_BLOCK_FINDING", "M4_DEPENDENCY_BLOCKED", "M5_DISTRIBUTION_BLOCKED":
		return p.text("A blocking finding exists in this artifact or its dependencies.", "이 패키지 또는 의존 패키지에서 차단 사유가 확인됐습니다.")
	case "M2_DYNAMIC_INSPECTION_UNAVAILABLE", "M4_HOST_LIFECYCLE_UNSUPPORTED", "M5_NON_WHEEL_PROMOTION_UNAVAILABLE", "M6_ELF_PLATFORM_UNSUPPORTED", "M6_DYNAMIC_INSPECTION_REQUIRED", "M6_FORMAT_NAME_MISMATCH", "M6_FORMAT_UNSUPPORTED":
		return p.text("This environment or artifact requires an unsupported capability.", "이 환경 또는 파일에 필요한 기능을 지원하지 않습니다.")
	case "M4_DEPENDENCY_REVIEW_REQUIRED", "M5_DISTRIBUTION_REVIEW_REQUIRED", "M6_RELEASE_REVIEW_REQUIRED":
		return p.text("The result requires review; automatic installation did not proceed.", "검토가 필요하여 자동 설치가 진행되지 않았습니다.")
	default:
		return p.text("See this stable policy reason code in the recorded result.", "기록된 결과의 정책 사유 코드를 확인하세요.")
	}
}
func (p *presentation) errorExplanation(code string) string {
	if p.interruption == "OPERATION_CANCELLED" {
		return p.text("The user cancelled this operation. This is not evidence of a broken runtime.", "사용자가 작업을 중단했습니다. 실행 환경 고장을 뜻하지 않습니다.")
	}
	if p.interruption == "OPERATION_TIMED_OUT" {
		return p.text("The operation reached its configured time bound.", "작업이 설정된 제한 시간에 도달했습니다.")
	}

	switch code {
	case "UPDATE_ADOPTION_REQUIRED":
		return p.text("This native project has no HAA ownership record. Use --adopt only to enroll a supported project after complete inspection.", "일반 프로젝트·가상환경에는 HAA 소유 기록이 없습니다. 지원하는 프로젝트를 전체 검사 후 등록하려면 --adopt를 명시하세요.")
	case "UPDATE_OWNERSHIP_INVALID":
		return p.text("Existing HAA ownership or metadata is inconsistent. Adoption cannot override it; check the original project state.", "기존 HAA 소유 기록 또는 메타데이터가 일치하지 않습니다. --adopt로 우회할 수 없으므로 원래 프로젝트 상태를 확인하세요.")
	case "UPDATE_ORIGINAL_STATE_UNSUPPORTED":
		return p.text("The existing dependency tree contains foreign or inconsistent files. They were not authorized for replacement.", "기존 의존성 폴더에 다른 파일이나 불일치가 있어 교체를 허용하지 않았습니다.")
	case "UPDATE_UNSUPPORTED_PROJECT":
		return p.text("This npm update supports one public direct dependency and its supported transitive graph. Legacy local-file controls need an explicit inspected install to establish version intent.", "현재 npm 업데이트는 공개 출처의 단일 직접 의존성과 지원하는 하위 의존성 그래프를 처리합니다. 이전 로컬 파일 참조는 명시적 검사·설치로 버전 선택 의도를 다시 지정해야 합니다.")
	case "UPDATE_SOURCE_CHANGED":
		return p.text("Original project files or the mutation lock changed during the update. The replacement was stopped.", "업데이트 도중 원래 프로젝트 파일 또는 변경 잠금이 바뀌어 교체를 중단했습니다.")
	case "UPDATE_PREPARATION_FAILED":
		return p.text("The existing project could not be prepared for a guarded update. Check its supported format, path and ownership.", "기존 프로젝트의 업데이트 준비에 실패했습니다. 지원 형식·경로·소유 상태를 확인하세요.")
	case "UPDATE_GUARD_CLOSE_FAILED":
		return p.text("Update guard cleanup failed. Confirm the target and ownership record before retrying; replacement may already have completed.", "업데이트 잠금 정리에 실패했습니다. 교체가 이미 끝났을 수 있으므로 대상 파일과 소유 기록을 확인한 뒤 재시도하세요.")
	case "ARTIFACT_RESOLVE_FAILED":
		return p.text("Exact version/dependency resolution failed; check source availability.", "정확한 버전·의존성 확인에 실패했습니다. 패키지 출처의 응답을 확인하세요.")
	case "ARTIFACT_ACQUIRE_FAILED":
		return p.text("Artifact download failed; check connectivity and source availability.", "파일 다운로드에 실패했습니다. 연결 상태와 패키지 출처의 응답을 확인하세요.")
	case "PROMOTION_FAILED", "STAGING_FAILED":
		return p.text("Verified-file installation failed; check target availability and permissions.", "검증 파일 설치에 실패했습니다. 대상 경로와 권한을 확인하세요.")
	default:
		return p.text("The operation did not complete. Check the reported checks, limitations and runtime readiness with helox doctor.", "작업이 완료되지 않았습니다. 위 검사·제한 사유와 helox doctor의 실행 환경 상태를 확인하세요.")
	}
}
func (p *presentation) nextAction(status, policy, promotion string) {
	if p.interruption != "" {
		p.line("%s", p.text("Retry only after the current operation and its cleanup have finished.", "현재 작업과 정리가 끝난 뒤 필요한 경우 다시 실행하세요."))
		return
	}

	switch {
	case status == "FAILED" || promotion == "FAILED":
		p.line("%s", p.text("Next: resolve the reported error before retrying.", "다음: 표시된 오류를 해결한 뒤 다시 실행하세요."))
	case policy == "BLOCK":
		p.line("%s", p.text("Next: review the blocking reason; this operation did not authorize installation.", "다음: 차단 근거를 확인하세요. 이 작업은 설치를 허용하지 않았습니다."))
	case policy == "MANUAL_REVIEW":
		p.line("%s", p.text("Next: review the reasons and limitations; automatic installation is withheld.", "다음: 사유와 검사 한계를 검토하세요. 자동 설치는 보류됐습니다."))
	case status == "COMPLETED" && policy == "ALLOW" && promotion == "":
		p.line("%s", p.text("Inspection completed. Inspect does not install files; ALLOW applies to this bounded inspection.", "검사가 완료됐습니다. inspect는 파일을 설치하지 않으며 ALLOW는 이번 검사 범위의 판정입니다."))
	case promotion == "COMPLETED":
		p.line("%s", p.text("Verified files were installed at the target shown above.", "검증한 파일을 위 설치 위치로 반입했습니다."))
	default:
		p.line("%s", p.text("The operation is incomplete; no successful installation is claimed.", "작업이 미완료 상태이며 설치 성공으로 판단할 수 없습니다."))
	}
}

func (p *presentation) stateLabel(state string) string {
	var label string
	switch state {
	case "COMPLETED":
		label = p.text("Completed", "완료")
	case "FAILED":
		label = p.text("Failed", "실패")
	case "INCOMPLETE":
		label = p.text("Incomplete", "미완료")
	case "NOT_PERFORMED", "NOT_EXECUTED":
		label = p.text("Not performed", "미수행")
	case "UNAVAILABLE":
		label = p.text("Unavailable", "사용 불가")
	case "SUPPORTED":
		label = p.text("Supported", "지원")
	case "UNSUPPORTED":
		label = p.text("Unsupported", "미지원")
	case "ALLOW":
		label = p.text("Allowed", "허용")
	case "BLOCK":
		label = p.text("Blocked", "차단")
	case "MANUAL_REVIEW":
		label = p.text("Review required", "검토 필요")
	default:
		return safeTerminal(state)
	}
	return label + " (" + state + ")"
}

func (p *presentation) renderProjectSummary(source, kind string, count int, digest string) error {
	p.reported = true
	p.line("\n╭─ Heliopause · %s", p.paint(p.text("Operation completed", "작업 완료"), "32"))
	p.line("│ %s: %s", p.text("Source", "출처"), safeTerminal(source))
	label := kind
	switch kind {
	case "Modules":
		label = p.text("Modules", "모듈")
	case "Crates":
		label = p.text("Crates", "Rust 패키지")
	case "Providers":
		label = p.text("Providers", "Provider")
	}
	p.line("│ %s: %d", label, count)
	p.line("│ %s: %s", p.text("Control digest", "입력·제어 파일 해시"), safeTerminal(digest))
	p.line("╰────────────────────────────────────────")
	return p.writeErr
}
