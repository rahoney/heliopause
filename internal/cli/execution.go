package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/spf13/cobra"
)

type ExecutionRunner interface {
	Run(context.Context, application.ExecutionPreparationRequest) (application.ExecutionResult, error)
}

const npxExecutionEnglish = "Freshly quarantine one public npm package and its complete dependency set, then run its exact Node entry in a disposable read-only sandbox on Linux amd64. No Host project, HOME, credentials, stdin, TTY, network or generated-file export is supplied.\n\nPlace all HAA options before the package. Everything after the package is literal program data; one immediately following -- is a separator. Use --command before the package when its bin name differs or there are multiple bins. Shell commands, runtime overrides, extra packages and local/global cache execution are unsupported. Arguments are limited to 128 and 32KiB. Output is private by default: only byte counts are reported. --show-output explicitly displays independent ASCII-escaped untrusted streams, each captured up to 1MiB. Quarantine approval, execution outcome and guest cleanup are reported separately."
const npxExecutionKorean = "공개 npm 패키지 하나와 전체 의존성을 새로 검역한 뒤 Linux amd64의 일회성 읽기 전용 sandbox에서 정확한 Node entry를 실행합니다. Host 프로젝트·HOME·credential·stdin·TTY·네트워크·생성 파일 반출은 제공하지 않습니다.\n\nHAA 옵션은 모두 패키지 앞에 두세요. 패키지 뒤는 프로그램의 literal 인자이며 바로 뒤 -- 하나는 구분자입니다. bin 이름이 다르거나 여러 개면 패키지 앞 --command로 명시하세요. Shell·임의 runtime·추가 패키지·로컬/global cache 실행은 지원하지 않습니다. 프로그램 인자는 최대128개/합계32KiB입니다. 기본 출력에는 바이트 수만 보고합니다. --show-output을 명시하면 stream별 최대1MiB의 신뢰하지 않는 출력을 각각 ASCII escape로 표시합니다. 검역 승인·실행 결과·guest 정리를 구분합니다."

func npxExecutionDescriptor(npm bool) packageClientDescriptor {
	path, use := []string{"npx"}, "npx <package> [--] [args...]"
	if npm {
		path, use = []string{"npm", "exec"}, "exec <package> [--] [args...]"
	}
	return packageClientDescriptor{id: NPXPackageClient, path: path, parents: []packageClientParent{{"Inspect npm packages", "npm 패키지를 검역합니다"}}, route: "npm", use: use,
		shortEN: "Quarantine and run an exact public npm Node entry", shortKO: "공개 npm의 정확한 Node entry를 검역한 뒤 격리 실행합니다", longEN: npxExecutionEnglish, longKO: npxExecutionKorean,
		execution: true, validate: validateNPMExecutionArgs, validateRaw: validateNPMExecutionRawArgs, flags: configureNPMExecutionFlags,
		examples: []string{"helox npx --command tsc --show-output typescript@5.9.3 -- --help", "helox --output json npm exec --command tsc typescript@5.9.3 -- --help"},
		flagText: map[string]packageClientParent{"command": {"Exact bin name when explicit selection is required", "명시 선택할 정확한 bin 이름"}, "show-output": {"Display private untrusted streams as ASCII-escaped data", "비공개 프로그램 출력을 ASCII escape 데이터로 명시 표시"}}}
}

func configureNPMExecutionFlags(command *cobra.Command) {
	command.Flags().String("command", "", "Exact selected bin name")
	command.Flags().Bool("show-output", false, "Display untrusted ASCII-escaped program output")
	command.Flags().SetInterspersed(false)
}

func validateNPMExecutionRawArgs(args []string) error {
	start := -1
	for i, arg := range args {
		if arg == "npx" || arg == "exec" && i > 0 && args[i-1] == "npm" {
			start = i + 1
			break
		}
	}
	return validateExecutionRawOptions(args, start, "--command")
}

func validateExecutionRawOptions(args []string, start int, valueOption string) error {
	counts := map[string]int{}
	for i := start; i >= 0 && i < len(args); i++ {
		arg := args[i]
		if arg == "--" || !strings.HasPrefix(arg, "-") {
			break
		}
		name, _, equals := strings.Cut(arg, "=")
		if name == valueOption || name == "--show-output" {
			counts[name]++
			if counts[name] > 1 {
				return errors.New("execution selection and output options must not repeat before the package")
			}
		}
		if !equals && (name == valueOption || name == "--lang" || name == "--output" || name == "--color" || name == "--progress") {
			i++
		}
	}
	return nil
}

func readNPMExecutionRequest(command *cobra.Command, args []string) (application.ExecutionPreparationRequest, error) {
	if len(args) == 0 {
		return application.ExecutionPreparationRequest{}, errors.New("execution requires one public npm package")
	}
	reference, err := artifactnpm.ParseReference(args[0])
	if err != nil {
		return application.ExecutionPreparationRequest{}, errors.New("execution requires a supported public npm package reference")
	}
	selected, err := command.Flags().GetString("command")
	if err != nil || command.Flags().Changed("command") && selected == "" {
		return application.ExecutionPreparationRequest{}, errors.New("explicit execution command must be nonempty")
	}
	arguments := args[1:]
	if len(arguments) != 0 && arguments[0] == "--" {
		arguments = arguments[1:]
	}
	return application.NewExecutionPreparationRequest(reference, selected, arguments)
}

func validateNPMExecutionArgs(command *cobra.Command, args []string) error {
	_, err := readNPMExecutionRequest(command, args)
	return err
}

// Fixed text prevents main from replaying raw backend/program diagnostics.
type executionCommandError struct {
	code  int
	cause error
}

func (e executionCommandError) Error() string {
	return "execution failed; see the separate execution report"
}
func (e executionCommandError) ExitCode() int { return e.code }
func (e executionCommandError) Unwrap() error { return e.cause }

func AddNPMExecution(root *cobra.Command, runner ExecutionRunner) error {
	if root == nil || runner == nil {
		return errors.New("execution CLI requires a root and current workflow")
	}
	for _, operation := range []string{"npx", "exec"} {
		command, err := packageClientCommand(root, NPXPackageClient, operation)
		if err != nil {
			return err
		}
		command.RunE = func(command *cobra.Command, args []string) error {
			request, err := readNPMExecutionRequest(command, args)
			if err != nil {
				return err
			}
			return runExecutionRequest(command, runner, request)
		}
	}
	return nil
}

func runExecutionRequest(command *cobra.Command, runner ExecutionRunner, request application.ExecutionPreparationRequest) error {
	ctx, finish := beginProgress(command.Context(), command.OutOrStdout(), machineOutput(command))
	defer func() { _ = finish() }()
	result, cause := runner.Run(ctx, request)
	progressErr := finish()
	show, _ := command.Flags().GetBool("show-output")
	report := newExecutionReport(result, cause, show)
	reportErr := writeExecutionReport(command.OutOrStdout(), report, machineOutput(command))
	code := executionProcessCode(result, cause)
	if reportErr != nil || progressErr != nil {
		code = 1
	}
	if code != 0 || cause != nil || reportErr != nil || progressErr != nil {
		return executionCommandError{code, errors.Join(cause, reportErr, progressErr)}
	}
	return nil
}
