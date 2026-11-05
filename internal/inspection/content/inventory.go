package content

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
)

// MemberFact is measured by the existing archive owner after complete bounded
// validation. It is controller data, never a scanner/Artifact completion claim.
// Empty Language marks DATA; UnsupportedSource explicitly prevents zero-source
// applicability. Native DATA must not also be declared as parser source.
type MemberFact struct {
	Path              string
	Size              uint64
	Digest            domain.ContentDigest
	Language          string
	Binary            bool
	UnsupportedSource bool
}

const (
	SourceCoverageEvidenceID        = "source-content-coverage"
	SourceSignalsEvidenceID         = "source-content-signals-summary"
	ArchiveBinaryCoverageEvidenceID = "binary-members-summary-0"
	ArchiveBinarySignalsEvidenceID  = "binary-members-summary-1"
	InventoryEvidenceID             = "content-inventory-summary"
)

// EvidenceRequirement is a readonly declaration for current archive approval.
// Consumers compare metadata; they do not run the analyzer or Policy.
type EvidenceRequirement struct {
	ID         domain.EvidenceID
	Check      domain.CheckRequirement
	Kind       string
	ExtraKinds []string
}

func ArchiveEvidenceRequirements() ([]EvidenceRequirement, error) {
	checks := make(map[string]domain.CheckRequirement)
	for _, declare := range []func() ([]domain.CheckRequirement, error){SourceChecks, BinaryChecks, InventoryChecks} {
		declared, err := declare()
		if err != nil {
			return nil, err
		}
		for _, check := range declared {
			checks[check.ID().String()] = check
		}
	}
	var result []EvidenceRequirement
	for _, item := range []struct {
		id, check, kind string
		extras          []string
	}{
		{SourceCoverageEvidenceID, "source-content", "source-content-coverage", nil},
		{SourceSignalsEvidenceID, "source-content-signals", "source-content-signals", []string{"source-content-signal"}},
		{ArchiveBinaryCoverageEvidenceID, "binary-content", "binary-content-inventory", []string{"binary-content-coverage"}},
		{ArchiveBinarySignalsEvidenceID, "binary-content-signals", "binary-content-inventory", []string{"binary-content-signals"}},
		{InventoryEvidenceID, "content-inventory", "content-inventory", nil},
	} {
		id, err := domain.NewEvidenceID(item.id)
		if err != nil {
			return nil, err
		}
		check, known := checks[item.check]
		if !known {
			return nil, errors.New("archive Evidence declaration has no matching check")
		}
		result = append(result, EvidenceRequirement{id, check, item.kind, item.extras})
	}
	return result, nil
}

// InventoryLimits are supplied by the archive owner, whose complete parser
// already enforces them. They do not alter the separate source parser budgets.
type InventoryLimits struct {
	MaxMembers int
	MaxBytes   uint64
}

// Inventory is an immutable census for one exact acquired subject. Its
// constructor validates facts; the archive owner owns byte authenticity and
// supplies every regular member only after whole-archive validation.
type Inventory struct {
	limits      InventoryLimits
	artifact    domain.AcquiredArtifact
	members     map[string]MemberFact
	digest      string
	sources     int
	binaries    int
	unsupported int
}

