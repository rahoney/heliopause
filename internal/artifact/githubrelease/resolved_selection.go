package githubrelease

import (
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// SelectionForResolved exposes only the exact normalized selector and size from
// the existing frozen locator. Display names cannot prove repository ownership.
func SelectionForResolved(resolved domain.ResolvedArtifact) (Selector, uint64, error) {
	bound, err := parseAssetLocator(resolved.AcquisitionLocator())
	if err != nil {
		return Selector{}, 0, err
	}
	identity := resolved.Identity()
	asset := ResolvedAsset{selector: bound.selector, releaseID: bound.releaseID, assetID: bound.assetID, digest: bound.digest, size: bound.size}
	if identity.Source().String() != SourceName || identity.Name() != bound.selector.Owner()+"-"+bound.selector.Repo() || identity.Version() != bound.selector.Tag() || identity.Variant() != bound.selector.Asset() || resolved.DeclaredIntegrity() != "sha256:"+bound.digest.String() || assetLocator(asset) != resolved.AcquisitionLocator() {
		return Selector{}, 0, errors.New("GitHub resolved identity differs from its frozen selection")
	}
	return bound.selector, bound.size, nil
}
