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
}

func NewReleaseSetInspector(intakeRoot string, manifest func() (releaseinstall.ReleaseManifest, error), native ports.Inspection) (*ReleaseSetInspector, error) {
	if manifest == nil || native == nil {
		return nil, errors.New("release set inspection requires frozen metadata and native checks")
	}
	metadata, err := NewReleaseManifestInspector(intakeRoot)
	if err != nil {
		return nil, err
	}
	return &ReleaseSetInspector{manifest, metadata, native}, nil
}

func (i *ReleaseSetInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	if i == nil || i.manifest == nil || i.metadata == nil || i.native == nil || ctx == nil {
		return domain.InspectionReport{}, errors.New("release set inspection is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return domain.InspectionReport{}, err
	}
	manifest, err := i.manifest()
	if err != nil {
		return domain.InspectionReport{}, err
	}
	if artifact.Identity().Source().String() != "github-release" || artifact.Identity().Name() != "rahoney-heliopause" || artifact.Identity().Version() != manifest.Version() {
		return domain.InspectionReport{}, errors.New("release inspection subject differs from the frozen selection")
	}
	name := artifact.Identity().Variant()
	if name == "helox-release-manifest.json" {
		if manifest.SHA256() != artifact.Digest().String() {
			return domain.InspectionReport{}, errors.New("release manifest input differs from the frozen selection")
		}
		return i.metadata.Inspect(ctx, artifact)
	}
	var selected releaseinstall.ReleaseFile
	for _, file := range manifest.Files() {
		if file.Name == name {
			selected = file
			break
		}
	}
	if selected.Name == "" || selected.SHA256 != artifact.Digest().String() || uint64(selected.Size) != artifact.SizeBytes() {
		return domain.InspectionReport{}, errors.New("release inspection subject is absent from the manifest")
	}
	if name == "helox-release-sbom.cdx.json" || name == "helox-runtime-images.json" {
		return i.inspectMetadata(ctx, artifact, manifest)
	}
	native, err := manifest.NativeUpdateFiles()
	if err != nil {
		return domain.InspectionReport{}, err
	}
	for _, file := range native {
		if file.Name == name {
			return i.native.Inspect(ctx, artifact)
		}
	}
	return domain.InspectionReport{}, errors.New("release subject is not selected for this platform")
}

func (i *ReleaseSetInspector) inspectMetadata(ctx context.Context, artifact domain.AcquiredArtifact, manifest releaseinstall.ReleaseManifest) (report domain.InspectionReport, resultErr error) {
	limit := int64(1 << 20)
	if artifact.Identity().Variant() == "helox-release-sbom.cdx.json" {
		limit = 8 << 20
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
	} else {
		err = releaseinstall.ValidateRuntimeImages(body)
	}
	if err != nil {
		return i.invalidMetadata(artifact)
	}
	return i.metadata.static.report(artifact, "release-metadata-schema", "Fixed passive release metadata schema and source or runtime identity were checked; native payload checks are separate.", "")
}

func (i *ReleaseSetInspector) invalidMetadata(artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return i.metadata.static.report(artifact, "release-metadata-schema", "Fixed passive release metadata schema or identity is invalid.", "SELF_UPDATE_METADATA_INVALID")
}
