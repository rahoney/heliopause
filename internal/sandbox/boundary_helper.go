package sandbox

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	boundaryHelperPath              = "/haa-runtime/haa-boundary"
	boundarySetprivPath             = "/usr/bin/setpriv"
	boundaryHelperMount             = "/haa-runtime:rw,exec,nosuid,nodev,size=1m,uid=0,gid=0,mode=0755"
	boundaryLaunchMode              = "--launch"
	boundaryPythonHandoffMode       = "--handoff-python"
	boundaryELFHandoffMode          = "--handoff-elf"
	boundaryNodeHandoffMode         = "--handoff-node"
	boundaryBunHandoffMode          = "--handoff-bun"
	boundaryOriginLaunchMode        = "--origin-launch"
	boundaryOriginPythonHandoffMode = "--origin-handoff-python"
	boundaryOriginELFHandoffMode    = "--origin-handoff-elf"
	boundaryOriginNodeHandoffMode   = "--origin-handoff-node"
	boundaryOriginBunHandoffMode    = "--origin-handoff-bun"
	// Docker exec starts only this fixed, root-owned HAA helper as root. The
	// helper immediately uses setpriv before it execs every root-entry target.
	// No artifact-selected executable is a Docker exec entrypoint.
	boundaryBootstrapUser = "0:0"
)

const boundaryDemotionArguments = "--reuid=1000 --regid=1000 --clear-groups --inh-caps=-all --ambient-caps=-all --bounding-set=-all --no-new-privs --"

const boundaryAlreadyDemotedValidation = `already_demoted() {
  uid= gid= groups= cap_inh= cap_prm= cap_eff= cap_bnd= cap_amb=
  while IFS=: read -r key value; do
    set -- $value
    case "$key" in
      Uid) [ "$#" -eq 4 ] && [ "$1" = 1000 ] && [ "$2" = 1000 ] && [ "$3" = 1000 ] && [ "$4" = 1000 ] && uid=1 ;;
      Gid) [ "$#" -eq 4 ] && [ "$1" = 1000 ] && [ "$2" = 1000 ] && [ "$3" = 1000 ] && [ "$4" = 1000 ] && gid=1 ;;
      Groups) [ "$#" -eq 0 ] && groups=1 ;;
      CapInh) [ "$#" -eq 1 ] && [ "$1" = 0000000000000000 ] && cap_inh=1 ;;
      CapPrm) [ "$#" -eq 1 ] && [ "$1" = 0000000000000000 ] && cap_prm=1 ;;
      CapEff) [ "$#" -eq 1 ] && [ "$1" = 0000000000000000 ] && cap_eff=1 ;;
      CapBnd) [ "$#" -eq 1 ] && [ "$1" = 0000000000000000 ] && cap_bnd=1 ;;
      CapAmb) [ "$#" -eq 1 ] && [ "$1" = 0000000000000000 ] && cap_amb=1 ;;
    esac
  done < /proc/self/status
  [ "$uid:$gid:$groups:$cap_inh:$cap_prm:$cap_eff:$cap_bnd:$cap_amb" = 1:1:1:1:1:1:1:1 ]
}`

