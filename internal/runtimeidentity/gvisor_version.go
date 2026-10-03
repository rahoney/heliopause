package runtimeidentity

import (
	"regexp"
	"strings"
)

// ExpectedGVisorRunscVersion is the source-built stamp produced by the
// canonical tagless, patched checkout. The release label is metadata, not
// the version embedded in this binary.
func ExpectedGVisorRunscVersion() string {
	return GVisorCommit[:12] + "-dirty"
}

var runscSpecVersion = regexp.MustCompile(`^spec: [0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.-]+)?$`)

// ValidateGVisorRunscVersionOutput accepts the complete two-line output of
// the pinned runsc --version command, including its OCI spec version line.
func ValidateGVisorRunscVersionOutput(output []byte) bool {
	if len(output) == 0 || len(output) > 256 || output[len(output)-1] != '\n' {
		return false
	}
	lines := strings.Split(string(output[:len(output)-1]), "\n")
	return len(lines) == 2 &&
		lines[0] == "runsc version "+ExpectedGVisorRunscVersion() &&
		runscSpecVersion.MatchString(lines[1])
}
