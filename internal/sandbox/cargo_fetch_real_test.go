//go:build linux && realcorpus

package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/hosttool"
)

// Native selection DATA, not public source authentication, quarantine or build.
func TestCargoPublicFetchOptionalSelectionIntegration(t *testing.T) {
	if os.Getenv("HELOX_CARGO_FETCH_SELECTION_INTEGRATION") != "1" {
		t.Skip("requires declared protected Cargo fetch SDK")
	}
	if os.Geteuid() == 0 {
		t.Fatal("requires ordinary-user client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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
	runner, err := NewLinuxCargoRunner(executor, supervisor.Observer(), newIntegrationResolverPolicyService(t))
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := NewCargoResolver(runner)
	if err != nil {
		t.Fatal(err)
	}
	root := cargoSourceFixture(t)
	if err = os.Remove(filepath.Join(root, "Cargo.lock")); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	manifest := []byte("[package]\nname='haa_fetch_probe'\nversion='0.1.0'\nedition='2021'\n[dependencies]\nitoa={version='=1.0.17',optional=true}\n")
	if err = os.WriteFile(filepath.Join(root, "Cargo.toml"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	controls, err := cargoFrozenControls(map[string][]byte{"Cargo.toml": manifest})
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(root)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := resolver.ResolveProjectFetch(ctx, install, controls)
	if err != nil {
		t.Fatal("native Cargo fetch selection", err)
	}
	if !selected.Valid() || selected.HasRequestedResolution() || len(selected.Snapshot().Dependencies()) != 1 {
		t.Fatalf("fetch omitted an inactive optional dependency: valid=%t primary=%t registry_packages=%d", selected.Valid(), selected.HasRequestedResolution(), len(selected.Snapshot().Dependencies()))
	}
	artifact := selected.Snapshot().Dependencies()[0]
	if artifact.Identity().Name() != "itoa" || artifact.Identity().Version() != "1.0.17" {
		t.Fatal("fetch selected wrong package")
	}
	if _, err = os.Lstat(filepath.Join(root, "Cargo.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("selection published original lock before inspection", err)
	}
	t.Log("cargo_fetch_native inactive_optional=INCLUDED primary=NONE original_lock=ABSENT public_auth/quarantine/build=NOT_EXERCISED")
}
