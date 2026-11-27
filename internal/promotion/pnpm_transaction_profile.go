package promotion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type pnpmProjectProfile struct {
	owner          *approvedPNPMProjectGuard
	prepared       *PNPMPreparedProject
	retain         map[string]bool
	approval       pnpmProjectApproval
	statePublished bool
}

func (p *pnpmProjectProfile) guard() error          { return p.owner.verifyPinned() }
func (p *pnpmProjectProfile) verifyOriginal() error { return p.owner.VerifyUnchanged(p.owner.ctx) }
func (p *pnpmProjectProfile) phase(name string) error {
	if err := p.owner.ctx.Err(); err != nil {
		return errors.Join(err, context.Cause(p.owner.ctx))
	}
	if p.owner.owner.checkpoint != nil {
		return p.owner.owner.checkpoint(name)
	}
	return nil
}

func (p *pnpmProjectProfile) verifyMember(name, location string) error {
	if err := p.guard(); err != nil {
		return err
	}
	before, known := p.owner.members[name]
	if !known {
		return errors.New("pnpm early member inventory is incomplete")
	}
	if before == nil {
		if _, err := os.Lstat(location); !errors.Is(err, os.ErrNotExist) {
			return errors.New("pnpm absent original member appeared")
		}
		return nil
	}
	return before.verify(location)
}

func (p *pnpmProjectProfile) workspaceMember(name, location string) (*npmTransactionMember, error) {
	prefix := pnpmDiskMember(name)
	if location != filepath.Join(p.prepared.directory, prefix) {
		return nil, errors.New("pnpm selected member path differs from prepared workspace")
	}
	if err := p.prepared.verifyBoundary(); err != nil {
		return nil, err
	}
	if name == "package.json" || name == p.owner.owner.lockName() {
		if err := p.prepared.verifyEntry(p.owner.ctx, prefix); err != nil {
			return nil, err
		}
		entry := p.prepared.entries[prefix]
		device, inode, ok := pypiObjectIdentity(entry.info)
		if !ok {
			return nil, errors.New("pnpm prepared control identity is unavailable")
		}
		snapshot := pypiFileSnapshot{Digest: entry.digest, Size: entry.info.Size(), Mode: entry.info.Mode(), Device: device, Inode: inode, info: entry.info, readBound: entry.info.Size() + 1}
		return &npmTransactionMember{files: map[string]pypiFileSnapshot{".": snapshot}, directories: map[string]os.FileInfo{}}, nil
	}
	entries := map[string]pnpmPreparedEntry{}
	for key, entry := range p.prepared.entries {
		if key == prefix {
			entries["."] = entry
		} else if relative, ok := strings.CutPrefix(key, prefix+"/"); ok {
			entries[relative] = entry
		}
	}
	member := &pnpmOwnedMember{entries: entries, boundary: p.guard}
	if err := member.verify(p.owner.ctx, location); err != nil {
		return nil, err
	}
	if name == "node_modules" {
		return &npmTransactionMember{pnpm: member}, nil
	}
	// Metadata stays on the existing regular-only path; it cannot select the
	// native link-capable inventory merely by a filename or document field.
	regular, err := snapshotNPMTransactionMember(location)
	if err != nil {
		return nil, err
	}
	if err := member.verify(p.owner.ctx, location); err != nil {
		return nil, err
	}
	return regular, nil
}

func (p *pnpmProjectProfile) releaseWorkspaceMember(name string) {
	prefix := pnpmDiskMember(name)
	for key := range p.prepared.entries {
		if key == prefix || strings.HasPrefix(key, prefix+"/") {
			delete(p.prepared.entries, key)
		}
	}
}

type pnpmTargetMarker struct {
	Schema                                  int
	Operation, Manifest, Graph, CacheDigest string
}

