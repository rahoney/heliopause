package promotion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

// Synthetic metadata with the existing independent whole UV-backend wheel and
// script proof fixture. This is not an actual native pipx producer claim.
func makeNativePipxOriginalFixture(t *testing.T, mode string) nativeToolOriginalFixture {
	t.Helper()
	f := makeNativeToolOriginalFixture(t, mode)
	uv, err := artifactpypi.ParseUVToolReceipt(f.raw)
	if err != nil {
		t.Fatal(err)
	}
	receipt := pipxTargetReceipt(t, f.install, f.bin)
	receipt, err = receipt.WithPipxLayout(f.root, f.bin, uv.InterpreterPath())
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(receipt.Body(), &wire); err != nil {
		t.Fatal(err)
	}
	main := wire["main_package"].(map[string]any)
	arguments := []string{"--no-build", "--no-sources", "--no-config", "--no-cache", "--no-python-downloads", "--no-managed-python", "--link-mode=copy", "--index-url=https://pypi.org/simple/"}
	if mode == "unsupported-recipe" {
		arguments = arguments[1:]
	}
	if mode == "default-recipe" {
		arguments = nil
	}
	main["pip_args"] = arguments
	var names []string
	var paths []map[string]string
	for _, c := range uv.Commands() {
		names = append(names, c.Name)
		paths = append(paths, map[string]string{"__type__": "Path", "__Path__": filepath.Join(f.root, "bin", c.Name)})
	}
	main["apps"], main["app_paths"] = names, paths
	body, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifactpypi.ParsePipxToolReceipt(body); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(f.root, uvToolHostReceipt)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, pipxToolHostReceipt), body, 0600); err != nil {
		t.Fatal(err)
	}
	f.raw = body
	f.request, err = artifactpypi.ParseUpgradeRequirement("haa-tool-fixture")
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func beginNativePipxFixture(ctx context.Context, f nativeToolOriginalFixture) (*toolFreshTarget, error) {
	return beginToolNativeOriginalTargetForClient(ctx, f.install, f.request, f.bin, []string{f.intake, f.staging}, f.native, toolClientPipx)
}

func TestNativePipxOriginalUsesIndependentWholeBackendProofBeforeGuard(t *testing.T) {
	f := makeNativePipxOriginalFixture(t, "valid")
	g, err := beginNativePipxFixture(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := g.Close(); err != nil {
			t.Error(err)
		}
	}()
	if len(f.ports.resolves) != 1 || len(f.ports.acquires) != 1 || g.original == nil || g.original.pipx == nil || g.original.native == nil || !g.original.native.ownership.Present() {
		t.Fatal("complete original source proof missing")
	}
	if !bytes.Equal(g.original.native.rawReceipt.Body(), f.raw) || bytes.Equal(g.original.pipx.Receipt().Body(), f.raw) || g.original.native.rawReceipt.Name() != pipxToolOriginalNativeReceiptControl {
		t.Fatal("raw pipx JSON and derived projection conflated")
	}
	if g.original.pipx.Request().Requirement() != "haa-tool-fixture" || g.original.pipx.Receipt().Requirement().Requirement() != "haa-tool-fixture<2,>=1" || len(g.original.links) != 2 {
		t.Fatal("raw original/new request/links not separate")
	}
	if err := g.original.pipx.Receipt().ValidateLayout(f.root, f.bin, filepath.Join(f.root, "bin/python")); err != nil {
		t.Fatal(err)
	}
	controls, err := uvToolNativeOriginalControls(context.Background(), g)
	if err != nil || len(controls) != 2 || controls[0].Name() != pipxToolOriginalNativeReceiptControl {
		t.Fatal("native raw/source provenance missing", err)
	}
	if _, err := os.Lstat(filepath.Join(f.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unapproved enrollment wrote ledger")
	}
	if other, err := beginNativePipxFixture(context.Background(), f); err == nil {
		_ = other.Close()
		t.Fatal("concurrent original entered")
	}
	if len(f.ports.resolves) != 1 {
		t.Fatal("concurrent guard reached source")
	}
	after, err := os.ReadFile(filepath.Join(f.root, pipxToolHostReceipt))
	if err != nil || !bytes.Equal(after, f.raw) {
		t.Fatal("raw native receipt rewritten")
	}
}

func TestNativePipxOriginalRefusesUnprovedDataAndKeepsRawState(t *testing.T) {
	for _, mode := range []string{"unsupported-recipe", "default-recipe", "other-interpreter", "foreign-link", "missing-receipt-command", "unowned-receipt-command", "archive-tamper", "script-code", "extra-cache", "cache-before"} {
		t.Run(mode, func(t *testing.T) {
			f := makeNativePipxOriginalFixture(t, mode)
			g, err := beginNativePipxFixture(context.Background(), f)
			if err == nil || g != nil {
				if g != nil {
					_ = g.Close()
				}
				t.Fatal("unproved original accepted")
			}
			after, err := os.ReadFile(filepath.Join(f.root, pipxToolHostReceipt))
			if err != nil || !bytes.Equal(after, f.raw) {
				t.Fatal("refused raw receipt changed")
			}
			if _, err := os.Lstat(filepath.Join(f.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("refused original wrote ownership")
			}
			for _, p := range []string{filepath.Join(f.root, ".heliopause-pypi-transaction.lock"), filepath.Join(f.bin, toolBinTransactionLock)} {
				if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("failed original retained guard", p, err)
				}
			}
			if (mode == "unsupported-recipe" || mode == "default-recipe" || mode == "other-interpreter" || mode == "foreign-link") && len(f.ports.resolves) != 0 {
				t.Fatal("static/held original failure reached source lookup")
			}
		})
	}
}

func TestPipxNativeCapabilityIsExplicitCopiedAndRequiresSameIntake(t *testing.T) {
	f := makeNativePipxOriginalFixture(t, "valid")
	selector := &pipxFailedUpdate{pipxFailedSelection: pipxFailedSelection{err: errors.New("new native selection stopped")}}
	p, err := NewPipxToolUpdatePreparation(&PyPIPromotion{stagingRoot: f.staging, runner: unavailableDockerRunner{}, goos: "linux", goarch: "amd64"}, selector, f.bin, []string{f.intake, f.staging})
	if err != nil {
		t.Fatal(err)
	}
	if session, err := p.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), "haa-tool-fixture", f.install, true); err == nil || session != nil || len(f.ports.resolves) != 0 {
		t.Fatal("native capability inferred from receipt")
	}
	if _, err := p.WithNativeEnrollment(f.ports, f.ports, filepath.Dir(f.intake)); err == nil {
		t.Fatal("foreign intake accepted")
	}
	enabled, err := p.WithNativeEnrollment(f.ports, f.ports, f.intake)
	if err != nil {
		t.Fatal(err)
	}
	if p.shared.native != nil || enabled.shared.native == nil {
		t.Fatal("native capability mutated prior preparation")
	}
	session, err := enabled.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), "haa-tool-fixture", f.install, true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.ResolveDependencies(context.Background(), session.Reference(), f.install)
	if !errors.Is(err, selector.err) || selector.upgradeCalls != 1 || selector.calls != 0 {
		t.Fatal("proved original bypassed current native selection or primary failure", err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}
