package cli

import (
	"context"
	"errors"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const goUpgradeDescriptionEnglish = "Inspect the complete selected public module graph before updating dependency controls. Use get module@version for an exact version, or get -u module to upgrade one existing requirement and its imported dependencies.\n\nUse --adopt with -u to explicitly enroll supported native or changed controls. SDK and non-require directive changes are refused. Build behavior requires the separate helox go build inspection."
const goUpgradeDescriptionKorean = "선택된 전체 공개 모듈을 검사한 뒤 의존성 설정을 갱신합니다. 정확한 버전은 get 모듈@버전, 기존 의존성 하나와 그 의존성의 업데이트는 get -u 모듈을 사용하세요.\n\n지원하는 일반 프로젝트나 변경한 설정을 등록하려면 -u와 --adopt를 함께 명시하세요. SDK와 require 외 지시문의 변경은 거부합니다. 빌드 동작은 별도 helox go build에서 검사합니다."

type GoModuleUpgrader interface {
	Upgrade(context.Context, domain.InstallContext, string, bool) (domain.ProjectVerifiedSet, error)
}

func validateGoGetArgs(command *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(command, args); err != nil {
		return err
	}
	upgrade, _ := command.Flags().GetBool("upgrade")
	adopt, _ := command.Flags().GetBool("adopt")
	if upgrade {
		return artifactgo.ValidateUpgradeSelector(args[0])
	}
	if adopt {
		return errors.New("go get --adopt requires -u")
	}
	_, err := artifactgo.ParseReference(args[0])
	return err
}

// AddGoModuleUpgrade augments the existing exact-get binding without changing
// its response contract or sharing mutable enrollment intent across requests.
func AddGoModuleUpgrade(root *cobra.Command, upgrader GoModuleUpgrader) error {
	if root == nil || upgrader == nil {
		return errors.New("go upgrade requires command and guarded use case")
	}
	command := findLeaf(root, "go", "get")
	if command == nil || command.RunE == nil {
		return errors.New("go get command is not configured")
	}
	exactGet := command.RunE
	command.RunE = func(command *cobra.Command, args []string) error {
		if err := validateGoGetArgs(command, args); err != nil {
			return err
		}
		upgrade, _ := command.Flags().GetBool("upgrade")
		if !upgrade {
			return exactGet(command, args)
		}
		install, err := currentGoProjectContext()
		if err != nil {
			return err
		}
		adopt, _ := command.Flags().GetBool("adopt")
		ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finish() }()
		verified, operationErr := upgrader.Upgrade(ctx, install, args[0], adopt)
		if operationErr == nil && !verified.Valid() {
			operationErr = errors.New("go upgrade returned no completed project approval")
		}
		progressErr := finish()
		reportErr := writeProjectUpdateReport(command.OutOrStdout(), verified, operationErr, args[0], "go-proxy")
		return errors.Join(operationErr, progressErr, reportErr)
	}
	return nil
}
