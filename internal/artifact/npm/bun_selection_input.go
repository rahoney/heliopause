package npm

import (
	"bytes"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type BunSelectionIntent string

const (
	BunAdd     BunSelectionIntent = "add"
	BunInstall BunSelectionIntent = "install"
	BunUpdate  BunSelectionIntent = "update"
)

// BunSelectionInput transports only validated public project data. The caller
// still chooses no executable, runtime, registry, configuration or host cache.
type BunSelectionInput struct {
	intent                  BunSelectionIntent
	selector, name, version string
	controls                []domain.ProjectControlFile
}

func NewBunSelectionInput(intent BunSelectionIntent, selector string, controls []domain.ProjectControlFile, version string) (BunSelectionInput, error) {
	if len(controls) < 1 || len(controls) > 2 {
		return BunSelectionInput{}, errBunLock
	}
	byName := map[string]domain.ProjectControlFile{}
	for _, control := range controls {
		name := control.Name()
		if name != "package.json" && name != "bun.lock" || byName[name].Name() != "" || control.Present() && len(control.Body()) == 0 {
			return BunSelectionInput{}, errBunLock
		}
		byName[name] = control
	}
	manifest := byName["package.json"]
	if !manifest.Present() {
		return BunSelectionInput{}, errBunLock
	}
	_, groups, err := bunManifestConstraints(manifest.Body(), version)
	if err != nil {
		return BunSelectionInput{}, err
	}
	lock, exists := byName["bun.lock"]
	if !exists {
		lock, err = domain.NewProjectControlFile("bun.lock", nil, false)
		if err != nil {
			return BunSelectionInput{}, err
		}
	}
	if lock.Present() {
		if _, err := ParseBunLock(manifest.Body(), lock, version); err != nil {
			return BunSelectionInput{}, err
		}
	}
	name := ""
	switch intent {
	case BunInstall:
		if selector != "" {
			return BunSelectionInput{}, errBunLock
		}
	case BunAdd:
		name = selector
		if index := strings.LastIndexByte(selector, '@'); index > 0 {
			name = selector[:index]
			if !supportedBunConstraint(selector[index+1:]) {
				return BunSelectionInput{}, errBunLock
			}
		}
		if validatePackageName(name) != nil {
			return BunSelectionInput{}, errBunLock
		}
	case BunUpdate:
		if selector != "" {
			if validatePackageName(selector) != nil || groups["dependencies"][selector] == "" && groups["devDependencies"][selector] == "" {
				return BunSelectionInput{}, errBunLock
			}
			name = selector
		}
	default:
		return BunSelectionInput{}, errBunLock
	}
	return BunSelectionInput{intent, selector, name, version, []domain.ProjectControlFile{manifest, lock}}, nil
}

func (i BunSelectionInput) Intent() BunSelectionIntent { return i.intent }
func (i BunSelectionInput) Selector() string           { return i.selector }
func (i BunSelectionInput) ClientVersion() string      { return i.version }
func (i BunSelectionInput) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), i.controls...)
}

func (i BunSelectionInput) ParseOutput(manifest []byte, lock domain.ProjectControlFile) (BunLock, error) {
	if len(i.controls) != 2 {
		return BunLock{}, errBunLock
	}
	// Reconstruct even this private value before interpreting output.
	if _, err := NewBunSelectionInput(i.intent, i.selector, i.controls, i.version); err != nil {
		return BunLock{}, err
	}
	_, before, err := bunManifestConstraints(i.controls[0].Body(), i.version)
	if err != nil {
		return BunLock{}, err
	}
	_, after, err := bunManifestConstraints(manifest, i.version)
	if err != nil || !samePNPMManifestMetadata(i.controls[0].Body(), manifest) {
		// This existing helper compares common npm non-dependency metadata;
		// the Bun parser, native solver and presence rules remain separate.
		return BunLock{}, errBunLock
	}
	for _, group := range []string{"dependencies", "devDependencies"} {
		addTarget := i.intent == BunAdd && (group == "dependencies" && before["devDependencies"][i.name] == "" || group == "devDependencies" && before[group][i.name] != "")
		for name, spec := range before[group] {
			actual, exists := after[group][name]
			if !exists || actual != spec && !(i.intent == BunUpdate && (i.name == "" || i.name == name) || addTarget && i.name == name) {
				return BunLock{}, errBunLock
			}
		}
		for name := range after[group] {
			if _, exists := before[group][name]; !exists && !(addTarget && i.name == name) {
				return BunLock{}, errBunLock
			}
		}
		if addTarget && after[group][i.name] == "" {
			return BunLock{}, errBunLock
		}
	}
	if i.intent == BunInstall && i.controls[1].Present() && (!lock.Present() || lock.Name() != "bun.lock" || !bytes.Equal(i.controls[1].Body(), lock.Body())) {
		return BunLock{}, errBunLock
	}
	selected, err := ParseBunLock(manifest, lock, i.version)
	if err != nil {
		return BunLock{}, err
	}
	if i.intent == BunAdd {
		if index := strings.LastIndexByte(i.selector, '@'); index > 0 && exactVersion.MatchString(i.selector[index+1:]) {
			wanted := strings.TrimPrefix(i.selector[index+1:], "v")
			for _, root := range selected.roots {
				if root.Name == i.name {
					for _, pkg := range selected.packages {
						if pkg.Key == root.Instance && pkg.Version != wanted {
							return BunLock{}, errBunLock
						}
					}
				}
			}
		}
	}
	return selected, nil
}
