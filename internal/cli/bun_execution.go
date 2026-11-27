package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/spf13/cobra"
)

func bunRunExecutionDescriptor(nested bool) packageClientDescriptor {
	d := pnpmRunExecutionDescriptor("pnx")
	d.id, d.path, d.use = BunRunPackageClient, []string{"bunx"}, "bunx [--package <package>] <package-or-command> [--] [args...]"
	d.shortEN, d.shortKO = "Quarantine and run a public npm entry selected by Bun", "Bun으로 선택한 공개 npm entry를 검역한 뒤 격리 실행합니다"
	d.longEN = "Select one bare or exact public package with pinned Bun, freshly quarantine every native dependency, then run its measured entry in a bounded read-only sandbox. Use --package <package> <command> for multiple bins or a command different from the package basename. Closed Node and Bun shebangs select their pinned runtime. Use --bun to run a verified JavaScript or TypeScript entry with pinned Bun; a TypeScript entry without a runtime declaration requires --bun. Runtime provisioning, project/PATH/global-cache lookup, shell commands and alternate registries are unavailable. --show-output displays private program streams as ASCII-escaped data."
	d.longKO = "고정 Bun으로 공개 패키지 이름 또는 정확한 버전 하나를 선택하고 모든 native 의존성을 새로 검역한 뒤 원본을 측정한 entry를 제한된 읽기 전용 sandbox에서 실행합니다. bin이 여러 개이거나 명령이 패키지 basename과 다르면 --package <패키지> <명령>으로 명시하세요. 닫힌 Node/Bun shebang은 각각 고정 runtime을 선택합니다. --bun은 검증된 JavaScript·TypeScript entry를 고정 Bun으로 실행하며, runtime 선언이 없는 TypeScript entry는 --bun이 필요합니다. runtime 설치·프로젝트/PATH/전역 cache 검색·shell·다른 registry는 지원하지 않습니다. --show-output은 비공개 프로그램 출력을 ASCII escape 데이터로 표시합니다."
	d.validate = func(command *cobra.Command, args []string) error {
		_, err := readBunRunExecutionRequest(command, args)
		return err
	}
	d.flags = func(command *cobra.Command) {
		configureRunExecutionFlags(command, "package")
		command.Flags().Bool("bun", false, "Use pinned Bun for the selected JavaScript or TypeScript entry")
	}
	d.flagText["bun"] = packageClientParent{"Run the selected JavaScript or TypeScript entry with pinned Bun", "선택한 JavaScript·TypeScript entry를 고정 Bun으로 실행합니다"}
	d.validateRaw = validateBunRunExecutionRawArgs
	d.examples = []string{"helox bunx --package typescript@5.9.3 --show-output tsc -- --help", "helox --output json bun x semver@7.8.5 -- 1.2.3"}
	if nested {
		d.path, d.use = []string{"bun", "x"}, "x [--package <package>] <package-or-command> [--] [args...]"
		d.parents = []packageClientParent{{"Quarantine project packages with Bun", "Bun으로 프로젝트 패키지를 검역합니다"}}
	}
	return d
}

func validateBunRunExecutionRawArgs(args []string) error {
	start := -1
	for i, arg := range args {
		if arg == "bunx" || arg == "x" && i > 0 && args[i-1] == "bun" {
			start = i + 1
			break
		}
	}
	if err := validateExecutionRawOptions(args, start, "--package"); err != nil {
		return err
	}
	count := 0
	for i := start; i >= 0 && i < len(args); i++ {
		arg := args[i]
		if arg == "--" || !strings.HasPrefix(arg, "-") {
			break
		}
		name, _, equals := strings.Cut(arg, "=")
		if name == "--bun" {
			count++
			if count > 1 {
				return errors.New("bun runtime selection must not repeat")
			}
		}
		if !equals && (name == "--package" || name == "--lang" || name == "--output" || name == "--color" || name == "--progress") {
			i++
		}
	}
	return nil
}

func readBunRunExecutionRequest(command *cobra.Command, args []string) (application.ExecutionPreparationRequest, error) {
	return readNativeNodeRunExecutionRequest(command, args, artifactnpm.ValidateBunRunReference, "bun")
}

func AddBunRunExecution(root *cobra.Command, runner ExecutionRunner) error {
	if root == nil || runner == nil {
		return errors.New("bun execution CLI requires root and current workflow")
	}
	for _, operation := range []string{"bunx", "x"} {
		command, err := packageClientCommand(root, BunRunPackageClient, operation)
		if err != nil {
			return err
		}
		command.RunE = func(command *cobra.Command, args []string) error {
			request, err := readBunRunExecutionRequest(command, args)
			if err != nil {
				return err
			}
			selected := runner
			force, err := command.Flags().GetBool("bun")
			if err != nil {
				return err
			}
			if force {
				capability, ok := runner.(interface {
					RunBun(context.Context, application.ExecutionPreparationRequest) (application.ExecutionResult, error)
				})
				if !ok {
					return errors.New("bun program execution requires its pinned runtime consumer")
				}
				selected = bunProgramRunner{capability}
			}
			return runExecutionRequest(command, selected, request)
		}
	}
	return nil
}

type bunProgramRunner struct {
	current interface {
		RunBun(context.Context, application.ExecutionPreparationRequest) (application.ExecutionResult, error)
	}
}

func (r bunProgramRunner) Run(ctx context.Context, request application.ExecutionPreparationRequest) (application.ExecutionResult, error) {
	return r.current.RunBun(ctx, request)
}
