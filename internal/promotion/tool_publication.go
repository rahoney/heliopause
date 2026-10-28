package promotion

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const uvToolHostReceiptControl = "uv-tool-host-receipt.toml"

type toolPublishedLink struct {
	Name, Path, Source, Target string
	Device, Inode              uint64
	info                       os.FileInfo
	published                  bool
	rolledBack                 bool
}

type toolPublication struct {
	original        *toolOriginalPublication
	ctx             context.Context
	guard           *toolFreshTarget
	receipt         pypiDestination
	receiptSnapshot pypiFileSnapshot
	private         string
	privateRoot     *os.Root
	privateInfo     os.FileInfo
	links           []toolPublishedLink
	cleanupDone     bool
	cleanupErr      error
}

type toolPublicationJournal struct {
	Original             *toolOriginalLinkJournal `json:",omitempty"`
	Workspace            *toolWorkspaceJournal    `json:",omitempty"`
	BinRoot, PrivateRoot string
	Receipt              pypiDestination
	Links                []toolPublishedLink
}

func (g *toolFreshTarget) commitTool(ctx context.Context, destinations []pypiDestination, receipt domain.ProjectControlFile, temporary string) (resultErr error) {
	if err := g.verify(ctx); err != nil {
		return err
	}
	if g.tx.tool != nil || g.tx.project != nil || len(g.commands) == 0 || !receipt.Present() || receipt.Name() != g.client.receiptControl() {
		return errors.New("tool publication requires one frozen controller receipt")
	}
	parsed, err := g.client.parseReceipt(receipt.Body())
	nativeRequirement := g.requirement
	if g.original != nil {
		nativeRequirement = g.original.input.Receipt().Requirement()
		if !g.original.selected.Valid() || !bytes.Equal(g.original.selected.Body(), receipt.Body()) {
			return errors.New("original tool receipt differs from frozen selection")
		}
	}
	if err != nil || !reflect.DeepEqual(g.client.exposedCommands(parsed, g.binRoot), g.commands) || !parsed.Requirement().SameConstraints(nativeRequirement) {
		return errors.New("tool publication receipt differs from frozen native selection")
	}
	if err := parsed.ValidateLayout(g.tx.plan.root, g.binRoot, filepath.Join(g.tx.plan.root, "bin", "python")); err != nil {
		return err
	}
	for _, command := range g.commands {
		found := false
		for _, d := range destinations {
			if d.Scheme == "scripts" && d.Relative == command.Name && d.Distribution == g.requirement.Project() {
				info, err := os.Lstat(d.Source)
				if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 || d.ArtifactDigest == "" || !g.requirement.AcceptsVersion(d.Version) || !nativeRequirement.AcceptsVersion(d.Version) {
					return errors.New("tool command lacks its main distribution executable output")
				}
				found = true
			}
		}
		if !found {
			return errors.New("tool command lacks its selected distribution output")
		}
	}
	p := &toolPublication{guard: g, ctx: ctx}
	defer func() {
		cleanupErr := p.cleanup()
		if !g.tx.journalStarted {
			cleanupErr = errors.Join(cleanupErr, p.disposeOriginal())
		}
		if cleanupErr != nil {
			g.tx.recovery = true
		}
		resultErr = errors.Join(resultErr, cleanupErr)
	}()
	if !filepath.IsAbs(temporary) || filepath.Clean(temporary) != temporary || trustedExistingDirectory(temporary) != nil {
		return errors.New("tool receipt source workspace is unsafe")
	}
	file, err := os.CreateTemp(temporary, ".haa-tool-receipt-")
	if err != nil {
		return err
	}
	p.receipt.Source = file.Name()
	_, writeErr := file.Write(receipt.Body())
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	p.receiptSnapshot, err = snapshotBoundedPromotionFile(file.Name(), 1<<20)
	if err != nil {
		return err
	}
	if p.receiptSnapshot.Digest != receipt.Digest().String() {
		return errors.New("private tool receipt copy differs")
	}
	p.receipt = pypiDestination{Scheme: "tool-control", Relative: g.client.receiptName(), Final: filepath.Join(g.tx.plan.root, g.client.receiptName()), Source: file.Name(), Digest: receipt.Digest().String(), Size: int64(len(receipt.Body()))}
	p.private, err = os.MkdirTemp(g.binRoot, ".haa-tool-links-")
	if err != nil {
		return err
	}
	p.privateInfo, err = os.Lstat(p.private)
	if err != nil {
		return err
	}
	p.privateRoot, err = os.OpenRoot(p.private)
	if err != nil {
		return err
	}
	opened, err := p.privateRoot.Stat(".")
	if err != nil || !os.SameFile(p.privateInfo, opened) {
		return errors.New("private tool link root changed")
	}
	if err := g.tx.freezeParents(p.private); err != nil {
		return err
	}
	for _, command := range g.commands {
		target := filepath.Join(g.tx.plan.root, "bin", command.Name)
		if err := p.privateRoot.Symlink(target, command.Name); err != nil {
			return err
		}
		info, err := p.privateRoot.Lstat(command.Name)
		if err != nil {
			return err
		}
		device, inode, ok := pypiObjectIdentity(info)
		if !ok {
			return errors.New("private tool link identity unavailable")
		}
		p.links = append(p.links, toolPublishedLink{Name: command.Name, Path: command.Path, Source: filepath.Join(p.private, command.Name), Target: target, Device: device, Inode: inode, info: info})
	}
	if err := g.verify(ctx); err != nil {
		return err
	}
	p.original, err = prepareToolOriginalPublication(p)
	if err != nil {
		return err
	}
	g.tx.tool = p
	err = g.tx.commitSelectionWithProject(destinations, g.original != nil, false)
	if err == nil && g.original != nil {
		g.original.committed = true
	}
	return err
}

