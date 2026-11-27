package runtimeidentity

import "fmt"

// DockerEngineSupported requires a canonical stable engine version at or above
// the compiled security minimum. Prerelease and malformed values fail closed.
func DockerEngineSupported(version string) bool {
	var major, minor, patch, minimumMajor, minimumMinor, minimumPatch int
	if _, err := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch); err != nil ||
		major < 0 || minor < 0 || patch < 0 || fmt.Sprintf("%d.%d.%d", major, minor, patch) != version {
		return false
	}
	if _, err := fmt.Sscanf(DockerMinimumEngine, "%d.%d.%d", &minimumMajor, &minimumMinor, &minimumPatch); err != nil {
		return false
	}
	return major > minimumMajor || major == minimumMajor && (minor > minimumMinor || minor == minimumMinor && patch >= minimumPatch)
}
