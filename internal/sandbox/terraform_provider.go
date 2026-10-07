package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os/exec"

	artifactterraform "github.com/rahoney/heliopause/internal/artifact/terraformprovider"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// TerraformProviderBackend observes the exact rehashed package ELF with a
// single fixed -help invocation. It performs no Terraform RPC, schema query,
// cloud operation or Host/provider execution outside the isolated lifecycle.
type TerraformProviderBackend struct{ elf *GitHubELFBackend }

func NewLinuxTerraformProviderBackend(intake string, executor TrustedExecutor, observer TraceObserver) (*TerraformProviderBackend, error) {
	elf, err := newLinuxGitHubELFBackend(intake, executor, observer)
	if err != nil {
		return nil, err
	}
	return &TerraformProviderBackend{elf}, nil
}

func (b *TerraformProviderBackend) Execute(ctx context.Context, request domain.SandboxRequest) (domain.SandboxResult, error) {
	if b == nil || b.elf == nil {
		return domain.SandboxResult{}, errors.New("terraform provider backend is unavailable")
	}
	return b.elf.executeELF(ctx, request, "terraform-registry", "terraform-provider", b.introduce, b.probeTerminal)
}

// The fixed help experiment accepts external exit 0 (help) or 1 (plugin host
// required). Neither proves RPC behavior. Exit 1 requires the existing exact
// consumed direct-exec admission; stdout, error strings and bare ExitErrors
// cannot supply this authority. Complete observation and cleanup remain required.
func (b *TerraformProviderBackend) probeTerminal(ctx context.Context, container string) (string, error) {
	runner, ok := b.elf.runner.(discardCommandRunner)
	if !ok {
		return "", errors.New("terraform bounded discard runner is unavailable")
	}
	err := runner.RunDiscard(ctx, "docker", boundaryExecArguments(container, boundaryELFHandoffMode, "/work/artifact", "-help")...)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err == nil && ctx.Err() == nil {
		return "terraform-help-exit-0", nil
	}
	var observed *observedDirectExecExit
	var exit *exec.ExitError
	if !errors.Is(err, errObserverAuthorityAmbiguous) && errors.As(err, &observed) && errors.As(observed, &exit) && exit.ExitCode() == 1 {
		return "terraform-help-exit-1", nil
	}
	return "", err
}

// The collector's bounded kind counts contain accepted records before a fault.
// Preserve only suspicious facts; the caller retains INCOMPLETE and the first
// observer failure. These partial counts cannot attest coverage or completion.
func retainedELFProbeFacts(counts map[string]uint64) []domain.SandboxObservation {
	var facts []domain.SandboxObservation
	for _, kind := range []string{"filesystem-outside-workspace", "honeytoken-access", "network-attempt", "process-exec-unexpected"} {
		count := counts[kind]
		if count == 0 || count > domain.MaximumObservationSummaryCount {
			continue
		}
		category, subject, ok := traceObservation(kind)
		if !ok {
			continue
		}
		fact, err := domain.NewCountedSandboxObservation(category, subject, count)
		if err == nil {
			facts = append(facts, fact)
		}
	}
	return facts
}

func (b *TerraformProviderBackend) introduce(ctx context.Context, container string, artifact domain.AcquiredArtifact) error {
	bundle, err := artifactterraform.ReadIntake(b.elf.intakeRoot, artifact)
	if err != nil {
		return err
	}
	reference, err := artifactterraform.IdentityReference(artifact.Identity())
	if err != nil {
		return err
	}
	contents, err := artifactterraform.InspectPackage(ctx, bundle, reference.Locator())
	if err != nil {
		return err
	}
	input, ok := b.elf.runner.(inputCommandRunner)
	if !ok {
		return errors.New("terraform provider stream runner is unavailable")
	}
	return input.RunInput(ctx, bytes.NewReader(contents.Files[contents.Executable]), "docker", boundaryInputExecArguments(container, boundaryLaunchMode, "/bin/sh", "-ceu", "umask 077; cat > /work/artifact; chmod 500 /work/artifact")...)
}