func (p *toolPublication) journal() *toolPublicationJournal {
	if p == nil {
		return nil
	}
	return &toolPublicationJournal{Original: p.original.journal(), Workspace: p.guard.work.journal(), BinRoot: p.guard.binRoot, PrivateRoot: p.private, Receipt: p.receipt, Links: append([]toolPublishedLink(nil), p.links...)}
}

func toolLinkMatches(root *os.Root, name string, link toolPublishedLink) bool {
	info, err := root.Lstat(name)
	if err != nil || info.Mode()&os.ModeSymlink == 0 || !os.SameFile(link.info, info) || info.Mode() != link.info.Mode() {
		return false
	}
	target, err := root.Readlink(name)
	return err == nil && target == link.Target
}

func (p *toolPublication) verifyBoundary() error {
	if p == nil {
		return nil
	}
	if err := p.original.verify(); err != nil {
		return err
	}
	g := p.guard
	if g == nil {
		return errors.New("tool publication guard unavailable")
	}
	if err := g.bin.verify(); err != nil {
		return err
	}
	if err := toolOwnedDirectory(g.binRoot); err != nil {
		return err
	}
	for _, member := range g.interpreter {
		info, err := os.Lstat(member.path)
		if err != nil || !os.SameFile(member.info, info) || member.info.Mode() != info.Mode() {
			return errors.New("tool publication interpreter identity changed")
		}
		if member.file != nil {
			if !member.file.matches(member.path) {
				return errors.New("tool publication interpreter bytes changed")
			}
		} else {
			link, err := os.Readlink(member.path)
			if err != nil || link != member.link {
				return errors.New("tool publication interpreter alias changed")
			}
		}
	}
	return nil
}

func (t *pypiVenvTransaction) prepareToolControls() ([]pypiDestination, error) {
	if t.tool == nil {
		return nil, nil
	}
	p := t.tool
	if err := p.verifyBoundary(); err != nil {
		return nil, err
	}
	before, frozen := t.before[p.receipt.Final]
	if !frozen || (p.original == nil && before != nil) || (p.original != nil && (before == nil || !before.matches(p.receipt.Final))) || !p.receiptSnapshot.matches(p.receipt.Source) {
		return nil, errors.New("tool receipt lacks frozen absence or private copy")
	}
	if err := t.freezeParents(filepath.Dir(p.receipt.Source)); err != nil {
		return nil, err
	}
	t.outputs[p.receipt.Final] = p.receiptSnapshot
	return []pypiDestination{p.receipt}, nil
}

func (p *toolPublication) publishLinks() error {
	if p == nil {
		return nil
	}
	t := p.guard.tx
	if err := p.original.backup(); err != nil {
		return err
	}
	for i := range p.links {
		link := &p.links[i]
		if err := p.ctx.Err(); err != nil {
			return err
		}
		if err := t.step("before-tool-link:" + link.Name); err != nil {
			return err
		}
		if err := p.verifyBoundary(); err != nil {
			return err
		}
		if !toolLinkMatches(p.privateRoot, link.Name, *link) {
			return errors.New("private tool link changed before publication")
		}
		if err := t.journalIntent("tool-link", link.Path); err != nil {
			return err
		}
		if err := t.rename(link.Source, link.Path); err != nil {
			return err
		}
		link.published = true
		if err := t.appendJournal(struct{ ToolLink toolPublishedLink }{*link}); err != nil {
			return err
		}
		if !toolLinkMatches(p.guard.bin.root, link.Name, *link) {
			return errors.New("published tool link changed")
		}
		if err := t.step("after-tool-link:" + link.Name); err != nil {
			return err
		}
	}
	return syncDirectory(p.guard.binRoot)
}

