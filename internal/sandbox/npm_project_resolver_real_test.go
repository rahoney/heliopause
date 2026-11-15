//go:build linux && realcorpus

package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

// Native selection DATA only: public archive binding and whole snapshot are
// checked, but this parent does not claim inspection or Host installation.
func TestNPMPublicWholeProjectSelectionIntegration(t *testing.T) {
	if os.Getenv("HELOX_NPM_PROJECT_SELECTION_INTEGRATION") != "1" {
		t.Skip("requires declared protected npm whole-project selection integration")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user client and protected infrastructure")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	executor, err := hosttool.NewSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	}()
	supervisor := integrationObserverSupervisor(t)
	defer func() {
		if err := supervisor.Close(); err != nil {
			t.Error(err)
		}
	}()
	native, err := NewLinuxNPMResolverWithExecutorAndPolicy(executor, supervisor.Observer(), newIntegrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source, err := artifactnpm.NewPublicResolver(filepath.Join(root, "intake"))
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(root)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewNPMProjectInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	controls := func(t *testing.T, manifest, lock []byte) []domain.ProjectControlFile {
		t.Helper()
		a, err := domain.NewProjectControlFile("package.json", manifest, true)
		if err != nil {
			t.Fatal(err)
		}
		b, err := domain.NewProjectControlFile("package-lock.json", lock, lock != nil)
		if err != nil {
			t.Fatal(err)
		}
		return []domain.ProjectControlFile{a, b}
	}
	selectProject := func(t *testing.T, label string, intent artifactnpm.NPMProjectIntent, files []domain.ProjectControlFile, deny bool) NPMNativeProjectSelection {
		t.Helper()
		input, err := artifactnpm.NewNPMProjectInput(intent, files, resolverNPMVersion)
		if err != nil {
			t.Fatal(label, err)
		}
		start := time.Now()
		selected, err := native.SelectProject(ctx, input)
		if deny {
			if err == nil || len(selected.Controls()) != 0 {
				t.Fatal(label, "native invalid selection was accepted")
			}
			t.Logf("npm_project_native case=%s denied=true seconds=%.3f", label, time.Since(start).Seconds())
			return selected
		}
		if err != nil {
			t.Fatal(label, err)
		}
		bound, err := source.BindNPMProjectLock(ctx, selected.Selection())
		if err != nil || !bound.SourceBound() {
			t.Fatal(label, "source binding", err)
		}
		snapshot, err := bound.ProjectSnapshot(install, selected.Controls(), NPMProjectRuntimeIdentity())
		if err != nil || !snapshot.Valid() {
			t.Fatal(label, "whole snapshot", err)
		}
		t.Logf("npm_project_native case=%s placements=%d archives=%d dependency_free=%t seconds=%.3f", label, len(bound.Lock().Packages()), len(bound.Artifacts()), snapshot.DependencyFree(), time.Since(start).Seconds())
		return selected
	}
	t.Run("install-ci", func(t *testing.T) {
		manifest := []byte(`{"name":"haa-native-selection","private":true,"dependencies":{"is-number":"7.0.0","is-odd":"3.0.1"}}`)
		initial := selectProject(t, "install-initial", artifactnpm.NPMProjectInstall, controls(t, manifest, nil), false)
		if len(initial.Selection().Roots()) != 2 || len(initial.Selection().Packages()) != 3 {
			t.Fatal("two roots/transitive version placement differs from native expectation")
		}
		frozen := selectProject(t, "ci-frozen", artifactnpm.NPMProjectCI, initial.Controls(), false)
		if !bytes.Equal(frozen.Controls()[1].Body(), initial.Controls()[1].Body()) {
			t.Fatal("ci rewrote native lock bytes")
		}
		var forged map[string]json.RawMessage
		if err := json.Unmarshal(initial.Controls()[1].Body(), &forged); err != nil {
			t.Fatal(err)
		}
		var packages map[string]map[string]any
		if err := json.Unmarshal(forged["packages"], &packages); err != nil {
			t.Fatal(err)
		}
		packages["node_modules/is-number"]["version"] = "6.0.0"
		forged["packages"], err = json.Marshal(packages)
		if err != nil {
			t.Fatal(err)
		}
		stale, err := json.Marshal(forged)
		if err != nil {
			t.Fatal(err)
		}
		selectProject(t, "ci-incompatible-node", artifactnpm.NPMProjectCI, controls(t, manifest, stale), true)
		changed := bytes.Replace(manifest, []byte(`"is-number":"7.0.0"`), []byte(`"is-number":"6.0.0"`), 1)
		updated := selectProject(t, "install-changed", artifactnpm.NPMProjectInstall, controls(t, changed, initial.Controls()[1].Body()), false)
		if len(updated.Selection().Packages()) != 2 {
			t.Fatal("native changed selection retained stale/unreachable placement")
		}
	})
	t.Run("peer-metadata-only", func(t *testing.T) {
		selectProject(t, "peer-metadata-only", artifactnpm.NPMProjectInstall, controls(t, []byte(`{"dependencies":{"debug":"4.4.3","supports-color":"10.2.2"}}`), nil), false)
	})
	t.Run("declared-peer", func(t *testing.T) {
		selected := selectProject(t, "declared-peer", artifactnpm.NPMProjectInstall, controls(t, []byte(`{"dependencies":{"react":"19.2.0","react-dom":"19.2.0"}}`), nil), false)
		found := false
		for _, pkg := range selected.Selection().Packages() {
			if pkg.Name == "react-dom" {
				found = pkg.PeerDependencies["react"] != "" && pkg.Targets["react"] == "node_modules/react"
			}
		}
		if !found {
			t.Fatal("native declared peer was not bound to its actual placement")
		}
	})
	t.Run("optional-platform", func(t *testing.T) {
		optional := selectProject(t, "optional-platform", artifactnpm.NPMProjectInstall, controls(t, []byte(`{"optionalDependencies":{"fsevents":"2.3.3"}}`), nil), false)
		if !optional.Selection().DependencyFree() {
			t.Fatal("Darwin-only optional package was selected for Linux runtime")
		}
	})
}
