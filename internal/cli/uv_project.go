package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

type UVProjectRequest struct {
	ProjectRoot string
	Intent      artifactpypi.UVProjectIntent
	Selector    string
	Install     domain.InstallContext
}
type UVProjectOutcome struct {
	Verified                   domain.ProjectVerifiedSet
	ManifestID                 domain.ContentDigest
	ManifestHandle, SBOMHandle string
}
type UVProjectApplier interface {
	ApplyUVProject(context.Context, UVProjectRequest) (UVProjectOutcome, error)
}

const uvProjectEnglish = "Quarantine the complete public PyPI wheel set selected by UV before adding a dependency or synchronizing one virtual project. Use the current directory or --project, and an existing pinned-runtime virtual environment selected by --target (defaults to project/.venv). Every selected wheel is inspected again, including unchanged versions. pyproject.toml, uv.lock and owned environment files are committed together; user files are preserved. Sync retires every proven unselected distribution, including an explicitly empty dependencies array. An edited manifest may retain its previous public uv.lock before resolution.\n\nSupports one virtual project on Linux amd64. Use a public name or PEP 508 version requirement for add. Workspaces, build projects, groups/extras/markers, local/VCS/direct sources, other indexes and arbitrary native options are unsupported. Existing installed distributions require HAA ownership. Native project adoption, lock-only changes, runtime provisioning and uv run are separate unsupported operations in this lane."
const uvProjectKorean = "UV가 선택한 공개 PyPI wheel 전체를 검역한 뒤 virtual project에 의존성을 추가하거나 동기화합니다. 현재 디렉터리 또는 --project를 사용하며, --target의 기존 고정 runtime 가상환경을 대상으로 합니다(기본값: project/.venv). 같은 버전도 선택된 wheel 전체를 새로 검사합니다. pyproject.toml·uv.lock과 소유권이 확인된 환경 파일을 함께 반영하고 사용자 파일은 보존합니다. Sync는 선택에서 빠진 소유 패키지를 제거하며 명시적 빈 dependencies 배열도 지원합니다. Manifest를 편집했을 때 이전 공개 uv.lock을 남긴 채 새로 선택할 수 있습니다.\n\nLinux amd64의 단일 virtual project를 지원합니다. Add에는 공개 패키지 이름이나 PEP 508 버전 조건을 사용하세요. Workspace·build project·group/extra/marker·local/VCS/direct 출처·다른 index·임의 native 옵션은 지원하지 않습니다. 기존 설치 패키지에는 HAA 소유권이 필요합니다. Native project 인계·lock-only 변경·runtime 준비·uv run은 이 경로에서 지원하지 않습니다."

var uvProjectFlagText = map[string]packageClientParent{
	"project": {"Existing virtual project (defaults to current directory)", "기존 virtual project(기본값: 현재 디렉터리)"},
	"target":  {"Existing pinned-runtime venv (defaults to project/.venv)", "기존 고정 runtime 가상환경(기본값: project/.venv)"},
}

func configureUVProjectFlags(command *cobra.Command) {
	command.Flags().String("project", "", uvProjectFlagText["project"].en)
	command.Flags().Lookup("target").Usage = uvProjectFlagText["target"].en
}
func validateUVProjectRawArgs(args []string) error {
	counts := map[string]int{}
	for _, arg := range args {
		name, _, _ := strings.Cut(arg, "=")
		if name == "--project" || name == "--target" {
			counts[name]++
			if counts[name] > 1 {
				return errors.New("UV project options must not repeat")
			}
		}
	}
	return nil
}
func validateUVProjectArgs(command *cobra.Command, args []string) error {
	_, err := readUVProjectRequest(command, args)
	return err
}

