package sandbox

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	boundaryHelperPath        = "/haa-runtime/haa-boundary"
	boundaryHelperMount       = "/haa-runtime:rw,exec,nosuid,nodev,size=1m,uid=0,gid=0,mode=0755"
	boundaryLaunchMode        = "--launch"
	boundaryPythonHandoffMode = "--handoff-python"
	boundaryELFHandoffMode    = "--handoff-elf"
	boundaryExecUser          = "1000:1000"
)

// boundaryHelper is installed by the trusted controller into a root-owned,
// container-private executable tmpfs. It has two deliberately asymmetric
// operations: launch establishes the Sentry boundary for a direct exec, while
// handoff only removes trust before artifact execution. The helper never
// interprets artifact input.
const boundaryHelper = "#!/bin/sh\nset -eu\ncase \"${1-}\" in\n  --launch) shift; exec \"$@\" ;;\n  --handoff-python|--handoff-elf) shift; exec \"$@\" ;;\n  -c) exec /bin/sh \"$@\" ;;\n  *) exit 125 ;;\nesac\n"

// boundaryContainerCommand runs as the OCI init root solely long enough to
// install the fixed controller helper into its root-owned tmpfs. The helper is
// moved into place atomically before any user-1000 exec is accepted.
func boundaryContainerCommand() string {
	quoted := strings.ReplaceAll(boundaryHelper, "'", "'\\\"'\\\"'")
	return "set -eu; tmp=/haa-runtime/.haa-boundary.tmp; printf '%s' '" + quoted +
		"' > \"$tmp\"; chown 0:0 \"$tmp\"; chmod 0555 \"$tmp\"; mv \"$tmp\" " +
		boundaryHelperPath + "; exec sleep infinity"
}

func boundaryExecArguments(containerID, mode string, command ...string) []string {
	arguments := []string{"exec", "--user", boundaryExecUser, containerID, boundaryHelperPath, mode}
	return append(arguments, command...)
}

// awaitBoundaryHelper is a bounded readiness check. It never invokes an
// unwrapped docker exec: the first executable transition is the HAA helper.
func awaitBoundaryHelper(ctx context.Context, runner CommandRunner, containerID string) error {
	if runner == nil || !containerIDPattern.MatchString(containerID) {
		return errors.New("sandbox boundary helper runner is unavailable")
	}
	for attempt := 0; attempt < 20; attempt++ {
		if _, err := runner.Output(ctx, "docker", boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/true")...); err == nil {
			return nil
		}
		if attempt == 19 {
			break
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.New("sandbox boundary helper is unavailable")
		case <-timer.C:
		}
	}
	return errors.New("sandbox boundary helper is unavailable")
}
