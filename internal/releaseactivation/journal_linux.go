//go:build linux

package releaseactivation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rahoney/heliopause/internal/releaseinstall"
	"golang.org/x/mod/semver"
)

const (
	journalSchema    = "helox.protected-update/v1"
	policyConfigPath = "/etc/heliopause/network-policy.json"
	hostConfigPath   = "/etc/heliopause/host-tools.json"
	policyUnitPath   = "/etc/systemd/system/haa-network-policy-helper.service"
)

type savedConfig struct {
	Body   []byte `json:"body"`
	SHA256 string `json:"sha256"`
	Mode   uint32 `json:"mode"`
	GID    uint32 `json:"gid"`
}

type updateJournal struct {
	Schema         string         `json:"schema"`
	State          string         `json:"state"`
	Old            registration   `json:"old"`
	Version        string         `json:"version"`
	ManifestSHA256 string         `json:"manifest_sha256"`
	Stage          string         `json:"stage"`
	Configs        [3]savedConfig `json:"configs"`
}

func configPaths() [3]string { return [3]string{policyConfigPath, hostConfigPath, policyUnitPath} }

func bodySHA256(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }

func parseJournal(body []byte) (updateJournal, error) {
	var journal updateJournal
	if len(body) == 0 || len(body) > 128<<10 || json.Unmarshal(body, &journal) != nil {
		return journal, ErrTransition
	}
	canonical, err := json.Marshal(journal)
	if err != nil || !bytes.Equal(canonical, body) {
		return updateJournal{}, ErrTransition
	}
	if err := journal.validate(); err != nil {
		return updateJournal{}, err
	}
	return journal, nil
}

func (j updateJournal) validate() error {
	if j.Schema != journalSchema || !validJournalState(j.State) || !exactVersion(j.Version) || !sha256Text(j.ManifestSHA256) || !strings.HasPrefix(j.Stage, ".prepared-") || len(j.Stage) > 128 || filepath.Base(j.Stage) != j.Stage || strings.ContainsAny(j.Stage, "\x00\r\n\\") {
		return ErrTransition
	}
	old, err := json.Marshal(j.Old)
	if err != nil {
		return ErrTransition
	}
	if _, err := parseRegistration(old); err != nil {
		return errors.Join(ErrTransition, err)
	}
	comparison := semver.Compare(j.Version, j.Old.Version)
	if comparison < 0 || (comparison == 0 && j.ManifestSHA256 != j.Old.ManifestSHA256) {
		return ErrTransition
	}
	for _, config := range j.Configs {
		if len(config.Body) == 0 || len(config.Body) > 16<<10 || bodySHA256(config.Body) != config.SHA256 || config.Mode&^0777 != 0 || config.Mode&0133 != 0 || config.Mode&0400 == 0 || config.GID > 1<<31-1 {
			return ErrTransition
		}
	}
	return nil
}

func validJournalState(state string) bool {
	switch state {
	case "PREPARED", "SWITCHING", "VERIFYING", "COMMITTED", "ROLLING_BACK", "ROLLED_BACK", "RECOVERY_REQUIRED":
		return true
	}
	return false
}

func allowedJournalTransition(from, to string) bool {
	if to == "RECOVERY_REQUIRED" {
		return from != "COMMITTED" && from != "ROLLED_BACK" && validJournalState(from)
	}
	switch from {
	case "PREPARED":
		return to == "SWITCHING" || to == "ROLLING_BACK"
	case "SWITCHING":
		return to == "VERIFYING" || to == "ROLLING_BACK"
	case "VERIFYING":
		return to == "COMMITTED" || to == "ROLLING_BACK"
	case "ROLLING_BACK":
		return to == "ROLLED_BACK"
	case "RECOVERY_REQUIRED":
		return to == "ROLLING_BACK"
	}
	return false
}

