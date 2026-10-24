package sandbox

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type pnpmProjectGateConsumer struct {
	prepareCalls, executeCalls int
	cause                      error
	inventories                []artifactnpm.ArchiveInventory
}

func (f *pnpmProjectGateConsumer) PrepareInspection(_ context.Context, _ artifactnpm.PNPMInspectionInput, inventories []artifactnpm.ArchiveInventory) (artifactnpm.PNPMInspectionLayout, error) {
	f.prepareCalls++
	f.inventories = inventories
	return artifactnpm.PNPMInspectionLayout{}, f.cause
}
func (f *pnpmProjectGateConsumer) ExecutePNPMInstance(context.Context, artifactnpm.PNPMInspectionLayout, domain.AcquiredArtifact, string) (domain.SandboxResult, error) {
	f.executeCalls++
	return domain.SandboxResult{}, errors.New("unexpected instance execution")
}

func TestPNPMProjectStaticGateBeforeRuntimeAndPreservesFirstCause(t *testing.T) {
	for _, kind := range []string{"valid-static", "static-finding", "source-error", "oversized-compressed"} {
		t.Run(kind, func(t *testing.T) {
			root, input, inventories := pnpmPreparationInputFixture(t, PNPMResolverRuntimeIdentity())
			original := input.Artifacts()[0]
			manifest := inventories[0].Manifest()
			if kind == "static-finding" {
				manifest = []byte(`{"name":"foreign","version":"6.0.0"}`)
			}
			var output bytes.Buffer
			gzipWriter := gzip.NewWriter(&output)
			writer := tar.NewWriter(gzipWriter)
			if err := writer.WriteHeader(&tar.Header{Name: "package/package.json", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(manifest))}); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write(manifest); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := gzipWriter.Close(); err != nil {
				t.Fatal(err)
			}
			body := output.Bytes()
			sum := sha256.Sum256(body)
			digest, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
			if err != nil {
				t.Fatal(err)
			}
			size := uint64(len(body))
			if kind == "oversized-compressed" {
				size = artifactnpm.PNPMInspectionWorkspaceLimit + 1
			}
			declared, _ := original.DeclaredIntegrity()
			observed, _ := original.ObservedIntegrity()
			subject, err := domain.NewAcquiredArtifactWithIntegrity(original.Identity(), digest, original.ContentHandle(), size, declared, observed)
			if err != nil {
				t.Fatal(err)
			}
			node, err := domain.ProjectDependencyNodeID(subject.Identity())
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(root, strings.Split(subject.ContentHandle(), ":")[1], "tarball.tgz")
			if err := os.WriteFile(file, body, 0o600); err != nil {
				t.Fatal(err)
			}
			if kind == "source-error" {
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
			}
			cause := errors.New("controlled preparation refusal")
			consumer := &pnpmProjectGateConsumer{cause: cause}
			static, err := inspectionnpm.NewInspector(root)
			if err != nil {
				t.Fatal(err)
			}
			inspector, err := inspectionnpm.NewPNPMProjectInspector(input.Context(), static, consumer, consumer)
			if err != nil {
				t.Fatal(err)
			}
			acquired := map[domain.DependencyNodeID]domain.AcquiredArtifact{node: subject}
			reports, err := inspector.InspectProject(context.Background(), input.Context().Snapshot(), acquired)
			if len(reports) != 1 || consumer.executeCalls != 0 {
				t.Fatal("failed gate lost report or executed", err)
			}
			report := reports[node]
			checks := report.Executions()
			if checks[len(checks)-1].Status() != domain.ExecutionNotExecuted {
				t.Fatal("gate failure completed required dynamic")
			}
			if kind == "valid-static" {
				if !errors.Is(err, cause) || consumer.prepareCalls != 1 || len(consumer.inventories) != 1 || !consumer.inventories[0].Matches(subject) {
					t.Fatal("static producer not connected before prepare", err)
				}
			} else if consumer.prepareCalls != 0 {
				t.Fatal("bad static input reached native preparation")
			}
			if kind == "static-finding" && (err != nil || len(report.Findings()) != 1) {
				t.Fatal("static finding lost", err)
			}
			if kind == "source-error" && err == nil {
				t.Fatal("source error lost", err)
			}
			if kind == "oversized-compressed" && err == nil {
				t.Fatal("compressed gate deferred")
			}
			if _, err := inspector.Inspect(context.Background(), subject); err == nil {
				t.Fatal("single primary fallback permitted")
			}
			delete(acquired, node)
			if _, err := inspector.InspectProject(context.Background(), input.Context().Snapshot(), acquired); err == nil {
				t.Fatal("missing acquired coverage accepted")
			}
		})
	}
}

func TestPNPMProjectExplicitEmptyDoesNotPrepareOrExecute(t *testing.T) {
	root := t.TempDir()
	manifest := []byte(`{}`)
	lockBody := []byte("lockfileVersion: '9.0'\nsettings:\n  autoInstallPeers: true\n  excludeLinksFromLockfile: false\nimporters:\n  .: {}\n")
	lock, err := artifactnpm.ParsePNPMLock(manifest, lockBody, runtimeidentity.PNPMVersion)
	if err != nil {
		t.Fatal(err)
	}
	source, err := artifactnpm.NewResolver("https://registry.npmjs.org/", &http.Client{Transport: pnpmPreparationRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("empty snapshot resolved network metadata")
		return nil, errors.New("unexpected source request")
	})}, root)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := source.BindPNPMLock(context.Background(), lock)
	if err != nil {
		t.Fatal(err)
	}
	target, err := domain.NewInstallTarget(filepath.Join(root, "empty"))
	if err != nil {
		t.Fatal(err)
	}
	install, err := domain.NewNPMProjectInstallContext(target)
	if err != nil {
		t.Fatal(err)
	}
	controls := []domain.ProjectControlFile{pnpmTestControl(t, "package.json", string(manifest), true), pnpmTestControl(t, "pnpm-lock.yaml", string(lockBody), true)}
	frozen, err := selection.InspectionContext(install, controls, PNPMResolverRuntimeIdentity())
	if err != nil {
		t.Fatal(err)
	}
	static, err := inspectionnpm.NewInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	consumer := &pnpmProjectGateConsumer{}
	inspector, err := inspectionnpm.NewPNPMProjectInspector(frozen, static, consumer, consumer)
	if err != nil {
		t.Fatal(err)
	}
	reports, err := inspector.InspectProject(context.Background(), frozen.Snapshot(), nil)
	if err != nil || reports == nil || len(reports) != 0 || consumer.prepareCalls != 0 || consumer.executeCalls != 0 {
		t.Fatal("explicit empty invented an execution", err)
	}
}
