package cli

import (
	"errors"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/spf13/cobra"
)

const pypiUpgradeDescriptionEnglish = "Inspect and install a distribution into an existing Python 3.14 virtual environment.\n\nUse --upgrade or -U with one bare name, exact version (name==version) or supported numeric constraint. Default source: public PyPI; select official PyTorch explicitly with --source pytorch:cpu, pytorch:cu126, pytorch:cu130 or pytorch:cu132. Named local versions are preserved. Default strategy: only-if-needed. The complete selected dependency closure is freshly inspected before replacement, even for an unchanged version. Existing HAA ownership is validated. For native pip installations, explicitly use --adopt: supported public wheel files are compared with the authentic old wheel before replacement. Native PyTorch files, console scripts, cross-scheme files and nonstandard bytecode caches are unsupported."
const pypiUpgradeDescriptionKorean = "기존 Python 3.14 가상환경에 패키지를 검사한 뒤 설치합니다.\n\n패키지 하나를 업데이트하려면 --upgrade 또는 -U를 사용하세요. 이름, 정확한 버전(name==version), 지원하는 숫자 버전 제약을 지정할 수 있습니다. 기본 출처는 공개 PyPI입니다. 공식 PyTorch는 --source pytorch:cpu, pytorch:cu126, pytorch:cu130 또는 pytorch:cu132로 명시하며 local version을 보존합니다. 기본 전략은 only-if-needed이며 버전이 같아도 선택된 전체 의존성을 새로 검사한 뒤 교체합니다. 기존 HAA 소유권 기록을 검증합니다. 일반 pip 설치를 인계받으려면 --adopt를 명시하세요. 지원하는 공개 wheel 파일은 공식 이전 wheel의 원본과 대조한 뒤 교체합니다. 일반 설치의 PyTorch 파일, console script, 다른 scheme 파일, 비표준 bytecode 캐시는 지원하지 않습니다."

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
	profile := artifactpypi.PublicPyPIProfile()
	if source != "" && source != "pypi" {
		var ok bool
		profile, ok = artifactpypi.PyTorchProfile(strings.TrimPrefix(source, "pytorch:"))
		if !strings.HasPrefix(source, "pytorch:") || !ok {
			return errors.New("unsupported Python source profile")
		}
	}
	if _, err := artifactpypi.ParseUpgradeRequirementForSource(args[0], profile.Source()); err != nil {
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
