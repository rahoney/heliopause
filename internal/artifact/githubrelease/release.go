package githubrelease

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	maximumReleaseResponseBytes = 2 << 20
	maximumAssetBytes           = 512 << 20
)

// ResolvedAsset is bounded adapter data for one uniquely selected public
// release asset. It has no authority to download or promote bytes.
type ResolvedAsset struct {
	releaseID uint64
	assetID   uint64
	selector  Selector
	digest    domain.ContentDigest
	size      uint64
	immutable bool
	published time.Time
}

func (a ResolvedAsset) ReleaseID() uint64            { return a.releaseID }
func (a ResolvedAsset) AssetID() uint64              { return a.assetID }
func (a ResolvedAsset) Selector() Selector           { return a.selector }
func (a ResolvedAsset) Digest() domain.ContentDigest { return a.digest }
func (a ResolvedAsset) Size() uint64                 { return a.size }
func (a ResolvedAsset) Immutable() bool              { return a.immutable }
func (a ResolvedAsset) PublishedAt() time.Time       { return a.published }

// ParseReleaseForSelector decodes only the release fields required to bind a
// supplied selector to one exact asset. Unknown upstream fields are ignored;
// missing or ambiguous required fields are rejected.
func ParseReleaseForSelector(selector Selector, body []byte) (ResolvedAsset, error) {
	if selector.owner == "" || len(body) == 0 || len(body) > maximumReleaseResponseBytes {
		return ResolvedAsset{}, errors.New("GitHub Release response is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	var release releaseResponse
	if err := decoder.Decode(&release); err != nil || ensureEOF(decoder) != nil {
		return ResolvedAsset{}, errors.New("GitHub Release response is invalid")
	}
	if release.ID == 0 || release.Draft || release.TagName != selector.tag || release.PublishedAt == "" {
		return ResolvedAsset{}, errors.New("GitHub Release response does not match exact selector")
	}
	published, err := time.Parse(time.RFC3339, release.PublishedAt)
	if err != nil || published.IsZero() {
		return ResolvedAsset{}, errors.New("GitHub Release publication time is invalid")
	}
	var selected *releaseAssetResponse
	for index := range release.Assets {
		asset := &release.Assets[index]
		if asset.Name != selector.asset {
			continue
		}
		if selected != nil {
			return ResolvedAsset{}, errors.New("GitHub Release asset is ambiguous")
		}
		selected = asset
	}
	if selected == nil || selected.ID == 0 || selected.State != "uploaded" || selected.Size <= 0 || selected.Size > maximumAssetBytes {
		return ResolvedAsset{}, errors.New("GitHub Release asset is incomplete")
	}
	digestText, ok := strings.CutPrefix(selected.Digest, "sha256:")
	if !ok {
		return ResolvedAsset{}, errors.New("GitHub Release asset requires SHA-256 digest")
	}
	digest, err := domain.NewSHA256Digest(digestText)
	if err != nil {
		return ResolvedAsset{}, errors.New("GitHub Release asset digest is invalid")
	}
	return ResolvedAsset{releaseID: release.ID, assetID: selected.ID, selector: selector, digest: digest, size: uint64(selected.Size), immutable: release.Immutable, published: published.UTC()}, nil
}

type releaseResponse struct {
	ID          uint64                 `json:"id"`
	TagName     string                 `json:"tag_name"`
	Draft       bool                   `json:"draft"`
	Immutable   bool                   `json:"immutable"`
	PublishedAt string                 `json:"published_at"`
	Assets      []releaseAssetResponse `json:"assets"`
}

type releaseAssetResponse struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("unexpected JSON data")
	}
	return nil
}
