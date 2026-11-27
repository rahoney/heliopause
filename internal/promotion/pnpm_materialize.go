package promotion

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type pnpmHostEntry struct {
	name, link string
	kind       byte
	mode       os.FileMode
	body       []byte
}

type pnpmPreparedEntry struct {
	info         os.FileInfo
	digest, link string
}

// PNPMPreparedProject owns an unpublished, private sibling workspace. It grants
// no target mutation, retained approval or package execution authority.
type PNPMPreparedProject struct {
	directory, name      string
	parentName           string
	root, parent         *os.Root
	parentInfo, rootInfo os.FileInfo
	entries              map[string]pnpmPreparedEntry
	closed               bool
	beforeRemove         func() error // Diagnostic abort; per-entry ownership checks still follow.
}

func (p *PNPMPreparedProject) Directory() string {
	if p == nil || p.closed {
		return ""
	}
	return p.directory
}

func pnpmSameOwner(a, b os.FileInfo) bool {
	x, ok := a.Sys().(*syscall.Stat_t)
	y, other := b.Sys().(*syscall.Stat_t)
	return ok && other && x.Uid == y.Uid && x.Gid == y.Gid
}

func readPNPMHostRegular(ctx context.Context, root *os.Root, name string, before os.FileInfo, limit int64) (body []byte, resultErr error) {
	if ctx == nil || before == nil || !before.Mode().IsRegular() || !pypiSingleLink(before) || before.Size() < 0 || before.Size() > limit {
		return nil, errors.New("pnpm Host file is not bounded single-link regular content")
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, f.Close())
		if resultErr != nil {
			body = nil
		}
	}()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) || !opened.Mode().IsRegular() {
		return nil, errors.Join(errors.New("pnpm Host file changed while opening"), err)
	}
	body, err = io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	after, statErr := f.Stat()
	current, currentErr := root.Lstat(name)
	if statErr != nil || currentErr != nil || !os.SameFile(before, after) || !os.SameFile(before, current) || after.Mode() != before.Mode() || current.Mode() != before.Mode() || !pnpmSameOwner(before, after) || !pnpmSameOwner(before, current) || !pypiSingleLink(after) || !pypiSingleLink(current) || int64(len(body)) != before.Size() || after.Size() != before.Size() || current.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) || !current.ModTime().Equal(before.ModTime()) {
		return nil, errors.Join(errors.New("pnpm Host file changed during read"), statErr, currentErr)
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	return body, nil
}

func (c *PNPMVerifiedCache) hostEntries(ctx context.Context, staged domain.StagedProjectSet) ([]pnpmHostEntry, error) {
	data, err := c.OpenProjectCache(ctx, staged)
	if err != nil {
		return nil, err
	}
	return npmProjectHostEntries(ctx, data, staged.Digest(), c.frozen.Controls(), func(doc pnpmCacheDocument) (pnpmProjectRecords, error) {
		return c.projectRecords(ctx, staged.Set(), doc)
	}, npmProjectCachePNPM)
}

