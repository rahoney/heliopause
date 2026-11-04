// Package gomodule performs bounded, non-executing Go module source inspection.
package gomodule

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"

	artifactgo "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/inspection/content"
)

const checkProfileName = "go-module-static-v1"

type StaticInspector struct {
	intakeRoot string
	analyzer   ports.SourceAnalysis
}

// NewStaticInspector binds the exact controlled intake without accepting a project path.
func NewStaticInspector(intakeRoot string, analyzers ...ports.SourceAnalysis) (*StaticInspector, error) {
	if !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || intakeRoot == "/" {
		return nil, errors.New("go inspector intake root is invalid")
	}
	if len(analyzers) > 1 || len(analyzers) == 1 && analyzers[0] == nil {
		return nil, errors.New("go content inspection requires one configured analyzer")
	}
	inspector := &StaticInspector{intakeRoot: intakeRoot}
	if len(analyzers) == 1 {
		inspector.analyzer = analyzers[0]
	}
	return inspector, nil
}

func (i *StaticInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i != nil && i.analyzer != nil {
		return i.InspectAnalyzed(ctx, artifact, i.analyzer)
	}
	return i.inspect(ctx, artifact, nil, nil, nil)
}

// InspectSources retains parser DATA in the existing checked member pass.
// Successful structure inspection alone is not a content-analysis verdict.
func (i *StaticInspector) InspectSources(ctx context.Context, artifact domain.AcquiredArtifact) ([]ports.SourceAnalysisFile, domain.InspectionReport, error) {
	var sources []ports.SourceAnalysisFile
	report, err := i.inspect(ctx, artifact, &sources, nil, nil)
	if err != nil || len(report.Findings()) != 0 {
		return nil, report, err
	}
	return sources, report, nil
}

// InspectContent collects source DATA and native facts in the same checked ZIP
// pass. Go syntax and archive validation remain prerequisites for publication.
func (i *StaticInspector) InspectContent(ctx context.Context, artifact domain.AcquiredArtifact) (content.Inventory, []ports.SourceAnalysisFile, domain.InspectionReport, error) {
	native, err := content.NewBinaryInspection(artifact)
	if err != nil {
		return content.Inventory{}, nil, domain.InspectionReport{}, err
	}
	var sources []ports.SourceAnalysisFile
	var inventory content.Inventory
	static, err := i.inspect(ctx, artifact, &sources, native, &inventory)
	if err != nil || len(static.Findings()) != 0 {
		return content.Inventory{}, nil, static, err
	}
	nativeReport, nativeErr := native.Report(ctx, inventory)
	if len(nativeReport.Executions()) == 0 {
		return content.Inventory{}, nil, static, nativeErr
	}
	census, censusErr := inventory.Report(ctx)
	if censusErr != nil {
		return content.Inventory{}, nil, static, errors.Join(nativeErr, censusErr)
	}
	report, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{static, nativeReport, census})
	return inventory, sources, report, errors.Join(nativeErr, err)
}

// InspectAnalyzed composes the existing fixed source analyzer with the same
// authenticated inventory. Neither parser output nor file names grant approval.
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

