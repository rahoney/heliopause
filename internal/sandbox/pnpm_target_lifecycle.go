package sandbox

import (
	"errors"
	"path"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

const pnpmTargetLifecycleCommand = `mkdir -p /tmp/.npm; HOME=/tmp npm_config_cache=/tmp/.npm npm_config_script_shell=/haa-runtime/haa-boundary npm --prefix /tmp/haa-pnpm-input/project rebuild --ignore-scripts=false --bin-links=false --foreground-scripts --offline --no-update-notifier --workspaces=false --no-audit --no-fund --dangerously-allow-all-scripts "$1"`

// This constrains a control-plane locator, never artifact identity or safety.
// The caller must select it from the fully verified frozen instance layout.
func validPNPMTargetLifecyclePath(target string) bool {
	const prefix = pnpmProjectDirectory + "/node_modules/.pnpm/"
	if len(target) > 512 || !strings.HasPrefix(target, prefix) || path.Clean(target) != target {
		return false
	}
	slot, name, found := strings.Cut(strings.TrimPrefix(target, prefix), "/node_modules/")
	if !found || len(slot) < 2 || len(slot) > artifactnpm.PNPMVirtualStoreDirMaxLength || !strings.Contains(slot[1:], "@") && !pnpmHashedTargetSlot(slot) || len(name) == 0 {
		return false
	}
	for _, value := range []string{slot, name} {
		for _, char := range value {
			if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("@+_.-/", char) {
				continue
			}
			return false
		}
	}
	if strings.Contains(slot, "/") || strings.Contains(target, "\\") {
		return false
	}
	parts := strings.Split(name, "/")
	if len(parts) == 1 {
		return !strings.HasPrefix(name, "@") && name != "." && name != ".."
	}
	return len(parts) == 2 && strings.HasPrefix(parts[0], "@") && len(parts[0]) > 1 && parts[1] != "" && parts[1] != "." && parts[1] != ".."
}

func pnpmHashedTargetSlot(slot string) bool {
	const prefix = artifactnpm.PNPMVirtualStoreDirMaxLength - 33
	if len(slot) != artifactnpm.PNPMVirtualStoreDirMaxLength || slot[prefix] != '_' {
		return false
	}
	for _, char := range slot[prefix+1:] {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

func pnpmTargetLifecycleExecArguments(container, target string) ([]string, error) {
	if !exactObservationContainerID(container) || !validPNPMTargetLifecyclePath(target) {
		return nil, errors.New("pnpm target lifecycle requires one fixed instance path and exact container")
	}
	return boundaryExecArguments(container, boundaryLaunchMode, "/bin/sh", "-ceu", pnpmTargetLifecycleCommand, "helox-pnpm-target", target), nil
}

func pnpmTargetLifecycleExpectedTopology() ([]observerMountExpectation, bool) {
	base, ok := observerExpectedTopology("npm-lifecycle")
	if !ok {
		return nil, false
	}
	return append(base, observerMountExpectation{pnpmProjectRoot, "workspace", "/tmp", "9p", false, false, false, false}), true
}
