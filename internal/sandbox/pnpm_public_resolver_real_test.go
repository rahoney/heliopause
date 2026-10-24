//go:build linux && realcorpus

package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

// The installed typed helper must authenticate this actual client executable.
// Native controls are still unapproved data; no archives or Policy are exercised.
func TestPNPMPublicNativeResolverIntegration(t *testing.T) {
	if os.Geteuid() == 0 || os.Getenv("HELOX_PNPM_PUBLIC_RESOLVER_INTEGRATION") != "1" {
		t.Fatal("requires ordinary-user authenticated public PNPM qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
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
	resolver, err := NewLinuxPNPMResolverWithExecutorAndPolicy(executor, supervisor.Observer(), integrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	source, err := artifactnpm.NewPublicResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	qualify := func(t *testing.T, name string, input artifactnpm.PNPMSelectionInput, artifacts, instances, roots int) PNPMNativeSelection {
		t.Helper()
		selected, err := resolver.SelectProject(ctx, input)
		if err != nil {
			t.Fatalf("actual native selection %s failed: %v", name, err)
		}
		bound, err := source.BindPNPMLock(ctx, selected.Selection())
		if err != nil {
			t.Fatalf("actual public binding %s failed: %v", name, err)
		}
		if !bound.SourceBound() || len(bound.Artifacts()) != artifacts || len(bound.Lock().Snapshots()) != instances || len(bound.Lock().Roots()) != roots {
			t.Fatal("native instances/public source binding differ")
		}
		for _, control := range selected.Controls() {
			sum := sha256.Sum256(control.Body())
			t.Logf("pnpm_selected_control name=%s bytes=%d sha256=%s", control.Name(), len(control.Body()), hex.EncodeToString(sum[:]))
		}
		t.Logf("pnpm_public_native case=%s artifacts=%d instances=%d roots=%d source_bound=true archives_and_policy=NOT_EXECUTED", name, artifacts, instances, roots)
		return selected
	}
	var original PNPMNativeSelection
	for _, item := range []struct {
		name, selector, manifest    string
		intent                      artifactnpm.PNPMSelectionIntent
		artifacts, instances, roots int
	}{
		{"single", "", `{"dependencies":{"is-number":"6.0.0"}}`, artifactnpm.PNPMInstall, 1, 1, 1},
		{"transitive", "", `{"dependencies":{"is-odd":"3.0.1"}}`, artifactnpm.PNPMInstall, 2, 2, 1},
		{"peer", "", `{"dependencies":{"react":"18.3.1","use-sync-external-store":"1.2.0"}}`, artifactnpm.PNPMInstall, 4, 4, 2},
		{"add", "is-number@6.0.0", `{}`, artifactnpm.PNPMAdd, 1, 1, 1},
	} {
		passed := t.Run(item.name, func(t *testing.T) {
			input := pnpmSelectionInputFixture(t, item.intent, item.selector, item.manifest)
			selected := qualify(t, item.name, input, item.artifacts, item.instances, item.roots)
			if item.name == "single" {
				original = selected
			}
		})
		if !passed {
			t.FailNow()
		}
	}
	if !t.Run("frozen-install", func(t *testing.T) {
		input, err := artifactnpm.NewPNPMSelectionInput(artifactnpm.PNPMInstall, "", original.Controls(), runtimeidentity.PNPMVersion)
		if err != nil {
			t.Fatal(err)
		}
		selected := qualify(t, "frozen-install", input, 1, 1, 1)
		for i, c := range selected.Controls() {
			if !bytes.Equal(c.Body(), original.Controls()[i].Body()) {
				t.Fatal("frozen controls changed")
			}
		}
	}) {
		t.FailNow()
	}
	// Explicit inert update fixture: widen the constraint around the actual v6
	// lock. It grants no enrollment or Host publication authority.
	manifest := pnpmTestControl(t, "package.json", `{"dependencies":{"is-number":">=6.0.0 <8.0.0"}}`, true)
	old := original.Controls()[1].Body()
	if bytes.Count(old, []byte("specifier: 6.0.0")) != 1 {
		t.Fatal("actual seed shape changed")
	}
	lockBody := bytes.Replace(old, []byte("specifier: 6.0.0"), []byte("specifier: '>=6.0.0 <8.0.0'"), 1)
	lock, err := domain.NewProjectControlFile("pnpm-lock.yaml", lockBody, true)
	if err != nil {
		t.Fatal(err)
	}
	seed := []domain.ProjectControlFile{manifest, lock}
	var updated PNPMNativeSelection
	for _, item := range []struct {
		name, selector string
		noop           bool
	}{
		{"selected-update", "is-number", false}, {"update-no-op", "is-number", true}, {"full-update", "", false},
	} {
		if !t.Run(item.name, func(t *testing.T) {
			controls := seed
			if item.noop {
				controls = updated.Controls()
			}
			input, err := artifactnpm.NewPNPMSelectionInput(artifactnpm.PNPMUpdate, item.selector, controls, runtimeidentity.PNPMVersion)
			if err != nil {
				t.Fatal(err)
			}
			selected := qualify(t, item.name, input, 1, 1, 1)
			roots := selected.Selection().Roots()
			packages := selected.Selection().Packages()
			if roots[0].Specifier != ">=6.0.0 <8.0.0" || packages[0].Name != "is-number" || packages[0].Version != "7.0.0" {
				t.Fatal("native update did not preserve the supported range/select v7")
			}
			if item.noop {
				for i, c := range selected.Controls() {
					if !bytes.Equal(c.Body(), controls[i].Body()) {
						t.Fatal("native no-op changed controls")
					}
				}
			}
			if item.name == "selected-update" {
				updated = selected
			}
		}) {
			t.FailNow()
		}
	}
}