func (i *StaticInspector) inspect(ctx context.Context, artifact domain.AcquiredArtifact, sources *[]ports.SourceAnalysisFile, native *content.BinaryInspection, inventory *content.Inventory) (domain.InspectionReport, error) {
	if ctx == nil || ctx.Err() != nil || i == nil {
		return domain.InspectionReport{}, errors.New("go module static inspection request is invalid")
	}
	bundle, err := artifactgo.ReadIntake(i.intakeRoot, artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	mod, err := modfile.Parse("go.mod", bundle.Mod(), nil)
	if err != nil || mod.Module == nil || mod.Module.Mod.Path != artifact.Identity().Name() {
		return report(artifact, "M12_GO_MODULE_IDENTITY_MISMATCH")
	}
	for _, control := range bundle.RequiredMods() {
		mod, err := modfile.Parse("go.mod", control.Body, nil)
		if err != nil || mod.Module == nil || mod.Module.Mod.Path != artifact.Identity().Name() {
			return report(artifact, "M12_GO_MODULE_IDENTITY_MISMATCH")
		}
	}
	reader, err := bundle.ZipReader()
	if err != nil {
		return report(artifact, "M12_GO_MODULE_ARCHIVE_INVALID")
	}
	_, err = artifactgo.CheckedFiles(reader, artifact.Identity())
	if err != nil {
		return report(artifact, "M12_GO_MODULE_ARCHIVE_INVALID")
	}
	var members []content.MemberFact
	// CheckedFiles validates this exact ZIP order; do not hash a map iteration.
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return domain.InspectionReport{}, err
		}
		language := ""
		if sources != nil {
			language = content.SourceLanguage(file.Name)
		}
		goSource := strings.HasSuffix(file.Name, ".go")
		if !goSource && language == "" && sources == nil {
			continue
		}
		r, err := file.Open()
		if err != nil {
			return report(artifact, "M12_GO_MODULE_ARCHIVE_INVALID")
		}
		var prefix []byte
		if sources != nil {
			prefix = make([]byte, min(file.UncompressedSize64, 512))
			if _, err := io.ReadFull(r, prefix); err != nil {
				_ = r.Close()
				return report(artifact, "M12_GO_MODULE_ARCHIVE_INVALID")
			}
			language = content.SourceLanguage(file.Name, prefix)
		}
		rel := strings.TrimPrefix(file.Name, artifact.Identity().Name()+"@"+artifact.Identity().Version()+"/")
		payload := io.MultiReader(bytes.NewReader(prefix), r)
		binaryMember := native != nil && content.IsBinary(file.Name, prefix)
		if !goSource && language == "" && !binaryMember {
			digest := sha256.New()
			var destination io.Writer = io.Discard
			if inventory != nil {
				destination = digest
			}
			n, readErr := io.Copy(destination, io.LimitReader(payload, artifactgo.MaxModuleFileBytes+1))
			closeErr := r.Close()
			if readErr != nil || closeErr != nil || uint64(n) != file.UncompressedSize64 {
				return report(artifact, "M12_GO_MODULE_ARCHIVE_INVALID")
			}
			if inventory != nil {
				sum, _ := domain.NewSHA256Digest(hex.EncodeToString(digest.Sum(nil)))
				members = append(members, content.MemberFact{Path: rel, Size: file.UncompressedSize64, Digest: sum, UnsupportedSource: content.UnsupportedSource(rel, prefix, uint32(file.Mode().Perm()))})
			}
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(payload, artifactgo.MaxModuleFileBytes+1))
		closeErr := r.Close()
		if readErr != nil || closeErr != nil || len(body) > artifactgo.MaxModuleFileBytes || uint64(len(body)) != file.UncompressedSize64 {
			return report(artifact, "M12_GO_MODULE_ARCHIVE_INVALID")
		}
		// Go's package discovery excludes directories named exactly testdata.
		// These bytes are still authenticated, bounded and read above (including
		// ZIP integrity). This is syntax applicability, never execution authority:
		// an explicit package/import may select them and must pass the compiler
		// in the observed build. No package or vendor name grants an exemption.
		var digest domain.ContentDigest
		if native != nil || inventory != nil {
			sum := sha256.Sum256(body)
			digest, _ = domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		}
		if inventory != nil {
			memberLanguage := language
			if binaryMember {
				memberLanguage = ""
			}
			members = append(members, content.MemberFact{Path: rel, Size: file.UncompressedSize64, Digest: digest, Language: memberLanguage, Binary: binaryMember})
		}
		if binaryMember {
			_ = native.Inspect(ctx, content.BinaryMember{Path: rel, Body: body, Mode: uint32(file.Mode().Perm()), Digest: digest})
		}
		if sources != nil && language != "" && !binaryMember {
			*sources = append(*sources, ports.SourceAnalysisFile{Path: rel, Language: language, Body: body})
		}
		if !goSource || isTestData(rel) {
			continue
		}
		// Parsing never imports or executes project code. Comments and directives
		// remain data; go generate and arbitrary tool execution are not invoked.
		if _, err := parser.ParseFile(token.NewFileSet(), "module.go", body, parser.AllErrors|parser.SkipObjectResolution); err != nil {
			return report(artifact, "M12_GO_SOURCE_INVALID")
		}
	}
	if inventory != nil {
		*inventory, err = content.NewInventory(artifact, members, content.InventoryLimits{MaxMembers: artifactgo.MaxModuleFiles, MaxBytes: artifactgo.MaxModuleExpandedBytes})
		if err != nil {
			return domain.InspectionReport{}, err
		}
	}
	return report(artifact, "")
}

func isTestData(relative string) bool {
	for _, directory := range strings.Split(path.Dir(relative), "/") {
		if directory == "testdata" {
			return true
		}
	}
	return false
}

func report(artifact domain.AcquiredArtifact, code string) (domain.InspectionReport, error) {
	checkID, _ := domain.NewCheckID("go-module-static")
	profile, _ := domain.CheckProfileDigest(checkProfileName, "")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "", profile)
	evidenceID, _ := domain.NewEvidenceID("go-module-static-result")
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "go-module-static", "Bounded controlled archive and Go source syntax inspected without execution. Testdata assets retain archive integrity checks; explicit selection requires compiler checks in the observed build.")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	var findings []domain.Finding
	if code != "" {
		finding, err := domain.NewFinding(code, []domain.EvidenceID{evidenceID})
		if err != nil {
			return domain.InspectionReport{}, err
		}
		findings = append(findings, finding)
	}
	return domain.NewInspectionReport(check, findings, []domain.Evidence{evidence})
}

func (i *StaticInspector) ExpectedChecks(ctx context.Context, artifact domain.AcquiredArtifact, kind domain.CheckKind) ([]domain.CheckRequirement, error) {
	if i == nil || ctx == nil || kind != domain.CheckInspection || artifact.Identity().Source() != artifactgo.Source() {
		return nil, errors.New("go static declaration is invalid")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, _ := domain.NewCheckID("go-module-static")
	profile, _ := domain.CheckProfileDigest(checkProfileName, "")
	requirement, err := domain.NewCheckRequirement(id, kind, true, profile)
	if err != nil {
		return nil, err
	}
	result := []domain.CheckRequirement{requirement}
	if i.analyzer != nil {
		for _, declare := range []func() ([]domain.CheckRequirement, error){content.BinaryChecks, content.InventoryChecks, content.SourceChecks} {
			checks, err := declare()
			if err != nil {
				return nil, err
			}
			result = append(result, checks...)
		}
	}
	return result, nil
}
