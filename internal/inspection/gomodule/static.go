// Package gomodule performs bounded, non-executing Go module source inspection.
package gomodule

import (
	"context"
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
)

type StaticInspector struct{ intakeRoot string }

// NewStaticInspector binds the exact controlled intake without accepting a project path.
func NewStaticInspector(intakeRoot string) (*StaticInspector, error) {
	if !filepath.IsAbs(intakeRoot) || filepath.Clean(intakeRoot) != intakeRoot || intakeRoot == "/" {
		return nil, errors.New("go inspector intake root is invalid")
	}
	return &StaticInspector{intakeRoot: intakeRoot}, nil
}

func (i *StaticInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
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
		if !strings.HasSuffix(file.Name, ".go") {
			continue
		}
		r, err := file.Open()
		if err != nil {
			return report(artifact, "M12_GO_MODULE_ARCHIVE_INVALID")
		}
		body, readErr := io.ReadAll(io.LimitReader(r, artifactgo.MaxModuleFileBytes+1))
		closeErr := r.Close()
		if readErr != nil || closeErr != nil || len(body) > artifactgo.MaxModuleFileBytes {
			return report(artifact, "M12_GO_MODULE_ARCHIVE_INVALID")
		}
		// Go's package discovery excludes directories named exactly testdata.
		// These bytes are still authenticated, bounded and read above (including
		// ZIP integrity). This is syntax applicability, never execution authority:
		// an explicit package/import may select them and must pass the compiler
		// in the observed build. No package or vendor name grants an exemption.
		rel := strings.TrimPrefix(file.Name, artifact.Identity().Name()+"@"+artifact.Identity().Version()+"/")
		if isTestData(rel) {
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
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
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
