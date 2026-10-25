package npm

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errBunInspectionLayout = errors.New("bun hoisted layout differs from its frozen archives, instances or bins")

type bunLayoutTarget struct {
	artifact domain.AcquiredArtifact
	instance string
}

type bunLayoutRequirement struct {
	kind       byte
	size, mode int64
	digest     domain.ContentDigest
	link       string
}

// BunInspectionLayout binds measured static inputs. Lifecycle/dynamic coverage,
// Policy and publication are separate consumers; zero values grant no execution.
type BunInspectionLayout struct {
	input    BunInspectionInput
	layout   BunLayoutArchive
	targets  map[bunLayoutTarget]string
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
	wanted := map[string]bunLayoutRequirement{}
	add := func(name string, requirement bunLayoutRequirement) error {
		if !validInventoryPath(name) || name != "node_modules" && !strings.HasPrefix(name, "node_modules/") {
			return errBunInspectionLayout
		}
		if previous, present := wanted[name]; present && previous != requirement {
			return errBunInspectionLayout
		}
		wanted[name] = requirement
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if previous, present := wanted[parent]; present && previous.kind != tar.TypeDir {
				return errBunInspectionLayout
			}
			wanted[parent] = bunLayoutRequirement{kind: tar.TypeDir}
		}
		if len(wanted) > pnpmLayoutEntryLimit {
			return errBunInspectionLayout
		}
		return nil
	}
	packages := map[string]BunLockedPackage{}
	targets := map[bunLayoutTarget]string{}
	for _, pkg := range i.context.selection.lock.packages {
		packages[pkg.Key] = pkg
	}
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
		targets[bunLayoutTarget{inventory.artifact, pkg.Key}] = root
		if err := add(root, bunLayoutRequirement{kind: tar.TypeDir}); err != nil {
			return BunInspectionLayout{}, err
		}
		for _, file := range inventory.entries {
			// The private cache envelope removes only group/other write bits.
			// Original packed modes remain in the measured source inventory.
			requirement := bunLayoutRequirement{kind: tar.TypeReg, size: file.Size, mode: file.Mode &^ 0o022, digest: file.Digest}
			if file.Directory {
				requirement = bunLayoutRequirement{kind: tar.TypeDir}
			}
			if err := add(root+"/"+file.Path, requirement); err != nil {
				return BunInspectionLayout{}, err
			}
		}
		for name, target := range pkg.Targets {
			resolved, exists := resolveBunTreeDependency(pkg.Key, name, packages)
			if !exists || resolved.Key != target {
				return BunInspectionLayout{}, errBunInspectionLayout
			}
		}
		modules := strings.TrimSuffix(root, "/"+pkg.Name)
		for command, file := range bins {
			bin := modules + "/.bin/" + command
			link := pnpmBinRelativePath(path.Dir(bin), root+"/"+file)
			if err := add(bin, bunLayoutRequirement{kind: tar.TypeSymlink, mode: 0o777, link: link}); err != nil {
				return BunInspectionLayout{}, err
			}
		}
	}
	entries := layout.Entries()
	if len(entries) != len(wanted) {
		return BunInspectionLayout{}, fmt.Errorf("%w: expected=%d observed=%d", errBunInspectionLayout, len(wanted), len(entries))
	}
	for _, actual := range entries {
		if err := ctx.Err(); err != nil {
			return BunInspectionLayout{}, errors.Join(errBunInspectionLayout, err, context.Cause(ctx))
		}
		expected, present := wanted[actual.Name]
		if !present || actual.Kind != expected.kind || actual.UID != uid || actual.GID != gid {
			return BunInspectionLayout{}, errBunInspectionLayout
		}
		switch expected.kind {
		case tar.TypeDir:
			if actual.Mode < 0 || actual.Mode > 0o777 || actual.Mode&0o022 != 0 || actual.Mode&0o700 != 0o700 {
				return BunInspectionLayout{}, errBunInspectionLayout
			}
		case tar.TypeSymlink:
			if actual.Mode != 0o777 || actual.Link != expected.link {
				return BunInspectionLayout{}, errBunInspectionLayout
			}
		case tar.TypeReg:
			if actual.Mode != expected.mode || actual.Mode&0o022 != 0 || actual.Mode&0o600 != 0o600 || actual.Size != expected.size || actual.Digest != expected.digest {
				return BunInspectionLayout{}, errBunInspectionLayout
			}
		default:
			return BunInspectionLayout{}, errBunInspectionLayout
		}
	}
	manifest, err := bunInspectionManifest(i.context.selection.lock)
	if err != nil {
		return BunInspectionLayout{}, err
	}
	digest, err := bunInspectionLayoutDigest(i, layout, manifest)
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
	value, present := l.targets[bunLayoutTarget{artifact, instance}]
	if !present || l.digest.String() == "" {
		return "", errBunInspectionLayout
	}
	return value, nil
}

func (l BunInspectionLayout) MatchesInput(input BunInspectionInput) bool {
	return l.digest.String() != "" && l.input.Valid() && input.Valid() &&
		l.input.context.MatchesSnapshot(input.context.Snapshot()) && l.input.context.RuntimeIdentity() == input.context.RuntimeIdentity() && slices.Equal(l.input.artifacts, input.artifacts)
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
