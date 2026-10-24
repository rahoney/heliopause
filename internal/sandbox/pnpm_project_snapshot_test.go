package sandbox

import (
	"context"
	"errors"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestPNPMNativeSnapshotUsesExplicitEmptyControlsAndPreservesCancellation(t *testing.T) {
	manifest := pnpmTestControl(t, "package.json", `{}`, true)
	lock := pnpmTestControl(t, "pnpm-lock.yaml", "lockfileVersion: '9.0'\nsettings: {autoInstallPeers: true, excludeLinksFromLockfile: false}\nimporters: {'.': {}}\n", true)
	parsed, err := artifactnpm.ParsePNPMLock(manifest.Body(), lock.Body(), runtimeidentity.PNPMVersion)
	if err != nil {
		t.Fatal(err)
	}
	selected := PNPMNativeSelection{controls: []domain.ProjectControlFile{manifest, lock}, selection: parsed}
	source, err := artifactnpm.NewPublicResolver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget("/tmp/haa-pnpm-empty-snapshot")
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewNPMProjectInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	// An explicit empty selection needs no registry request; it still binds
	// controls and the real generated producer identity and is not an ALLOW.
	snapshot, err := selected.BindProjectSnapshot(context.Background(), source, install)
	if err != nil || !snapshot.DependencyFree() || len(snapshot.ControlDigests()) != 2 {
		t.Fatalf("explicit empty snapshot lost: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := selected.BindProjectSnapshot(ctx, source, install); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	for _, input := range []struct {
		name    string
		ctx     context.Context
		install domain.InstallContext
	}{
		{name: "missing-context", install: install},
		{name: "missing-install", ctx: context.Background()},
	} {
		if _, err := selected.BindProjectSnapshot(input.ctx, source, input.install); err == nil {
			t.Fatalf("invalid input accepted: %s", input.name)
		}
	}
	if _, err := selected.BindProjectSnapshot(context.Background(), nil, install); err == nil {
		t.Fatal("missing public source accepted")
	}
	if _, err := (PNPMNativeSelection{}).BindProjectSnapshot(context.Background(), source, install); err == nil {
		t.Fatal("zero selection accepted as empty")
	}
}
