package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

type PythonRequirementsRequest struct {
	Requirements artifactpypi.RequirementsRequest
	Install      domain.InstallContext
}

type PythonRequirementsOutcome struct {
	Verified                   domain.ProjectVerifiedSet
	Noop                       bool
	ManifestID                 domain.ContentDigest
	ManifestHandle, SBOMHandle string
}

type PythonRequirementsInstaller interface {
	InstallRequirements(context.Context, PythonRequirementsRequest) (PythonRequirementsOutcome, error)
}

const requirementsInstallEnglish = "\n\nFor additive batch installation, use -r/--requirement requirements.txt (repeatable), -c/--constraint, or multiple public requirements. Ranges, extras, target-runtime markers, relative includes and declared hashes use the fixed native resolver. The complete selected set is quarantined before one atomic installation; unrelated installed packages remain. Direct/VCS/editable/private sources and host environment expansion are unsupported. An empty active request is a no-op."
const requirementsInstallKorean = "\n\n일괄 추가 설치는 -r/--requirement requirements.txt(반복 가능), -c/--constraint 또는 여러 공개 requirements를 사용합니다. 범위·extras·대상 실행 환경 marker·상대 include·선언 hash를 고정 native resolver로 해석합니다. 선택 전체를 검역한 뒤 하나의 원자 설치로 반영하고 기존 무관 패키지는 보존합니다. Direct/VCS/editable/private 출처와 Host 환경변수 확장은 지원하지 않습니다. 활성 요청이 없으면 검사·설치를 건너뜁니다."

var requirementsFlagText = map[string]packageClientParent{
	"requirement":    {"Install the complete quarantined requirements file (repeatable)", "Requirements 파일 전체를 검역한 뒤 설치합니다(반복 가능)"},
	"constraint":     {"Constrain selection with a relative requirements file (repeatable)", "Requirements 파일로 선택 버전을 제한합니다(반복 가능)"},
	"require-hashes": {"Require declared hashes for every selected distribution", "선택된 모든 배포 파일에 선언된 hash를 요구합니다"},
}

func configureRequirementsInstallFlags(command *cobra.Command) {
	command.Flags().StringArrayP("requirement", "r", nil, requirementsFlagText["requirement"].en)
	command.Flags().StringArrayP("constraint", "c", nil, requirementsFlagText["constraint"].en)
	command.Flags().Bool("require-hashes", false, requirementsFlagText["require-hashes"].en)
}

// Keep already supported single-root paths and their source/upgrade semantics.
// Files, multiple roots and expanded public grammar use the additive set lane.
func usesPythonRequirements(command *cobra.Command, args []string) bool {
	for _, name := range []string{"requirement", "constraint", "require-hashes"} {
		if command.Flags().Changed(name) {
			return true
		}
	}
	if len(args) != 1 {
		return true
	}
	descriptor, ok := packageClientForCommand(command)
	if !ok {
		return false
	}
	if descriptor.id == UVPipPackageClient {
		if strings.Contains(args[0], "==") && !strings.ContainsAny(args[0], "[];<>!~*,") {
			return false
		}
		_, err := parseUVPipInstallReference(args[0], artifactpypi.PublicPyPIProfile().Source())
		return err != nil
	}
	source, _ := command.Flags().GetString("source")
	if source != "" && source != "pypi" {
		return false
	}
	upgrade, _ := command.Flags().GetBool("upgrade")
	if upgrade {
		_, err := artifactpypi.ParseUpgradeRequirement(args[0])
		return err != nil
	}
	_, err := artifactpypi.ParseReference(args[0])
	return err != nil
}

