//go:build linux && realcorpus

package promotion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	evidencelocal "github.com/rahoney/heliopause/internal/evidence/local"
)

// This qualifies real signal/process boundaries against structural source/SRI
// fixtures. Public native/source/Policy qualification is a separate consumer.
func TestBunTransactionSignalAndCrashQualification(t *testing.T) {
	if os.Getenv("HELOX_BUN_FAILURE_QUALIFICATION") != "1" || os.Geteuid() == 0 {
		t.Fatal("declare ordinary-user Bun failure qualification")
	}
	if mode := os.Getenv("HELOX_BUN_FAILURE_CHILD"); mode != "" {
		runBunFailureChild(t, mode)
		return
	}
	for _, mode := range []string{"sigint", "crash"} {
		t.Run(mode, func(t *testing.T) {
			work := t.TempDir()
			read, write, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer read.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			command := exec.CommandContext(ctx, executable, "-test.run=^TestBunTransactionSignalAndCrashQualification$", "-test.v", "-test.timeout=12s")
			command.Env = append(os.Environ(), "HELOX_BUN_FAILURE_CHILD="+mode, "TMPDIR="+work)
			command.ExtraFiles = []*os.File{write}
			var logs bytes.Buffer
			command.Stdout = &logs
			command.Stderr = &logs
			if err := command.Start(); err != nil {
				_ = write.Close()
				t.Fatal(err)
			}
			_ = write.Close()
			done := false
			defer func() {
				if !done {
					_ = command.Process.Kill()
					_ = command.Wait()
				}
			}()
			decoder := json.NewDecoder(read)
			var ready pnpmFailureReady
			if err := decoder.Decode(&ready); err != nil {
				waitErr := command.Wait()
				done = true
				t.Fatal("child did not reach exact partial publication", err, waitErr, logs.String())
			}
			if !strings.HasPrefix(ready.Target, work+string(os.PathSeparator)) {
				t.Fatal("child target escaped owned temporary root")
			}
			signalToSend := os.Signal(os.Interrupt)
			if mode == "crash" {
				signalToSend = syscall.SIGKILL
			}
			if err := command.Process.Signal(signalToSend); err != nil {
				t.Fatal(err)
			}
			waitErr := command.Wait()
			done = true
			if mode == "sigint" {
				if waitErr != nil {
					t.Fatal("SIGINT rollback failed", waitErr, logs.String())
				}
				var verdict struct{ Canceled, OriginalRestored, Unpublished bool }
				if err := decoder.Decode(&verdict); err != nil || !verdict.Canceled || !verdict.OriginalRestored || !verdict.Unpublished {
					t.Fatal("missing actual SIGINT restoration proof", err, verdict)
				}
				t.Log("actual_sigint_after_lock_publish: canceled=true original_restored=true independent_state_absent=true publication=false")
				return
			}
			var exit *exec.ExitError
			if !errors.As(waitErr, &exit) || exit.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
				t.Fatal("child was not actually killed", waitErr)
			}
			journals, err := filepath.Glob(filepath.Join(ready.Target, ".heliopause-npm-commit-*"))
			if err != nil || len(journals) != 1 {
				t.Fatal("crash backup journal missing", err, journals)
			}
			original, err := os.ReadFile(filepath.Join(journals[0], "package.json"))
			if err != nil || string(original) != "{}" {
				t.Fatal("crash lost original bytes", err)
			}
			lock, err := os.Lstat(filepath.Join(ready.Target, ".heliopause-npm-transaction.lock"))
			if err != nil || !lock.Mode().IsRegular() {
				t.Fatal("crash ownership lock missing", err)
			}
			store, err := evidencelocal.NewStore(ready.Evidence)
			if err != nil {
				t.Fatal(err)
			}
			operation, err := domain.ParseOperationID(ready.Operation)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			owner, err := NewBunProjectPromotion(ready.Intake, ready.Evidence, ready.Cache, ready.State, store, operation, func(domain.ProjectDependencySnapshot) (*BunVerifiedCache, error) {
				calls++
				return nil, errors.New("must not access selected cache")
			})
			if err != nil {
				t.Fatal(err)
			}
			target, err := domain.NewInstallTarget(ready.Target)
			if err != nil {
				t.Fatal(err)
			}
			install, err := domain.NewNPMProjectInstallContext(target)
			if err != nil {
				t.Fatal(err)
			}
			if guard, err := owner.Begin(context.Background(), install); err == nil || guard != nil || calls != 0 {
				t.Fatal("crashed project reentered mutation", err)
			}
			after, err := os.ReadFile(filepath.Join(journals[0], "package.json"))
			if err != nil || !bytes.Equal(after, original) {
				t.Fatal("reentry damaged recovery original", err)
			}
			current, err := os.Lstat(filepath.Join(ready.Target, ".heliopause-npm-transaction.lock"))
			if err != nil || !os.SameFile(lock, current) {
				t.Fatal("reentry removed crash lock", err)
			}
			t.Log("actual_sigkill_after_lock_publish: crash_lock_retained=true original_backup_retained=true reentry_refused=true cache_reads=0")
		})
	}
}

