package promotion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// GitHubReleasePromotion publishes one already-staged standalone asset. It
// never resolves or downloads a Release and never grants target authority to a sandbox.
type GitHubReleasePromotion struct{ stagingRoot string }

func NewGitHubReleasePromotion(stagingRoot string) (*GitHubReleasePromotion, error) {
	if !filepath.IsAbs(stagingRoot) {
		return nil, errors.New("GitHub Release Promotion requires absolute staging root")
	}
	return &GitHubReleasePromotion{stagingRoot: filepath.Clean(stagingRoot)}, nil
}

func (p *GitHubReleasePromotion) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (result domain.PromotedInstall, resultErr error) {
	if ctx == nil || ctx.Err() != nil || p == nil || !bundle.Valid() || staged.ManifestID() != bundle.ManifestID() || staged.Handle() != "staging:"+bundle.ManifestID().String() || verifyDocuments(bundle) != nil {
		return domain.PromotedInstall{}, errors.New("GitHub Release staged bundle binding is invalid")
	}
	inspections := bundle.Set().Inspected().Inspections()
	if len(inspections) != 1 {
		return domain.PromotedInstall{}, errors.New("GitHub Release Promotion requires exactly one inspected asset")
	}
	inspection := inspections[0]
	artifact := inspection.Artifact()
	if artifact.Identity().Source().String() != "github-release" || artifact.Identity().Variant() == "" || inspection.PolicyDecision().Decision() != domain.DecisionAllow {
		return domain.PromotedInstall{}, errors.New("GitHub Release Promotion asset is invalid")
	}
	stagedRoot := filepath.Join(p.stagingRoot, bundle.ManifestID().String())
	if filepath.Dir(stagedRoot) != p.stagingRoot || rejectSymlinkPath(stagedRoot) != nil || verifyStagedRecords(stagedRoot, bundle) != nil {
		return domain.PromotedInstall{}, errors.New("GitHub Release staged records are unavailable")
	}
	target := install.Target().String()
	parent := filepath.Dir(target)
	if trustedExistingDirectory(parent) != nil {
		return domain.PromotedInstall{}, errors.New("GitHub Release target parent is untrusted")
	}
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		return domain.PromotedInstall{}, errors.New("GitHub Release target already exists")
	}
	temporary, err := os.MkdirTemp(parent, "."+filepath.Base(target)+".haa-")
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			if err := os.RemoveAll(temporary); err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("remove incomplete GitHub Release target: %w", err))
			}
		}
	}()
	source := filepath.Join(stagedRoot, "artifacts", artifact.Digest().String()+".asset")
	destination := filepath.Join(temporary, artifact.Identity().Variant())
	if filepath.Base(destination) != artifact.Identity().Variant() {
		return domain.PromotedInstall{}, errors.New("GitHub Release target filename is invalid")
	}
	if err := copyDigestFile(source, destination, artifact.Digest().String()); err != nil {
		return domain.PromotedInstall{}, err
	}
	mode := os.FileMode(0o400)
	for _, check := range inspection.Checks() {
		if check.ID().String() == "github-release-elf-dynamic" && check.Status() == domain.ExecutionCompleted {
			mode = 0o500
		}
	}
	if err := os.Chmod(destination, mode); err != nil {
		return domain.PromotedInstall{}, err
	}
	if err := syncTree(temporary); err != nil {
		return domain.PromotedInstall{}, err
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		return domain.PromotedInstall{}, errors.New("GitHub Release target appeared before publish")
	}
	currentParent, err := os.Stat(parent)
	if err != nil || !os.SameFile(parentInfo, currentParent) || trustedExistingDirectory(parent) != nil {
		return domain.PromotedInstall{}, errors.New("GitHub Release target parent changed before publish")
	}
	if err := renameNoReplace(temporary, target); err != nil {
		return domain.PromotedInstall{}, err
	}
	cleanup = false
	if err := syncDirectory(parent); err != nil {
		return domain.PromotedInstall{}, err
	}
	return domain.NewPromotedInstall(bundle.ManifestID(), install.Target())
}
