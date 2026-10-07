package sandbox

import (
	"context"
	"encoding/hex"
	"errors"
	"runtime"
	"strings"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// GoRuntime is the immutable M12 public-module resolver/build toolchain basis.
// These values are infrastructure configuration, never project/tool output.
type GoRuntime struct{ ImageReference, GoVersion, Architecture string }

func PinnedGoRuntime() GoRuntime {
	return GoRuntime{runtimeidentity.GoImageReference, runtimeidentity.GoVersion, runtimeidentity.GoArchitecture}
}

type GoCapability struct {
	Available      bool
	LimitationCode string
	Runtime        GoRuntime
}

func ProbeGo(ctx context.Context, executor Executor) (GoCapability, error) {
	return probeGo(ctx, runtime.GOOS, runtime.GOARCH, executor)
}

func probeGo(ctx context.Context, operatingSystem, architecture string, executor Executor) (GoCapability, error) {
	locked := PinnedGoRuntime()
	if ctx == nil {
		return GoCapability{Runtime: locked}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return GoCapability{Runtime: locked}, err
	}
	if architecture != locked.Architecture {
		return GoCapability{LimitationCode: "M12_GO_LINUX_AMD64_ONLY", Runtime: locked}, nil
	}
	limitation, err := probeGVisorRuntime(ctx, operatingSystem, executor, "M12_GO_LINUX_AMD64_ONLY", "M12_GO_RUNTIME_UNAVAILABLE", "M12_GO_RUNTIME_VERSION_UNSUPPORTED")
	if err != nil || limitation != "" {
		return GoCapability{LimitationCode: limitation, Runtime: locked}, err
	}
	image, err := executor.Output(ctx, "docker", "image", "inspect", locked.ImageReference, "--format", "{{.Id}} {{.Architecture}}")
	fields := strings.Fields(string(image))
	if err != nil || len(fields) != 2 || !validGoImageID(fields[0]) || fields[1] != locked.Architecture {
		return GoCapability{LimitationCode: "M12_GO_IMAGE_UNAVAILABLE", Runtime: locked}, nil
	}
	return GoCapability{Available: true, Runtime: locked}, nil
}

func validGoImageID(id string) bool {
	if !strings.HasPrefix(id, "sha256:") || len(id) != 71 || id != strings.ToLower(id) {
		return false
	}
	_, err := hex.DecodeString(id[len("sha256:"):])
	return err == nil
}