func runBunFailureChild(t *testing.T, mode string) {
	t.Helper()
	if mode != "sigint" && mode != "crash" {
		t.Fatal("unknown failure child mode")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	f, owner, staged, update := bunTransactionFixture(t)
	target := f.cache.frozen.Snapshot().Context().Target().String()
	before, err := os.Lstat(filepath.Join(target, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	notify := os.NewFile(3, "parent-proof")
	if notify == nil {
		t.Fatal("parent proof pipe missing")
	}
	defer notify.Close()
	reached := false
	owner.engine.checkpoint = func(phase string) error {
		if phase != "after-publish:bun.lock" {
			return nil
		}
		if reached {
			return errors.New("checkpoint repeated")
		}
		reached = true
		if err := json.NewEncoder(notify).Encode(pnpmFailureReady{target, owner.engine.intakeRoot, owner.engine.evidenceRoot, owner.engine.cacheRoot, owner.engine.stateRoot, owner.engine.operation.String()}); err != nil {
			return err
		}
		<-ctx.Done()
		return nil
	}
	guard, err := owner.Begin(ctx, f.cache.frozen.Snapshot().Context())
	if err != nil {
		t.Fatal(err)
	}
	err = guard.Commit(ctx, update, staged)
	if mode == "crash" {
		t.Fatal("crash child returned without SIGKILL", err)
	}
	if !reached || !errors.Is(err, context.Canceled) {
		t.Fatal("mid-publication SIGINT lost original cancellation", err)
	}
	if err := guard.Close(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(target, "package.json"))
	if err != nil || string(body) != "{}" {
		t.Fatal("SIGINT lost original bytes", err)
	}
	after, err := os.Lstat(filepath.Join(target, "package.json"))
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
		t.Fatal("SIGINT replaced original inode/mode", err)
	}
	for _, name := range []string{"bun.lock", "node_modules", ".heliopause", ".heliopause-npm-transaction.lock"} {
		if _, err := os.Lstat(filepath.Join(target, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("uncommitted member survived SIGINT", name, err)
		}
	}
	entries, err := os.ReadDir(owner.engine.stateRoot)
	if err != nil || len(entries) != 0 {
		t.Fatal("unpublished independent state survived SIGINT", err)
	}
	journals, err := filepath.Glob(filepath.Join(target, ".heliopause-npm-commit-*"))
	if err != nil || len(journals) != 0 {
		t.Fatal("known SIGINT rollback journal survived", err)
	}
	_, _, _, published := owner.PublishedRecords()
	if published {
		t.Fatal("SIGINT published records")
	}
	if err := json.NewEncoder(notify).Encode(struct{ Canceled, OriginalRestored, Unpublished bool }{true, true, true}); err != nil {
		t.Fatal(err)
	}
}
