//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const historyPath = stateRoot + "/last-protected-update.json"

type activationServices interface {
	active(context.Context, string, uint32) (*serviceProcess, error)
	drained(context.Context, *controllerPeer, *serviceProcess) error
	resourcesDrained(context.Context, uint32, int, int) error
	quiesce(context.Context, updateJournal, InstalledRelease, *InstalledRelease) error
	start(context.Context, string, uint32) (*serviceProcess, error)
	runtimeHealthy(context.Context) error
	Close() error
}

type activationChecks struct {
	installed func(context.Context) (InstalledRelease, error)
	recorded  func(context.Context, registration) (InstalledRelease, error)
	health    func(context.Context, InstalledRelease, InstalledRelease, *controllerPeer, string, bool) (*healthWorkspace, error)
}

type rootActivation struct {
	old              InstalledRelease
	stage            *preparedRelease
	peer             *controllerPeer
	journal          *durableJournal
	services         activationServices
	checks           activationChecks
	install, restore *releaseinstall.Installer
	workspace        *healthWorkspace
	retired          bool
}

// prepareRootActivation is reached after the server's installer-owner lock,
// old kernel peer and independently verified payload. Production dependencies
// are fixed here; no request field can select a verifier, command or target.
func prepareRootActivation(ctx context.Context, stage *preparedRelease, peer *controllerPeer) (_ *rootActivation, resultErr error) {
	if ctx == nil || stage == nil || peer == nil {
		return nil, ErrUntrusted
	}
	managed, err := managedStatePresent()
	if err != nil {
		return nil, err
	}
	if !managed {
		return nil, ErrUnmanaged
	}
	old, err := ReadInstalledRelease(ctx)
	if err != nil {
		return nil, err
	}
	if err := verifyInstallerController(old); err != nil {
		return nil, err
	}
	if peer.uid != old.ClientUID() || peer.gid != old.ClientGID() {
		return nil, ErrCaller
	}
	if err := peer.alive(); err != nil {
		return nil, err
	}
	value, err := newPreparedJournal(ctx, old, stage)
	if err != nil {
		return nil, err
	}
	if _, _, err := parseNativeConfigs(value); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(old.record)
	if err != nil {
		return nil, ErrUntrusted
	}
	actual, err := readProtected(ctx, registrationPath, 16<<10)
	if err != nil || !bytes.Equal(canonical, actual) {
		return nil, errors.Join(ErrUntrusted, err)
	}
	services, err := newServiceController(ctx)
	if err != nil {
		return nil, err
	}
	transaction := &rootActivation{old: old, stage: stage, peer: peer, services: services, checks: activationChecks{ReadInstalledRelease, readRecordedRelease, runCompatibilityHealth}}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, transaction.Close())
		}
	}()
	process, err := services.active(ctx, old.Version(), old.ClientGID())
	if err != nil {
		return transaction, err
	}
	if err := process.Close(); err != nil {
		return transaction, err
	}
	transaction.install, err = releaseinstall.NewSystemInstaller(protectedBundleVerifier{old, stage.proof})
	if err != nil {
		return transaction, err
	}
	transaction.restore, err = releaseinstall.NewSystemInstaller(protectedBundleVerifier{old, old.proof})
	if err != nil {
		return transaction, err
	}
	if err := storeReleaseProof(ctx, stage.release, old); err != nil {
		return transaction, err
	}
	transaction.journal, err = persistPreparedJournal(ctx, value)
	if err != nil {
		return transaction, err
	}
	return transaction, nil
}

func (a *rootActivation) Close() error {
	if a == nil {
		return nil
	}
	var result error
	if a.services != nil {
		result = errors.Join(result, a.services.Close())
	}
	if a.stage != nil && (a.retired || journalConfirmedAbsent()) {
		result = errors.Join(result, a.stage.Close())
	}
	return result
}

func journalConfirmedAbsent() bool {
	_, err := os.Lstat(journalPath)
	return errors.Is(err, os.ErrNotExist)
}

func (a *rootActivation) controllerPID() int {
	if a.peer != nil && a.peer.alive() == nil {
		return int(a.peer.pid)
	}
	return 0
}

