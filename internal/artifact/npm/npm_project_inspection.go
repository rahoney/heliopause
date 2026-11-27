package npm

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errNPMProjectInspection = errors.New("npm complete inspection input differs from its exact source binding")

// NPMProjectInspectionContext binds current controls/runtime and every native
// placement beside a source-neutral snapshot. It authorizes no artifact code.
type NPMProjectInspectionContext struct {
	selection NPMProjectSelection
	snapshot  domain.ProjectDependencySnapshot
	controls  []domain.ProjectControlFile
	runtime   string
}

func (s NPMProjectSelection) InspectionContext(install domain.InstallContext, controls []domain.ProjectControlFile, runtimeIdentity string) (NPMProjectInspectionContext, error) {
	snapshot, err := s.ProjectSnapshot(install, controls, runtimeIdentity)
	if err != nil {
		return NPMProjectInspectionContext{}, err
	}
	return NPMProjectInspectionContext{s, snapshot, append([]domain.ProjectControlFile(nil), controls...), runtimeIdentity}, nil
}

func (c NPMProjectInspectionContext) Valid() bool {
	return c.selection.SourceBound() && c.selection.lock.valid && c.snapshot.Valid() && c.runtime != "" && len(c.controls) == 2
}
func (c NPMProjectInspectionContext) Snapshot() domain.ProjectDependencySnapshot { return c.snapshot }
func (c NPMProjectInspectionContext) Lock() NPMProjectLock                       { return c.selection.Lock() }
func (c NPMProjectInspectionContext) RuntimeIdentity() string                    { return c.runtime }
func (c NPMProjectInspectionContext) ClientVersion() string                      { return c.selection.lock.version }
func (c NPMProjectInspectionContext) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), c.controls...)
}
func (c NPMProjectInspectionContext) MatchesSnapshot(snapshot domain.ProjectDependencySnapshot) bool {
	return c.Valid() && snapshot.Valid() && snapshot.Context() == c.snapshot.Context() && snapshot.Source() == c.snapshot.Source() && snapshot.GraphDigest() == c.snapshot.GraphDigest() && snapshot.DependencyFree() == c.snapshot.DependencyFree() && slices.Equal(snapshot.ControlDigests(), c.snapshot.ControlDigests()) && slices.Equal(snapshot.Dependencies(), c.snapshot.Dependencies())
}

type NPMProjectInspectionInput struct {
	context   NPMProjectInspectionContext
	artifacts []domain.AcquiredArtifact
	instances map[domain.AcquiredArtifact][]NPMProjectPackage
}

func (c NPMProjectInspectionContext) BindAcquired(snapshot domain.ProjectDependencySnapshot, acquired map[domain.DependencyNodeID]domain.AcquiredArtifact) (NPMProjectInspectionInput, error) {
	if !c.MatchesSnapshot(snapshot) || len(acquired) != len(snapshot.Dependencies()) {
		return NPMProjectInspectionInput{}, errNPMProjectInspection
	}
	artifacts, err := bindNPMProjectAcquired(snapshot, acquired, errNPMProjectInspection)
	if err != nil {
		return NPMProjectInspectionInput{}, err
	}
	byNameVersion := make(map[string]domain.AcquiredArtifact, len(artifacts))
	for _, artifact := range artifacts {
		byNameVersion[artifact.Identity().Name()+"@"+artifact.Identity().Version()] = artifact
	}
	instances := make(map[domain.AcquiredArtifact][]NPMProjectPackage, len(artifacts))
	for _, pkg := range c.selection.lock.Packages() {
		artifact, present := byNameVersion[pkg.Name+"@"+pkg.Version]
		if !present {
			return NPMProjectInspectionInput{}, errNPMProjectInspection
		}
		instances[artifact] = append(instances[artifact], pkg)
	}
	if len(instances) != len(artifacts) {
		return NPMProjectInspectionInput{}, errNPMProjectInspection
	}
	return NPMProjectInspectionInput{c, artifacts, instances}, nil
}

func (i NPMProjectInspectionInput) Valid() bool {
	return i.context.Valid() && len(i.artifacts) == len(i.context.selection.artifacts) && len(i.instances) == len(i.artifacts)
}
func (i NPMProjectInspectionInput) Context() NPMProjectInspectionContext { return i.context }
func (i NPMProjectInspectionInput) Artifacts() []domain.AcquiredArtifact {
	return append([]domain.AcquiredArtifact(nil), i.artifacts...)
}

