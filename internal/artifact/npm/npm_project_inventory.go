package npm

import (
	"github.com/rahoney/heliopause/internal/core/domain"
)

// Complete measured inputs only; callers validate their frozen typed context.
func indexNPMProjectInventories(artifacts []domain.AcquiredArtifact, inventories []ArchiveInventory, mismatch error) (map[domain.AcquiredArtifact]ArchiveInventory, error) {
	if len(artifacts) == 0 || len(inventories) != len(artifacts) {
		return nil, mismatch
	}
	expected := make(map[domain.AcquiredArtifact]bool, len(artifacts))
	for _, artifact := range artifacts {
		expected[artifact] = true
	}
	if len(expected) != len(artifacts) {
		return nil, mismatch
	}
	bySubject := make(map[domain.AcquiredArtifact]ArchiveInventory, len(inventories))
	for _, inventory := range inventories {
		if !inventory.Valid() || !expected[inventory.artifact] {
			return nil, mismatch
		}
		if _, duplicate := bySubject[inventory.artifact]; duplicate {
			return nil, mismatch
		}
		bySubject[inventory.artifact] = inventory
	}
	return bySubject, nil
}
