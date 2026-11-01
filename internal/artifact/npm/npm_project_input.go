package npm

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type NPMProjectIntent string

const (
	NPMProjectInstall NPMProjectIntent = "install"
	NPMProjectCI      NPMProjectIntent = "ci"
)

var errNPMProjectInput = errors.New("npm project controls or selection intent are invalid or unsupported")

// NPMProjectInput freezes a whole-project request, without choosing a primary
// package or granting quarantine/publication authority.
type NPMProjectInput struct {
	intent   NPMProjectIntent
	version  string
	controls []domain.ProjectControlFile
}

func NewNPMProjectInput(intent NPMProjectIntent, controls []domain.ProjectControlFile, clientVersion string) (NPMProjectInput, error) {
	if intent != NPMProjectInstall && intent != NPMProjectCI || !exactVersion.MatchString(clientVersion) || len(controls) != 2 {
		return NPMProjectInput{}, errNPMProjectInput
	}
	values := map[string]domain.ProjectControlFile{}
	for _, control := range controls {
		if control.Name() != "package.json" && control.Name() != "package-lock.json" || values[control.Name()].Name() != "" || control.Present() && len(control.Body()) == 0 {
			return NPMProjectInput{}, errNPMProjectInput
		}
		values[control.Name()] = control
	}
	manifest, lock := values["package.json"], values["package-lock.json"]
	if !manifest.Present() || intent == NPMProjectCI && !lock.Present() {
		return NPMProjectInput{}, errNPMProjectInput
	}
	if _, err := NPMProjectRootLifecycleScripts(manifest.Body()); err != nil {
		return NPMProjectInput{}, err
	}
	groups, err := npmProjectConstraints(manifest.Body(), clientVersion)
	if err != nil {
		return NPMProjectInput{}, err
	}
	if lock.Present() {
		locked, err := npmProjectLockConstraints(lock.Body(), clientVersion)
		if err != nil || intent == NPMProjectCI && !sameNPMProjectConstraints(groups, locked) {
			return NPMProjectInput{}, errNPMProjectInput
		}
	}
	return NPMProjectInput{intent, clientVersion, []domain.ProjectControlFile{manifest, lock}}, nil
}

func (i NPMProjectInput) Intent() NPMProjectIntent { return i.intent }
func (i NPMProjectInput) ClientVersion() string    { return i.version }
func (i NPMProjectInput) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), i.controls...)
}

// ValidateOutput preserves the original manifest byte-for-byte. CI additionally
// preserves the existing lock; install may generate or reconcile that lock.
// Native selection, archive binding and whole-graph inspection remain required.
func (i NPMProjectInput) ValidateOutput(manifest, lock []byte) ([]domain.ProjectControlFile, error) {
	validated, err := NewNPMProjectInput(i.intent, i.controls, i.version)
	if err != nil || !bytes.Equal(manifest, validated.controls[0].Body()) || i.intent == NPMProjectCI && !bytes.Equal(lock, validated.controls[1].Body()) {
		return nil, errNPMProjectInput
	}
	groups, err := npmProjectConstraints(manifest, i.version)
	if err != nil {
		return nil, err
	}
	locked, err := npmProjectLockConstraints(lock, i.version)
	if err != nil || !sameNPMProjectConstraints(groups, locked) {
		return nil, errNPMProjectInput
	}
	result := make([]domain.ProjectControlFile, 0, 2)
	for _, item := range []struct {
		name string
		body []byte
	}{{"package.json", manifest}, {"package-lock.json", lock}} {
		control, err := domain.NewProjectControlFile(item.name, item.body, true)
		if err != nil {
			return nil, err
		}
		result = append(result, control)
	}
	return result, nil
}

var npmProjectDependencyGroups = [...]string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"}

func npmProjectConstraints(body []byte, clientVersion string) (map[string]map[string]string, error) {
	if strictControlJSON(body) != nil {
		return nil, errNPMProjectInput
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return nil, errNPMProjectInput
	}
	for _, key := range []string{"workspaces", "overrides", "resolutions", "bundledDependencies", "bundleDependencies", "pnpm", "devEngines"} {
		if value, exists := fields[key]; exists && string(bytes.TrimSpace(value)) != "{}" && string(bytes.TrimSpace(value)) != "[]" {
			return nil, errNPMProjectInput
		}
	}
	if value, exists := fields["packageManager"]; exists {
		var manager string
		if json.Unmarshal(value, &manager) != nil || manager != "npm@"+clientVersion {
			return nil, errNPMProjectInput
		}
	}
	if value, exists := fields["scripts"]; exists {
		var scripts map[string]string
		if json.Unmarshal(value, &scripts) != nil || scripts == nil {
			return nil, errNPMProjectInput
		}
	}
	groups := map[string]map[string]string{}
	names := map[string]bool{}
	for _, name := range npmProjectDependencyGroups {
		var group map[string]string
		if value, exists := fields[name]; exists {
			if json.Unmarshal(value, &group) != nil || group == nil {
				return nil, errNPMProjectInput
			}
		}
		for pkg, spec := range group {
			if validatePackageName(pkg) != nil || spec != strings.TrimSpace(spec) || !publicDependencySpec.MatchString(spec) {
				return nil, errNPMProjectInput
			}
			names[pkg] = true
		}
		groups[name] = group
	}
	if len(names) > 256 {
		return nil, errNPMProjectInput
	}
	if !validNPMProjectPeerMeta(fields) {
		return nil, errNPMProjectInput
	}
	return groups, nil
}

func sameNPMProjectConstraints(a, b map[string]map[string]string) bool {
	for _, group := range npmProjectDependencyGroups {
		if !maps.Equal(a[group], b[group]) {
			return false
		}
	}
	return true
}

// Validate all original lock locators before invoking npm, including entries
// which a changed manifest may later discard. No credentials or alternate
// registry source may reach the isolated selector through an old lock.
func npmProjectLockConstraints(body []byte, clientVersion string) (map[string]map[string]string, error) {
	if strictControlJSON(body) != nil {
		return nil, errNPMProjectInput
	}
	var lock struct {
		LockfileVersion int
		Packages        map[string]json.RawMessage
	}
	if json.Unmarshal(body, &lock) != nil || lock.LockfileVersion != 3 || len(lock.Packages) == 0 || len(lock.Packages) > pnpmLockNodeLimit+1 {
		return nil, errNPMProjectInput
	}
	groups, err := npmProjectConstraints(lock.Packages[""], clientVersion)
	if err != nil {
		return nil, err
	}
	for location, value := range lock.Packages {
		if location == "" {
			continue
		}
		name, valid := packageNameFromLockPath(location)
		var item packageLockPackage
		if !valid || json.Unmarshal(value, &item) != nil || item.Name != "" && item.Name != name || item.Link || item.InBundle || !exactVersion.MatchString(item.Version) || !validRegistryTarball(item.Resolved) || !validSHA512SRI(item.Integrity) {
			return nil, errNPMProjectInput
		}
		for _, dependencies := range []map[string]string{item.Dependencies, item.OptionalDependencies, item.PeerDependencies} {
			for dependency, spec := range dependencies {
				if validatePackageName(dependency) != nil || spec != strings.TrimSpace(spec) || !publicDependencySpec.MatchString(spec) {
					return nil, errNPMProjectInput
				}
			}
		}
	}
	return groups, nil
}
