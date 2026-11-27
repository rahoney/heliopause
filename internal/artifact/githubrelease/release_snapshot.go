package githubrelease

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sort"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const maximumFrozenAssets = 64

// ReleaseSnapshot freezes discovery only. Neither immutable=true nor a
// GitHub-declared checksum authenticates the publisher or permits installation.
type ReleaseSnapshot struct {
	client    *Client
	selector  Selector
	releaseID uint64
	immutable bool
	assets    map[string]domain.ResolvedArtifact
}

func (s ReleaseSnapshot) ReleaseID() uint64 { return s.releaseID }
func (s ReleaseSnapshot) Immutable() bool   { return s.immutable }

func (s ReleaseSnapshot) Names() []string {
	names := make([]string, 0, len(s.assets))
	for name := range s.assets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ResolveAsset performs no network access and cannot select another tag/repo.
func (s ReleaseSnapshot) ResolveAsset(name string) (domain.ArtifactReference, domain.ResolvedArtifact, error) {
	if s.client == nil || s.releaseID == 0 {
		return domain.ArtifactReference{}, domain.ResolvedArtifact{}, errors.New("release snapshot is unavailable")
	}
	resolved, found := s.assets[name]
	if !found {
		return domain.ArtifactReference{}, domain.ResolvedArtifact{}, errors.New("asset is absent from frozen release")
	}
	reference, err := ParseReference(s.selector.Owner() + "/" + s.selector.Repo() + "@" + s.selector.Tag() + "#" + name)
	if err != nil {
		return domain.ArtifactReference{}, domain.ResolvedArtifact{}, errors.New("frozen release asset is invalid")
	}
	return reference, resolved, nil
}

// Attestations acquires proof for the exact frozen subject, never a supplied URL.
func (s ReleaseSnapshot) Attestations(ctx context.Context, name string) ([][]byte, error) {
	_, resolved, err := s.ResolveAsset(name)
	if err != nil {
		return nil, err
	}
	return s.client.readAttestations(ctx, resolved)
}

// FreezeRelease obtains one exact-tag response and normalizes every bounded
// asset through the existing parser. Subsequent selections use only this map.
func (c *Client) FreezeRelease(ctx context.Context, reference domain.ArtifactReference) (ReleaseSnapshot, error) {
	if err := validRequest(ctx, c); err != nil {
		return ReleaseSnapshot{}, err
	}
	if reference.Source().String() != SourceName {
		return ReleaseSnapshot{}, errors.New("release snapshot requires a GitHub Release reference")
	}
	selector, err := ParseSelector(reference.Locator())
	if err != nil {
		return ReleaseSnapshot{}, err
	}
	body, err := c.release(ctx, selector)
	if err != nil {
		return ReleaseSnapshot{}, err
	}
	primary, err := ParseReleaseForSelector(selector, body)
	if err != nil {
		return ReleaseSnapshot{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	var release releaseResponse
	if decoder.Decode(&release) != nil || ensureEOF(decoder) != nil || len(release.Assets) == 0 || len(release.Assets) > maximumFrozenAssets {
		return ReleaseSnapshot{}, errors.New("frozen release asset set is invalid")
	}
	assets := make(map[string]domain.ResolvedArtifact, len(release.Assets))
	ids := make(map[uint64]bool, len(release.Assets))
	for _, item := range release.Assets {
		if _, found := assets[item.Name]; found || ids[item.ID] {
			return ReleaseSnapshot{}, errors.New("frozen release assets are ambiguous")
		}
		selected, err := ParseSelector(selector.Owner() + "/" + selector.Repo() + "@" + selector.Tag() + "#" + item.Name)
		if err != nil {
			return ReleaseSnapshot{}, errors.New("frozen release asset name is invalid")
		}
		asset, err := ParseReleaseForSelector(selected, body)
		if err != nil {
			return ReleaseSnapshot{}, err
		}
		resolved, err := resolvedArtifactForAsset(reference.Source(), asset)
		if err != nil {
			return ReleaseSnapshot{}, err
		}
		assets[item.Name] = resolved
		ids[item.ID] = true
	}
	return ReleaseSnapshot{client: c, selector: selector, releaseID: primary.ReleaseID(), immutable: primary.Immutable(), assets: assets}, nil
}
