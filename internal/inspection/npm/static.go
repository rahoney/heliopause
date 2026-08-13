// Package npm performs bounded static inspection of controlled npm tarballs.
package npm

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

const (
	decodedLimit  = 200 << 20
	entryLimit    = 10000
	fileLimit     = 20 << 20
	manifestLimit = 1 << 20
)

// Inspector reads only the controlled Intake root and never extracts an archive.
type Inspector struct{ intakeRoot string }

// NewInspector creates an npm static inspector for one explicit Intake root.
func NewInspector(intakeRoot string) (*Inspector, error) {
	if !filepath.IsAbs(intakeRoot) {
		return nil, errors.New("npm intake root must be absolute")
	}
	return &Inspector{intakeRoot: filepath.Clean(intakeRoot)}, nil
}

// Inspect validates archive structure and package manifest without executing or extracting content.
func (i *Inspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if ctx == nil {
		return domain.InspectionReport{}, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	if i == nil || artifact.Identity().Source().String() != "npm" {
		return domain.InspectionReport{}, errors.New("npm static inspection requires an acquired npm Artifact")
	}
	path, err := i.tarballPath(artifact.ContentHandle())
	if err != nil {
		return domain.InspectionReport{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return domain.InspectionReport{}, fmt.Errorf("open controlled npm tarball: %w", err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return i.violationReport(artifact, "M2_ARCHIVE_INVALID", "npm tarball gzip structure is invalid.")
	}
	defer gzipReader.Close()
	reader := tar.NewReader(io.LimitReader(gzipReader, decodedLimit+1))
	seen := map[string]bool{}
	entries, manifests := 0, 0
	for {
		header, nextErr := reader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return i.violationReport(artifact, "M2_ARCHIVE_INVALID", "npm tarball archive structure is invalid.")
		}
		entries++
		if entries > entryLimit {
			return i.violationReport(artifact, "M2_ARCHIVE_LIMIT_EXCEEDED", "npm tarball entry limit was exceeded.")
		}
		name, valid := validEntryPath(header.Name)
		if !valid || seen[name] {
			return i.violationReport(artifact, "M2_ARCHIVE_PATH_INVALID", "npm tarball contains an unsafe or duplicate entry path.")
		}
		seen[name] = true
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			return i.violationReport(artifact, "M2_ARCHIVE_TYPE_INVALID", "npm tarball contains an unsupported entry type.")
		}
		if header.Size > fileLimit {
			return i.violationReport(artifact, "M2_ARCHIVE_LIMIT_EXCEEDED", "npm tarball file-size limit was exceeded.")
		}
		if name == "package/package.json" {
			manifests++
			if manifests != 1 {
				return i.violationReport(artifact, "M2_MANIFEST_INVALID", "npm tarball contains an invalid package manifest.")
			}
			body, readErr := readBounded(reader, manifestLimit)
			if readErr != nil {
				return i.violationReport(artifact, "M2_MANIFEST_INVALID", "npm package manifest is invalid or exceeds its limit.")
			}
			if err := validateManifest(body, artifact.Identity()); err != nil {
				return i.violationReport(artifact, "M2_MANIFEST_INVALID", "npm package manifest does not match the resolved artifact identity.")
			}
		}
	}
	if manifests != 1 {
		return i.violationReport(artifact, "M2_MANIFEST_INVALID", "npm tarball must contain exactly one package manifest.")
	}
	return i.report(artifact, nil, fmt.Sprintf("npm static archive inspection completed with %d entries.", entries))
}

func (i *Inspector) tarballPath(handle string) (string, error) {
	parts := strings.Split(handle, ":")
	if len(parts) != 3 || parts[0] != "intake" || parts[2] != "tarball" {
		return "", errors.New("npm content handle is invalid")
	}
	runID, err := domain.ParseRunID(parts[1])
	if err != nil {
		return "", errors.New("npm content handle is invalid")
	}
	return filepath.Join(i.intakeRoot, runID.String(), "tarball.tgz"), nil
}

func validEntryPath(value string) (string, bool) {
	if value == "" || len(value) > 4096 || strings.HasPrefix(value, "/") || !strings.HasPrefix(value, "package/") {
		return "", false
	}
	clean := path.Clean(value)
	return clean, clean == value && clean != "package" && !strings.HasPrefix(clean, "../")
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, errors.New("bounded read failed")
	}
	return b, nil
}

func validateManifest(body []byte, identity domain.ResolvedArtifactIdentity) error {
	var manifest struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil || manifest.Name != identity.Name() || manifest.Version != identity.Version() {
		return errors.New("manifest identity mismatch")
	}
	return nil
}

func (i *Inspector) violationReport(artifact domain.AcquiredArtifact, code, summary string) (domain.InspectionReport, error) {
	return i.report(artifact, []string{code}, summary)
}

func (i *Inspector) report(artifact domain.AcquiredArtifact, codes []string, summary string) (domain.InspectionReport, error) {
	checkID, err := domain.NewCheckID("npm-static-archive")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	execution, err := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	evidenceID, err := domain.NewEvidenceID("npm-static-archive-result")
	if err != nil {
		return domain.InspectionReport{}, err
	}
	evidence, err := domain.NewEvidence(evidenceID, checkID, artifact.Identity(), artifact.Digest(), "npm-static-archive", summary)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	findings := make([]domain.Finding, 0, len(codes))
	for _, code := range codes {
		finding, findingErr := domain.NewFinding(code, []domain.EvidenceID{evidenceID})
		if findingErr != nil {
			return domain.InspectionReport{}, findingErr
		}
		findings = append(findings, finding)
	}
	return domain.NewInspectionReport(execution, findings, []domain.Evidence{evidence})
}
