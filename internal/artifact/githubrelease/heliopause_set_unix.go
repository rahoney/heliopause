//go:build linux || darwin

package githubrelease

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const heliopauseManifestName = "helox-release-manifest.json"

// HeliopauseReleaseSet freezes one platform's complete release declarations.
// It emits candidates only: provenance, native inspection and Policy remain
// mandatory in the existing install workflow. No root mutation occurs here.
type HeliopauseReleaseSet struct {
	mu              sync.RWMutex
	replayMu        sync.Mutex
	replayed        map[string]replayedReleaseRun
	client          *Client
	reference       domain.ArtifactReference
	install         domain.InstallContext
	lockSHA256      string
	root            *os.Root
	rootInfo        os.FileInfo
	owned           map[string]os.FileInfo
	view            *FrozenIntake
	manifest        releaseinstall.ReleaseManifest
	bundles         [][]byte
	entries         map[string]FrozenEntry
	resolution      domain.DependencyResolution
	started, closed bool
	closeErr        error
}

func NewHeliopauseReleaseSet(client *Client, version, activeLockSHA256 string, install domain.InstallContext) (*HeliopauseReleaseSet, error) {
	if client == nil || !client.production || !install.Valid() || install.Mode() != domain.InstallGitHubAsset {
		return nil, errors.New("native release set requires a production source and managed target")
	}
	if _, err := domain.NewSHA256Digest(activeLockSHA256); err != nil {
		return nil, errors.New("active runtime lock identity is invalid")
	}
	if err := ValidateHeliopauseVersion(version); err != nil {
		return nil, err
	}
	reference, err := ParseReference("rahoney/heliopause@" + version + "#" + heliopauseManifestName)
	if err != nil {
		return nil, errors.New("an exact official release version is required")
	}
	return &HeliopauseReleaseSet{client: client, reference: reference, install: install, lockSHA256: activeLockSHA256, owned: map[string]os.FileInfo{}, entries: map[string]FrozenEntry{}}, nil
}

func (s *HeliopauseReleaseSet) Reference() domain.ArtifactReference {
	if s == nil {
		return domain.ArtifactReference{}
	}
	return s.reference
}

func (s *HeliopauseReleaseSet) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (resolution domain.DependencyResolution, resultErr error) {
	if s == nil || s.client == nil || ctx == nil {
		return resolution, errors.New("native release set request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return resolution, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || reference != s.reference || install != s.install {
		return resolution, errors.New("native release selection or original target changed")
	}
	if s.started {
		return s.resolution, nil
	}
	s.started = true
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, ctx.Err(), s.closeLocked())
		}
	}()
	if err := s.openRoot(); err != nil {
		return resolution, err
	}
	snapshot, err := s.client.FreezeRelease(ctx, reference)
	if err != nil {
		return resolution, err
	}
	if !snapshot.Immutable() {
		return resolution, errors.New("native update requires an immutable published release")
	}
	manifestRef, manifestResolved, err := snapshot.ResolveAsset(heliopauseManifestName)
	if err != nil {
		return resolution, err
	}
	_, manifestSize, err := SelectionForResolved(manifestResolved)
	if err != nil || manifestSize > 1<<20 {
		return resolution, errors.New("release manifest size is invalid or excessive")
	}
	manifestEntry, err := s.acquireOriginal(ctx, manifestRef, manifestResolved)
	if err != nil {
		return resolution, err
	}
	body, err := s.readManifest(ctx, manifestEntry)
	if err != nil {
		return resolution, err
	}
	metadata, err := releaseinstall.ParseReleaseManifest(body)
	if err != nil {
		return resolution, err
	}
	if metadata.Version() != manifestResolved.Identity().Version() || metadata.RuntimeLockSHA256() != s.lockSHA256 || metadata.GVisorRelease() != runtimeidentity.GVisorRelease || metadata.GVisorCommit() != runtimeidentity.GVisorCommit {
		return resolution, errors.New("release version or runtime lock differs from the active installation")
	}
	selected, err := metadata.NativeUpdatePayloads()
	if err != nil {
		return resolution, err
	}
	entries := []FrozenEntry{manifestEntry}
	var total uint64 = manifestSize
	for _, file := range selected {
		ref, resolved, err := snapshot.ResolveAsset(file.Name)
		if err != nil {
			return resolution, err
		}
		_, size, err := SelectionForResolved(resolved)
		if err != nil || size != uint64(file.Size) || resolved.DeclaredIntegrity() != "sha256:"+file.SHA256 || size > 1<<30-total {
			return resolution, errors.New("frozen release asset differs from its manifest or set byte limit")
		}
		total += size
		entries = append(entries, FrozenEntry{Reference: ref, Resolved: resolved})
	}
	proof, err := snapshot.Attestations(ctx, heliopauseManifestName)
	if err != nil {
		return resolution, err
	}
	for index := 1; index < len(entries); index++ {
		entry, err := s.acquireOriginal(ctx, entries[index].Reference, entries[index].Resolved)
		if err != nil {
			return resolution, err
		}
		entries[index] = entry
	}
	view, err := NewFrozenIntake(ctx, s.client.intakeRoot, entries)
	if err != nil {
		return resolution, err
	}
	s.view = view
	resolution, err = heliopauseDependencyGraph(entries, manifestEntry.Acquired.Digest())
	if err != nil {
		return resolution, err
	}
	s.manifest, s.bundles, s.resolution = metadata, proof, resolution
	for _, entry := range entries {
		s.entries[entry.Resolved.Identity().Variant()] = entry
	}
	return resolution, nil
}

