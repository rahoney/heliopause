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

type StaticInspector struct{ intakeRoot string }

// NewStaticInspector binds the exact controlled intake without accepting a project path.
func NewStaticInspector(intakeRoot string) (*StaticInspector, error) {
	if !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || intakeRoot == "/" {
		return nil, errors.New("go inspector intake root is invalid")
	}
	return &StaticInspector{intakeRoot: intakeRoot}, nil
}

func (i *StaticInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return i.inspect(ctx, artifact, nil, nil)
}

// InspectSources retains parser DATA in the existing checked member pass.
// Successful structure inspection alone is not a content-analysis verdict.
func (i *StaticInspector) InspectSources(ctx context.Context, artifact domain.AcquiredArtifact) ([]ports.SourceAnalysisFile, domain.InspectionReport, error) {
	var sources []ports.SourceAnalysisFile
	report, err := i.inspect(ctx, artifact, &sources, nil)
	if err != nil || len(report.Findings()) != 0 {
		return nil, report, err
	}
	return sources, report, nil
}

// InspectContent collects source DATA and native facts in the same checked ZIP
// pass. Go syntax and archive validation remain prerequisites for publication.
func (i *StaticInspector) InspectContent(ctx context.Context, artifact domain.AcquiredArtifact) ([]ports.SourceAnalysisFile, domain.InspectionReport, error) {
	native, err := content.NewBinaryInspection(artifact)
	if err != nil {
		return nil, domain.InspectionReport{}, err
	}
	var sources []ports.SourceAnalysisFile
	static, err := i.inspect(ctx, artifact, &sources, native)
	if err != nil || len(static.Findings()) != 0 {
		return nil, static, err
	}
	nativeReport, nativeErr := native.Report(ctx)
	if len(nativeReport.Executions()) == 0 {
		return nil, static, nativeErr
	}
	report, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{static, nativeReport})
	return sources, report, errors.Join(nativeErr, err)
}

func (i *StaticInspector) inspect(ctx context.Context, artifact domain.AcquiredArtifact, sources *[]ports.SourceAnalysisFile, native *content.BinaryInspection) (domain.InspectionReport, error) {
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
	files, err := artifactgo.CheckedFiles(reader, artifact.Identity())
	if err != nil {
		return report(artifact, "M12_GO_MODULE_ARCHIVE_INVALID")
	}
	for _, file := range files {
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
		payload := io.MultiReader(bytes.NewReader(prefix), r)
		binaryMember := native != nil && content.IsBinary(file.Name, prefix)
		if !goSource && language == "" && !binaryMember {
			n, readErr := io.Copy(io.Discard, io.LimitReader(payload, artifactgo.MaxModuleFileBytes+1))
			closeErr := r.Close()
			if readErr != nil || closeErr != nil || uint64(n) != file.UncompressedSize64 {
				return report(artifact, "M12_GO_MODULE_ARCHIVE_INVALID")
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
		rel := strings.TrimPrefix(file.Name, artifact.Identity().Name()+"@"+artifact.Identity().Version()+"/")
		if binaryMember {
			sum := sha256.Sum256(body)
			digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
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
	return []domain.CheckRequirement{requirement}, nil
}
