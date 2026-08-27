package pypi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// StaticInspector reads an acquired PyPI distribution only from trusted intake.
type StaticInspector struct {
	intakeRoot string
	target     artifactpypi.WheelTarget
}

func NewStaticInspector(intakeRoot string, target artifactpypi.WheelTarget) (*StaticInspector, error) {
	if !filepath.IsAbs(intakeRoot) || target.Python == "" || target.ABI == "" || target.Platform == "" {
		return nil, errors.New("PyPI static inspector configuration is invalid")
	}
	return &StaticInspector{intakeRoot: filepath.Clean(intakeRoot), target: target}, nil
}

func (i *StaticInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	_, report, err := i.inspect(ctx, artifact)
	return report, err
}

func (i *StaticInspector) InspectWheel(ctx context.Context, artifact domain.AcquiredArtifact) (artifactpypi.WheelInspection, domain.InspectionReport, error) {
	inspection, report, err := i.inspect(ctx, artifact)
	if err != nil || artifact.Identity().Variant() != "wheel" {
		return artifactpypi.WheelInspection{}, report, err
	}
	wheel, ok := inspection.(artifactpypi.WheelInspection)
	if !ok {
		// Finding-only static results intentionally have no typed inspection.
		// Preserve them so CompositeInspector can fail closed before dynamic
		// inspection instead of masking the concrete finding.
		if len(report.Findings()) != 0 {
			return artifactpypi.WheelInspection{}, report, nil
		}
		return artifactpypi.WheelInspection{}, domain.InspectionReport{}, errors.New("PyPI wheel static inspection is unavailable")
	}
	return wheel, report, nil
}

func (i *StaticInspector) InspectSdist(ctx context.Context, artifact domain.AcquiredArtifact) (artifactpypi.SdistInspection, error) {
	inspection, _, err := i.inspect(ctx, artifact)
	if err != nil || artifact.Identity().Variant() != "sdist" {
		return artifactpypi.SdistInspection{}, err
	}
	sdist, ok := inspection.(artifactpypi.SdistInspection)
	if !ok {
		return artifactpypi.SdistInspection{}, errors.New("PyPI sdist static inspection is unavailable")
	}
	return sdist, nil
}

func (i *StaticInspector) inspect(ctx context.Context, artifact domain.AcquiredArtifact) (any, domain.InspectionReport, error) {
	if ctx == nil || ctx.Err() != nil || i == nil {
		return nil, domain.InspectionReport{}, errors.New("PyPI static inspection request is invalid")
	}
	if _, ok := artifactpypi.ProfileForSource(artifact.Identity().Source()); !ok {
		return nil, domain.InspectionReport{}, errors.New("PyPI static inspection source is unsupported")
	}
	path, filename, err := i.artifactPath(artifact)
	if err != nil {
		return nil, domain.InspectionReport{}, err
	}
	declared, ok := artifact.DeclaredIntegrity()
	if !ok || !strings.HasPrefix(declared, "sha256:") {
		return nil, domain.InspectionReport{}, errors.New("PyPI static inspection lacks declared SHA-256")
	}
	declared = strings.TrimPrefix(declared, "sha256:")
	file, err := os.Open(path)
	if err != nil {
		return nil, domain.InspectionReport{}, errors.New("open controlled PyPI distribution")
	}
	defer file.Close()
	var summary string
	switch artifact.Identity().Variant() {
	case "wheel", "derived-wheel":
		resourcePolicy := artifactpypi.ResourcePolicyFromContext(ctx)
		info, inspectErr := artifactpypi.InspectWheelForSource(file, int64(artifact.SizeBytes()), filename, declared, i.target, resourcePolicy.WheelLimits(), artifact.Identity().Source())
		if inspectErr != nil {
			return nil, i.violation(artifact, "M5_WHEEL_STATIC_INVALID"), nil
		}
		var uncompressed int64
		for _, item := range info.Files {
			if item.Size < 0 || uncompressed > resourcePolicy.WheelLimits().MaxUncompressed-item.Size {
				return nil, i.violation(artifact, "M5_WHEEL_RESOURCE_EXCEEDED"), nil
			}
			uncompressed += item.Size
		}
		if artifactpypi.ChargeUncompressedFromContext(ctx, uncompressed) != nil {
			return nil, i.violation(artifact, "M5_WHEEL_RESOURCE_EXCEEDED"), nil
		}
		summary = fmt.Sprintf("PyPI wheel static inspection completed with %d RECORD entries.", len(info.Files))
		return info, i.complete(artifact, "pypi-wheel-static", summary), nil
	case "sdist":
		info, inspectErr := artifactpypi.InspectSdist(file, filename, declared, artifactpypi.DefaultSdistLimits())
		if inspectErr != nil {
			return nil, i.violation(artifact, "M5_SDIST_STATIC_INVALID"), nil
		}
		summary = "PyPI PEP 517 source distribution static inspection completed."
		return info, i.complete(artifact, "pypi-sdist-static", summary), nil
	default:
		return nil, domain.InspectionReport{}, errors.New("PyPI distribution variant is unsupported")
	}
}