func readPythonRequirementsRequest(command *cobra.Command, args []string) (PythonRequirementsRequest, error) {
	var request PythonRequirementsRequest
	descriptor, ok := packageClientForCommand(command)
	if !ok || descriptor.id != PipPackageClient && descriptor.id != UVPipPackageClient {
		return request, errors.New("requirements installation needs a pip or UV pip command")
	}
	intent := artifactpypi.RequirementsRequest{Client: "pip"}
	if descriptor.id == UVPipPackageClient {
		intent.Client = "uv"
	}
	if command.Flags().Lookup("source") != nil {
		source, _ := command.Flags().GetString("source")
		if source != "" && source != "pypi" {
			return request, errors.New("requirements files and multiple roots require public PyPI; named source single-root commands retain their separate contract")
		}
	}
	if command.Flags().Lookup("inspection-prerequisites") != nil && command.Flags().Changed("inspection-prerequisites") {
		return request, errors.New("requirements installation does not accept single-root inspection prerequisites")
	}
	if command.Flags().Lookup("upgrade-package") != nil && command.Flags().Changed("upgrade-package") {
		return request, errors.New("requirements installation supports --upgrade; --upgrade-package requires the existing single-root command")
	}
	intent.Upgrade, _ = command.Flags().GetBool("upgrade")
	intent.Adopt, _ = command.Flags().GetBool("adopt")
	intent.RequireHashes, _ = command.Flags().GetBool("require-hashes")
	for _, group := range []struct {
		name        string
		destination *[]string
	}{{"requirement", &intent.Sources}, {"constraint", &intent.Constraints}} {
		values := command.Flags().Lookup(group.name).Value.(interface{ GetSlice() []string }).GetSlice()
		for _, value := range values {
			if value == "" || value == "-" || strings.ContainsAny(value, "\x00\r\n") {
				return request, errors.New("requirements options need bounded regular file paths")
			}
			name, err := filepath.Abs(value)
			if err != nil {
				return request, err
			}
			info, err := os.Lstat(name)
			if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > int64(artifactpypi.RequirementsInputByteLimit) {
				return request, errors.New("requirements source must be an available bounded regular file")
			}
			*group.destination = append(*group.destination, name)
		}
	}
	if len(intent.Sources)+len(intent.Constraints) > artifactpypi.MaximumRequirementsFiles {
		return request, errors.New("requirements source count exceeds bounds")
	}
	for _, value := range args {
		if strings.Contains(value, "@") {
			reference, err := artifactpypi.ParseReference(value)
			if err != nil {
				return request, errors.New("direct requirement sources are unsupported")
			}
			name, _ := artifactpypi.RequestedProject(reference)
			version, exact, _ := artifactpypi.RequestedVersion(reference)
			if !exact {
				return request, errors.New("ambiguous requirements reference")
			}
			value = name + "==" + version
		}
		intent.Positional = append(intent.Positional, value)
	}
	if len(intent.Positional) > 0 {
		if _, err := artifactpypi.NewRequirementsInput(nil, nil, intent.Positional); err != nil {
			return request, err
		}
	}
	if len(intent.Sources)+len(intent.Constraints)+len(intent.Positional) == 0 {
		return request, errors.New("requirements installation needs a requirement or a requirements file")
	}
	target, _ := command.Flags().GetString("target")
	installTarget, err := activePythonVenvForCommand(target, strings.Join(descriptor.path, " "))
	if err != nil {
		return request, err
	}
	if strings.Contains(installTarget.String(), ",") {
		return request, errors.New("requirements target cannot contain a comma")
	}
	install, err := domain.NewPythonVenvInstallContext(installTarget)
	if err != nil {
		return request, err
	}
	return PythonRequirementsRequest{Requirements: intent, Install: install}, nil
}

func AddPythonRequirementsInstall(root *cobra.Command, id PackageClientID, service PythonRequirementsInstaller) error {
	if root == nil || service == nil || id != PipPackageClient && id != UVPipPackageClient {
		return errors.New("requirements installation needs an existing client and guarded service")
	}
	command, err := PackageClientInstallCommand(root, id)
	if err != nil {
		return err
	}
	previous := command.RunE
	if previous == nil {
		return errors.New("requirements installation must retain the existing single-root binding")
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		if !usesPythonRequirements(command, args) {
			return previous(command, args)
		}
		request, err := readPythonRequirementsRequest(command, args)
		if err != nil {
			return err
		}
		ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finish() }()
		outcome, cause := service.InstallRequirements(ctx, request)
		if cause == nil {
			if outcome.Noop {
				if outcome.Verified.Valid() || outcome.ManifestID.String() != "" || outcome.ManifestHandle != "" || outcome.SBOMHandle != "" {
					cause = errors.New("requirements no-op reported inspection or publication")
				}
			} else if !outcome.Verified.Valid() || outcome.Verified.Inspected().Snapshot().Context() != request.Install || outcome.Verified.Inspected().Snapshot().Source() != artifactpypi.PublicPyPIProfile().Source() || outcome.ManifestID.String() == "" || outcome.ManifestHandle == "" || outcome.SBOMHandle == "" {
				cause = errors.New("requirements installation returned no complete approved publication records")
			}
		}
		progressErr := finish()
		if cause == nil && outcome.Noop {
			p := presentationFor(command.OutOrStdout())
			if !p.human() {
				return errors.Join(progressErr, json.NewEncoder(command.OutOrStdout()).Encode(cargoUpdateReport{Schema: "helox.project-install-result/v1", Operation: "INSTALL", Status: "SKIPPED", Promotion: "NOT_PERFORMED", Source: "pypi", Entries: []cargoUpdateEntry{}}))
			}
			p.reportHeader("INSTALL", "SKIPPED", "", "")
			p.line("%s", p.text("No active requirements; inspection and installation were skipped.", "활성 requirements이 없어 검사와 설치를 건너뛰었습니다."))
			return errors.Join(progressErr, p.writeErr)
		}
		var records *machineVerifiedSet
		if cause == nil {
			records = &machineVerifiedSet{ManifestID: outcome.ManifestID.String(), SBOMHandle: outcome.SBOMHandle, StagingHandle: strings.TrimSuffix(outcome.ManifestHandle, ":manifest.json"), EntryCount: len(outcome.Verified.Inspected().Inspections())}
		}
		reportErr := writeProjectMutationReportWithRecords(command.OutOrStdout(), outcome.Verified, cause, "", "pypi", records, true)
		return errors.Join(cause, progressErr, reportErr)
	}
	return nil
}
