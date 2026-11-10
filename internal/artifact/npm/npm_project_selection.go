package npm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errNPMProjectBinding = errors.New("npm complete selection differs from its controls or independent registry metadata")

// NPMProjectSelection binds unique archives while retaining every placement.
// It is unapproved input to whole-project inspection, not an ALLOW/cache receipt.
type NPMProjectSelection struct {
	bound     bool
	lock      NPMProjectLock
	artifacts []domain.ResolvedArtifact
}

func (s NPMProjectSelection) SourceBound() bool    { return s.bound }
func (s NPMProjectSelection) Lock() NPMProjectLock { return s.lock }
func (s NPMProjectSelection) Artifacts() []domain.ResolvedArtifact {
	return append([]domain.ResolvedArtifact(nil), s.artifacts...)
}

func (r *Resolver) BindNPMProjectLock(ctx context.Context, lock NPMProjectLock) (NPMProjectSelection, error) {
	if ctx == nil || r == nil || r.registry == nil || r.client == nil || !lock.valid || r.registry.Scheme != "https" || !strings.EqualFold(r.registry.Host, "registry.npmjs.org") || r.registry.Path != "/" {
		return NPMProjectSelection{}, errNPMProjectBinding
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	selection := NPMProjectSelection{lock: lock}
	type candidate struct {
		facts    NPMProjectPackage
		artifact domain.ResolvedArtifact
	}
	candidates := map[string]candidate{}
	for _, pkg := range lock.packages {
		if err := ctx.Err(); err != nil {
			return NPMProjectSelection{}, errors.Join(errNPMProjectBinding, err)
		}
		key := pkg.Name + "@" + pkg.Version
		facts := pkg
		facts.Location, facts.Targets, facts.HasInstallScript = "", nil, false
		value, exists := candidates[key]
		if !exists {
			body, err := r.fetchMetadataBody(ctx, pkg.Name, pkg.Version)
			if err != nil {
				return NPMProjectSelection{}, errors.Join(errNPMProjectBinding, err)
			}
			var metadata npmVersionMetadata
			if !utf8.Valid(body) || strictControlJSON(body) != nil || json.Unmarshal(body, &metadata) != nil {
				return NPMProjectSelection{}, errNPMProjectBinding
			}
			source, err := domain.NewSourceID("npm")
			if err != nil {
				return NPMProjectSelection{}, err
			}
			artifact, err := r.resolvedVersion(source, pkg.Name, pkg.Version, metadata)
			if err != nil {
				return NPMProjectSelection{}, errors.Join(errNPMProjectBinding, err)
			}
			if !npmProjectMetadataMatches(pkg, body, lock.version) {
				return NPMProjectSelection{}, errNPMProjectBinding
			}
			value = candidate{facts, artifact}
			candidates[key] = value
			selection.artifacts = append(selection.artifacts, artifact)
		}
		identity := value.artifact.Identity()
		if identity.Source().String() != "npm" || identity.Name() != pkg.Name || identity.Version() != pkg.Version || identity.Variant() != "tarball" || value.artifact.AcquisitionLocator() != pkg.Registry || value.artifact.DeclaredIntegrity() != pkg.Integrity || !reflect.DeepEqual(facts, value.facts) {
			return NPMProjectSelection{}, errNPMProjectBinding
		}
	}
	if err := ctx.Err(); err != nil {
		return NPMProjectSelection{}, errors.Join(errNPMProjectBinding, err)
	}
	selection.bound = true
	return selection, nil
}

// Published development dependencies are not installation inputs of a package.
// Project-root validation must not accidentally install or validate those roots.
func npmProjectMetadataMatches(pkg NPMProjectPackage, body []byte, clientVersion string) bool {
	if strictControlJSON(body) != nil {
		return false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return false
	}
	projection := map[string]json.RawMessage{}
	for _, key := range []string{"dependencies", "optionalDependencies", "peerDependencies", "peerDependenciesMeta", "bundledDependencies", "bundleDependencies"} {
		if value, exists := fields[key]; exists {
			projection[key] = value
		}
	}
	projected, err := json.Marshal(projection)
	if err != nil {
		return false
	}
	groups, err := npmProjectConstraints(projected, clientVersion)
	if err != nil || !maps.Equal(groups["dependencies"], pkg.Dependencies) || !maps.Equal(groups["optionalDependencies"], pkg.OptionalDependencies) || !maps.Equal(groups["peerDependencies"], pkg.PeerDependencies) {
		return false
	}
	var item npmProjectPackageWire
	if json.Unmarshal(body, &item) != nil || !slices.Equal(item.OS, pkg.OS) || !slices.Equal(item.CPU, pkg.CPU) || !slices.Equal(item.LibC, pkg.LibC) {
		return false
	}
	optionalPeers := map[string]bool{}
	for name, meta := range item.PeerDependenciesMeta {
		optionalPeers[name] = meta.Optional
	}
	if !maps.Equal(optionalPeers, pkg.OptionalPeers) {
		return false
	}
	bins := map[string]string{}
	if len(item.Bin) != 0 {
		bins, err = bunPackageBins(item.Bin, pkg.Name)
		if err != nil {
			return false
		}
	}
	return maps.Equal(bins, pkg.Bins)
}

// ProjectSnapshot binds actual runtime, controls and full native wiring. The
// Domain snapshot deduplicates archives without flattening placement identity.
func (s NPMProjectSelection) ProjectSnapshot(install domain.InstallContext, controls []domain.ProjectControlFile, runtimeIdentity string) (domain.ProjectDependencySnapshot, error) {
	if !s.bound || !s.lock.valid || !install.Valid() || install.Mode() != domain.InstallNPMProject || len(controls) != 2 || runtimeIdentity == "" || len(runtimeIdentity) > 1024 || !utf8.ValidString(runtimeIdentity) || strings.ContainsFunc(runtimeIdentity, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return domain.ProjectDependencySnapshot{}, errNPMProjectBinding
	}
	values := map[string]domain.ProjectControlFile{}
	var digests []domain.ProjectControlDigest
	for _, control := range controls {
		if control.Name() != "package.json" && control.Name() != "package-lock.json" || values[control.Name()].Name() != "" || !control.Present() {
			return domain.ProjectDependencySnapshot{}, errNPMProjectBinding
		}
		values[control.Name()] = control
		digest, err := domain.NewProjectControlDigest(control.Name(), control.Digest())
		if err != nil {
			return domain.ProjectDependencySnapshot{}, err
		}
		digests = append(digests, digest)
	}
	manifest, lock := values["package.json"].Body(), values["package-lock.json"].Body()
	if s.lock.manifestSHA != sha256.Sum256(manifest) || s.lock.lockSHA != sha256.Sum256(lock) {
		return domain.ProjectDependencySnapshot{}, errNPMProjectBinding
	}
	parsed, err := ParseNPMProjectLock(manifest, lock, s.lock.version)
	if err != nil || !reflect.DeepEqual(parsed, s.lock) {
		return domain.ProjectDependencySnapshot{}, errNPMProjectBinding
	}
	wanted := map[string]NPMProjectPackage{}
	for _, pkg := range s.lock.packages {
		key := pkg.Name + "@" + pkg.Version
		if before, exists := wanted[key]; exists && (before.Registry != pkg.Registry || before.Integrity != pkg.Integrity) {
			return domain.ProjectDependencySnapshot{}, errNPMProjectBinding
		}
		wanted[key] = pkg
	}
	if len(wanted) != len(s.artifacts) {
		return domain.ProjectDependencySnapshot{}, errNPMProjectBinding
	}
	seen := map[string]bool{}
	for _, artifact := range s.artifacts {
		identity := artifact.Identity()
		key := identity.Name() + "@" + identity.Version()
		pkg, exists := wanted[key]
		if !exists || seen[key] || identity.Source().String() != "npm" || identity.Variant() != "tarball" || pkg.Registry != artifact.AcquisitionLocator() || pkg.Integrity != artifact.DeclaredIntegrity() {
			return domain.ProjectDependencySnapshot{}, errNPMProjectBinding
		}
		seen[key] = true
	}
	document := struct {
		Schema, Target, Mode, ClientVersion, Runtime string
		ManifestSHA256, LockSHA256                   string
		Roots                                        []NPMProjectRoot
		Packages                                     []NPMProjectPackage
	}{"npm-project-selection/v1", install.Target().String(), string(install.Mode()), s.lock.version, runtimeIdentity, values["package.json"].Digest().String(), values["package-lock.json"].Digest().String(), s.lock.Roots(), s.lock.Packages()}
	body, err := json.Marshal(document)
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	sum := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	source, err := domain.NewSourceID("npm")
	if err != nil {
		return domain.ProjectDependencySnapshot{}, err
	}
	if parsed.DependencyFree() {
		return domain.NewDependencyFreeProjectSnapshot(install, source, digests, digest)
	}
	return domain.NewProjectDependencySnapshot(install, source, digests, s.artifacts, digest)
}
