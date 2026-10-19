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
type GitHubReleasePromotion struct {
	stagingRoot  string
	stateRoot    string
	evidenceRoot string
	evidence     ProjectEvidenceReader
	checkpoint   func(string) error
}

func NewGitHubReleasePromotion(stagingRoot string) (*GitHubReleasePromotion, error) {
	if !filepath.IsAbs(stagingRoot) {
		return nil, errors.New("GitHub Release Promotion requires absolute staging root")
	}
	return &GitHubReleasePromotion{stagingRoot: filepath.Clean(stagingRoot)}, nil
}

func (p *GitHubReleasePromotion) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (result domain.PromotedInstall, resultErr error) {
	if !install.Valid() || !install.RequiresNewTarget() {
		return result, errors.New("ordinary GitHub install requires a new target")
	}
	return p.promote(ctx, staged, bundle, install, nil)
}

func (p *GitHubReleasePromotion) promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext, managed *githubReplacementSession) (result domain.PromotedInstall, resultErr error) {
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
	if p.stateRoot != "" {
		if managed == nil {
			var err error
			managed, err = p.begin(ctx, install, "", false)
			if err != nil {
				return result, err
			}
			defer func() { resultErr = errors.Join(resultErr, managed.Close()) }()
		}
		defer func() {
			if resultErr != nil && !managed.committed {
				resultErr = errors.Join(resultErr, managed.abort())
			}
		}()
		var err error
		managed.selected, managed.selectedBody, err = p.approval(ctx, bundle, install)
		if err != nil {
			return result, err
		}
		if err := managed.verify(ctx); err != nil {
			return result, err
		}
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
	if managed == nil || !managed.replace {
		if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
			return domain.PromotedInstall{}, errors.New("GitHub Release target already exists")
		}
	}
	prefix := "." + filepath.Base(target) + ".haa-"
	if managed != nil {
		prefix = managed.recoveryPrefix()
	}
	temporary, err := os.MkdirTemp(parent, prefix)
	if err != nil {
		return domain.PromotedInstall{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			var err error
			if managed == nil {
				err = os.RemoveAll(temporary)
			} // Managed abort owns cleanup and rollback together.
			if err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("remove incomplete GitHub Release target: %w", err))
			}
		}
	}()
	if managed != nil {
		managed.selectedPath = temporary
		managed.selectedInfo, err = os.Lstat(temporary)
		if err != nil {
			return result, err
		}
	}
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
	if managed != nil {
		if err := managed.recordSelected(); err != nil {
			return result, err
		}
	}
	if err := syncTree(temporary); err != nil {
		return domain.PromotedInstall{}, err
	}
	if managed != nil {
		if err := managed.beforePublish(ctx); err != nil {
			return result, err
		}
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
	if managed != nil {
		managed.selectedPath = target
		managed.targetPublished = true
	}
	if err := syncDirectory(parent); err != nil {
		return domain.PromotedInstall{}, err
	}
	if managed != nil {
		if err := managed.afterPublish(ctx); err != nil {
			return result, err
		}
	}
	return domain.NewPromotedInstall(bundle.ManifestID(), install.Target())
}