func (i *StaticInspector) artifactPath(artifact domain.AcquiredArtifact) (string, string, error) {
	parts := strings.Split(artifact.ContentHandle(), ":")
	if len(parts) != 3 || parts[0] != "intake" || parts[2] != artifact.Identity().Variant() {
		return "", "", errors.New("PyPI intake handle is invalid")
	}
	runID, err := domain.ParseRunID(parts[1])
	if err != nil {
		return "", "", errors.New("PyPI intake handle is invalid")
	}
	directory := filepath.Join(i.intakeRoot, runID.String())
	filenameRecord := "filename"
	if artifact.Identity().Variant() == "derived-wheel" {
		filenameRecord = "derived-filename"
	}
	filenameBytes, err := os.ReadFile(filepath.Join(directory, filenameRecord))
	filename := string(filenameBytes)
	if err != nil || filename == "" || strings.ContainsAny(filename, `/\\`) {
		return "", "", errors.New("PyPI intake filename is invalid")
	}
	variant, ok := pypiVariant(filename)
	if !ok || (variant != artifact.Identity().Variant() && !(variant == "wheel" && artifact.Identity().Variant() == "derived-wheel")) {
		return "", "", errors.New("PyPI intake filename does not match Artifact")
	}
	name := map[string]string{"wheel": "wheel.whl", "derived-wheel": "derived.whl", "sdist": "sdist.tar.gz"}[variant]
	if artifact.Identity().Variant() == "derived-wheel" {
		name = "derived.whl"
	}
	if name == "" {
		return "", "", errors.New("PyPI intake variant is unsupported")
	}
	return filepath.Join(directory, name), filename, nil
}

func pypiVariant(filename string) (string, bool) {
	if strings.HasSuffix(filename, ".whl") {
		return "wheel", true
	}
	if strings.HasSuffix(filename, ".tar.gz") {
		return "sdist", true
	}
	return "", false
}

func (i *StaticInspector) complete(artifact domain.AcquiredArtifact, kind, summary string) domain.InspectionReport {
	checkID, _ := domain.NewCheckID("pypi-static-archive")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("pypi-static-archive-result")
	evidence, _ := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), kind, summary)
	report, _ := domain.NewInspectionReport(check, nil, []domain.Evidence{evidence})
	return report
}

func (i *StaticInspector) violation(artifact domain.AcquiredArtifact, code string) domain.InspectionReport {
	checkID, _ := domain.NewCheckID("pypi-static-archive")
	check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	evidenceID, _ := domain.NewEvidenceID("pypi-static-archive-result")
	evidence, _ := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "pypi-static-archive", "PyPI static archive inspection rejected the distribution.")
	finding, _ := domain.NewFinding(code, []domain.EvidenceID{evidenceID})
	report, _ := domain.NewInspectionReport(check, []domain.Finding{finding}, []domain.Evidence{evidence})
	return report
}