// Index the complete placement set once. Each archive's instances are read in
// O(its placements), without rescanning the whole graph for every archive.
func (i NPMProjectInspectionInput) InstancesFor(artifact domain.AcquiredArtifact) ([]NPMProjectPackage, error) {
	instances, present := i.instances[artifact]
	if !i.Valid() || !present || len(instances) == 0 {
		return nil, errNPMProjectInspection
	}
	return cloneNPMProjectPackages(instances), nil
}

// All project consumers share the existing fixed workspace ceiling and byte /
// entry accounting engine. Native materialization remains under its hard caps.
const NPMProjectInspectionWorkspaceLimit = PNPMInspectionWorkspaceLimit

type NPMProjectInspectionBudget = npmInspectionBudget

func (i NPMProjectInspectionInput) BeginWorkspace() (*NPMProjectInspectionBudget, error) {
	if !i.Valid() || len(i.artifacts) == 0 {
		return nil, errNPMProjectInspection
	}
	instances := make([]string, 0, len(i.context.selection.lock.packages))
	for _, pkg := range i.context.selection.lock.packages {
		instances = append(instances, pkg.Name+"@"+pkg.Version)
	}
	return beginNPMInspectionBudget(i.artifacts, instances)
}

func (i NPMProjectInspectionInput) CheckWorkspace(inventories []ArchiveInventory) error {
	if !i.Valid() {
		return errNPMProjectInspection
	}
	bySubject, err := indexNPMProjectInventories(i.artifacts, inventories, errNPMProjectInspection)
	if err != nil {
		return err
	}
	budget, err := i.BeginWorkspace()
	if err != nil {
		return err
	}
	for _, inventory := range bySubject {
		if err := budget.Add(inventory); err != nil {
			return err
		}
	}
	return budget.Complete()
}

// OfflineControls rewrites only selected acquisition locators to the exact
// measured local tarballs. Original public controls remain frozen separately;
// this private recipe is never published as the user's manifest or lock.
func (i NPMProjectInspectionInput) OfflineControls() ([]domain.ProjectControlFile, error) {
	if !i.Valid() {
		return nil, errNPMProjectInspection
	}
	values := map[string]domain.ProjectControlFile{}
	for _, control := range i.context.controls {
		values[control.Name()] = control
	}
	var document map[string]json.RawMessage
	var packages map[string]map[string]json.RawMessage
	if json.Unmarshal(values["package-lock.json"].Body(), &document) != nil || json.Unmarshal(document["packages"], &packages) != nil {
		return nil, errNPMProjectInspection
	}
	for artifact, instances := range i.instances {
		locator, err := json.Marshal("file:.heliopause/artifacts/" + artifact.Digest().String() + ".tgz")
		if err != nil {
			return nil, err
		}
		for _, instance := range instances {
			entry, present := packages[instance.Location]
			if !present || entry == nil {
				return nil, errNPMProjectInspection
			}
			entry["resolved"] = locator
		}
	}
	var err error
	document["packages"], err = json.Marshal(packages)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	lock, err := domain.NewProjectControlFile("package-lock.json", body, true)
	if err != nil {
		return nil, err
	}
	return []domain.ProjectControlFile{values["package.json"], lock}, nil
}

type NPMProjectInspectionLayout struct {
	input    NPMProjectInspectionInput
	layout   PNPMLayoutArchive
	targets  map[npmLayoutTarget]string
	digest   domain.ContentDigest
	manifest []byte
}

