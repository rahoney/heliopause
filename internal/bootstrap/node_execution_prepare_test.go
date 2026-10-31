package bootstrap

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/rahoney/heliopause/internal/application"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/evidence"
	"github.com/rahoney/heliopause/internal/evidence/local"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/policy"
	"github.com/rahoney/heliopause/internal/promotion"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	verificationnpm "github.com/rahoney/heliopause/internal/verification/npm"
)

// This fixture provides archive DATA and a fixed graph, never an npm/Docker
// execution. Real static reading, integrity records, Evidence and staging are
// exercised with the explicit M1 fixture Policy, not production dynamic proof.
type nodePreparationDataFixture struct {
	intake             string
	body               []byte
	resolved           domain.ResolvedArtifact
	resolution         domain.DependencyResolution
	digest             domain.ContentDigest
	integrity          string
	resolves, acquires int
}

func (f *nodePreparationDataFixture) Resolve(context.Context, domain.ArtifactReference) (domain.ResolvedArtifact, error) {
	return domain.ResolvedArtifact{}, errors.New("individual artifact re-resolution is forbidden")
}
func (f *nodePreparationDataFixture) ResolveExecutionDependencies(context.Context, domain.ArtifactReference) (domain.DependencyResolution, error) {
	f.resolves++
	return f.resolution, nil
}
func (f *nodePreparationDataFixture) Acquire(ctx context.Context, run domain.RunID, subject domain.ResolvedArtifact) (domain.AcquiredArtifact, error) {
	if err := ctx.Err(); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if subject != f.resolved {
		return domain.AcquiredArtifact{}, errors.New("unexpected fixture acquisition")
	}
	f.acquires++
	directory := filepath.Join(f.intake, run.String())
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	if err := os.WriteFile(filepath.Join(directory, "tarball.tgz"), f.body, 0o600); err != nil {
		return domain.AcquiredArtifact{}, err
	}
	return domain.NewAcquiredArtifactWithIntegrity(subject.Identity(), f.digest, "intake:"+run.String()+":tarball", uint64(len(f.body)), f.integrity, f.integrity)
}

