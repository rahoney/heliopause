package bootstrap_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/bootstrap"
)

type uvNativeToolFile struct {
	Device, Inode, Links uint64
	Mode                 os.FileMode
	Size                 int64
	SHA256, Link         string
}

func uvNativeToolFiles(t *testing.T, root string) map[string]uvNativeToolFile {
	t.Helper()
	result := map[string]uvNativeToolFile{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return errors.New("native tool fixture identity unavailable")
		}
		file := uvNativeToolFile{Device: uint64(stat.Dev), Inode: stat.Ino, Links: stat.Nlink, Mode: info.Mode(), Size: info.Size()}
		if info.Mode().IsRegular() {
			if info.Size() > 16<<20 {
				return errors.New("native tool fixture file exceeds bounds")
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(body)
			file.SHA256 = hex.EncodeToString(sum[:])
		} else if info.Mode()&os.ModeSymlink != 0 {
			file.Link, err = os.Readlink(path)
			if err != nil {
				return err
			}
		} else {
			return errors.New("native tool fixture has a special file")
		}
		result[path] = file
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func uvNativeToolImageInterpreter(t *testing.T) []byte {
	t.Helper()
	path, expected := os.Getenv("HELOX_TOOL_TARGET_PYTHON_FIXTURE"), os.Getenv("HELOX_TOOL_TARGET_PYTHON_SHA256")
	if !filepath.IsAbs(path) || len(expected) != 64 {
		t.Fatal("declare pinned-image interpreter DATA and receipt")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > 16<<20 {
		t.Fatal("invalid interpreter DATA fixture", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(io.LimitReader(file, (16<<20)+1))
	closeErr := file.Close()
	after, statErr := os.Lstat(path)
	sum := sha256.Sum256(body)
	if readErr != nil || closeErr != nil || statErr != nil || !os.SameFile(before, after) || int64(len(body)) != before.Size() || hex.EncodeToString(sum[:]) != expected || !bytes.HasPrefix(body, []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatal("interpreter DATA differs from image receipt")
	}
	return body
}

func TestLinuxUVToolNativeAdoptionCLIIntegration(t *testing.T) {
	if os.Getenv("HELOX_UV_TOOL_NATIVE_ADOPTION_INTEGRATION") != "1" {
		t.Skip("requires authenticated pinned native UV tool CLI environment")
	}
	if os.Geteuid() == 0 {
		t.Fatal("native tool qualification requires ordinary user")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	t.Cleanup(cancel)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "cache")
	t.Setenv("XDG_CACHE_HOME", cache)
	native := filepath.Join(root, "native")
	if err := os.Mkdir(native, 0700); err != nil {
		t.Fatal(err)
	}
	target, bin := filepath.Join(native, "tools", "cowsay"), filepath.Join(native, "commands")
	uvNativeToolOriginalFixture(t, ctx, target, bin, uvNativeToolImageInterpreter(t), "cowsay==6.0")
	note := filepath.Join(target, "user-note")
	if err := os.WriteFile(note, []byte("preserve native user data"), 0600); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(target, "uv-receipt.toml")
	originalFiles := uvNativeToolFiles(t, native)
	for _, path := range []string{receiptPath, filepath.Join(target, "bin", "cowsay"), filepath.Join(target, "pyvenv.cfg")} {
		file, present := originalFiles[path]
		if !present || !file.Mode.IsRegular() || file.Links != 1 {
			t.Fatal("native fixture data is not a private regular file", path)
		}
	}
	inventory, err := json.Marshal(originalFiles)
	if err != nil || len(inventory) > 64<<10 {
		t.Fatal("native fixture inventory exceeds diagnostic bound", err)
	}
	t.Logf("actual_native_tool_original_inventory=%s", inventory)
	originalReceipt, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := artifactpypi.ParseUVToolReceipt(originalReceipt)
	if err != nil || parsed.Requirement().Requirement() != "cowsay==6.0" {
		t.Fatal("native producer did not retain its actual exact public requirement", err)
	}
	t.Logf("actual_native_tool_raw_receipt=%q", originalReceipt)
	args := []string{"--lang", "en", "--output", "json", "uv", "tool", "upgrade", "cowsay", "--target", target, "--bin-dir", bin}
	refuse := func(flags ...string) {
		t.Helper()
		before := uvNativeToolFiles(t, native)
		var out, diagnostic bytes.Buffer
		err := bootstrap.Run(ctx, append(append([]string{}, args...), flags...), &out, &diagnostic)
		var report struct {
			Error    struct{ Code string }
			Status   string `json:"operation_status"`
			Checks   []json.RawMessage
			Evidence []json.RawMessage `json:"evidence_references"`
		}
		decoded := json.Unmarshal(out.Bytes(), &report)
		expected := report.Error.Code == "UPDATE_PREPARATION_FAILED" || report.Error.Code == "UPDATE_ORIGINAL_STATE_UNSUPPORTED"
		if err == nil || decoded != nil || !expected || report.Status != "FAILED" || len(report.Checks) != 0 || len(report.Evidence) != 0 {
			t.Fatal("native original refusal did not reach the ownership guard", err, out.String(), diagnostic.String())
		}
		if !reflect.DeepEqual(before, uvNativeToolFiles(t, native)) {
			t.Fatal("refused native original changed files/links/identity")
		}
		if _, err := os.Lstat(filepath.Join(target, ".heliopause", "pypi-transaction.json")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("refused native original wrote a ledger", err)
		}
	}
	refuse()
	// A source ownership proof cannot publish this previously unmanaged tool
	// when the current, real Evidence store cannot record the new inspection.
	evidenceRoot := filepath.Join(cache, "heliopause", "evidence")
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	beforeEvidenceFailure := uvNativeToolFiles(t, native)
	if err := os.Chmod(evidenceRoot, 0500); err != nil {
		t.Fatal(err)
	}
	var rejected, diagnostic bytes.Buffer
	currentErr := bootstrap.Run(ctx, append(append([]string{}, args...), "--adopt"), &rejected, &diagnostic)
	restoreErr := os.Chmod(evidenceRoot, 0700)
	if restoreErr != nil || currentErr == nil || !errors.Is(currentErr, fs.ErrPermission) || !strings.Contains(currentErr.Error(), "Evidence") || !reflect.DeepEqual(beforeEvidenceFailure, uvNativeToolFiles(t, native)) {
		t.Fatal("native current Evidence failure bypassed or changed original", currentErr, restoreErr, rejected.String(), diagnostic.String())
	}
	if _, err := os.Lstat(filepath.Join(target, ".heliopause", "pypi-transaction.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("native current Evidence failure left an ownership ledger", err)
	}
	t.Log("actual_native_tool_current_evidence_failure refused=true original_inodes_bytes_modes_links_unchanged=true")
	for _, path := range []string{filepath.Join(target, "bin", "cowsay"), receiptPath} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(append([]byte{}, body...), []byte("\ninvalid native fixture mutation\n")...), 0600); err != nil {
			t.Fatal(err)
		}
		refuse("--adopt")
		if err := os.WriteFile(path, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	seen := map[string]bool{}
	for _, step := range []string{"native-exact-pin", "managed-repeat-adopt", "native-bounded-version-change"} {
		t.Run(step, func(t *testing.T) {
			target, originalReceipt, note := target, originalReceipt, note
			phaseArgs := append([]string{}, args...)
			wantRequirement, _ := artifactpypi.ParseUpgradeRequirement("cowsay==6.0")
			if step == "native-bounded-version-change" {
				bounded := filepath.Join(root, "bounded-native")
				if err := os.Mkdir(bounded, 0700); err != nil {
					t.Fatal(err)
				}
				target = filepath.Join(bounded, "tools", "cowsay")
				boundedBin := filepath.Join(bounded, "commands")
				uvNativeToolOriginalFixture(t, ctx, target, boundedBin, uvNativeToolImageInterpreter(t), "cowsay>=6.0,<=6.1")
				originalReceipt, err = os.ReadFile(filepath.Join(target, "uv-receipt.toml"))
				if err != nil {
					t.Fatal(err)
				}
				old, err := os.ReadFile(filepath.Join(target, "lib/python3.14/site-packages/cowsay-6.1.dist-info/METADATA"))
				if err != nil || !bytes.Contains(old, []byte("Version: 6.1")) {
					t.Fatal("genuine bounded native old6.1 missing", err)
				}
				wantRequirement, _ = artifactpypi.ParseUpgradeRequirement("cowsay>=6.0,<=6.1")
				note = filepath.Join(target, "user-note")
				if err := os.WriteFile(note, []byte("preserve native user data"), 0600); err != nil {
					t.Fatal(err)
				}
				phaseArgs = []string{"--lang", "en", "--output", "json", "uv", "tool", "upgrade", "cowsay==6.0", "--target", target, "--bin-dir", boundedBin}
			}
			var out, diagnostic bytes.Buffer
			if err := bootstrap.Run(ctx, append(phaseArgs, "--adopt"), &out, &diagnostic); err != nil {
				t.Fatal("current native tool adoption failed", err, out.String(), diagnostic.String())
			}
			var report struct {
				Operation string
				Status    string `json:"operation_status"`
				Promotion string `json:"promotion_status"`
				Policy    struct{ Decision string }
				Checks    []struct {
					Required bool
					Status   string `json:"execution_status"`
				}
				Evidence []struct{ Handle string } `json:"evidence_references"`
				Verified struct {
					Manifest string `json:"manifest_id"`
					Count    int    `json:"entry_count"`
				} `json:"verified_set"`
			}
			if json.Unmarshal(out.Bytes(), &report) != nil || report.Operation != "UPDATE" || report.Status != "COMPLETED" || report.Promotion != "COMPLETED" || report.Policy.Decision != "ALLOW" || report.Verified.Manifest == "" || report.Verified.Count != 1 || len(report.Checks) < 3 || len(report.Evidence) < 3 {
				t.Fatal("current tool approval/report incomplete", out.String())
			}
			for _, check := range report.Checks {
				if check.Required && check.Status != "COMPLETED" {
					t.Fatal("native current required check incomplete")
				}
			}
			for _, evidence := range report.Evidence {
				if evidence.Handle == "" || seen[evidence.Handle] {
					t.Fatal("native current Evidence was reused")
				}
				seen[evidence.Handle] = true
			}
			manifest, err := os.ReadFile(filepath.Join(cache, "heliopause", "staging", report.Verified.Manifest, "manifest.json"))
			var record struct {
				Controls []struct {
					Name    string
					Content []byte
				} `json:"project_controls"`
			}
			wanted := 8
			if step != "managed-repeat-adopt" {
				wanted = 10
			}
			if err != nil || json.Unmarshal(manifest, &record) != nil || len(record.Controls) != wanted {
				t.Fatal("native current Manifest/control binding missing", err)
			}
			published, err := os.ReadFile(filepath.Join(target, "uv-receipt.toml"))
			if err != nil {
				t.Fatal(err)
			}
			raw, source, host := false, false, false
			for _, control := range record.Controls {
				switch control.Name {
				case "uv-tool-original-native-receipt.toml":
					raw = bytes.Equal(control.Content, originalReceipt)
				case "pypi-update-native-ownership.json":
					source = len(control.Content) > 0
				case "uv-tool-host-receipt.toml":
					host = bytes.Equal(control.Content, published)
				}
			}
			if !host || (step != "managed-repeat-adopt" && (!raw || !source)) || (step == "managed-repeat-adopt" && (raw || source)) {
				t.Fatal("raw source proof/managed current approval conflated")
			}
			receipt, err := artifactpypi.ParseUVToolReceipt(published)
			if err != nil || !receipt.Requirement().SameConstraints(wantRequirement) {
				t.Fatal("authentic native original constraint changed", err)
			}
			metadata, err := os.ReadFile(filepath.Join(target, "lib/python3.14/site-packages/cowsay-6.0.dist-info/METADATA"))
			if err != nil || !bytes.Contains(metadata, []byte("Version: 6.0")) {
				t.Fatal("selected current native version missing", err)
			}
			if step == "native-bounded-version-change" {
				if _, err := os.Lstat(filepath.Join(target, "lib/python3.14/site-packages/cowsay-6.1.dist-info/METADATA")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("retired authentic native old6.1 remained", err)
				}
			}
			if body, err := os.ReadFile(note); err != nil || string(body) != "preserve native user data" {
				t.Fatal("native user file changed", err)
			}
			t.Logf("actual_native_tool_adoption case=%s controls=%d current_evidence=%d original_requirement=%s selected=6.0", step, wanted, len(seen), wantRequirement.Requirement())
		})
	}
}
