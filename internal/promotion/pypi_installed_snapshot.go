package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// pypiInstalledSnapshot captures original metadata without inventing a requested
// package. It grants no ownership or selection/publication authority.
type pypiInstalledSnapshot struct {
	directories   []string
	metadata      map[string]pypiFileSnapshot
	emptyMetadata map[string]map[string]os.FileInfo
	files         []artifactpypi.InstalledMetadataFile
}

func capturePyPIInstalledSnapshot(transaction *pypiVenvTransaction) (pypiInstalledSnapshot, error) {
	if transaction == nil {
		return pypiInstalledSnapshot{}, errors.New("installed metadata requires a held transaction")
	}
	plan := transaction.plan
	var err error
	s := pypiInstalledSnapshot{metadata: map[string]pypiFileSnapshot{}, emptyMetadata: map[string]map[string]os.FileInfo{}}
	s.directories, err = pypiUpgradeMetadataDirectories(plan.site)
	if err != nil {
		return pypiInstalledSnapshot{}, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", err)
	}
	s.files = []artifactpypi.InstalledMetadataFile{}
	total := 0
	retiredBudget := pypiUpgradeDirectoryLimit
	for _, name := range s.directories {
		directory := filepath.Join(plan.site, name)
		if err := transaction.freezeParents(directory); err != nil {
			return pypiInstalledSnapshot{}, err
		}
		tree, empty, err := snapshotPyPIRetiredMetadata(directory, retiredBudget)
		if err != nil {
			return pypiInstalledSnapshot{}, err
		}
		if empty {
			// The existing transaction removes obsolete owned files while
			// retaining directories. A directory-only tree is no distribution;
			// freeze every original identity and membership, never ownership.
			for relative, before := range tree {
				path := filepath.Join(directory, relative)
				if err := transaction.freezeParents(path); err != nil {
					return pypiInstalledSnapshot{}, err
				}
				if !sameRetiredMetadataDirectory(before, transaction.directories[path]) {
					return pypiInstalledSnapshot{}, errors.New("retired metadata directory changed while freezing")
				}
			}
			retiredBudget -= len(tree)
			s.emptyMetadata[directory] = tree
			continue
		}
		if err := rejectPyPIUpgradeDirectSource(directory); err != nil {
			return pypiInstalledSnapshot{}, err
		}
		path := filepath.Join(directory, "METADATA")
		snapshot, err := snapshotBoundedPromotionFile(path, 1<<20)
		if err != nil {
			return pypiInstalledSnapshot{}, err
		}
		body, err := readBoundedPromotionFile(path, 1<<20)
		if err != nil {
			return pypiInstalledSnapshot{}, err
		}
		total += len(body)
		if total > 4<<20 {
			return pypiInstalledSnapshot{}, errors.New("PyPI installed metadata exceeds upgrade input bound")
		}
		sum := sha256.Sum256(body)
		if snapshot.Digest != hex.EncodeToString(sum[:]) {
			return pypiInstalledSnapshot{}, errors.New("PyPI installed metadata changed while reading")
		}
		s.metadata[path] = snapshot
		s.files = append(s.files, artifactpypi.InstalledMetadataFile{Directory: name, Body: body})
	}
	return s, nil
}

func (s pypiInstalledSnapshot) metadataFiles() []artifactpypi.InstalledMetadataFile {
	files := make([]artifactpypi.InstalledMetadataFile, len(s.files))
	for i, file := range s.files {
		files[i] = artifactpypi.InstalledMetadataFile{Directory: file.Directory, Body: append([]byte(nil), file.Body...)}
	}
	return files
}

func (s pypiInstalledSnapshot) verify(ctx context.Context, transaction *pypiVenvTransaction) error {
	if ctx == nil || transaction == nil || s.metadata == nil || s.emptyMetadata == nil || s.files == nil {
		return errors.New("installed metadata snapshot is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := transaction.checkBefore(); err != nil {
		return ports.NewInstallUpdateFailure("UPDATE_SOURCE_CHANGED", err)
	}
	names, err := pypiUpgradeMetadataDirectories(transaction.plan.site)
	if err != nil || !reflect.DeepEqual(names, s.directories) {
		return ports.NewInstallUpdateFailure("UPDATE_SOURCE_CHANGED", errors.New("installed distribution inventory changed"))
	}
	for directory, before := range s.emptyMetadata {
		tree, empty, err := snapshotPyPIRetiredMetadata(directory, len(before))
		if err != nil || !empty || len(tree) != len(before) {
			return ports.NewInstallUpdateFailure("UPDATE_SOURCE_CHANGED", errors.New("retired metadata tree changed"))
		}
		for relative, original := range before {
			if !sameRetiredMetadataDirectory(original, tree[relative]) {
				return ports.NewInstallUpdateFailure("UPDATE_SOURCE_CHANGED", errors.New("retired metadata directory identity changed"))
			}
		}
	}
	for path, before := range s.metadata {
		now, err := snapshotBoundedPromotionFile(path, 1<<20)
		if err != nil || !os.SameFile(before.info, now.info) || before.Digest != now.Digest || before.Size != now.Size || before.Mode != now.Mode || rejectPyPIUpgradeDirectSource(filepath.Dir(path)) != nil {
			return ports.NewInstallUpdateFailure("UPDATE_SOURCE_CHANGED", errors.New("installed dependency metadata changed"))
		}
	}
	return ctx.Err()
}

// This bounds allocation as well as parsing. Legacy/editable metadata is not
// silently omitted from the installed dependency constraints.
func pypiUpgradeMetadataDirectories(site string) ([]string, error) {
	file, err := os.Open(site)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	entries, err := file.ReadDir(pypiUpgradeDirectoryLimit + 1)
	if err != nil && err != io.EOF || len(entries) > pypiUpgradeDirectoryLimit {
		return nil, errors.New("PyPI environment inventory exceeds supported bounds")
	}
	names := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".egg-info") || strings.HasSuffix(name, ".egg-link") {
			return nil, errors.New("legacy or editable installed metadata is unsupported")
		}
		if !strings.HasSuffix(name, ".dist-info") {
			continue
		}
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil, errors.New("installed metadata directory is untrusted")
		}
		names = append(names, name)
		if len(names) > artifactpypi.UpgradeInputDistributionLimit {
			return nil, errors.New("PyPI installed distribution count exceeds supported bounds")
		}
	}
	sort.Strings(names)
	return names, nil
}

func rejectPyPIUpgradeDirectSource(directory string) error {
	if _, err := os.Lstat(filepath.Join(directory, "direct_url.json")); !errors.Is(err, os.ErrNotExist) {
		return errors.New("installed direct-source or editable selection is unsupported")
	}
	return nil
}
