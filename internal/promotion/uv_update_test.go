package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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

type uvGuardResolverFixture struct {
	t      *testing.T
	graph  domain.LockedDependencyGraph
	intent artifactpypi.UVSelectionIntent
	calls  int
	before func()
	mutate func(domain.DependencyResolution) domain.DependencyResolution
}

func (r *uvGuardResolverFixture) ResolveDependencyUpdate(_ context.Context, _ domain.ArtifactReference, _ domain.InstallContext, controls []domain.ProjectControlFile) (domain.DependencyResolution, error) {
	r.t.Helper()
	r.calls++
	input, intent, err := artifactpypi.ParseUVSelectionRequest(controls, sandbox.PinnedPythonRuntime().PythonVersion)
	if err != nil || intent != r.intent {
		r.t.Fatal("guard changed original input or intent", intent, err)
	}
	if r.before != nil {
		r.before()
	}
	artifact := r.graph.Nodes()[0].Artifact()
	name, version := artifact.Identity().Name(), artifact.Identity().Version()
	changes := []string{}
	found := false
	for _, installed := range input.Installed() {
		if installed.Project() == name {
			found = true
		}
		if installed.Project() == name && installed.Version() != version {
			changes = append(changes, fmt.Sprintf(`{"name":%q,"version":%q,"action":"uninstalled"}`, name, installed.Version()), fmt.Sprintf(`{"name":%q,"version":%q,"action":"installed"}`, name, version))
		}
	}
	if !found {
		changes = append(changes, fmt.Sprintf(`{"name":%q,"version":%q,"action":"installed"}`, name, version))
	}
	delta := []byte(`{"schema":{"version":"preview"},"dry_run":true,"changes":[` + strings.Join(changes, ",") + `]}`)
	lock := []byte(fmt.Sprintf("lock-version = \"1.0\"\ncreated-by = \"uv\"\nrequires-python = \">=3.14\"\n[[packages]]\nname = %q\nversion = %q\n[[packages.wheels]]\nurl = %q\nsize = 1\nhashes = { sha256 = %q }\n", name, version, artifact.AcquisitionLocator(), strings.TrimPrefix(artifact.DeclaredIntegrity(), "sha256:")))
	for i, body := range [][]byte{delta, lock} {
		control, err := domain.NewProjectControlFile([]string{artifactpypi.UVSelectionDeltaControlName, artifactpypi.UVSelectionLockControlName}[i], body, true)
		if err != nil {
			r.t.Fatal(err)
		}
		controls = append(controls, control)
	}
	sum := sha256.Sum256(lock)
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	resolution, err := domain.NewDependencyResolution(r.graph, sandbox.UVResolverRuntimeIdentity(), digest)
	if err != nil {
		r.t.Fatal(err)
	}
	resolution, err = resolution.WithProjectControls(controls)
	if err != nil {
		r.t.Fatal(err)
	}
	if r.mutate != nil {
		resolution = r.mutate(resolution)
	}
	return resolution, nil
}

type uvUpdateGuardFixture struct {
	root, records string
	plan          pypiVenvPlan
	install       domain.InstallContext
	seed          domain.VerifiedBundle
	runner        *fixturePyPIUpgradeRunner
	resolver      *uvGuardResolverFixture
	preparation   *UVUpdatePreparation
}

