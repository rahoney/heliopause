// Package bootstrap owns the Heliopause composition root.
package bootstrap

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/application"
	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

// Run constructs and executes the CLI with process-owned dependencies.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	command, err := cli.New(stdout, stderr)
	if err != nil {
		return err
	}
	if len(args) > 0 && args[0] == "npm" {
		cacheRoot, err := os.UserCacheDir()
		if err != nil {
			return err
		}
		root := filepath.Join(cacheRoot, "heliopause")
		artifact, err := artifactnpm.NewPublicResolver(filepath.Join(root, "intake"))
		if err != nil {
			return err
		}
		staticInspection, err := inspectionnpm.NewInspector(filepath.Join(root, "intake"))
		if err != nil {
			return err
		}
		probedSandbox, err := sandbox.NewProbedSandbox(sandbox.Probe)
		if err != nil {
			return err
		}
		dynamicInspection, err := inspectionnpm.NewDynamicInspector(probedSandbox)
		if err != nil {
			return err
		}
		inspection, err := inspectionnpm.NewCompositeInspector(staticInspection, dynamicInspection)
		if err != nil {
			return err
		}
		evidence, err := local.NewStore(filepath.Join(root, "evidence"))
		if err != nil {
			return err
		}
		service, err := application.NewInspectService(artifact, verificationnpm.IntegrityVerifier{}, inspection, evidence, policy.M3{}, domain.NewOperationID, domain.NewRunID)
		if err != nil {
			return err
		}
		if err := cli.AddNPMInspect(command, service); err != nil {
			return err
		}
	}
	command.SetArgs(args)

	return command.ExecuteContext(ctx)
}