func NewInventory(artifact domain.AcquiredArtifact, members []MemberFact, limits ...InventoryLimits) (Inventory, error) {
	selected := InventoryLimits{ports.SourceAnalysisTargetLimit, ports.SourceAnalysisTotalLimit}
	if len(limits) > 1 {
		return Inventory{}, errors.New("content inventory accepts one archive-owner bound")
	}
	if len(limits) == 1 {
		selected = limits[0]
	}
	if artifact.Digest().String() == "" || selected.MaxMembers <= 0 || selected.MaxBytes == 0 || len(members) == 0 || len(members) > selected.MaxMembers {
		return Inventory{}, errors.New("content inventory subject or member count is invalid")
	}
	result := Inventory{artifact: artifact, limits: selected, members: make(map[string]MemberFact, len(members))}
	var total uint64
	digest := sha256.New()
	for _, member := range members {
		_, duplicate := result.members[member.Path]
		if member.Path == "" || len(member.Path) > 4096 || member.Path != path.Clean(member.Path) || path.IsAbs(member.Path) || member.Path == "." || member.Path == ".." || strings.HasPrefix(member.Path, "../") || strings.ContainsAny(member.Path, "\\\x00\r\n") || duplicate || member.Digest.String() == "" || member.Size > selected.MaxBytes-total || member.Binary && (member.Language != "" || member.UnsupportedSource) || member.Language != "" && member.UnsupportedSource {
			return Inventory{}, errors.New("content inventory has invalid, duplicate or excessive member facts")
		}
		switch member.Language {
		case "":
		case "javascript", "typescript", "python", "go", "rust", "hcl", "bash":
			result.sources++
		default:
			return Inventory{}, errors.New("content inventory parser is undeclared")
		}
		if member.Binary {
			result.binaries++
		}
		if member.UnsupportedSource {
			result.unsupported++
		}
		total += member.Size
		result.members[member.Path] = member
		if err := json.NewEncoder(digest).Encode([]any{member.Path, member.Size, member.Digest.String(), member.Language, member.Binary, member.UnsupportedSource}); err != nil {
			return Inventory{}, err
		}
	}
	result.digest = hex.EncodeToString(digest.Sum(nil))
	return result, nil
}

func (i Inventory) Matches(artifact domain.AcquiredArtifact) bool {
	return i.digest != "" && len(i.members) != 0 && i.artifact == artifact
}

func (i Inventory) validateSources(artifact domain.AcquiredArtifact, files []ports.SourceAnalysisFile) error {
	if !i.Matches(artifact) || len(files) != i.sources {
		return errors.New("source selection differs from the complete content inventory")
	}
	seen := make(map[string]bool, len(files))
	for _, file := range files {
		member, exists := i.members[file.Path]
		sum := sha256.Sum256(file.Body)
		if !exists || seen[file.Path] || member.Language == "" || member.Language != file.Language || member.Size != uint64(len(file.Body)) || member.Digest.String() != hex.EncodeToString(sum[:]) {
			return errors.New("source DATA differs from its measured inventory member")
		}
		seen[file.Path] = true
	}
	return nil
}

func InventoryChecks() ([]domain.CheckRequirement, error) {
	id, _ := domain.NewCheckID("content-inventory")
	profile, _ := domain.CheckProfileDigest("content-inventory-v2", "complete-source+native+unsupported+data;archive-owner-bounds;separate-source-budgets")
	check, err := domain.NewCheckRequirement(id, domain.CheckInspection, true, profile)
	return []domain.CheckRequirement{check}, err
}

func (i Inventory) Report(ctx context.Context) (domain.InspectionReport, error) {
	if ctx == nil || !i.Matches(i.artifact) {
		return domain.InspectionReport{}, errors.New("complete content inventory is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	checks, err := InventoryChecks()
	if err != nil {
		return domain.InspectionReport{}, err
	}
	check := checks[0]
	execution, _ := domain.NewCheckExecution(check.ID(), domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", check.ProfileDigest())
	summary, err := json.Marshal(struct {
		Schema      string `json:"schema"`
		Inventory   string `json:"inventory_sha256"`
		Members     int    `json:"members"`
		Sources     int    `json:"sources"`
		Binaries    int    `json:"binaries"`
		Unsupported int    `json:"unsupported_sources"`
		MaxMembers  int    `json:"max_members"`
		MaxBytes    uint64 `json:"max_bytes"`
	}{"content-inventory-v2", i.digest, len(i.members), i.sources, i.binaries, i.unsupported, i.limits.MaxMembers, i.limits.MaxBytes})
	if err != nil {
		return domain.InspectionReport{}, err
	}
	id, _ := domain.NewEvidenceID(InventoryEvidenceID)
	fact, err := domain.NewEvidence(id, check.ID(), i.artifact.Identity(), i.artifact.Digest(), "content-inventory", string(summary), check.ProfileDigest())
	if err != nil {
		return domain.InspectionReport{}, err
	}
	return domain.NewInspectionReport(execution, nil, []domain.Evidence{fact})
}
