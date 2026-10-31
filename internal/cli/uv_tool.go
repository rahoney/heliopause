package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const UVToolPackageClient PackageClientID = "uv-tool"

const uvToolInstallEnglish = "Quarantine the complete public PyPI wheel set selected by uv tool install, then install the tool into an existing empty virtual environment for the pinned Python version and expose its commands. Every selected wheel receives current inspection and approval.\n\nSpecify --target as an existing absolute virtual-environment path whose final directory is the normalized package name, and --bin-dir as an existing separate command directory. Use a public package name, exact version or supported numeric version condition. Supports fresh tools on Linux amd64; provide the Python runtime and virtual environment beforehand. Command collisions preserve the existing files."
const uvToolInstallKorean = "uv tool install이 선택한 공개 PyPI wheel과 의존성 전체를 검역한 뒤, 고정 Python 버전의 기존 빈 가상환경에 툴을 설치하고 명령을 노출합니다. 선택한 wheel 전체를 현재 검사와 승인으로 확인합니다.\n\n--target에는 마지막 디렉터리가 정규화된 패키지명인 기존 가상환경의 절대 경로를, --bin-dir에는 별도의 기존 명령 디렉터리 절대 경로를 지정하세요. 공개 패키지명·정확한 버전·지원하는 숫자 버전 조건을 사용할 수 있습니다. Linux amd64의 새 툴 설치를 지원하며 Python runtime과 가상환경은 미리 준비해야 합니다. 명령이 충돌하면 기존 파일을 보존합니다."

var uvToolInstallFlagText = map[string]packageClientParent{
	"target":  {"Existing empty virtual environment named after the package (absolute path)", "패키지명으로 끝나는 기존 빈 가상환경의 절대 경로"},
	"bin-dir": {"Existing separate directory for exposed tool commands (absolute path)", "명령을 노출할 별도 기존 디렉터리의 절대 경로"},
}

func uvToolInstallDescriptor() packageClientDescriptor {
	return packageClientDescriptor{
		id: UVToolPackageClient, path: []string{"uv", "tool", "install"}, parents: []packageClientParent{{"Quarantine Python packages with UV", "UV로 Python 패키지를 검역합니다"}, {"Inspect and install persistent Python tools", "Python 툴을 검역한 뒤 설치합니다"}}, route: "pypi", use: "install <requirement>", shortEN: "Quarantine and install a fresh public Python tool", shortKO: "공개 Python 툴을 검역한 뒤 새로 설치합니다", longEN: uvToolInstallEnglish, longKO: uvToolInstallKorean, validate: validateUVToolInstallArgs, validateRaw: validateUVToolInstallRawArgs, parse: parseUVToolInstallReference, flags: configureUVToolInstallFlags, upgrade: func(*cobra.Command, []string) (packageUpgradeMode, error) { return packageNoUpgrade, nil }, flagText: uvToolInstallFlagText, examples: []string{"helox uv tool install cowsay==6.1 --target /absolute/path/cowsay --bin-dir /absolute/path/bin"},
	}
}

func configureUVToolInstallFlags(command *cobra.Command) {
	command.Flags().String("bin-dir", "", uvToolInstallFlagText["bin-dir"].en)
}

func parseUVToolInstallReference(input string, source domain.SourceID) (domain.ArtifactReference, error) {
	if source != artifactpypi.PublicPyPIProfile().Source() {
		return domain.ArtifactReference{}, errors.New("UV tool install requires public PyPI")
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement(input)
	if err != nil {
		return domain.ArtifactReference{}, err
	}
	return domain.NewArtifactReference(source, requirement.Requirement())
}

func validateUVToolInstallRawArgs(args []string) error {
	return validateToolPathRawArgs(args, "UV tool install")
}

func validateToolPathRawArgs(args []string, label string) error {
	counts := map[string]int{}
	for _, arg := range args {
		name, _, _ := strings.Cut(arg, "=")
		if name == "--target" || name == "--bin-dir" {
			counts[name]++
			if counts[name] > 1 {
				return errors.New(label + " paths must not repeat")
			}
		}
	}
	return nil
}

// Syntax preflight creates no runtime/cache and grants no ownership or Policy.
// The prepared install session freezes both actual roots independently.
func validateUVToolInstallArgs(command *cobra.Command, args []string) error {
	return validateUVToolArgs(command, args, "install")
}

func validateUVToolArgs(command *cobra.Command, args []string, operation string) error {
	return validateToolPathArgs(command, args, "UV tool "+operation)
}

func validateToolPathArgs(command *cobra.Command, args []string, label string) error {
	if err := cobra.ExactArgs(1)(command, args); err != nil {
		return err
	}
	reference, err := parseUVToolInstallReference(args[0], artifactpypi.PublicPyPIProfile().Source())
	if err != nil {
		return err
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement(reference.Locator())
	if err != nil {
		return err
	}
	target, err := command.Flags().GetString("target")
	if err != nil {
		return err
	}
	bin, err := command.Flags().GetString("bin-dir")
	if err != nil {
		return err
	}
	if !command.Flags().Changed("target") || !command.Flags().Changed("bin-dir") || target == "" || bin == "" {
		return errors.New(label + " requires explicit --target and --bin-dir")
	}
	for _, value := range []string{target, bin} {
		if _, err := domain.NewInstallTarget(value); err != nil || strings.ContainsAny(value, ",:\\\x00\r\n") {
			return errors.New(label + " paths must be supported absolute directories")
		}
		resolved, err := filepath.EvalSymlinks(value)
		if err != nil || resolved != value {
			return errors.New(label + " paths or parents are unavailable or symlinked")
		}
		info, err := os.Lstat(value)
		if err != nil || !info.IsDir() {
			return errors.New(label + " paths must be existing directories")
		}
	}
	if filepath.Base(target) != requirement.Project() {
		return errors.New(label + " target directory must match the normalized package name")
	}
	if target == bin || strings.HasPrefix(target+"/", bin+"/") || strings.HasPrefix(bin+"/", target+"/") {
		return errors.New(label + " target and command directories must be separate")
	}
	return nil
}

func AddUVToolInstall(root *cobra.Command, installer Installer) error {
	if installer == nil {
		return errors.New("UV tool install requires a configured installer")
	}
	return addPackageClientInstall(root, UVToolPackageClient, map[string]Installer{"pypi": installer}, nil)
}
