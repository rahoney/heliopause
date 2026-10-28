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
	const cacheName = "node_modules/.package-lock.json"
	present := false
	for _, entry := range native.entries {
		if entry.Name != cacheName {
			continue
		}
		if entry.Kind != tar.TypeReg || entry.Size > 4<<20 || !pnpmLayoutEntryModeOwner(entry, pnpmLayoutRequirement{kind: tar.TypeReg}, 1000, 1000) {
			return NodeExecutionLayout{}, errNodeExecutionLayout
		}
		present = true
	}
	if !present {
		return BindNodeExecutionLayout(ctx, execution, inventories, native)
	}
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	for _, entry := range native.entries {
		if ctx.Err() != nil {
			return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, ctx.Err(), context.Cause(ctx))
		}
		if entry.Name == cacheName {
			continue
		}
		if entry.Kind != tar.TypeReg && entry.Kind != tar.TypeDir || entry.Size < 0 || entry.Size > int64(PNPMLayoutArchiveLimit-output.Len()) {
			return NodeExecutionLayout{}, errNodeExecutionLayout
		}
		var body []byte
		if entry.Kind == tar.TypeReg {
			var err error
			body, err = native.ReadFile(entry.Name)
			if err != nil {
				return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
			}
		}
		header := &tar.Header{Name: entry.Name, Typeflag: entry.Kind, Mode: entry.Mode, Size: entry.Size, Uid: entry.UID, Gid: entry.GID}
		if err := writer.WriteHeader(header); err != nil {
			return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
		}
		if _, err := writer.Write(body); err != nil {
			return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
		}
		if output.Len() > PNPMLayoutArchiveLimit {
			return NodeExecutionLayout{}, errNodeExecutionLayout
		}
	}
	if err := writer.Close(); err != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	cleaned, err := ParsePNPMLayoutArchive(ctx, output.Bytes())
	if err != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	return BindNodeExecutionLayout(ctx, execution, inventories, cleaned)
}