func (s *HeliopauseReleaseSet) openRoot() error {
	name := s.client.intakeRoot
	if err := os.MkdirAll(name, 0700); err != nil {
		return errors.New("release intake root is unavailable")
	}
	canonical, err := filepath.EvalSymlinks(name)
	info, statErr := os.Lstat(name)
	if err != nil || canonical != name || statErr != nil || !privateIntakeDirectory(info) {
		return errors.New("release intake root is untrusted")
	}
	root, err := os.OpenRoot(name)
	if err != nil {
		return errors.New("release intake root is unavailable")
	}
	s.root, s.rootInfo = root, info
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		return errors.New("release intake root changed")
	}
	return nil
}

func (s *HeliopauseReleaseSet) acquireOriginal(ctx context.Context, reference domain.ArtifactReference, resolved domain.ResolvedArtifact) (FrozenEntry, error) {
	current, err := os.Lstat(s.client.intakeRoot)
	if err != nil || !privateIntakeDirectory(current) || !os.SameFile(s.rootInfo, current) {
		return FrozenEntry{}, errors.New("release intake root changed")
	}
	run, err := domain.NewRunID()
	if err != nil {
		return FrozenEntry{}, err
	}
	acquired, err := s.client.Acquire(ctx, run, resolved)
	if err != nil {
		return FrozenEntry{}, err
	}
	info, err := s.root.Lstat(run.String())
	if err != nil || !privateIntakeDirectory(info) {
		return FrozenEntry{}, errors.New("acquired release Run identity is unavailable")
	}
	s.owned[run.String()] = info
	return FrozenEntry{Reference: reference, Resolved: resolved, Acquired: acquired}, nil
}

func (s *HeliopauseReleaseSet) readManifest(ctx context.Context, entry FrozenEntry) (body []byte, resultErr error) {
	file, err := openFrozenArtifact(s.root, entry.Acquired)
	if err != nil {
		return nil, err
	}
	defer func() {
		if file.Close() != nil {
			resultErr = errors.Join(resultErr, errors.New("close release manifest input failed"))
		}
	}()
	body, err = io.ReadAll(frozenContextReader{ctx: ctx, reader: io.NewSectionReader(file, 0, 1<<20+1)})
	if err != nil {
		return nil, errors.Join(errors.New("read release manifest input failed"), ctx.Err())
	}
	sum := sha256.Sum256(body)
	if len(body) > 1<<20 || uint64(len(body)) != entry.Acquired.SizeBytes() || hex.EncodeToString(sum[:]) != entry.Acquired.Digest().String() {
		return nil, errors.New("release manifest input changed")
	}
	return body, nil
}