func npmProjectHostEntries(ctx context.Context, data string, digest domain.ContentDigest, controls []domain.ProjectControlFile, records func(pnpmCacheDocument) (pnpmProjectRecords, error), client npmProjectCacheClient) (entries []pnpmHostEntry, resultErr error) {
	spec, known := npmProjectCacheIdentity(client)
	if !known || records == nil {
		return nil, errors.New("unknown project materialization client")
	}
	cache, err := os.OpenRoot(filepath.Dir(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, cache.Close())
		if resultErr != nil {
			entries = nil
		}
	}()
	receipt, _, err := readGoTransactionControl(cache, "receipt.json")
	if err != nil || pnpmCacheHash(receipt) != digest.String() {
		return nil, errors.Join(errors.New("pnpm prepared cache receipt changed"), err)
	}
	var doc pnpmCacheDocument
	if json.Unmarshal(receipt, &doc) != nil {
		return nil, errors.New("pnpm prepared cache receipt is invalid")
	}
	if !doc.DependencyFree {
		info, err := cache.Lstat("data/" + pnpmCacheArchiveName)
		if err != nil {
			return nil, err
		}
		if info.Mode().Perm() != 0o444 {
			return nil, errors.New("pnpm prepared cache archive is not sealed")
		}
		body, err := readPNPMHostRegular(ctx, cache, "data/"+pnpmCacheArchiveName, info, artifactnpm.PNPMLayoutArchiveLimit)
		if err != nil {
			return nil, err
		}
		found := false
		for _, record := range doc.Files {
			if record.Path == pnpmCacheArchiveName && record.Kind == "file" && record.Size == int64(len(body)) && record.SHA256 == pnpmCacheHash(body) {
				found = true
			}
		}
		if !found {
			return nil, errors.New("pnpm prepared native bytes differ from sealed approval")
		}
		view, err := artifactnpm.ParsePNPMLayoutArchive(ctx, body)
		if err != nil {
			return nil, err
		}
		byName := map[string]byte{}
		for _, entry := range view.Entries() {
			byName[entry.Name] = entry.Kind
		}
		var copied int64
		for _, entry := range view.Entries() {
			item := pnpmHostEntry{name: entry.Name, link: entry.Link, kind: entry.Kind, mode: os.FileMode(entry.Mode)}
			if entry.Kind == tar.TypeReg || entry.Kind == tar.TypeLink {
				item.body, err = view.ReadFile(entry.Name)
				if err != nil {
					return nil, err
				}
				copied += int64(len(item.body))
				if copied > artifactnpm.PNPMLayoutArchiveLimit {
					return nil, errors.New("pnpm private copies exceed the fixed native transport bound")
				}
				item.kind = tar.TypeReg
				item.link = ""
			} else if entry.Kind == tar.TypeSymlink {
				target := path.Join(path.Dir(entry.Name), entry.Link)
				kind := byName[target]
				if kind != tar.TypeDir && kind != tar.TypeReg && kind != tar.TypeLink {
					return nil, errors.New("pnpm Host link lacks an explicit owned physical target")
				}
			}
			entries = append(entries, item)
		}
	} else {
		// An explicit empty project has no native tree; this directory is controller-owned.
		entries = append(entries, pnpmHostEntry{name: "node_modules", kind: tar.TypeDir, mode: 0o755})
	}
	for _, control := range controls {
		entries = append(entries, pnpmHostEntry{name: control.Name(), kind: tar.TypeReg, mode: 0o644, body: control.Body()})
	}
	material, err := records(doc)
	if err != nil {
		return nil, err
	}
	if material.manifest.String() != doc.Manifest {
		return nil, errors.New("pnpm prepared project Manifest changed")
	}
	entries = append(entries, pnpmHostEntry{name: ".heliopause", kind: tar.TypeDir, mode: 0o700})
	for _, record := range []struct {
		name string
		body []byte
	}{{"manifest.json", material.document}, {"sbom.cdx.json", material.sbom}, {spec.provenanceName, material.provenance}} {
		found := false
		for _, file := range doc.Records {
			if file.Path == record.name && file.Kind == "file" && file.Size == int64(len(record.body)) && file.SHA256 == pnpmCacheHash(record.body) {
				found = true
			}
		}
		if !found {
			return nil, errors.New("pnpm prepared records differ from sealed approval")
		}
		entries = append(entries, pnpmHostEntry{name: ".heliopause/" + record.name, kind: tar.TypeReg, mode: 0o444, body: record.body})
	}
	return entries, nil
}

// PrepareProject copies only the approved bytes into a fresh target sibling.
// Every native hardlink becomes a private regular copy; links are internal only.
func (c *PNPMVerifiedCache) PrepareProject(ctx context.Context, staged domain.StagedProjectSet) (prepared *PNPMPreparedProject, resultErr error) {
	entries, err := c.hostEntries(ctx, staged)
	if err != nil {
		return nil, err
	}
	return prepareNPMProject(ctx, c.frozen.Snapshot().Context().Target().String(), entries, c.beforePrepareSync, npmProjectCachePNPM)
}

