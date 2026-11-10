package promotion

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"syscall"
	"unicode/utf8"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
)

// A native member becomes owned only after its complete census matches an
// independent controller approval. A fresh snapshot alone grants no authority.
type pnpmOwnedMember struct {
	entries  map[string]pnpmPreparedEntry
	boundary func() error
}

func capturePNPMOwnedMember(ctx context.Context, name string) (member *pnpmOwnedMember, resultErr error) {
	if ctx == nil || trustedExistingDirectory(name) != nil {
		return nil, errors.New("pnpm native member root is untrusted")
	}
	info, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, root.Close())
		if resultErr != nil {
			member = nil
		}
	}()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.Join(errors.New("pnpm native member root changed"), err)
	}
	m := &pnpmOwnedMember{entries: map[string]pnpmPreparedEntry{}}
	var total, names int64
	err = fs.WalkDir(root.FS(), ".", func(relative string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return errors.Join(err, context.Cause(ctx))
		}
		names += int64(len(relative))
		if len(m.entries) >= npmSnapshotEntryLimit || names > artifactnpm.PNPMLayoutArchiveLimit || !utf8.ValidString(relative) || path.Clean(relative) != relative || strings.ContainsAny(relative, "\\:\x00\r\n") {
			return errors.New("pnpm native census path or entry bound is invalid")
		}
		before, err := root.Lstat(relative)
		if err != nil {
			return err
		}
		entry := pnpmPreparedEntry{info: before}
		switch {
		case before.IsDir():
			if before.Mode()&^os.ModePerm != os.ModeDir || before.Mode().Perm()&0o022 != 0 || before.Mode().Perm()&0o700 != 0o700 {
				return errors.New("pnpm native directory mode is untrusted")
			}
		case before.Mode().IsRegular():
			total += before.Size()
			if before.Mode()&^os.ModePerm != 0 || before.Mode().Perm()&0o022 != 0 || before.Size() < 0 || total > artifactnpm.PNPMLayoutArchiveLimit {
				return errors.New("pnpm native file mode or copy bound is invalid")
			}
			body, err := readPNPMHostRegular(ctx, root, relative, before, artifactnpm.ArchiveFileLimit)
			if err != nil {
				return err
			}
			entry.digest = pnpmCacheHash(body)
		case before.Mode()&os.ModeSymlink != 0:
			if before.Mode()&^os.ModePerm != os.ModeSymlink || !pypiSingleLink(before) {
				return errors.New("pnpm native link has unsupported identity")
			}
			entry.link, err = root.Readlink(relative)
			if err != nil {
				return err
			}
			if entry.link == "" || len(entry.link) > 4096 || !utf8.ValidString(entry.link) || path.Clean(entry.link) != entry.link || path.IsAbs(entry.link) || strings.ContainsAny(entry.link, "\\:\x00\r\n") {
				return errors.New("pnpm native link text is invalid")
			}
		default:
			return errors.New("pnpm native member contains a special file")
		}
		m.entries[relative] = entry
		return nil
	})
	if err != nil {
		return nil, err
	}
	for name, entry := range m.entries {
		if entry.info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target := path.Join(path.Dir(name), entry.link)
		physical, known := m.entries[target]
		if !known || !physical.info.IsDir() && !physical.info.Mode().IsRegular() {
			return nil, errors.New("pnpm native link has no explicit owned physical target")
		}
	}
	if err := m.verify(ctx, name); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *pnpmOwnedMember) verify(ctx context.Context, name string) (resultErr error) {
	if ctx == nil || m == nil || len(m.entries) == 0 || trustedExistingDirectory(name) != nil {
		return errors.New("pnpm native member is unavailable")
	}
	info, err := os.Lstat(name)
	if err != nil || !os.SameFile(m.entries["."].info, info) {
		return errors.Join(errors.New("pnpm native member root identity changed"), err)
	}
	root, err := os.OpenRoot(name)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	count := 0
	err = fs.WalkDir(root.FS(), ".", func(relative string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return errors.Join(err, context.Cause(ctx))
		}
		if err := verifyPNPMOwnedEntry(ctx, root, m.entries, relative); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	if count != len(m.entries) {
		return errors.New("pnpm owned native member is missing")
	}
	return nil
}

func (m *pnpmOwnedMember) remove(name string) (resultErr error) {
	if err := m.verify(context.Background(), name); err != nil {
		return err
	}
	root, err := os.OpenRoot(name)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	var paths []string
	for name := range m.entries {
		if name != "." {
			paths = append(paths, name)
		}
	}
	slices.SortFunc(paths, func(a, b string) int { return len(b) - len(a) })
	for _, relative := range paths {
		if m.boundary != nil {
			if err := m.boundary(); err != nil {
				return err
			}
		}
		current, err := os.Lstat(name)
		before := m.entries["."].info
		if err != nil || !os.SameFile(before, current) || before.Mode() != current.Mode() || !pnpmSameOwner(before, current) {
			return errors.Join(errors.New("pnpm native removal boundary changed"), err)
		}
		if err := verifyPNPMOwnedEntry(context.Background(), root, m.entries, relative); err != nil {
			return err
		}
		if err := root.Remove(relative); err != nil {
			return err
		}
	}
	info, err := os.Lstat(name)
	if err != nil || !os.SameFile(m.entries["."].info, info) || info.Mode() != m.entries["."].info.Mode() {
		return errors.Join(errors.New("pnpm native removal root changed"), err)
	}
	return os.Remove(name)
}

// Persistent approval stores this bounded census digest, not inode snapshots.
// Inodes remain request-local and are checked before every later mutation.
func pnpmMemberDigest(entries map[string]pnpmPreparedEntry) (string, error) {
	type record struct {
		Name, Kind, Digest, Link string
		Mode                     uint32
		UID, GID                 uint32
		Size                     int64
	}
	var names []string
	for name := range entries {
		names = append(names, name)
	}
	slices.Sort(names)
	rows := make([]record, 0, len(names))
	for _, name := range names {
		entry := entries[name]
		stat, ok := entry.info.Sys().(*syscall.Stat_t)
		if !ok {
			return "", errors.New("pnpm census ownership is unavailable")
		}
		kind := "file"
		size := entry.info.Size()
		if entry.info.IsDir() {
			kind = "directory"
			size = 0
		} else if entry.info.Mode()&os.ModeSymlink != 0 {
			kind = "symlink"
			size = 0
		}
		rows = append(rows, record{name, kind, entry.digest, entry.link, uint32(entry.info.Mode()), stat.Uid, stat.Gid, size})
	}
	body, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	return pnpmCacheHash(body), nil
}

func verifyPNPMOwnedEntry(ctx context.Context, root *os.Root, entries map[string]pnpmPreparedEntry, name string) error {
	for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
		before, known := entries[parent]
		now, err := root.Lstat(parent)
		if !known || err != nil || !before.info.IsDir() || !now.IsDir() || !os.SameFile(before.info, now) || before.info.Mode() != now.Mode() || !pnpmSameOwner(before.info, now) {
			return errors.Join(errors.New("pnpm prepared ancestor directory changed"), err)
		}
	}
	before, known := entries[name]
	if !known {
		return errors.New("pnpm prepared project contains a foreign member")
	}
	now, err := root.Lstat(name)
	if err != nil || !os.SameFile(before.info, now) || before.info.Mode() != now.Mode() || !pnpmSameOwner(before.info, now) {
		return errors.Join(errors.New("pnpm prepared project member identity changed"), err)
	}
	if now.Mode().IsRegular() {
		body, err := readPNPMHostRegular(ctx, root, name, before.info, before.info.Size())
		if err != nil {
			return err
		}
		if pnpmCacheHash(body) != before.digest {
			return errors.New("pnpm prepared project bytes changed")
		}
	} else if now.Mode()&os.ModeSymlink != 0 {
		link, err := root.Readlink(name)
		if err != nil || link != before.link || !pypiSingleLink(now) {
			return errors.Join(errors.New("pnpm prepared internal link changed"), err)
		}
	} else if !now.IsDir() {
		return errors.New("pnpm prepared project has a special file")
	}
	return nil
}
