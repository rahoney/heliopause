package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func toolPublicationFixture(t *testing.T) (*toolFreshTarget, []pypiDestination, domain.ProjectControlFile, string) {
	t.Helper()
	install, requirement, bin, storage := toolTargetFixture(t)
	g, err := beginToolFreshTarget(context.Background(), install, requirement, bin, []string{storage})
	if err != nil {
		t.Fatal(err)
	}
	receipt := toolTargetReceipt(t, install, bin)
	if err := g.freezeCommands(context.Background(), receipt); err != nil {
		t.Fatal(err)
	}
	control, err := domain.NewProjectControlFile(uvToolHostReceiptControl, receipt.Body(), true)
	if err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(install.Target().String(), ".haa-tool-test-work")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	var destinations []pypiDestination
	for _, value := range []struct{ scheme, name, body string }{{"site", "haa_tool_fixture/__init__.py", "inert module bytes"}, {"scripts", "haa-alt", "inert command one"}, {"scripts", "haa-demo", "inert command two"}} {
		path := filepath.Join(work, value.scheme, value.name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value.body), 0700); err != nil {
			t.Fatal(err)
		}
		snapshot, err := snapshotPyPIFile(path)
		if err != nil {
			t.Fatal(err)
		}
		destinations = append(destinations, pypiDestination{Scheme: value.scheme, Relative: value.name, Distribution: "haa-tool-fixture", Version: "1.0.0", ArtifactDigest: strings.Repeat("a", 64), Source: path, Digest: snapshot.Digest, Size: snapshot.Size})
	}
	return g, destinations, control, work
}

func requireToolOriginalsRestored(t *testing.T, g *toolFreshTarget) {
	t.Helper()
	for _, path := range []string{filepath.Join(g.tx.plan.root, uvToolHostReceipt), filepath.Join(g.tx.plan.root, pypiVenvMetadata), filepath.Join(g.tx.plan.site, "haa_tool_fixture/__init__.py"), filepath.Join(g.tx.plan.root, "bin/haa-alt"), filepath.Join(g.tx.plan.root, "bin/haa-demo"), filepath.Join(g.binRoot, "haa-alt"), filepath.Join(g.binRoot, "haa-demo")} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("tool output survived rollback: %s (%v)", path, err)
		}
	}
	if !g.tx.config.matches(filepath.Join(g.tx.plan.root, "pyvenv.cfg")) {
		t.Fatal("original venv configuration changed")
	}
}

func TestToolPublicationCommitsNativeReceiptLinksAndWheelOwnershipTogether(t *testing.T) {
	g, destinations, receipt, work := toolPublicationFixture(t)
	if err := g.commitTool(context.Background(), destinations, receipt, work); err != nil {
		t.Fatal(err)
	}
	for _, command := range g.commands {
		link, err := os.Readlink(command.Path)
		if err != nil || link != filepath.Join(g.tx.plan.root, "bin", command.Name) {
			t.Fatal("exposed command binding differs")
		}
	}
	body, err := os.ReadFile(filepath.Join(g.tx.plan.root, uvToolHostReceipt))
	if err != nil || string(body) != string(receipt.Body()) {
		t.Fatal("native receipt differs from approved control")
	}
	state, _, err := g.tx.plan.readState()
	if err != nil || len(state.Files) != 3 {
		t.Fatal("distribution ownership changed")
	}
	for _, d := range state.Files {
		if d.Scheme == "tool-control" || d.Relative == uvToolHostReceipt {
			t.Fatal("native receipt disguised as distribution file")
		}
	}
	if _, err := os.Lstat(g.tx.tool.private); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("prepared link workspace retained after success")
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(g.binRoot, toolBinTransactionLock), filepath.Join(g.tx.plan.root, ".heliopause-pypi-transaction.lock")} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("success retained mutation guard")
		}
	}
}