func (p *toolPublication) verifyPublished() error {
	if p == nil {
		return nil
	}
	if err := p.ctx.Err(); err != nil {
		return err
	}
	if err := p.verifyBoundary(); err != nil {
		return err
	}
	if err := p.guard.verifyCommandCaseCollisions(); err != nil {
		return err
	}
	for _, link := range p.links {
		if !link.published || !toolLinkMatches(p.guard.bin.root, link.Name, link) {
			return errors.New("tool command link reconciliation failed")
		}
	}
	return nil
}

func (p *toolPublication) rollbackLinks() error {
	if p == nil {
		return nil
	}
	if err := p.verifyBoundary(); err != nil {
		return err
	}
	for i := len(p.links) - 1; i >= 0; i-- {
		link := p.links[i]
		if !link.published || link.rolledBack {
			continue
		}
		if !toolLinkMatches(p.guard.bin.root, link.Name, link) {
			return errors.New("foreign tool link prevents rollback")
		}
		if err := p.guard.tx.remove(link.Path); err != nil {
			return err
		}
		p.links[i].rolledBack = true
	}
	if err := p.original.rollback(); err != nil {
		return err
	}
	return syncDirectory(p.guard.binRoot)
}

func (p *toolPublication) cleanup() (resultErr error) {
	if p.cleanupDone {
		return p.cleanupErr
	}
	p.cleanupDone = true
	defer func() { p.cleanupErr = resultErr }()
	if err := p.original.verify(); err != nil {
		return err
	}
	if p.receipt.Source != "" {
		if _, err := os.Lstat(p.receipt.Source); err == nil {
			if p.receiptSnapshot.info == nil || !p.receiptSnapshot.matches(p.receipt.Source) {
				resultErr = errors.Join(resultErr, errors.New("private tool receipt cleanup uncertain"))
			} else {
				resultErr = errors.Join(resultErr, os.Remove(p.receipt.Source))
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, err)
		}
	}
	if p.private == "" {
		return resultErr
	}
	if p.privateRoot == nil {
		return errors.Join(resultErr, errors.New("private tool link directory retained without identity"))
	}
	defer func() { resultErr = errors.Join(resultErr, p.privateRoot.Close()) }()
	info, err := os.Lstat(p.private)
	if err != nil || !os.SameFile(p.privateInfo, info) || info.Mode() != p.privateInfo.Mode() {
		return errors.Join(resultErr, errors.New("private tool link root cleanup changed"))
	}
	file, err := p.privateRoot.Open(".")
	if err != nil {
		return errors.Join(resultErr, err)
	}
	entries, readErr := file.ReadDir(len(p.links) + 1)
	closeErr := file.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) || closeErr != nil {
		return errors.Join(resultErr, readErr, closeErr)
	}
	known := map[string]toolPublishedLink{}
	for _, link := range p.links {
		if !link.published {
			known[link.Name] = link
		}
	}
	if len(entries) != len(known) {
		return errors.Join(resultErr, errors.New("private tool links contain foreign or missing data"))
	}
	for _, entry := range entries {
		link, ok := known[entry.Name()]
		if !ok || !toolLinkMatches(p.privateRoot, entry.Name(), link) {
			return errors.Join(resultErr, errors.New("private tool link cleanup member changed"))
		}
	}
	if p.guard.tx.journalStarted {
		if err := p.guard.tx.journalIntent("tool-private-cleanup", p.private); err != nil {
			return errors.Join(resultErr, err)
		}
	}
	for _, entry := range entries {
		if err := p.privateRoot.Remove(entry.Name()); err != nil {
			return errors.Join(resultErr, err)
		}
	}
	if err := p.guard.bin.verify(); err != nil {
		return errors.Join(resultErr, err)
	}
	if err := p.guard.bin.root.Remove(filepath.Base(p.private)); err != nil {
		return errors.Join(resultErr, err)
	}
	delete(p.guard.tx.directories, p.private)
	if p.guard.tx.journalStarted {
		if err := p.guard.tx.appendJournal(struct{ ToolPrivateRemoved string }{p.private}); err != nil {
			return errors.Join(resultErr, err)
		}
	}
	if p.guard.work != nil {
		resultErr = errors.Join(resultErr, p.guard.work.Close())
	}
	return errors.Join(resultErr, syncDirectory(p.guard.binRoot))
}
