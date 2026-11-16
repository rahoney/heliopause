package cli

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const cargoFetchDescriptionEnglish = "Quarantine the complete public crates.io lock set into the Heliopause cache. A missing Cargo.lock is generated; an existing lock is preserved and mismatches are refused. This prepares dependencies only. Use helox cargo build for observed build scripts, procedural macros and project compilation."
const cargoFetchDescriptionKorean = "현재 Cargo.lock의 전체 공개 crates.io 의존성을 검사한 뒤 Heliopause 캐시에 준비합니다. Lock이 없으면 생성하며 기존 lock은 보존하고 불일치는 거부합니다. 이 명령은 의존성 준비만 수행합니다. 빌드 스크립트·프로시저 매크로·프로젝트 컴파일 관찰은 helox cargo build를 사용하세요."

type CargoFetcher interface {
	Fetch(context.Context, domain.InstallContext) (domain.ProjectVerifiedSet, error)
}

// AddCargoFetch binds preparation to the existing project inspection/commit
// owner. It neither executes the project nor forwards native manager options.
func AddCargoFetch(root *cobra.Command, fetcher CargoFetcher) error {
	if root == nil || fetcher == nil {
		return errors.New("cargo fetch requires a guarded preparation use case")
	}
	command := findLeaf(root, "cargo", "fetch")
	if command == nil {
		return errors.New("cargo fetch command is not registered")
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		if err := cobra.NoArgs(command, args); err != nil {
			return err
		}
		install, err := currentGoProjectContext()
		if err != nil {
			return err
		}
		ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		verified, cause := fetcher.Fetch(ctx, install)
		progressErr := finish()
		facts := buildProjectMutationReport(verified, cause, "", "crates-io", nil, false)
		facts.report.Schema = "helox.project-fetch-result/v1"
		facts.report.Operation = "FETCH"
		facts.report.Scope = "DEPENDENCY_PREPARATION"
		if facts.report.Error != nil && facts.report.Error.Code == "PROJECT_UPDATE_NOT_COMPLETED" {
			facts.report.Error.Code = "PROJECT_FETCH_NOT_COMPLETED"
			facts.report.Error.Message = "Dependency preparation was not confirmed complete."
		}
		return errors.Join(cause, progressErr, writeProjectMutationFacts(command.OutOrStdout(), facts, cause))
	}
	return nil
}
