package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func toolTargetFixture(t *testing.T) (domain.InstallContext, artifactpypi.UpgradeRequirement, string, string) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("tool target publication is Linux amd64 only")
	}
	parent := realPromotionRoot(t)
	root := filepath.Join(parent, "haa-tool-fixture")
	bin := filepath.Join(parent, "commands")
	storage := filepath.Join(parent, "staging")
	for _, path := range []string{filepath.Join(root, "bin"), filepath.Join(root, "lib/python3.14/site-packages"), bin, storage} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "pyvenv.cfg"), []byte("version = 3.14.7\ninclude-system-site-packages = false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	// An inert executable fixture is fingerprinted as data and never launched.
	if err := os.WriteFile(filepath.Join(root, "bin/python3.14"), []byte("inert interpreter bytes"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("python3.14", filepath.Join(root, "bin/python")); err != nil {
		t.Fatal(err)
	}
	requirement, err := artifactpypi.ParseUpgradeRequirement("haa-tool-fixture>=1,<2")
	if err != nil {
		t.Fatal(err)
	}
	return toolMaterializerContext(t, root), requirement, bin, storage
}

func toolTargetReceipt(t *testing.T, install domain.InstallContext, bin string) artifactpypi.ToolReceipt {
	t.Helper()
	body, err := os.ReadFile("../artifact/pypi/testdata/persistent-tool/uv-installed.toml")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := artifactpypi.ParseUVToolReceipt(body)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := receipt.WithUVLayout(install.Target().String(), bin)
	if err != nil {
		t.Fatal(err)
	}
	return bound
}

func TestToolTargetFreezesBothRootsAndCommandAbsence(t *testing.T) {
	install, requirement, bin, storage := toolTargetFixture(t)
	guard, err := beginToolFreshTarget(context.Background(), install, requirement, bin, []string{storage})
	if err != nil {
		t.Fatal(err)
	}
	if other, err := beginToolFreshTarget(context.Background(), install, requirement, bin, []string{storage}); err == nil {
		_ = other.Close()
		t.Fatal("concurrent target entered")
	}
	if err := guard.freezeCommands(context.Background(), toolTargetReceipt(t, install, bin)); err != nil {
		t.Fatal(err)
	}
	if err := guard.freezeCommands(context.Background(), toolTargetReceipt(t, install, bin)); err == nil {
		t.Fatal("repeated selection accepted")
	}
	path := filepath.Join(bin, "haa-demo")
	if err := os.WriteFile(path, []byte("foreign command"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := guard.verify(context.Background()); err == nil {
		t.Fatal("late command collision accepted")
	}
	if err := guard.Close(); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "foreign command" {
		t.Fatal("foreign command removed")
	}
	for _, path := range []string{filepath.Join(bin, toolBinTransactionLock), filepath.Join(install.Target().String(), ".heliopause-pypi-transaction.lock")} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("clean guard leaked")
		}
	}
}

func TestToolTargetRefusesOriginalDistributionReceiptAndUnsafeRoots(t *testing.T) {
	for _, mutation := range []string{"receipt", "site", "version", "writable-bin", "overlap", "namespace", "missing-interpreter", "alias-cycle"} {
		t.Run(mutation, func(t *testing.T) {
			install, requirement, bin, storage := toolTargetFixture(t)
			root := install.Target().String()
			var err error
			switch mutation {
			case "receipt":
				err = os.WriteFile(filepath.Join(root, uvToolHostReceipt), []byte("native original"), 0600)
			case "site":
				err = os.WriteFile(filepath.Join(root, "lib/python3.14/site-packages/foreign.pth"), []byte("inert"), 0600)
			case "version":
				err = os.WriteFile(filepath.Join(root, "pyvenv.cfg"), []byte("version = 3.14.6\n"), 0600)
			case "writable-bin":
				err = os.Chmod(bin, 0777)
			case "overlap":
				bin = filepath.Join(root, "bin")
			case "namespace":
				requirement, err = artifactpypi.ParseUpgradeRequirement("other>=1")
			case "missing-interpreter":
				err = os.Remove(filepath.Join(root, "bin/python3.14"))
			case "alias-cycle":
				err = os.Remove(filepath.Join(root, "bin/python3.14"))
				if err == nil {
					err = os.Symlink("python", filepath.Join(root, "bin/python3.14"))
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if result, err := beginToolFreshTarget(context.Background(), install, requirement, bin, []string{storage}); err == nil || result != nil {
				if result != nil {
					_ = result.Close()
				}
				t.Fatal("unsupported target accepted")
			}
			if _, err := os.Lstat(filepath.Join(root, ".heliopause-pypi-transaction.lock")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed preflight leaked venv lock")
			}
		})
	}
}

func TestToolTargetPreservesOccupiedCommandsAndInterpreterDrift(t *testing.T) {
	for _, mutation := range []string{"collision", "case-collision", "interpreter-bytes", "interpreter-alias"} {
		t.Run(mutation, func(t *testing.T) {
			install, requirement, bin, storage := toolTargetFixture(t)
			guard, err := beginToolFreshTarget(context.Background(), install, requirement, bin, []string{storage})
			if err != nil {
				t.Fatal(err)
			}
			var foreign string
			switch mutation {
			case "collision":
				foreign = filepath.Join(bin, "haa-demo")
				err = os.WriteFile(foreign, []byte("keep"), 0600)
			case "case-collision":
				foreign = filepath.Join(bin, "HAA-DEMO")
				err = os.WriteFile(foreign, []byte("keep"), 0600)
			case "interpreter-bytes":
				err = os.WriteFile(filepath.Join(install.Target().String(), "bin/python3.14"), []byte("changed inert bytes"), 0700)
			case "interpreter-alias":
				path := filepath.Join(install.Target().String(), "bin/python")
				err = os.Remove(path)
				if err == nil {
					err = os.Symlink("foreign-python", path)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := guard.freezeCommands(context.Background(), toolTargetReceipt(t, install, bin)); err == nil {
				t.Fatal("foreign state accepted")
			}
			if err := guard.Close(); err != nil {
				t.Fatal(err)
			}
			if foreign != "" {
				body, err := os.ReadFile(foreign)
				if err != nil || string(body) != "keep" {
					t.Fatal("foreign command lost")
				}
			}
		})
	}
}

func TestToolTargetLockDriftRetainsBothRecoveryGuards(t *testing.T) {
	install, requirement, bin, storage := toolTargetFixture(t)
	guard, err := beginToolFreshTarget(context.Background(), install, requirement, bin, []string{storage})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(bin, toolBinTransactionLock)
	if err := os.WriteFile(path, []byte("foreign lock"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := guard.Close(); err == nil {
		t.Fatal("changed lock removed")
	}
	for _, path := range []string{path, filepath.Join(install.Target().String(), ".heliopause-pypi-transaction.lock")} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatal("uncertain guard lost")
		}
	}
}