func TestToolPublicationFaultsRollbackReceiptLinksAndWholeVenv(t *testing.T) {
	for _, phase := range []string{"after-publish:tool-control", "metadata", "before-tool-link:haa-alt", "after-tool-link:haa-alt", "after-tool-link:haa-demo", "sync", "reconcile"} {
		t.Run(phase, func(t *testing.T) {
			g, destinations, receipt, work := toolPublicationFixture(t)
			primary := errors.New("injected tool publication fault")
			g.tx.fault = func(at string) error {
				if at == phase {
					return primary
				}
				return nil
			}
			if err := g.commitTool(context.Background(), destinations, receipt, work); !errors.Is(err, primary) {
				t.Fatalf("primary error lost: %v", err)
			}
			requireToolOriginalsRestored(t, g)
			if g.tx.recovery {
				t.Fatal("ordinary fault required recovery")
			}
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestToolPublicationCancellationAfterFirstLinkRollsBack(t *testing.T) {
	g, destinations, receipt, work := toolPublicationFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g.tx.fault = func(at string) error {
		if at == "after-tool-link:haa-alt" {
			cancel()
		}
		return nil
	}
	if err := g.commitTool(ctx, destinations, receipt, work); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	requireToolOriginalsRestored(t, g)
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestToolPublicationForeignPublishedLinkRetainsJournalAndBothGuards(t *testing.T) {
	g, destinations, receipt, work := toolPublicationFixture(t)
	primary := errors.New("foreign link race")
	g.tx.fault = func(at string) error {
		if at != "after-tool-link:haa-alt" {
			return nil
		}
		path := filepath.Join(g.binRoot, "haa-alt")
		if err := os.Rename(path, path+".saved"); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("/foreign/tool", path); err != nil {
			t.Fatal(err)
		}
		return primary
	}
	if err := g.commitTool(context.Background(), destinations, receipt, work); !errors.Is(err, primary) || !g.tx.recovery {
		t.Fatalf("uncertain rollback not retained: %v", err)
	}
	if link, err := os.Readlink(filepath.Join(g.binRoot, "haa-alt")); err != nil || link != "/foreign/tool" {
		t.Fatal("foreign link removed")
	}
	ledger, err := os.ReadFile(filepath.Join(g.tx.backup, "ledger.json"))
	if err != nil || !strings.Contains(string(ledger), "ToolLink") || !strings.Contains(string(ledger), "PrivateRoot") {
		t.Fatal("tool recovery identity missing")
	}
	_ = g.Close()
	for _, path := range []string{g.tx.backup, filepath.Join(g.binRoot, toolBinTransactionLock), filepath.Join(g.tx.plan.root, ".heliopause-pypi-transaction.lock")} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatal("uncertain recovery data removed")
		}
	}
}

func TestToolPublicationLatePrivateCleanupFailureRestoresVenvAndPreservesForeignData(t *testing.T) {
	g, destinations, receipt, work := toolPublicationFixture(t)
	var foreign string
	g.tx.fault = func(at string) error {
		if at == "reconcile" {
			foreign = filepath.Join(g.tx.tool.private, "foreign")
			if err := os.WriteFile(foreign, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}
	if err := g.commitTool(context.Background(), destinations, receipt, work); err == nil || !g.tx.recovery {
		t.Fatal("late cleanup uncertainty accepted")
	}
	requireToolOriginalsRestored(t, g)
	if body, err := os.ReadFile(foreign); err != nil || string(body) != "keep" {
		t.Fatal("foreign private bytes lost")
	}
	_ = g.Close()
	for _, path := range []string{g.tx.backup, filepath.Join(g.binRoot, toolBinTransactionLock), filepath.Join(g.tx.plan.root, ".heliopause-pypi-transaction.lock")} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatal("cleanup recovery boundary removed")
		}
	}
}

func TestToolPublicationReleaseStillRejectsChangedPersistentRoots(t *testing.T) {
	g, destinations, receipt, work := toolPublicationFixture(t)
	if err := g.commitTool(context.Background(), destinations, receipt, work); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(g.binRoot, 0750); err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err == nil {
		t.Fatal("changed persistent bin root accepted at release")
	}
	for _, path := range []string{filepath.Join(g.binRoot, toolBinTransactionLock), filepath.Join(g.tx.plan.root, ".heliopause-pypi-transaction.lock")} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatal("uncertain persistent root guard removed")
		}
	}
}
