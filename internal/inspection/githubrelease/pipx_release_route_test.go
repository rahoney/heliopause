package githubrelease

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type pipxNativeRouteSpy struct{ calls int }

func (s *pipxNativeRouteSpy) Inspect(context.Context, domain.AcquiredArtifact) (domain.InspectionReport, error) {
	s.calls++
	return domain.InspectionReport{}, errors.New("ZIP must not reach native ELF inspector")
}

func TestPipxSelectedReleaseCannotFallbackToNativeOrMissingRuntime(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0", "helox-release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	value["version"] = "v9.8.7"
	assets := value["assets"].([]any)
	for name, role := range map[string]string{"haa-release-installer-linux-amd64": "release-installer", "pnpm-linux-amd64": "pnpm", "pnpm-NOTICE.txt": "release-metadata", "bun-linux-amd64": "bun", "bun-NOTICE.txt": "release-metadata", runtimeidentity.PipxRuntimeAssetName: "pipx-runtime", "pipx-NOTICE.txt": "release-metadata"} {
		hash := strings.Repeat("a", 64)
		size := int64(123)
		if name == runtimeidentity.PipxRuntimeAssetName {
			hash = runtimeidentity.PipxToolSHA256
			size = runtimeidentity.PipxToolSize
		}
		if name == "pipx-NOTICE.txt" {
			hash = runtimeidentity.PipxNoticesSHA256
			size = runtimeidentity.PipxNoticesSize
		}
		assets = append(assets, map[string]any{"name": name, "role": role, "platform": "linux/amd64", "sha256": hash, "size": size})
	}
	value["assets"] = assets
	body, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := releaseinstall.ParseReleaseManifest(body)
	if err != nil {
		t.Fatal(err)
	}
	native := &pipxNativeRouteSpy{}
	inspector, err := NewReleaseSetInspector(t.TempDir(), func() (releaseinstall.ReleaseManifest, error) { return manifest, nil }, native, nil)
	if err != nil {
		t.Fatal(err)
	}
	artifact, _, _ := pipxReportFixture(t)
	report, err := inspector.Inspect(context.Background(), artifact)
	if err != nil || native.calls != 0 || report.Execution().ID().String() != pipxRuntimeCheck || !report.Execution().Required() || report.Execution().Status() != domain.ExecutionIncomplete || len(report.Evidence()) != 0 {
		t.Fatal("missing ZIP runner fell back or completed", err)
	}
}
