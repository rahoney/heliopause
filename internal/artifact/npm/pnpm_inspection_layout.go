package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errPNPMInspectionLayout = errors.New("pnpm inspection layout differs from its frozen archives, instances or generated metadata")

type pnpmLayoutTarget struct {
	artifact domain.AcquiredArtifact
	instance string
}

// PNPMInspectionLayout binds measured static inputs only. Dynamic observation,
// Policy, promotion and Host metadata publication remain separate. Its zero
// value has no targets, files or digest and authorizes no execution.
type PNPMInspectionLayout struct {
	input    PNPMInspectionInput
	layout   PNPMLayoutArchive
	targets  map[pnpmLayoutTarget]string
	digest   domain.ContentDigest
	manifest []byte
}

type pnpmLayoutRequirement struct {
	kind       byte
	size       int64
	digest     domain.ContentDigest
	executable bool
	link, bin  string
	metadata   bool
}

// BindLayout compares every native entry with the complete frozen input.
// Inventories must come from the trusted static producer's actual byte reads.
// expectedStoreRoot is the caller-owned inspection guest path, never a native
// claim or a Host store/promotion destination.
func (i PNPMInspectionInput) BindLayout(ctx context.Context, inventories []ArchiveInventory, layout PNPMLayoutArchive, expectedStoreRoot string) (PNPMInspectionLayout, error) {
	if ctx == nil {
		return PNPMInspectionLayout{}, errPNPMInspectionLayout
	}
	if err := ctx.Err(); err != nil {
		return PNPMInspectionLayout{}, errors.Join(errPNPMInspectionLayout, err, context.Cause(ctx))
	}
	if !i.Valid() || len(i.artifacts) == 0 || len(inventories) != len(i.artifacts) || len(layout.entries) == 0 {
		return PNPMInspectionLayout{}, errPNPMInspectionLayout
	}
	bySubject := make(map[domain.AcquiredArtifact]ArchiveInventory, len(inventories))
	for _, inventory := range inventories {
		if !inventory.Valid() || !slices.Contains(i.artifacts, inventory.artifact) {
			return PNPMInspectionLayout{}, errPNPMInspectionLayout
		}
		if _, duplicate := bySubject[inventory.artifact]; duplicate {
			return PNPMInspectionLayout{}, errPNPMInspectionLayout
		}
		bySubject[inventory.artifact] = inventory
	}
	modules, err := parsePNPMLayoutModules(i, layout, expectedStoreRoot)
	if err != nil || checkPNPMVirtualLock(i, layout) != nil {
		return PNPMInspectionLayout{}, errPNPMInspectionLayout
	}
	packages := map[string]PNPMLockedPackage{}
	for _, pkg := range i.context.selection.lock.packages {
		packages[pkg.Key] = pkg
	}
	inventoryByPackage := map[string]ArchiveInventory{}
	for _, artifact := range i.artifacts {
		inventory, present := bySubject[artifact]
		if !present {
			return PNPMInspectionLayout{}, errPNPMInspectionLayout
		}
		inventoryByPackage[artifact.Identity().Name()+"@"+artifact.Identity().Version()] = inventory
	}
	wanted := map[string]pnpmLayoutRequirement{}
	add := func(name string, requirement pnpmLayoutRequirement) error {
		if !validInventoryPath(name) || name != "node_modules" && !strings.HasPrefix(name, "node_modules/") {
			return errPNPMInspectionLayout
		}
		if previous, duplicate := wanted[name]; duplicate && previous != requirement {
			return errPNPMInspectionLayout
		}
		wanted[name] = requirement
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if previous, present := wanted[parent]; present && previous.kind != tar.TypeDir {
				return errPNPMInspectionLayout
			}
			wanted[parent] = pnpmLayoutRequirement{kind: tar.TypeDir}
		}
		if len(wanted) > pnpmLayoutEntryLimit {
			return errPNPMInspectionLayout
		}
		return nil
	}
	roots := map[string]string{}
	moduleRoots := map[string]string{}
	bins := map[string]map[string]string{}
	targets := map[pnpmLayoutTarget]string{}
	physical := map[string]bool{}
	for _, instance := range i.context.selection.lock.snapshots {
		if err := ctx.Err(); err != nil {
			return PNPMInspectionLayout{}, errors.Join(errPNPMInspectionLayout, err, context.Cause(ctx))
		}
		pkg, present := packages[instance.Package]
		inventory, acquired := inventoryByPackage[instance.Package]
		directory, err := PNPMInstanceDirectory(instance.Key)
		if !present || !acquired || err != nil || physical[directory] {
			return PNPMInspectionLayout{}, errPNPMInspectionLayout
		}
		physical[directory] = true
		root := "node_modules/.pnpm/" + directory + "/node_modules/" + pkg.Name
		roots[instance.Key] = root
		moduleRoots[instance.Key] = "node_modules/.pnpm/" + directory + "/node_modules"
		targets[pnpmLayoutTarget{inventory.artifact, instance.Key}] = root
		declaredBins, err := pnpmInventoryBins(inventory)
		if err != nil {
			return PNPMInspectionLayout{}, err
		}
		bins[instance.Key] = declaredBins
		binFiles := map[string]bool{}
		for _, file := range declaredBins {
			binFiles[file] = true
		}
		for _, file := range inventory.entries {
			requirement := pnpmLayoutRequirement{kind: tar.TypeReg, size: file.Size, digest: file.Digest, executable: file.Mode&0o111 != 0 || binFiles[file.Path]}
			if file.Directory {
				requirement = pnpmLayoutRequirement{kind: tar.TypeDir}
			}
			if err := add(root+"/"+file.Path, requirement); err != nil {
				return PNPMInspectionLayout{}, err
			}
		}
	}
	addLink := func(name, target string) error {
		if target == "" {
			return errPNPMInspectionLayout
		}
		return add(name, pnpmLayoutRequirement{kind: tar.TypeSymlink, link: pnpmBinRelativePath(path.Dir(name), target)})
	}
	addBins := func(modulesDir, instance string) error {
		for command, file := range bins[instance] {
			target := roots[instance] + "/" + file
			if err := add(modulesDir+"/.bin/"+command, pnpmLayoutRequirement{kind: tar.TypeReg, executable: true, bin: target}); err != nil {
				return err
			}
		}
		return nil
	}
	for _, instance := range i.context.selection.lock.snapshots {
		root := roots[instance.Key]
		if err := addBins(root+"/node_modules", instance.Key); err != nil {
			return PNPMInspectionLayout{}, err
		}
		for name, dependency := range instance.Dependencies {
			if err := addLink(moduleRoots[instance.Key]+"/"+name, roots[dependency]); err != nil {
				return PNPMInspectionLayout{}, err
			}
			if err := addBins(root+"/node_modules", dependency); err != nil {
				return PNPMInspectionLayout{}, err
			}
		}
	}
	for _, root := range i.context.selection.lock.roots {
		if err := addLink("node_modules/"+root.Name, roots[root.Snapshot]); err != nil {
			return PNPMInspectionLayout{}, err
		}
		if err := addBins("node_modules", root.Snapshot); err != nil {
			return PNPMInspectionLayout{}, err
		}
	}
	for key, names := range modules.HoistedDependencies {
		for name := range names {
			if err := addLink("node_modules/.pnpm/node_modules/"+name, roots[key]); err != nil {
				return PNPMInspectionLayout{}, err
			}
			if err := addBins("node_modules/.pnpm/node_modules", key); err != nil {
				return PNPMInspectionLayout{}, err
			}
		}
	}
	for _, name := range []string{"node_modules/.modules.yaml", "node_modules/.pnpm/lock.yaml", pnpmWorkspaceCachePath} {
		if err := add(name, pnpmLayoutRequirement{kind: tar.TypeReg, metadata: true}); err != nil {
			return PNPMInspectionLayout{}, err
		}
	}
	// Empty native build/bin directories are harmless only in these exact
	// generated locations. Their contents still require complete coverage.
	optionalDirectories := map[string]bool{"node_modules/.bin": true, "node_modules/.pnpm/node_modules": true, "node_modules/.pnpm/node_modules/.bin": true}
	for _, root := range roots {
		optionalDirectories[root+"/node_modules"] = true
		optionalDirectories[root+"/node_modules/.bin"] = true
	}
	for _, actual := range layout.entries {
		if actual.Kind == tar.TypeDir && optionalDirectories[actual.Name] {
			if err := add(actual.Name, pnpmLayoutRequirement{kind: tar.TypeDir}); err != nil {
				return PNPMInspectionLayout{}, err
			}
		}
	}
	if len(wanted) != len(layout.entries) {
		return PNPMInspectionLayout{}, errPNPMInspectionLayout
	}
	directories := map[string]bool{}
	for name, requirement := range wanted {
		if requirement.kind == tar.TypeDir {
			directories[name] = true
		}
	}
	for _, actual := range layout.entries {
		if err := ctx.Err(); err != nil {
			return PNPMInspectionLayout{}, errors.Join(errPNPMInspectionLayout, err, context.Cause(ctx))
		}
		requirement, present := wanted[actual.Name]
		if !present || !pnpmInspectionEntryMode(actual, requirement) {
			return PNPMInspectionLayout{}, errPNPMInspectionLayout
		}
		if requirement.kind == tar.TypeDir {
			if actual.Kind != tar.TypeDir {
				return PNPMInspectionLayout{}, errPNPMInspectionLayout
			}
			continue
		}
		if requirement.kind == tar.TypeSymlink {
			if actual.Kind != tar.TypeSymlink || actual.Link != requirement.link {
				return PNPMInspectionLayout{}, errPNPMInspectionLayout
			}
			continue
		}
		file, regular := layout.files[actual.Name]
		if !regular || actual.Kind != tar.TypeReg && actual.Kind != tar.TypeLink {
			return PNPMInspectionLayout{}, errPNPMInspectionLayout
		}
		if requirement.metadata {
			if actual.Kind != tar.TypeReg {
				return PNPMInspectionLayout{}, errPNPMInspectionLayout
			}
			continue
		}
		if requirement.bin != "" {
			if err := checkPNPMBoundBin(layout, actual.Name, requirement.bin, directories); err != nil {
				return PNPMInspectionLayout{}, err
			}
			continue
		}
		if int64(file.end-file.start) != requirement.size || actual.Digest != requirement.digest {
			return PNPMInspectionLayout{}, errPNPMInspectionLayout
		}
	}
	// pnpm's freshness cache is neither inspection input nor execution data.
	// Account for its exact native entry, then remove it from the bound payload.
	layout, err = discardPNPMWorkspaceCache(layout)
	if err != nil {
		return PNPMInspectionLayout{}, err
	}
	manifest, err := pnpmInspectionManifest(i.context.selection.lock)
	if err != nil {
		return PNPMInspectionLayout{}, err
	}
	digest, err := pnpmInspectionLayoutDigest(i, layout, modules, manifest)
	if err != nil {
		return PNPMInspectionLayout{}, err
	}
	if err := ctx.Err(); err != nil {
		return PNPMInspectionLayout{}, errors.Join(errPNPMInspectionLayout, err, context.Cause(ctx))
	}
	return PNPMInspectionLayout{i, layout, targets, digest, manifest}, nil
}

