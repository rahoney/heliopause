package cli

import (
	"errors"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/spf13/cobra"
)

func pnpmRunExecutionDescriptor(alias string) packageClientDescriptor {
	d := packageClientDescriptor{id: PNPMRunPackageClient, path: []string{alias}, route: "npm", use: alias + " [--package <package>] <package-or-command> [--] [args...]",
		shortEN: "Quarantine and run a public npm Node entry with PNPM", shortKO: "PNPM으로 공개 npm의 Node entry를 검역한 뒤 격리 실행합니다",
		longEN:    "Select one bare or exact public package with pinned PNPM, freshly quarantine the complete native dependency set, then run its declared Node entry in a bounded sandbox. Use --package <package> <command> when command selection is explicit. Otherwise select the sole bin or package basename; ambiguous bins are refused. Runtime provisioning, project/PATH/global-cache lookup, shell commands and alternate registries are unavailable. --show-output displays private program streams as ASCII-escaped data.",
		longKO:    "고정 PNPM으로 공개 패키지 이름 또는 정확한 버전 하나를 선택하고 전체 의존성을 새로 검역한 뒤 선언된 Node entry를 제한된 격리 환경에서 실행합니다. 명령을 직접 선택할 때는 --package <패키지> <명령>을 사용합니다. 생략하면 유일한 bin 또는 패키지 basename을 선택하며 모호하면 거부합니다. Runtime 설치·프로젝트/PATH/전역 cache 검색·shell·다른 registry는 지원하지 않습니다. --show-output은 비공개 프로그램 출력을 ASCII escape 데이터로 표시합니다.",
		execution: true, flags: func(command *cobra.Command) { configureRunExecutionFlags(command, "package") },
		validate: func(command *cobra.Command, args []string) error {
			_, err := readPNPMRunExecutionRequest(command, args)
			return err
		},
		validateRaw: validatePNPMRunExecutionRawArgs,
		examples:    []string{"helox pnpm dlx --package typescript@5.9.3 --show-output tsc -- --help", "helox --output json pnx semver@7.7.2 -- 1.2.3"},
		flagText:    map[string]packageClientParent{"package": {"Public package providing the explicit command", "명시한 명령을 제공할 공개 패키지"}, "show-output": {"Display private untrusted streams as ASCII-escaped data", "비공개 프로그램 출력을 ASCII escape 데이터로 명시 표시"}},
	}
	if alias == "dlx" {
		d.path = []string{"pnpm", "dlx"}
		d.parents = []packageClientParent{{"Quarantine public npm packages with PNPM", "PNPM으로 공개 npm 패키지를 검역합니다"}}
	}
	return d
}

func validatePNPMRunExecutionRawArgs(args []string) error {
	start := -1
	for i, arg := range args {
		if arg == "pnx" || arg == "pnpx" || arg == "dlx" && i > 0 && args[i-1] == "pnpm" {
			start = i + 1
			break
		}
	}
	return validateExecutionRawOptions(args, start, "--package")
}

func readPNPMRunExecutionRequest(command *cobra.Command, args []string) (application.ExecutionPreparationRequest, error) {
	if len(args) == 0 {
		return application.ExecutionPreparationRequest{}, errors.New("pnpm execution requires one public package or explicit command")
	}
	from, err := command.Flags().GetString("package")
	if err != nil || command.Flags().Changed("package") && from == "" {
		return application.ExecutionPreparationRequest{}, errors.New("explicit --package must be nonempty")
	}
	input, selected := args[0], ""
	if from != "" {
		input, selected = from, args[0]
	}
	reference, err := artifactnpm.ParseReference(input)
	if err != nil || artifactnpm.ValidatePNPMRunReference(reference) != nil {
		return application.ExecutionPreparationRequest{}, errors.New("pnpm execution requires a bare or exact public package without provisioning")
	}
	arguments := args[1:]
	if len(arguments) > 0 && arguments[0] == "--" {
		arguments = arguments[1:]
	}
	return application.NewExecutionPreparationRequest(reference, selected, arguments)
}

func AddPNPMRunExecution(root *cobra.Command, runner ExecutionRunner) error {
	if root == nil || runner == nil {
		return errors.New("pnpm execution CLI requires root and current workflow")
	}
	for _, operation := range []string{"dlx", "pnx", "pnpx"} {
		command, err := packageClientCommand(root, PNPMRunPackageClient, operation)
		if err != nil {
			return err
		}
		command.RunE = func(command *cobra.Command, args []string) error {
			request, err := readPNPMRunExecutionRequest(command, args)
			if err != nil {
				return err
			}
			return runExecutionRequest(command, runner, request)
		}
	}
	return nil
}
