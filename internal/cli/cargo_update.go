package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const cargoUpdateDescriptionEnglish = "Inspect every selected public crate before updating the current project's existing Cargo.lock. Preserve manifest version ranges and features.\n\nOptionally select one package by name or name@locked-version. The version identifies the original locked package, not the proposed version. Use --adopt to explicitly enroll supported native or changed project controls. Build scripts and macros require the separate helox cargo build inspection."
const cargoUpdateDescriptionKorean = "현재 프로젝트의 기존 Cargo.lock에 선택된 모든 공개 crate를 검사한 뒤 업데이트합니다. Manifest의 버전 범위와 feature 설정은 유지합니다.\n\n-p로 패키지 이름 또는 이름@기존잠금버전을 하나 지정할 수 있습니다. 버전은 새 버전 고정이 아닌 기존 패키지 식별에 사용됩니다. 지원하는 일반 프로젝트나 변경한 설정을 등록하려면 --adopt를 명시하세요. Build script와 macro의 동작 검사는 별도 helox cargo build에서 수행합니다."

type CargoUpdater interface {
	Update(context.Context, domain.InstallContext, string, bool) (domain.ProjectVerifiedSet, error)
}

// Read the actual slice, since pflag GetStringArray round-trips through CSV
// and collapses one explicitly empty occurrence into an absent selector.
func cargoUpdateSelectors(command *cobra.Command) []string {
	return command.Flags().Lookup("package").Value.(interface{ GetSlice() []string }).GetSlice()
}

func validateCargoUpdateArgs(command *cobra.Command, args []string) error {
	if err := cobra.NoArgs(command, args); err != nil {
		return err
	}
	selectors := cargoUpdateSelectors(command)
	var err error
	if len(selectors) > 1 || (len(selectors) == 1 && selectors[0] == "") {
		return errors.New("cargo update accepts at most one nonempty -p/--package selector")
	}
	if len(selectors) == 1 {
		_, err = artifactcargo.ParseUpdateSelector(selectors[0])
	}
	return err
}

func AddCargoUpdate(root *cobra.Command, updater CargoUpdater) error {
	if root == nil || updater == nil {
		return errors.New("cargo update requires command and guarded use case")
	}
	command := findLeaf(root, "cargo", "update")
	if command == nil {
		return errors.New("cargo update command is not registered")
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		if err := validateCargoUpdateArgs(command, args); err != nil {
			return err
		}
		install, err := currentGoProjectContext()
		if err != nil {
			return err
		}
		selectors := cargoUpdateSelectors(command)
		selector := ""
		if len(selectors) == 1 {
			selector = selectors[0]
		}
		adopt, _ := command.Flags().GetBool("adopt")
		ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finish() }()
		verified, operationErr := updater.Update(ctx, install, selector, adopt)
		if operationErr == nil && !verified.Valid() {
			operationErr = errors.New("cargo update returned no completed project approval")
		}
		progressErr := finish()
		// Rendering does not turn a display failure into an unconfirmed rollback.
		reportErr := writeCargoUpdateReport(command.OutOrStdout(), verified, operationErr, selector)
		return errors.Join(operationErr, progressErr, reportErr)
	}
	return nil
}

type projectUpdateFacts interface {
	Inspected() domain.InspectedProjectSet
	Decision() domain.PolicyDecision
}

type cargoUpdateEntry struct {
	Identity machineIdentity            `json:"identity"`
	Digest   machineDigest              `json:"digest"`
	RunID    string                     `json:"run_id"`
	Checks   []machineCheck             `json:"checks"`
	Evidence []machineEvidenceReference `json:"evidence_references"`
	Policy   machinePolicy              `json:"policy"`
}

type cargoUpdateReport struct {
	Schema         string             `json:"schema_version"`
	Operation      string             `json:"operation"`
	Status         string             `json:"operation_status"`
	Promotion      string             `json:"promotion_status"`
	Source         string             `json:"source_id"`
	Selector       string             `json:"selector,omitempty"`
	Graph          string             `json:"graph_digest,omitempty"`
	DependencyFree bool               `json:"dependency_free"`
	Entries        []cargoUpdateEntry `json:"entries"`
	Policy         *machinePolicy     `json:"policy,omitempty"`
	Error          *machineError      `json:"error,omitempty"`
}

func projectUpdatePolicy(decision domain.PolicyDecision) machinePolicy {
	return machinePolicy{string(decision.Decision()), decision.PolicyID(), decision.Version(), decision.Reasons()}
}

func writeCargoUpdateReport(output io.Writer, verified domain.ProjectVerifiedSet, cause error, selector string) error {
	return writeProjectUpdateReport(output, verified, cause, selector, "crates-io")
}

