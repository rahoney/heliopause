package npm

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// PNPMOriginalLayout is an unverified, immutable Host census view. It cannot
// yield an inspection layout or authorize preparation, execution or Policy.
type PNPMOriginalLayout struct {
	layout   PNPMLayoutArchive
	uid, gid int
}

// ParsePNPMOriginalLayout preserves actual ownership. The caller must capture
// and guard the live tree independently; a tar header is no ownership proof.
func ParsePNPMOriginalLayout(ctx context.Context, body []byte, uid, gid int) (PNPMOriginalLayout, error) {
	layout, err := parsePNPMTreeArchiveOwner(ctx, body, "node_modules", uid, gid)
	if err != nil {
		return PNPMOriginalLayout{}, err
	}
	for _, entry := range layout.entries {
		if entry.Kind == tar.TypeLink {
			return PNPMOriginalLayout{}, errPNPMLayoutArchive
		}
	}
	return PNPMOriginalLayout{layout, uid, gid}, nil
}

// ProveOriginalLayout checks the complete old tree against independently
// acquired static archives. Its digest records ownership proof only; it is
// never a current verification, observation, cache or promotion approval.
func (i PNPMInspectionInput) ProveOriginalLayout(ctx context.Context, inventories []ArchiveInventory, original PNPMOriginalLayout) (domain.ContentDigest, error) {
	if ctx == nil || !i.Valid() || len(i.artifacts) == 0 || len(original.layout.entries) == 0 {
		return domain.ContentDigest{}, errPNPMInspectionLayout
	}
	if err := ctx.Err(); err != nil {
		return domain.ContentDigest{}, errors.Join(err, context.Cause(ctx))
	}
	if err := i.CheckWorkspace(inventories); err != nil {
		return domain.ContentDigest{}, err
	}
	bySubject, err := i.indexInventories(inventories)
	if err != nil {
		return domain.ContentDigest{}, err
	}
	modules, err := parsePNPMLayoutModulesFormat(i, original.layout)
	if err != nil {
		return domain.ContentDigest{}, err
	}
	// The producer's store path is bounded format data. It is never opened,
	// followed or passed to the current guest's store-root authority check.
	if !strings.HasPrefix(modules.StoreDir, "/") || path.Clean(modules.StoreDir) != modules.StoreDir ||
		!strings.HasSuffix(modules.StoreDir, "/v11") || !validInventoryPath(strings.TrimPrefix(modules.StoreDir, "/")) {
		return domain.ContentDigest{}, errPNPMInspectionLayout
	}
	if err := checkPNPMVirtualLock(i, original.layout); err != nil {
		return domain.ContentDigest{}, err
	}
	if _, err := i.checkLayoutFiles(ctx, bySubject, original.layout, modules, original.uid, original.gid, false); err != nil {
		return domain.ContentDigest{}, err
	}
	if _, present := original.layout.files[pnpmWorkspaceCachePath]; present {
		if _, err := discardPNPMWorkspaceCache(original.layout); err != nil {
			return domain.ContentDigest{}, err
		}
	}
	type source struct{ Name, Version, SHA256 string }
	sources := make([]source, 0, len(i.artifacts))
	for _, subject := range i.artifacts {
		sources = append(sources, source{subject.Identity().Name(), subject.Identity().Version(), subject.Digest().String()})
	}
	hash := sha256.Sum256(original.layout.body)
	body, err := json.Marshal(struct {
		Schema, Graph, Census string
		Sources               []source
	}{
		"pnpm-original-source-proof/v1", i.context.snapshot.GraphDigest().String(), hex.EncodeToString(hash[:]), sources,
	})
	if err != nil {
		return domain.ContentDigest{}, err
	}
	if err := ctx.Err(); err != nil {
		return domain.ContentDigest{}, errors.Join(err, context.Cause(ctx))
	}
	hash = sha256.Sum256(body)
	return domain.NewSHA256Digest(hex.EncodeToString(hash[:]))
}