func pnpmInspectionEntryMode(actual PNPMLayoutEntry, wanted pnpmLayoutRequirement) bool {
	if actual.UID != 1000 || actual.GID != 1000 || actual.Mode < 0 || actual.Mode > 0o777 {
		return false
	}
	if wanted.kind == tar.TypeSymlink {
		return actual.Mode == 0o777
	}
	if actual.Mode&0o022 != 0 || actual.Mode&0o600 != 0o600 {
		return false
	}
	if wanted.kind == tar.TypeDir {
		return actual.Mode&0o700 == 0o700
	}
	if wanted.executable {
		return actual.Mode&0o100 != 0
	}
	return actual.Mode&0o111 == 0
}

func checkPNPMBoundBin(layout PNPMLayoutArchive, shim, target string, directories map[string]bool) error {
	file, present := layout.files[target]
	if !present {
		return errPNPMInspectionLayout
	}
	content := layout.body[file.start:file.end]
	if !pnpmNodeBinTarget(content, target) {
		return errPNPMInspectionLayout
	}
	code, err := layout.ReadFile(shim)
	if err != nil {
		return errPNPMInspectionLayout
	}
	modules, err := ParsePNPMNodeBinShim(code, shim, target)
	if err != nil {
		return errPNPMInspectionLayout
	}
	owner := path.Dir(target)
	// The bin may be nested below its package root. A module search entry is
	// allowed only in the target's actual ancestry or the consumer's modules.
	allowed := map[string]bool{path.Dir(path.Dir(shim)): true}
	for strings.HasPrefix(owner, "node_modules/") {
		if path.Base(owner) == "node_modules" {
			allowed[owner] = true
		} else {
			allowed[owner+"/node_modules"] = true
		}
		owner = path.Dir(owner)
	}
	allowed["node_modules"] = true
	for _, module := range modules {
		if !directories[module] || !allowed[module] {
			return errPNPMInspectionLayout
		}
	}
	return nil
}

