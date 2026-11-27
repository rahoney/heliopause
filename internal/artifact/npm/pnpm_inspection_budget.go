package npm

import (
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// PNPMInspectionWorkspaceLimit is the fixed inspection workspace capacity.
// Both native and Artifact project volumes consume this one owner. It is not
// a release version, a user override or permission to exceed runtime budgets.
const PNPMInspectionWorkspaceLimit = 128 << 20

var errPNPMInspectionBudget = errors.New("pnpm complete inspection input exceeds its fixed workspace budget")

func (i PNPMInspectionInput) indexInventories(inventories []ArchiveInventory) (map[domain.AcquiredArtifact]ArchiveInventory, error) {
	if !i.Valid() {
		return nil, errPNPMInspectionLayout
	}
	return indexNPMProjectInventories(i.artifacts, inventories, errPNPMInspectionLayout)
}

// CheckWorkspace admits only complete, exact static inputs and charges archive
// bytes, the unique expanded store and every logical peer instance. Native
// hardlink deduplication never increases this authorization. Generated metadata
// and bins still live under the hard volume ceiling; admission is not proof
// that native preparation can complete, nor a source or Policy approval.
func (i PNPMInspectionInput) CheckWorkspace(inventories []ArchiveInventory) error {
	bySubject, err := i.indexInventories(inventories)
	if err != nil {
		return err
	}
	budget, err := i.BeginWorkspace()
	if err != nil {
		return err
	}
	for _, inventory := range bySubject {
		if err := budget.Add(inventory); err != nil {
			return err
		}
	}
	return budget.Complete()
}

// PNPMInspectionBudget charges measured inventories during static collection.
// It retains no archive payload and grants neither authenticity nor execution.
// Any invalid addition permanently invalidates the collection.
type PNPMInspectionBudget = npmInspectionBudget

// npmInspectionBudget is the shared bounded accounting engine, not an approval.
type npmInspectionBudget struct {
	expected        map[domain.AcquiredArtifact]int
	seen            map[domain.AcquiredArtifact]bool
	charged         uint64
	uniqueEntries   int
	instanceEntries int
	failure         error
}

// BeginWorkspace charges every compressed subject before archive collection.
// Expanded bytes and logical peer copies are charged immediately by Add.
func (i PNPMInspectionInput) BeginWorkspace() (*PNPMInspectionBudget, error) {
	if !i.Valid() || len(i.artifacts) == 0 {
		return nil, errPNPMInspectionLayout
	}
	instances := make([]string, 0, len(i.context.selection.lock.snapshots))
	for _, instance := range i.context.selection.lock.snapshots {
		instances = append(instances, instance.Package)
	}
	return beginNPMInspectionBudget(i.artifacts, instances)
}

func beginNPMInspectionBudget(artifacts []domain.AcquiredArtifact, instancePackages []string) (*npmInspectionBudget, error) {
	budget := &PNPMInspectionBudget{expected: make(map[domain.AcquiredArtifact]int, len(artifacts)), seen: make(map[domain.AcquiredArtifact]bool, len(artifacts))}
	byPackage := make(map[string]domain.AcquiredArtifact, len(artifacts))
	for _, subject := range artifacts {
		if !budget.charge(subject.SizeBytes()) {
			return nil, errPNPMInspectionBudget
		}
		byPackage[subject.Identity().Name()+"@"+subject.Identity().Version()] = subject
		budget.expected[subject] = 0
	}
	for _, instance := range instancePackages {
		subject, present := byPackage[instance]
		if !present {
			return nil, errPNPMInspectionLayout
		}
		budget.expected[subject]++
	}
	for _, count := range budget.expected {
		if count == 0 {
			return nil, errPNPMInspectionLayout
		}
	}
	return budget, nil
}

func (b *npmInspectionBudget) charge(amount uint64) bool {
	if amount > PNPMInspectionWorkspaceLimit-b.charged {
		return false
	}
	b.charged += amount
	return true
}

// Add rejects a duplicate, foreign or excessive inventory before it is retained
// by the caller. Failed counters cannot later attest complete coverage.
func (b *npmInspectionBudget) Add(inventory ArchiveInventory) error {
	if b == nil || b.expected == nil {
		return errPNPMInspectionLayout
	}
	if b.failure != nil {
		return b.failure
	}
	count, present := b.expected[inventory.artifact]
	if !inventory.Valid() || !present || b.seen[inventory.artifact] {
		b.failure = errPNPMInspectionLayout
		return b.failure
	}
	entries := len(inventory.entries)
	if entries > pnpmLayoutEntryLimit-b.uniqueEntries || entries > (pnpmLayoutEntryLimit-b.instanceEntries)/count {
		b.failure = errPNPMInspectionBudget
		return b.failure
	}
	b.uniqueEntries += entries
	b.instanceEntries += entries * count
	for _, entry := range inventory.entries {
		if entry.Directory {
			continue
		}
		// Charge the unique store plus each logical instance without overflow.
		for copies := 0; copies <= count; copies++ {
			if !b.charge(uint64(entry.Size)) {
				b.failure = errPNPMInspectionBudget
				return b.failure
			}
		}
	}
	b.seen[inventory.artifact] = true
	return nil
}

// Complete requires all exact subjects and no previous collection failure.
func (b *npmInspectionBudget) Complete() error {
	if b == nil || b.expected == nil {
		return errPNPMInspectionLayout
	}
	if b.failure != nil {
		return b.failure
	}
	if len(b.seen) != len(b.expected) {
		return errPNPMInspectionLayout
	}
	return nil
}
