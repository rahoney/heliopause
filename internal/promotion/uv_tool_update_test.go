package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type uvToolUpgradeFailure struct {
	uvToolSelectionFailure
	upgrades int
}

func (r *uvToolUpgradeFailure) ResolveToolUpgrade(_ context.Context, input artifactpypi.UVToolUpgradeInput, wheels []artifactpypi.UVInstalledWheelControls) (sandbox.UVToolSelection, error) {
	r.upgrades++
	if r.before != nil {
		r.before()
	}
	return sandbox.UVToolSelection{}, r.err
}

func TestUVToolUpdateRetainsOriginalAndBothGuardsAcrossNativeFailure(t *testing.T) {
	install, bin, storage := managedToolOriginalFixture(t)
	primary := errors.New("native upgrade stopped")
	native := &uvToolUpgradeFailure{uvToolSelectionFailure: uvToolSelectionFailure{err: primary}}
	p, err := NewUVToolUpdatePreparation(&PyPIPromotion{stagingRoot: storage, runner: unavailableDockerRunner{}, goos: "linux", goarch: "amd64"}, native, bin, []string{storage})
	if err != nil {
		t.Fatal(err)
	}
	session, err := p.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), "haa-tool-fixture>=1.1,<2", install, false)
	if err != nil {
		t.Fatal(err)
	}
	if session.Reference().Locator() != "haa-tool-fixture>=1.1,<2" {
		t.Fatal("positional condition lost")
	}
	held := session.(*uvToolInstallSession)
	if held.guard.original.input.Receipt().Requirement().Requirement() == held.guard.requirement.Requirement() || len(held.originalWheels) != 1 {
		t.Fatal("original input lost")
	}
	native.before = func() {
		for _, path := range []string{filepath.Join(bin, toolBinTransactionLock), filepath.Join(install.Target().String(), ".heliopause-pypi-transaction.lock")} {
			if _, err := os.Lstat(path); err != nil {
				t.Fatal("native selection escaped guard", err)
			}
		}
		if concurrent, err := p.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), "haa-tool-fixture", install, false); err == nil {
			_ = concurrent.Close()
			t.Fatal("concurrent original entered")
		}
	}
	result, err := session.ResolveDependencies(context.Background(), session.Reference(), install)
	if !errors.Is(err, primary) || len(result.Graph().Nodes()) != 0 || native.upgrades != 1 || native.calls != 0 {
		t.Fatal("original error/dispatch changed", err)
	}
	if _, err := session.ResolveDependencies(context.Background(), session.Reference(), install); err == nil || native.upgrades != 1 {
		t.Fatal("repeated native upgrade")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := p.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), "haa-tool-fixture", install, true); err == nil || native.upgrades != 1 {
		t.Fatal("implicit native adoption accepted")
	}
}

func TestUVToolOriginalWheelCaptureRequiresExactHeldArchiveOwnership(t *testing.T) {
	for _, change := range []string{"none", "other archive", "missing ownership", "changed bytes"} {
		t.Run(change, func(t *testing.T) {
			install, bin, storage := managedToolOriginalFixture(t)
			request, _ := artifactpypi.ParseUpgradeRequirement("haa-tool-fixture")
			g, err := beginToolOriginalTarget(context.Background(), install, request, bin, []string{storage})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = g.Close() }()
			key := "site/haa_tool_fixture-1.0.0.dist-info/WHEEL"
			switch change {
			case "other archive":
				d := g.tx.current.Files[key]
				d.ArtifactDigest = strings.Repeat("b", 64)
				g.tx.current.Files[key] = d
			case "missing ownership":
				delete(g.tx.current.Files, key)
			case "changed bytes":
				path := filepath.Join(g.tx.plan.site, "haa_tool_fixture-1.0.0.dist-info/WHEEL")
				if err := os.WriteFile(path, []byte("Wheel-Version: 1.0\nTag: py3-none-any\nGenerator: changed\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			controls, err := captureUVToolOriginalWheels(context.Background(), g)
			if change == "none" {
				if err != nil || len(controls) != 1 {
					t.Fatal("valid original rejected", err)
				}
			} else if err == nil || len(controls) != 0 {
				t.Fatal("unbound WHEEL accepted")
			}
		})
	}
}
