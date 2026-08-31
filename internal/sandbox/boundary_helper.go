package sandbox

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	boundaryHelperPath        = "/haa-runtime/haa-boundary"
	boundarySetprivPath       = "/usr/bin/setpriv"
	boundaryHelperMount       = "/haa-runtime:rw,exec,nosuid,nodev,size=1m,uid=0,gid=0,mode=0755"
	boundaryLaunchMode        = "--launch"
	boundaryPythonHandoffMode = "--handoff-python"
	boundaryELFHandoffMode    = "--handoff-elf"
	// Docker exec starts only this fixed, root-owned HAA helper as root. The
	// helper immediately uses setpriv before it execs every requested target.
	// No artifact-selected executable is a Docker exec entrypoint.
	boundaryBootstrapUser = "0:0"
)

const boundaryDemotionArguments = "--reuid=1000 --regid=1000 --clear-groups --inh-caps=-all --ambient-caps=-all --bounding-set=-all --no-new-privs --"

// boundaryHelper is installed by the trusted controller into a root-owned,
// container-private executable tmpfs. It has two deliberately asymmetric
// operations: launch establishes the Sentry boundary for a direct exec, while
// handoff only removes trust before artifact execution. The helper never
// interprets artifact input.
const boundaryHelper = "#!/bin/sh\nset -eu\ndemote() { exec " + boundarySetprivPath + " " + boundaryDemotionArguments + " \"$@\"; }\ncase \"${1-}\" in\n  --launch|--handoff-python|--handoff-elf) shift; demote \"$@\" ;;\n  -c) shift; demote /bin/sh -c \"$@\" ;;\n  *) exit 125 ;;\nesac\n"

// boundaryContainerCommand runs as the OCI init root solely long enough to
// install the fixed controller helper into its root-owned tmpfs. The helper is
// moved into place atomically before any user-1000 exec is accepted.
func boundaryContainerCommand() string {
	quoted := strings.ReplaceAll(boundaryHelper, "'", "'\\\"'\\\"'")
	return "set -eu; tmp=/haa-runtime/.haa-boundary.tmp; printf '%s' '" + quoted +
		"' > \"$tmp\"; chown 0:0 \"$tmp\"; chmod 0555 \"$tmp\"; mv \"$tmp\" " +
		boundaryHelperPath + "; exec " + boundarySetprivPath + " " + boundaryDemotionArguments + " /bin/sleep infinity"
}

const boundaryReadinessScript = "test \"$(id -u)\" = 1000; test \"$(id -g)\" = 1000; grep -Eq '^Uid:[[:space:]]+1000[[:space:]]+1000[[:space:]]+1000[[:space:]]+1000$' /proc/1/status; grep -Eq '^Gid:[[:space:]]+1000[[:space:]]+1000[[:space:]]+1000[[:space:]]+1000$' /proc/1/status; grep -Eq '^Groups:[[:space:]]*$' /proc/1/status; for field in CapInh CapPrm CapEff CapBnd CapAmb; do grep -Eq \"^${field}:[[:space:]]+0000000000000000$\" /proc/1/status; done; grep -Eq '^NoNewPrivs:[[:space:]]+1$' /proc/1/status; grep -Eq '^Uid:[[:space:]]+1000[[:space:]]+1000[[:space:]]+1000[[:space:]]+1000$' /proc/self/status; grep -Eq '^Gid:[[:space:]]+1000[[:space:]]+1000[[:space:]]+1000[[:space:]]+1000$' /proc/self/status; grep -Eq '^Groups:[[:space:]]*$' /proc/self/status; for field in CapInh CapPrm CapEff CapBnd CapAmb; do grep -Eq \"^${field}:[[:space:]]+0000000000000000$\" /proc/self/status; done; grep -Eq '^NoNewPrivs:[[:space:]]+1$' /proc/self/status"

func boundaryExecArguments(containerID, mode string, command ...string) []string {
	arguments := []string{"exec", "--user", boundaryBootstrapUser, containerID, boundaryHelperPath, mode}
	return append(arguments, command...)
}

func boundaryReadinessArguments(containerID string) []string {
	return boundaryExecArguments(containerID, boundaryLaunchMode, "/bin/sh", "-ceu", boundaryReadinessScript)
}

// awaitBoundaryHelper is a bounded readiness check. It never invokes an
// unwrapped docker exec: the first executable transition is the HAA helper.
func awaitBoundaryHelper(ctx context.Context, runner CommandRunner, containerID string) error {
	if runner == nil || !containerIDPattern.MatchString(containerID) {
		return errors.New("sandbox boundary helper runner is unavailable")
	}
	for attempt := 0; attempt < 20; attempt++ {
		if _, err := runner.Output(ctx, "docker", boundaryReadinessArguments(containerID)...); err == nil {
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
