package cli

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const goTidyDescriptionEnglish = "Freeze the current module's source and use the pinned Go tool to reconcile imports, tests and build tags. Inspect the complete selected public graph before atomically updating go.mod and go.sum. Use --adopt to enroll supported native or changed controls. Workspaces, nested modules, replace directives and SDK changes are refused. This does not run project code; use helox go build for observed compilation."
const goTidyDescriptionKorean = "현재 모듈 소스를 고정하고 고정 Go 도구로 import·테스트·빌드 조건에 맞춰 의존성을 정리합니다. 선택된 전체 공개 graph를 검사한 뒤 go.mod와 go.sum을 원자 반영합니다. 지원하는 일반 프로젝트나 변경한 설정은 --adopt로 명시 등록하세요. Workspace·중첩 모듈·replace·SDK 변경은 거부합니다. 프로젝트 코드는 실행하지 않으며 컴파일 관찰은 helox go build를 사용하세요."

type GoModuleTidier interface {
	Tidy(context.Context, domain.InstallContext, bool) (domain.ProjectVerifiedSet, error)
}

func AddGoModuleTidy(root *cobra.Command, tidier GoModuleTidier) error {
	if root == nil || tidier == nil {
		return errors.New("go tidy requires a guarded source use case")
	}
	command := findLeaf(findRootCommand(root, "go"), "mod", "tidy")
	if command == nil {
		return errors.New("go tidy command is not registered")
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		if err := cobra.NoArgs(command, args); err != nil {
			return err
		}
		install, err := currentGoProjectContext()
		if err != nil {
			return err
		}
		adopt, _ := command.Flags().GetBool("adopt")
		ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finish() }()
		verified, cause := tidier.Tidy(ctx, install, adopt)
		if cause == nil && !verified.Valid() {
			cause = errors.New("go tidy returned no completed project approval")
		}
		progressErr := finish()
		facts := buildProjectMutationReport(verified, cause, "", "go-proxy", nil, false)
		facts.report.Schema = "helox.project-tidy-result/v1"
		facts.report.Operation = "TIDY"
		facts.report.Scope = "DEPENDENCY_CONTROLS"
		return errors.Join(cause, progressErr, writeProjectMutationFacts(command.OutOrStdout(), facts, cause))
	}
	return nil
}
