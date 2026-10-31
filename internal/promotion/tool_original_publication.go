package promotion

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

type toolOriginalLink struct {
	toolPublishedLink
	Backup    string
	Backed    bool
	Discarded bool
}

type toolOriginalLinkJournal struct {
	Root  string
	Links []toolOriginalLink
}

type toolOriginalPublication struct {
	owner      *toolPublication
	path       string
	root       *os.Root
	info       os.FileInfo
	links      []toolOriginalLink
	disposed   bool
	disposeErr error
}

func prepareToolOriginalPublication(p *toolPublication) (*toolOriginalPublication, error) {
	if p.guard.original == nil {
		return nil, nil
	}
	o := &toolOriginalPublication{owner: p}
	var err error
	o.path, err = os.MkdirTemp(p.guard.binRoot, ".haa-tool-original-")
	if err != nil {
		return o, err
	}
	o.info, err = os.Lstat(o.path)
	if err != nil {
		return o, err
	}
	o.root, err = os.OpenRoot(o.path)
	if err != nil {
		return o, err
	}
	if err := p.guard.tx.freezeParents(o.path); err != nil {
		return o, err
	}
	for _, link := range p.guard.original.links {
		dev, ino, ok := pypiObjectIdentity(link.info)
		if !ok {
			return o, errors.New("original link journal identity unavailable")
		}
		link.Device = dev
		link.Inode = ino
		o.links = append(o.links, toolOriginalLink{toolPublishedLink: link, Backup: filepath.Join(o.path, link.Name)})
	}
	return o, o.verify()
}

func (o *toolOriginalPublication) journal() *toolOriginalLinkJournal {
	if o == nil {
		return nil
	}
	return &toolOriginalLinkJournal{Root: o.path, Links: append([]toolOriginalLink(nil), o.links...)}
}

func (o *toolOriginalPublication) verifyRoot() error {
	if o == nil || o.path == "" {
		return nil
	}
	if o.disposed {
		return errors.New("original link backup already disposed")
	}
	if o.root == nil || o.info == nil {
		return errors.New("original link backup identity unavailable")
	}
	if err := o.owner.guard.bin.verify(); err != nil {
		return err
	}
	info, err := os.Lstat(o.path)
	if err != nil || !os.SameFile(o.info, info) || info.Mode() != o.info.Mode() {
		return errors.New("original link backup root changed")
	}
	opened, err := o.root.Stat(".")
	if err != nil || !os.SameFile(o.info, opened) {
		return errors.New("original link backup descriptor changed")
	}
	return nil
}

func (o *toolOriginalPublication) verify() error {
	if o == nil || o.path == "" {
		return nil
	}
	if o.disposed {
		if o.disposeErr != nil {
			return o.disposeErr
		}
		if _, err := o.owner.guard.bin.root.Lstat(filepath.Base(o.path)); !errors.Is(err, os.ErrNotExist) {
			return errors.New("disposed original backup path reappeared")
		}
		return nil
	}
	if err := o.verifyRoot(); err != nil {
		return err
	}
	file, err := o.root.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := file.ReadDir(len(o.links) + 1)
	closeErr := file.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil {
		return errors.Join(readErr, closeErr)
	}
	known := map[string]bool{}
	for _, link := range o.links {
		if link.Backed && !link.Discarded {
			known[link.Name] = true
			info, err := o.root.Lstat(link.Name)
			if err != nil || !singleToolSymlink(info) || !toolLinkMatches(o.root, link.Name, link.toolPublishedLink) {
				return errors.New("original command backup changed")
			}
		}
		if !link.Backed {
			if !toolLinkMatches(o.owner.guard.bin.root, link.Name, link.toolPublishedLink) {
				return errors.New("original exposed link changed before retirement or after rollback")
			}
			continue
		}
		active := false
		for _, next := range o.owner.links {
			if next.Name == link.Name && next.published && !next.rolledBack {
				active = true
				if !toolLinkMatches(o.owner.guard.bin.root, next.Name, next) {
					return errors.New("replacement tool link changed")
				}
			}
		}
		if !active {
			if _, err := o.owner.guard.bin.root.Lstat(link.Name); !errors.Is(err, os.ErrNotExist) {
				return errors.New("retired command destination is occupied")
			}
		}
	}
	if len(entries) != len(known) {
		return errors.New("original link backup contains foreign or missing data")
	}
	for _, entry := range entries {
		if !known[entry.Name()] {
			return errors.New("unknown original link backup member")
		}
	}
	return nil
}

