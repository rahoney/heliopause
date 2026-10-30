package npm

import (
	"context"
	"errors"
	"path"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// BindBunNodeExecutionLayout requires the same current whole approval and
// completely inspected native layout. It grants no runtime admission; existing
// Node input readback, immutable mounts and external observation remain required.
func BindBunNodeExecutionLayout(ctx context.Context, execution domain.ExecutionContext, prepared BunInspectionLayout, primary ArchiveInventory) (NodeExecutionLayout, error) {
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
	selected, err := SelectExecutionEntry(primary, execution.Command())
	if err != nil || selected.Target().Path != execution.EntryPath() || selected.Target().Digest != execution.EntryDigest() || uint64(selected.Target().Size) != execution.EntrySize() {
		return NodeExecutionLayout{}, errors.Join(errNodeExecutionLayout, err)
	}
	root, err := prepared.Target(execution.EntryArtifact(), input.context.selection.lock.roots[0].Instance)
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
	layout, err := copyNativeNodeExecutionArchive(ctx, prepared)
	if err != nil {
		return NodeExecutionLayout{}, err
	}
	return bindNodeExecutionLayoutFacts(ctx, execution, layout, entry, "bun")
}
