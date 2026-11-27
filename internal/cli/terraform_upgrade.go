package cli

import (
	"context"
	"errors"
	"strings"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const terraformUpgradeDescriptionEnglish = "Run helox terraform init to inspect and install all required public providers. An existing complete lock is preserved; an absent lock permits first selection. Lock/configuration mismatch is refused. Use init namespace/type@version for an exact provider. Use helox terraform init -upgrade to select the newest stable version of every required provider within the current configuration constraints.\n\nUse --adopt to explicitly enroll supported providerless native controls or changed managed controls. Unmanaged installed provider/backend/module state is refused. Every selected provider receives fresh signature, archive and isolated help-probe checks, even if versions do not change. Provider RPC, plan/apply and cloud behavior are outside this installation check."
const terraformUpgradeDescriptionKorean = "helox terraform init은 필요한 모든 공개 Provider를 검사한 뒤 설치합니다. 기존 전체 lock은 보존하고 lock이 없을 때만 최초 선택을 만듭니다. Lock과 설정이 불일치하면 거부합니다. init namespace/type@version은 정확한 Provider를 지정합니다. 현재 설정의 버전 제약 안에서 모든 Provider를 최신 안정 버전으로 업데이트하려면 helox terraform init -upgrade를 사용하세요.\n\n아직 Provider가 설치되지 않은 일반 설정이나 변경한 관리 설정을 등록하려면 --adopt를 명시하세요. 소유권을 입증하지 못한 Provider·backend·module 설치는 거부합니다. 버전이 같아도 선택된 모든 Provider의 서명·archive와 격리된 help 실행을 새로 검사합니다. RPC·plan/apply·클라우드 동작은 이 설치 검사의 범위에 포함되지 않습니다."

type TerraformUpgrader interface {
	Upgrade(context.Context, domain.InstallContext, bool) (domain.ProjectVerifiedSet, error)
	InitProject(context.Context, domain.InstallContext) (domain.ProjectVerifiedSet, error)
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
	if err := cobra.MaximumNArgs(1)(command, args); err != nil {
		return err
	}
	if len(args) == 0 {
		return nil
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
		if !upgrade && len(args) != 0 {
			return exactInit(command, args)
		}
		install, err := currentGoProjectContext()
		if err != nil {
			return err
		}
		adopt, _ := command.Flags().GetBool("adopt")
		ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finish() }()
		var verified domain.ProjectVerifiedSet
		var operationErr error
		if upgrade {
			verified, operationErr = upgrader.Upgrade(ctx, install, adopt)
		} else {
			verified, operationErr = upgrader.InitProject(ctx, install)
		}
		if operationErr == nil && !verified.Valid() {
			operationErr = errors.New("terraform preparation returned no completed project approval")
		}
		progressErr := finish()
		reportErr := writeProjectMutationReportWithRecords(command.OutOrStdout(), verified, operationErr, "", "terraform-registry", nil, !upgrade)
		return errors.Join(operationErr, progressErr, reportErr)
	}
	return nil
}
