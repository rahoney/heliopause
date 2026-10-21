// Package releasefixture supplies unsigned release material for file-mechanics
// tests. It never modifies published evidence or creates a provenance receipt.
package releasefixture

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type Materials struct{ Manifest, SBOM, Images []byte }

// BindCurrentRuntime rebinds synthetic metadata to generated runtime identities.
// Any signature over the original manifest is invalid for the returned bytes.
// Callers must retain separate real-signature and incompatible-runtime tests.
func BindCurrentRuntime(manifest, sbom, images []byte) (Materials, error) {
	var m, s, im map[string]any
	for i, value := range []*map[string]any{&m, &s, &im} {
		if err := json.Unmarshal([][]byte{manifest, sbom, images}[i], value); err != nil {
			return Materials{}, err
		}
	}
	metadata, ok := s["metadata"].(map[string]any)
	if !ok {
		return Materials{}, errors.New("synthetic SBOM metadata missing")
	}
	component, ok := metadata["component"].(map[string]any)
	if !ok {
		return Materials{}, errors.New("synthetic SBOM component missing")
	}
	properties, ok := component["properties"].([]any)
	if !ok {
		return Materials{}, errors.New("synthetic SBOM properties missing")
	}
	goProperties := 0
	for _, raw := range properties {
		property, ok := raw.(map[string]any)
		if !ok {
			return Materials{}, errors.New("synthetic SBOM property invalid")
		}
		if property["name"] == "heliopause:go-version" {
			property["value"] = "go" + runtimeidentity.GoVersion
			goProperties++
		}
	}
	if goProperties != 1 {
		return Materials{}, errors.New("synthetic SBOM Go identity ambiguous")
	}
	nodeVersion, _, _ := strings.Cut(strings.TrimPrefix(runtimeidentity.NodeImageReference, "node:"), "-slim@")
	identities := map[string]struct{ reference, repository, version string }{
		"node":   {runtimeidentity.NodeImageReference, "docker.io/library/node", nodeVersion},
		"python": {runtimeidentity.PythonImageReference, "docker.io/library/python", runtimeidentity.PythonVersion},
		"golang": {runtimeidentity.GoImageReference, "docker.io/library/golang", runtimeidentity.GoVersion},
		"rust":   {runtimeidentity.RustImageReference, "docker.io/library/rust", runtimeidentity.RustVersion},
	}
	entries, ok := im["images"].([]any)
	if !ok || len(entries) != len(identities) {
		return Materials{}, errors.New("synthetic image inventory mismatch")
	}
	for _, raw := range entries {
		item, ok := raw.(map[string]any)
		if !ok {
			return Materials{}, errors.New("synthetic image entry invalid")
		}
		name, _ := item["name"].(string)
		identity, found := identities[name]
		if !found {
			return Materials{}, errors.New("synthetic image identity ambiguous")
		}
		_, digest, _ := strings.Cut(identity.reference, "@")
		item["reference"], item["repository"], item["digest"], item["runtime_version"] = identity.reference, identity.repository, digest, identity.version
		delete(identities, name)
	}
	m["runtime_lock"] = map[string]any{"sha256": runtimeidentity.RuntimeLockSHA256, "gvisor_release": runtimeidentity.GVisorRelease, "gvisor_commit": runtimeidentity.GVisorCommit}
	result := Materials{}
	var err error
	result.SBOM, err = json.Marshal(s)
	if err != nil {
		return Materials{}, err
	}
	result.Images, err = json.Marshal(im)
	if err != nil {
		return Materials{}, err
	}
	for key, body := range map[string][]byte{"sbom": result.SBOM, "runtime_images": result.Images} {
		record, ok := m[key].(map[string]any)
		if !ok {
			return Materials{}, errors.New("synthetic manifest file record missing")
		}
		record["size"], record["sha256"] = len(body), fmt.Sprintf("%x", sha256.Sum256(body))
	}
	result.Manifest, err = json.Marshal(m)
	return result, err
}
