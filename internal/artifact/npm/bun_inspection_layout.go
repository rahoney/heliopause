package npm

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errBunInspectionLayout = errors.New("bun hoisted layout differs from its frozen archives, instances or bins")

// BunInspectionLayout binds measured static inputs. Lifecycle/dynamic coverage,
// Policy and publication are separate consumers; zero values grant no execution.
type BunInspectionLayout struct {
	input    BunInspectionInput
	layout   BunLayoutArchive
	targets  map[npmLayoutTarget]string
	digest   domain.ContentDigest
	manifest []byte
}

func bunInstanceRoot(key string) (string, error) {
	parts, ok := bunKeyParts(key)
	if !ok {
		return "", errBunInspectionLayout
	}
	return "node_modules/" + strings.Join(parts, "/node_modules/"), nil
}

// BindLayout requires the whole fixed copyfile/hoisted tree. A cache hit or
// native name/version check cannot substitute for these source byte checks.
func (i BunInspectionInput) BindLayout(ctx context.Context, inventories []ArchiveInventory, layout BunLayoutArchive) (BunInspectionLayout, error) {
	return i.bindLayoutForOwner(ctx, inventories, layout, 1000, 1000)
}

func (i BunInspectionInput) bindLayoutForOwner(ctx context.Context, inventories []ArchiveInventory, layout BunLayoutArchive, uid, gid int) (BunInspectionLayout, error) {
	if ctx == nil {
		return BunInspectionLayout{}, errBunInspectionLayout
	}
	if err := ctx.Err(); err != nil {
		return BunInspectionLayout{}, errors.Join(errBunInspectionLayout, err, context.Cause(ctx))
	}
	if err := i.CheckWorkspace(inventories); err != nil {
		return BunInspectionLayout{}, err
	}
	bySubject, err := i.indexInventories(inventories)
	if err != nil {
		return BunInspectionLayout{}, err
	}
	byPackage := map[string]ArchiveInventory{}
	for subject, inventory := range bySubject {
		byPackage[subject.Identity().Name()+"@"+subject.Identity().Version()] = inventory
	}
	packages := map[string]BunLockedPackage{}
	targets := map[npmLayoutTarget]string{}
	for _, pkg := range i.context.selection.lock.packages {
		packages[pkg.Key] = pkg
	}
	placements := make([]npmHoistedPlacement, 0, len(packages))
	for _, pkg := range i.context.selection.lock.packages {
		if err := ctx.Err(); err != nil {
			return BunInspectionLayout{}, errors.Join(errBunInspectionLayout, err, context.Cause(ctx))
		}
		inventory, present := byPackage[pkg.Name+"@"+pkg.Version]
		root, err := bunInstanceRoot(pkg.Key)
		if !present || err != nil {
			return BunInspectionLayout{}, errBunInspectionLayout
		}
		bins, err := bunArchiveMetadata(pkg, inventory)
		if err != nil {
			return BunInspectionLayout{}, err
		}
		targets[npmLayoutTarget{inventory.artifact, pkg.Key}] = root
		placements = append(placements, npmHoistedPlacement{root, inventory, bins})
		for name, target := range pkg.Targets {
			resolved, exists := resolveBunTreeDependency(pkg.Key, name, packages)
			if !exists || resolved.Key != target {
				return BunInspectionLayout{}, errBunInspectionLayout
			}
		}
	}
	if err := validateNPMHoistedLayout(ctx, placements, layout.native, uid, gid); err != nil {
		return BunInspectionLayout{}, errors.Join(errBunInspectionLayout, err)
	}
	manifest, err := bunInspectionManifest(i.context.selection.lock)
	if err != nil {
		return BunInspectionLayout{}, err
	}
	digest, err := npmInspectionLayoutDigest("bun-inspection-layout/v1", i.context.bindingDigest(), i.context.runtime, i.artifacts, layout.Entries(), manifest)
	if err != nil {
		return BunInspectionLayout{}, err
	}
	if err := ctx.Err(); err != nil {
		return BunInspectionLayout{}, errors.Join(errBunInspectionLayout, err, context.Cause(ctx))
	}
	return BunInspectionLayout{i, layout, targets, digest, manifest}, nil
}

func bunArchiveMetadata(pkg BunLockedPackage, inventory ArchiveInventory) (map[string]string, error) {
	fields, err := bunJSONObject(inventory.manifest)
	if err != nil || !supportedBunSourceMetadata(fields) {
		return nil, errBunInspectionLayout
	}
	dependencies, err := bunOptionalConstraints(fields, "dependencies")
	if err != nil || !maps.Equal(dependencies, pkg.Dependencies) {
		return nil, errBunInspectionLayout
	}
	peers, err := bunOptionalConstraints(fields, "peerDependencies")
	if err != nil || !maps.Equal(peers, pkg.PeerDependencies) {
		return nil, errBunInspectionLayout
	}
	bins, err := pnpmInventoryBins(inventory)
	if err != nil || !maps.Equal(bins, pkg.Bins) {
		return nil, errBunInspectionLayout
	}
	return bins, nil
}

func (l BunInspectionLayout) Target(artifact domain.AcquiredArtifact, instance string) (string, error) {
	value, present := l.targets[npmLayoutTarget{artifact, instance}]
	if !present || l.digest.String() == "" {
		return "", errBunInspectionLayout
	}
	return value, nil
}

func (l BunInspectionLayout) MatchesInput(input BunInspectionInput) bool {
	return l.digest.String() != "" && l.input.Valid() && input.Valid() &&
		l.input.context.matchesContext(input.context) && l.input.context.RuntimeIdentity() == input.context.RuntimeIdentity() && slices.Equal(l.input.artifacts, input.artifacts)
}

func (l BunInspectionLayout) Digest() domain.ContentDigest { return l.digest }
func (l BunInspectionLayout) RuntimeIdentity() string      { return l.input.Context().RuntimeIdentity() }
func (l BunInspectionLayout) InspectionManifest() []byte   { return append([]byte(nil), l.manifest...) }
func (l BunInspectionLayout) Entries() []BunLayoutEntry    { return l.layout.Entries() }
func (l BunInspectionLayout) ReadFile(name string) ([]byte, error) {
	if l.digest.String() == "" {
		return nil, errBunInspectionLayout
	}
	return l.layout.ReadFile(name)
}