func newUVUpdateGuardFixture(t *testing.T, intent artifactpypi.UVSelectionIntent, noop bool) uvUpdateGuardFixture {
	t.Helper()
	root, plan := makePypiVenvFixture(t)
	oldVersion := "0.9.0"
	if noop {
		oldVersion = "1.0.0"
	}
	old := venvDestinations(t, map[string]string{"site/haa_promotion_fixture-" + oldVersion + ".dist-info/METADATA": "Name: haa-promotion-fixture\nVersion: " + oldVersion + "\n\n", "site/haa_promotion_fixture/__init__.py": "old payload", "site/obsolete.py": "obsolete"}, "haa-promotion-fixture")
	for i := range old {
		old[i].Version = oldVersion
	}
	if err := plan.commit("", old); err != nil {
		t.Fatal(err)
	}
	runVenvFixture(t, plan, map[string]string{"site/other-1.0.dist-info/METADATA": "Name: other\nVersion: 1.0\n\n", "site/other.py": "retained unrelated"}, "other")
	cfg := "home = /usr/local/bin\nimplementation = CPython\nuv = 0.12.0\nversion_info = " + sandbox.PinnedPythonRuntime().PythonVersion + "\ninclude-system-site-packages = false\n"
	if err := os.WriteFile(filepath.Join(root, "pyvenv.cfg"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	records := realPromotionRoot(t)
	seed, _ := validStagedPyPIFixture(t, records)
	resolver := &uvGuardResolverFixture{t: t, graph: seed.Set().Inspected().Graph(), intent: intent}
	runner := &fixturePyPIUpgradeRunner{t: t}
	promoter, _ := newPyPIPromotion(filepath.Join(records, "staging"), runner, "linux", "amd64")
	preparation, err := NewUVUpdatePreparation(promoter, resolver, intent)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(root)
	install, _ := domain.NewPythonVenvInstallContext(target)
	return uvUpdateGuardFixture{root, records, plan, install, seed, runner, resolver, preparation}
}

func (f uvUpdateGuardFixture) begin(t *testing.T) *pypiUpdateSession {
	t.Helper()
	session, err := f.preparation.BeginInstallUpdate(context.Background(), artifactpypi.PublicPyPIProfile().Source(), "haa-promotion-fixture", f.install, false)
	if err != nil {
		t.Fatal(err)
	}
	s := session.(*pypiUpdateSession)
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func TestUVUpdateGuardKeepsOriginalOwnershipAndTransactionForAllNativeIntents(t *testing.T) {
	for _, intent := range []artifactpypi.UVSelectionIntent{artifactpypi.UVInstallSelection, artifactpypi.UVUpgradeSelection, artifactpypi.UVUpgradePackageSelection} {
		for _, noop := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/noop=%t", intent, noop), func(t *testing.T) {
				f := newUVUpdateGuardFixture(t, intent, noop)
				s := f.begin(t)
				if f.resolver.calls != 0 {
					t.Fatal("selection ran before guard")
				}
				if other, err := f.preparation.BeginInstallUpdate(context.Background(), s.Reference().Source(), "haa-promotion-fixture", f.install, false); err == nil || other != nil {
					t.Fatal("concurrent guard accepted")
				}
				resolution, err := s.ResolveDependencies(context.Background(), s.Reference(), f.install)
				if err != nil || len(resolution.Graph().Nodes()) != 1 || len(resolution.ProjectControls()) != 4 {
					t.Fatal(resolution, err)
				}
				if _, err := s.ResolveDependencies(context.Background(), s.Reference(), f.install); err == nil || f.resolver.calls != 1 {
					t.Fatal("successful selection repeated")
				}
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
				if err := s.Close(); err != nil {
					t.Fatal("close not idempotent", err)
				}
				lock, err := acquirePyPIVenvGuard(f.root)
				if err != nil {
					t.Fatal("guard not released", err)
				}
				if err := lock.release(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func mutateUVGuardControl(t *testing.T, resolution domain.DependencyResolution, name string, mutate func([]byte) []byte) domain.DependencyResolution {
	t.Helper()
	controls := resolution.ProjectControls()
	for i, control := range controls {
		if control.Name() == name {
			var err error
			controls[i], err = domain.NewProjectControlFile(name, mutate(control.Body()), true)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	result, err := resolution.WithProjectControls(controls)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestUVUpdateGuardRejectsUnboundNativeSelectionBeforeMaterialization(t *testing.T) {
	for _, mode := range []string{"runtime", "digest", "original", "intent", "delta", "lock-version", "lock-url", "lock-hash", "missing-control", "foreign-control", "before-original", "after-original", "after-config", "cancel", "reference", "context"} {
		t.Run(mode, func(t *testing.T) {
			f := newUVUpdateGuardFixture(t, artifactpypi.UVUpgradeSelection, false)
			s := f.begin(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			changeOriginal := func() {
				if err := os.WriteFile(filepath.Join(f.plan.site, "other-1.0.dist-info/METADATA"), []byte("Name: other\nVersion: 2.0\n\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			ref, install := s.Reference(), f.install
			switch mode {
			case "before-original":
				changeOriginal()
			case "after-original":
				f.resolver.before = changeOriginal
			case "after-config":
				f.resolver.before = func() {
					if err := os.WriteFile(filepath.Join(f.root, "pyvenv.cfg"), []byte("changed"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			case "cancel":
				f.resolver.before = cancel
			case "reference":
				ref, _ = artifactpypi.ParseReference("other")
			case "context":
				install, _ = domain.NewInstallContext(f.install.Target())
			default:
				f.resolver.mutate = func(r domain.DependencyResolution) domain.DependencyResolution {
					switch mode {
					case "runtime", "digest":
						runtime, digest := r.RuntimeIdentity(), r.LockfileDigest()
						if mode == "runtime" {
							runtime = "python:pip"
						} else {
							digest, _ = domain.NewSHA256Digest(strings.Repeat("b", 64))
						}
						out, err := domain.NewDependencyResolution(r.Graph(), runtime, digest)
						if err != nil {
							t.Fatal(err)
						}
						out, err = out.WithProjectControls(r.ProjectControls())
						if err != nil {
							t.Fatal(err)
						}
						return out
					case "missing-control", "foreign-control":
						controls := r.ProjectControls()[:3]
						if mode == "foreign-control" {
							foreign, _ := domain.NewProjectControlFile("foreign.json", []byte("{}"), true)
							controls = append(controls, foreign)
						}
						out, err := r.WithProjectControls(controls)
						if err != nil {
							t.Fatal(err)
						}
						return out
					}
					name := artifactpypi.UVSelectionLockControlName
					if mode == "original" {
						name = artifactpypi.UpgradeInputControlName
					}
					if mode == "intent" {
						name = artifactpypi.UVSelectionIntentControlName
					}
					if mode == "delta" {
						name = artifactpypi.UVSelectionDeltaControlName
					}
					mutated := mutateUVGuardControl(t, r, name, func(body []byte) []byte {
						switch mode {
						case "original":
							return append(body, ' ')
						case "intent":
							return []byte(`{"version":1,"intent":"install"}`)
						case "delta":
							return []byte(`{"schema":{"version":"preview"},"dry_run":true,"changes":[]}`)
						case "lock-version":
							return []byte(strings.Replace(string(body), `version = "1.0.0"`, `version = "2.0.0"`, 1))
						case "lock-url":
							return []byte(strings.Replace(string(body), "files.pythonhosted.org", "foreign.invalid", 1))
						case "lock-hash":
							return []byte(strings.Replace(string(body), strings.TrimPrefix(r.Graph().Nodes()[0].Artifact().DeclaredIntegrity(), "sha256:"), strings.Repeat("b", 64), 1))
						}
						return body
					})
					// Rebind the test lock digest so malformed archive identity is
					// rejected independently of the outer content hash check.
					if strings.HasPrefix(mode, "lock-") {
						for _, control := range mutated.ProjectControls() {
							if control.Name() == artifactpypi.UVSelectionLockControlName {
								sum := sha256.Sum256(control.Body())
								digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
								out, err := domain.NewDependencyResolution(mutated.Graph(), mutated.RuntimeIdentity(), digest)
								if err != nil {
									t.Fatal(err)
								}
								out, err = out.WithProjectControls(mutated.ProjectControls())
								if err != nil {
									t.Fatal(err)
								}
								return out
							}
						}
					}
					return mutated
				}
			}
			resolution, err := s.ResolveDependencies(ctx, ref, install)
			if err == nil || len(resolution.Graph().Nodes()) != 0 || f.runner.calls != 0 {
				t.Fatal("invalid selection accepted", mode, err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation cause lost", err)
			}
			if (mode == "before-original" || mode == "reference" || mode == "context") && f.resolver.calls != 0 {
				t.Fatal("invalid request launched resolver")
			}
		})
	}
}

func TestUVUpdateGuardReusesExactPublicationRetirementAndRollback(t *testing.T) {
	for _, mode := range []string{"upgrade", "noop", "partial-publication", "precommit-drift", "precommit-cancel", "manifest-substitution", "staged-tamper"} {
		t.Run(mode, func(t *testing.T) {
			f := newUVUpdateGuardFixture(t, artifactpypi.UVUpgradePackageSelection, mode == "noop")
			before, _, err := f.plan.readState()
			if err != nil {
				t.Fatal(err)
			}
			s := f.begin(t)
			resolution, err := s.ResolveDependencies(context.Background(), s.Reference(), f.install)
			if err != nil {
				t.Fatal(err)
			}
			controls := resolution.ProjectControls()
			if mode == "manifest-substitution" {
				controls = controls[:1]
			}
			op, _ := domain.NewOperationID()
			bundle, err := evidence.BuildVerifiedBundle(evidence.ManifestContext{OperationID: op, InstallContext: f.install, ResolverRuntime: resolution.RuntimeIdentity(), LockfileDigest: resolution.LockfileDigest(), ProjectControls: controls}, f.seed.Set())
			if err != nil {
				t.Fatal(err)
			}
			staging, _ := NewLocalStaging(filepath.Join(f.records, "intake"), filepath.Join(f.records, "evidence"), filepath.Join(f.records, "staging"))
			staged, err := staging.Stage(context.Background(), bundle)
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("controlled publication failure")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			payload := filepath.Join(f.plan.site, "haa_promotion_fixture/__init__.py")
			switch mode {
			case "partial-publication":
				s.transaction.fault = func(step string) error {
					if step == "after-publish:site" {
						return cause
					}
					return nil
				}
			case "precommit-drift":
				f.runner.before = func() {
					if err := os.WriteFile(payload, []byte("user edit"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			case "precommit-cancel":
				f.runner.before = cancel
			case "staged-tamper":
				path := filepath.Join(f.records, "staging", bundle.ManifestID().String(), "artifacts", f.seed.Set().Inspected().Inspections()[0].Artifact().Digest().String()+".whl")
				if err := os.Chmod(path, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("substituted"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			promoted, err := s.Promote(ctx, staged, bundle, f.install)
			success := mode == "upgrade" || mode == "noop"
			if success != (err == nil) || success && promoted.ManifestID() != bundle.ManifestID() {
				t.Fatal("publication result", mode, err)
			}
			if mode == "partial-publication" && !errors.Is(err, cause) || mode == "precommit-cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("primary cause lost", err)
			}
			if (mode == "manifest-substitution" || mode == "staged-tamper") && f.runner.calls != 0 {
				t.Fatal("unbound selection reached materializer")
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			want := "old payload"
			if success {
				want = "value = 42\n"
			}
			if mode == "precommit-drift" {
				want = "user edit"
			}
			body, err := os.ReadFile(payload)
			if err != nil || string(body) != want {
				t.Fatal("old/new file mismatch", string(body), err)
			}
			body, err = os.ReadFile(filepath.Join(f.plan.site, "other.py"))
			if err != nil || string(body) != "retained unrelated" {
				t.Fatal("unrelated state changed", err)
			}
			if success {
				if _, err := os.Lstat(filepath.Join(f.plan.site, "obsolete.py")); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("obsolete owned file retained", err)
				}
			} else if mode != "precommit-drift" {
				if err := f.plan.verifyState(before); err != nil {
					t.Fatal("rollback failed", err)
				}
			}
		})
	}
}

func TestUVUpdateGuardRefusesUnverifiedNativeOwnershipAndUnsupportedIntent(t *testing.T) {
	for _, mode := range []string{"adopt", "source", "intent", "nil", "no-ledger", "partial-ledger", "empty", "staging-overlap"} {
		t.Run(mode, func(t *testing.T) {
			f := newUVUpdateGuardFixture(t, artifactpypi.UVUpgradeSelection, false)
			source := artifactpypi.PublicPyPIProfile().Source()
			var ctx context.Context = context.Background()
			switch mode {
			case "staging-overlap":
				f.preparation.shared.promoter.stagingRoot = filepath.Join(f.root, "staging")
			case "source":
				source, _ = domain.NewSourceID("pytorch-cpu")
			case "intent":
				if p, err := NewUVUpdatePreparation(f.preparation.shared.promoter, f.resolver, "unknown"); err == nil || p != nil {
					t.Fatal("unknown native intent accepted")
				}
				return
			case "nil":
				ctx = nil
			case "no-ledger":
				if err := os.Remove(filepath.Join(f.root, ".heliopause/pypi-transaction.json")); err != nil {
					t.Fatal(err)
				}
			case "partial-ledger":
				if err := os.Remove(filepath.Join(f.plan.site, "haa_promotion_fixture-0.9.0.dist-info/METADATA")); err != nil {
					t.Fatal(err)
				}
			case "empty":
				root, _ := makePypiVenvFixture(t)
				target, _ := domain.NewInstallTarget(root)
				f.install, _ = domain.NewPythonVenvInstallContext(target)
			}
			guard, err := f.preparation.BeginInstallUpdate(ctx, source, "haa-promotion-fixture", f.install, mode == "adopt")
			if err == nil || guard != nil || f.resolver.calls != 0 {
				t.Fatal("unverified input accepted", mode, err)
			}
			if mode == "no-ledger" && ports.InstallUpdateFailureCode(err) != "UPDATE_ADOPTION_REQUIRED" {
				t.Fatal("wrong ownership failure", err)
			}
		})
	}
}
