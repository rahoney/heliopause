package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/cli"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/sandbox"
)

var errUVSyncIntakeReached = errors.New("native original intake reached")

// The native original lookup runs in promoter.Begin, before project inspection
// establishes its own storage operation. Only the context this service built
// can satisfy the shared storage reservation at that point.
type uvSyncOriginalIntakeProbe struct {
	resolved  domain.ResolvedArtifact
	filename  string
	operation domain.OperationID
	acquires  int
}

func (p *uvSyncOriginalIntakeProbe) ResolveDependencies(context.Context, domain.ArtifactReference, domain.InstallContext) (domain.DependencyResolution, error) {
	nodeID, err := domain.NewDependencyNodeID("original-demo")
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	node, err := domain.NewLockedDependencyWithRecordPath(nodeID, domain.DependencyPrimary, p.resolved, p.filename, false)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{node}, nil)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	digest, err := domain.NewSHA256Digest(strings.TrimPrefix(p.resolved.DeclaredIntegrity(), "sha256:"))
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return domain.NewDependencyResolution(graph, sandbox.UVResolverRuntimeIdentity(), digest)
}

func (*uvSyncOriginalIntakeProbe) Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	return domain.ResolvedArtifact{}, errors.New("original lookup must use the isolated resolver")
}

func (p *uvSyncOriginalIntakeProbe) Acquire(ctx context.Context, _ domain.RunID, _ domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	p.acquires++
	p.operation = ports.StorageOperationFromContext(ctx)
	return domain.AcquiredArtifact{}, errUVSyncIntakeReached
}

type uvSyncUnusedEvidence struct{ pythonClientEvidence }
type uvSyncUnusedInspection struct{ ports.Inspection }
type uvSyncUnusedResolver struct {
	ports.ProjectLockUpdateResolver
}

func TestUVPipSyncAdoptionCarriesStorageOperationIntoOriginalIntake(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	venv := filepath.Join(root, "venv")
	version := strings.Split(sandbox.PinnedPythonRuntime().PythonVersion, ".")
	site := filepath.Join(venv, "lib", "python"+version[0]+"."+version[1], "site-packages")
	const dist = "demo-1.0.dist-info"
	for _, dir := range []string{filepath.Join(site, dist), filepath.Join(venv, "bin")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(venv, "pyvenv.cfg"), []byte("version = "+sandbox.PinnedPythonRuntime().PythonVersion+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var record strings.Builder
	for _, file := range []struct{ name, body string }{{"demo.py", "VALUE = 1\n"}, {dist + "/METADATA", "Metadata-Version: 2.1\nName: demo\nVersion: 1.0\n\n"}, {dist + "/INSTALLER", "uv"}} {
		if err := os.WriteFile(filepath.Join(site, filepath.FromSlash(file.name)), []byte(file.body), 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(file.body))
		fmt.Fprintf(&record, "%s,sha256=%s,%d\n", file.name, base64.RawURLEncoding.EncodeToString(sum[:]), len(file.body))
	}
	record.WriteString(dist + "/RECORD,,\n")
	if err := os.WriteFile(filepath.Join(site, dist, "RECORD"), []byte(record.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "requirements.txt")
	if err := os.WriteFile(source, []byte("packaging==25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stagingRoot := filepath.Join(root, "staging")
	promoter, err := promotion.NewPyPIPromotion(stagingRoot)
	if err != nil {
		t.Fatal(err)
	}
	const filename = "demo-1.0-py3-none-any.whl"
	identity, err := domain.NewResolvedArtifactIdentity(artifactpypi.PublicPyPIProfile().Source(), "demo", "1.0", "wheel")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := domain.NewResolvedArtifact(identity, "https://files.pythonhosted.org/packages/fixture/"+filename, "sha256:"+hex.EncodeToString(make([]byte, sha256.Size)))
	if err != nil {
		t.Fatal(err)
	}
	probe := &uvSyncOriginalIntakeProbe{resolved: resolved, filename: filename}
	service := &uvPipSyncService{promoter: promoter, resolver: uvSyncUnusedResolver{}, bindings: pythonClientBindings{
		promoter: promoter, nativeLookup: probe, intake: probe, intakeRoot: filepath.Join(root, "intake"), evidenceRoot: filepath.Join(root, "evidence"), stagingRoot: stagingRoot,
		inspector: uvSyncUnusedInspection{}, evidence: uvSyncUnusedEvidence{},
	}}
	target, err := domain.NewInstallTarget(venv)
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewPythonVenvInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Sync(context.Background(), cli.UVPipSyncRequest{SourceFile: source, Format: artifactpypi.UVSyncRequirements, Adopt: true, Install: install})
	if !errors.Is(err, errUVSyncIntakeReached) || ports.InstallUpdateFailureCode(err) != "UPDATE_ORIGINAL_STATE_UNSUPPORTED" {
		t.Fatalf("native original intake was not the observed boundary: %v", err)
	}
	if probe.acquires != 1 || probe.operation.String() == "" {
		t.Fatalf("native original intake lacked the storage operation: acquires=%d operation=%q", probe.acquires, probe.operation.String())
	}
}
