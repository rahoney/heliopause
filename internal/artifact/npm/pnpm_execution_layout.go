package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"path"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// BindPNPMNodeExecutionLayout requires the same current whole approval and
// completely inspected native layout. It grants no runtime admission; existing
// Node input readback, immutable mounts and external observation remain required.
func BindPNPMNodeExecutionLayout(ctx context.Context, execution domain.ExecutionContext, prepared PNPMInspectionLayout, primary ArchiveInventory) (NodeExecutionLayout, error) {
	if ctx == nil || !execution.Valid() || !prepared.input.context.MatchesExecutionResolution(execution.Resolution()) || !primary.Matches(execution.EntryArtifact()) {
		return NodeExecutionLayout{}, errNodeExecutionLayout
	}
	acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{}
	for _, inspection := range execution.Set().Inspected().Inspections() {
		acquired[inspection.Node()] = inspection.Artifact()
	}
	input, err := prepared.input.context.BindExecutionAcquired(execution.Resolution().Graph(), acquired)
	if err != nil || !prepared.MatchesInput(input) {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	selected, err := SelectPNPMExecutionEntry(primary, execution.Command())
	if err != nil || selected.Target().Path != execution.EntryPath() || selected.Target().Digest != execution.EntryDigest() || uint64(selected.Target().Size) != execution.EntrySize() {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	root, err := prepared.Target(execution.EntryArtifact(), input.context.selection.lock.roots[0].Snapshot)
	if err != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	entry := path.Join(root, execution.EntryPath())
	body, err := prepared.ReadFile(entry)
	if err != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	if _, err := selected.BindNodeTarget(body); err != nil {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	layout, err := copyPNPMExecutionArchive(ctx, prepared)
	if err != nil {
		return NodeExecutionLayout{}, err
	}
	return bindNodeExecutionLayoutFacts(ctx, execution, layout, entry, true)
}

// Expand measured hardlinks into separate regular copies before introduction.
// Relative symlinks retain the exact topology already checked by BindLayout.
func copyPNPMExecutionArchive(ctx context.Context, prepared PNPMInspectionLayout) (PNPMLayoutArchive, error) {
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	for _, entry := range prepared.Entries() {
		if ctx.Err() != nil {
			return PNPMLayoutArchive{}, errors.Join(errNodeExecutionLayout, ctx.Err(), context.Cause(ctx))
		}
		var body []byte
		if entry.Kind == tar.TypeReg || entry.Kind == tar.TypeLink {
			var err error
			body, err = prepared.ReadFile(entry.Name)
			if err != nil || len(body) > PNPMInspectionWorkspaceLimit-output.Len() {
				return PNPMLayoutArchive{}, errors.Join(errNodeExecutionLayout, err)
			}
			entry.Kind, entry.Link, entry.Size = tar.TypeReg, "", int64(len(body))
		}
		if err := writer.WriteHeader(&tar.Header{Name: entry.Name, Typeflag: entry.Kind, Mode: entry.Mode, Size: entry.Size, Linkname: entry.Link, Uid: entry.UID, Gid: entry.GID}); err != nil {
			return PNPMLayoutArchive{}, errors.Join(errNodeExecutionLayout, err)
		}
		if _, err := writer.Write(body); err != nil || output.Len() > PNPMInspectionWorkspaceLimit {
			return PNPMLayoutArchive{}, errors.Join(errNodeExecutionLayout, err)
		}
	}
	if err := writer.Close(); err != nil || output.Len() > PNPMInspectionWorkspaceLimit {
		return PNPMLayoutArchive{}, errors.Join(errNodeExecutionLayout, err)
	}
	return ParsePNPMLayoutArchive(ctx, output.Bytes())
}
