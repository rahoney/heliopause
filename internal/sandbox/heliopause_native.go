package sandbox

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// HeliopauseNativeBackend selects the fixed startup probe for the current
// release's PNPM member. Every member still undergoes ordinary ARTIFACT
// isolation and observation; selecting a probe confers no trusted-tool role.
type HeliopauseNativeBackend struct{ elf *GitHubELFBackend }

func NewLinuxHeliopauseNativeBackendWithExecutor(intakeRoot string, executor TrustedExecutor, observer TraceObserver) (*HeliopauseNativeBackend, error) {
	elf, err := newLinuxGitHubELFBackend(intakeRoot, executor, observer)
	if err != nil {
		return nil, err
	}
	return &HeliopauseNativeBackend{elf: elf}, nil
}

func (b *HeliopauseNativeBackend) Execute(ctx context.Context, request domain.SandboxRequest) (domain.SandboxResult, error) {
	if b == nil || b.elf == nil || ctx == nil {
		return domain.SandboxResult{}, errors.New("heliopause native sandbox is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return domain.SandboxResult{}, err
	}
	artifact := request.Artifact()
	if artifact.Identity().Variant() != runtimeidentity.PNPMNativeAssetName {
		return b.elf.Execute(ctx, request)
	}
	if artifact.Identity().Source().String() != "github-release" ||
		artifact.Identity().Name() != "rahoney-heliopause" ||
		artifact.Digest().String() != runtimeidentity.PNPMToolSHA256 ||
		artifact.SizeBytes() != runtimeidentity.PNPMToolSize {
		return domain.SandboxResult{}, errors.New("heliopause PNPM probe requires the exact current native member")
	}
	return b.elf.executeELF(ctx, request, "github-release", "github-elf", b.elf.introduce, b.probePNPM)
}

func (b *HeliopauseNativeBackend) probePNPM(ctx context.Context, container string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	runner, ok := b.elf.runner.(discardCommandRunner)
	if !ok {
		return "", errors.New("heliopause native bounded discard runner is unavailable")
	}
	err := runner.RunDiscard(ctx, "docker", boundaryExecArguments(container, boundaryELFHandoffMode, "/work/artifact", "--version")...)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	// Output and a nonzero help status cannot attest a successful probe.
	return "", err
}
