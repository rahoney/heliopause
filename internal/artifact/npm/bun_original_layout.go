package npm

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// BunOriginalLayout is unapproved immutable Host census data, not a guest layout.
type BunOriginalLayout struct {
	layout   BunLayoutArchive
	uid, gid int
}

func ParseBunOriginalLayout(ctx context.Context, body []byte, uid, gid int) (BunOriginalLayout, error) {
	layout, err := parsePNPMTreeArchiveOwner(ctx, body, "node_modules", uid, gid)
	if err != nil {
		return BunOriginalLayout{}, err
	}
	for _, entry := range layout.entries {
		if entry.Kind == tar.TypeLink {
			return BunOriginalLayout{}, errBunInspectionLayout
		}
	}
	return BunOriginalLayout{BunLayoutArchive{layout}, uid, gid}, nil
}

// ProveOriginalLayout verifies the complete old source closure. It yields only
// ownership provenance; current inspection/Evidence/Policy remain mandatory.
func (i BunInspectionInput) ProveOriginalLayout(ctx context.Context, inventories []ArchiveInventory, original BunOriginalLayout) (domain.ContentDigest, error) {
	if ctx == nil || !i.Valid() || !i.context.snapshot.Valid() || len(i.artifacts) == 0 || len(original.layout.native.entries) == 0 {
		return domain.ContentDigest{}, errBunInspectionLayout
	}
	if _, err := i.bindLayoutForOwner(ctx, inventories, original.layout, original.uid, original.gid); err != nil {
		return domain.ContentDigest{}, err
	}
	type source struct{ Name, Version, SHA256 string }
	sources := make([]source, 0, len(i.artifacts))
	for _, subject := range i.artifacts {
		sources = append(sources, source{subject.Identity().Name(), subject.Identity().Version(), subject.Digest().String()})
	}
	census := sha256.Sum256(original.layout.native.body)
	body, err := json.Marshal(struct {
		Schema, Graph, Census string
		Sources               []source
	}{"bun-original-source-proof/v1", i.context.snapshot.GraphDigest().String(), hex.EncodeToString(census[:]), sources})
	if err != nil {
		return domain.ContentDigest{}, err
	}
	if err := ctx.Err(); err != nil {
		return domain.ContentDigest{}, errors.Join(err, context.Cause(ctx))
	}
	sum := sha256.Sum256(body)
	return domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
}
