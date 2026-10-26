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

func managedToolOriginalFixture(t *testing.T) (domain.InstallContext, string, string) {
	t.Helper()
	g, destinations, receipt, work := toolPublicationFixture(t)
	for _, value := range []struct{ name, body string }{{"METADATA", "Metadata-Version: 2.4\nName: haa-tool-fixture\nVersion: 1.0.0\n"}, {"WHEEL", "Wheel-Version: 1.0\nTag: py3-none-any\n"}} {
		relative := "haa_tool_fixture-1.0.0.dist-info/" + value.name
		source := filepath.Join(work, "site", relative)
		if err := os.MkdirAll(filepath.Dir(source), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(source, []byte(value.body), 0600); err != nil {
			t.Fatal(err)
		}
		before, err := snapshotPyPIFile(source)
		if err != nil {
			t.Fatal(err)
		}
		destinations = append(destinations, pypiDestination{Scheme: "site", Relative: relative, Distribution: "haa-tool-fixture", Version: "1.0.0", ArtifactDigest: strings.Repeat("a", 64), Source: source, Digest: before.Digest, Size: before.Size})
	}
	var wire map[string]any
	if err := toml.Unmarshal(receipt.Body(), &wire); err != nil {
		t.Fatal(err)
	}
	options := wire["tool"].(map[string]any)["options"].(map[string]any)
	delete(options, "no-index")
	delete(options, "find-links")
	body, err := toml.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := artifactpypi.ParseUVToolReceipt(body)
	if err != nil {
		t.Fatal(err)
	}
	control, err := domain.NewProjectControlFile(uvToolHostReceiptControl, parsed.Body(), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.commitTool(context.Background(), destinations, control, work); err != nil {
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

func TestToolOriginalGuardRetainsOwnedWholeStateAndSeparateRequest(t *testing.T) {
	install, bin, storage := managedToolOriginalFixture(t)
	request, _ := artifactpypi.ParseUpgradeRequirement("haa-tool-fixture>=1.1,<2")
	g, err := beginToolOriginalTarget(context.Background(), install, request, bin, []string{storage})
	if err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(bin, "unrelated-user-file")
	if err := os.WriteFile(note, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if g.original == nil || g.original.input.Request().Requirement() != request.Requirement() || !g.original.input.Receipt().Requirement().AcceptsVersion("1.0.0") {
		t.Fatal("original and positional input collapsed")
	}
	if other, err := beginToolOriginalTarget(context.Background(), install, request, bin, []string{storage}); err == nil {
		_ = other.Close()
		t.Fatal("concurrent original entered")
	}
	if err := g.verify(context.Background()); err != nil {
		t.Fatal(err)
	}
	receipt := g.original.input.Receipt()
	control, _ := domain.NewProjectControlFile(uvToolHostReceiptControl, receipt.Body(), true)
	if err := g.freezeCommands(context.Background(), receipt); err == nil {
		t.Fatal("fresh command publication accepted original")
	}
	if err := g.commitTool(context.Background(), nil, control, ""); err == nil {
		t.Fatal("fresh transaction accepted original")
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(bin, toolBinTransactionLock), filepath.Join(install.Target().String(), ".heliopause-pypi-transaction.lock")} {
		if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("guard retained", p, err)
		}
	}
	if b, err := os.ReadFile(note); err != nil || string(b) != "keep" {
		t.Fatal("unrelated command data changed")
	}
}

func TestToolOriginalGuardRejectsUnownedAndChangedOriginalBeforeNative(t *testing.T) {
	for _, mode := range []string{"missing-receipt", "payload-drift", "unmanaged-distribution", "missing-owner", "nonexecutable-main-script", "different-link", "regular-command", "case-collision", "aliased-link", "local-recipe", "wrong-layout"} {
		t.Run(mode, func(t *testing.T) {
			install, bin, storage := managedToolOriginalFixture(t)
			root := install.Target().String()
			receipt := filepath.Join(root, uvToolHostReceipt)
			link := filepath.Join(bin, "haa-demo")
			switch mode {
			case "missing-receipt":
				if err := os.Remove(receipt); err != nil {
					t.Fatal(err)
				}
			case "payload-drift":
				if err := os.WriteFile(filepath.Join(root, "lib/python3.14/site-packages/haa_tool_fixture/__init__.py"), []byte("changed"), 0700); err != nil {
					t.Fatal(err)
				}
			case "unmanaged-distribution":
				dir := filepath.Join(root, "lib/python3.14/site-packages/other-1.0.0.dist-info")
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "METADATA"), []byte("Metadata-Version: 2.4\nName: other\nVersion: 1.0.0\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing-owner":
				if err := os.Remove(filepath.Join(root, pypiVenvMetadata)); err != nil {
					t.Fatal(err)
				}
			case "nonexecutable-main-script":
				if err := os.Chmod(filepath.Join(root, "bin/haa-demo"), 0600); err != nil {
					t.Fatal(err)
				}
			case "different-link", "regular-command":
				if err := os.Remove(link); err != nil {
					t.Fatal(err)
				}
				if mode == "different-link" {
					if err := os.Symlink("/unrelated/program", link); err != nil {
						t.Fatal(err)
					}
				} else if err := os.WriteFile(link, []byte("user"), 0600); err != nil {
					t.Fatal(err)
				}
			case "case-collision":
				if err := os.WriteFile(filepath.Join(bin, "HAA-DEMO"), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			case "aliased-link":
				if err := os.Link(link, filepath.Join(bin, "user-link-alias")); err != nil {
					t.Fatal(err)
				}
			case "local-recipe":
				f, err := os.OpenFile(receipt, os.O_APPEND|os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				_, e := f.WriteString("find-links = [\"/private/wheels\"]\n")
				if err := errors.Join(e, f.Close()); err != nil {
					t.Fatal(err)
				}
			case "wrong-layout":
				b, err := os.ReadFile(receipt)
				if err != nil {
					t.Fatal(err)
				}
				parsed, err := artifactpypi.ParseUVToolReceipt(b)
				if err != nil {
					t.Fatal(err)
				}
				other, err := parsed.WithUVLayout(root, filepath.Join(filepath.Dir(root), "other-bin"))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(receipt, other.Body(), 0600); err != nil {
					t.Fatal(err)
				}
			}
			request, _ := artifactpypi.ParseUpgradeRequirement("haa-tool-fixture")
			g, err := beginToolOriginalTarget(context.Background(), install, request, bin, []string{storage})
			if err == nil {
				_ = g.Close()
				t.Fatal("invalid original accepted")
			}
			if g != nil {
				t.Fatal("failed guard returned usable original")
			}
			for _, p := range []string{filepath.Join(bin, toolBinTransactionLock), filepath.Join(root, ".heliopause-pypi-transaction.lock")} {
				if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("refused original leaked lock", p, err)
				}
			}
		})
	}
}

func TestToolOriginalGuardDetectsDriftWhileHoldingBothRoots(t *testing.T) {
	for _, mode := range []string{"receipt", "payload", "new-distribution", "same-link-new-inode", "aliased-link", "interpreter"} {
		t.Run(mode, func(t *testing.T) {
			install, bin, storage := managedToolOriginalFixture(t)
			root := install.Target().String()
			request, _ := artifactpypi.ParseUpgradeRequirement("haa-tool-fixture")
			g, err := beginToolOriginalTarget(context.Background(), install, request, bin, []string{storage})
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "receipt":
				path := filepath.Join(root, uvToolHostReceipt)
				b, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(b, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
			case "payload":
				if err := os.WriteFile(filepath.Join(root, "lib/python3.14/site-packages/haa_tool_fixture/__init__.py"), []byte("changed"), 0700); err != nil {
					t.Fatal(err)
				}
			case "new-distribution":
				if err := os.Mkdir(filepath.Join(root, "lib/python3.14/site-packages/other-1.0.0.dist-info"), 0700); err != nil {
					t.Fatal(err)
				}
			case "same-link-new-inode":
				path := filepath.Join(bin, "haa-demo")
				other := filepath.Join(bin, "prepared-other-link")
				if err := os.Symlink(filepath.Join(root, "bin/haa-demo"), other); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(other, path); err != nil {
					t.Fatal(err)
				}
			case "aliased-link":
				if err := os.Link(filepath.Join(bin, "haa-demo"), filepath.Join(bin, "foreign-alias")); err != nil {
					t.Fatal(err)
				}
			case "interpreter":
				if err := os.WriteFile(filepath.Join(root, "bin/python3.14"), []byte("changed interpreter"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if err := g.verify(context.Background()); err == nil {
				t.Fatal("original drift accepted")
			}
			closeErr := g.Close()
			linkChanged := mode == "same-link-new-inode" || mode == "aliased-link"
			if linkChanged && (closeErr == nil || !g.tx.recovery) {
				t.Fatal("uncertain link did not preserve recovery")
			}
			for _, p := range []string{filepath.Join(bin, toolBinTransactionLock), filepath.Join(root, ".heliopause-pypi-transaction.lock")} {
				_, err := os.Lstat(p)
				if linkChanged && err != nil || !linkChanged && !errors.Is(err, os.ErrNotExist) {
					t.Fatal("guard disposition differs", p, err, closeErr)
				}
			}
		})
	}
}
