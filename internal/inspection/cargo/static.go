// Package cargo owns bounded non-executing crate archive inspection. Rust
// compilation, build scripts, and macros require the observed build workflow.
package cargo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"sort"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/inspection/content"
)

const checkProfileName = "cargo-crate-static-v1"

type StaticInspector struct{ intakeRoot string }

func NewStaticInspector(intakeRoot string) (*StaticInspector, error) {
	if !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || intakeRoot == "/" {
		return nil, errors.New("cargo inspector intake root is invalid")
	}
	return &StaticInspector{intakeRoot}, nil
}

func (i *StaticInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return i.inspect(ctx, artifact, nil, nil, nil)
}

// InspectSources selects from the already validated bounded crate byte map;
// it neither reparses the archive nor runs Rust or build scripts.
func (i *StaticInspector) InspectSources(ctx context.Context, artifact domain.AcquiredArtifact) ([]ports.SourceAnalysisFile, domain.InspectionReport, error) {
	var sources []ports.SourceAnalysisFile
	report, err := i.inspect(ctx, artifact, &sources, nil, nil)
	if err != nil || len(report.Findings()) != 0 {
		return nil, report, err
	}
	return sources, report, nil
}

// InspectContent retains source DATA and inspects native members from the same
// verified map. It returns a complete census for the separate source analyzer.
func (i *StaticInspector) InspectContent(ctx context.Context, artifact domain.AcquiredArtifact) (content.Inventory, []ports.SourceAnalysisFile, domain.InspectionReport, error) {
	var sources []ports.SourceAnalysisFile
	var binaries []content.BinaryMember
	var inventory content.Inventory
	static, err := i.inspect(ctx, artifact, &sources, &binaries, &inventory)
	if err != nil || len(static.Findings()) != 0 {
		return content.Inventory{}, nil, static, err
	}
	inspection, err := content.NewBinaryInspection(artifact)
	if err != nil {
		return content.Inventory{}, nil, static, err
	}
	for _, member := range binaries {
		_ = inspection.Inspect(ctx, member)
	}
	native, nativeErr := inspection.Report(ctx, inventory)
	if len(native.Executions()) == 0 {
		return content.Inventory{}, nil, static, nativeErr
	}
	census, censusErr := inventory.Report(ctx)
	if censusErr != nil {
		return content.Inventory{}, nil, static, errors.Join(nativeErr, censusErr)
	}
	report, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{static, native, census})
	return inventory, sources, report, errors.Join(nativeErr, err)
}

// InspectAnalyzed runs the fixed source analyzer after the same checked census.
// It is a content entry point; bootstrap/cache activation remains explicit.
func (i *StaticInspector) InspectAnalyzed(ctx context.Context, artifact domain.AcquiredArtifact, analyzer ports.SourceAnalysis) (domain.InspectionReport, error) {
	inventory, sources, report, contentErr := i.InspectContent(ctx, artifact)
	if !inventory.Matches(artifact) {
		return report, contentErr
	}
	source, sourceErr := content.InspectSources(ctx, analyzer, artifact, sources, inventory)
	if len(source.Executions()) == 0 {
		return report, errors.Join(contentErr, sourceErr)
	}
	combined, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{report, source})
	return combined, errors.Join(contentErr, sourceErr, err)
}

func (i *StaticInspector) inspect(ctx context.Context, artifact domain.AcquiredArtifact, sources *[]ports.SourceAnalysisFile, binaries *[]content.BinaryMember, inventory *content.Inventory) (domain.InspectionReport, error) {
	if i == nil || ctx == nil {
		return domain.InspectionReport{}, errors.New("cargo inspection request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	body, err := artifactcargo.ReadIntake(i.intakeRoot, artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	files, archiveErr := artifactcargo.ArchiveFiles(body, artifact.Identity())
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	if archiveErr == nil && sources != nil {
		declared := map[string]bool{}
		if binaries != nil {
			declared, err = artifactcargo.DeclaredSourcePaths(files)
			if err != nil {
				return domain.InspectionReport{}, err
			}
		}
		// ArchiveFiles owns these bytes. Sort selected keys only; no second
		// decompression, body copy or graph charge is needed.
		var names []string
		for name := range files {
			if inventory != nil || declared[name] || content.SourceLanguage(name, files[name]) != "" || binaries != nil && content.IsBinary(name, files[name]) {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		var members []content.MemberFact
		for _, name := range names {
			binary := binaries != nil && content.IsBinary(name, files[name])
			language := content.SourceLanguage(name, files[name])
			if declared[name] {
				language = "rust"
			}
			if binary {
				language = ""
			}
			var digest domain.ContentDigest
			if binary || inventory != nil {
				sum := sha256.Sum256(files[name])
				digest, _ = domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
			}
			if inventory != nil {
				members = append(members, content.MemberFact{Path: name, Size: uint64(len(files[name])), Digest: digest, Language: language, Binary: binary, UnsupportedSource: !binary && language == "" && content.UnsupportedSource(name, files[name], 0)})
			}
			if binary {
				// ArchiveFiles does not retain permission bits. Zero records
				// unavailable archive mode; ELF permissions are decoded inside.
				*binaries = append(*binaries, content.BinaryMember{Path: name, Body: files[name], Digest: digest})
				continue
			}
			if language != "" {
				*sources = append(*sources, ports.SourceAnalysisFile{Path: name, Language: language, Body: files[name]})
			}
		}
		if inventory != nil {
			*inventory, err = content.NewInventory(artifact, members)
			if err != nil {
				return domain.InspectionReport{}, err
			}
		}
	}
	checkID, _ := domain.NewCheckID("cargo-crate-static")
	profile, _ := domain.CheckProfileDigest(checkProfileName, "")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
	evidenceID, _ := domain.NewEvidenceID("cargo-crate-static-result")
	summary := "Bounded archive paths, types, compression integrity and normalized Cargo manifest identity checked without execution. Rust source and build-time execution require the isolated observed compiler workflow."
	if archiveErr != nil {
		summary = "Crate archive validation failed on bounded content or normalized manifest identity."
	}
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "cargo-crate-static", summary)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	var findings []domain.Finding
	if archiveErr != nil {
		finding, err := domain.NewFinding("M12_CARGO_ARCHIVE_INVALID", []domain.EvidenceID{evidenceID})
		if err != nil {
			return domain.InspectionReport{}, err
		}
		findings = append(findings, finding)
	}
	return domain.NewInspectionReport(check, findings, []domain.Evidence{evidence})
}

func (p *StaticInspector) ExpectedChecks(ctx context.Context, artifact domain.AcquiredArtifact, kind domain.CheckKind) ([]domain.CheckRequirement, error) {
	if p == nil || ctx == nil || kind != domain.CheckInspection || artifact.Identity().Source() != artifactcargo.Source() {
		return nil, errors.New("cargo check declaration requires its configured source")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, _ := domain.NewCheckID("cargo-crate-static")
	profile, _ := domain.CheckProfileDigest(checkProfileName, "")
	requirement, err := domain.NewCheckRequirement(id, domain.CheckInspection, true, profile)
	if err != nil {
		return nil, err
	}
	return []domain.CheckRequirement{requirement}, nil
}
