package cli

import (
	"errors"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/spf13/cobra"
)

const pypiUpgradeDescriptionEnglish = "Inspect and install a distribution into an existing Python 3.14 virtual environment.\n\nUse --upgrade or -U with one public PyPI bare name, exact version (name==version) or supported numeric constraint. Default strategy: only-if-needed. The complete selected dependency closure is inspected before replacement. Current upgrade wiring requires validated HAA ownership; native venv enrollment and named PyTorch upgrades are not yet qualified."
const pypiUpgradeDescriptionKorean = "기존 Python 3.14 가상환경에 패키지를 검사한 뒤 설치합니다.\n\n공개 PyPI 패키지 하나를 업데이트하려면 --upgrade 또는 -U를 사용하세요. 이름, 정확한 버전(name==version), 지원하는 숫자 버전 제약을 지정할 수 있습니다. 기본 전략은 only-if-needed이며, 선택된 전체 의존성을 검사한 뒤 교체합니다. 현재 업데이트 연결은 유효한 HAA 소유권 기록이 필요합니다. 일반 venv 등록과 PyTorch 출처의 업데이트 검증은 아직 완료되지 않았습니다."

func validatePyPIInstallArgs(command *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(command, args); err != nil {
		return err
	}
	upgrade, err := command.Flags().GetBool("upgrade")
	if err != nil {
		return err
	}
	adopt, err := command.Flags().GetBool("adopt")
	if err != nil {
		return err
	}
	if !upgrade {
		if adopt {
			return errors.New("pip --adopt requires --upgrade")
		}
		return nil
	}
	source, err := command.Flags().GetString("source")
	if err != nil {
		return err
	}
	if source != "" && source != "pypi" {
		return errors.New("guarded upgrades for named PyTorch sources are not yet supported")
	}
	if _, err := artifactpypi.ParseUpgradeRequirement(args[0]); err != nil {
		return err
	}
	target, err := command.Flags().GetString("target")
	if err != nil {
		return err
	}
	value, err := activePythonVenv(target)
	if err != nil {
		return err
	}
	if strings.Contains(value.String(), ",") {
		return errors.New("pip upgrade target cannot contain a comma")
	}
	return nil
}
