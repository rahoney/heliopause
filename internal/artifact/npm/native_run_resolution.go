package npm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// nativeNodeRunResolution freezes public request, native controls and runtime.
// It carries no user program argv, installation target or approval.
func nativeNodeRunResolution(graph domain.LockedDependencyGraph, reference domain.ArtifactReference, runtimeIdentity string, controls []domain.ProjectControlFile, client string) (domain.DependencyResolution, error) {
	type controlDigest struct{ Name, SHA256 string }
	document := struct {
		Schema, Source, Reference, Runtime string
		Controls                           []controlDigest
	}{Schema: client + "-selection/v1", Source: reference.Source().String(), Reference: reference.Locator(), Runtime: runtimeIdentity}
	for _, control := range controls {
		document.Controls = append(document.Controls, controlDigest{control.Name(), control.Digest().String()})
	}
	body, err := json.Marshal(document)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	sum := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return domain.NewDependencyResolution(graph, client+"-isolated:"+runtimeIdentity, digest)
}
