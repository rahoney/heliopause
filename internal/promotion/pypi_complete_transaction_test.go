package promotion

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func heldCompleteTestTransaction(t *testing.T, plan pypiVenvPlan) (*pypiVenvTransaction, func()) {
	t.Helper()
	tx, err := beginPyPIVenvTransaction(plan)
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	closeTransaction := func() {
		if !closed {
			closed = true
			if err := tx.close(); err != nil {
				t.Error(err)
			}
		}
	}
	t.Cleanup(closeTransaction)
	return tx, closeTransaction
}

func completeVenvOriginals(t *testing.T) pypiVenvPlan {
	t.Helper()
	_, plan := makePypiVenvFixture(t)
	runVenvFixture(t, plan, map[string]string{"site/alpha.py": "old alpha", "site/alpha-1.0.dist-info/METADATA": "Name: alpha\nVersion: 1.0\n\n"}, "alpha")
	runVenvFixture(t, plan, map[string]string{"site/beta.py": "old beta", "site/beta-1.0.dist-info/METADATA": "Name: beta\nVersion: 1.0\n\n", "scripts/beta-tool": "old beta tool"}, "beta")
	if err := os.WriteFile(filepath.Join(plan.site, "user-notes.txt"), []byte("preserve user data"), 0o600); err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestCompleteVenvRetiresAllProvenUnselectedOwners(t *testing.T) {
	plan := completeVenvOriginals(t)
	cfg, err := snapshotPyPIFile(filepath.Join(plan.root, "pyvenv.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	tx, _ := heldCompleteTestTransaction(t, plan)
	desired := venvDestinations(t, map[string]string{"site/alpha.py": "new alpha", "site/alpha-1.0.dist-info/METADATA": "Name: alpha\nVersion: 1.0\n\n"}, "alpha")
	if err := tx.commitComplete(desired, false); err != nil {
		t.Fatal(err)
	}
	state, present, err := plan.readState()
	if err != nil || !present || len(state.Files) != len(desired) {
		t.Fatal("complete ownership retained an unselected owner", err)
	}
	for _, path := range []string{filepath.Join(plan.site, "beta.py"), filepath.Join(plan.site, "beta-1.0.dist-info/METADATA"), filepath.Join(plan.root, "bin/beta-tool")} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("unselected proven output remains", path, err)
		}
	}
	if body, err := os.ReadFile(filepath.Join(plan.site, "user-notes.txt")); err != nil || string(body) != "preserve user data" || !cfg.matches(filepath.Join(plan.root, "pyvenv.cfg")) {
		t.Fatal("complete transaction changed user/config state", err)
	}
}

func TestCompleteVenvEmptyUsesMetadataAbsenceAndAllowsNextTransaction(t *testing.T) {
	for _, initiallyOwned := range []bool{false, true} {
		t.Run(map[bool]string{false: "already-empty", true: "retire-all"}[initiallyOwned], func(t *testing.T) {
			var plan pypiVenvPlan
			if initiallyOwned {
				plan = completeVenvOriginals(t)
			} else {
				_, plan = makePypiVenvFixture(t)
			}
			tx, closeTransaction := heldCompleteTestTransaction(t, plan)
			if err := tx.commitComplete(nil, true); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(filepath.Join(plan.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("empty complete transaction wrote an invalid empty ledger", err)
			}
			state, present, err := plan.readState()
			if err != nil || present || len(state.Files) != 0 {
				t.Fatal("empty complete state cannot be read normally", err)
			}
			if initiallyOwned {
				for _, path := range []string{"alpha.py", "beta.py", "alpha-1.0.dist-info/METADATA", "beta-1.0.dist-info/METADATA"} {
					if _, err := os.Lstat(filepath.Join(plan.site, path)); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("empty complete retained proven files", err)
					}
				}
				if body, err := os.ReadFile(filepath.Join(plan.site, "user-notes.txt")); err != nil || string(body) != "preserve user data" {
					t.Fatal("empty complete deleted user files", err)
				}
			} else if _, err := os.Lstat(filepath.Join(plan.root, ".heliopause")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("already-empty operation created unused controller directory", err)
			}
			closeTransaction()
			next, _ := heldCompleteTestTransaction(t, plan)
			if err := next.commit(venvDestinations(t, map[string]string{"site/new.py": "next ordinary install"}, "new")); err != nil {
				t.Fatal("normal transaction after explicit empty failed", err)
			}
		})
	}
}

func TestCompleteVenvRequiresMatchingEmptyIntentAndKeepsLegacyRefusal(t *testing.T) {
	for _, mode := range []string{"legacy-empty", "missing-empty-intent", "nonempty-with-empty-intent"} {
		t.Run(mode, func(t *testing.T) {
			plan := completeVenvOriginals(t)
			original, err := snapshotPyPIFile(filepath.Join(plan.site, "alpha.py"))
			if err != nil {
				t.Fatal(err)
			}
			tx, _ := heldCompleteTestTransaction(t, plan)
			switch mode {
			case "legacy-empty":
				err = tx.commit(nil)
			case "missing-empty-intent":
				err = tx.commitComplete(nil, false)
			case "nonempty-with-empty-intent":
				err = tx.commitComplete(venvDestinations(t, map[string]string{"site/alpha.py": "new"}, "alpha"), true)
			}
			if err == nil || !original.matches(filepath.Join(plan.site, "alpha.py")) || tx.backup != "" {
				t.Fatal("invalid empty intent mutated original", err)
			}
		})
	}
}

func TestCompleteVenvEmptyFaultRestoresOriginalFilesAndMetadata(t *testing.T) {
	for _, phase := range []string{"metadata", "sync", "reconcile"} {
		t.Run(phase, func(t *testing.T) {
			plan := completeVenvOriginals(t)
			paths := []string{filepath.Join(plan.site, "alpha.py"), filepath.Join(plan.site, "beta.py"), filepath.Join(plan.root, pypiVenvMetadata)}
			before := map[string]pypiFileSnapshot{}
			for _, path := range paths {
				value, err := snapshotPyPIFile(path)
				if err != nil {
					t.Fatal(err)
				}
				before[path] = value
			}
			tx, _ := heldCompleteTestTransaction(t, plan)
			failure := errors.New("injected complete empty failure")
			tx.fault = func(step string) error {
				if step == phase {
					return failure
				}
				return nil
			}
			if err := tx.commitComplete(nil, true); !errors.Is(err, failure) || tx.recovery {
				t.Fatal("complete empty failure was not rolled back", err)
			}
			for path, original := range before {
				if !original.matches(path) {
					t.Fatal("complete empty rollback lost original bytes/inode", path)
				}
			}
		})
	}
}

func TestCompleteVenvForeignMetadataIsNotOverwrittenDuringRollback(t *testing.T) {
	plan := completeVenvOriginals(t)
	tx, closeTransaction := heldCompleteTestTransaction(t, plan)
	meta := filepath.Join(plan.root, pypiVenvMetadata)
	failure := errors.New("failure after foreign metadata appeared")
	tx.fault = func(step string) error {
		if step == "sync" {
			if err := os.WriteFile(meta, []byte("foreign metadata must survive"), 0o600); err != nil {
				return err
			}
			return failure
		}
		return nil
	}
	if err := tx.commitComplete(nil, true); !errors.Is(err, failure) || !tx.recovery {
		t.Fatal("foreign metadata did not retain original recovery boundary", err)
	}
	if body, err := os.ReadFile(meta); err != nil || string(body) != "foreign metadata must survive" {
		t.Fatal("rollback overwrote foreign metadata", err)
	}
	for _, member := range []string{"metadata", "ledger.json", "file-0"} {
		if _, err := os.Stat(filepath.Join(tx.backup, member)); err != nil {
			t.Fatal("recovery did not retain original material", member, err)
		}
	}
	closeTransaction()
	if next, err := beginPyPIVenvTransaction(plan); err == nil {
		_ = next.close()
		t.Fatal("uncertain complete recovery permitted another mutation")
	}
}

func TestCompleteVenvReplacesEntireSharedOwnerSet(t *testing.T) {
	_, plan := makePypiVenvFixture(t)
	shared := venvDestinations(t, map[string]string{"site/shared.py": "old shared content"}, "alpha")[0]
	shared.ArtifactDigest = strings.Repeat("a", 64)
	shared.SharedOwners = []pypiOwner{{"alpha", "1.0", shared.ArtifactDigest}, {"beta", "1.0", strings.Repeat("b", 64)}}
	if err := plan.commit("", []pypiDestination{shared}); err != nil {
		t.Fatal(err)
	}
	tx, _ := heldCompleteTestTransaction(t, plan)
	desired := venvDestinations(t, map[string]string{"site/shared.py": "new fully approved alpha"}, "alpha")
	if _, _, err := tx.prepare(desired); err == nil {
		t.Fatal("legacy partial-shared refusal changed")
	}
	if err := tx.commitComplete(desired, false); err != nil {
		t.Fatal(err)
	}
	state, _, err := plan.readState()
	if err != nil || len(state.Files["site/shared.py"].owners()) != 1 || state.Files["site/shared.py"].Distribution != "alpha" {
		t.Fatal("complete transaction retained a retired shared owner", err)
	}
}
