package cli

import (
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/spf13/cobra"
)

const uvRunExecutionEnglish = "Freshly quarantine one public PyPI package and its complete selected wheel set, then run its exact console entry with the pinned Python in a disposable read-only sandbox on Linux amd64. No Host project, HOME, credentials, stdin, TTY, network or generated-file export is supplied.\n\nUse a bare package or exact name@version/name==version. When package and command names differ, put --from <package> before the command. Put all HAA options before the package or command; everything following is literal program data, with one immediately following -- treated as a separator. Selection is always fresh and isolated. Interpreter commands, runtime overrides/downloads, extras, extra packages, local/VCS/URL input and custom indexes are unsupported. Arguments are limited to 128 and 32KiB. Output is private by default: only byte counts are reported. --show-output explicitly displays independent ASCII-escaped untrusted streams, each captured up to 1MiB. Quarantine approval, execution outcome and guest cleanup are reported separately."
const uvRunExecutionKorean = "공개 PyPI 패키지 하나와 선택된 wheel·의존성 전체를 새로 검역한 뒤 Linux amd64의 일회성 읽기 전용 sandbox에서 고정 Python으로 정확한 console entry를 실행합니다. Host 프로젝트·HOME·credential·stdin·TTY·네트워크·생성 파일 반출은 제공하지 않습니다.\n\n패키지명 또는 정확한 name@version/name==version을 사용하세요. 패키지와 명령 이름이 다르면 명령 앞 --from <package>로 지정하세요. HAA 옵션은 모두 패키지·명령 앞에 두며 뒤는 프로그램의 literal 인자입니다. 바로 뒤 -- 하나는 구분자입니다. 매번 새로 격리 선택합니다. Interpreter 명령·runtime 지정/다운로드·extras·추가 패키지·local/VCS/URL·임의 index는 지원하지 않습니다. 프로그램 인자는 최대128개/합계32KiB입니다. 기본 출력은 바이트 수만 보고하며 --show-output을 명시하면 stream별 최대1MiB의 신뢰하지 않는 출력을 각각 ASCII escape로 표시합니다. 검역 승인·실행 결과·guest 정리를 구분합니다."

func uvRunExecutionDescriptor(tool bool) packageClientDescriptor {
	path, use := []string{"uvx"}, "uvx [--from <package>] <tool-or-command> [--] [args...]"
	var parents []packageClientParent
	if tool {
		path = []string{"uv", "tool", "run"}
		use = "run [--from <package>] <tool-or-command> [--] [args...]"
		parents = []packageClientParent{{"Quarantine Python packages with UV", "UV로 Python 패키지를 검역합니다"}, {"Quarantine Python tools", "Python 툴을 검역합니다"}}
	}
	return packageClientDescriptor{id: UVRunPackageClient, path: path, parents: parents, route: "pypi", use: use, shortEN: "Quarantine and run an exact public Python console", shortKO: "공개 Python console을 검역한 뒤 격리 실행합니다", longEN: uvRunExecutionEnglish, longKO: uvRunExecutionKorean, execution: true, validate: validateUVRunExecutionArgs, validateRaw: validateUVRunExecutionRawArgs, flags: configureUVRunExecutionFlags, examples: []string{"helox uvx --show-output cowsay@6.1 -- -t hello", "helox --output json uv tool run --from python-slugify==8.0.4 slugify -- hello world"}, flagText: map[string]packageClientParent{"from": {"Public package when its console command differs", "console 명령 이름이 다를 때 선택할 공개 패키지"}, "show-output": {"Display private untrusted streams as ASCII-escaped data", "비공개 프로그램 출력을 ASCII escape 데이터로 명시 표시"}}}
}
func configureUVRunExecutionFlags(command *cobra.Command) {
	configureRunExecutionFlags(command, "from")
}
func configureRunExecutionFlags(command *cobra.Command, selectionFlag string) {
	command.Flags().String(selectionFlag, "", "Public package when command differs")
	command.Flags().Bool("show-output", false, "Display untrusted ASCII-escaped program output")
	command.Flags().SetInterspersed(false)
}
func validateUVRunExecutionRawArgs(args []string) error {
	start := -1
	uv, tool := false, false
	for i, arg := range args {
		if arg == "uvx" {
			start = i + 1
			break
		}
		if arg == "uv" {
			uv = true
		}
		if uv && arg == "tool" {
			tool = true
		}
		if tool && arg == "run" {
			start = i + 1
			break
		}
	}
	return validateExecutionRawOptions(args, start, "--from")
}

