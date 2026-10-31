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

const uvProjectLockEnglish = "Quarantine every public PyPI wheel selected by UV before upgrading an existing virtual project's uv.lock. Use the current directory or --project. pyproject.toml remains unchanged. The complete selected set is inspected again, including unchanged versions, and the result includes verified Manifest/SBOM references.\n\nThis command updates only the lock file; it does not create, inspect or install into a virtual environment. A later sync performs a new complete quarantine. Supports one virtual project with an existing supported public uv.lock on Linux amd64. Workspaces, build projects, groups/extras/markers, local/VCS/direct sources, other indexes and arbitrary native options are unsupported."
const uvProjectLockKorean = "UV가 선택한 공개 PyPI wheel 전체를 검역한 뒤 기존 virtual project의 uv.lock을 갱신합니다. 현재 디렉터리 또는 --project를 사용하며 pyproject.toml은 보존합니다. 같은 버전도 선택 전체를 새로 검사하고 검증된 Manifest/SBOM 참조를 제공합니다.\n\n잠금 파일만 갱신하며 가상환경을 생성·조회하거나 패키지를 설치하지 않습니다. 나중 sync에서도 전체 검역을 다시 수행합니다. Linux amd64에서 지원되는 기존 공개 uv.lock이 있는 단일 virtual project를 지원합니다. Workspace·build project·group/extra/marker·local/VCS/direct 출처·다른 index·임의 native 옵션은 지원하지 않습니다."

var uvProjectLockFlagText = map[string]packageClientParent{
	"project": {"Existing virtual project (defaults to current directory)", "기존 virtual project(기본값: 현재 디렉터리)"},
	"upgrade": {"Explicitly upgrade the existing public lock; no environment installation", "기존 공개 lock을 명시적으로 갱신(환경 설치 없음)"},
}

func configureUVProjectLockFlags(command *cobra.Command) {
	command.Flags().String("project", "", uvProjectLockFlagText["project"].en)
	command.Flags().Bool("upgrade", false, uvProjectLockFlagText["upgrade"].en)
	if flag := command.Flags().Lookup("target"); flag != nil {
		flag.Hidden = true
	}
}
func validateUVProjectLockRawArgs(args []string) error {
	counts := map[string]int{}
	for _, arg := range args {
		name, _, _ := strings.Cut(arg, "=")
		if name == "--project" || name == "--upgrade" {
			counts[name]++
			if counts[name] > 1 {
				return errors.New("UV lock options must not repeat")
			}
		}
	}
	return nil
}
func validateUVProjectLockArgs(command *cobra.Command, args []string) error {
	_, err := readUVProjectLockRequest(command, args)
	return err
}

func readUVProjectLockRequest(command *cobra.Command, args []string) (UVProjectRequest, error) {
	if err := cobra.NoArgs(command, args); err != nil {
		return UVProjectRequest{}, err
	}
	upgrade, err := command.Flags().GetBool("upgrade")
	if err != nil || !upgrade {
		return UVProjectRequest{}, errors.New("UV lock requires explicit --upgrade")
	}
	if command.Flags().Changed("target") {
		return UVProjectRequest{}, errors.New("UV lock updates project controls and has no venv target")
	}
	project, input, err := readUVProjectSource(command, artifactpypi.UVProjectLockUpgrade, "")
	if err != nil {
		return UVProjectRequest{}, err
	}
	target, err := domain.NewInstallTarget(project)
	if err != nil {
		return UVProjectRequest{}, err
	}
	install, err := domain.NewPythonProjectControlContext(target)
	if err != nil {
		return UVProjectRequest{}, err
	}
	return UVProjectRequest{ProjectRoot: project, Intent: input.Intent(), Install: install}, nil
}

// Both project operations use the same bounded original-source preflight.
// The actual mutation guard independently recaptures and holds these inputs.
func readUVProjectSource(command *cobra.Command, intent artifactpypi.UVProjectIntent, selector string) (canonical string, input artifactpypi.UVProjectInput, resultErr error) {
	project, err := command.Flags().GetString("project")
	if err != nil {
		return "", input, err
	}
	if command.Flags().Changed("project") && project == "" {
		return "", input, errors.New("UV --project must not be empty")
	}
	if project == "" {
		project, err = os.Getwd()
		if err != nil {
			return "", input, err
		}
	}
	project, err = filepath.Abs(project)
	if err != nil {
		return "", input, err
	}
	resolved, err := filepath.EvalSymlinks(project)
	if err != nil || resolved != project || project == "/" {
		return "", input, errors.New("UV project requires an existing canonical directory without symlinked ancestry")
	}
	before, err := os.Lstat(project)
	if err != nil || !before.IsDir() {
		return "", input, errors.New("UV project directory is unavailable")
	}
	root, err := os.OpenRoot(project)
	if err != nil {
		return "", input, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, root.Close())
		if resultErr != nil {
			canonical = ""
			input = artifactpypi.UVProjectInput{}
		}
	}()
	manifest, err := readProjectPreflightControl(root, artifactpypi.UVProjectInputControl, false, 4<<20, "UV project")
	if err != nil {
		return "", input, err
	}
	lock, err := readProjectPreflightControl(root, artifactpypi.UVNativeLockControl, true, 4<<20, "UV project")
	if err != nil {
		return "", input, err
	}
	input, err = artifactpypi.ParseUVProjectInput(manifest.Body(), lock.Body(), lock.Present(), intent, selector)
	if err != nil {
		return "", input, err
	}
	if lock.Present() {
		if err := artifactpypi.ValidateUVProjectOriginalLock(input, lock.Body()); err != nil {
			return "", input, err
		}
	}
	opened, err := root.Stat(".")
	current, currentErr := os.Lstat(project)
	if err != nil || currentErr != nil || !os.SameFile(before, opened) || !os.SameFile(before, current) || opened.Mode() != before.Mode() || current.Mode() != before.Mode() {
		return "", input, errors.New("UV project directory changed during preflight")
	}
	return project, input, nil
}
