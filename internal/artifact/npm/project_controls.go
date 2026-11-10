package npm

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const ProjectControlLimit = 4 << 20

var publicDependencySpec = regexp.MustCompile(`^[0-9A-Za-z.*~^<>=|+_ -]{1,128}$`)

// ProjectRootReference identifies the existing supported single direct root.
// Version/range selection stays with pinned npm, never a Host executable.
func ProjectRootReference(manifest []byte) (domain.ArtifactReference, error) {
	dependencies, err := projectDependencies(manifest)
	if err != nil {
		return domain.ArtifactReference{}, err
	}
	for name := range dependencies {
		return ParseReference(name)
	}
	return domain.ArtifactReference{}, errors.New("npm project has no supported root")
}

// ParseProjectControls binds preserved public constraints to an exact registry
// lock graph. Local paths, aliases, workspaces and unsupported roots fail closed.
func ParseProjectControls(reference domain.ArtifactReference, manifest, lock []byte) (domain.LockedDependencyGraph, error) {
	dependencies, err := projectDependencies(manifest)
	if err != nil || strictControlJSON(lock) != nil {
		return domain.LockedDependencyGraph{}, errors.New("npm public selection controls are unsupported")
	}
	var document struct{ Packages map[string]json.RawMessage }
	if json.Unmarshal(lock, &document) != nil {
		return domain.LockedDependencyGraph{}, errors.New("npm project lock is invalid")
	}
	root, err := projectDependencies(document.Packages[""])
	if err != nil || len(root) != len(dependencies) {
		return domain.LockedDependencyGraph{}, errors.New("npm root constraints differ from the manifest")
	}
	for name, spec := range dependencies {
		if root[name] != spec {
			return domain.LockedDependencyGraph{}, errors.New("npm root constraints changed during selection")
		}
	}
	return ParsePackageLockV3(reference, lock)
}

func projectDependencies(body []byte) (map[string]string, error) {
	if strictControlJSON(body) != nil {
		return nil, errors.New("npm manifest is invalid or exceeds bounds")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return nil, errors.New("npm manifest is not an object")
	}
	for _, name := range []string{"devDependencies", "optionalDependencies", "peerDependencies", "workspaces", "overrides", "resolutions", "bundledDependencies", "bundleDependencies"} {
		if value, ok := fields[name]; ok && string(value) != "{}" && string(value) != "[]" {
			return nil, errors.New("npm project uses unsupported dependency semantics")
		}
	}
	var dependencies map[string]string
	if json.Unmarshal(fields["dependencies"], &dependencies) != nil || len(dependencies) != 1 {
		return nil, errors.New("npm project requires exactly one supported direct dependency")
	}
	for name, spec := range dependencies {
		if validatePackageName(name) != nil || spec != strings.TrimSpace(spec) || !publicDependencySpec.MatchString(spec) {
			return nil, errors.New("npm project requires public registry constraints; local-file legacy controls require an explicit inspected install")
		}
	}
	return dependencies, nil
}

// Controls are untrusted local input. Reject duplicate keys, excessive depth,
// trailing values and token floods before interpreting a dependency selection.
func strictControlJSON(body []byte) error {
	if len(body) == 0 || len(body) > ProjectControlLimit {
		return errors.New("npm control exceeds bounds")
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	budget := 200000
	var value func(int) error
	value = func(depth int) error {
		budget--
		if depth > 32 || budget < 0 {
			return errors.New("npm control nesting exceeds bounds")
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		delimiter, compound := token.(json.Delim)
		if !compound {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return errors.New("duplicate npm control key")
				}
				seen[name] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return errors.New("invalid npm control delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); !errors.Is(err, io.EOF) {
		return errors.New("npm control contains trailing JSON")
	}
	return nil
}

// SameSelectionGraph compares normalized acquisition identity, exact install
// paths, lifecycle flags and edges; equal package names alone are insufficient.
func SameSelectionGraph(a, b domain.LockedDependencyGraph) bool {
	if a.Primary() != b.Primary() || len(a.Nodes()) != len(b.Nodes()) || len(a.Edges()) != len(b.Edges()) {
		return false
	}
	byID := map[domain.DependencyNodeID]domain.LockedDependency{}
	for _, node := range a.Nodes() {
		byID[node.Node()] = node
	}
	for _, node := range b.Nodes() {
		before, ok := byID[node.Node()]
		if !ok || before.Artifact() != node.Artifact() || before.Role() != node.Role() || before.RecordPath() != node.RecordPath() || before.HostInstallAction() != node.HostInstallAction() {
			return false
		}
	}
	edges := map[domain.DependencyEdge]bool{}
	for _, edge := range a.Edges() {
		edges[edge] = true
	}
	for _, edge := range b.Edges() {
		if !edges[edge] {
			return false
		}
	}
	return true
}