func (a *rootActivation) activate(ctx context.Context) (resultErr error) {
	if ctx == nil || a == nil || a.journal == nil || a.journal.value.State != "PREPARED" || a.peer == nil {
		return ErrTransition
	}
	wait, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	exclusive, err := acquireExclusiveRuntime(wait)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, exclusive.Close()) }()
	if err := a.peer.alive(); err != nil {
		return err
	}
	current, err := a.checks.installed(ctx)
	if err != nil || current.record != a.old.record {
		return errors.Join(ErrUntrusted, err)
	}
	if err := verifyRuntimeConfigs(ctx, a.journal.value, true); err != nil {
		return err
	}
	process, err := a.services.active(ctx, a.old.Version(), a.old.ClientGID())
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, process.Close()) }()
	if err := a.services.drained(ctx, a.peer, process); err != nil {
		return err
	}
	if err := a.journal.transition(ctx, "SWITCHING"); err != nil {
		return err
	}
	// Once switching is durable every error is handled under this same EX.
	defer func() {
		if resultErr == nil {
			return
		}
		recovery, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := a.refreshJournal(recovery); err != nil {
			resultErr = errors.Join(resultErr, ErrTransition, err)
			return
		}
		if a.journal.value.State == "COMMITTED" {
			resultErr = errors.Join(resultErr, ErrTransition)
			return
		}
		if err := a.rollback(recovery); err != nil {
			resultErr = errors.Join(resultErr, err, a.requireRecovery(recovery))
		}
	}()
	if err := a.services.quiesce(ctx, a.journal.value, a.old, nil); err != nil {
		return err
	}
	if err := a.services.resourcesDrained(ctx, a.old.ClientUID(), a.controllerPID(), 0); err != nil {
		return err
	}
	if err := switchRuntimeConfigs(ctx, a.journal.value, false); err != nil {
		return err
	}
	existing := nativeVersionPath(a.journal.value.Version, "helox-linux-amd64")
	if _, err := os.Lstat(filepath.Dir(existing)); err == nil {
		// Reuse the verified existing-version primitive for same-version or a
		// prior complete candidate. Fresh quarantine and proof checks still run.
		if _, err := a.install.ActivateExisting(ctx, releaseinstall.Request{BundleDirectory: filepath.Dir(existing), InstallRoot: releaseinstall.SystemInstallRoot}); err != nil {
			return err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if _, err := a.install.Install(ctx, releaseinstall.Request{BundleDirectory: a.stage.directory, InstallRoot: releaseinstall.SystemInstallRoot}); err != nil {
			return err
		}
	} else {
		return errors.Join(ErrUntrusted, err)
	}
	if err := switchRegistration(ctx, a.journal.value, false); err != nil {
		return err
	}
	if err := a.journal.transition(ctx, "VERIFYING"); err != nil {
		return err
	}
	next, err := a.checks.installed(ctx)
	if err != nil || next.record != candidateRegistration(a.journal.value) {
		return errors.Join(ErrUntrusted, err)
	}
	newProcess, err := a.services.start(ctx, next.Version(), next.ClientGID())
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, newProcess.Close()) }()
	a.workspace, err = a.checks.health(ctx, a.old, next, a.peer, a.journal.value.Stage, true)
	if err != nil {
		return err
	}
	confirmed, err := a.checks.installed(ctx)
	if err != nil || confirmed.record != next.record {
		return errors.Join(ErrUntrusted, err)
	}
	if err := verifyRuntimeConfigs(ctx, a.journal.value, false); err != nil {
		return err
	}
	if err := a.services.drained(ctx, a.peer, newProcess); err != nil {
		return err
	}
	if err := a.cleanupOwnedStage(); err != nil {
		return err
	}
	if err := a.journal.transition(ctx, "COMMITTED"); err != nil {
		return err
	}
	return a.retire(ctx)
}

