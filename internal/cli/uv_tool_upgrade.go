package cli

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

const uvToolUpgradeEnglish = "Quarantine the complete public PyPI wheel set selected by uv tool upgrade, then replace a tool and its exposed commands. Every selected wheel receives fresh inspection and approval, including unchanged versions.\n\nSpecify --target as the existing virtual environment and --bin-dir as its existing separate command directory. The target directory must be the normalized package name. Existing tools require HAA ownership or explicit --adopt of the complete original native UV public-wheel installation after independent old-wheel verification. Adoption supports the fixed --no-build --no-sources --link-mode copy recipe; ordinary native defaults and other recipes are unsupported.\n\nUse one public name, exact version or supported numeric version condition. UV retains the original installation requirement and adds the condition from this command; an exact original version therefore stays pinned. Supports Linux amd64. Runtime provisioning, additional native options and other indexes are unsupported."
const uvToolUpgradeKorean = "uv tool upgrade가 선택한 공개 PyPI wheel과 의존성 전체를 검역한 뒤, 툴과 노출 명령을 교체합니다. 같은 버전을 선택해도 전체를 새로 검사하고 승인받습니다.\n\n--target에는 기존 가상환경을, --bin-dir에는 기존의 별도 명령 디렉터리를 지정하세요. Target의 마지막 디렉터리는 정규화된 패키지명이어야 합니다. 기존 툴은 HAA 소유권이 필요하며, 일반 UV 설치는 --adopt로 원본 전체를 공식 이전 wheel과 독립 대조한 뒤 명시적으로 인계받습니다. 인계는 --no-build --no-sources --link-mode copy로 설치한 공개 wheel 형식에 한정하며 일반 native 기본값이나 다른 설치 형식은 지원하지 않습니다.\n\n공개 패키지명·정확한 버전·지원하는 숫자 조건을 사용할 수 있습니다. 원래 설치 조건을 유지하며 이번 명령의 조건을 추가하므로, 정확한 버전으로 설치한 툴은 해당 버전을 유지합니다. Linux amd64를 지원합니다. Runtime 준비·추가 native 옵션·다른 index는 지원하지 않습니다."

func uvToolUpgradeDescriptor() packageClientDescriptor {
	d := uvToolInstallDescriptor()
	d.path, d.use = []string{"uv", "tool", "upgrade"}, "upgrade <requirement>"
	d.shortEN, d.shortKO = "Quarantine and upgrade a public Python tool", "공개 Python 툴 전체를 검역한 뒤 갱신합니다"
	d.longEN, d.longKO = uvToolUpgradeEnglish, uvToolUpgradeKorean
	d.validate = func(command *cobra.Command, args []string) error { return validateUVToolArgs(command, args, "upgrade") }
	d.flags = configureUVToolUpgradeFlags
	d.validateRaw = validateUVToolUpgradeRawArgs
	d.upgrade = func(*cobra.Command, []string) (packageUpgradeMode, error) { return packageUpgradeClosure, nil }
	d.flagText = map[string]packageClientParent{
		"target":  {"Existing tool virtual environment (absolute path)", "기존 툴 가상환경의 절대 경로"},
		"bin-dir": uvToolInstallFlagText["bin-dir"],
		"adopt":   {"Explicitly enroll the complete supported native UV tool after independent old-wheel verification", "공식 이전 wheel과 원본 전체를 독립 대조한 뒤 지원하는 일반 UV 툴을 등록합니다"},
	}
	d.examples = []string{"helox uv tool upgrade cowsay --target /absolute/path/cowsay --bin-dir /absolute/path/bin"}
	return d
}

func configureUVToolUpgradeFlags(command *cobra.Command) {
	configureUVToolInstallFlags(command)
	command.Flags().Bool("adopt", false, "Explicitly enroll the complete supported native UV tool after independent old-wheel verification")
}

func validateUVToolUpgradeRawArgs(args []string) error {
	if err := validateUVToolInstallRawArgs(args); err != nil {
		return err
	}
	count := 0
	for _, arg := range args {
		name, _, _ := strings.Cut(arg, "=")
		if name == "--adopt" {
			count++
		}
	}
	if count > 1 {
		return errors.New("UV tool adoption flag must not repeat")
	}
	return nil
}

func UVToolUpgradeCommand(root *cobra.Command) (*cobra.Command, error) {
	return packageClientCommand(root, UVToolPackageClient, "upgrade")
}

func AddUVToolUpgrade(root *cobra.Command, updater Installer) error {
	if updater == nil {
		return errors.New("UV tool upgrade requires a configured updater")
	}
	return addPackageClientOperation(root, UVToolPackageClient, "upgrade", map[string]Installer{"pypi": updater}, map[packageUpgradeMode]map[string]Installer{packageUpgradeClosure: {"pypi": updater}})
}
