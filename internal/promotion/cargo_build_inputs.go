package promotion

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

const maxCargoBuildSourceArchiveBytes = artifactcargo.MaxCrateExpandedBytes + (2*artifactcargo.MaxCrateFiles+2)*512
const maxCargoBuildCacheArchiveBytes = maxCargoProjectCacheBytes + (4*maxCargoProjectCacheFiles+2)*512

// OpenBuildInputs consumes retained independent approval, never a marker or a
// raw vendor pathname. A build must still reconcile the offline frozen graph.
func (g *approvedCargoProjectGuard) OpenBuildInputs(ctx context.Context) (domain.ProjectDependencySnapshot, string, error) {
	if g == nil || len(g.originalState) == 0 {
		return domain.ProjectDependencySnapshot{}, "", errors.New("cargo build requires complete independent project approval")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	if err := g.verifyRetainedApproval(ctx); err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	var doc cargoProjectApproval
	if json.Unmarshal(g.originalState, &doc) != nil {
		return domain.ProjectDependencySnapshot{}, "", errors.New("cargo build retained approval is invalid")
	}
	graph, err := domain.NewSHA256Digest(doc.Approval.Graph)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	controls := make([]domain.ProjectControlDigest, 0, 2)
	for _, control := range g.plan.controls {
		value, err := domain.NewProjectControlDigest(control.Name(), control.Digest())
		if err != nil {
			return domain.ProjectDependencySnapshot{}, "", err
		}
		controls = append(controls, value)
	}
	dependencies := make([]domain.ResolvedArtifact, 0, len(doc.Approval.Entries))
	for _, entry := range doc.Approval.Entries {
		identity, err := domain.NewResolvedArtifactIdentity(artifactcargo.Source(), entry.Crate, entry.Version, "crate")
		if err != nil {
			return domain.ProjectDependencySnapshot{}, "", err
		}
		endpoint, err := artifactcargo.DownloadURL(entry.Crate, entry.Version)
		if err != nil {
			return domain.ProjectDependencySnapshot{}, "", err
		}
		value, err := domain.NewResolvedArtifact(identity, endpoint, entry.Integrity)
		if err != nil {
			return domain.ProjectDependencySnapshot{}, "", err
		}
		dependencies = append(dependencies, value)
	}
	var snapshot domain.ProjectDependencySnapshot
	if doc.Approval.DependencyFree {
		snapshot, err = domain.NewDependencyFreeProjectSnapshot(g.context, artifactcargo.Source(), controls, graph)
	} else {
		snapshot, err = domain.NewProjectDependencySnapshot(g.context, artifactcargo.Source(), controls, dependencies, graph)
	}
	if err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	digest, err := domain.NewSHA256Digest(doc.CacheDigest)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	cache, err := g.owner.cache.openDocument(ctx, doc.Cache, digest, doc.Approval)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, "", err
	}
	return snapshot, cache, g.VerifyUnchanged(ctx)
}

