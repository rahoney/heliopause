package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type uvInstallResolutionFixture struct {
	resolution domain.DependencyResolution
	calls      int
	before     func()
	err        error
}

func (r *uvInstallResolutionFixture) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	r.calls++
	if r.before != nil {
		r.before()
	}
	return r.resolution, r.err
}

type uvInstallFixture struct {
	root, records string
	plan          pypiVenvPlan
	reference     domain.ArtifactReference
	install       domain.InstallContext
	seed          domain.VerifiedBundle
	resolver      *uvInstallResolutionFixture
	runner        *fixturePyPIUpgradeRunner
	preparation   *UVInstallPreparation
}

func newUVInstallFixture(t *testing.T, uvConfig bool) uvInstallFixture {
	t.Helper()
	root, _ := makePypiVenvFixture(t)
	if uvConfig {
		cfg := "home = /usr/local/bin\nimplementation = CPython\nuv = 0.13.4\nversion_info = " + sandbox.PinnedPythonRuntime().PythonVersion + "\ninclude-system-site-packages = false\n"
		if err := os.WriteFile(filepath.Join(root, "pyvenv.cfg"), []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := discoverUVPythonVenv(root)
	if err != nil {
		t.Fatal(err)
	}
	records := realPromotionRoot(t)
	seed, _ := validStagedPyPIFixture(t, records)
	digest, _ := domain.NewSHA256Digest(strings.Repeat("a", 64))
	resolution, err := domain.NewDependencyResolution(seed.Set().Inspected().Graph(), sandbox.UVResolverRuntimeIdentity(), digest)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &uvInstallResolutionFixture{resolution: resolution}
	runner := &fixturePyPIUpgradeRunner{t: t}
	promoter, _ := newPyPIPromotion(filepath.Join(records, "staging"), runner, "linux", "amd64")
	preparation, err := NewUVInstallPreparation(promoter, resolver)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(root)
	install, _ := domain.NewPythonVenvInstallContext(target)
	reference, _ := artifactpypi.ParseReference("haa-promotion-fixture@1.0.0")
	return uvInstallFixture{root, records, plan, reference, install, seed, resolver, runner, preparation}
}

func TestUVInstallUnsupportedIntentCannotAcquireTargetGuard(t *testing.T) {
	for _, mode := range []string{"source", "reference", "target-mode", "nil-context", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			f := newUVInstallFixture(t, true)
			var ctx context.Context = context.Background()
			ref, install := f.reference, f.install
			switch mode {
			case "source":
				source, _ := domain.NewSourceID("npm")
				ref, _ = domain.NewArtifactReference(source, "haa-promotion-fixture@1.0.0")
			case "reference":
				ref, _ = domain.NewArtifactReference(f.reference.Source(), "https://foreign.invalid/wheel.whl")
			case "target-mode":
				install, _ = domain.NewInstallContext(f.install.Target())
			case "nil-context":
				ctx = nil
			case "cancel":
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			}
			guard, err := f.preparation.BeginInstall(ctx, ref, install)
			if err == nil || guard != nil || f.resolver.calls != 0 {
				t.Fatal(mode, guard, err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation cause lost", err)
			}
			if _, err := os.Lstat(filepath.Join(f.root, ".heliopause")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("unsupported intent touched target", err)
			}
		})
	}
}

func (f uvInstallFixture) begin(t *testing.T) *uvInstallSession {
	t.Helper()
	guard, err := f.preparation.BeginInstall(context.Background(), f.reference, f.install)
	if err != nil {
		t.Fatal(err)
	}
	if _, update := guard.(ports.InstallUpdateSession); update {
		t.Fatal("ordinary session acquired update identity")
	}
	s := guard.(*uvInstallSession)
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func TestUVInstallHoldsGuardBeforeNativeSelection(t *testing.T) {
	for _, uvConfig := range []bool{false, true} {
		f := newUVInstallFixture(t, uvConfig)
		s := f.begin(t)
		if f.resolver.calls != 0 {
			t.Fatal("selection ran during preparation")
		}
		if other, err := f.preparation.BeginInstall(context.Background(), f.reference, f.install); err == nil || other != nil {
			t.Fatal("concurrent target guard accepted")
		}
		resolution, err := s.ResolveDependencies(context.Background(), f.reference, f.install)
		if err != nil || resolution.RuntimeIdentity() != sandbox.UVResolverRuntimeIdentity() || f.resolver.calls != 1 {
			t.Fatal(resolution, err)
		}
		if _, err := s.ResolveDependencies(context.Background(), f.reference, f.install); err == nil || f.resolver.calls != 1 {
			t.Fatal("selection repeated")
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		if err := s.Close(); err != nil {
			t.Fatal("close was not idempotent", err)
		}
		retry, err := f.preparation.BeginInstall(context.Background(), f.reference, f.install)
		if err != nil {
			t.Fatal("guard not released", err)
		}
		if err := retry.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUVInstallRefusesNonemptyStateBeforeNativeSelection(t *testing.T) {
	for _, mode := range []string{"native", "legacy", "managed"} {
		f := newUVInstallFixture(t, false)
		switch mode {
		case "native":
			if err := os.Mkdir(filepath.Join(f.plan.site, "foreign-1.0.dist-info"), 0o700); err != nil {
				t.Fatal(err)
			}
		case "legacy":
			if err := os.WriteFile(filepath.Join(f.plan.site, "foreign.egg-link"), []byte("foreign"), 0o600); err != nil {
				t.Fatal(err)
			}
		case "managed":
			runVenvFixture(t, f.plan, map[string]string{"site/managed.py": "old payload"}, "managed")
		}
		guard, err := f.preparation.BeginInstall(context.Background(), f.reference, f.install)
		if err == nil || guard != nil || f.resolver.calls != 0 {
			t.Fatal(mode, guard, err)
		}
		lock, err := acquirePyPIVenvGuard(f.root)
		if err != nil {
			t.Fatal("failed preparation retained guard", mode, err)
		}
		if err := lock.release(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUVInstallRejectsSelectionDriftAndPreservesCause(t *testing.T) {
	for _, mode := range []string{"before-inventory", "after-inventory", "after-config", "runtime", "controls", "version", "reference", "context", "failure", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			f := newUVInstallFixture(t, true)
			if mode == "version" {
				f.reference, _ = artifactpypi.ParseReference("haa-promotion-fixture@0.9.0")
			}
			s := f.begin(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			primary := errors.New("controlled native failure")
			addMetadata := func() {
				if err := os.Mkdir(filepath.Join(f.plan.site, "foreign-1.0.dist-info"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			ref, install := f.reference, f.install
			switch mode {
			case "before-inventory":
				addMetadata()
			case "after-inventory":
				f.resolver.before = addMetadata
			case "after-config":
				f.resolver.before = func() {
					if err := os.WriteFile(filepath.Join(f.root, "pyvenv.cfg"), []byte("version = "+sandbox.PinnedPythonRuntime().PythonVersion+"\nprompt = changed\n"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			case "runtime":
				f.resolver.resolution, _ = domain.NewDependencyResolution(f.resolver.resolution.Graph(), "python:pip", f.resolver.resolution.LockfileDigest())
			case "controls":
				control, _ := domain.NewProjectControlFile("unexpected.json", []byte("{}"), true)
				f.resolver.resolution, _ = f.resolver.resolution.WithProjectControls([]domain.ProjectControlFile{control})
			case "reference":
				ref, _ = artifactpypi.ParseReference("other@1.0")
			case "context":
				install, _ = domain.NewInstallContext(f.install.Target())
			case "failure":
				f.resolver.err = primary
			case "cancel":
				f.resolver.before = cancel
			}
			resolution, err := s.ResolveDependencies(ctx, ref, install)
			if err == nil || len(resolution.Graph().Nodes()) != 0 || f.runner.calls != 0 {
				t.Fatal(mode, resolution, err)
			}
			if mode == "failure" && !errors.Is(err, primary) || mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("primary cause lost", err)
			}
			if (mode == "reference" || mode == "context" || mode == "before-inventory") && f.resolver.calls != 0 {
				t.Fatal("invalid selection launched native resolver")
			}
		})
	}
}

func TestUVInstallUsesExistingExactPublicationAndRollback(t *testing.T) {
	for _, mode := range []string{"install", "partial-publication", "precommit-drift", "precommit-metadata", "precommit-cancel", "manifest-runtime", "manifest-digest", "manifest-target", "manifest-mode", "manifest-controls", "staged-tamper"} {
		t.Run(mode, func(t *testing.T) {
			f := newUVInstallFixture(t, true)
			s := f.begin(t)
			resolution, err := s.ResolveDependencies(context.Background(), f.reference, f.install)
			if err != nil {
				t.Fatal(err)
			}
			op, _ := domain.NewOperationID()
			input := evidence.ManifestContext{OperationID: op, InstallContext: f.install, ResolverRuntime: resolution.RuntimeIdentity(), LockfileDigest: resolution.LockfileDigest()}
			switch mode {
			case "manifest-runtime":
				input.ResolverRuntime = "python:pip"
			case "manifest-digest":
				input.LockfileDigest, _ = domain.NewSHA256Digest(strings.Repeat("b", 64))
			case "manifest-target":
				target, _ := domain.NewInstallTarget(f.root + "-other")
				input.InstallContext, _ = domain.NewPythonVenvInstallContext(target)
			case "manifest-mode":
				input.InstallContext, _ = domain.NewInstallContext(f.install.Target())
			case "manifest-controls":
				control, _ := domain.NewProjectControlFile("unexpected.json", []byte("{}"), true)
				input.ProjectControls = []domain.ProjectControlFile{control}
			}
			bundle, err := evidence.BuildVerifiedBundle(input, f.seed.Set())
			if err != nil {
				t.Fatal(err)
			}
			staging, _ := NewLocalStaging(filepath.Join(f.records, "intake"), filepath.Join(f.records, "evidence"), filepath.Join(f.records, "staging"))
			staged, err := staging.Stage(context.Background(), bundle)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			primary := errors.New("controlled publication failure")
			payload := filepath.Join(f.plan.site, "haa_promotion_fixture", "__init__.py")
			switch mode {
			case "partial-publication":
				s.transaction.fault = func(step string) error {
					if step == "after-publish:site" {
						return primary
					}
					return nil
				}
			case "precommit-drift":
				f.runner.before = func() {
					if err := os.MkdirAll(filepath.Dir(payload), 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(payload, []byte("user edit"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			case "precommit-metadata":
				f.runner.before = func() {
					if err := os.Mkdir(filepath.Join(f.plan.site, "foreign-1.0.dist-info"), 0o700); err != nil {
						t.Fatal(err)
					}
				}
			case "precommit-cancel":
				f.runner.before = cancel
			case "staged-tamper":
				artifact := f.seed.Set().Inspected().Inspections()[0].Artifact()
				path := filepath.Join(f.records, "staging", bundle.ManifestID().String(), "artifacts", artifact.Digest().String()+".whl")
				if err := os.Chmod(path, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("substituted"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			promoted, err := s.Promote(ctx, staged, bundle, f.install)
			if (mode == "install") != (err == nil) || mode == "install" && promoted.ManifestID() != bundle.ManifestID() {
				t.Fatal(mode, promoted, err)
			}
			if mode == "partial-publication" && !errors.Is(err, primary) || mode == "precommit-cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("primary cause lost", err)
			}
			if strings.HasPrefix(mode, "manifest-") || mode == "staged-tamper" {
				if f.runner.calls != 0 {
					t.Fatal("unbound selection reached materializer")
				}
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			body, readErr := os.ReadFile(payload)
			switch mode {
			case "install":
				if readErr != nil || string(body) != "value = 42\n" {
					t.Fatal(string(body), readErr)
				}
			case "precommit-drift":
				if readErr != nil || string(body) != "user edit" {
					t.Fatal("user edit lost", readErr)
				}
			default:
				if !errors.Is(readErr, os.ErrNotExist) {
					t.Fatal("failed installation left payload", readErr)
				}
			}
		})
	}
}
