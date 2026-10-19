package githubrelease

import (
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestResolvedSelectionUsesExactRepositoryInsteadOfDisplayName(t *testing.T) {
	source, _ := domain.NewSourceID(SourceName)
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	for _, repo := range []string{"owner-one/repo", "owner/one-repo"} {
		selector, _ := ParseSelector(repo + "@v2#tool.tar.gz")
		identity, _ := domain.NewResolvedArtifactIdentity(source, "owner-one-repo", "v2", "tool.tar.gz")
		locator := assetLocator(ResolvedAsset{selector: selector, releaseID: 1, assetID: 2, digest: digest, size: 17})
		resolved, _ := domain.NewResolvedArtifact(identity, locator, "sha256:"+digest.String())
		actual, size, err := SelectionForResolved(resolved)
		if err != nil || size != 17 || actual.Locator() != selector.Locator() {
			t.Fatal(actual, size, err)
		}
		for _, bad := range []string{locator + "&asset=other", strings.Replace(locator, "release=1", "release=0", 1), strings.Replace(locator, "asset_id=2", "asset_id=02", 1)} {
			value, _ := domain.NewResolvedArtifact(identity, bad, "sha256:"+digest.String())
			if _, _, err := SelectionForResolved(value); err == nil {
				t.Fatal("malformed/noncanonical frozen locator accepted", bad)
			}
		}
		wrong, _ := domain.NewResolvedArtifact(identity, locator, "sha256:"+strings.Repeat("b", 64))
		if _, _, err := SelectionForResolved(wrong); err == nil {
			t.Fatal("declared digest swap accepted")
		}
	}
}
