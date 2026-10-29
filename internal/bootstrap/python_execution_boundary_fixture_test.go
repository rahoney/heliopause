//go:build linux && realcorpus

package bootstrap

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionpypi "github.com/rahoney/heliopause/internal/inspection/pypi"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/sandbox"
	verificationpypi "github.com/rahoney/heliopause/internal/verification/pypi"
)

const pythonBoundaryName = "haa-python-runtime-fixture"
const pythonBoundaryFilename = "haa_python_runtime_fixture-1.0.0-py3-none-any.whl"
const pythonBoundaryModule = `def main():
    import json
    import os
    import signal
    import sys
    import time
    mode = sys.argv[1]
    if mode == 'argv':
        os.write(1, json.dumps(sys.argv[2:]).encode())
    elif mode == 'exit37':
        return 37
    elif mode in ('term', 'kill'):
        os.write(1, b'signal\n')
        os.kill(os.getpid(), signal.SIGTERM if mode == 'term' else signal.SIGKILL)
    elif mode in ('cancel', 'deadline'):
        os.write(1, b'ready\n')
        while True:
            time.sleep(1)
    elif mode in ('stdout-overflow', 'stderr-overflow'):
        fd, body = (1, b'a' * 65536) if mode == 'stdout-overflow' else (2, b'b' * 65536)
        while True:
            os.write(fd, body)
    else:
        raise ValueError('unknown controlled fixture case')
`

// This adapter supplies unverified fixture DATA only. It does not stand in for
// public source authentication, native selection, inspection or Policy.
type pythonBoundaryAcquirer struct {
	root     string
	body     []byte
	resolved domain.ResolvedArtifact
	digest   domain.ContentDigest
}

// InspectResolved consumes the explicit unverified fixture graph. There is no
// native/public resolver behind this test-only acquisition adapter.
func (*pythonBoundaryAcquirer) Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	return domain.ResolvedArtifact{}, errors.New("controlled acquisition adapter cannot resolve public references")
}

