package cli

import (
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/application"
	artifactgithub "github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const githubReplacementDescriptionEnglish = "Inspect and install one exact public GitHub Release asset into a new directory. Use --replace with an explicit --target to inspect and replace a proven HAA-managed installation in the same repository. Every replacement repeats all required checks, even for the same tag. Existing files without independent HAA approval cannot be adopted. Archives stay opaque; embedded programs are not executed."
const githubReplacementDescriptionKorean = "정확한 공개 GitHub Release 파일 하나를 검사한 뒤 새 디렉터리에 설치합니다. 기존 HAA 관리 설치를 같은 저장소의 파일로 교체하려면 --target과 --replace를 명시하세요. 태그가 같아도 모든 필수 검사를 새로 수행합니다. 독립 HAA 승인 기록이 없는 기존 파일은 인수하지 않습니다. Archive는 압축을 풀지 않고 보관하며 내부 프로그램은 실행하지 않습니다."

func validateGitHubReplacementIntent(root *cobra.Command, args []string) error {
	command, remaining, err := root.Find(args)
	if err != nil || command.CommandPath() != "helox github install" {
		return err
	}
	seen := false
	for i := 0; i < len(remaining); i++ {
		arg := remaining[i]
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "--replace=") || arg == "--replace" && seen {
			return errors.New("github install accepts --replace once without an explicit value")
		}
		if arg == "--replace" {
			seen = true
		}
		if strings.HasPrefix(arg, "--") && !strings.Contains(arg, "=") {
			flag := command.Flag(strings.TrimPrefix(arg, "--"))
			if flag != nil && flag.NoOptDefVal == "" && i+1 < len(remaining) {
				i++
			}
		}
	}
	return nil
}

func validateGitHubInstallArgs(command *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(command, args); err != nil {
		return err
	}
	if _, err := artifactgithub.ParseReference(args[0]); err != nil {
		return err
	}
	replace, _ := command.Flags().GetBool("replace")
	if replace {
		target, _ := command.Flags().GetString("target")
		if !command.Flags().Changed("target") || target == "" {
			return errors.New("github install --replace requires an explicit --target")
		}
		if _, err := domain.NewInstallTarget(target); err != nil {
			return err
		}
	}
	return nil
}

func AddGitHubReleaseReplacement(root *cobra.Command, updater Installer) error {
	if root == nil || updater == nil {
		return errors.New("github replacement requires command and guarded installer")
	}
	command := findLeaf(root, "github", "install")
	if command == nil || command.RunE == nil {
		return errors.New("github install must be bound before replacement")
	}
	installNew := command.RunE
	command.RunE = func(command *cobra.Command, args []string) error {
		if err := validateGitHubInstallArgs(command, args); err != nil {
			return err
		}
		replace, _ := command.Flags().GetBool("replace")
		if !replace {
			return installNew(command, args)
		}
		reference, err := artifactgithub.ParseReference(args[0])
		if err != nil {
			return err
		}
		target, _ := command.Flags().GetString("target")
		value, err := domain.NewInstallTarget(target)
		if err != nil {
			return err
		}
		install, err := domain.NewGitHubAssetInstallContext(value)
		if err != nil {
			return err
		}
		request, err := application.NewInstallUpdateRequest(reference.Source(), reference.Locator(), install, false)
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
