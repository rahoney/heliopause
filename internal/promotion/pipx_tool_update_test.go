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

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func managedPipxOriginalFixture(t *testing.T) (domain.InstallContext, string, string) {
	t.Helper()
	g, outputs, receipt, work := pipxPublicationFixture(t)
	for _, v := range []struct{ name, body string }{{"METADATA", "Metadata-Version: 2.4\nName: haa-tool-fixture\nVersion: 1.0.0\n"}, {"WHEEL", "Wheel-Version: 1.0\nTag: py3-none-any\n"}} {
		relative := "haa_tool_fixture-1.0.0.dist-info/" + v.name
		path := filepath.Join(work, "site", relative)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(v.body), 0600); err != nil {
			t.Fatal(err)
		}
		snap, err := snapshotPyPIFile(path)
		if err != nil {
			t.Fatal(err)
		}
		outputs = append(outputs, pypiDestination{Scheme: "site", Relative: relative, Distribution: "haa-tool-fixture", Version: "1.0.0", ArtifactDigest: strings.Repeat("a", 64), Source: path, Digest: snap.Digest, Size: snap.Size})
	}
	var wire map[string]any
	if err := json.Unmarshal(receipt.Body(), &wire); err != nil {
		t.Fatal(err)
	}
	wire["main_package"].(map[string]any)["pip_args"] = []string{"--no-build", "--no-sources", "--no-config", "--no-cache", "--no-python-downloads", "--no-managed-python", "--link-mode=copy", "--index-url=https://pypi.org/simple/"}
	body, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err = domain.NewProjectControlFile(pipxToolHostReceiptControl, body, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.commitTool(context.Background(), outputs, receipt, work); err != nil {
		t.Fatal(err)
	}
	install := toolMaterializerContext(t, g.tx.plan.root)
	bin := g.binRoot
	storage := filepath.Join(filepath.Dir(g.tx.plan.root), "staging")
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	return install, bin, storage
}

type pipxFailedUpdate struct {
	pipxFailedSelection
	upgradeCalls int
	original     artifactpypi.PipxToolUpgradeInput
	wheels       []artifactpypi.UVInstalledWheelControls
}

func (p *pipxFailedUpdate) ResolveToolUpgrade(_ context.Context, input artifactpypi.PipxToolUpgradeInput, wheels []artifactpypi.UVInstalledWheelControls) (sandbox.PipxToolSelection, error) {
	p.upgradeCalls++
	p.original = input
	p.wheels = wheels
	if p.before != nil {
		p.before()
	}
	return sandbox.PipxToolSelection{}, p.err
}

func TestPipxManagedOriginalRetainsOwnershipLocksAndSeparateBareRequest(t *testing.T) {
	install, bin, storage := managedPipxOriginalFixture(t)
	primary := errors.New("native pipx upgrade stopped")
	selector := &pipxFailedUpdate{pipxFailedSelection: pipxFailedSelection{err: primary}}
	preparation, err := NewPipxToolUpdatePreparation(&PyPIPromotion{stagingRoot: storage, runner: unavailableDockerRunner{}, goos: "linux", goarch: "amd64"}, selector, bin, []string{storage})
	if err != nil {
		t.Fatal(err)
	}
	session, err := preparation.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), "haa-tool-fixture", install, false)
	if err != nil {
		t.Fatal(err)
	}
	guard := session.(*uvToolInstallSession).guard
	beforeReceipt := guard.tx.before[filepath.Join(install.Target().String(), pipxToolHostReceipt)]
	selector.before = func() {
		if !beforeReceipt.matches(filepath.Join(install.Target().String(), pipxToolHostReceipt)) {
			t.Fatal("original receipt changed")
		}
		if other, err := preparation.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), "haa-tool-fixture", install, false); err == nil {
			_ = other.Close()
			t.Fatal("concurrent original entered")
		}
	}
	result, err := session.ResolveDependencies(context.Background(), session.Reference(), install)
	if !errors.Is(err, primary) || len(result.Graph().Nodes()) != 0 || selector.upgradeCalls != 1 || selector.calls != 0 {
		t.Fatal("original routed fresh or primary lost", err)
	}
	if selector.original.Request().Requirement() != "haa-tool-fixture" || selector.original.Receipt().Requirement().Requirement() != "haa-tool-fixture<2,>=1" || len(selector.wheels) != 1 {
		t.Fatal("original constraint and new request conflated")
	}
	if _, err := session.ResolveDependencies(context.Background(), session.Reference(), install); err == nil || selector.upgradeCalls != 1 {
		t.Fatal("repeated native upgrade")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPipxManagedOriginalRejectsChangedOrUnownedBeforeNative(t *testing.T) {
	for _, mutation := range []string{"receipt-version", "receipt-client", "missing-ownership", "wheel", "link", "selector", "adopt", "cancel"} {
		t.Run(mutation, func(t *testing.T) {
			install, bin, storage := managedPipxOriginalFixture(t)
			root := install.Target().String()
			selector := &pipxFailedUpdate{}
			preparation, err := NewPipxToolUpdatePreparation(&PyPIPromotion{stagingRoot: storage, runner: unavailableDockerRunner{}, goos: "linux", goarch: "amd64"}, selector, bin, []string{storage})
			if err != nil {
				t.Fatal(err)
			}
			request := "haa-tool-fixture"
			adopt := false
			ctx := context.Background()
			switch mutation {
			case "receipt-version":
				p := filepath.Join(root, pipxToolHostReceipt)
				b, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				b = bytes.Replace(b, []byte(`"package_version":"1.0.0"`), []byte(`"package_version":"1.1.0"`), 1)
				if err := os.WriteFile(p, b, 0600); err != nil {
					t.Fatal(err)
				}
			case "receipt-client":
				b, err := os.ReadFile("../artifact/pypi/testdata/persistent-tool/uv-installed.toml")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, pipxToolHostReceipt), b, 0600); err != nil {
					t.Fatal(err)
				}
			case "missing-ownership":
				if err := os.Remove(filepath.Join(root, pypiVenvMetadata)); err != nil {
					t.Fatal(err)
				}
			case "wheel":
				if err := os.WriteFile(filepath.Join(root, "lib/python3.14/site-packages/haa_tool_fixture-1.0.0.dist-info/WHEEL"), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "link":
				p := filepath.Join(bin, "haa-demo")
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("/foreign/interpreter", p); err != nil {
					t.Fatal(err)
				}
			case "selector":
				request = "haa-tool-fixture>=1"
			case "adopt":
				adopt = true
			case "cancel":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			}
			session, err := preparation.BeginInstallUpdate(ctx, artifactpypi.PublicPyPIProfile().Source(), request, install, adopt)
			if err == nil || session != nil || selector.calls != 0 || selector.upgradeCalls != 0 {
				if session != nil {
					_ = session.Close()
				}
				t.Fatal("unsupported original accepted", err)
			}
			if _, err := os.Lstat(filepath.Join(root, ".heliopause-pypi-transaction.lock")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed original leaked lock", err)
			}
		})
	}
}