// BindLayout checks every archive member at every native placement, including
// bins and their exact internal links. It grants no execution or Policy result.
func (i NPMProjectInspectionInput) BindLayout(ctx context.Context, inventories []ArchiveInventory, layout PNPMLayoutArchive) (NPMProjectInspectionLayout, error) {
	if ctx == nil || !i.Valid() {
		return NPMProjectInspectionLayout{}, errNPMProjectInspection
	}
	if err := i.CheckWorkspace(inventories); err != nil {
		return NPMProjectInspectionLayout{}, err
	}
	bySubject, err := indexNPMProjectInventories(i.artifacts, inventories, errNPMProjectInspection)
	if err != nil {
		return NPMProjectInspectionLayout{}, err
	}
	placements := make([]npmHoistedPlacement, 0, len(i.context.selection.lock.packages))
	targets := map[npmLayoutTarget]string{}
	for subject, instances := range i.instances {
		inventory, present := bySubject[subject]
		if !present || !inventory.Matches(subject) {
			return NPMProjectInspectionLayout{}, errNPMProjectInspection
		}
		bins, err := pnpmInventoryBins(inventory)
		if err != nil {
			return NPMProjectInspectionLayout{}, errors.Join(errNPMProjectInspection, err)
		}
		for _, instance := range instances {
			if !npmProjectMetadataMatches(instance, inventory.manifest, i.context.ClientVersion()) || !maps.Equal(bins, instance.Bins) {
				return NPMProjectInspectionLayout{}, errNPMProjectInspection
			}
			placements = append(placements, npmHoistedPlacement{instance.Location, inventory, bins})
			targets[npmLayoutTarget{subject, instance.Location}] = instance.Location
		}
	}
	if err := validateNPMHoistedLayout(ctx, placements, layout, 1000, 1000); err != nil {
		return NPMProjectInspectionLayout{}, errors.Join(errNPMProjectInspection, err)
	}
	manifest, err := i.inspectionManifest()
	if err != nil {
		return NPMProjectInspectionLayout{}, err
	}
	digest, err := npmInspectionLayoutDigest("npm-project-inspection-layout/v1", i.context.snapshot.GraphDigest(), i.context.runtime, i.artifacts, layout.Entries(), manifest)
	if err != nil || ctx.Err() != nil {
		return NPMProjectInspectionLayout{}, errors.Join(errNPMProjectInspection, err, ctx.Err(), context.Cause(ctx))
	}
	return NPMProjectInspectionLayout{i, layout, targets, digest, manifest}, nil
}

// The per-dependency lifecycle project has only selected version pins and no
// root scripts. The original root project's lifecycle is a separate subject.
func (i NPMProjectInspectionInput) inspectionManifest() ([]byte, error) {
	packages := map[string]NPMProjectPackage{}
	for _, pkg := range i.context.selection.lock.packages {
		packages[pkg.Location] = pkg
	}
	manifest := map[string]any{"private": true}
	groups := map[string]map[string]string{}
	for _, root := range i.context.selection.lock.roots {
		pkg, selected := packages[root.Location]
		if !selected {
			continue // Explicitly omitted optional/platform roots remain in the frozen lock.
		}
		if groups[root.Group] == nil {
			groups[root.Group] = map[string]string{}
		}
		if prior, exists := groups[root.Group][root.Name]; exists && prior != pkg.Version {
			return nil, errNPMProjectInspection
		}
		groups[root.Group][root.Name] = pkg.Version
	}
	for group, dependencies := range groups {
		manifest[group] = dependencies
	}
	return json.Marshal(manifest)
}

func (l NPMProjectInspectionLayout) Target(subject domain.AcquiredArtifact, instance string) (string, error) {
	root, present := l.targets[npmLayoutTarget{subject, instance}]
	if !present || l.digest.String() == "" {
		return "", errNPMProjectInspection
	}
	return root, nil
}
func (l NPMProjectInspectionLayout) MatchesInput(input NPMProjectInspectionInput) bool {
	return l.digest.String() != "" && l.input.Valid() && input.Valid() && l.input.context.MatchesSnapshot(input.context.Snapshot()) && l.input.context.runtime == input.context.runtime && slices.Equal(l.input.artifacts, input.artifacts)
}
func (l NPMProjectInspectionLayout) Digest() domain.ContentDigest { return l.digest }
func (l NPMProjectInspectionLayout) RuntimeIdentity() string      { return l.input.context.runtime }
func (l NPMProjectInspectionLayout) InspectionManifest() []byte {
	return append([]byte(nil), l.manifest...)
}
func (l NPMProjectInspectionLayout) Entries() []PNPMLayoutEntry { return l.layout.Entries() }
func (l NPMProjectInspectionLayout) ReadFile(name string) ([]byte, error) {
	if l.digest.String() == "" {
		return nil, errNPMProjectInspection
	}
	return l.layout.ReadFile(name)
}

func (i NPMProjectInspectionInput) BindMaterializedLayout(ctx context.Context, inventories []ArchiveInventory, native PNPMLayoutArchive) (NPMProjectInspectionLayout, error) {
	if ctx == nil || !i.Valid() {
		return NPMProjectInspectionLayout{}, errNPMProjectInspection
	}
	cleaned, err := stripNPMHiddenLockCache(ctx, native, true)
	if err != nil {
		return NPMProjectInspectionLayout{}, errors.Join(errNPMProjectInspection, err)
	}
	return i.BindLayout(ctx, inventories, cleaned)
}
