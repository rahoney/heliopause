//go:build linux

package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type uvPublicCompleteSourceCase struct {
	name           string
	format         artifactpypi.UVSyncSourceFormat
	body           []byte
	allow          bool
	entries        int
	closureFailure bool
}

func uvPublicCompleteSourceCases() ([]uvPublicCompleteSourceCase, error) {
	body, err := os.ReadFile(filepath.Join("..", "artifact", "pypi", "testdata", "uv-0.13.4-requests.pylock.toml"))
	if err != nil {
		return nil, err
	}
	lock, err := artifactpypi.ParseUVProjectLock(body, PinnedPythonRuntime().PythonVersion)
	if err != nil {
		return nil, err
	}
	var complete, roots strings.Builder
	rootOnly := ""
	for _, p := range lock.Packages() {
		fmt.Fprintf(&complete, "%s==%s\n", p.Name, p.Version)
		if p.Name == "idna" || p.Name == "certifi" {
			fmt.Fprintf(&roots, "%s==%s\n", p.Name, p.Version)
		}
		if p.Name == "requests" {
			rootOnly = fmt.Sprintf("%s==%s\n", p.Name, p.Version)
		}
	}
	if rootOnly == "" {
		return nil, errors.New("canonical UV fixture lost requests")
	}
	return []uvPublicCompleteSourceCase{
		{"two-roots", artifactpypi.UVSyncRequirements, []byte(roots.String()), false, 2, false},
		{"complete-closure", artifactpypi.UVSyncRequirements, []byte(complete.String()), false, len(lock.Packages()), false},
		{"missing-dependency", artifactpypi.UVSyncRequirements, []byte(rootOnly), false, 0, true},
		{"explicit-empty-requirements", artifactpypi.UVSyncRequirements, nil, true, 0, false},
		{"explicit-empty-pylock", artifactpypi.UVSyncPylock, uvExplicitEmptyLock(), false, 0, false},
	}, nil
}

// Invoked by the already-required actual public UV integration entry. Reuse
// its registered caller, pinned runtime, typed policy and observer supervisor.
func testUVPublicCompleteSelections(t *testing.T, resolver *UVResolver) {
	t.Helper()
	cases, err := uvPublicCompleteSourceCases()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range cases {
		input, err := artifactpypi.ParseUVSyncInput(item.format, item.body, item.allow, PinnedPythonRuntime().PythonVersion)
		if err != nil {
			t.Fatal(err)
		}
		original, err := NewUVSyncOriginalControls(input)
		if err != nil {
			t.Fatal(err)
		}
		target, err := domain.NewInstallTarget(filepath.Join(t.TempDir(), "venv"))
		if err != nil {
			t.Fatal(err)
		}
		install, err := domain.NewPythonVenvInstallContext(target)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		result, err := resolver.ResolveProjectLockUpdate(ctx, install, original, "")
		cancel()
		if _, statErr := os.Lstat(target.String()); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("complete resolver touched user target: %v", statErr)
		}
		if item.closureFailure {
			if err == nil || err.Error() != "uv selected dependency constraint is unsupported or unsatisfied" || result.Valid() {
				t.Fatalf("actual incomplete source did not fail independent closure: %v", err)
			}
			t.Logf("uv_public_complete_source case=%s independent_closure_rejected=true", item.name)
			continue
		}
		if err != nil {
			t.Fatalf("actual complete UV source %s failed: %v", item.name, err)
		}
		snapshot := result.Snapshot()
		if !result.Valid() || result.HasRequestedResolution() || snapshot.Context() != install || snapshot.DependencyFree() != (item.entries == 0) || len(snapshot.Dependencies()) != item.entries || len(result.SelectedControls()) != 4 {
			t.Fatalf("actual complete UV source %s retained a wrong selection", item.name)
		}
		for _, artifact := range snapshot.Dependencies() {
			if artifact.Identity().Source() != artifactpypi.PublicPyPIProfile().Source() || artifact.Identity().Variant() != "wheel" || !strings.HasPrefix(artifact.DeclaredIntegrity(), "sha256:") {
				t.Fatal("complete source lost exact public wheel binding")
			}
		}
		t.Logf("uv_public_complete_source case=%s entries=%d dependency_free=%t controls=%d runtime=%s", item.name, len(snapshot.Dependencies()), snapshot.DependencyFree(), len(result.SelectedControls()), UVResolverRuntimeIdentity())
	}
}

// This same input constructor also runs without a native runtime so compiled
// test caller cwd can be checked before the privileged integration setup.
func TestUVPublicCompleteSourceInputs(t *testing.T) {
	cases, err := uvPublicCompleteSourceCases()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 5 {
		t.Fatalf("complete source cases=%d, want 5", len(cases))
	}
	for _, item := range cases {
		input, err := artifactpypi.ParseUVSyncInput(item.format, item.body, item.allow, PinnedPythonRuntime().PythonVersion)
		if err != nil || input.ShouldSkip() {
			t.Fatalf("invalid actual source input %s: %v", item.name, err)
		}
		if controls, err := NewUVSyncOriginalControls(input); err != nil || len(controls) != 4 {
			t.Fatalf("actual source controls %s: %v", item.name, err)
		}
	}
}
