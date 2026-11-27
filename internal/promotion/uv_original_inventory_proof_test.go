package promotion

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

type originalProofArchive struct {
	resolved          domain.ResolvedArtifact
	archive, filename string
}

// Fixed synthetic archives exercise the real independent intake/footprint
// checks. These ports perform no public resolution or Artifact execution.
type originalProofSetPorts struct {
	root               string
	entries            map[string]originalProofArchive
	closeOriginal      func() error
	corrupt            string
	resolves, acquires []string
}

func (p *originalProofSetPorts) ResolveDependencies(ctx context.Context, ref domain.ArtifactReference, _ domain.InstallContext) (domain.DependencyResolution, error) {
	if err := ctx.Err(); err != nil {
		return domain.DependencyResolution{}, err
	}
	project, _, _ := strings.Cut(ref.Locator(), "@")
	entry, ok := p.entries[project]
	if !ok || ref.Source() != entry.resolved.Identity().Source() || ref.Locator() != project+"@1.0.0" {
		return domain.DependencyResolution{}, errors.New("unexpected old subject")
	}
	p.resolves = append(p.resolves, project)
	nodeID, _ := domain.NewDependencyNodeID("original-" + project)
	node, _ := domain.NewLockedDependencyWithRecordPath(nodeID, domain.DependencyPrimary, entry.resolved, entry.filename, false)
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	digest, _ := domain.NewSHA256Digest(strings.TrimPrefix(entry.resolved.DeclaredIntegrity(), "sha256:"))
	return domain.NewDependencyResolution(graph, sandbox.UVResolverRuntimeIdentity(), digest)
}

func (*originalProofSetPorts) Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	return domain.ResolvedArtifact{}, errors.New("use the isolated original lookup")
}

func (p *originalProofSetPorts) Acquire(ctx context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if err := ctx.Err(); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	project := resolved.Identity().Name()
	entry, ok := p.entries[project]
	if !ok || entry.resolved != resolved {
		return domain.AcquiredArtifact{}, errors.New("unexpected old archive")
	}
	p.acquires = append(p.acquires, project)
	body, err := os.ReadFile(entry.archive)
	if err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if p.corrupt == project {
		body = []byte("substituted excluded original archive")
	}
	directory := filepath.Join(p.root, run.String())
	if err := os.Mkdir(directory, 0o700); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if err := os.WriteFile(filepath.Join(directory, "wheel.whl"), body, 0o600); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if err := os.WriteFile(filepath.Join(directory, "filename"), []byte(entry.filename), 0o600); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	digest, _ := domain.NewSHA256Digest(strings.TrimPrefix(resolved.DeclaredIntegrity(), "sha256:"))
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), digest, "intake:"+run.String()+":wheel", uint64(len(body)), resolved.DeclaredIntegrity())
}

