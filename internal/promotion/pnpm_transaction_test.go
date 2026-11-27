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

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Mechanical transaction fixture: actual static/Store/cache producers with
// explicitly synthetic source/SRI. Fresh native/Policy workflow is separate.
func pnpmTransactionFixture(t *testing.T) (pnpmCacheFixture, *PNPMProjectPromotion, domain.StagedProjectSet, domain.ProjectDependencyUpdate) {
	t.Helper()
	f := newPNPMCacheFixture(t, false)
	target := f.cache.frozen.Snapshot().Context().Target().String()
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "package.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := domain.NewProjectControlFile("package.json", []byte(`{}`), true)
	if err != nil {
		t.Fatal(err)
	}
	absent, err := domain.NewProjectControlFile("pnpm-lock.yaml", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	update, err := domain.NewProjectDependencyUpdateFromSnapshot([]domain.ProjectControlFile{original, absent}, f.cache.frozen.Controls(), f.cache.frozen.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	owner, err := NewPNPMProjectPromotion(f.cache.intakeRoot, f.cache.evidenceRoot, f.cache.cacheRoot, filepath.Join(filepath.Dir(f.cache.cacheRoot), "state"), f.cache.evidence, f.cache.operation, func(domain.ProjectDependencySnapshot) (*PNPMVerifiedCache, error) { return f.cache, nil })
	if err != nil {
		t.Fatal(err)
	}
	staged, err := f.cache.StageProject(context.Background(), f.set)
	if err != nil {
		t.Fatal(err)
	}
	return f, owner, staged, update
}

func TestPNPMTransactionCommitsIndependentApprovalAndRetainsNoopPayload(t *testing.T) {
	f, owner, staged, update := pnpmTransactionFixture(t)
	ctx := context.Background()
	g, err := owner.Begin(ctx, f.cache.frozen.Snapshot().Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := owner.PublishedRecords(); ok {
		t.Fatal("records exposed before target publication")
	}
	if _, err := acquireNPMProjectGuard(g.install.Target().String()); err == nil {
		t.Fatal("other npm client acquired the same target")
	}
	if err := g.Commit(ctx, update, staged); err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	id, manifest, sbom, ok := owner.PublishedRecords()
	if !ok || id.String() == "" || manifest != staged.ContentHandle()+":manifest.json" || sbom != staged.ContentHandle()+":sbom.cdx.json" {
		t.Fatal("successful publication records missing")
	}
	target := f.cache.frozen.Snapshot().Context().Target().String()
	code := filepath.Join(target, "node_modules/.pnpm/is-number@6.0.0/node_modules/is-number/package.json")
	before, err := os.Lstat(code)
	if err != nil {
		t.Fatal(err)
	}
	controlBefore, err := os.Lstat(filepath.Join(target, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	g, err = owner.Begin(ctx, f.cache.frozen.Snapshot().Context())
	if err != nil {
		t.Fatal("reopen independently approved target", err)
	}
	if _, _, _, ok := owner.PublishedRecords(); ok {
		t.Fatal("new request reused previous publication records")
	}
	noop, err := domain.NewProjectDependencyUpdateFromSnapshot(g.Controls(), f.cache.frozen.Controls(), f.cache.frozen.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Commit(ctx, noop, staged); err != nil {
		t.Fatal("retained mechanical approval replay", err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := os.Lstat(code)
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("no-op rewrote original native files", err)
	}
	controlAfter, err := os.Lstat(filepath.Join(target, "package.json"))
	if err != nil || !os.SameFile(controlBefore, controlAfter) || controlAfter.Mode() != 0o600 {
		t.Fatal("no-op rewrote original control identity/mode", err)
	}
	state, err := os.ReadDir(owner.stateRoot)
	if err != nil || len(state) != 1 {
		t.Fatal("independent approval not durably retained", err)
	}
}

func TestPNPMTransactionRollsBackForwardPhaseFailures(t *testing.T) {
	for _, phase := range []string{"before-backup:package.json", "after-backup:package.json", "before-publish:package.json", "after-publish:package.json", "after-publish:pnpm-lock.yaml", "after-publish:node_modules", "after-publish:haa", "before-metadata", "after-metadata", "before-state", "after-state"} {
		t.Run(phase, func(t *testing.T) {
			f, owner, staged, update := pnpmTransactionFixture(t)
			ctx := context.Background()
			cause := errors.New("controlled forward phase failure")
			owner.checkpoint = func(actual string) error {
				if actual == phase {
					return cause
				}
				return nil
			}
			g, err := owner.Begin(ctx, f.cache.frozen.Snapshot().Context())
			if err != nil {
				t.Fatal(err)
			}
			if err := g.Commit(ctx, update, staged); !errors.Is(err, cause) {
				t.Fatal("phase failure lost original cause", err)
			}
			if _, _, _, ok := owner.PublishedRecords(); ok {
				t.Fatal("failed transaction exposed publication records")
			}
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
			root := f.cache.frozen.Snapshot().Context().Target().String()
			body, err := os.ReadFile(filepath.Join(root, "package.json"))
			if err != nil || string(body) != `{}` {
				t.Fatal("original control was not restored", err)
			}
			for _, name := range []string{"pnpm-lock.yaml", "node_modules", ".heliopause"} {
				if _, err := os.Lstat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("uncommitted member survived", name, err)
				}
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".heliopause-npm-commit-") {
					t.Fatal("known rollback journal was not removed")
				}
			}
			state, err := os.ReadDir(owner.stateRoot)
			if err != nil || len(state) != 0 {
				t.Fatal("uncommitted independent state survived", err)
			}
		})
	}
}

func TestPNPMManagedRollbackRestoresPayloadAndIndependentState(t *testing.T) {
	f, owner, staged, update := pnpmTransactionFixture(t)
	ctx := context.Background()
	g, err := owner.Begin(ctx, f.cache.frozen.Snapshot().Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Commit(ctx, update, staged); err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	g, err = owner.Begin(ctx, f.cache.frozen.Snapshot().Context())
	if err != nil {
		t.Fatal(err)
	}
	oldState := bytes.Clone(g.originalState)
	g.previous = nil // Exercise full-copy rollback, not the no-op optimization.
	root := g.install.Target().String()
	code := filepath.Join(root, "node_modules/.pnpm/is-number@6.0.0/node_modules/is-number/package.json")
	before, err := os.Lstat(code)
	if err != nil {
		t.Fatal(err)
	}
	update, err = domain.NewProjectDependencyUpdateFromSnapshot(g.Controls(), f.cache.frozen.Controls(), f.cache.frozen.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("controlled managed publication failure")
	owner.checkpoint = func(phase string) error {
		if phase == "after-state" {
			return cause
		}
		return nil
	}
	if err := g.Commit(ctx, update, staged); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := os.Lstat(code)
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("managed rollback replaced original payload inode", err)
	}
	state, err := os.ReadFile(filepath.Join(owner.stateRoot, pnpmCacheHash([]byte(root))+".json"))
	if err != nil || !bytes.Equal(state, oldState) {
		t.Fatal("independent original state was not restored", err)
	}
}

func TestPNPMGuardRejectsDriftBeforeSelectedCacheAccess(t *testing.T) {
	for _, mutation := range []string{"control", "absent-lock", "foreign-payload", "cancel"} {
		t.Run(mutation, func(t *testing.T) {
			f, owner, staged, update := pnpmTransactionFixture(t)
			calls := 0
			owner.selectedCache = func(domain.ProjectDependencySnapshot) (*PNPMVerifiedCache, error) { calls++; return f.cache, nil }
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			g, err := owner.Begin(ctx, f.cache.frozen.Snapshot().Context())
			if err != nil {
				t.Fatal(err)
			}
			if calls != 0 {
				t.Fatal("guard accessed a cache before selection")
			}
			root := g.install.Target().String()
			var foreign string
			switch mutation {
			case "control":
				foreign = filepath.Join(root, "package.json")
			case "absent-lock":
				foreign = filepath.Join(root, "pnpm-lock.yaml")
			case "foreign-payload":
				foreign = filepath.Join(root, "node_modules")
			case "cancel":
				cancel()
			}
			if foreign != "" {
				if err := os.WriteFile(foreign, []byte("foreign bytes"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := g.Commit(ctx, update, staged); err == nil {
				t.Fatal("changed/canceled project was committed")
			}
			if calls != 0 {
				t.Fatal("invalid original reached selected cache")
			}
			if foreign != "" {
				if body, err := os.ReadFile(foreign); err != nil || string(body) != "foreign bytes" {
					t.Fatal("foreign original was changed", err)
				}
			}
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPNPMCommitRejectsAnotherActualOperation(t *testing.T) {
	f, owner, staged, update := pnpmTransactionFixture(t)
	g, err := owner.Begin(context.Background(), f.cache.frozen.Snapshot().Context())
	if err != nil {
		t.Fatal(err)
	}
	other, err := domain.NewOperationID()
	if err != nil {
		t.Fatal(err)
	}
	copyCache := *f.cache
	copyCache.operation = other
	owner.selectedCache = func(domain.ProjectDependencySnapshot) (*PNPMVerifiedCache, error) { return &copyCache, nil }
	if err := g.Commit(context.Background(), update, staged); err == nil {
		t.Fatal("another operation's cache was committed")
	}
	if body, err := os.ReadFile(filepath.Join(g.install.Target().String(), "package.json")); err != nil || string(body) != `{}` {
		t.Fatal("rejected request mutated original", err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPNPMIndependentApprovalRejectsUntrustedRetainedMembers(t *testing.T) {
	for _, mutation := range []string{"payload", "foreign-metadata", "hardlink", "escape-link", "state", "cache", "evidence", "journal"} {
		t.Run(mutation, func(t *testing.T) {
			f, owner, staged, update := pnpmTransactionFixture(t)
			ctx := context.Background()
			install := f.cache.frozen.Snapshot().Context()
			g, err := owner.Begin(ctx, install)
			if err != nil {
				t.Fatal(err)
			}
			if err := g.Commit(ctx, update, staged); err != nil {
				t.Fatal(err)
			}
			if err := g.Close(); err != nil {
				t.Fatal(err)
			}
			root := install.Target().String()
			code := filepath.Join(root, "node_modules/.pnpm/is-number@6.0.0/node_modules/is-number/package.json")
			var foreign string
			switch mutation {
			case "payload":
				foreign = code
			case "foreign-metadata":
				foreign = filepath.Join(root, ".heliopause/foreign.json")
			case "hardlink":
				if err := os.Link(code, filepath.Join(filepath.Dir(root), "outside-alias")); err != nil {
					t.Fatal(err)
				}
			case "escape-link":
				link := filepath.Join(root, "node_modules/is-number")
				if err := os.Remove(link); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("../../outside", link); err != nil {
					t.Fatal(err)
				}
			case "state":
				foreign = filepath.Join(owner.stateRoot, pnpmCacheHash([]byte(root))+".json")
			case "cache":
				data, err := f.cache.OpenProjectCache(ctx, staged)
				if err != nil {
					t.Fatal(err)
				}
				foreign = filepath.Join(filepath.Dir(data), "receipt.json")
			case "evidence":
				entries, err := os.ReadDir(filepath.Join(f.cache.evidenceRoot, f.run))
				if err != nil {
					t.Fatal(err)
				}
				if len(entries) == 0 {
					t.Fatal("fixture has no stored Evidence")
				}
				foreign = filepath.Join(f.cache.evidenceRoot, f.run, entries[0].Name())
			case "journal":
				if err := os.Mkdir(filepath.Join(root, ".heliopause-npm-commit-interrupted"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if foreign != "" {
				if err := os.Chmod(foreign, 0o600); err != nil && !errors.Is(err, os.ErrNotExist) {
					t.Fatal(err)
				}
				if err := os.WriteFile(foreign, []byte("foreign bytes"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			for _, adopt := range []bool{false, true} {
				guard, err := owner.BeginUpgrade(ctx, install, adopt)
				if err == nil {
					_ = guard.Close()
					t.Fatal("untrusted retained state was accepted", mutation, adopt)
				}
			}
			if foreign != "" {
				if body, err := os.ReadFile(foreign); err != nil || string(body) != "foreign bytes" {
					t.Fatal("rejected retained data was deleted", err)
				}
			}
		})
	}
}

func TestPNPMExplicitEnrollmentOnlyAcceptsChangedControls(t *testing.T) {
	f, owner, staged, update := pnpmTransactionFixture(t)
	ctx := context.Background()
	install := f.cache.frozen.Snapshot().Context()
	g, err := owner.Begin(ctx, install)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Commit(ctx, update, staged); err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(install.Target().String(), "package.json")
	body, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	doc["description"] = json.RawMessage(`"explicit local edit"`)
	body, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if g, err := owner.Begin(ctx, install); err == nil {
		_ = g.Close()
		t.Fatal("silent enrollment accepted")
	}
	g, err = owner.BeginUpgrade(ctx, install, true)
	if err != nil {
		t.Fatal("explicit controls enrollment rejected", err)
	}
	if string(g.Controls()[0].Body()) != string(body) {
		t.Fatal("enrollment did not freeze current controls")
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPNPMRollbackConflictPreservesForeignOutputAndRecoveryJournal(t *testing.T) {
	f, owner, staged, update := pnpmTransactionFixture(t)
	ctx := context.Background()
	install := f.cache.frozen.Snapshot().Context()
	cause := errors.New("controlled conflict after native publication")
	foreign := filepath.Join(install.Target().String(), "node_modules/foreign")
	owner.checkpoint = func(phase string) error {
		if phase == "after-publish:node_modules" {
			if err := os.WriteFile(foreign, []byte("preserve me"), 0o600); err != nil {
				return err
			}
			return cause
		}
		return nil
	}
	g, err := owner.Begin(ctx, install)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Commit(ctx, update, staged); !errors.Is(err, cause) || !strings.Contains(err.Error(), "rollback is incomplete") {
		t.Fatal("conflict lost cause or claimed clean rollback", err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(foreign); err != nil || string(body) != "preserve me" {
		t.Fatal("foreign output deleted", err)
	}
	entries, err := os.ReadDir(install.Target().String())
	if err != nil {
		t.Fatal(err)
	}
	journals := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".heliopause-npm-commit-") {
			journals++
		}
	}
	if journals != 1 {
		t.Fatal("uncertain recovery journal missing", journals)
	}
	if g, err := owner.BeginUpgrade(ctx, install, true); err == nil {
		_ = g.Close()
		t.Fatal("new mutation entered an unresolved journal")
	}
}
