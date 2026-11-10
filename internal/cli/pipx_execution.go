package cli

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func pipxRunExecutionDescriptor() packageClientDescriptor {
	d := uvRunExecutionDescriptor(false)
	d.id, d.path, d.use = PipxRunPackageClient, []string{"pipx", "run"}, "run [--spec <package>] <package-or-command> [--] [args...]"
	d.parents = []packageClientParent{{"Quarantine public Python tools with pipx", "pipx로 공개 Python 툴을 검역합니다"}}
	d.longEN = strings.ReplaceAll(uvRunExecutionEnglish, "--from", "--spec") + "\n\nSelection uses the pinned pipx with its UV backend. A primary-name request can select its sole declared console. The pipx.run entry-point group is unsupported."
	d.longKO = strings.ReplaceAll(uvRunExecutionKorean, "--from", "--spec") + "\n\n고정 pipx와 UV backend로 선택합니다. 패키지명 요청은 유일한 console을 선택할 수 있습니다. pipx.run entry-point group은 지원하지 않습니다."
	d.flags = func(command *cobra.Command) { configureRunExecutionFlags(command, "spec") }
	d.validate = func(command *cobra.Command, args []string) error {
		_, err := readPythonRunExecutionRequest(command, args, "spec")
		return err
	}
	d.validateRaw = func(args []string) error {
		start := -1
		pipx := false
		for i, arg := range args {
			if arg == "pipx" {
				pipx = true
			}
			if pipx && arg == "run" {
				start = i + 1
				break
			}
		}
		return validateExecutionRawOptions(args, start, "--spec")
	}
	d.examples = []string{"helox pipx run --show-output cowsay@6.1 -- -t hello", "helox --output json pipx run --spec python-slugify==8.0.4 slugify -- hello world"}
	d.flagText = map[string]packageClientParent{"spec": {"Public package when its console command differs", "console 명령 이름이 다를 때 선택할 공개 패키지"}, "show-output": d.flagText["show-output"]}
	return d
}

func AddPipxRunExecution(root *cobra.Command, runner ExecutionRunner) error {
	if root == nil || runner == nil {
		return errors.New("pipx execution CLI requires root and current workflow")
	}
	command, err := packageClientCommand(root, PipxRunPackageClient, "run")
	if err != nil {
		return err
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		request, err := readPythonRunExecutionRequest(command, args, "spec")
		if err != nil {
			return err
		}
		return runExecutionRequest(command, runner, request)
	}
	return nil
}
