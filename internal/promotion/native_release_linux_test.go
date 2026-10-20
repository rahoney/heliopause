//go:build linux

package promotion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/releaseactivation"
	"github.com/rahoney/heliopause/internal/releaseinstall"
)

type nativePromotionFixture struct {
	root     string
	bundle   domain.VerifiedBundle
	staged   domain.StagedSet
	install  domain.InstallContext
	manifest releaseinstall.ReleaseManifest
	files    map[string][]byte
}

// This is a typed ALLOW/staging fixture, not signed native4 or sandbox evidence.
func newNativePromotionFixture(t *testing.T, kind string) nativePromotionFixture {
	t.Helper()
	root := realPromotionRoot(t)
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte {
		body, err := os.ReadFile(filepath.Join("..", "..", "docs", "planning", "evidence", "m13-003-release-build", "v0.1.0", name))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	var record map[string]any
	if err := json.Unmarshal(read("helox-release-manifest.json"), &record); err != nil {
		t.Fatal(err)
	}
	assets := append(record["assets"].([]any), map[string]any{"name": "haa-release-installer-linux-amd64", "platform": "linux/amd64", "role": "release-installer"})
	files := map[string][]byte{"helox-release-sbom.cdx.json": read("helox-release-sbom.cdx.json"), "helox-runtime-images.json": read("helox-runtime-images.json")}
	for _, raw := range assets {
		asset := raw.(map[string]any)
		name := asset["name"].(string)
		if asset["platform"] == "linux/amd64" && !strings.HasSuffix(name, ".deb") && !strings.HasSuffix(name, ".rpm") {
			body := []byte("staged-native-only-" + name)
			sum := sha256.Sum256(body)
			files[name] = body
			asset["size"], asset["sha256"] = len(body), hex.EncodeToString(sum[:])
		}
	}
	record["assets"] = assets
	body, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := releaseinstall.ParseReleaseManifest(body)
	if err != nil {
		t.Fatal(err)
	}
	files["helox-release-manifest.json"] = body
	if kind == "missing-member" {
		delete(files, "helox-linux-amd64")
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	source, _ := domain.NewSourceID("github-release")
	primary, _ := domain.NewDependencyNodeID("helox-release-manifest.json")
	nodes := []domain.LockedDependency{}
	edges := []domain.DependencyEdge{}
	inspections := []domain.DependencyInspection{}
	for _, name := range names {
		nameIdentity := "rahoney-heliopause"
		if kind == "foreign-repository" {
			nameIdentity = "other-heliopause"
		}
		identity, _ := domain.NewResolvedArtifactIdentity(source, nameIdentity, manifest.Version(), name)
		sum := sha256.Sum256(files[name])
		digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
		integrity := "sha256:" + digest.String()
		resolved, _ := domain.NewResolvedArtifact(identity, "github-release://fixture/"+name, integrity)
		node, _ := domain.NewDependencyNodeID(name)
		role := domain.DependencyTransitive
		if node == primary {
			role = domain.DependencyPrimary
		} else {
			edge, _ := domain.NewDependencyEdge(primary, node)
			edges = append(edges, edge)
		}
		dependency, err := domain.NewLockedDependencyWithRecordPath(node, role, resolved, name, false)
		if err != nil {
			t.Fatal(err)
		}
		nodes = append(nodes, dependency)
		run, _ := domain.NewRunID()
		dir := filepath.Join(root, "intake", run.String())
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "asset"), files[name], 0600); err != nil {
			t.Fatal(err)
		}
		artifact, _ := domain.NewAcquiredArtifactWithDeclaredIntegrity(identity, digest, "intake:"+run.String()+":github-release", uint64(len(files[name])), integrity)
		checkID, _ := domain.NewCheckID("native-staging-fixture")
		check, _ := domain.NewCheckExecution(checkID, domain.CheckInspection, true, domain.CapabilitySupported, domain.ExecutionCompleted, "")
		evidenceID, _ := domain.NewEvidenceID("native-staging-fixture")
		reference, _ := domain.NewEvidenceReference(evidenceID, "evidence:"+run.String())
		decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "native-staging-fixture", 1, []string{"FIXTURE_ONLY"})
		inspection, err := domain.NewDependencyInspection(node, run, artifact, []domain.CheckExecution{check}, []domain.EvidenceReference{reference}, decision)
		if err != nil {
			t.Fatal(err)
		}
		inspections = append(inspections, inspection)
	}
	graph, err := domain.NewLockedDependencyGraph(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := domain.NewInspectedDependencySet(graph, inspections)
	if err != nil {
		t.Fatal(err)
	}
	decision, _ := domain.NewPolicyDecision(domain.DecisionAllow, "native-staging-fixture", 1, []string{"FIXTURE_ONLY"})
	set, err := domain.NewVerifiedSet(inspected, decision)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := domain.NewInstallTarget(releaseinstall.SystemInstallRoot)
	install, _ := domain.NewGitHubAssetInstallContext(target)
	contextInstall := install
	if kind == "wrong-manifest-target" {
		target, _ := domain.NewInstallTarget(filepath.Join(root, "other-target"))
		contextInstall, _ = domain.NewGitHubAssetInstallContext(target)
	}
	op, _ := domain.NewOperationID()
	lock, _ := domain.NewSHA256Digest(strings.Repeat("d", 64))
	bundle, err := evidence.BuildVerifiedBundle(evidence.ManifestContext{OperationID: op, InstallContext: contextInstall, ResolverRuntime: "native-staging-fixture", LockfileDigest: lock}, set)
	if err != nil {
		t.Fatal(err)
	}
	stager, err := NewLocalStaging(filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging"))
	if err != nil {
		t.Fatal(err)
	}
	staged, err := stager.Stage(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	return nativePromotionFixture{root, bundle, staged, install, manifest, files}
}

func TestNativePromotionBindsCompleteSealedSetAndHeldDescriptors(t *testing.T) {
	for _, kind := range []string{"success", "path-replaced-after-open", "changed-payload", "hardlink", "symlink", "writable", "size-mismatch", "record-tampered", "proof-error", "wrong-target", "wrong-manifest-target", "foreign-repository", "missing-member"} {
		t.Run(kind, func(t *testing.T) {
			f := newNativePromotionFixture(t, kind)
			called := 0
			primaryError := errors.New("frozen proof unavailable")
			stageRoot := filepath.Join(f.root, "staging", f.bundle.ManifestID().String())
			asset := f.bundle.Set().Inspected().Inspections()[0].Artifact()
			assetPath := filepath.Join(stageRoot, "artifacts", asset.Digest().String()+".asset")
			write := func(path string, body []byte) {
				if err := os.Chmod(path, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, body, 0400); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, 0400); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "changed-payload":
				body := append([]byte(nil), f.files[asset.Identity().Variant()]...)
				body[0] ^= 1
				write(assetPath, body)
			case "hardlink":
				if err := os.Link(assetPath, filepath.Join(f.root, "extra-link")); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(assetPath, assetPath+".saved"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(assetPath+".saved", assetPath); err != nil {
					t.Fatal(err)
				}
			case "writable":
				if err := os.Chmod(assetPath, 0600); err != nil {
					t.Fatal(err)
				}
			case "size-mismatch":
				write(assetPath, []byte("short"))
			case "record-tampered":
				write(filepath.Join(stageRoot, manifestFilename), []byte("{}"))
			case "wrong-target":
				target, _ := domain.NewInstallTarget(filepath.Join(f.root, "target"))
				f.install, _ = domain.NewGitHubAssetInstallContext(target)
			}
			promoter := &NativeReleasePromotion{stagingRoot: filepath.Join(f.root, "staging"), proof: func(artifact domain.AcquiredArtifact) (releaseinstall.ReleaseManifest, [][]byte, error) {
				if artifact.Identity().Variant() != "helox-release-manifest.json" {
					t.Fatal("wrong proof subject")
				}
				if kind == "proof-error" {
					return releaseinstall.ReleaseManifest{}, nil, primaryError
				}
				return f.manifest, [][]byte{[]byte("component-only proof")}, nil
			}, activate: func(_ context.Context, body []byte, proofs [][]byte, open func(string) (io.ReadCloser, error)) error {
				called++
				if !bytes.Equal(body, f.files["helox-release-manifest.json"]) || len(proofs) != 1 {
					t.Fatal("different manifest/proof")
				}
				if kind == "path-replaced-after-open" {
					if err := os.Rename(assetPath, assetPath+".original"); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(assetPath, []byte("replacement"), 0400); err != nil {
						t.Fatal(err)
					}
				}
				for name, expected := range f.files {
					if name == "helox-release-manifest.json" {
						continue
					}
					reader, err := open(name)
					if err != nil {
						return err
					}
					actual, err := io.ReadAll(reader)
					closeErr := reader.Close()
					if err != nil || closeErr != nil || !bytes.Equal(actual, expected) {
						t.Fatal("staging was reopened or selected bytes changed", name, err, closeErr)
					}
				}
				if _, err := open("unselected"); err == nil {
					t.Fatal("foreign payload admitted")
				}
				return nil
			}}
			result, err := promoter.Promote(context.Background(), f.staged, f.bundle, f.install)
			positive := kind == "success" || kind == "path-replaced-after-open"
			if (err == nil) != positive || positive && result.Target() != f.install.Target() || !positive && called != 0 || positive && called != 1 {
				t.Fatal(kind, result, called, err)
			}
			if kind == "proof-error" && !errors.Is(err, primaryError) {
				t.Fatal("lost proof failure", err)
			}
		})
	}
}

func TestNativePromotionCannotConfigureAnUnmanagedInstaller(t *testing.T) {
	if _, err := NewNativeReleasePromotion("/tmp/staging", nil, nil, nil); !errors.Is(err, releaseactivation.ErrUnmanaged) {
		t.Fatal(err)
	}
}
