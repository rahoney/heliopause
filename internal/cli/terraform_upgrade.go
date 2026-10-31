package cli

import (
	"context"
	"errors"
	"strings"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const terraformUpgradeDescriptionEnglish = "Inspect and install exact public providers with init namespace/type@version. Use helox terraform init -upgrade to select the newest stable version of every required provider within the current configuration constraints.\n\nUse --adopt to explicitly enroll supported providerless native controls or changed managed controls. Unmanaged installed provider/backend/module state is refused. Every selected provider receives fresh signature, archive and isolated help-probe checks, even if versions do not change. Provider RPC, plan/apply and cloud behavior are outside this installation check."
const terraformUpgradeDescriptionKorean = "init namespace/type@version으로 정확한 공개 Provider를 검사한 뒤 설치합니다. 현재 설정의 버전 제약 안에서 모든 Provider를 최신 안정 버전으로 업데이트하려면 helox terraform init -upgrade를 사용하세요.\n\n아직 Provider가 설치되지 않은 일반 설정이나 변경한 관리 설정을 등록하려면 --adopt를 명시하세요. 소유권을 입증하지 못한 Provider·backend·module 설치는 거부합니다. 버전이 같아도 선택된 모든 Provider의 서명·archive와 격리된 help 실행을 새로 검사합니다. RPC·plan/apply·클라우드 동작은 이 설치 검사의 범위에 포함되지 않습니다."

type TerraformUpgrader interface {
	Upgrade(context.Context, domain.InstallContext, bool) (domain.ProjectVerifiedSet, error)
}

// Only this exact native spelling is normalized. Reconstructing the discovered
// command retains flag order/values, including globals before the command.
func normalizeTerraformInitArgs(root *cobra.Command, args []string) ([]string, error) {
	command, remaining, err := root.Find(args)
	if err != nil || command.CommandPath() != "helox terraform init" {
		return args, err
	}
	normalized := []string{"terraform", "init"}
	seen := false
	for i := 0; i < len(remaining); i++ {
		arg := remaining[i]
		if arg == "--" {
			normalized = append(normalized, remaining[i:]...)
			break
		}
		if arg == "-upgrade" || arg == "--upgrade" {
			if seen {
				return nil, errors.New("terraform init accepts -upgrade only once")
			}
			seen = true
			normalized = append(normalized, "--upgrade")
			continue
		}
		if strings.HasPrefix(arg, "-upgrade=") || strings.HasPrefix(arg, "--upgrade=") {
			return nil, errors.New("terraform init upgrade accepts no explicit value or false mode")
		}
		normalized = append(normalized, arg)
		if strings.HasPrefix(arg, "--") && !strings.Contains(arg, "=") {
			flag := command.Flag(strings.TrimPrefix(arg, "--"))
			if flag != nil && flag.NoOptDefVal == "" && i+1 < len(remaining) {
				i++
				normalized = append(normalized, remaining[i])
			}
		}
	}
	root.SetArgs(normalized)
	return normalized, nil
}

func validateTerraformInitArgs(command *cobra.Command, args []string) error {
	upgrade, _ := command.Flags().GetBool("upgrade")
	if upgrade {
		return cobra.NoArgs(command, args)
	}
	if command.Flags().Changed("adopt") {
		return errors.New("terraform init --adopt requires -upgrade")
	}
	if err := cobra.ExactArgs(1)(command, args); err != nil {
		return err
	}
	_, err := artifactterraform.ParseReference(args[0])
	return err
}

func AddTerraformUpgrade(root *cobra.Command, upgrader TerraformUpgrader) error {
	if root == nil || upgrader == nil {
		return errors.New("terraform upgrade requires command and guarded use case")
	}
	command := findLeaf(root, "terraform", "init")
	if command == nil || command.RunE == nil {
		return errors.New("terraform init command is not configured")
	}
	exactInit := command.RunE
	command.RunE = func(command *cobra.Command, args []string) error {
		if err := validateTerraformInitArgs(command, args); err != nil {
			return err
		}
		upgrade, _ := command.Flags().GetBool("upgrade")
		if !upgrade {
			return exactInit(command, args)
		}
		install, err := currentGoProjectContext()
		if err != nil {
			return err
		}
		adopt, _ := command.Flags().GetBool("adopt")
		ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finish() }()
		verified, operationErr := upgrader.Upgrade(ctx, install, adopt)
		if operationErr == nil && !verified.Valid() {
			operationErr = errors.New("terraform upgrade returned no completed project approval")
		}
		progressErr := finish()
		reportErr := writeProjectUpdateReport(command.OutOrStdout(), verified, operationErr, "", "terraform-registry")
		return errors.Join(operationErr, progressErr, reportErr)
	}
	return nil
}
