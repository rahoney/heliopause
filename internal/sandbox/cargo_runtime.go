package sandbox

import (
	"context"
	"encoding/hex"
	"errors"
	"runtime"
	"strings"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// CargoRuntime is infrastructure configuration, independent of project config,
// rustup shims, cargo output, and the caller's installed toolchain.
type CargoRuntime struct{ ImageReference, RustVersion, Architecture, Target string }

func PinnedCargoRuntime() CargoRuntime {
	return CargoRuntime{runtimeidentity.RustImageReference, runtimeidentity.RustVersion, runtimeidentity.RustArchitecture, runtimeidentity.RustTarget}
}

func (r CargoRuntime) CargoBinary() string {
	return "/usr/local/rustup/toolchains/" + r.RustVersion + "-" + r.Target + "/bin/cargo"
}

func (r CargoRuntime) RustcBinary() string {
	return "/usr/local/rustup/toolchains/" + r.RustVersion + "-" + r.Target + "/bin/rustc"
}

type CargoCapability struct {
	Available      bool
	LimitationCode string
	Runtime        CargoRuntime
}

func ProbeCargo(ctx context.Context, executor Executor) (CargoCapability, error) {
	return probeCargo(ctx, runtime.GOOS, runtime.GOARCH, executor)
}

func probeCargo(ctx context.Context, operatingSystem, architecture string, executor Executor) (CargoCapability, error) {
	locked := PinnedCargoRuntime()
	if ctx == nil {
		return CargoCapability{Runtime: locked}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return CargoCapability{Runtime: locked}, err
	}
	if architecture != locked.Architecture {
		return CargoCapability{LimitationCode: "M12_CARGO_LINUX_AMD64_ONLY", Runtime: locked}, nil
	}
	limitation, err := probeGVisorRuntime(ctx, operatingSystem, executor, "M12_CARGO_LINUX_AMD64_ONLY", "M12_CARGO_RUNTIME_UNAVAILABLE", "M12_CARGO_RUNTIME_VERSION_UNSUPPORTED")
	if err != nil || limitation != "" {
		return CargoCapability{LimitationCode: limitation, Runtime: locked}, err
	}
	image, err := executor.Output(ctx, "docker", "image", "inspect", locked.ImageReference, "--format", "{{.Id}} {{.Architecture}}")
	fields := strings.Fields(string(image))
	if err != nil || len(fields) != 2 || fields[1] != locked.Architecture || !strings.HasPrefix(fields[0], "sha256:") || len(fields[0]) != 71 || fields[0] != strings.ToLower(fields[0]) {
		return CargoCapability{LimitationCode: "M12_CARGO_IMAGE_UNAVAILABLE", Runtime: locked}, nil
	}
	if _, err := hex.DecodeString(fields[0][7:]); err != nil {
		return CargoCapability{LimitationCode: "M12_CARGO_IMAGE_UNAVAILABLE", Runtime: locked}, nil
	}
	return CargoCapability{Available: true, Runtime: locked}, nil
}
