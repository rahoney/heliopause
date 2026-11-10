package runtimeidentity_test

import (
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestDockerEngineSupported(t *testing.T) {
	for _, tc := range []struct {
		version string
		want    bool
	}{
		{runtimeidentity.DockerMinimumEngine, true},
		{"30.0.0", true},
		{"29.8.1", false},
		{"29.9.2-rc.1", false},
		{"29.9.2suffix", false},
		{"30.-1.0", false},
		{"030.0.0", false},
		{"+30.0.0", false},
		{"30.0.0\n", false},
		{"", false},
	} {
		if got := runtimeidentity.DockerEngineSupported(tc.version); got != tc.want {
			t.Errorf("DockerEngineSupported(%q) = %v, want %v", tc.version, got, tc.want)
		}
	}
}
