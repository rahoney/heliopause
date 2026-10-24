package npm

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// PNPMSelectionIntent describes selection only, before quarantine or publication.
type PNPMSelectionIntent string

const (
	PNPMAdd     PNPMSelectionIntent = "add"
	PNPMInstall PNPMSelectionIntent = "install"
	PNPMUpdate  PNPMSelectionIntent = "update"
)

// PNPMSelectionInput is an immutable, preflight-validated public project input.
// It transports no configuration, workspace, credential or executable file.
type PNPMSelectionInput struct {
	intent                  PNPMSelectionIntent
	selector, name, version string
	controls                []domain.ProjectControlFile
}

func NewPNPMSelectionInput(intent PNPMSelectionIntent, selector string, controls []domain.ProjectControlFile, clientVersion string) (PNPMSelectionInput, error) {
	if len(controls) < 1 || len(controls) > 2 {
		return PNPMSelectionInput{}, errPNPMLock
	}
	values := map[string][]byte{}
	seen := map[string]bool{}
	for _, control := range controls {
		name := control.Name()
		if name != "package.json" && name != "pnpm-lock.yaml" || seen[name] || control.Present() && len(control.Body()) == 0 {
			return PNPMSelectionInput{}, errPNPMLock
		}
		seen[name] = true
		if control.Present() {
			values[name] = control.Body()
		}
	}
	groups, err := pnpmManifestConstraints(values["package.json"], clientVersion)
	if err != nil {
		return PNPMSelectionInput{}, err
	}
	if lock, exists := values["pnpm-lock.yaml"]; exists {
		if _, err := ParsePNPMLock(values["package.json"], lock, clientVersion); err != nil {
			return PNPMSelectionInput{}, err
		}
	}
	name := ""
	switch intent {
	case PNPMInstall:
		if selector != "" {
			return PNPMSelectionInput{}, errPNPMLock
		}
	case PNPMAdd:
		name = selector
		index := strings.LastIndexByte(selector, '@')
		if index > 0 {
			name = selector[:index]
			spec := selector[index+1:]
			if spec != strings.TrimSpace(spec) || !publicDependencySpec.MatchString(spec) {
				return PNPMSelectionInput{}, errPNPMLock
			}
		}
		if validatePackageName(name) != nil {
			return PNPMSelectionInput{}, errPNPMLock
		}
	case PNPMUpdate:
		if selector != "" {
			if validatePackageName(selector) != nil || groups["dependencies"][selector] == "" && groups["devDependencies"][selector] == "" {
				return PNPMSelectionInput{}, errPNPMLock
			}
			name = selector
		}
	default:
		return PNPMSelectionInput{}, errPNPMLock
	}
	result := PNPMSelectionInput{intent: intent, selector: selector, name: name, version: clientVersion}
	for _, name := range []string{"package.json", "pnpm-lock.yaml"} {
		body, present := values[name]
		control, err := domain.NewProjectControlFile(name, body, present)
		if err != nil {
			return PNPMSelectionInput{}, err
		}
		result.controls = append(result.controls, control)
	}
	return result, nil
}

func (i PNPMSelectionInput) Intent() PNPMSelectionIntent { return i.intent }
func (i PNPMSelectionInput) Selector() string            { return i.selector }
func (i PNPMSelectionInput) ClientVersion() string       { return i.version }
func (i PNPMSelectionInput) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), i.controls...)
}

// ParseOutput keeps native peer instances and rejects changes beyond the intent.
// Neither this selection nor its controls constitute an ALLOW decision.
func (i PNPMSelectionInput) ParseOutput(manifest, lock []byte) (PNPMLock, error) {
	if len(i.controls) != 2 {
		return PNPMLock{}, errPNPMLock
	}
	before, err := pnpmManifestConstraints(i.controls[0].Body(), i.version)
	if err != nil {
		return PNPMLock{}, err
	}
	after, err := pnpmManifestConstraints(manifest, i.version)
	if err != nil {
		return PNPMLock{}, err
	}
	if !samePNPMManifestMetadata(i.controls[0].Body(), manifest) {
		return PNPMLock{}, errPNPMLock
	}
	for _, group := range []string{"dependencies", "devDependencies"} {
		allowedAdd := i.intent == PNPMAdd && (group == "dependencies" && before["devDependencies"][i.name] == "" || group == "devDependencies" && before[group][i.name] != "")
		for name, spec := range before[group] {
			actual, exists := after[group][name]
			if !exists || actual != spec && !(i.intent == PNPMUpdate && (i.name == "" || i.name == name) || allowedAdd && i.name == name) {
				return PNPMLock{}, errPNPMLock
			}
		}
		for name := range after[group] {
			if _, exists := before[group][name]; !exists && !(allowedAdd && i.name == name) {
				return PNPMLock{}, errPNPMLock
			}
		}
		if allowedAdd && after[group][i.name] == "" {
			return PNPMLock{}, errPNPMLock
		}
	}
	selected, err := ParsePNPMLock(manifest, lock, i.version)
	if err != nil {
		return PNPMLock{}, err
	}
	if i.intent == PNPMAdd {
		index := strings.LastIndexByte(i.selector, '@')
		if index > 0 && exactVersion.MatchString(i.selector[index+1:]) {
			for _, root := range selected.Roots() {
				if root.Name == i.name {
					_, version, _, err := pnpmPackageKey(root.Snapshot)
					if err != nil || version != i.selector[index+1:] {
						return PNPMLock{}, errPNPMLock
					}
				}
			}
		}
	}
	return selected, nil
}

func samePNPMManifestMetadata(before, after []byte) bool {
	var a, b map[string]json.RawMessage
	if json.Unmarshal(before, &a) != nil || json.Unmarshal(after, &b) != nil {
		return false
	}
	for _, fields := range []map[string]json.RawMessage{a, b} {
		delete(fields, "dependencies")
		delete(fields, "devDependencies")
	}
	// Canonical JSON normalizes formatting and object order while retaining
	// every non-dependency field and its value, including lifecycle scripts.
	var av, bv any
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	ad, bd := json.NewDecoder(bytes.NewReader(ab)), json.NewDecoder(bytes.NewReader(bb))
	ad.UseNumber()
	bd.UseNumber()
	if ad.Decode(&av) != nil || bd.Decode(&bv) != nil {
		return false
	}
	ab, _ = json.Marshal(av)
	bb, _ = json.Marshal(bv)
	return bytes.Equal(ab, bb)
}