func (a *rootActivation) cleanupOwnedStage() error {
	if a.workspace != nil {
		if err := a.workspace.Close(); err != nil {
			return err
		}
	}
	if a.stage != nil {
		if err := a.stage.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (a *rootActivation) refreshJournal(ctx context.Context) error {
	actual, err := readDurableJournal(ctx)
	if err != nil {
		return err
	}
	expected, found := a.journal.value, actual.value
	expected.State, found.State = "", ""
	one, err := json.Marshal(expected)
	if err != nil {
		return ErrTransition
	}
	two, err := json.Marshal(found)
	if err != nil || !bytes.Equal(one, two) {
		return ErrTransition
	}
	a.journal = actual
	return nil
}

func (a *rootActivation) requireRecovery(ctx context.Context) error {
	if a == nil || a.journal == nil {
		return ErrTransition
	}
	if err := a.refreshJournal(ctx); err != nil {
		return errors.Join(ErrTransition, err)
	}
	if a.journal.value.State == "COMMITTED" || a.journal.value.State == "ROLLED_BACK" {
		return ErrTransition
	}
	return errors.Join(ErrTransition, a.journal.transition(ctx, "RECOVERY_REQUIRED"))
}

// abortPrepared changes no runtime state and is safe even when another old
// command still holds SH. It retires only a demonstrably unchanged old setup.
func (a *rootActivation) abortPrepared(ctx context.Context) error {
	if a == nil || a.journal == nil {
		return ErrTransition
	}
	if err := a.refreshJournal(ctx); err != nil {
		return err
	}
	if a.journal.value.State != "PREPARED" {
		return ErrTransition
	}
	current, err := a.checks.installed(ctx)
	if err != nil || current.record != a.old.record {
		return errors.Join(ErrUntrusted, err)
	}
	if err := verifyRuntimeConfigs(ctx, a.journal.value, true); err != nil {
		return err
	}
	process, err := a.services.active(ctx, a.old.Version(), a.old.ClientGID())
	if err != nil {
		return err
	}
	if err := process.Close(); err != nil {
		return err
	}
	if err := a.services.runtimeHealthy(ctx); err != nil {
		return err
	}
	if err := a.cleanupOwnedStage(); err != nil {
		return err
	}
	if err := a.journal.transition(ctx, "ROLLING_BACK"); err != nil {
		return err
	}
	if err := a.journal.transition(ctx, "ROLLED_BACK"); err != nil {
		return err
	}
	return a.retire(ctx)
}

func (a *rootActivation) rollback(ctx context.Context) (resultErr error) {
	if err := a.refreshJournal(ctx); err != nil {
		return err
	}
	if a.journal.value.State == "COMMITTED" || a.journal.value.State == "ROLLED_BACK" {
		return ErrTransition
	}
	old, err := a.checks.recorded(ctx, a.journal.value.Old)
	if err != nil {
		return err
	}
	if old.record != a.old.record {
		return ErrUntrusted
	}
	var candidate *InstalledRelease
	if next, err := a.checks.recorded(ctx, candidateRegistration(a.journal.value)); err == nil {
		candidate = &next
	}
	if err := restorableRuntimeConfigs(ctx, a.journal.value); err != nil {
		return err
	}
	if err := a.services.quiesce(ctx, a.journal.value, old, candidate); err != nil {
		return err
	}
	if err := a.services.resourcesDrained(ctx, old.ClientUID(), a.controllerPID(), 0); err != nil {
		return err
	}
	if a.workspace != nil {
		if err := a.workspace.Close(); err != nil {
			return err
		}
		a.workspace = nil
	}
	if a.journal.value.State != "ROLLING_BACK" {
		if err := a.journal.transition(ctx, "ROLLING_BACK"); err != nil {
			return err
		}
	}
	if err := switchRuntimeConfigs(ctx, a.journal.value, true); err != nil {
		return err
	}
	if _, err := a.restore.ActivateExisting(ctx, releaseinstall.Request{BundleDirectory: filepath.Dir(old.ClientPath()), InstallRoot: releaseinstall.SystemInstallRoot}); err != nil {
		return err
	}
	if err := switchRegistration(ctx, a.journal.value, true); err != nil {
		return err
	}
	process, err := a.services.start(ctx, old.Version(), old.ClientGID())
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, process.Close()) }()
	current, err := a.checks.installed(ctx)
	if err != nil || current.record != old.record {
		return errors.Join(ErrUntrusted, err)
	}
	if err := verifyRuntimeConfigs(ctx, a.journal.value, true); err != nil {
		return err
	}
	if err := a.services.runtimeHealthy(ctx); err != nil {
		return err
	}
	if a.peer != nil && a.peer.alive() == nil && verifyInstallerController(old) == nil {
		a.workspace, err = a.checks.health(ctx, old, old, a.peer, a.journal.value.Stage, false)
		if err != nil {
			return err
		}
	}
	if err := a.services.resourcesDrained(ctx, old.ClientUID(), a.controllerPID(), process.pid); err != nil {
		return err
	}
	if err := a.cleanupOwnedStage(); err != nil {
		return err
	}
	if err := a.journal.transition(ctx, "ROLLED_BACK"); err != nil {
		return err
	}
	return a.retire(ctx)
}

