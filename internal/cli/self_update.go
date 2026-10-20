package cli

import (
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/application"
	artifactgithub "github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const selfUpdateDescriptionEnglish = "Quarantine one exact official Heliopause release with the currently installed trusted controller, then activate only the complete allowed set. Requires a healthy managed native installation on Linux/WSL2 amd64 with the same runtime lock. Development and OS-package installations are not adopted. Administrator authentication may be prepared with sudo -v; the CLI itself runs without sudo. A transition or recovery error can mean the installed version already changed: resolve recovery before retrying."
const selfUpdateDescriptionKorean = "현재 설치된 신뢰된 Heliopause로 정확한 공식 릴리스 하나를 검역한 뒤, 전체 검사가 허용한 세트만 활성화합니다. 동일한 runtime lock을 사용하는 Linux/WSL2 amd64의 정상 native 관리 설치가 필요합니다. 개발용·OS 패키지 설치는 자동 인수하지 않습니다. 필요하면 sudo -v로 관리자 인증을 먼저 받으세요. CLI 자체는 sudo 없이 실행합니다. 전환·복구 오류에서는 설치 버전이 이미 바뀌었을 수도 있으므로 복구 상태를 해결한 뒤 다시 실행하세요."

func newSelfUpdateCommand() *cobra.Command {
	command := &cobra.Command{Use: "self-update", Short: "Quarantine and update Heliopause to one exact official version", Long: selfUpdateDescriptionEnglish, Args: validateSelfUpdateArgs, RunE: func(*cobra.Command, []string) error { return errors.New("self-update command is not configured") }}
	command.Flags().String("version", "", "Exact official Heliopause version (vX.Y.Z)")
	return command
}

func validateSelfUpdateArgs(command *cobra.Command, args []string) error {
	if err := cobra.NoArgs(command, args); err != nil {
		return err
	}
	version, err := command.Flags().GetString("version")
	if err != nil {
		return err
	}
	return artifactgithub.ValidateHeliopauseVersion(version)
}

func validateSelfUpdateIntent(root *cobra.Command, args []string) error {
	command, remaining, err := root.Find(args)
	if err != nil || command.CommandPath() != "helox self-update" {
		return err
	}
	count := 0
	for i := 0; i < len(remaining); i++ {
		arg := remaining[i]
		if arg == "--" {
			break
		}
		if arg == "--version" || strings.HasPrefix(arg, "--version=") {
			count++
			if arg == "--version" {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "--") && !strings.Contains(arg, "=") {
			if flag := command.Flag(strings.TrimPrefix(arg, "--")); flag != nil && flag.NoOptDefVal == "" {
				i++
			}
		}
	}
	if count > 1 {
		return errors.New("self-update accepts --version exactly once")
	}
	return nil
}

func AddSelfUpdate(root *cobra.Command, updater Installer) error {
	if root == nil || updater == nil {
		return errors.New("self-update requires the guarded complete-set updater")
	}
	command := findRootCommand(root, "self-update")
	if command == nil {
		return errors.New("self-update command is not registered")
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		if err := validateSelfUpdateArgs(command, args); err != nil {
			return err
		}
		version, _ := command.Flags().GetString("version")
		source, _ := domain.NewSourceID("github-release")
		target, _ := domain.NewInstallTarget("/usr/lib/heliopause")
		install, _ := domain.NewGitHubAssetInstallContext(target)
		request, err := application.NewInstallUpdateRequest(source, version, install, false)
		if err != nil {
			return err
		}
		presentationFor(command.OutOrStdout()).selfUpdate = true
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
