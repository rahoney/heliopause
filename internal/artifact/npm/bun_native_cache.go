package npm

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errBunNativeCache = errors.New("bun native cache differs from its complete measured source")

// BunNativeCacheEntry is measured preparation data only. Its marker grants no
// source approval, execution, Policy or publication authority.
type BunNativeCacheEntry struct {
	input     BunInspectionInput
	inventory ArchiveInventory
	native    PNPMLayoutArchive
	root      string
	bound     bool
}

// NativeCacheFolder derives a fixed native locator from an exact frozen subject.
// The truncated digest locates data; BindNativeCache checks the full inventory.
func (i BunInspectionInput) NativeCacheFolder(subject domain.AcquiredArtifact) (string, error) {
	if !i.Valid() || !slices.Contains(i.artifacts, subject) {
		return "", errBunNativeCache
	}
	sri, present := subject.DeclaredIntegrity()
	if !present || !validSHA512SRI(sri) {
		return "", errBunNativeCache
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(sri, "sha512-"))
	if err != nil || len(decoded) != 64 {
		return "", errBunNativeCache
	}
	return "@T@sha512-" + hex.EncodeToString(decoded[:16]) + "@@@1", nil
}

func bunRegistryCacheFolder(subject domain.AcquiredArtifact) (string, error) {
	identity := subject.Identity()
	if identity.Source().String() != "npm" || identity.Variant() != "tarball" || validatePackageName(identity.Name()) != nil ||
		!strictBunVersion(identity.Version()) || strings.ContainsAny(identity.Version(), "-+") {
		return "", errBunNativeCache
	}
	return identity.Name() + "@" + identity.Version() + "@@@1", nil
}

func bunNativeInventorySupported(inventory ArchiveInventory) error {
	if !inventory.Valid() {
		return errBunNativeCache
	}
	if _, err := bunRegistryCacheFolder(inventory.artifact); err != nil {
		return err
	}
	for _, entry := range inventory.entries {
		if entry.Path == ".bun-tag" || !entry.Directory && entry.Mode&^0o022&0o600 != 0o600 {
			return errBunNativeCache
		}
	}
	return nil
}

func (i BunInspectionInput) checkNativeCacheInventory(inventory ArchiveInventory) error {
	if err := bunNativeInventorySupported(inventory); err != nil {
		return err
	}
	for _, pkg := range i.context.selection.lock.packages {
		if pkg.Name == inventory.artifact.Identity().Name() && pkg.Version == inventory.artifact.Identity().Version() {
			_, err := bunArchiveMetadata(pkg, inventory)
			return err
		}
	}
	return errBunNativeCache
}

// CheckNativeCacheInputs rejects unsupported data before tool/DNS resources.
func (i BunInspectionInput) CheckNativeCacheInputs(inventories []ArchiveInventory) error {
	if err := i.CheckWorkspace(inventories); err != nil {
		return err
	}
	bySubject, err := i.indexInventories(inventories)
	if err != nil {
		return err
	}
	locators := map[string]domain.AcquiredArtifact{}
	for subject, inventory := range bySubject {
		if err := i.checkNativeCacheInventory(inventory); err != nil {
			return err
		}
		root, err := i.NativeCacheFolder(subject)
		if err != nil {
			return err
		}
		if old, present := locators[root]; present && old != subject {
			return errBunNativeCache
		}
		locators[root] = subject
	}
	return checkBunRegistryCachePaths(inventories)
}

// Charge implied directory structure as well as original tar entries before
// asking the native tool to create a potentially deep cache tree.
func checkBunRegistryCachePaths(inventories []ArchiveInventory) error {
	seen := map[string]byte{"cache": tar.TypeDir}
	for _, inventory := range inventories {
		folder, err := bunRegistryCacheFolder(inventory.artifact)
		if err != nil {
			return err
		}
		for _, entry := range inventory.entries {
			name := "cache/" + folder + "/" + entry.Path
			if !validInventoryPath(name) {
				return errBunNativeCache
			}
			kind := byte(tar.TypeReg)
			if entry.Directory {
				kind = tar.TypeDir
			}
			if old, present := seen[name]; present && old != kind {
				return errBunNativeCache
			}
			seen[name] = kind
			for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
				if old, present := seen[parent]; present && old != tar.TypeDir {
					return errBunNativeCache
				}
				seen[parent] = tar.TypeDir
				if len(seen) > pnpmLayoutEntryLimit {
					return errBunNativeCache
				}
			}
			if len(seen) > pnpmLayoutEntryLimit {
				return errBunNativeCache
			}
		}
	}
	return nil
}