func originalUVInventoryFixture(t *testing.T) (pypiVenvPlan, *pypiVenvTransaction, pypiNativeInventory, *originalProofSetPorts) {
	t.Helper()
	_, plan := makePypiVenvFixture(t)
	root := realPromotionRoot(t)
	p := &originalProofSetPorts{root: filepath.Join(root, "intake"), entries: map[string]originalProofArchive{}}
	for _, project := range []string{"proof-alpha", "proof-beta"} {
		module := strings.ReplaceAll(project, "-", "_")
		dist := module + "-1.0.0.dist-info"
		filename := module + "-1.0.0-py3-none-any.whl"
		archivePath := filepath.Join(root, filename)
		writeNamedFixtureWheel(t, archivePath, project)
		body, err := os.ReadFile(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		identity, _ := domain.NewResolvedArtifactIdentity(artifactpypi.PublicPyPIProfile().Source(), project, "1.0.0", "wheel")
		resolved, err := domain.NewResolvedArtifact(identity, "https://files.pythonhosted.org/packages/fixture/"+filename, "sha256:"+hex.EncodeToString(sum[:]))
		if err != nil {
			t.Fatal(err)
		}
		p.entries[project] = originalProofArchive{resolved, archivePath, filename}
		archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			t.Fatal(err)
		}
		files := map[string][]byte{dist + "/INSTALLER": []byte("uv"), dist + "/REQUESTED": {}}
		for _, file := range archive.File {
			if file.Name == dist+"/RECORD" {
				continue
			}
			reader, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			data, readErr := io.ReadAll(reader)
			closeErr := reader.Close()
			if readErr != nil || closeErr != nil {
				t.Fatal(readErr, closeErr)
			}
			files[file.Name] = data
		}
		names := make([]string, 0, len(files))
		for name := range files {
			names = append(names, name)
		}
		sort.Strings(names)
		var record bytes.Buffer
		writer := csv.NewWriter(&record)
		for _, name := range names {
			path := filepath.Join(plan.site, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, files[name], 0o600); err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(files[name])
			if err := writer.Write([]string{name, "sha256=" + base64.RawURLEncoding.EncodeToString(digest[:]), strconv.Itoa(len(files[name]))}); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Write([]string{dist + "/RECORD", "", ""}); err != nil {
			t.Fatal(err)
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(plan.site, dist, "RECORD"), record.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	transaction, err := beginPyPIVenvTransaction(plan)
	if err != nil {
		t.Fatal(err)
	}
	p.closeOriginal = sync.OnceValue(transaction.close)
	t.Cleanup(func() {
		if err := p.closeOriginal(); err != nil {
			t.Error(err)
		}
	})
	original, err := capturePyPIInstalledSnapshot(transaction)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := capturePyPINativeInventoryFromMetadata(context.Background(), transaction, original.metadataFiles())
	if err != nil {
		t.Fatal(err)
	}
	return plan, transaction, inventory, p
}

func TestNativeUVWholeOriginalProofIncludesExcludedDistribution(t *testing.T) {
	for _, mode := range []string{"complete", "excluded-archive-tamper"} {
		t.Run(mode, func(t *testing.T) {
			plan, transaction, inventory, original := originalUVInventoryFixture(t)
			target, _ := domain.NewInstallTarget(plan.root)
			install, _ := domain.NewPythonVenvInstallContext(target)
			enrollment := &pypiNativeEnrollment{resolver: original, intake: original, intakeRoot: original.root, client: nativeUVClient}
			nodeID, _ := domain.NewDependencyNodeID("selected-original")
			entry := original.entries["proof-alpha"]
			node, _ := domain.NewLockedDependencyWithRecordPath(nodeID, domain.DependencyPrimary, entry.resolved, entry.filename, false)
			graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
			if err != nil {
				t.Fatal(err)
			}
			subset, err := enrollment.selectOwnership(context.Background(), graph, install, inventory)
			var selected struct{ Proofs []pypiNativeProof }
			if err != nil || json.Unmarshal(subset.control.Body(), &selected) != nil || len(selected.Proofs) != 1 || selected.Proofs[0].Project != "proof-alpha" {
				t.Fatal("ordinary update changed its selected-only original proof", err)
			}
			if mode == "excluded-archive-tamper" {
				original.corrupt = "proof-beta"
			}
			// No composite ArtifactReference or primary graph supplies this input.
			complete, err := enrollment.selectAllOwnership(context.Background(), install, inventory)
			if mode == "excluded-archive-tamper" {
				if err == nil || complete.control.Present() {
					t.Fatal("excluded original bypassed independent old-archive proof", err)
				}
			} else {
				var all struct{ Proofs []pypiNativeProof }
				if err != nil || json.Unmarshal(complete.control.Body(), &all) != nil || len(all.Proofs) != 2 {
					t.Fatal("complete original proof lost a subject", err)
				}
				for _, proof := range all.Proofs {
					archive, ok := original.entries[proof.Project]
					if !ok || proof.Source != "pypi" || proof.Filename != archive.filename || "sha256:"+proof.SHA256 != archive.resolved.DeclaredIntegrity() || len(proof.Files) == 0 {
						t.Fatal("complete proof lost authentic original binding")
					}
				}
				cache := filepath.Join(plan.site, "proof_beta", "__pycache__")
				found := false
				for _, directory := range complete.absentDirectories {
					found = found || directory == cache
				}
				if !found {
					t.Fatal("excluded original lost its cache namespace guard")
				}
			}
			if strings.Join(original.resolves, ",") != "proof-alpha,proof-alpha,proof-beta" || strings.Join(original.acquires, ",") != "proof-alpha,proof-alpha,proof-beta" {
				t.Fatal("complete proof did not acquire every actual old wheel")
			}
			if len(transaction.current.Files) != 0 {
				t.Fatal("original proof published HAA ownership")
			}
			if _, err := os.Lstat(filepath.Join(plan.root, pypiVenvMetadata)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("original proof published ownership metadata", err)
			}
			remaining, err := os.ReadDir(original.root)
			if err != nil || len(remaining) != 0 {
				t.Fatal("old archive proof did not clean exact intake scopes", err)
			}
		})
	}
}
