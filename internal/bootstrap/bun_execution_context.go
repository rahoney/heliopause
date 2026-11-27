package bootstrap

import (
	"errors"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func newBunProgramExecutionContext(operation domain.OperationID, resolution domain.DependencyResolution, set domain.VerifiedSet, proof artifactnpm.BunExecutionEntry, arguments []string, force bool) (domain.ExecutionContext, error) {
	if !proof.Valid() || !force && !proof.BunShebang() && !proof.NodeCompatible() {
		return domain.ExecutionContext{}, errors.New("bun execution requires a measured declared entry or an explicit Bun runtime")
	}
	runtime := runtimeidentity.NodeImageReference
	if force || proof.BunShebang() {
		runtime = sandbox.BunExecutionRuntimeIdentity()
	}
	return newRuntimeExecutionContext(operation, resolution, set, proof.Entry(), arguments, runtime)
}
