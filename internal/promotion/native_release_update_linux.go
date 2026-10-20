//go:build linux

package promotion

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"time"

	artifactgithub "github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/releaseactivation"
	"github.com/rahoney/heliopause/internal/releaseinstall"
)

// NativeReleaseSource supplies frozen content and copied provenance facts.
// It cannot select an install operation or grant Policy approval.
type NativeReleaseSource interface {
	ports.Artifact
	ports.DependencyResolver
	Reference() domain.ArtifactReference
	ReleaseProof(domain.AcquiredArtifact) (releaseinstall.ReleaseManifest, [][]byte, error)
	DiscardReplayRuns(context.Context) error
	Close() error
}

type NativeReleaseContent struct {
	Source       NativeReleaseSource
	Verification ports.Verification
	Inspection   ports.Inspection
}

// NativeReleaseFactory only composes ports. Production construction is inert;
// the existing resolver performs acquisition after this session owns the update.
type NativeReleaseFactory func(version, activeRuntimeLockSHA256 string, install domain.InstallContext) (NativeReleaseContent, error)

type NativeReleaseUpdatePreparation struct {
	lease       *releaseactivation.RuntimeLease
	stagingRoot string
	factory     NativeReleaseFactory
	drain       func() error
}

func NewNativeReleaseUpdatePreparation(lease *releaseactivation.RuntimeLease, stagingRoot string, factory NativeReleaseFactory, drain func() error) (*NativeReleaseUpdatePreparation, error) {
	if lease == nil {
		return nil, NativeReleaseUpdateFailure(releaseactivation.ErrUnmanaged)
	}
	if !filepath.IsAbs(stagingRoot) || filepath.Clean(stagingRoot) != stagingRoot || factory == nil || drain == nil {
		return nil, NativeReleaseUpdateFailure(releaseactivation.ErrUntrusted)
	}
	return &NativeReleaseUpdatePreparation{lease, stagingRoot, factory, drain}, nil
}

func (p *NativeReleaseUpdatePreparation) BeginInstallUpdate(ctx context.Context, source domain.SourceID, version string, install domain.InstallContext, adopt bool) (session ports.InstallUpdateSession, resultErr error) {
	if p == nil || ctx == nil || source.String() != "github-release" || !install.Valid() || install.Mode() != domain.InstallGitHubAsset || install.Target().String() != releaseinstall.SystemInstallRoot || adopt || artifactgithub.ValidateHeliopauseVersion(version) != nil {
		return nil, NativeReleaseUpdateFailure(releaseactivation.ErrUntrusted)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	owner, err := releaseactivation.AcquireUpdateOwner(ctx, p.lease)
	if err != nil {
		return nil, NativeReleaseUpdateFailure(err)
	}
	defer func() {
		if session == nil {
			resultErr = errors.Join(resultErr, NativeReleaseUpdateFailure(owner.Close()))
		}
	}()
	old, err := p.lease.Installed()
	if err != nil {
		return nil, NativeReleaseUpdateFailure(err)
	}
	content, err := p.factory(version, old.Manifest().RuntimeLockSHA256(), install)
	if err != nil || content.Source == nil || content.Verification == nil || content.Inspection == nil {
		if content.Source != nil {
			err = errors.Join(err, content.Source.Close())
		}
		return nil, NativeReleaseUpdateFailure(errors.Join(releaseactivation.ErrUntrusted, err))
	}
	expected, err := artifactgithub.ParseReference("rahoney/heliopause@" + version + "#helox-release-manifest.json")
	if err != nil || content.Source.Reference() != expected {
		return nil, NativeReleaseUpdateFailure(errors.Join(releaseactivation.ErrUntrusted, err, content.Source.Close()))
	}
	promoter, err := NewNativeReleasePromotion(p.stagingRoot, p.lease, content.Source.ReleaseProof, func() error {
		ports.ReportProgress(ctx, ports.Progress{Stage: "drain"})
		if err := p.drain(); err != nil {
			return ports.NewInstallUpdateFailure("SELF_UPDATE_RUNTIME_CLEANUP_FAILED", err)
		}
		ports.ReportProgress(ctx, ports.Progress{Stage: "activate"})
		return nil
	})
	if err != nil {
		return nil, NativeReleaseUpdateFailure(errors.Join(err, content.Source.Close()))
	}
	return &nativeReleaseUpdateSession{NativeReleaseSource: content.Source, Verification: content.Verification, Inspection: content.Inspection, promoter: promoter, owner: owner, drain: p.drain}, nil
}

type nativeReleaseUpdateSession struct {
	NativeReleaseSource
	ports.Verification
	ports.Inspection
	promoter  *NativeReleasePromotion
	owner     *releaseactivation.UpdateOwner
	drain     func() error
	closeOnce sync.Once
	closeErr  error
}

var _ ports.InstallUpdateContentSession = (*nativeReleaseUpdateSession)(nil)

func (s *nativeReleaseUpdateSession) Promote(ctx context.Context, staged domain.StagedSet, bundle domain.VerifiedBundle, install domain.InstallContext) (domain.PromotedInstall, error) {
	result, err := s.promoter.Promote(ctx, staged, bundle, install)
	return result, NativeReleaseUpdateFailure(err)
}

func (s *nativeReleaseUpdateSession) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		var failures []error
		if err := s.drain(); err != nil {
			// Unknown execution can still reference replay/staging payloads.
			failures = append(failures, ports.NewInstallUpdateFailure("SELF_UPDATE_RUNTIME_CLEANUP_FAILED", err))
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := errors.Join(s.promoter.Close(), s.NativeReleaseSource.DiscardReplayRuns(ctx)); err != nil {
				failures = append(failures, errors.Join(releaseactivation.ErrCleanup, err))
			}
			cancel()
		}
		if err := errors.Join(s.NativeReleaseSource.Close(), s.owner.Close()); err != nil {
			failures = append(failures, errors.Join(releaseactivation.ErrCleanup, err))
		}
		s.closeErr = NativeReleaseUpdateFailure(errors.Join(failures...))
	})
	return s.closeErr
}

// NativeReleaseUpdateFailure exposes only controller-selected stable reasons.
// Original causes remain available; arbitrary helper/candidate output is not a code.
func NativeReleaseUpdateFailure(err error) error {
	if err == nil {
		return nil
	}
	if ports.InstallUpdateFailureCode(err) != "" {
		return err
	}
	for _, reason := range []error{releaseactivation.ErrTransition, releaseactivation.ErrBusy, releaseactivation.ErrUnmanaged, releaseactivation.ErrCaller, releaseactivation.ErrUntrusted, releaseactivation.ErrProtocol, releaseactivation.ErrHealth, releaseactivation.ErrInstaller, releaseactivation.ErrCleanup} {
		if errors.Is(err, reason) {
			return ports.NewInstallUpdateFailure(reason.Error(), err)
		}
	}
	return ports.NewInstallUpdateFailure("SELF_UPDATE_FAILED", err)
}
