package sandbox

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// GoBuildSourceReader reuses the private input reader's anchored, bounded
// archive/hash checks without creating a runtime or interpreting security facts.
type GoBuildSourceReader struct{ intake string }

func NewGoBuildSourceReader(intake string) (*GoBuildSourceReader, error) {
	if !filepath.IsAbs(intake) || filepath.Clean(intake) != intake || intake == "/" {
		return nil, errGoBuildInput
	}
	return &GoBuildSourceReader{intake}, nil
}

func (r *GoBuildSourceReader) VerifyBuildSource(ctx context.Context, source domain.AcquiredArtifact) (resultErr error) {
	if r == nil || ctx == nil {
		return errGoBuildInput
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	input, err := openGoBuildInput(r.intake, source, "project")
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, input.close()) }()
	manifest := &goBuildInputManifest{members: map[string]goBuildInputMember{}, controls: map[string][]byte{}, storage: 16 << 20}
	return input.scan(ctx, manifest, nil)
}