func writeProjectUpdateReport(output io.Writer, verified domain.ProjectVerifiedSet, cause error, selector, source string) error {
	report := cargoUpdateReport{Schema: "helox.project-update-result/v1", Operation: "UPDATE", Status: "FAILED", Promotion: "NOT_COMPLETED", Source: source, Selector: selector, Entries: []cargoUpdateEntry{}}
	var inspected domain.InspectedProjectSet
	var decision domain.PolicyDecision
	if cause == nil && verified.Valid() {
		inspected, decision = verified.Inspected(), verified.Decision()
		report.Status, report.Promotion = "COMPLETED", "COMPLETED"
	} else {
		var facts projectUpdateFacts
		if errors.As(cause, &facts) {
			inspected, decision = facts.Inspected(), facts.Decision()
		}
		code := interruptionCode(cause)
		if code == "" {
			code = "PROJECT_UPDATE_NOT_COMPLETED"
		}
		report.Error = &machineError{Code: code, Message: "Project update was not confirmed complete."}
	}
	var checks []domain.CheckExecution
	if inspected.Valid() {
		report.Graph = inspected.Snapshot().GraphDigest().String()
		report.DependencyFree = inspected.Snapshot().DependencyFree()
		policy := projectUpdatePolicy(decision)
		report.Policy = &policy
		for _, inspection := range inspected.Inspections() {
			a := inspection.Artifact()
			id := a.Identity()
			entry := cargoUpdateEntry{Identity: machineIdentity{id.Source().String(), id.Name(), id.Version(), id.Variant()}, Digest: machineDigest{a.Digest().Algorithm(), a.Digest().String()}, RunID: inspection.RunID().String(), Checks: []machineCheck{}, Evidence: []machineEvidenceReference{}, Policy: projectUpdatePolicy(inspection.PolicyDecision())}
			for _, check := range inspection.Checks() {
				limitation, _ := check.LimitationCode()
				entry.Checks = append(entry.Checks, machineCheck{check.ID().String(), string(check.Kind()), check.Required(), string(check.Capability()), string(check.Status()), limitation})
				checks = append(checks, check)
			}
			for _, evidence := range inspection.Evidence() {
				entry.Evidence = append(entry.Evidence, machineEvidenceReference{evidence.ID().String(), evidence.Handle()})
			}
			report.Entries = append(report.Entries, entry)
		}
	}
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
	if source == "go-proxy" {
		p.line("%s %s", p.text("Dependency control update:", "의존성 설정 업데이트:"), p.stateLabel(report.Promotion))
	} else {
		p.line("%s %s", p.text("Lockfile update:", "잠금파일 업데이트:"), p.stateLabel(report.Promotion))
	}
	if selector != "" {
		p.line("%s %s", p.text("Selected package:", "선택한 패키지:"), safeTerminal(selector))
	}
	if inspected.Valid() {
		if source == "go-proxy" {
			p.line("%s %d", p.text("Inspected modules:", "검사한 모듈 수:"), len(report.Entries))
		} else {
			p.line("%s %d", p.text("Inspected crates:", "검사한 crate 수:"), len(report.Entries))
		}
		for index, entry := range report.Entries {
			if index == 64 {
				p.line("… %s", p.text("See JSON for remaining entries.", "나머지 항목은 JSON으로 확인하세요."))
				break
			}
			p.line("  %s@%s  %s", safeTerminal(entry.Identity.Name), safeTerminal(entry.Identity.Version), safeTerminal(entry.Policy.Decision))
			for _, check := range entry.Checks {
				if check.ExecutionStatus != "COMPLETED" || check.Capability != "SUPPORTED" {
					p.line("    %s: %s %s", safeTerminal(check.ID), safeTerminal(check.ExecutionStatus), safeTerminal(check.LimitationCode))
				}
			}
			if p.verbose {
				p.line("    SHA256: %s | Run: %s", entry.Digest.Value, entry.RunID)
				for _, e := range entry.Evidence {
					p.line("    Evidence: %s", safeTerminal(e.Handle))
				}
			}
		}
		p.renderChecks(checks)
		p.renderReasons(decision.Reasons())
	}
	if report.Error != nil {
		p.line("%s %s", p.text("Error:", "오류:"), report.Error.Code)
		p.line("%s", p.text("The update is incomplete. Check the original project and any recovery state before retrying.", "업데이트 완료가 확인되지 않았습니다. 재실행 전에 원본 프로젝트와 복구 상태를 확인하세요."))
		p.reportedError = true
	}
	if source == "go-proxy" {
		p.line("%s", p.text("Build behavior is inspected separately with helox go build.", "빌드 동작은 helox go build에서 별도로 검사합니다."))
	} else {
		p.line("%s", p.text("Build script and macro behavior is inspected separately with helox cargo build.", "Build script와 macro 동작은 helox cargo build에서 별도로 검사합니다."))
	}
	return p.writeErr
}