func (o *toolOriginalPublication) backup() error {
	if o == nil {
		return nil
	}
	t := o.owner.guard.tx
	for i := range o.links {
		link := &o.links[i]
		if err := o.owner.ctx.Err(); err != nil {
			return err
		}
		if err := t.step("before-tool-retire:" + link.Name); err != nil {
			return err
		}
		if err := o.owner.verifyBoundary(); err != nil {
			return err
		}
		if err := t.journalIntent("tool-original-backup", link.Path); err != nil {
			return err
		}
		if err := t.rename(link.Path, link.Backup); err != nil {
			return err
		}
		link.Backed = true
		if err := t.appendJournal(struct{ ToolOriginalLink toolOriginalLink }{*link}); err != nil {
			return err
		}
		if err := o.verify(); err != nil {
			return err
		}
		if err := t.step("after-tool-retire:" + link.Name); err != nil {
			return err
		}
	}
	return syncDirectory(o.owner.guard.binRoot)
}

func (o *toolOriginalPublication) rollback() error {
	if o == nil {
		return nil
	}
	for i := len(o.links) - 1; i >= 0; i-- {
		link := &o.links[i]
		if !link.Backed {
			continue
		}
		if link.Discarded {
			return errors.New("original command backup was disposed; recovery required")
		}
		if err := o.verify(); err != nil {
			return err
		}
		if err := o.owner.guard.tx.journalIntent("tool-original-restore", link.Path); err != nil {
			return err
		}
		if err := o.owner.guard.tx.rename(link.Backup, link.Path); err != nil {
			return err
		}
		link.Backed = false
		if err := o.owner.guard.tx.appendJournal(struct{ ToolOriginalRestore toolOriginalLink }{*link}); err != nil {
			return err
		}
	}
	return syncDirectory(o.owner.guard.binRoot)
}

// Disposal is after successful rollback/private cleanup and before ordinary
// file-backup disposal. Any uncertainty preserves that ordinary backup/journal.
func (p *toolPublication) disposeOriginal() error {
	if p == nil || p.original == nil {
		return nil
	}
	o := p.original
	if o.disposed {
		return o.disposeErr
	}
	if o.path == "" {
		return nil
	}
	if p.guard.tx.journalStarted {
		if err := p.guard.tx.step("before-tool-original-disposal"); err != nil {
			return err
		}
	}
	if err := o.verify(); err != nil {
		return err
	}
	for i := range o.links {
		link := &o.links[i]
		if !link.Backed || link.Discarded {
			continue
		}
		if err := o.verifyRoot(); err != nil {
			return err
		}
		if !toolLinkMatches(o.root, link.Name, link.toolPublishedLink) {
			return errors.New("original command backup changed before disposal")
		}
		if p.guard.tx.journalStarted {
			if err := p.guard.tx.journalIntent("tool-original-dispose", link.Backup); err != nil {
				return err
			}
		}
		if err := o.root.Remove(link.Name); err != nil {
			return err
		}
		link.Discarded = true
	}
	if err := o.verify(); err != nil {
		return err
	}
	file, err := o.root.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := file.ReadDir(1)
	closeErr := file.Close()
	if len(entries) != 0 || !errors.Is(readErr, io.EOF) || closeErr != nil {
		return errors.New("original link backup is not empty before disposal")
	}
	if err := o.root.Close(); err != nil {
		return err
	}
	o.disposed = true
	o.disposeErr = p.guard.bin.root.Remove(filepath.Base(o.path))
	if o.disposeErr == nil {
		o.disposeErr = syncDirectory(p.guard.binRoot)
	}
	return o.disposeErr
}