func (p *pnpmProjectProfile) publishMarker(t *npmProjectTransaction) error {
	marker, err := json.Marshal(pnpmTargetMarker{1, p.approval.Approval.Operation, p.approval.Approval.Manifest, p.approval.Approval.Graph, p.approval.CacheDigest})
	if err != nil {
		return err
	}
	directory := t.target("haa")
	if err := trustedExistingDirectory(directory); err != nil {
		return err
	}
	if err := writeNPMTransactionMetadata(directory, filepath.Join(directory, p.owner.owner.markerName()), marker); err != nil {
		return err
	}
	snapshot, err := snapshotNPMFile(filepath.Join(directory, p.owner.owner.markerName()))
	if err != nil {
		return err
	}
	member := t.outputs["haa"]
	if member == nil || member.pnpm != nil {
		return errors.New("pnpm transaction metadata inventory is unavailable")
	}
	member.files[p.owner.owner.markerName()] = snapshot
	return syncDirectory(directory)
}

func npmPNPMMemberDigest(member *npmTransactionMember) (string, error) {
	if member == nil {
		return "", errors.New("pnpm approved member is unavailable")
	}
	if member.pnpm != nil {
		return pnpmMemberDigest(member.pnpm.entries)
	}
	entries := map[string]pnpmPreparedEntry{}
	for name, info := range member.directories {
		entries[name] = pnpmPreparedEntry{info: info}
	}
	for name, file := range member.files {
		entries[name] = pnpmPreparedEntry{info: file.info, digest: file.Digest}
	}
	return pnpmMemberDigest(entries)
}

func (p *pnpmProjectProfile) publishApproval(t *npmProjectTransaction) error {
	if err := p.phase("before-state"); err != nil {
		return err
	}
	p.approval.Members = nil
	for _, name := range t.memberNames() {
		if p.absentSelected(name) {
			if _, err := os.Lstat(t.target(name)); !errors.Is(err, os.ErrNotExist) {
				return errors.New("absent selected output appeared")
			}
		}
		digest, err := npmApprovedMemberDigest(p.owner.owner.client, name, t.outputs[name], p.approval.Approval)
		if err != nil {
			return err
		}
		p.approval.Members = append(p.approval.Members, pnpmMemberApproval{name, digest})
	}
	body, err := json.Marshal(p.approval)
	if err != nil || len(body) > pnpmCacheReceiptLimit {
		return errors.Join(errors.New("pnpm retained approval exceeds its fixed bound"), err)
	}
	if err := p.owner.writeState(body, &p.statePublished); err != nil {
		return err
	}
	return p.phase("after-state")
}

func (p *pnpmProjectProfile) rollbackApproval() error {
	if !p.statePublished {
		return nil
	}
	if err := p.owner.verifyPinned(); err != nil {
		return err
	}
	if len(p.owner.originalState) == 0 {
		if err := p.owner.state.Remove(p.owner.stateName); err != nil {
			return err
		}
		p.owner.currentState = nil
		p.owner.currentStateInfo = nil
		return syncPNPMDirectory(p.owner.state, ".", p.owner.stateInfo)
	}
	var restored bool
	return p.owner.writeState(p.owner.originalState, &restored)
}

func (g *approvedPNPMProjectGuard) writeState(body []byte, published *bool) (resultErr error) {
	if err := g.verifyPinned(); err != nil {
		return err
	}
	id, err := domain.NewRunID()
	if err != nil {
		return err
	}
	name := "." + g.owner.clientPrefix() + "-approval-" + id.String()
	f, err := g.state.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err := g.state.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, err)
		}
	}()
	n, writeErr := f.Write(body)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil || n != len(body) {
		return errors.Join(errors.New("persist pnpm independent approval"), writeErr, syncErr, closeErr)
	}
	if err := g.verifyPinned(); err != nil {
		return err
	}
	if g.currentStateInfo == nil {
		err = g.state.Link(name, g.stateName)
	} else {
		err = g.state.Rename(name, g.stateName)
	}
	if err != nil {
		return err
	}
	*published = true
	// Remove the initial publication's temporary hardlink before accepting a
	// single-link retained approval or making its directory entry durable.
	if err := g.state.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	info, err := g.state.Lstat(g.stateName)
	if err != nil {
		return err
	}
	g.currentState = bytes.Clone(body)
	g.currentStateInfo = info
	return syncPNPMDirectory(g.state, ".", g.stateInfo)
}
