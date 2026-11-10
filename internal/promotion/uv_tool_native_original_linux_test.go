package promotion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type nativeToolOriginalFixture struct {
	install                          domain.InstallContext
	request                          artifactpypi.UpgradeRequirement
	root, bin, intake, staging, base string
	raw                              []byte
	ports                            *originalProofSetPorts
	native                           *pypiNativeEnrollment
}

func makeNativeToolOriginalFixture(t *testing.T, mode string) nativeToolOriginalFixture {
	t.Helper()
	f := makeNativeToolScriptFixture(t, mode)
	if err := f.close(); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(f.root)
	bin, intake, staging := filepath.Join(parent, "commands"), filepath.Join(parent, "intake"), filepath.Join(parent, "staging")
	for _, p := range []string{bin, intake, staging} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	base := filepath.Join(parent, "base-python3.14")
	if err := os.Rename(filepath.Join(f.root, "bin/python3.14"), base); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../base-python3.14", filepath.Join(f.root, "bin/python3.14")); err != nil {
		t.Fatal(err)
	}
	install := toolMaterializerContext(t, f.root)
	receipt := toolTargetReceipt(t, install, bin)
	var wire map[string]any
	if err := toml.Unmarshal(receipt.Body(), &wire); err != nil {
		t.Fatal(err)
	}
	tool := wire["tool"].(map[string]any)
	tool["python"] = base
	options := tool["options"].(map[string]any)
	delete(options, "no-index")
	delete(options, "find-links")
	if mode == "unsupported-recipe" {
		delete(options, "no-build")
	}
	if mode == "other-interpreter" {
		other := filepath.Join(parent, "unrelated-python")
		body, err := os.ReadFile(base)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(other, body, 0700); err != nil {
			t.Fatal(err)
		}
		tool["python"] = other
	}
	entries := tool["entrypoints"].([]any)
	if mode == "missing-receipt-command" {
		tool["entrypoints"] = entries[:1]
	}
	if mode == "unowned-receipt-command" {
		tool["entrypoints"] = append(entries, map[string]any{"name": "foreign", "install-path": filepath.Join(bin, "foreign"), "from": "haa-tool-fixture"})
	}
	body, err := toml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := artifactpypi.ParseUVToolReceipt(body)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, uvToolHostReceipt), body, 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range raw.Commands() {
		target := filepath.Join(f.root, "bin", command.Name)
		if mode == "foreign-link" && command.Name == "haa-demo" {
			target = base
		}
		if err := os.Symlink(target, command.Path); err != nil {
			t.Fatal(err)
		}
	}
	if mode == "cache-before" {
		if err := os.Mkdir(filepath.Join(f.root, "lib/python3.14/site-packages/haa_tool_fixture/__pycache__"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	var archive struct {
		Filename string
		Body     []byte
	}
	data, err := os.ReadFile("testdata/uv-native-tool-wheel.json")
	if err != nil || json.Unmarshal(data, &archive) != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(parent, archive.Filename)
	if err := os.WriteFile(archivePath, archive.Body, 0600); err != nil {
		t.Fatal(err)
	}
	identity, _ := domain.NewResolvedArtifactIdentity(artifactpypi.PublicPyPIProfile().Source(), "haa-tool-fixture", "1.0.0", "wheel")
	resolved, err := domain.NewResolvedArtifact(identity, "https://files.pythonhosted.org/packages/fixture/"+archive.Filename, "sha256:"+f.digest)
	if err != nil {
		t.Fatal(err)
	}
	ports := &originalProofSetPorts{root: intake, entries: map[string]originalProofArchive{"haa-tool-fixture": {resolved, archivePath, archive.Filename}}}
	if mode == "archive-tamper" {
		ports.corrupt = "haa-tool-fixture"
	}
	native := &pypiNativeEnrollment{resolver: ports, intake: ports, intakeRoot: intake, client: nativeUVClient}
	request, err := artifactpypi.ParseUpgradeRequirement("haa-tool-fixture>=1.1,<2")
	if err != nil {
		t.Fatal(err)
	}
	return nativeToolOriginalFixture{install, request, f.root, bin, intake, staging, base, body, ports, native}
}

func (f nativeToolOriginalFixture) begin(ctx context.Context) (*toolFreshTarget, error) {
	return beginToolNativeOriginalTarget(ctx, f.install, f.request, f.bin, []string{f.intake, f.staging}, f.native)
}

func TestNativeUVToolOriginalAuthenticatesBeforeReturningHeldGuard(t *testing.T) {
	f := makeNativeToolOriginalFixture(t, "valid")
	g, err := f.begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := g.Close(); err != nil {
			t.Error(err)
		}
	})
	if len(f.ports.resolves) != 1 || len(f.ports.acquires) != 1 || g.original == nil || g.original.native == nil || !g.original.native.ownership.Present() {
		t.Fatal("complete independent original source proof missing")
	}
	if !bytes.Equal(g.original.native.rawReceipt.Body(), f.raw) || bytes.Equal(g.original.input.Receipt().Body(), f.raw) {
		t.Fatal("raw receipt and derived Host metadata projection were conflated")
	}
	if err := g.original.input.Receipt().ValidateLayout(f.root, f.bin, filepath.Join(f.root, "bin/python")); err != nil {
		t.Fatal(err)
	}
	if len(g.original.links) != 2 || len(g.tx.current.Files) == 0 || g.original.input.Request().Requirement() != f.request.Requirement() {
		t.Fatal("original commands/ownership/request were not held independently")
	}
	if _, err := os.Lstat(filepath.Join(f.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("enrollment wrote an on-disk ledger", err)
	}
	if err := g.verify(context.Background()); err != nil {
		t.Fatal(err)
	}
	if other, err := f.begin(context.Background()); err == nil {
		_ = other.Close()
		t.Fatal("concurrent original entered")
	}
	if len(f.ports.resolves) != 1 {
		t.Fatal("concurrent guard reached source lookup")
	}
	after, err := os.ReadFile(filepath.Join(f.root, uvToolHostReceipt))
	if err != nil || !bytes.Equal(after, f.raw) {
		t.Fatal("raw native receipt was rewritten", err)
	}
}

func TestNativeUVToolOriginalRejectsUnprovedStateAndPreservesRawFiles(t *testing.T) {
	for _, mode := range []string{"unsupported-recipe", "other-interpreter", "foreign-link", "missing-receipt-command", "unowned-receipt-command", "archive-tamper", "script-code", "extra-cache", "cache-before"} {
		t.Run(mode, func(t *testing.T) {
			f := makeNativeToolOriginalFixture(t, mode)
			g, err := f.begin(context.Background())
			if err == nil || g != nil {
				if g != nil {
					_ = g.Close()
				}
				t.Fatal("unproved native original accepted")
			}
			after, readErr := os.ReadFile(filepath.Join(f.root, uvToolHostReceipt))
			if readErr != nil || !bytes.Equal(after, f.raw) {
				t.Fatal("refused original receipt changed", readErr)
			}
			if _, err := os.Lstat(filepath.Join(f.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("refused original wrote ownership", err)
			}
			for _, lock := range []string{filepath.Join(f.root, ".heliopause-pypi-transaction.lock"), filepath.Join(f.bin, toolBinTransactionLock)} {
				if _, err := os.Lstat(lock); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("refused original leaked a guard", lock, err)
				}
			}
			if (mode == "unsupported-recipe" || mode == "other-interpreter" || mode == "foreign-link") && len(f.ports.resolves) != 0 {
				t.Fatal("unsupported original reached source lookup")
			}
		})
	}
}

func TestNativeUVToolOriginalKeepsDefaultRefusalAndHeldRawBaseFingerprint(t *testing.T) {
	f := makeNativeToolOriginalFixture(t, "valid")
	if g, err := beginToolOriginalTarget(context.Background(), f.install, f.request, f.bin, []string{f.intake, f.staging}); err == nil {
		_ = g.Close()
		t.Fatal("managed/default guard silently adopted a native original")
	}
	if len(f.ports.resolves) != 0 {
		t.Fatal("default refusal invoked original source lookup")
	}
	g, err := f.begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := g.Close(); err != nil {
			t.Error(err)
		}
	})
	before, err := os.ReadFile(f.base)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.base, []byte("changed inert base interpreter DATA"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := g.verify(context.Background()); err == nil {
		t.Fatal("raw base interpreter drift accepted")
	}
	if err := os.WriteFile(f.base, before, 0700); err != nil {
		t.Fatal(err)
	}
	if err := g.verify(context.Background()); err != nil {
		t.Fatal("restored held state differs", err)
	}
	if !strings.Contains(string(g.original.native.rawReceipt.Body()), f.base) {
		t.Fatal("raw base claim lost")
	}
}