func (g *approvedCargoProjectGuard) FreezeBuildInputs(ctx context.Context, run domain.RunID, selector string) (domain.ProjectBuildInputs, error) {
	if g == nil || g.buildInputs != nil || run.String() == "" || selector != "default" {
		return domain.ProjectBuildInputs{}, errors.New("cargo build inputs are invalid or already frozen")
	}
	snapshot, cachePath, err := g.OpenBuildInputs(ctx)
	if err != nil {
		return domain.ProjectBuildInputs{}, err
	}
	local, _ := domain.NewSourceID("project-local")
	project := sha256.Sum256([]byte(g.context.Target().String()))
	key := hex.EncodeToString(project[:])
	identity, err := domain.NewResolvedArtifactIdentity(local, "project-"+key, "snapshot", "cargo-source")
	if err != nil {
		return domain.ProjectBuildInputs{}, err
	}
	archive, err := g.source.BuildArchive()
	if err != nil {
		return domain.ProjectBuildInputs{}, err
	}
	source, err := freezeProjectBuildArchive(ctx, g.owner.cache.intakeRoot, identity, maxCargoBuildSourceArchiveBytes, func(w io.Writer) error { _, err := w.Write(archive); return err })
	if err != nil {
		return domain.ProjectBuildInputs{}, err
	}
	cache, err := g.snapshotBuildCache(ctx, snapshot, cachePath, key)
	if err != nil {
		return domain.ProjectBuildInputs{}, err
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return domain.ProjectBuildInputs{}, err
	}
	current, currentPath, err := g.OpenBuildInputs(ctx)
	if err != nil || currentPath != cachePath || !reflect.DeepEqual(current, snapshot) {
		return domain.ProjectBuildInputs{}, errors.New("cargo build approval changed while freezing inputs")
	}
	inputs, err := domain.NewProjectBuildInputs(snapshot, source, cache, run, selector)
	if err != nil {
		return domain.ProjectBuildInputs{}, err
	}
	g.buildInputs = &inputs
	return inputs, nil
}

func (g *approvedCargoProjectGuard) snapshotBuildCache(ctx context.Context, snapshot domain.ProjectDependencySnapshot, cachePath, key string) (result domain.AcquiredArtifact, resultErr error) {
	var approval cargoProjectApproval
	if json.Unmarshal(g.originalState, &approval) != nil {
		return result, errors.New("cargo cache approval is invalid")
	}
	parent, err := os.OpenRoot(filepath.Dir(cachePath))
	if err != nil {
		return result, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, parent.Close())
		if resultErr != nil {
			result = domain.AcquiredArtifact{}
		}
	}()
	body, _, err := readGoTransactionControl(parent, cargoCacheReceipt)
	if err != nil {
		return result, err
	}
	hash := sha256.Sum256(body)
	if hex.EncodeToString(hash[:]) != approval.CacheDigest {
		return result, errors.New("cargo cache receipt changed")
	}
	var doc cargoCacheDocument
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&doc) != nil {
		return result, errors.New("cargo cache receipt is invalid")
	}
	expected := approval.Approval
	expected.Files = doc.Files
	canonical, err := json.Marshal(expected)
	if err != nil || !bytes.Equal(body, canonical) {
		return result, errors.New("cargo cache inventory differs from approval")
	}
	before, err := parent.Lstat("vendor")
	if err != nil || before.Mode() != os.ModeDir|0o755 {
		return result, errors.New("cargo vendor root is invalid")
	}
	root, err := parent.OpenRoot("vendor")
	if err != nil {
		return result, err
	}
	roots := []*os.Root{root}
	defer func() {
		for i := len(roots) - 1; i >= 0; i-- {
			resultErr = errors.Join(resultErr, roots[i].Close())
		}
		if resultErr != nil {
			result = domain.AcquiredArtifact{}
		}
	}()
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		return result, errors.New("cargo vendor root identity changed")
	}
	local, _ := domain.NewSourceID("project-local")
	identity, err := domain.NewResolvedArtifactIdentity(local, "cache-"+key, snapshot.GraphDigest().String(), "cargo-cache")
	if err != nil {
		return result, err
	}
	return freezeProjectBuildArchive(ctx, g.owner.cache.intakeRoot, identity, maxCargoBuildCacheArchiveBytes, func(w io.Writer) error {
		archive := tar.NewWriter(w)
		copyErr := copyApprovedProjectBuildCache(ctx, root, doc.Files, archive, &roots, maxCargoProjectCacheFiles, maxCargoProjectCacheBytes)
		return errors.Join(copyErr, archive.Close())
	})
}

func (g *approvedCargoProjectGuard) VerifyBuildSource(ctx context.Context) error {
	if g == nil || g.buildInputs == nil {
		return errors.New("cargo build source was not frozen")
	}
	return g.VerifyUnchanged(ctx)
}
