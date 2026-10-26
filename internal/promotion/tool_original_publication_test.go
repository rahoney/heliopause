package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func toolOriginalPublicationFixture(t *testing.T, noop bool) (*toolFreshTarget, []pypiDestination, domain.ProjectControlFile, string) {
	t.Helper()
	install, bin, storage := managedToolOriginalFixture(t)
	request, _ := artifactpypi.ParseUpgradeRequirement("haa-tool-fixture")
	g, err := beginToolOriginalTarget(context.Background(), install, request, bin, []string{storage})
	if err != nil {
		t.Fatal(err)
	}
	fixture := "uv-upgraded.toml"
	version := "1.1.0"
	commands := []string{"haa-demo", "haa-new"}
	if noop {
		fixture = "uv-installed.toml"
		version = "1.0.0"
		commands = []string{"haa-alt", "haa-demo"}
	}
	body, err := os.ReadFile("../artifact/pypi/testdata/persistent-tool/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := toml.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	opts := wire["tool"].(map[string]any)["options"].(map[string]any)
	delete(opts, "no-index")
	delete(opts, "find-links")
	body, err = toml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := artifactpypi.ParseUVToolReceipt(body)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err = receipt.WithUVLayout(g.tx.plan.root, g.binRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.freezeCommands(context.Background(), receipt); err != nil {
		t.Fatal(err)
	}
	control, err := domain.NewProjectControlFile(uvToolHostReceiptControl, receipt.Body(), true)
	if err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(g.tx.plan.root, ".haa-tool-upgrade-test-work")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	values := []struct{ scheme, name, body string }{{"site", "haa_tool_fixture/__init__.py", "new inert module"}, {"site", "haa_tool_fixture-" + version + ".dist-info/METADATA", "Metadata-Version: 2.4\nName: haa-tool-fixture\nVersion: " + version + "\n"}, {"site", "haa_tool_fixture-" + version + ".dist-info/WHEEL", "Wheel-Version: 1.0\nTag: py3-none-any\n"}}
	for _, name := range commands {
		values = append(values, struct{ scheme, name, body string }{"scripts", name, "new inert command " + name})
	}
	var destinations []pypiDestination
	for _, v := range values {
		path := filepath.Join(work, v.scheme, v.name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(v.body), 0700); err != nil {
			t.Fatal(err)
		}
		s, err := snapshotPyPIFile(path)
		if err != nil {
			t.Fatal(err)
		}
		destinations = append(destinations, pypiDestination{Scheme: v.scheme, Relative: v.name, Distribution: "haa-tool-fixture", Version: version, ArtifactDigest: strings.Repeat("b", 64), Source: path, Digest: s.Digest, Size: s.Size})
	}
	return g, destinations, control, work
}

func requireOriginalToolRestored(t *testing.T, g *toolFreshTarget) {
	t.Helper()
	for path, before := range g.tx.before {
		if before != nil && !before.matches(path) {
			t.Fatal("original file identity/bytes/mode not restored", path)
		}
	}
	if g.tx.metadata == nil || !g.tx.metadata.matches(filepath.Join(g.tx.plan.root, pypiVenvMetadata)) {
		t.Fatal("original ownership ledger not restored")
	}
	for _, link := range g.original.links {
		if !toolLinkMatches(g.bin.root, link.Name, link) {
			t.Fatal("original link inode/target not restored", link.Name)
		}
	}
	for _, path := range []string{filepath.Join(g.binRoot, "haa-new"), filepath.Join(g.tx.plan.root, "bin/haa-new")} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("new command survived rollback", path, err)
		}
	}
}

func TestToolCompletePublicationReplacesRetiresAndRestoresNativeOwnership(t *testing.T) {
	for _, noop := range []bool{false, true} {
		t.Run(map[bool]string{false: "changed", true: "same-version"}[noop], func(t *testing.T) {
			g, destinations, receipt, work := toolOriginalPublicationFixture(t, noop)
			if err := g.commitTool(context.Background(), destinations, receipt, work); err != nil {
				t.Fatal(err)
			}
			if !g.original.committed || !g.tx.tool.original.disposed {
				t.Fatal("complete original participant did not finish")
			}
			for _, command := range g.commands {
				if target, err := os.Readlink(command.Path); err != nil || target != filepath.Join(g.tx.plan.root, "bin", command.Name) {
					t.Fatal("new tool command differs", command.Name, err)
				}
			}
			if !noop {
				for _, path := range []string{filepath.Join(g.binRoot, "haa-alt"), filepath.Join(g.tx.plan.root, "bin/haa-alt"), filepath.Join(g.tx.plan.site, "haa_tool_fixture-1.0.0.dist-info/METADATA")} {
					if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("retired output remains", path, err)
					}
				}
			}
			state, _, err := g.tx.plan.readState()
			if err != nil || len(state.Files) != len(destinations) {
				t.Fatal("new complete ownership differs", err)
			}
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{g.tx.backup, g.tx.tool.private, g.tx.tool.original.path, filepath.Join(g.binRoot, toolBinTransactionLock), filepath.Join(g.tx.plan.root, ".heliopause-pypi-transaction.lock")} {
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("successful transaction retained owned state", path, err)
				}
			}
		})
	}
}