func pnpmNodeBinTarget(body []byte, name string) bool {
	if bytes.HasPrefix(body, []byte("#!")) {
		head := body[:min(len(body), 512)]
		line, _, _ := bytes.Cut(head, []byte{'\n'})
		fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(string(line), "#!")))
		return slices.Equal(fields, []string{"/usr/bin/env", "node"}) || slices.Equal(fields, []string{"/usr/bin/env", "-S", "node"}) ||
			slices.Equal(fields, []string{"/usr/local/bin/node"}) || slices.Equal(fields, []string{"/usr/bin/node"})
	}
	return slices.Contains([]string{".js", ".cjs", ".mjs"}, path.Ext(name))
}

// Target returns a relative inspection locator for one exact subject/instance.
func (l PNPMInspectionLayout) Target(artifact domain.AcquiredArtifact, instance string) (string, error) {
	target, present := l.targets[pnpmLayoutTarget{artifact, instance}]
	if !present || l.digest.String() == "" {
		return "", errPNPMInspectionLayout
	}
	return target, nil
}

func (l PNPMInspectionLayout) Digest() domain.ContentDigest { return l.digest }

// InspectionManifest returns only private, frozen dependency data. User root
// scripts/config/metadata stay in the controller-owned original controls.
func (l PNPMInspectionLayout) InspectionManifest() []byte { return bytes.Clone(l.manifest) }

func (l PNPMInspectionLayout) Entries() []PNPMLayoutEntry { return l.layout.Entries() }
func (l PNPMInspectionLayout) ReadFile(name string) ([]byte, error) {
	if l.digest.String() == "" {
		return nil, errPNPMInspectionLayout
	}
	return l.layout.ReadFile(name)
}
