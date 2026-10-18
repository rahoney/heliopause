package cli

import (
	"errors"
	"os"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const npmUpdateDescriptionEnglish = "Inspect the complete supported dependency graph before updating an npm project. Preserve its package.json version constraints.\n\nSupports a public-registry project with one direct dependency and its supported transitive graph. Use --adopt explicitly for supported native state. Existing local-file controls require an explicit inspected install to establish version intent."
const npmUpdateDescriptionKorean = "npm 프로젝트의 버전 제약을 유지하며 새 버전과 지원하는 전체 의존성을 검사한 뒤 업데이트합니다.\n\n공개 registry의 단일 직접 의존성과 지원하는 하위 의존성 그래프를 처리합니다. 일반 npm 프로젝트를 등록하려면 --adopt를 명시하세요. 이전 로컬 파일 참조는 명시적 검사·설치로 버전 선택 의도를 다시 지정해야 합니다."

func validateNPMUpdateArgs(command *cobra.Command, args []string) error {
	if err := cobra.MaximumNArgs(1)(command, args); err != nil {
		return err
	}
	if len(args) != 0 {
		reference, err := artifactnpm.ParseReference(args[0])
		if err != nil {
			return err
		}
		name, err := artifactnpm.RequestedPackageName(reference)
		if err != nil || name != args[0] {
			return errors.New("npm update accepts a package name; existing version constraints are preserved")
		}
	}
	target, err := command.Flags().GetString("target")
	if err != nil {
		return err
	}
	if target != "" {
		_, err = domain.NewInstallTarget(target)
	}
	return err
}

func AddNPMUpdate(root *cobra.Command, updater Installer) error {
	if root == nil || updater == nil {
		return errors.New("npm update requires command and guarded use case")
	}
	command := findLeaf(root, "npm", "update")
	if command == nil {
		return errors.New("npm update command is not registered")
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		if err := validateNPMUpdateArgs(command, args); err != nil {
			return err
		}
		selector := ""
		if len(args) != 0 {
			selector = args[0]
		}
		target, err := command.Flags().GetString("target")
		if err != nil {
			return err
		}
		if target == "" {
			target, err = os.Getwd()
			if err != nil {
				return errors.New("resolve npm update project directory")
			}
		}
		value, err := domain.NewInstallTarget(target)
		if err != nil {
			return err
		}
		install, err := domain.NewNPMProjectInstallContext(value)
		if err != nil {
			return err
		}
		adopt, err := command.Flags().GetBool("adopt")
		if err != nil {
			return err
		}
		source, _ := domain.NewSourceID("npm")
		request, err := application.NewInstallUpdateRequest(source, selector, install, adopt)
		if err != nil {
			return err
		}
		code, err := ExecuteInstall(contextOrBackground(command.Context()), updater, request, machineOutput(command), command.OutOrStdout())
		if err != nil {
			return err
		}
		if code != 0 {
			return ExitError{Code: code}
		}
		return nil
	}
	return nil
}