func TestToolCompletePublicationFaultsRestoreOriginalLinksFilesAndReceipt(t *testing.T) {
	for _, phase := range []string{"after-publish:tool-control", "metadata", "before-tool-retire:haa-alt", "after-tool-retire:haa-alt", "after-tool-retire:haa-demo", "before-tool-link:haa-demo", "after-tool-link:haa-demo", "after-tool-link:haa-new", "sync", "reconcile"} {
		t.Run(phase, func(t *testing.T) {
			g, destinations, receipt, work := toolOriginalPublicationFixture(t, false)
			primary := errors.New("complete tool fault")
			g.tx.fault = func(at string) error {
				if at == phase {
					return primary
				}
				return nil
			}
			err := g.commitTool(context.Background(), destinations, receipt, work)
			if !errors.Is(err, primary) {
				t.Fatal("primary failure replaced", err)
			}
			requireOriginalToolRestored(t, g)
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestToolCompletePublicationCancellationRestoresWholeOriginal(t *testing.T) {
	g, destinations, receipt, work := toolOriginalPublicationFixture(t, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g.tx.fault = func(at string) error {
		if at == "after-tool-retire:haa-alt" {
			cancel()
		}
		return nil
	}
	if err := g.commitTool(ctx, destinations, receipt, work); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
	requireOriginalToolRestored(t, g)
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestToolCompletePublicationPreservesForeignLinksAndOriginalBackups(t *testing.T) {
	for _, mode := range []string{"published-link", "old-backup", "private-cleanup"} {
		t.Run(mode, func(t *testing.T) {
			g, destinations, receipt, work := toolOriginalPublicationFixture(t, false)
			primary := errors.New("foreign tool mutation")
			var foreign string
			g.tx.fault = func(at string) error {
				if mode == "published-link" && at == "after-tool-link:haa-demo" {
					foreign = filepath.Join(g.binRoot, "haa-demo")
					if err := os.Remove(foreign); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(foreign, []byte("foreign"), 0600); err != nil {
						t.Fatal(err)
					}
					return primary
				}
				if at == "reconcile" && mode != "published-link" {
					root := g.tx.tool.private
					if mode == "old-backup" {
						root = g.tx.tool.original.path
					}
					foreign = filepath.Join(root, "foreign-user")
					if err := os.WriteFile(foreign, []byte("foreign"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return nil
			}
			err := g.commitTool(context.Background(), destinations, receipt, work)
			if err == nil || !g.tx.recovery {
				t.Fatal("foreign state reported success", err)
			}
			if mode == "published-link" && !errors.Is(err, primary) {
				t.Fatal("primary fault lost", err)
			}
			if b, err := os.ReadFile(foreign); err != nil || string(b) != "foreign" {
				t.Fatal("foreign data removed", err)
			}
			if mode == "private-cleanup" {
				requireOriginalToolRestored(t, g)
			}
			_ = g.Close()
			for _, path := range []string{g.tx.backup, filepath.Join(g.binRoot, toolBinTransactionLock), filepath.Join(g.tx.plan.root, ".heliopause-pypi-transaction.lock")} {
				if _, err := os.Lstat(path); err != nil {
					t.Fatal("recovery state not preserved", path, err)
				}
			}
		})
	}
}

func TestToolCompletePublicationKeepsRecoveryOnOriginalBackupDisposalFailure(t *testing.T) {
	g, destinations, receipt, work := toolOriginalPublicationFixture(t, false)
	primary := errors.New("old link disposal unavailable")
	g.tx.fault = func(at string) error {
		if at == "before-tool-original-disposal" {
			return primary
		}
		return nil
	}
	err := g.commitTool(context.Background(), destinations, receipt, work)
	if !errors.Is(err, primary) || !g.tx.recovery || g.original.committed {
		t.Fatal("post-publication cleanup uncertainty reported success", err)
	}
	for _, link := range g.tx.tool.original.links {
		if !link.Backed || link.Discarded || !toolLinkMatches(g.tx.tool.original.root, link.Name, link.toolPublishedLink) {
			t.Fatal("old link backup not retained", link.Name)
		}
	}
	_ = g.Close()
	for _, path := range []string{g.tx.backup, g.tx.tool.original.path, filepath.Join(g.binRoot, toolBinTransactionLock), filepath.Join(g.tx.plan.root, ".heliopause-pypi-transaction.lock")} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatal("post-commit recovery missing", path, err)
		}
	}
}

func TestToolCompletePublicationRejectsNewCommandCollisionsBeforeAndAfterMutation(t *testing.T) {
	for _, mode := range []string{"external-before", "internal-before", "case-before", "case-during"} {
		t.Run(mode, func(t *testing.T) {
			g, destinations, receipt, work := toolOriginalPublicationFixture(t, false)
			foreign := filepath.Join(g.binRoot, "haa-new")
			if mode == "internal-before" {
				foreign = filepath.Join(g.tx.plan.root, "bin/haa-new")
			}
			if mode == "case-before" || mode == "case-during" {
				foreign = filepath.Join(g.binRoot, "HAA-NEW")
			}
			add := func() {
				if err := os.WriteFile(foreign, []byte("keep foreign"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "case-during" {
				g.tx.fault = func(at string) error {
					if at == "after-tool-link:haa-new" {
						add()
					}
					return nil
				}
			} else {
				add()
			}
			if err := g.commitTool(context.Background(), destinations, receipt, work); err == nil {
				t.Fatal("new command collision accepted")
			}
			if b, err := os.ReadFile(foreign); err != nil || string(b) != "keep foreign" {
				t.Fatal("foreign collision changed", err)
			}
			for _, link := range g.original.links {
				if !toolLinkMatches(g.bin.root, link.Name, link) {
					t.Fatal("old command did not survive collision", link.Name)
				}
			}
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