func (f *pythonBoundaryAcquirer) Acquire(ctx context.Context, run domain.RunID, resolved domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if ctx == nil || ctx.Err() != nil || f == nil || resolved != f.resolved || run.String() == "" {
		return domain.AcquiredArtifact{}, errors.New("invalid controlled Python acquisition")
	}
	if err := os.MkdirAll(f.root, 0o700); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	canonical, err := filepath.EvalSymlinks(f.root)
	if err != nil || canonical != f.root {
		return domain.AcquiredArtifact{}, errors.New("controlled intake root is not canonical")
	}
	directory := filepath.Join(f.root, run.String())
	if filepath.Dir(directory) != f.root {
		return domain.AcquiredArtifact{}, errors.New("controlled run escapes intake")
	}
	if err := os.Mkdir(directory, 0o700); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	for _, file := range []struct {
		name string
		body []byte
		mode os.FileMode
	}{
		{"wheel.whl", f.body, 0o600}, {"filename", []byte(pythonBoundaryFilename), 0o400},
	} {
		out, err := os.OpenFile(filepath.Join(directory, file.name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, file.mode)
		if err != nil {
			return domain.AcquiredArtifact{}, err
		}
		_, writeErr := out.Write(file.body)
		if err := errors.Join(writeErr, out.Close()); err != nil {
			return domain.AcquiredArtifact{}, err
		}
	}
	sum := sha256.Sum256(f.body)
	if hex.EncodeToString(sum[:]) != f.digest.String() {
		return domain.AcquiredArtifact{}, errors.New("controlled wheel changed")
	}
	return domain.NewAcquiredArtifactWithDeclaredIntegrity(resolved.Identity(), f.digest, "intake:"+run.String()+":wheel", uint64(len(f.body)), resolved.DeclaredIntegrity())
}

func pythonBoundarySource(t *testing.T, root string) (*pythonBoundaryAcquirer, domain.DependencyResolution) {
	t.Helper()
	const info = "haa_python_runtime_fixture-1.0.0.dist-info/"
	files := []struct{ name, body string }{
		{"haa_python_runtime_fixture.py", pythonBoundaryModule},
		{info + "METADATA", "Metadata-Version: 2.3\nName: " + pythonBoundaryName + "\nVersion: 1.0.0\nRequires-Python: >=3.14\n\nControlled boundary fixture.\n"},
		{info + "WHEEL", "Wheel-Version: 1.0\nGenerator: haa-controlled-boundary-fixture\nRoot-Is-Purelib: true\nTag: py3-none-any\n"},
		{info + "entry_points.txt", "[console_scripts]\nhaa-runtime-probe = haa_python_runtime_fixture:main\n"},
	}
	var record bytes.Buffer
	rows := csv.NewWriter(&record)
	for _, file := range files {
		sum := sha256.Sum256([]byte(file.body))
		if err := rows.Write([]string{file.name, "sha256=" + base64.RawURLEncoding.EncodeToString(sum[:]), fmt.Sprint(len(file.body))}); err != nil {
			t.Fatal(err)
		}
	}
	if err := rows.Write([]string{info + "RECORD", "", ""}); err != nil {
		t.Fatal(err)
	}
	rows.Flush()
	if err := rows.Error(); err != nil {
		t.Fatal(err)
	}
	files = append(files, struct{ name, body string }{info + "RECORD", record.String()})
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for _, file := range files {
		header := &zip.FileHeader{Name: file.name, Method: zip.Deflate}
		header.SetMode(0o644)
		out, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write([]byte(file.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive.Bytes())
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	identity, err := domain.NewResolvedArtifactIdentity(artifactpypi.PublicPyPIProfile().Source(), pythonBoundaryName, "1.0.0", "wheel")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := domain.NewResolvedArtifact(identity, "https://files.pythonhosted.org/packages/controlled/"+pythonBoundaryFilename, "sha256:"+digest.String())
	if err != nil {
		t.Fatal(err)
	}
	node, err := domain.NewDependencyNodeID("controlled-python-boundary")
	if err != nil {
		t.Fatal(err)
	}
	locked, err := domain.NewLockedDependencyWithRecordPath(node, domain.DependencyPrimary, resolved, pythonBoundaryFilename, false)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{locked}, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := domain.NewDependencyResolution(graph, "controlled-python-boundary-fixture", digest)
	if err != nil {
		t.Fatal(err)
	}
	return &pythonBoundaryAcquirer{root, bytes.Clone(archive.Bytes()), resolved, digest}, resolution
}

func pythonBoundaryExecution(t *testing.T, ctx context.Context, root string, backend *sandbox.PythonDynamicBackend, source *pythonBoundaryAcquirer, resolution domain.DependencyResolution, arguments []string) (domain.ExecutionContext, artifactpypi.ConsoleExecutionEntry) {
	t.Helper()
	runtime := sandbox.PinnedPythonRuntime()
	static, err := inspectionpypi.NewStaticInspector(root+"/intake", artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag})
	if err != nil {
		t.Fatal(err)
	}
	dynamic, err := inspectionpypi.NewDynamicInspector(backend)
	if err != nil {
		t.Fatal(err)
	}
	composite, err := inspectionpypi.NewCompositeInspector(static, dynamic)
	if err != nil {
		t.Fatal(err)
	}
	store, err := local.NewStore(root + "/evidence")
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewExecutionInspectService(source, verificationpypi.IntegrityVerifier{}, composite, store, policy.M3{}, policy.M5{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := service.InspectResolved(ctx, resolution)
	if err != nil || !inspection.Valid() || inspection.Set().Decision().Decision() != domain.DecisionAllow || inspection.Set().Decision().PolicyID() != "m5-pypi-pip" {
		t.Fatal("controlled fixture lacks actual whole current Python approval", err)
	}
	entries := inspection.Set().Inspected().Inspections()
	if len(entries) != 1 || entries[0].PolicyDecision().PolicyID() != "m3-npm-dynamic-inspect" || len(entries[0].Evidence()) < 3 || len(entries[0].Checks()) < 3 {
		t.Fatal("current fixture facts missing")
	}
	for _, check := range entries[0].Checks() {
		if check.Required() && (check.Capability() != domain.CapabilitySupported || check.Status() != domain.ExecutionCompleted) {
			t.Fatal("current fixture required check incomplete")
		}
	}
	for _, evidence := range entries[0].Evidence() {
		if _, _, err := store.ReadReference(ctx, entries[0].RunID(), evidence, entries[0].Artifact().Identity(), entries[0].Artifact().Digest()); err != nil {
			t.Fatal("current fixture Evidence unreadable", err)
		}
	}
	measured, err := (pythonConsoleExecutionBinder{static}).BindExecutionEntry(ctx, inspection, "haa-runtime-probe")
	if err != nil {
		t.Fatal(err)
	}
	execution, err := domain.NewExecutionContext(domain.ExecutionContextData{
		OperationID: inspection.OperationID(), Resolution: resolution, Set: inspection.Set(), EntryArtifact: measured.EntryArtifact(), Command: measured.Command(), EntryPath: measured.EntryPath(), EntryDigest: measured.EntryDigest(), EntrySize: measured.EntrySize(), RuntimeIdentity: measured.RuntimeIdentity(), Arguments: arguments,
	})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := static.ReadConsoleExecutionInventory(ctx, execution.EntryArtifact())
	if err != nil {
		t.Fatal(err)
	}
	entry, err := artifactpypi.SelectConsoleExecutionEntry(inventory, "haa-runtime-probe")
	if err != nil {
		t.Fatal(err)
	}
	return execution, entry
}