func heliopauseDependencyGraph(entries []FrozenEntry, digest domain.ContentDigest) (domain.DependencyResolution, error) {
	nodes := make([]domain.LockedDependency, 0, len(entries))
	edges := make([]domain.DependencyEdge, 0, len(entries)-1)
	var primary domain.DependencyNodeID
	for index, entry := range entries {
		name := entry.Resolved.Identity().Variant()
		sum := sha256.Sum256([]byte(name))
		id, err := domain.NewDependencyNodeID("release-" + hex.EncodeToString(sum[:12]))
		if err != nil {
			return domain.DependencyResolution{}, err
		}
		role := domain.DependencyTransitive
		if index == 0 {
			role = domain.DependencyPrimary
			primary = id
		}
		node, err := domain.NewLockedDependencyWithRecordPath(id, role, entry.Resolved, name, false)
		if err != nil {
			return domain.DependencyResolution{}, err
		}
		nodes = append(nodes, node)
		if index > 0 {
			edge, err := domain.NewDependencyEdge(primary, id)
			if err != nil {
				return domain.DependencyResolution{}, err
			}
			edges = append(edges, edge)
		}
	}
	graph, err := domain.NewLockedDependencyGraph(nodes, edges)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return domain.NewDependencyResolution(graph, "heliopause-frozen-release-api-"+APIVersion, digest)
}

func (s *HeliopauseReleaseSet) Resolve(ctx context.Context, reference domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	if s == nil {
		return domain.ResolvedArtifact{}, errors.New("native release set is unavailable")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed || s.view == nil {
		return domain.ResolvedArtifact{}, errors.New("native release set is unresolved or closed")
	}
	return s.view.Resolve(ctx, reference)
}

func (s *HeliopauseReleaseSet) Acquire(ctx context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if s == nil {
		return domain.AcquiredArtifact{}, errors.New("native release set is unavailable")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed || s.view == nil {
		return domain.AcquiredArtifact{}, errors.New("native release set is unresolved or closed")
	}
	artifact, err := s.view.Acquire(ctx, run, resolved)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	owned, err := s.captureReplayRun(run, artifact)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	s.replayMu.Lock()
	if s.replayed == nil {
		s.replayed = map[string]replayedReleaseRun{}
	}
	s.replayed[run.String()] = owned
	s.replayMu.Unlock()
	return artifact, nil
}

// ReleaseProof returns copied source facts only for a member of the held set.
// The Verification provider must authenticate them; source metadata is not ALLOW.
func (s *HeliopauseReleaseSet) ReleaseProof(artifact domain.AcquiredArtifact) (releaseinstall.ReleaseManifest, [][]byte, error) {
	if s == nil {
		return releaseinstall.ReleaseManifest{}, nil, errors.New("native release proof is unavailable")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, found := s.entries[artifact.Identity().Variant()]
	declared, ok := artifact.DeclaredIntegrity()
	if s.closed || s.view == nil || !found || artifact.Identity() != entry.Acquired.Identity() || artifact.Digest() != entry.Acquired.Digest() || artifact.SizeBytes() != entry.Acquired.SizeBytes() || !ok || declared != entry.Resolved.DeclaredIntegrity() {
		return releaseinstall.ReleaseManifest{}, nil, errors.New("artifact is absent from the frozen release proof set")
	}
	proof := make([][]byte, len(s.bundles))
	for index, body := range s.bundles {
		proof[index] = append([]byte(nil), body...)
	}
	return s.manifest, proof, nil
}

func (s *HeliopauseReleaseSet) Manifest() (releaseinstall.ReleaseManifest, error) {
	if s == nil {
		return releaseinstall.ReleaseManifest{}, errors.New("native release manifest is unavailable")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed || s.view == nil {
		return releaseinstall.ReleaseManifest{}, errors.New("native release manifest is unresolved or closed")
	}
	return s.manifest, nil
}

func (s *HeliopauseReleaseSet) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeLocked()
}

func (s *HeliopauseReleaseSet) closeLocked() error {
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	var failures []error
	if s.view != nil {
		failures = append(failures, s.view.Close())
	}
	if s.root != nil {
		names := make([]string, 0, len(s.owned))
		for name := range s.owned {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			info, err := s.root.Lstat(name)
			if err != nil || !privateIntakeDirectory(info) || !os.SameFile(s.owned[name], info) {
				failures = append(failures, errors.New("release acquisition cleanup identity is uncertain"))
				continue
			}
			if s.root.RemoveAll(name) != nil {
				failures = append(failures, errors.New("release acquisition cleanup failed"))
			}
		}
		if s.root.Close() != nil {
			failures = append(failures, errors.New("close release acquisition root failed"))
		}
	}
	s.bundles = nil
	s.closeErr = errors.Join(failures...)
	return s.closeErr
}