func TestPipxManagedPublicationRetiresCommandsAndRollsBackWholeOriginal(t *testing.T) {
	for _, phase := range []string{"", "after-publish:tool-control", "metadata", "after-tool-link:haa-alt", "sync", "reconcile"} {
		t.Run(phase, func(t *testing.T) {
			install, bin, storage := managedPipxOriginalFixture(t)
			request, _ := artifactpypi.ParseUpgradeRequirement("haa-tool-fixture")
			g, err := beginToolTargetForClient(context.Background(), install, request, bin, []string{storage}, true, nil, toolClientPipx)
			if err != nil {
				t.Fatal(err)
			}
			original := g.original.receipt().Body()
			var wire map[string]any
			if err := json.Unmarshal(original, &wire); err != nil {
				t.Fatal(err)
			}
			main := wire["main_package"].(map[string]any)
			main["package_or_url"] = "haa-tool-fixture"
			main["apps"] = []string{"haa-alt"}
			main["app_paths"] = []map[string]string{{"__type__": "Path", "__Path__": filepath.Join(g.tx.plan.root, "bin", "haa-alt")}}
			body, err := json.Marshal(wire)
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := artifactpypi.ParsePipxToolReceipt(body)
			if err != nil {
				t.Fatal(err)
			}
			if err := g.freezeCommands(context.Background(), receipt); err != nil {
				t.Fatal(err)
			}
			control, err := domain.NewProjectControlFile(pipxToolHostReceiptControl, body, true)
			if err != nil {
				t.Fatal(err)
			}
			work := filepath.Join(g.tx.plan.root, ".haa-pipx-update-test")
			if err := os.Mkdir(work, 0700); err != nil {
				t.Fatal(err)
			}
			var outputs []pypiDestination
			for _, old := range g.tx.current.Files {
				if old.Scheme == "scripts" && old.Relative == "haa-demo" {
					continue
				}
				path := filepath.Join(work, old.Scheme, old.Relative)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				bound, err := g.tx.plan.bind(old)
				if err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(bound.Final)
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(bound.Final)
				if err != nil {
					t.Fatal(err)
				}
				if old.Relative == "haa_tool_fixture/__init__.py" {
					data = []byte("new inert module bytes")
				}
				if err := os.WriteFile(path, data, info.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
				snap, err := snapshotPyPIFile(path)
				if err != nil {
					t.Fatal(err)
				}
				old.Source = path
				old.Digest = snap.Digest
				old.Size = snap.Size
				outputs = append(outputs, old)
			}
			primary := errors.New("pipx managed commit fault")
			if phase != "" {
				g.tx.fault = func(at string) error {
					if at == phase {
						return primary
					}
					return nil
				}
			}
			err = g.commitTool(context.Background(), outputs, control, work)
			if phase == "" {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := os.Lstat(filepath.Join(bin, "haa-demo")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("retired external command survived")
				}
				if _, err := os.Lstat(filepath.Join(g.tx.plan.root, "bin", "haa-demo")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("retired internal script survived")
				}
				published, err := os.ReadFile(filepath.Join(g.tx.plan.root, pipxToolHostReceipt))
				if err != nil || !bytes.Equal(published, body) {
					t.Fatal("new bare receipt not committed")
				}
			} else {
				if !errors.Is(err, primary) {
					t.Fatal("primary failure lost", err)
				}
				requireOriginalToolRestored(t, g)
				published, err := os.ReadFile(filepath.Join(g.tx.plan.root, pipxToolHostReceipt))
				if err != nil || !bytes.Equal(published, original) {
					t.Fatal("original receipt not restored")
				}
			}
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
