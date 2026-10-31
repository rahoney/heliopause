package cli

import (
	"context"
	"errors"
	"fmt"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

type GoModuleBuilder interface {
	Build(context.Context, domain.InstallContext, string) (domain.OperationResult, domain.PublishedProjectBuild, error)
}

func AddGoModuleBuild(root *cobra.Command, builder GoModuleBuilder) error {
	if root == nil || builder == nil {
		return errors.New("go build command requires a guarded build use case")
	}
	command := findLeaf(root, "go", "build")
	if command == nil {
		return errors.New("go build command is not registered")
	}
	command.Args = func(_ *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("go build requires one package selector")
		}
		return artifactgo.ValidateBuildPackage(args[0])
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		install, err := currentGoProjectContext()
		if err != nil {
			return err
		}
		progressContext, finishProgress := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finishProgress() }()
		result, published, operationErr := builder.Build(progressContext, install, args[0])
		progressErr := finishProgress()
		operationErr = errors.Join(operationErr, progressErr)
		if result.OperationID().String() != "" {
			if err := WriteHumanResult(command.OutOrStdout(), result); err != nil {
				return errors.Join(operationErr, err)
			}
		}
		if operationErr != nil {
			return operationErr
		}
		if code := ExitCode(result); code != 0 {
			return ExitError{Code: code}
		}
		if !published.Valid() {
			return errors.New("go build output publication is incomplete")
		}
		if p := presentationFor(command.OutOrStdout()); p.human() {
			p.line("%s %s", p.text("Build output:", "빌드 결과:"), safeTerminal(published.RelativeDirectory()))
			return p.writeErr
		}
		_, err = fmt.Fprintf(command.OutOrStdout(), "Build output: %s\n", published.RelativeDirectory())
		return err
	}
	return nil
}
