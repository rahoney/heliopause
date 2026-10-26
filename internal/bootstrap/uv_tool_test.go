package bootstrap

import (
	"os"
	"testing"
)

func TestUVToolInstallServiceMissingBindingsDoNotPrepareRuntime(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)
	t.Setenv("PATH", root)
	if service, err := newUVToolInstallService(pythonClientBindings{}, root); err == nil || service != nil {
		t.Fatal("missing reviewed tool bindings accepted")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("missing bindings prepared runtime or cache")
	}
}