func readUVProjectRequest(command *cobra.Command, args []string) (request UVProjectRequest, resultErr error) {
	intent := artifactpypi.UVProjectIntent(command.Name())
	selector := ""
	switch intent {
	case artifactpypi.UVProjectAdd:
		if err := cobra.ExactArgs(1)(command, args); err != nil {
			return request, err
		}
		selector = args[0]
		if selector == "" {
			return request, errors.New("UV add requires one public dependency")
		}
	case artifactpypi.UVProjectSync:
		if err := cobra.NoArgs(command, args); err != nil {
			return request, err
		}
	default:
		return request, errors.New("unsupported UV project operation")
	}
	project, err := command.Flags().GetString("project")
	if err != nil {
		return request, err
	}
	if command.Flags().Changed("project") && project == "" {
		return request, errors.New("UV --project must not be empty")
	}
	if project == "" {
		project, err = os.Getwd()
		if err != nil {
			return request, err
		}
	}
	project, err = filepath.Abs(project)
	if err != nil {
		return request, err
	}
	resolved, err := filepath.EvalSymlinks(project)
	if err != nil || resolved != project || project == "/" {
		return request, errors.New("UV project requires an existing canonical directory without symlinked ancestry")
	}
	before, err := os.Lstat(project)
	if err != nil || !before.IsDir() {
		return request, errors.New("UV project directory is unavailable")
	}
	root, err := os.OpenRoot(project)
	if err != nil {
		return request, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, root.Close())
		if resultErr != nil {
			request = UVProjectRequest{}
		}
	}()
	manifest, err := readProjectPreflightControl(root, artifactpypi.UVProjectInputControl, false, 4<<20, "UV project")
	if err != nil {
		return request, err
	}
	lock, err := readProjectPreflightControl(root, artifactpypi.UVNativeLockControl, true, 4<<20, "UV project")
	if err != nil {
		return request, err
	}
	input, err := artifactpypi.ParseUVProjectInput(manifest.Body(), lock.Body(), lock.Present(), intent, selector)
	if err != nil {
		return request, err
	}
	if lock.Present() {
		if err := artifactpypi.ValidateUVProjectOriginalLock(input, lock.Body()); err != nil {
			return request, err
		}
	}
	opened, err := root.Stat(".")
	current, currentErr := os.Lstat(project)
	if err != nil || currentErr != nil || !os.SameFile(before, opened) || !os.SameFile(before, current) || opened.Mode() != before.Mode() || current.Mode() != before.Mode() {
		return request, errors.New("UV project directory changed during preflight")
	}
	target, err := command.Flags().GetString("target")
	if err != nil {
		return request, err
	}
	if command.Flags().Changed("target") && target == "" {
		return request, errors.New("UV --target must not be empty")
	}
	if target == "" {
		target = filepath.Join(project, ".venv")
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return request, err
	}
	resolved, err = filepath.EvalSymlinks(target)
	if err != nil || resolved != target || strings.Contains(target, ",") {
		return request, errors.New("UV project requires an existing venv without symlinked ancestry or mount delimiters")
	}
	info, err := os.Lstat(target)
	if err != nil || !info.IsDir() {
		return request, errors.New("UV project target must be an existing venv directory")
	}
	for _, name := range []string{artifactpypi.UVProjectInputControl, artifactpypi.UVNativeLockControl} {
		rel, err := filepath.Rel(target, filepath.Join(project, name))
		if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return request, errors.New("UV project controls must be outside the venv target")
		}
	}
	installTarget, err := domain.NewInstallTarget(target)
	if err != nil {
		return request, err
	}
	install, err := domain.NewPythonVenvInstallContext(installTarget)
	if err != nil {
		return request, err
	}
	return UVProjectRequest{project, intent, input.Selector(), install}, nil
}

func AddUVProject(root *cobra.Command, service UVProjectApplier) error {
	if root == nil || service == nil {
		return errors.New("UV project requires guarded application")
	}
	for _, operation := range []string{"add", "sync"} {
		command, err := packageClientCommand(root, UVProjectPackageClient, operation)
		if err != nil {
			return err
		}
		command.RunE = func(command *cobra.Command, args []string) error {
			request, err := readUVProjectRequest(command, args)
			if err != nil {
				return err
			}
			ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
			defer func() { _ = finish() }()
			outcome, cause := service.ApplyUVProject(ctx, request)
			prefix := strings.TrimSuffix(outcome.ManifestHandle, ":manifest.json")
			if cause == nil && (!outcome.Verified.Valid() || outcome.Verified.Inspected().Snapshot().Context() != request.Install || outcome.Verified.Inspected().Snapshot().Source() != artifactpypi.PublicPyPIProfile().Source() || outcome.ManifestID.String() == "" || prefix == outcome.ManifestHandle || !strings.HasPrefix(prefix, "project-cache:") || outcome.SBOMHandle != prefix+":sbom.cdx.json") {
				cause = errors.New("UV project returned no exact completed publication records")
			}
			progressErr := finish()
			var records *machineVerifiedSet
			if cause == nil {
				records = &machineVerifiedSet{ManifestID: outcome.ManifestID.String(), SBOMHandle: outcome.SBOMHandle, StagingHandle: prefix, EntryCount: len(outcome.Verified.Inspected().Inspections())}
			}
			reportErr := writeProjectMutationReportWithRecords(command.OutOrStdout(), outcome.Verified, cause, request.Selector, "pypi", records, request.Intent == artifactpypi.UVProjectAdd)
			return errors.Join(cause, progressErr, reportErr)
		}
	}
	return nil
}