func nodePreparationData(t *testing.T, program string) (*application.ExecutionPreparationService, *nodePreparationDataFixture, nodeExecutionEntryBinder, string, application.ExecutionPreparationRequest) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	intake, evidenceRoot, stagingRoot := filepath.Join(root, "intake"), filepath.Join(root, "evidence"), filepath.Join(root, "staging")
	var body bytes.Buffer
	gzipWriter := gzip.NewWriter(&body)
	writer := tar.NewWriter(gzipWriter)
	for _, member := range []struct{ name, body string }{
		{"package/cli.js", program},
		{"package/package.json", `{"name":"tiny","version":"1.2.3","bin":"cli.js"}`},
	} {
		if err := writer.WriteHeader(&tar.Header{Name: member.name, Mode: 0o644, Typeflag: tar.TypeReg, Size: int64(len(member.body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(member.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	sha := sha256.Sum256(body.Bytes())
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sha[:]))
	sri := sha512.Sum512(body.Bytes())
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(sri[:])
	source, _ := domain.NewSourceID("npm")
	identity, _ := domain.NewResolvedArtifactIdentity(source, "tiny", "1.2.3", "tarball")
	resolved, err := domain.NewResolvedArtifact(identity, "https://registry.npmjs.org/tiny/-/tiny-1.2.3.tgz", integrity)
	if err != nil {
		t.Fatal(err)
	}
	node, _ := domain.NewDependencyNodeID("tiny")
	dependency, err := domain.NewLockedDependency(node, domain.DependencyPrimary, resolved)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := domain.NewLockedDependencyGraph([]domain.LockedDependency{dependency}, nil)
	if err != nil {
		t.Fatal(err)
	}
	lock, _ := domain.NewSHA256Digest(hex.EncodeToString(sha[:]))
	resolution, err := domain.NewDependencyResolution(graph, "fixed-graph-DATA-fixture", lock)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &nodePreparationDataFixture{intake: intake, body: bytes.Clone(body.Bytes()), resolved: resolved, resolution: resolution, digest: digest, integrity: integrity}
	static, err := inspectionnpm.NewInspector(intake)
	if err != nil {
		t.Fatal(err)
	}
	store, err := local.NewStore(evidenceRoot)
	if err != nil {
		t.Fatal(err)
	}
	staging, err := promotion.NewLocalStaging(intake, evidenceRoot, stagingRoot)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := application.NewExecutionInspectService(fixture, verificationnpm.IntegrityVerifier{}, static, store, policy.M1{}, policy.M4{}, domain.NewOperationID, domain.NewRunID)
	if err != nil {
		t.Fatal(err)
	}
	binder := nodeExecutionEntryBinder{static}
	service, err := application.NewExecutionPreparationService(fixture, fresh, binder, evidence.Generator{}, staging)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := domain.NewArtifactReference(source, "tiny@1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	request, err := application.NewExecutionPreparationRequest(reference, "", []string{"private-argv-fixture", "a b", ""})
	if err != nil {
		t.Fatal(err)
	}
	return service, fixture, binder, stagingRoot, request
}

func TestNodeExecutionPreparationDataOnlyActualStorageHandoff(t *testing.T) {
	service, fixture, binder, stagingRoot, request := nodePreparationData(t, "console.log('DATA fixture')\n")
	prepared, err := service.Prepare(context.Background(), request)
	if err != nil || !prepared.Valid() {
		t.Fatal("real DATA handoff failed", err)
	}
	execution := prepared.ExecutionContext()
	if execution.RuntimeIdentity() != runtimeidentity.NodeImageReference || execution.Command() != "tiny" || execution.EntryPath() != "cli.js" || !slices.Equal(execution.Arguments(), request.Arguments()) || fixture.resolves != 1 || fixture.acquires != 1 {
		t.Fatal("Node pin/current request/one selection binding lost")
	}
	stagedRoot := filepath.Join(stagingRoot, prepared.Bundle().ManifestID().String())
	manifest, err := os.ReadFile(filepath.Join(stagedRoot, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	sbom, err := os.ReadFile(filepath.Join(stagedRoot, "sbom.cdx.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(manifest, prepared.Bundle().ManifestDocument()) || !bytes.Equal(sbom, prepared.Bundle().SBOMDocument()) || bytes.Contains(manifest, []byte("private-argv-fixture")) || bytes.Contains(sbom, []byte("private-argv-fixture")) {
		t.Fatal("staging changed records or persisted argv")
	}
	archives := 0
	if err := filepath.WalkDir(filepath.Join(stagedRoot, "artifacts"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Equal(body, fixture.body) {
			archives++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if archives != 1 {
		t.Fatal("exact original archive was not stored once", archives)
	}
	second, err := service.Prepare(context.Background(), request)
	if err != nil || !second.Valid() || second.Inspection().OperationID() == prepared.Inspection().OperationID() || second.Bundle().ManifestID() == prepared.Bundle().ManifestID() || fixture.resolves != 2 || fixture.acquires != 2 {
		t.Fatal("second preparation reused approval/records", err)
	}
	first := prepared.Inspection().Set().Inspected().Inspections()[0]
	if err := os.WriteFile(filepath.Join(fixture.intake, first.RunID().String(), "tarball.tgz"), []byte("tampered source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if bound, err := binder.BindExecutionEntry(context.Background(), prepared.Inspection(), ""); err == nil || bound.Valid() {
		t.Fatal("staging or old approval replaced changed original entry bytes")
	}
}

func TestNodeExecutionPreparationDataRejectsOtherInterpreterBeforeStaging(t *testing.T) {
	service, _, _, stagingRoot, request := nodePreparationData(t, "#!/usr/bin/env python\n")
	prepared, err := service.Prepare(context.Background(), request)
	if err == nil || prepared.Valid() || prepared.FailureCode() != "ENTRY_BINDING_FAILED" || !prepared.Inspection().Valid() {
		t.Fatal("other interpreter reached Node staging", err)
	}
	if _, err := os.Lstat(stagingRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed entry created a staging tree", err)
	}
}

func TestNPMExecutionPreparationFactoryRequiresActualRuntimeAndStorage(t *testing.T) {
	if service, err := newNPMExecutionPreparationService(nil, nil, nil, nil, nil, nil); err == nil || service != nil {
		t.Fatal("factory accepted missing actual adapters")
	}
	if bound, err := (nodeExecutionEntryBinder{}).BindExecutionEntry(context.Background(), application.ExecutionInspection{}, ""); err == nil || bound.Valid() {
		t.Fatal("binder accepted missing current inspection/reader")
	}
}
