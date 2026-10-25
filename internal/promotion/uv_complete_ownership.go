package promotion

import (
	"context"
	"errors"
	"os"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// The existing UV complete-original proof is shared by pip sync and project
// publication. It updates only held prestate; approval/publication stay separate.
func prepareUVCompleteOwnership(ctx context.Context, tx *pypiVenvTransaction, original pypiInstalledSnapshot, install domain.InstallContext, native *pypiNativeEnrollment, adopt bool, verify func(context.Context) error) (domain.ProjectControlFile, error) {
	metadata := original.metadataFiles()
	managed, err := managedPyPIInventoryFromMetadata(tx, metadata)
	if err != nil {
		return domain.ProjectControlFile{}, ports.NewInstallUpdateFailure("UPDATE_OWNERSHIP_INVALID", err)
	}
	if len(metadata) > len(managed) {
		if !adopt {
			return domain.ProjectControlFile{}, ports.NewInstallUpdateFailure("UPDATE_ADOPTION_REQUIRED", errors.New("complete sync requires ownership of every original distribution"))
		}
		if native == nil {
			return domain.ProjectControlFile{}, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", errors.New("UV sync original enrollment collaborators are unavailable"))
		}
		inventory, err := capturePyPINativeInventoryFromMetadata(ctx, tx, metadata)
		if err != nil {
			return domain.ProjectControlFile{}, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", err)
		}
		selection, err := native.selectAllOwnership(ctx, install, inventory)
		if err != nil {
			return domain.ProjectControlFile{}, ports.NewInstallUpdateFailure("UPDATE_ORIGINAL_STATE_UNSUPPORTED", err)
		}
		if len(selection.destinations) == 0 || !selection.control.Present() {
			return domain.ProjectControlFile{}, errors.New("complete original ownership proof is missing")
		}
		for _, directory := range selection.absentDirectories {
			if err := tx.freezeParents(directory); err != nil {
				return domain.ProjectControlFile{}, err
			}
			if _, err := os.Lstat(directory); !errors.Is(err, os.ErrNotExist) {
				return domain.ProjectControlFile{}, errors.New("UV sync original cache namespace appeared")
			}
		}
		if err := verify(ctx); err != nil {
			return domain.ProjectControlFile{}, err
		}
		for _, destination := range selection.destinations {
			if _, exists := tx.current.Files[destination.key()]; exists {
				return domain.ProjectControlFile{}, errors.New("UV sync original proof overlaps another owner")
			}
			tx.current.Files[destination.key()] = destination
		}
		return selection.control, nil
	}
	return domain.ProjectControlFile{}, nil
}
