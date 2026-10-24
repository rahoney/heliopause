package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

func nativeRecordFixture(t *testing.T, plan pypiVenvPlan, extra map[string]string) artifactpypi.UpgradeInput {
	t.Helper()
	metadata := "Name: native\nVersion: 1.0\n\n"
	files := map[string]string{"native.py": "value = 1\n", "native-1.0.dist-info/METADATA": metadata}
	for name, body := range extra {
		files[name] = body
	}
	var record bytes.Buffer
	writer := csv.NewWriter(&record)
	for name, body := range files {
		path := filepath.Join(plan.site, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(body))
		if err := writer.Write([]string{name, "sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]), strconv.Itoa(len(body))}); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Write([]string{"native-1.0.dist-info/RECORD", "", ""}); err != nil {
		t.Fatal(err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plan.site, "native-1.0.dist-info/RECORD"), record.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	requirement, _ := artifactpypi.ParseUpgradeRequirement("native")
	input, err := artifactpypi.NewUpgradeInput(requirement, []artifactpypi.InstalledMetadataFile{{Directory: "native-1.0.dist-info", Body: []byte(metadata)}}, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestPyPIUpgradeRejectsPartialLedgerBeforeSelectionWithoutAdoption(t *testing.T) {
	for _, owner := range []string{"native", "retained"} {
		t.Run(owner, func(t *testing.T) {
			_, plan := makePypiVenvFixture(t)
			runVenvFixture(t, plan, map[string]string{"site/" + owner + ".py": "value = 1\n"}, owner)
			_ = nativeRecordFixture(t, plan, nil)
			target, _ := domain.NewInstallTarget(plan.root)
			install, _ := domain.NewPythonVenvInstallContext(target)
			resolver := &pypiUpgradeSelectionFixture{}
			promoter, _ := newPyPIPromotion(filepath.Join(realPromotionRoot(t), "staging"), unavailableDockerRunner{}, "linux", "amd64")
			preparation, _ := NewPyPIUpdatePreparation(promoter, resolver)
			source, _ := domain.NewSourceID("pypi")
			session, err := preparation.BeginInstallUpdate(context.Background(), source, "native", install, false)
			if err == nil {
				_ = session.Close()
				t.Fatal("partial HAA ownership was accepted without exact owned METADATA")
			}
			if ports.InstallUpdateFailureCode(err) != "UPDATE_OWNERSHIP_INVALID" || resolver.calls != 0 {
				t.Fatal("partial ownership reached selection or lost its reason", err)
			}
		})
	}
}

func TestNativePyPIRecordUsesActualVenvDepthWithoutPrivateOutputAliases(t *testing.T) {
	plan := pypiVenvPlan{root: "/tmp/native-venv", site: "/tmp/native-venv/lib/python3.14/site-packages"}
	for value, expected := range map[string]string{"native/module.py": plan.site + "/native/module.py", "../../../bin/tool": plan.root + "/bin/tool", "../../../share/man/tool.1": plan.root + "/share/man/tool.1", "../../../include/site/python3.14/native/header.h": plan.root + "/include/site/python3.14/native/header.h"} {
		destination, err := nativePyPIRecordDestination(plan, value)
		if err != nil || destination.Final != expected {
			t.Fatal(value, destination, err)
		}
	}
	for _, value := range []string{"", "/tmp/escape", "C:\\escape", "native/../other.py", "native//a.py", "native/./a.py", "../../bin/tool", "../../../../bin/tool", "../../../pyvenv.cfg", "../../../.heliopause/state", "../../../lib/python3.14/site-packages/reentry.py", "../../../bin/../share/reentry", "native/a.py\n"} {
		if _, err := nativePyPIRecordDestination(plan, value); err == nil {
			t.Fatal("untrusted alias or private recipe admitted", value)
		}
	}
}

func TestNativePyPIInventoryOnlyFreezesClaimsAndDetectsOriginalDrift(t *testing.T) {
	for _, mode := range []string{"payload", "record", "metadata"} {
		t.Run(mode, func(t *testing.T) {
			root, plan := makePypiVenvFixture(t)
			input := nativeRecordFixture(t, plan, map[string]string{"../../../bin/native-tool": "harmless script\n"})
			transaction, err := beginPyPIVenvTransaction(plan)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := transaction.close(); err != nil {
					t.Fatal(err)
				}
			}()
			inventory, err := capturePyPINativeInventory(context.Background(), transaction, input)
			if err != nil || len(inventory.projects) != 1 || len(inventory.projects["native"].files) != 4 || len(transaction.current.Files) != 0 || transaction.metadata != nil {
				t.Fatal("raw claims became ownership or were lost", inventory, err)
			}
			if _, err := os.Lstat(filepath.Join(root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("pre-inspection inventory wrote ownership", err)
			}
			path := filepath.Join(plan.site, "native.py")
			if mode == "record" {
				path = filepath.Join(plan.site, "native-1.0.dist-info/RECORD")
			}
			if mode == "metadata" {
				path = filepath.Join(plan.site, "native-1.0.dist-info/METADATA")
			}
			if err := os.WriteFile(path, []byte("concurrent user edit"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := transaction.checkBefore(); err == nil {
				t.Fatal("native original drift was ignored")
			}
		})
	}
}

func TestNativePyPIInventoryRejectsFalsePathsLinksHashesAndAmbiguousClaims(t *testing.T) {
	for _, mode := range []string{"escape", "private-depth", "symlink", "hardlink", "false-hash", "duplicate", "missing-self", "cancelled", "oversize", "metadata-oversize"} {
		t.Run(mode, func(t *testing.T) {
			_, plan := makePypiVenvFixture(t)
			input := nativeRecordFixture(t, plan, nil)
			recordPath := filepath.Join(plan.site, "native-1.0.dist-info/RECORD")
			body, err := os.ReadFile(recordPath)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "escape":
				body = append(body, []byte("../../../pyvenv.cfg,,\n")...)
			case "private-depth":
				body = append(body, []byte("../../bin/native-tool,,\n")...)
			case "symlink":
				path := filepath.Join(plan.site, "native.py")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(recordPath, path); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(filepath.Join(plan.site, "native.py"), filepath.Join(plan.site, "user-file")); err != nil {
					t.Fatal(err)
				}
			case "false-hash":
				body = []byte(strings.ReplaceAll(string(body), "sha256=", "sha256=A"))
			case "duplicate":
				body = append(body, []byte("native-1.0.dist-info/RECORD,,\n")...)
			case "missing-self":
				body = []byte(strings.ReplaceAll(string(body), "native-1.0.dist-info/RECORD,,\n", ""))
			case "oversize", "metadata-oversize":
				path := filepath.Join(plan.site, "native.py")
				limit := nativePyPIFileLimit
				if mode == "metadata-oversize" {
					path = filepath.Join(plan.site, "native-1.0.dist-info/METADATA")
					limit = nativePyPIRecordLimit
				}
				file, err := os.OpenFile(path, os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				err = file.Truncate(limit + 1)
				closeErr := file.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
			}
			if err := os.WriteFile(recordPath, body, 0o600); err != nil {
				t.Fatal(err)
			}
			transaction, err := beginPyPIVenvTransaction(plan)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			_, err = capturePyPINativeInventory(ctx, transaction, input)
			if err == nil || len(transaction.current.Files) != 0 || mode == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatal("unproven/invalid native claims admitted", err)
			}
			if (mode == "oversize" || mode == "metadata-oversize") && !strings.Contains(err.Error(), "exceeds bounds") {
				t.Fatal("oversize file was read before size rejection", err)
			}
			if err := transaction.close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNativePyPIInventoryNeverReinterpretsManagedPrivateRecord(t *testing.T) {
	root, plan, _ := managedPyPIUpgradeFixture(t)
	// This owner has no native RECORD at all; its validated ledger is authority.
	metadata, err := os.ReadFile(filepath.Join(plan.site, "fixture-1.0.dist-info/METADATA"))
	if err != nil {
		t.Fatal(err)
	}
	requirement, _ := artifactpypi.ParseUpgradeRequirement("fixture")
	input, err := artifactpypi.NewUpgradeInput(requirement, []artifactpypi.InstalledMetadataFile{{Directory: "fixture-1.0.dist-info", Body: metadata}}, "3.14.7")
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := beginPyPIVenvTransaction(plan)
	if err != nil {
		t.Fatal(err)
	}
	before := len(transaction.current.Files)
	inventory, err := capturePyPINativeInventory(context.Background(), transaction, input)
	if err != nil || len(inventory.projects) != 0 || len(transaction.current.Files) != before {
		t.Fatal("managed state was reinterpreted", root, err)
	}
	if err := transaction.close(); err != nil {
		t.Fatal(err)
	}
}