func prepareNPMProject(ctx context.Context, target string, entries []pnpmHostEntry, beforePrepareSync func(*PNPMPreparedProject) error, client npmProjectCacheClient) (prepared *PNPMPreparedProject, resultErr error) {
	spec, known := npmProjectCacheIdentity(client)
	if !known {
		return nil, errors.New("unknown private project client")
	}
	parentName := filepath.Dir(target)
	if err := trustedExistingDirectory(parentName); err != nil {
		return nil, err
	}
	parentInfo, err := os.Lstat(parentName)
	if err != nil {
		return nil, err
	}
	parent, err := os.OpenRoot(parentName)
	if err != nil {
		return nil, err
	}
	p := &PNPMPreparedProject{parentName: parentName, parentInfo: parentInfo, parent: parent, entries: map[string]pnpmPreparedEntry{}}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, p.Close())
			prepared = nil
		}
	}()
	opened, err := parent.Stat(".")
	if err != nil || !os.SameFile(parentInfo, opened) {
		return nil, errors.Join(errors.New("pnpm prepared parent changed"), err)
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	run, err := domain.NewRunID()
	if err != nil {
		return nil, err
	}
	p.name = ".haa-" + spec.prefix + "-prepare-" + run.String()
	p.directory = filepath.Join(parentName, p.name)
	if err := parent.Mkdir(p.name, 0o700); err != nil {
		p.name = ""
		return nil, err
	}
	p.rootInfo, err = parent.Lstat(p.name)
	if err != nil {
		return nil, err
	}
	p.root, err = parent.OpenRoot(p.name)
	if err != nil {
		return nil, err
	}
	p.entries["."] = pnpmPreparedEntry{info: p.rootInfo}
	slices.SortFunc(entries, func(a, b pnpmHostEntry) int { return strings.Compare(a.name, b.name) })
	for _, kind := range []byte{tar.TypeDir, tar.TypeReg, tar.TypeSymlink} {
		for _, entry := range entries {
			if entry.kind != kind {
				continue
			}
			if err := ctx.Err(); err != nil {
				return nil, errors.Join(err, context.Cause(ctx))
			}
			if err := p.create(entry); err != nil {
				return nil, err
			}
		}
	}
	if beforePrepareSync != nil {
		if err := beforePrepareSync(p); err != nil {
			return nil, err
		}
	}
	if err := p.Verify(ctx); err != nil {
		return nil, err
	}
	var directories []string
	for name, entry := range p.entries {
		if entry.info.IsDir() {
			directories = append(directories, name)
		}
	}
	slices.SortFunc(directories, func(a, b string) int { return len(b) - len(a) })
	for _, name := range directories {
		if err := syncPNPMDirectory(p.root, name, p.entries[name].info); err != nil {
			return nil, err
		}
	}
	if err := syncPNPMDirectory(p.parent, ".", p.parentInfo); err != nil {
		return nil, err
	}
	if err := p.Verify(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PNPMPreparedProject) create(entry pnpmHostEntry) error {
	switch entry.kind {
	case tar.TypeDir:
		if err := p.root.Mkdir(entry.name, 0o700); err != nil {
			return err
		}
		f, err := p.root.Open(entry.name)
		if err != nil {
			return err
		}
		modeErr := f.Chmod(entry.mode)
		info, statErr := f.Stat()
		closeErr := f.Close()
		if statErr == nil {
			p.entries[entry.name] = pnpmPreparedEntry{info: info}
		}
		if modeErr != nil || statErr != nil || closeErr != nil {
			return errors.Join(modeErr, statErr, closeErr)
		}
		return nil
	case tar.TypeSymlink:
		if err := p.root.Symlink(entry.link, entry.name); err != nil {
			return err
		}
	case tar.TypeReg:
		f, err := p.root.OpenFile(entry.name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return err
		}
		n, writeErr := f.Write(entry.body)
		modeErr := f.Chmod(entry.mode)
		syncErr := f.Sync()
		info, statErr := f.Stat()
		closeErr := f.Close()
		if statErr == nil && n >= 0 && n <= len(entry.body) {
			p.entries[entry.name] = pnpmPreparedEntry{info: info, digest: pnpmCacheHash(entry.body[:n])}
		}
		if writeErr != nil || modeErr != nil || syncErr != nil || statErr != nil || closeErr != nil || n != len(entry.body) {
			return errors.Join(errors.New("write private pnpm project file"), writeErr, modeErr, syncErr, statErr, closeErr)
		}
		return nil
	default:
		return errors.New("pnpm private member has unsupported type")
	}
	info, err := p.root.Lstat(entry.name)
	if err != nil {
		return err
	}
	p.entries[entry.name] = pnpmPreparedEntry{info: info, link: entry.link}
	return nil
}

func (p *PNPMPreparedProject) verifyBoundary() error {
	parent, err := os.Lstat(p.parentName)
	if err != nil || !os.SameFile(parent, p.parentInfo) || parent.Mode() != p.parentInfo.Mode() || !pnpmSameOwner(parent, p.parentInfo) {
		return errors.Join(errors.New("pnpm prepared parent identity changed"), err)
	}
	root, err := p.parent.Lstat(p.name)
	if err != nil || !os.SameFile(root, p.rootInfo) || root.Mode() != p.rootInfo.Mode() || root.Mode().Perm() != 0o700 || !pnpmSameOwner(root, p.rootInfo) {
		return errors.Join(errors.New("pnpm prepared root identity changed"), err)
	}
	opened, err := p.root.Stat(".")
	if err != nil || !os.SameFile(p.rootInfo, opened) {
		return errors.Join(errors.New("pnpm prepared root descriptor changed"), err)
	}
	return nil
}

func (p *PNPMPreparedProject) verifyEntry(ctx context.Context, name string) error {
	return verifyPNPMOwnedEntry(ctx, p.root, p.entries, name)
}

func (p *PNPMPreparedProject) Verify(ctx context.Context) error {
	if p == nil || p.closed || p.root == nil || ctx == nil {
		return errors.New("pnpm prepared project is not active")
	}
	if err := p.verifyBoundary(); err != nil {
		return err
	}
	count := 0
	err := fs.WalkDir(p.root.FS(), ".", func(name string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return errors.Join(err, context.Cause(ctx))
		}
		if err := p.verifyEntry(ctx, name); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	if count != len(p.entries) {
		return errors.New("pnpm prepared project member is missing")
	}
	return nil
}

// Close removes only the exact owned inventory. Unknown data is preserved.
func (p *PNPMPreparedProject) Close() (resultErr error) {
	if p == nil || p.closed {
		return errors.New("pnpm prepared project is already closed")
	}
	defer func() {
		p.closed = true
		if p.root != nil {
			resultErr = errors.Join(resultErr, p.root.Close())
		}
		if p.parent != nil {
			resultErr = errors.Join(resultErr, p.parent.Close())
		}
	}()
	if p.name == "" {
		return nil
	}
	if err := p.Verify(context.Background()); err != nil {
		return err
	}
	if p.beforeRemove != nil {
		if err := p.beforeRemove(); err != nil {
			return err
		}
	}
	var names []string
	for name := range p.entries {
		if name != "." {
			names = append(names, name)
		}
	}
	slices.SortFunc(names, func(a, b string) int { return len(b) - len(a) })
	for _, name := range names {
		if err := p.verifyBoundary(); err != nil {
			return err
		}
		if err := p.verifyEntry(context.Background(), name); err != nil {
			return err
		}
		if err := p.root.Remove(name); err != nil {
			return err
		}
	}
	info, err := p.parent.Lstat(p.name)
	if err != nil || !os.SameFile(info, p.rootInfo) {
		return errors.Join(errors.New("pnpm prepared cleanup root changed"), err)
	}
	if err := p.parent.Remove(p.name); err != nil {
		return err
	}
	return syncPNPMDirectory(p.parent, ".", p.parentInfo)
}

func syncPNPMDirectory(root *os.Root, name string, before os.FileInfo) error {
	f, err := root.Open(name)
	if err != nil {
		return err
	}
	info, statErr := f.Stat()
	if statErr != nil || !info.IsDir() || !os.SameFile(before, info) || before.Mode() != info.Mode() || !pnpmSameOwner(before, info) {
		return errors.Join(errors.New("pnpm directory changed before sync"), statErr, f.Close())
	}
	return errors.Join(f.Sync(), f.Close())
}
