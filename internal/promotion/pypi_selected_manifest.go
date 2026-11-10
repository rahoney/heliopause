package promotion

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Both guarded update and ordinary UV installation verify the frozen resolver
// subject as well as the independently inspected graph and staged documents.
func verifyPythonSelectionManifest(bundle domain.VerifiedBundle, install domain.InstallContext, resolution domain.DependencyResolution, scope string) error {
	if err := verifyDocuments(bundle); err != nil {
		return err
	}
	var metadata struct {
		Resolver struct {
			Runtime string `json:"runtime_identity"`
			Lock    string `json:"lockfile_sha256"`
		} `json:"resolver"`
		Target struct {
			Path        string `json:"path"`
			RequiresNew bool   `json:"requires_new"`
		} `json:"target"`
		Controls []struct {
			Name    string `json:"name"`
			Content []byte `json:"content"`
		} `json:"project_controls"`
	}
	if json.Unmarshal(bundle.ManifestDocument(), &metadata) != nil || metadata.Target.Path != install.Target().String() || metadata.Target.RequiresNew != install.RequiresNewTarget() || metadata.Resolver.Lock != resolution.LockfileDigest().String() || metadata.Resolver.Runtime != resolution.RuntimeIdentity() || len(metadata.Controls) != len(resolution.ProjectControls()) {
		return errors.New(scope + " Manifest belongs to another selection")
	}
	expected := map[string][]byte{}
	for _, control := range resolution.ProjectControls() {
		expected[control.Name()] = control.Body()
	}
	for _, control := range metadata.Controls {
		if expected[control.Name] == nil || !bytes.Equal(expected[control.Name], control.Content) {
			return errors.New(scope + " Manifest control differs from selection")
		}
		delete(expected, control.Name)
	}
	if len(expected) != 0 {
		return errors.New(scope + " Manifest controls are incomplete")
	}
	return nil
}
