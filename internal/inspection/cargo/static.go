// Package cargo owns bounded non-executing crate archive inspection. Rust
// compilation, build scripts, and macros require the observed build workflow.
package cargo

import (
	"context"
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
	return i.inspect(ctx, artifact, nil)
}

// InspectSources selects from the already validated bounded crate byte map;
// it neither reparses the archive nor runs Rust or build scripts.
func (i *StaticInspector) InspectSources(ctx context.Context, artifact domain.AcquiredArtifact) ([]ports.SourceAnalysisFile, domain.InspectionReport, error) {
	var sources []ports.SourceAnalysisFile
	report, err := i.inspect(ctx, artifact, &sources)
	if err != nil || len(report.Findings()) != 0 {
		return nil, report, err
	}
	return sources, report, nil
}

func (i *StaticInspector) inspect(ctx context.Context, artifact domain.AcquiredArtifact, sources *[]ports.SourceAnalysisFile) (domain.InspectionReport, error) {
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
		// ArchiveFiles owns the validated map; order its selected keys only
		// so the parser's input and coverage digest are deterministic.
		var names []string
		for name := range files {
			if content.SourceLanguage(name, files[name]) != "" {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			*sources = append(*sources, ports.SourceAnalysisFile{Path: name, Language: content.SourceLanguage(name, files[name]), Body: files[name]})
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