func snapshotConfig(ctx context.Context, path string) (savedConfig, error) {
	body, err := readProtected(ctx, path, 16<<10)
	if err != nil {
		return savedConfig{}, err
	}
	file, identity, err := openProtectedFile(path)
	if err != nil {
		return savedConfig{}, err
	}
	closeErr := file.Close()
	if closeErr != nil {
		return savedConfig{}, ErrUntrusted
	}
	owner, ok := identity.Sys().(*syscall.Stat_t)
	if !ok || owner.Gid > 1<<31-1 {
		return savedConfig{}, ErrUntrusted
	}
	// Re-read through the same protected contract to bind bytes and metadata
	// when an unrelated administrator replaced a protected config meanwhile.
	actual, err := readProtected(ctx, path, 16<<10)
	if err != nil || !bytes.Equal(actual, body) {
		return savedConfig{}, ErrUntrusted
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(identity, current) {
		return savedConfig{}, ErrUntrusted
	}
	return savedConfig{body, bodySHA256(body), uint32(identity.Mode().Perm()), owner.Gid}, nil
}

func newPreparedJournal(ctx context.Context, current InstalledRelease, stage *preparedRelease) (updateJournal, error) {
	if ctx == nil || os.Geteuid() != 0 || stage == nil || stage.closed || current.record.Schema != registrationSchema {
		return updateJournal{}, ErrUntrusted
	}
	journal := updateJournal{Schema: journalSchema, State: "PREPARED", Old: current.record, Version: stage.release.manifest.Version(), ManifestSHA256: stage.release.manifest.SHA256(), Stage: filepath.Base(stage.directory)}
	if stage.directory != filepath.Join(releaseinstall.SystemStagingRoot, journal.Stage) {
		return updateJournal{}, ErrUntrusted
	}
	identity, err := os.Lstat(stage.directory)
	if err != nil || stage.identity == nil || !os.SameFile(identity, stage.identity) || !identity.IsDir() || identity.Mode().Perm() != 0700 || len(stage.proof) == 0 {
		return updateJournal{}, ErrUntrusted
	}
	if err := protectedDirectory(stage.directory); err != nil {
		return updateJournal{}, err
	}
	for index, path := range configPaths() {
		config, err := snapshotConfig(ctx, path)
		if err != nil {
			return updateJournal{}, err
		}
		journal.Configs[index] = config
	}
	if err := journal.validate(); err != nil {
		return updateJournal{}, err
	}
	return journal, nil
}

type durableJournal struct {
	value updateJournal
	body  []byte
}

func createPreparedJournal(ctx context.Context, current InstalledRelease, stage *preparedRelease) (*durableJournal, error) {
	value, err := newPreparedJournal(ctx, current, stage)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, ErrTransition
	}
	if err := replaceProtected(ctx, journalPath, nil, body, 0600, 0); err != nil {
		return nil, err
	}
	return &durableJournal{value, body}, nil
}

func readDurableJournal(ctx context.Context) (*durableJournal, error) {
	body, err := readProtected(ctx, journalPath, 128<<10)
	if err != nil {
		return nil, err
	}
	value, err := parseJournal(body)
	if err != nil {
		return nil, err
	}
	return &durableJournal{value, body}, nil
}

func (j *durableJournal) transition(ctx context.Context, state string) error {
	if j == nil || !allowedJournalTransition(j.value.State, state) {
		return ErrTransition
	}
	next := j.value
	next.State = state
	if err := next.validate(); err != nil {
		return err
	}
	body, err := json.Marshal(next)
	if err != nil {
		return ErrTransition
	}
	if err := replaceProtected(ctx, journalPath, j.body, body, 0600, 0); err != nil {
		return err
	}
	j.value, j.body = next, body
	return nil
}

// replaceProtected is private and used only with fixed installation/config
// paths. A nil expected value means confirmed absence, never blind overwrite.
func replaceProtected(ctx context.Context, path string, expected, body []byte, mode os.FileMode, gid uint32) (resultErr error) {
	if ctx == nil || os.Geteuid() != 0 || len(body) == 0 || len(body) > 128<<10 || mode&^0777 != 0 || mode&0133 != 0 || mode&0400 == 0 || gid > 1<<31-1 {
		return ErrUntrusted
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := protectedDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	var original os.FileInfo
	if expected == nil {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return ErrUntrusted
		}
	} else {
		current, err := readProtected(ctx, path, 128<<10)
		if err != nil || !bytes.Equal(current, expected) {
			return ErrUntrusted
		}
		original, err = os.Lstat(path)
		if err != nil {
			return ErrUntrusted
		}
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".haa-update-")
	if err != nil {
		return ErrUntrusted
	}
	temporary := file.Name()
	identity, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return errors.Join(ErrUntrusted, ErrCleanup)
	}
	defer func() {
		if file != nil {
			if err := file.Close(); err != nil {
				resultErr = errors.Join(resultErr, ErrUntrusted)
			}
		}
		if temporary != "" {
			current, err := os.Lstat(temporary)
			if err != nil || !os.SameFile(identity, current) {
				resultErr = errors.Join(resultErr, ErrCleanup)
			} else if err := os.Remove(temporary); err != nil {
				resultErr = errors.Join(resultErr, ErrCleanup)
			}
		}
	}()
	if err := file.Chown(0, int(gid)); err != nil {
		return ErrUntrusted
	}
	if err := file.Chmod(mode); err != nil {
		return ErrUntrusted
	}
	if count, err := file.Write(body); err != nil || count != len(body) {
		return errors.Join(ErrUntrusted, err, io.ErrShortWrite)
	}
	if err := file.Sync(); err != nil {
		return errors.Join(ErrUntrusted, err)
	}
	closeErr := file.Close()
	file = nil
	if closeErr != nil {
		return errors.Join(ErrUntrusted, closeErr)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if original != nil {
		current, err := os.Lstat(path)
		if err != nil || !os.SameFile(original, current) {
			return ErrUntrusted
		}
	} else if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return ErrUntrusted
	}
	if err := os.Rename(temporary, path); err != nil {
		return errors.Join(ErrUntrusted, err)
	}
	temporary = ""
	return syncProtectedDirectory(filepath.Dir(path))
}
