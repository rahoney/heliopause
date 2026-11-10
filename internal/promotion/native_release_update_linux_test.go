//go:build linux

package promotion

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/releaseactivation"
	"github.com/rahoney/heliopause/internal/releaseinstall"
)

type nativeSessionCloseSource struct {
	NativeReleaseSource
	calls                *[]string
	discardErr, closeErr error
}

func (s nativeSessionCloseSource) DiscardReplayRuns(context.Context) error {
	*s.calls = append(*s.calls, "discard")
	return s.discardErr
}
func (s nativeSessionCloseSource) Close() error {
	*s.calls = append(*s.calls, "source-close")
	return s.closeErr
}

func TestNativeUpdateSessionConfirmsDrainBeforeDiscardAndPreservesFirstCloseResult(t *testing.T) {
	for _, kind := range []string{"success", "unknown-runtime", "discard-failed", "source-close-failed"} {
		t.Run(kind, func(t *testing.T) {
			f := newNativePromotionFixture(t, "success")
			p := &NativeReleasePromotion{stagingRoot: filepath.Join(f.root, "staging"), proof: func(domain.AcquiredArtifact) (releaseinstall.ReleaseManifest, [][]byte, error) {
				return f.manifest, [][]byte{[]byte("fixture-only")}, nil
			}, activate: func(context.Context, []byte, [][]byte, func(string) (io.ReadCloser, error)) error { return nil }}
			if _, err := p.Promote(context.Background(), f.staged, f.bundle, f.install); err != nil {
				t.Fatal(err)
			}
			cause := errors.New("controlled cleanup uncertainty")
			calls := []string{}
			source := nativeSessionCloseSource{calls: &calls}
			if kind == "discard-failed" {
				source.discardErr = cause
			}
			if kind == "source-close-failed" {
				source.closeErr = cause
			}
			s := &nativeReleaseUpdateSession{NativeReleaseSource: source, promoter: p, drain: func() error {
				calls = append(calls, "drain")
				if kind == "unknown-runtime" {
					return cause
				}
				return nil
			}}
			results := make(chan error, 4)
			var group sync.WaitGroup
			for range 4 {
				group.Add(1)
				go func() { defer group.Done(); results <- s.Close() }()
			}
			group.Wait()
			close(results)
			for err := range results {
				if (err == nil) != (kind == "success") || kind != "success" && !errors.Is(err, cause) {
					t.Fatal(err)
				}
			}
			want := "drain,discard,source-close"
			if kind == "unknown-runtime" {
				want = "drain,source-close"
			}
			if strings.Join(calls, ",") != want {
				t.Fatal("cleanup order or duplicate close", calls)
			}
			_, err := os.Lstat(filepath.Join(f.root, "staging", f.bundle.ManifestID().String(), "artifacts"))
			if kind == "unknown-runtime" {
				if err != nil || p.closed {
					t.Fatal("unknown live payload discarded", err, p.closed)
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Fatal("independent safe staging cleanup skipped", err)
			}
			if kind == "unknown-runtime" && ports.InstallUpdateFailureCode(s.Close()) != "SELF_UPDATE_RUNTIME_CLEANUP_FAILED" {
				t.Fatal("cleanup uncertainty reason lost", s.Close())
			}
		})
	}
}

func TestNativeUpdatePreparationNeverCallsFactoryWithoutManagedOwner(t *testing.T) {
	called := false
	p, err := NewNativeReleaseUpdatePreparation(&releaseactivation.RuntimeLease{}, "/tmp/staging", func(string, string, domain.InstallContext) (NativeReleaseContent, error) {
		called = true
		return NativeReleaseContent{}, errors.New("must not construct source")
	}, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	source, _ := domain.NewSourceID("github-release")
	target, _ := domain.NewInstallTarget(releaseinstall.SystemInstallRoot)
	install, _ := domain.NewGitHubAssetInstallContext(target)
	for _, version := range []string{"v1.2.3", "latest", "v1.2"} {
		if session, err := p.BeginInstallUpdate(context.Background(), source, version, install, false); err == nil || session != nil || called {
			t.Fatal("factory ran without owner", session, called, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.BeginInstallUpdate(ctx, source, "v1.2.3", install, false); !errors.Is(err, context.Canceled) || called {
		t.Fatal(err, called)
	}
	if _, err := p.BeginInstallUpdate(context.Background(), source, "v1.2.3", install, true); err == nil || called {
		t.Fatal("native update adoption admitted", called, err)
	}
}

func TestNativeUpdateFailureReasonsAreControlledAndRetainCauses(t *testing.T) {
	for _, cause := range []error{releaseactivation.ErrTransition, releaseactivation.ErrBusy, releaseactivation.ErrUnmanaged, releaseactivation.ErrCaller, releaseactivation.ErrUntrusted, releaseactivation.ErrProtocol, releaseactivation.ErrHealth, releaseactivation.ErrInstaller, releaseactivation.ErrCleanup} {
		wrapped := NativeReleaseUpdateFailure(cause)
		if !errors.Is(wrapped, cause) || ports.InstallUpdateFailureCode(wrapped) != cause.Error() {
			t.Fatal(cause, wrapped)
		}
	}
	cause := errors.New("untrusted stdout is not a reason code")
	if wrapped := NativeReleaseUpdateFailure(cause); !errors.Is(wrapped, cause) || ports.InstallUpdateFailureCode(wrapped) != "SELF_UPDATE_FAILED" {
		t.Fatal(wrapped)
	}
}
