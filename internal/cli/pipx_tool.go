package cli

import (
	"errors"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/spf13/cobra"
)

const PipxPackageClient PackageClientID = "pipx"

const pipxInstallEnglish = "Quarantine the complete public PyPI wheel set selected by the pinned pipx and UV backend, then install a tool and its commands. Every selected wheel receives current inspection and approval.\n\nSpecify --target as an existing empty virtual environment for the pinned Python version and --bin-dir as a separate existing command directory. The target directory must be named after the normalized package. Use a public name, exact name==version (or name@version), or a supported numeric version condition. Supports Linux amd64. Runtime provisioning and additional pipx options are unsupported."
const pipxInstallKorean = "고정된 pipx와 UV backend가 선택한 공개 PyPI wheel·의존성 전체를 검사하고 승인한 뒤 툴과 명령을 설치합니다.\n\n--target에 고정 Python 버전의 기존 빈 가상환경, --bin-dir에 별도 기존 명령 디렉터리를 지정하세요. Target의 마지막 디렉터리는 정규화된 패키지명이어야 합니다. 공개 패키지명·정확한 name==version(name@version도 가능)·지원하는 숫자 조건을 사용할 수 있습니다. Linux amd64를 지원하며 runtime 준비와 추가 pipx 옵션은 제공하지 않습니다."
const pipxUpgradeEnglish = "Quarantine the complete public wheel set selected by pipx upgrade, then atomically replace an HAA-managed tool and its commands. Unchanged versions also receive new inspection and approval.\n\nUse one canonical bare package name with explicit --target and --bin-dir. Pipx upgrades independently of the old installation version constraint. Existing complete HAA ownership is required, or use --adopt for the complete native public UV-backend installation with the fixed --no-build --no-sources --link-mode copy recipe after independent old-wheel verification. Native defaults, other recipes, runtime provisioning, other indexes and additional pipx options are unsupported. Supports Linux amd64."
const pipxUpgradeKorean = "pipx upgrade가 선택한 공개 wheel 전체를 새로 검사·승인한 뒤 HAA가 관리하는 툴과 명령을 원자로 교체합니다. 같은 버전도 새로 검사합니다.\n\n정규화된 패키지 이름 하나와 명시적 --target·--bin-dir를 지정하세요. Pipx는 원래 설치의 버전 조건과 별도로 갱신 대상을 선택합니다. 기존 파일 전체의 HAA 소유 기록이 필요합니다. 일반 UV backend pipx 설치는 --no-build --no-sources --link-mode copy 형식에 한해 --adopt로 공식 이전 wheel과 원본 전체를 독립 대조한 뒤 인계받습니다. Native 기본값·다른 설치 형식·runtime 준비·다른 index·추가 pipx 옵션은 제공하지 않습니다. Linux amd64를 지원합니다."

func pipxInstallDescriptor() packageClientDescriptor {
	d := uvToolInstallDescriptor()
	d.id, d.path, d.use = PipxPackageClient, []string{"pipx", "install"}, "install <requirement>"
	d.parents = []packageClientParent{{"Quarantine public Python tools with pipx", "pipx로 공개 Python 툴을 검역합니다"}}
	d.shortEN, d.shortKO = "Quarantine and install a public Python tool with pipx", "pipx로 공개 Python 툴 전체를 검역한 뒤 설치합니다"
	d.longEN, d.longKO = pipxInstallEnglish, pipxInstallKorean
	d.validate = func(command *cobra.Command, args []string) error {
		return validateToolPathArgs(command, args, "pipx install")
	}
	d.validateRaw = func(args []string) error { return validateToolPathRawArgs(args, "pipx install") }
	d.examples = []string{"helox pipx install cowsay==6.1 --target /absolute/path/cowsay --bin-dir /absolute/path/bin"}
	return d
}
func pipxUpgradeDescriptor() packageClientDescriptor {
	d := pipxInstallDescriptor()
	d.path, d.use = []string{"pipx", "upgrade"}, "upgrade <name>"
	d.shortEN, d.shortKO = "Quarantine and upgrade an HAA-managed pipx tool", "HAA가 관리하는 pipx 툴 전체를 검역한 뒤 갱신합니다"
	d.longEN, d.longKO = pipxUpgradeEnglish, pipxUpgradeKorean
	d.validate = func(command *cobra.Command, args []string) error {
		if err := validateToolPathArgs(command, args, "pipx upgrade"); err != nil {
			return err
		}
		requirement, err := artifactpypi.ParseUpgradeRequirement(args[0])
		if err != nil || requirement.Requirement() != requirement.Project() || args[0] != requirement.Project() {
			return errors.New("pipx upgrade requires one canonical bare package name")
		}
		return nil
	}
	d.validateRaw = func(args []string) error {
		if err := validateToolPathRawArgs(args, "pipx upgrade"); err != nil {
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
			return errors.New("pipx adoption flag must not repeat")
		}
		return nil
	}
	d.flags = func(command *cobra.Command) {
		configureUVToolInstallFlags(command)
		command.Flags().Bool("adopt", false, "Authenticate the complete supported native UV-backend pipx original before current approval")
	}
	d.upgrade = func(*cobra.Command, []string) (packageUpgradeMode, error) { return packageUpgradeClosure, nil }
	d.flagText = map[string]packageClientParent{"target": {"Existing HAA-managed tool virtual environment (absolute path)", "HAA가 관리하는 기존 툴 가상환경의 절대 경로"}, "bin-dir": uvToolInstallFlagText["bin-dir"], "adopt": {"Independently authenticate the supported native UV-backend original before current approval", "지원하는 UV backend native 원본 전체를 독립 대조한 뒤 현재 검사를 진행합니다"}}
	d.examples = []string{"helox pipx upgrade cowsay --target /absolute/path/cowsay --bin-dir /absolute/path/bin"}
	return d
}
func PipxToolUpgradeCommand(root *cobra.Command) (*cobra.Command, error) {
	return packageClientCommand(root, PipxPackageClient, "upgrade")
}
func AddPipxToolInstall(root *cobra.Command, installer Installer) error {
	if installer == nil {
		return errors.New("pipx install requires a configured installer")
	}
	return addPackageClientInstall(root, PipxPackageClient, map[string]Installer{"pypi": installer}, nil)
}
func AddPipxToolUpgrade(root *cobra.Command, updater Installer) error {
	if updater == nil {
		return errors.New("pipx upgrade requires a configured updater")
	}
	return addPackageClientOperation(root, PipxPackageClient, "upgrade", map[string]Installer{"pypi": updater}, map[packageUpgradeMode]map[string]Installer{packageUpgradeClosure: {"pypi": updater}})
}
