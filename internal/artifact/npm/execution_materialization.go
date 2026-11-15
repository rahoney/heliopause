package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// BindMaterializedNodeExecutionLayout discards only npm's bounded root cache
// metadata. Its contents never authorize or enter execution. All remaining
// files still require the original complete source/entry binding.
func BindMaterializedNodeExecutionLayout(ctx context.Context, execution domain.ExecutionContext, inventories []ArchiveInventory, native PNPMLayoutArchive) (NodeExecutionLayout, error) {
	if ctx == nil || !execution.Valid() || len(native.entries) == 0 {
		return NodeExecutionLayout{}, errNodeExecutionLayout
	}
	if ctx.Err() != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, ctx.Err(), context.Cause(ctx))
	}
	cleaned, err := stripNPMHiddenLockCache(ctx, native, false)
	if err != nil {
		return NodeExecutionLayout{}, err
	}
	return BindNodeExecutionLayout(ctx, execution, inventories, cleaned)
}

// Cache metadata is bounded transport data, never source/Policy authority.
// npm projects retain verified bin links; the old execution lane rejects them.
func stripNPMHiddenLockCache(ctx context.Context, native PNPMLayoutArchive, allowBinLinks bool) (PNPMLayoutArchive, error) {
	const cacheName = "node_modules/.package-lock.json"
	present := false
	for _, entry := range native.entries {
		if entry.Name != cacheName {
			continue
		}
		if entry.Kind != tar.TypeReg || entry.Size > 4<<20 || !pnpmLayoutEntryModeOwner(entry, pnpmLayoutRequirement{kind: tar.TypeReg}, 1000, 1000) {
			return PNPMLayoutArchive{}, errNodeExecutionLayout
		}
		present = true
	}
	if !present {
		return native, nil
	}
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	for _, entry := range native.entries {
		if ctx.Err() != nil {
			return PNPMLayoutArchive{}, errors.Join(errNodeExecutionLayout, ctx.Err(), context.Cause(ctx))
		}
		if entry.Name == cacheName {
			continue
		}
		if entry.Kind != tar.TypeReg && entry.Kind != tar.TypeDir && !(allowBinLinks && entry.Kind == tar.TypeSymlink) || entry.Size < 0 || entry.Size > int64(PNPMLayoutArchiveLimit-output.Len()) {
			return PNPMLayoutArchive{}, errNodeExecutionLayout
		}
		var body []byte
		if entry.Kind == tar.TypeReg {
			var err error
			body, err = native.ReadFile(entry.Name)
			if err != nil {
				return PNPMLayoutArchive{}, errors.Join(errNodeExecutionLayout, err)
			}
		}
		header := &tar.Header{Name: entry.Name, Typeflag: entry.Kind, Mode: entry.Mode, Size: entry.Size, Uid: entry.UID, Gid: entry.GID, Linkname: entry.Link}
		if err := writer.WriteHeader(header); err != nil {
			return PNPMLayoutArchive{}, errors.Join(errNodeExecutionLayout, err)
		}
		if _, err := writer.Write(body); err != nil {
			return PNPMLayoutArchive{}, errors.Join(errNodeExecutionLayout, err)
		}
		if output.Len() > PNPMLayoutArchiveLimit {
			return PNPMLayoutArchive{}, errNodeExecutionLayout
		}
	}
	if err := writer.Close(); err != nil {
		return PNPMLayoutArchive{}, errors.Join(errNodeExecutionLayout, err)
	}
	cleaned, err := ParsePNPMLayoutArchive(ctx, output.Bytes())
	if err != nil {
		return PNPMLayoutArchive{}, errors.Join(errNodeExecutionLayout, err)
	}
	return cleaned, nil
}