// The initial package-console lane excludes interpreter-looking requests.
// A name such as python-slugify remains an ordinary public package.
func uvRunInterpreterReference(project string) bool {
	for _, stem := range []string{"python", "cpython", "pypy", "graalpy", "jython", "ironpython", "pyston"} {
		if !strings.HasPrefix(project, stem) {
			continue
		}
		tail := strings.TrimPrefix(project, stem)
		if tail == "" || tail == "w" || tail[0] >= '0' && tail[0] <= '9' || len(tail) > 1 && tail[0] == '-' && tail[1] >= '0' && tail[1] <= '9' || len(tail) > 1 && tail[0] == 'w' && tail[1] >= '0' && tail[1] <= '9' {
			return true
		}
	}
	return false
}
func readUVRunExecutionRequest(command *cobra.Command, args []string) (application.ExecutionPreparationRequest, error) {
	return readPythonRunExecutionRequest(command, args, "from")
}
func readPythonRunExecutionRequest(command *cobra.Command, args []string, selectionFlag string) (application.ExecutionPreparationRequest, error) {
	if len(args) == 0 {
		return application.ExecutionPreparationRequest{}, errors.New("execution requires one public Python package or explicit console")
	}
	from, err := command.Flags().GetString(selectionFlag)
	if err != nil || command.Flags().Changed(selectionFlag) && from == "" {
		return application.ExecutionPreparationRequest{}, errors.New("explicit --" + selectionFlag + " package must be nonempty")
	}
	input := args[0]
	if from != "" {
		input = from
	}
	reference, err := parseUVPipInstallReference(input, artifactpypi.PublicPyPIProfile().Source())
	if err != nil {
		return application.ExecutionPreparationRequest{}, errors.New("execution requires a bare or exact final public Python package")
	}
	project, err := artifactpypi.RequestedProject(reference)
	if err != nil || uvRunInterpreterReference(project) {
		return application.ExecutionPreparationRequest{}, errors.New("execution requires a package console; interpreter provisioning is unsupported")
	}
	selected := project
	if from != "" {
		selected = args[0]
	}
	if artifactpypi.ValidateConsoleExecutionCommand(selected) != nil {
		return application.ExecutionPreparationRequest{}, errors.New("execution requires an exact supported console name")
	}
	arguments := args[1:]
	if len(arguments) > 0 && arguments[0] == "--" {
		arguments = arguments[1:]
	}
	return application.NewExecutionPreparationRequest(reference, selected, arguments)
}
func validateUVRunExecutionArgs(command *cobra.Command, args []string) error {
	_, err := readUVRunExecutionRequest(command, args)
	return err
}
func AddUVRunExecution(root *cobra.Command, runner ExecutionRunner) error {
	if root == nil || runner == nil {
		return errors.New("native execution CLI requires root and current workflow")
	}
	for _, operation := range []string{"uvx", "run"} {
		command, err := packageClientCommand(root, UVRunPackageClient, operation)
		if err != nil {
			return err
		}
		command.RunE = func(command *cobra.Command, args []string) error {
			request, err := readUVRunExecutionRequest(command, args)
			if err != nil {
				return err
			}
			return runExecutionRequest(command, runner, request)
		}
	}
	return nil
}