func (a *rootActivation) retire(ctx context.Context) error {
	if a.journal.value.State != "COMMITTED" && a.journal.value.State != "ROLLED_BACK" {
		return ErrTransition
	}
	expectedState := a.journal.value.State
	if err := a.refreshJournal(ctx); err != nil {
		return err
	}
	if a.journal.value.State != expectedState {
		return ErrTransition
	}
	var expected []byte
	if _, err := os.Lstat(historyPath); err == nil {
		body, err := readProtected(ctx, historyPath, 128<<10)
		if err != nil {
			return err
		}
		previous, err := parseJournal(body)
		if err != nil || (previous.State != "COMMITTED" && previous.State != "ROLLED_BACK") {
			return ErrTransition
		}
		expected = body
	} else if !errors.Is(err, os.ErrNotExist) {
		return ErrUntrusted
	}
	if err := replaceProtected(ctx, historyPath, expected, a.journal.body, 0600, 0); err != nil {
		return err
	}
	file, identity, err := openProtectedFile(journalPath)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return ErrUntrusted
	}
	current, err := os.Lstat(journalPath)
	if err != nil || !os.SameFile(identity, current) {
		return ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(journalPath); err != nil {
		return errors.Join(ErrTransition, err)
	}
	if err := syncProtectedDirectory(stateRoot); err != nil {
		return err
	}
	a.retired = true
	return nil
}

func restorableRuntimeConfigs(ctx context.Context, journal updateJournal) error {
	derived, err := derivedRuntimeConfigs(journal)
	if err != nil {
		return err
	}
	for index, path := range configPaths() {
		actual, err := snapshotConfig(ctx, path)
		if err != nil {
			return err
		}
		if actual.Mode != journal.Configs[index].Mode || actual.GID != journal.Configs[index].GID || !bytes.Equal(actual.Body, journal.Configs[index].Body) && !bytes.Equal(actual.Body, derived[index]) {
			return ErrUntrusted
		}
	}
	return nil
}

func (c *serviceController) quiesce(ctx context.Context, journal updateJournal, old InstalledRelease, candidate *InstalledRelease) (resultErr error) {
	actual, err := readProtected(ctx, policyUnitPath, 16<<10)
	if err != nil {
		return err
	}
	oldUnit, err := nativePolicyUnit(old.Version())
	if err != nil {
		return err
	}
	newUnit, err := nativePolicyUnit(journal.Version)
	if err != nil {
		return err
	}
	diskVersion := old.Version()
	if !bytes.Equal(actual, oldUnit) {
		if !bytes.Equal(actual, newUnit) {
			return ErrUntrusted
		}
		diskVersion = journal.Version
	}
	if err := c.command(ctx, "daemon-reload"); err != nil {
		return err
	}
	state, err := c.query(ctx)
	if err != nil {
		return err
	}
	if state.pid == 0 {
		return c.stopped(ctx, nil)
	}
	if state.properties["ActiveState"] != "active" || state.properties["SubState"] != "running" || state.properties["ControlGroup"] != policyControlGroup {
		return ErrTransition
	}
	process, err := pinServiceProcess(state.pid, old.Version(), old.ClientGID())
	if err != nil && candidate != nil {
		process, err = pinServiceProcess(state.pid, candidate.Version(), candidate.ClientGID())
	}
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, process.Close()) }()
	return c.stopKnown(ctx, process, diskVersion)
}

func (c *serviceController) runtimeHealthy(ctx context.Context) error {
	for _, image := range []string{runtimeidentity.NodeImageReference, runtimeidentity.PythonImageReference} {
		body, err := c.output(ctx, 4096, "docker", "image", "inspect", image, "--format", "{{.Id}}")
		if err != nil {
			return err
		}
		identity := strings.TrimSpace(string(body))
		if !strings.HasPrefix(identity, "sha256:") || !sha256Text(strings.TrimPrefix(identity, "sha256:")) {
			return ErrHealth
		}
	}
	return nil
}