// BindNativeCache reads bounded tar data without decompressing or extracting a
// source package on the Host. Native Bun already expanded it inside isolation.
func (i BunInspectionInput) BindNativeCache(ctx context.Context, inventory ArchiveInventory, body []byte) (BunNativeCacheEntry, error) {
	if ctx == nil {
		return BunNativeCacheEntry{}, errBunNativeCache
	}
	if err := ctx.Err(); err != nil {
		return BunNativeCacheEntry{}, errors.Join(errBunNativeCache, err, context.Cause(ctx))
	}
	root, err := i.NativeCacheFolder(inventory.artifact)
	if err != nil {
		return BunNativeCacheEntry{}, err
	}
	if err := i.checkNativeCacheInventory(inventory); err != nil {
		return BunNativeCacheEntry{}, err
	}
	native, err := parsePNPMTreeArchive(ctx, body, root)
	if err != nil {
		return BunNativeCacheEntry{}, errors.Join(errBunNativeCache, err)
	}
	wanted := map[string]bunLayoutRequirement{root: {kind: tar.TypeDir}}
	add := func(name string, requirement bunLayoutRequirement) error {
		if !validInventoryPath(name) {
			return errBunNativeCache
		}
		if old, present := wanted[name]; present && old != requirement {
			return errBunNativeCache
		}
		wanted[name] = requirement
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if old, present := wanted[parent]; present && old.kind != tar.TypeDir {
				return errBunNativeCache
			}
			wanted[parent] = bunLayoutRequirement{kind: tar.TypeDir}
		}
		return nil
	}
	for _, entry := range inventory.entries {
		requirement := bunLayoutRequirement{kind: tar.TypeReg, size: entry.Size, mode: entry.Mode &^ 0o022, digest: entry.Digest}
		if entry.Directory {
			requirement = bunLayoutRequirement{kind: tar.TypeDir}
		}
		if err := add(root+"/"+entry.Path, requirement); err != nil {
			return BunNativeCacheEntry{}, err
		}
	}
	sri, _ := inventory.artifact.DeclaredIntegrity()
	sum := sha256.Sum256([]byte(sri))
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return BunNativeCacheEntry{}, err
	}
	if err := add(root+"/.bun-tag", bunLayoutRequirement{kind: tar.TypeReg, size: int64(len(sri)), mode: 0o644, digest: digest}); err != nil {
		return BunNativeCacheEntry{}, err
	}
	if len(wanted) != len(native.entries) {
		return BunNativeCacheEntry{}, errBunNativeCache
	}
	for _, actual := range native.entries {
		if err := ctx.Err(); err != nil {
			return BunNativeCacheEntry{}, errors.Join(errBunNativeCache, err, context.Cause(ctx))
		}
		expected, present := wanted[actual.Name]
		if !present || actual.UID != 1000 || actual.GID != 1000 || actual.Kind != expected.kind {
			return BunNativeCacheEntry{}, errBunNativeCache
		}
		mode := actual.Mode &^ 0o022
		if expected.kind == tar.TypeDir {
			if mode&0o700 != 0o700 {
				return BunNativeCacheEntry{}, errBunNativeCache
			}
			continue
		}
		if mode != expected.mode || actual.Size != expected.size || actual.Digest != expected.digest {
			return BunNativeCacheEntry{}, errBunNativeCache
		}
	}
	return BunNativeCacheEntry{i, inventory, native, root, true}, nil
}

func (e BunNativeCacheEntry) matches(input BunInspectionInput) bool {
	return e.bound && input.Valid() && e.input.Valid() && e.input.context.MatchesSnapshot(input.context.Snapshot()) &&
		e.input.context.RuntimeIdentity() == input.context.RuntimeIdentity() && slices.Equal(e.input.artifacts, input.artifacts)
}
