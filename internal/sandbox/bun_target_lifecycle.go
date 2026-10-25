package sandbox

import (
	"errors"
	"path"
	"strings"
)

const bunTargetLifecycleCommand = `mkdir -p /tmp/.npm; HOME=/tmp npm_config_cache=/tmp/.npm npm_config_script_shell=/haa-runtime/haa-boundary npm --prefix /tmp/haa-bun-input/project rebuild --ignore-scripts=false --bin-links=false --foreground-scripts --offline --no-update-notifier --workspaces=false --no-audit --no-fund --dangerously-allow-all-scripts "$1"`

// This validates a control locator selected from a bound layout, never identity.
func validBunTargetLifecyclePath(target string) bool {
	const prefix = bunProjectDirectory + "/node_modules/"
	if len(target) > 512 || !strings.HasPrefix(target, prefix) || path.Clean(target) != target {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(target, prefix), "/")
	segment := func(value string) bool {
		if value == "" || value == "." || value == ".." {
			return false
		}
		for _, c := range value {
			if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("_.-", c) {
				continue
			}
			return false
		}
		return true
	}
	for len(parts) > 0 {
		if strings.HasPrefix(parts[0], "@") {
			if len(parts) < 2 || !segment(strings.TrimPrefix(parts[0], "@")) || !segment(parts[1]) {
				return false
			}
			parts = parts[2:]
		} else {
			if !segment(parts[0]) {
				return false
			}
			parts = parts[1:]
		}
		if len(parts) == 0 {
			return true
		}
		if len(parts) < 2 || parts[0] != "node_modules" {
			return false
		}
		parts = parts[1:]
	}
	return false
}

func bunTargetLifecycleExecArguments(container, target string) ([]string, error) {
	if !exactObservationContainerID(container) || !validBunTargetLifecyclePath(target) {
		return nil, errors.New("bun target lifecycle requires one fixed instance path and exact container")
	}
	return boundaryExecArguments(container, boundaryLaunchMode, "/bin/sh", "-ceu", bunTargetLifecycleCommand, "helox-bun-target", target), nil
}

func bunTargetLifecycleExpectedTopology() ([]observerMountExpectation, bool) {
	base, ok := observerExpectedTopology("npm-lifecycle")
	if !ok {
		return nil, false
	}
	return append(base, observerMountExpectation{bunProjectRoot, "workspace", "/tmp", "9p", false, false, false, false}), true
}