// boundaryHelper is installed by the trusted controller into a root-owned,
// container-private executable tmpfs. It has two deliberately asymmetric
// operations: launch establishes the Sentry boundary for a direct exec, while
// handoff only removes trust before artifact execution. The helper never
// interprets artifact input.
const boundaryHelper = "#!/bin/sh\nset -eu\ndemote() { exec " + boundarySetprivPath + " " + boundaryDemotionArguments + " \"$@\"; }\nadmission() { [ \"${#1}\" -eq 64 ] || exit 125; case \"$1\" in *[!0123456789abcdef]*|'') exit 125 ;; esac; }\n" + boundaryAlreadyDemotedValidation + "\ncase \"${1-}\" in\n  --origin-launch) shift; admission \"${1-}\"; token=\"$1\"; shift; exec " + boundaryHelperPath + " --launch \"$token\" \"$@\" ;;\n  --origin-handoff-python) shift; admission \"${1-}\"; token=\"$1\"; shift; exec " + boundaryHelperPath + " --handoff-python \"$token\" \"$@\" ;;\n  --origin-handoff-elf) shift; admission \"${1-}\"; token=\"$1\"; shift; exec " + boundaryHelperPath + " --handoff-elf \"$token\" \"$@\" ;;\n  --origin-handoff-node) shift; admission \"${1-}\"; token=\"$1\"; shift; exec " + boundaryHelperPath + " --handoff-node \"$token\" \"$@\" ;;\n  --origin-handoff-bun) shift; admission \"${1-}\"; token=\"$1\"; shift; exec " + boundaryHelperPath + " --handoff-bun \"$token\" \"$@\" ;;\n  --launch|--handoff-python|--handoff-elf|--handoff-node|--handoff-bun) shift; admission \"${1-}\"; shift; demote \"$@\" ;;\n  -c) shift; already_demoted; exec /bin/sh -c \"$@\" ;;\n  *) exit 125 ;;\nesac\n"

// boundaryContainerCommand runs as the OCI init root solely long enough to
// install the fixed controller helper into its root-owned tmpfs. The helper is
// moved into place atomically before any user-1000 exec is accepted.
func boundaryContainerCommand() string {
	quoted := strings.ReplaceAll(boundaryHelper, "'", "'\\\"'\\\"'")
	return "set -eu; tmp=/haa-runtime/.haa-boundary.tmp; printf '%s' '" + quoted +
		"' > \"$tmp\"; chown 0:0 \"$tmp\"; chmod 0555 \"$tmp\"; mv \"$tmp\" " +
		boundaryHelperPath + "; exec " + boundarySetprivPath + " " + boundaryDemotionArguments + " /bin/sleep infinity"
}

// Reuse the credential validator and the existing CONTROL grep readers.
// Capture both fixed statuses once, then validate their fields with builtins.
var boundaryReadinessScript = func() string {
	reader := "done <<HAA_STATUS\n$(grep -E '^(Uid|Gid|Groups|CapInh|CapPrm|CapEff|CapBnd|CapAmb):' /proc/self/status)\nHAA_STATUS"
	self := strings.Replace(boundaryAlreadyDemotedValidation, "done < /proc/self/status", reader, 1)
	init := strings.Replace(self, "already_demoted()", "init_demoted()", 1)
	init = strings.Replace(init, "/proc/self/status", "/proc/1/status", 1)
	return init + "\n" + self + "\ninit_demoted\nalready_demoted"
}()

func boundaryOriginMode(mode string) string {
	switch mode {
	case boundaryLaunchMode:
		return boundaryOriginLaunchMode
	case boundaryPythonHandoffMode:
		return boundaryOriginPythonHandoffMode
	case boundaryELFHandoffMode:
		return boundaryOriginELFHandoffMode
	case boundaryNodeHandoffMode:
		return boundaryOriginNodeHandoffMode
	case boundaryBunHandoffMode:
		return boundaryOriginBunHandoffMode
	default:
		panic("invalid Docker boundary mode")
	}
}

func boundaryExecArguments(containerID, mode string, command ...string) []string {
	arguments := []string{"exec", "--user", boundaryBootstrapUser, containerID, boundaryHelperPath, boundaryOriginMode(mode)}
	return append(arguments, command...)
}

func boundaryInputExecArguments(containerID, mode string, command ...string) []string {
	arguments := []string{"exec", "-i", "--user", boundaryBootstrapUser, containerID, boundaryHelperPath, boundaryOriginMode(mode)}
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
		} else if errors.Is(err, errObserverAuthorityAmbiguous) {
			// An ARM/CANCEL uncertainty poisons this registration. Retrying the
			// same container/session cannot re-establish its ordering guarantee.
			return errors.New("sandbox boundary helper is unavailable")
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
