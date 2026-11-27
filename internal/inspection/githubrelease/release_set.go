package githubrelease

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/releaseinstall"
)

// ReleaseSetInspector selects fixed passive metadata checks or the existing
// native inspector for exact members only. It never executes metadata and
// never reduces the native inspector's required coverage.
type ReleaseSetInspector struct {
	manifest func() (releaseinstall.ReleaseManifest, error)
	metadata *ReleaseManifestInspector
	native   ports.Inspection
	pipx     ports.Inspection
}

func NewReleaseSetInspector(intakeRoot string, manifest func() (releaseinstall.ReleaseManifest, error), native, pipx ports.Inspection) (*ReleaseSetInspector, error) {
	if manifest == nil || native == nil {
		return nil, errors.New("release set inspection requires frozen metadata and native checks")
	}
	metadata, err := NewReleaseManifestInspector(intakeRoot)
	if err != nil {
		return nil, err
	}
	return &ReleaseSetInspector{manifest: manifest, metadata: metadata, native: native, pipx: pipx}, nil
}

func (i *ReleaseSetInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	manifest, err := i.selectedManifest(ctx, artifact)
	if err != nil {
		return domain.InspectionReport{}, err
	}
	switch artifact.Identity().Variant() {
	case "helox-release-manifest.json":
		return i.metadata.Inspect(ctx, artifact)
	case "pipx-runtime-linux-amd64.zip":
		if i.pipx == nil {
			return pipxRuntimeIncomplete("P4_PIPX_RUNTIME_INSPECTION_UNAVAILABLE")
		}
		return i.pipx.Inspect(ctx, artifact)
	case "helox-release-sbom.cdx.json", "helox-runtime-images.json", "pnpm-NOTICE.txt", "bun-NOTICE.txt", "pipx-NOTICE.txt":
		return i.inspectMetadata(ctx, artifact, manifest)
	default:
		return i.native.Inspect(ctx, artifact)
	}
}

func (i *ReleaseSetInspector) inspectMetadata(ctx context.Context, artifact domain.AcquiredArtifact, manifest releaseinstall.ReleaseManifest) (report domain.InspectionReport, resultErr error) {
	limit := int64(1 << 20)
	if artifact.Identity().Variant() == "helox-release-sbom.cdx.json" {
		limit = 8 << 20
	}
	if artifact.Identity().Variant() == "pnpm-NOTICE.txt" || artifact.Identity().Variant() == "bun-NOTICE.txt" || artifact.Identity().Variant() == "pipx-NOTICE.txt" {
		limit = 64 << 10
	}
	if artifact.SizeBytes() == 0 || artifact.SizeBytes() > uint64(limit) {
		return i.invalidMetadata(artifact)
	}
	file, err := i.metadata.static.openAsset(artifact)
	if err != nil {
		return report, err
	}
	defer func() {
		if file.Close() != nil {
			resultErr = errors.Join(resultErr, errors.New("close release metadata input failed"))
		}
	}()
	body, err := io.ReadAll(ports.DownloadReader(ctx, file, limit))
	if err != nil {
		return report, errors.Join(errors.New("read release metadata failed"), ctx.Err())
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	sum := sha256.Sum256(body)
	if uint64(len(body)) != artifact.SizeBytes() || hex.EncodeToString(sum[:]) != artifact.Digest().String() {
		return i.invalidMetadata(artifact)
	}
	if artifact.Identity().Variant() == "helox-release-sbom.cdx.json" {
		err = releaseinstall.ValidateReleaseSBOM(body, manifest)
	} else if artifact.Identity().Variant() == "pnpm-NOTICE.txt" {
		err = releaseinstall.ValidatePNPMNotices(body)
	} else if artifact.Identity().Variant() == "bun-NOTICE.txt" {
		err = releaseinstall.ValidateBunNotices(body)
	} else if artifact.Identity().Variant() == "pipx-NOTICE.txt" {
		err = releaseinstall.ValidatePipxNotices(body)
	} else {
		err = releaseinstall.ValidateRuntimeImagesForRelease(body, manifest)
	}
	if err != nil {
		return i.invalidMetadata(artifact)
	}
	return i.metadata.static.report(artifact, "release-metadata-schema", "Fixed passive release metadata schema and source or runtime identity were checked; native payload checks are separate.", "")
}

func (i *ReleaseSetInspector) invalidMetadata(artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return i.metadata.static.report(artifact, "release-metadata-schema", "Fixed passive release metadata schema or identity is invalid.", "SELF_UPDATE_METADATA_INVALID")
}

func (i *ReleaseSetInspector) ExpectedChecks(ctx context.Context, a domain.AcquiredArtifact, kind domain.CheckKind) ([]domain.CheckRequirement, error) {
	if kind != domain.CheckInspection {
		return nil, errors.New("release declaration kind is invalid")
	}
	if _, err := i.selectedManifest(ctx, a); err != nil {
		return nil, err
	}
	switch a.Identity().Variant() {
	case "helox-release-manifest.json":
		return i.metadata.ExpectedChecks(ctx, a, kind)
	case "pipx-runtime-linux-amd64.zip":
		id, _ := domain.NewCheckID(pipxRuntimeCheck)
		requirement, err := domain.NewCheckRequirement(id, kind, true, githubDynamicProfile(pipxRuntimeCheck))
		return []domain.CheckRequirement{requirement}, err
	case "helox-release-sbom.cdx.json", "helox-runtime-images.json", "pnpm-NOTICE.txt", "bun-NOTICE.txt", "pipx-NOTICE.txt":
		return i.metadata.static.ExpectedChecks(ctx, a, kind)
	default:
		planner, ok := i.native.(ports.CheckPlanner)
		if !ok {
			return nil, errors.New("native release declaration is unavailable")
		}
		return planner.ExpectedChecks(ctx, a, kind)
	}
}

// The same frozen member binding selects both declarations and actual checks.
func (i *ReleaseSetInspector) selectedManifest(ctx context.Context, a domain.AcquiredArtifact) (releaseinstall.ReleaseManifest, error) {
	var empty releaseinstall.ReleaseManifest
	if i == nil || i.manifest == nil || i.metadata == nil || i.native == nil || ctx == nil {
		return empty, errors.New("release set inspection is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	manifest, err := i.manifest()
	if err != nil {
		return empty, err
	}
	if a.Identity().Source().String() != "github-release" || a.Identity().Name() != "rahoney-heliopause" || a.Identity().Version() != manifest.Version() {
		return empty, errors.New("release inspection subject differs from the frozen selection")
	}
	name := a.Identity().Variant()
	if name == "helox-release-manifest.json" {
		if a.Digest().String() != manifest.SHA256() {
			return empty, errors.New("release manifest input differs from the frozen selection")
		}
		return manifest, nil
	}
	present := false
	for _, file := range manifest.Files() {
		if file.Name == name && file.SHA256 == a.Digest().String() && uint64(file.Size) == a.SizeBytes() {
			present = true
			break
		}
	}
	if !present {
		return empty, errors.New("release inspection subject is absent from the manifest")
	}
	switch name {
	case "pipx-runtime-linux-amd64.zip", "helox-release-sbom.cdx.json", "helox-runtime-images.json", "pnpm-NOTICE.txt", "bun-NOTICE.txt", "pipx-NOTICE.txt":
		return manifest, nil
	}
	native, err := manifest.NativeUpdateFiles()
	if err != nil {
		return empty, err
	}
	for _, file := range native {
		if file.Name == name {
			return manifest, nil
		}
	}
	return empty, errors.New("release subject is not selected for this platform")
}
