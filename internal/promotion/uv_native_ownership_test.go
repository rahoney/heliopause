package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

func nativeUVFixture(t *testing.T, mode string) (pypiVenvPlan, *pypiVenvTransaction, *pypiNativeDistribution, artifactpypi.WheelInspection, string, string) {
	t.Helper()
	plan, transaction, project, wheel, digest, archive := nativeWheelOwnershipFixture(t, "no-compile")
	if err := transaction.close(); err != nil {
		t.Fatal(err)
	}
	dist := project.directory
	if err := os.WriteFile(filepath.Join(plan.site, dist, "INSTALLER"), []byte("uv"), 0600); err != nil {
		t.Fatal(err)
	}
	if mode == "marker" {
		if err := os.WriteFile(filepath.Join(plan.site, dist, "INSTALLER"), []byte("uv\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if mode == "missing-marker" {
		if err := os.Remove(filepath.Join(plan.site, dist, "INSTALLER")); err != nil {
			t.Fatal(err)
		}
	}
	if mode == "missing-requested" {
		if err := os.Remove(filepath.Join(plan.site, dist, "REQUESTED")); err != nil {
			t.Fatal(err)
		}
	}
	if mode == "requested-content" {
		if err := os.WriteFile(filepath.Join(plan.site, dist, "REQUESTED"), []byte("untrusted"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if mode == "extra-cache" {
		if err := os.WriteFile(filepath.Join(plan.site, dist, "uv_cache.json"), []byte(`{"timestamp":null,"commit":null,"tags":null,"env":{},"directories":{}}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var record bytes.Buffer
	writer := csv.NewWriter(&record)
	if err := filepath.WalkDir(plan.site, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(plan.site, path)
		if err != nil {
			return err
		}
		if relative == filepath.Join(dist, "RECORD") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(body)
		return writer.Write([]string{filepath.ToSlash(relative), "sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]), strconv.Itoa(len(body))})
	}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Write([]string{dist + "/RECORD", "", ""}); err != nil {
		t.Fatal(err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plan.site, dist, "RECORD"), record.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if mode == "cache" || mode == "empty-cache" {
		path := filepath.Join(plan.site, "haa_promotion_fixture", "__pycache__")
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
		if mode == "cache" {
			if err := os.WriteFile(filepath.Join(path, "__init__.cpython-314.pyc"), []byte("unrecorded cache, never execute"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	metadata, err := os.ReadFile(filepath.Join(plan.site, dist, "METADATA"))
	if err != nil {
		t.Fatal(err)
	}
	requirement, _ := artifactpypi.ParseUpgradeRequirement("haa-promotion-fixture")
	input, err := artifactpypi.NewUpgradeInput(requirement, []artifactpypi.InstalledMetadataFile{{Directory: dist, Body: metadata}}, sandbox.PinnedPythonRuntime().PythonVersion)
	if err != nil {
		t.Fatal(err)
	}
	transaction, err = beginPyPIVenvTransaction(plan)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := capturePyPINativeInventory(context.Background(), transaction, input)
	if err != nil {
		_ = transaction.close()
		t.Fatal(err)
	}
	return plan, transaction, inventory.projects["haa-promotion-fixture"], wheel, digest, archive
}

func TestNativeUVProofUsesExactFormatAndAccountsForUnrecordedCaches(t *testing.T) {
	for _, mode := range []string{"valid", "marker", "missing-marker", "missing-requested", "requested-content", "extra-cache", "cache", "empty-cache", "resource-cache", "wheel-digest"} {
		t.Run(mode, func(t *testing.T) {
			fixtureMode := mode
			if mode == "resource-cache" {
				fixtureMode = "cache"
			}
			_, transaction, project, wheel, digest, _ := nativeUVFixture(t, fixtureMode)
			if mode == "resource-cache" {
				for index := range wheel.Surface.InstalledFiles {
					wheel.Surface.InstalledFiles[index].Role = artifactpypi.RuntimeRoleResource
				}
			}
			defer func() {
				if err := transaction.close(); err != nil {
					t.Error(err)
				}
			}()
			if mode == "wheel-digest" {
				wheel.ObservedSHA256 = strings.Repeat("f", 64)
			}
			destinations, files, err := proveNativePythonWheelFootprint(nativeUVClient, project, wheel, digest, func() (pypiNativeRuntime, error) {
				t.Fatal("UV profile attempted to accept bytecode")
				return pypiNativeRuntime{}, nil
			})
			if err == nil {
				_, err = nativeUVCacheAbsences(project, wheel)
			}
			if (mode == "valid" || mode == "missing-requested") != (err == nil) {
				t.Fatal(mode, err)
			}
			if mode == "valid" {
				if len(destinations) != len(project.files) || len(files) != len(project.files) {
					t.Fatal("incomplete native ownership")
				}
				found := false
				for _, file := range files {
					if file.Relative == project.directory+"/INSTALLER" {
						found = file.Role == "pinned-uv-installer-marker"
					}
				}
				if !found {
					t.Fatal("native client identity was lost")
				}
			}
		})
	}
}

// Fake ports serve the existing authenticated synthetic wheel only. Actual UV
// target generation and public lookup are a separate native integration gate.
type uvNativeOriginalPorts struct {
	*nativeOriginalPorts
	wrongRuntime bool
}

func (p *uvNativeOriginalPorts) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (domain.DependencyResolution, error) {
	r, err := p.nativeOriginalPorts.ResolveDependencies(ctx, reference, install)
	if err != nil || p.wrongRuntime {
		return r, err
	}
	return domain.NewDependencyResolution(r.Graph(), sandbox.UVResolverRuntimeIdentity(), r.LockfileDigest())
}

func TestNativeUVEnrollmentKeepsProofGuardManifestAndRollbackTogether(t *testing.T) {
	for _, mode := range []string{"adopt", "without-adopt", "wrong-runtime", "cache-before", "cache-precommit", "partial-publication", "old-wheel-tamper"} {
		t.Run(mode, func(t *testing.T) {
			fixtureMode := "valid"
			if mode == "cache-before" {
				fixtureMode = "cache"
			}
			plan, transaction, _, _, digest, archive := nativeUVFixture(t, fixtureMode)
			if err := transaction.close(); err != nil {
				t.Fatal(err)
			}
			records := realPromotionRoot(t)
			seed, _ := validStagedPyPIFixture(t, records)
			resolved := seed.Set().Inspected().Graph().Nodes()[0].Artifact()
			id := resolved.Identity()
			resolved, _ = domain.NewResolvedArtifact(id, resolved.AcquisitionLocator(), "sha256:"+digest)
			original := &uvNativeOriginalPorts{nativeOriginalPorts: &nativeOriginalPorts{root: filepath.Join(records, "intake"), archive: archive, resolved: resolved}, wrongRuntime: mode == "wrong-runtime"}
			if mode == "old-wheel-tamper" {
				original.corrupt = true
			}
			resolver := &uvGuardResolverFixture{t: t, graph: seed.Set().Inspected().Graph(), intent: artifactpypi.UVUpgradeSelection}
			runner := &fixturePyPIUpgradeRunner{t: t}
			promoter, _ := newPyPIPromotion(filepath.Join(records, "staging"), runner, "linux", "amd64")
			base, _ := NewUVUpdatePreparation(promoter, resolver, artifactpypi.UVUpgradeSelection)
			preparation, err := base.WithNativeEnrollment(original, original, original.root)
			if err != nil {
				t.Fatal(err)
			}
			if base.shared.native != nil {
				t.Fatal("immutable preparation was mutated")
			}
			target, _ := domain.NewInstallTarget(plan.root)
			install, _ := domain.NewPythonVenvInstallContext(target)
			guarded, err := preparation.BeginInstallUpdate(context.Background(), id.Source(), id.Name(), install, mode != "without-adopt")
			if mode == "without-adopt" {
				if guarded != nil || ports.InstallUpdateFailureCode(err) != "UPDATE_ADOPTION_REQUIRED" || resolver.calls != 0 {
					t.Fatal(err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			s := guarded.(*pypiUpdateSession)
			t.Cleanup(func() {
				if err := s.Close(); err != nil {
					t.Error(err)
				}
			})
			resolution, err := s.ResolveDependencies(context.Background(), s.Reference(), install)
			if mode == "wrong-runtime" || mode == "cache-before" || mode == "old-wheel-tamper" {
				if err == nil || runner.calls != 0 || ports.InstallUpdateFailureCode(err) != "UPDATE_ORIGINAL_STATE_UNSUPPORTED" {
					t.Fatal("unproven native ownership accepted", mode, err)
				}
				return
			}
			if err != nil || len(resolution.ProjectControls()) != 5 || len(s.transaction.current.Files) == 0 {
				t.Fatal("native proof not bound", err)
			}
			bundle := nativeVerifiedBundle(t, seed, resolution, install)
			staging, _ := NewLocalStaging(filepath.Join(records, "intake"), filepath.Join(records, "evidence"), filepath.Join(records, "staging"))
			staged, err := staging.Stage(context.Background(), bundle)
			if err != nil {
				t.Fatal(err)
			}
			cache := filepath.Join(plan.site, "haa_promotion_fixture", "__pycache__")
			if mode == "cache-precommit" {
				runner.before = func() {
					if err := os.Mkdir(cache, 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(cache, "user.pyc"), []byte("preserve user data"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			cause := errors.New("controlled publication failure")
			if mode == "partial-publication" {
				s.transaction.fault = func(step string) error {
					if step == "after-publish:site" {
						return cause
					}
					return nil
				}
			}
			_, err = s.Promote(context.Background(), staged, bundle, install)
			if (mode == "adopt") != (err == nil) {
				t.Fatal("wrong publication result", mode, err)
			}
			if mode == "partial-publication" && !errors.Is(err, cause) {
				t.Fatal("original failure lost", err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			if mode == "adopt" {
				if _, err := os.Stat(filepath.Join(plan.root, pypiVenvMetadata)); err != nil {
					t.Fatal("verified enrollment not published", err)
				}
			} else {
				if _, err := os.Lstat(filepath.Join(plan.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("failed enrollment published ownership", err)
				}
			}
			if mode == "cache-precommit" {
				body, err := os.ReadFile(filepath.Join(cache, "user.pyc"))
				if err != nil || string(body) != "preserve user data" {
					t.Fatal("foreign cache was removed", err)
				}
			}
		})
	}
}
