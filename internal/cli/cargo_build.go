package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Cargo build has fixed default-target semantics and accepts no command flags
// or selectors from an artifact. The use case owns locked inputs and approval.
func AddCargoBuild(root *cobra.Command, builder GoModuleBuilder) error {
	if root == nil || builder == nil {
		return errors.New("cargo build requires a guarded observed build use case")
	}
	command := findLeaf(root, "cargo", "build")
	if command == nil {
		return errors.New("cargo build command is not registered")
	}
	command.Args = cobra.NoArgs
	command.RunE = func(command *cobra.Command, _ []string) error {
		install, err := currentGoProjectContext()
		if err != nil {
			return err
		}
		result, published, operationErr := builder.Build(contextOrBackground(command.Context()), install, "default")
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
			return errors.New("cargo build output publication is incomplete")
		}
		_, err = fmt.Fprintf(command.OutOrStdout(), "Build output: %s\n", published.RelativeDirectory())
		return err
	}
	return nil
}
