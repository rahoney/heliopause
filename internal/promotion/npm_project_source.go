package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// Only local project source is frozen here. Owned dependency/metadata and
// hidden credential namespaces are excluded before reading their contents.
func (g *approvedPNPMProjectGuard) collectNPMProjectSource(ctx context.Context) ([]byte, []projectSourceMember, error) {
	if g == nil || g.owner.client != npmProjectCacheNPM {
		return nil, nil, errors.New("npm local source requires its current project guard")
	}
	if err := g.verifyPayloadUnchanged(ctx); err != nil {
		return nil, nil, err
	}
	var controls []domain.ProjectControlFile
	for _, control := range g.controls {
		if control.Present() {
			controls = append(controls, control)
		}
	}
	limit := int64(artifactnpm.NPMProjectInspectionWorkspaceLimit)
	return collectProjectSource(ctx, g.guard.root, g.guard.rootInfo, controls, projectSourceLimits{files: artifactnpm.ArchiveEntryLimit, fileBytes: artifactnpm.ArchiveFileLimit, expandedBytes: limit, archiveBytes: limit, preserveExecutable: true}, map[string]bool{"node_modules": true})
}

func (g *approvedPNPMProjectGuard) FreezeNPMProjectSource(ctx context.Context) (domain.AcquiredArtifact, error) {
	if g == nil || g.rootSource != nil {
		return domain.AcquiredArtifact{}, errors.New("npm local source is unavailable or already frozen")
	}
	body, members, err := g.collectNPMProjectSource(ctx)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	local, _ := domain.NewSourceID("project-local")
	name := sha256.Sum256([]byte(g.install.Target().String()))
	identity, err := domain.NewResolvedArtifactIdentity(local, "project-"+hex.EncodeToString(name[:]), "snapshot", "npm-source")
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	subject, err := freezeProjectBuildArchive(ctx, g.owner.intakeRoot, identity, int64(artifactnpm.NPMProjectInspectionWorkspaceLimit), func(w io.Writer) error { _, err := w.Write(body); return err })
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	g.rootSource = &projectSourcePlan{artifact: subject, members: members}
	if err := g.VerifyNPMProjectSource(ctx, subject); err != nil {
		g.rootSource = nil
		return domain.AcquiredArtifact{}, err
	}
	return subject, nil
}

// Recheck the live source independently of the approved dependency tree. The
// execution reader must separately verify the persisted transport bytes.
func (g *approvedPNPMProjectGuard) VerifyNPMProjectSource(ctx context.Context, subject domain.AcquiredArtifact) error {
	if g == nil || g.rootSource == nil || g.rootSource.artifact != subject {
		return errors.New("npm local source differs from the current frozen subject")
	}
	body, members, err := g.collectNPMProjectSource(ctx)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != subject.Digest().String() || uint64(len(body)) != subject.SizeBytes() || len(members) != len(g.rootSource.members) {
		return errors.New("npm local source bytes or membership changed")
	}
	for i, current := range members {
		before := g.rootSource.members[i]
		if current.name != before.name || current.digest != before.digest || !os.SameFile(current.info, before.info) || current.info.Mode() != before.info.Mode() || !current.info.ModTime().Equal(before.info.ModTime()) {
			return errors.New("npm local source identity changed")
		}
	}
	return nil
}

// Freeze whether native npm has a local install action. If no source needs
// execution, the guard still retains binding.gyp absence until publication.
func (g *approvedPNPMProjectGuard) NPMProjectRootLifecycleRequired(ctx context.Context) (bool, error) {
	if g == nil || g.owner.client != npmProjectCacheNPM {
		return false, errors.New("npm local lifecycle requires its project guard")
	}
	if err := g.verifyPayloadUnchanged(ctx); err != nil {
		return false, err
	}
	if g.rootLifecycleChecked {
		return g.rootLifecycleRequired, nil
	}
	phases, err := artifactnpm.NPMProjectRootLifecycleScripts(g.controls[0].Body())
	if err != nil {
		return false, err
	}
	info, err := g.guard.root.Lstat("binding.gyp")
	exists := err == nil
	if exists && (!info.Mode().IsRegular() || !pypiSingleLink(info)) {
		return false, errors.New("npm implicit lifecycle input is aliased or unsupported")
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	g.rootLifecycleChecked = true
	g.rootLifecycleRequired = len(phases) != 0 || exists
	g.rootBindingGyp = exists
	return g.rootLifecycleRequired, g.verifyPayloadUnchanged(ctx)
}
