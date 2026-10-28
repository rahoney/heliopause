package promotion

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func pipxTargetReceipt(t *testing.T, install domain.InstallContext, bin string) artifactpypi.ToolReceipt {
	t.Helper()
	body, err := os.ReadFile("../artifact/pypi/testdata/persistent-tool/pipx-installed.json")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := artifactpypi.ParsePipxToolReceipt(body)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err = toolClientPipx.projectReceipt(receipt, install.Target().String(), bin)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

type pipxFailedSelection struct {
	calls  int
	before func()
	err    error
}

func (p *pipxFailedSelection) ResolveToolInstall(context.Context, artifactpypi.UpgradeRequirement) (sandbox.PipxToolSelection, error) {
	p.calls++
	if p.before != nil {
		p.before()
	}
	return sandbox.PipxToolSelection{}, p.err
}

func TestPipxFreshPreparationRetainsBothGuardsAndPrimaryFailure(t *testing.T) {
	for _, late := range []bool{false, true} {
		install, requirement, bin, storage := toolTargetFixture(t)
		primary := errors.New("native pipx stopped")
		selector := &pipxFailedSelection{err: primary}
		preparation, err := NewPipxToolInstallPreparation(&PyPIPromotion{stagingRoot: storage, runner: unavailableDockerRunner{}, goos: "linux", goarch: "amd64"}, selector, bin, []string{storage})
		if err != nil {
			t.Fatal(err)
		}
		reference, _ := domain.NewArtifactReference(artifactpypi.PublicPyPIProfile().Source(), requirement.Requirement())
		session, err := preparation.BeginInstall(context.Background(), reference, install)
		if err != nil {
			t.Fatal(err)
		}
		selector.before = func() {
			for _, path := range []string{filepath.Join(bin, toolBinTransactionLock), filepath.Join(install.Target().String(), ".heliopause-pypi-transaction.lock")} {
				if _, err := os.Lstat(path); err != nil {
					t.Fatal("unheld native selection", err)
				}
			}
			if other, err := preparation.BeginInstall(context.Background(), reference, install); err == nil {
				_ = other.Close()
				t.Fatal("concurrent entry")
			}
			if late {
				if err := os.WriteFile(filepath.Join(install.Target().String(), uvToolHostReceipt), []byte("foreign receipt"), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
		result, err := session.ResolveDependencies(context.Background(), reference, install)
		if !errors.Is(err, primary) || len(result.Graph().Nodes()) != 0 || selector.calls != 1 {
			t.Fatal("primary failure or zero result lost", err)
		}
		if _, err := session.ResolveDependencies(context.Background(), reference, install); err == nil || selector.calls != 1 {
			t.Fatal("repeated native attempt")
		}
		if err := session.Close(); err != nil {
			t.Fatal(err)
		}
		if err := session.Close(); err != nil {
			t.Fatal(err)
		}
		if late {
			b, err := os.ReadFile(filepath.Join(install.Target().String(), uvToolHostReceipt))
			if err != nil || string(b) != "foreign receipt" {
				t.Fatal("foreign receipt removed")
			}
		}
	}
}

func TestPipxFreshGuardRejectsMixedClientsAndExistingReceipts(t *testing.T) {
	for _, client := range []toolClient{toolClientUV, toolClientPipx} {
		for _, name := range []string{uvToolHostReceipt, pipxToolHostReceipt} {
			install, requirement, bin, storage := toolTargetFixture(t)
			path := filepath.Join(install.Target().String(), name)
			if err := os.WriteFile(path, []byte("keep original"), 0600); err != nil {
				t.Fatal(err)
			}
			g, err := beginToolTargetForClient(context.Background(), install, requirement, bin, []string{storage}, false, nil, client)
			if err == nil || g != nil {
				if g != nil {
					_ = g.Close()
				}
				t.Fatal("existing receipt accepted", client, name)
			}
			b, err := os.ReadFile(path)
			if err != nil || string(b) != "keep original" {
				t.Fatal("original changed")
			}
		}
	}
	install, requirement, bin, storage := toolTargetFixture(t)
	g, err := beginToolTargetForClient(context.Background(), install, requirement, bin, []string{storage}, false, nil, toolClientPipx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := g.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := g.freezeCommands(context.Background(), toolTargetReceipt(t, install, bin)); err == nil {
		t.Fatal("UV native receipt entered pipx guard")
	}
	if err := g.freezeCommands(context.Background(), pipxTargetReceipt(t, install, bin)); err != nil {
		t.Fatal(err)
	}
	if _, err := beginToolTargetForClient(context.Background(), install, requirement, bin, []string{storage}, true, nil, toolClientPipx); err == nil {
		t.Fatal("original pipx routed into fresh")
	}
}

func pipxPublicationFixture(t *testing.T) (*toolFreshTarget, []pypiDestination, domain.ProjectControlFile, string) {
	t.Helper()
	install, requirement, bin, storage := toolTargetFixture(t)
	g, err := beginToolTargetForClient(context.Background(), install, requirement, bin, []string{storage}, false, nil, toolClientPipx)
	if err != nil {
		t.Fatal(err)
	}
	receipt := pipxTargetReceipt(t, install, bin)
	if err := g.freezeCommands(context.Background(), receipt); err != nil {
		t.Fatal(err)
	}
	control, err := domain.NewProjectControlFile(pipxToolHostReceiptControl, receipt.Body(), true)
	if err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(install.Target().String(), ".haa-pipx-test-work")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	var destinations []pypiDestination
	for _, v := range []struct{ scheme, name string }{{"site", "haa_tool_fixture/__init__.py"}, {"scripts", "haa-alt"}, {"scripts", "haa-demo"}} {
		path := filepath.Join(work, v.scheme, v.name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("inert output bytes"), 0700); err != nil {
			t.Fatal(err)
		}
		snap, err := snapshotPyPIFile(path)
		if err != nil {
			t.Fatal(err)
		}
		destinations = append(destinations, pypiDestination{Scheme: v.scheme, Relative: v.name, Distribution: requirement.Project(), Version: "1.0.0", ArtifactDigest: strings.Repeat("a", 64), Source: path, Digest: snap.Digest, Size: snap.Size})
	}
	return g, destinations, control, work
}

func TestPipxFreshPublicationUsesSameAtomicEngineAndRollback(t *testing.T) {
	for _, phase := range []string{"", "after-publish:tool-control", "metadata", "after-tool-link:haa-alt", "after-tool-link:haa-demo", "sync", "reconcile"} {
		t.Run(phase, func(t *testing.T) {
			g, outputs, receipt, work := pipxPublicationFixture(t)
			primary := errors.New("injected pipx publication failure")
			if phase != "" {
				g.tx.fault = func(at string) error {
					if at == phase {
						return primary
					}
					return nil
				}
			}
			err := g.commitTool(context.Background(), outputs, receipt, work)
			if phase == "" {
				if err != nil {
					t.Fatal(err)
				}
				body, err := os.ReadFile(filepath.Join(g.tx.plan.root, pipxToolHostReceipt))
				if err != nil || !bytes.Equal(body, receipt.Body()) {
					t.Fatal("receipt publication differs", err)
				}
				for _, name := range []string{"haa-alt", "haa-demo"} {
					link, err := os.Readlink(filepath.Join(g.binRoot, name))
					if err != nil || link != filepath.Join(g.tx.plan.root, "bin", name) {
						t.Fatal("wrong published link", err)
					}
				}
				state, _, stateErr := g.tx.plan.readState()
				if stateErr != nil || len(state.Files) != 3 {
					t.Fatal("no owned wheel outputs")
				}
			} else {
				if !errors.Is(err, primary) {
					t.Fatal("primary failure lost", err)
				}
				for _, path := range []string{filepath.Join(g.tx.plan.root, pipxToolHostReceipt), filepath.Join(g.tx.plan.root, pypiVenvMetadata), filepath.Join(g.tx.plan.site, "haa_tool_fixture/__init__.py"), filepath.Join(g.binRoot, "haa-alt"), filepath.Join(g.binRoot, "haa-demo"), filepath.Join(g.tx.plan.root, "bin", "haa-alt"), filepath.Join(g.tx.plan.root, "bin", "haa-demo")} {
					if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("rollback left output", path, err)
					}
				}
			}
			if _, err := os.Lstat(filepath.Join(g.tx.plan.root, uvToolHostReceipt)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("UV receipt fabricated")
			}
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPipxPublicationRejectsUVControlWithoutMutation(t *testing.T) {
	g, outputs, receipt, work := pipxPublicationFixture(t)
	wrong, _ := domain.NewProjectControlFile(uvToolHostReceiptControl, receipt.Body(), true)
	if err := g.commitTool(context.Background(), outputs, wrong, work); err == nil {
		t.Fatal("wrong client control accepted")
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(g.tx.plan.root, pipxToolHostReceipt)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("receipt published after wrong control")
	}
}
