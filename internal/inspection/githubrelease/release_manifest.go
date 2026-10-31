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

// ReleaseManifestInspector inspects only the fixed passive release manifest.
// It does not broaden the ordinary GitHub inspector's supported formats, and
// never substitutes metadata inspection for any native executable's checks.
type ReleaseManifestInspector struct{ static *StaticInspector }

func NewReleaseManifestInspector(intakeRoot string) (*ReleaseManifestInspector, error) {
	static, err := NewStaticInspector(intakeRoot)
	if err != nil {
		return nil, err
	}
	return &ReleaseManifestInspector{static}, nil
}

func (i *ReleaseManifestInspector) Inspect(ctx context.Context, artifact domain.AcquiredArtifact) (report domain.InspectionReport, resultErr error) {
	if i == nil || i.static == nil || ctx == nil || artifact.Identity().Source().String() != "github-release" || artifact.Identity().Name() != "rahoney-heliopause" || artifact.Identity().Variant() != "helox-release-manifest.json" {
		return report, errors.New("official release manifest inspection request is invalid")
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if artifact.SizeBytes() == 0 || artifact.SizeBytes() > 1<<20 {
		return i.invalid(artifact)
	}
	file, err := i.static.openAsset(artifact)
	if err != nil {
		return report, err
	}
	defer func() {
		if file.Close() != nil {
			resultErr = errors.Join(resultErr, errors.New("release manifest close failed"))
		}
	}()
	body, err := io.ReadAll(ports.DownloadReader(ctx, file, 1<<20))
	if err != nil {
		return report, errors.Join(errors.New("release manifest read failed"), ctx.Err())
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	sum := sha256.Sum256(body)
	if uint64(len(body)) != artifact.SizeBytes() || hex.EncodeToString(sum[:]) != artifact.Digest().String() {
		return i.invalid(artifact)
	}
	value, err := releaseinstall.ParseReleaseManifest(body)
	if err != nil || value.Version() != artifact.Identity().Version() {
		return i.invalid(artifact)
	}
	return i.static.report(artifact, "release-manifest-schema", "Official release manifest schema and fixed role bindings were checked; native payload inspection is separate.", "")
}

func (i *ReleaseManifestInspector) invalid(artifact domain.AcquiredArtifact) (domain.InspectionReport, error) {
	return i.static.report(artifact, "release-manifest-schema", "Official release manifest schema or fixed role bindings are invalid.", "SELF_UPDATE_MANIFEST_INVALID")
}
