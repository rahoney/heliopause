//go:build linux

package releasebootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	artifact "github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/releaseinstall"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
	"github.com/rahoney/heliopause/internal/verification/releaseprovenance"
)

type releaseSource interface {
	BootstrapMaterials() (releaseinstall.ReleaseManifest, [][]byte, error)
	OpenBootstrapMember(context.Context, string) (io.ReadCloser, uint64, error)
	Close() error
}
type preparationDependencies struct {
	source func(context.Context, string, string) (releaseSource, error)
	verify func(context.Context, releaseinstall.ReleaseManifest, [][]byte) ([]byte, error)
}

func (*Preparer) Prepare(ctx context.Context, version string) (ports.BootstrapPreparationResult, error) {
	return prepare(ctx, version, preparationDependencies{newPublicSource, verifyMaterials})
}

func newPublicSource(ctx context.Context, intake, version string) (releaseSource, error) {
	client, err := artifact.NewPublicClient(intake)
	if err != nil {
		return nil, err
	}
	target, err := domain.NewInstallTarget(releaseinstall.SystemInstallRoot)
	if err != nil {
		return nil, err
	}
	install, err := domain.NewGitHubAssetInstallContext(target)
	if err != nil {
		return nil, err
	}
	source, err := artifact.NewHeliopauseReleaseSet(client, version, runtimeidentity.RuntimeLockSHA256, install)
	if err != nil {
		return nil, err
	}
	_, err = source.ResolveDependencies(ctx, source.Reference(), install)
	return source, err
}

func bootstrapFiles(manifest releaseinstall.ReleaseManifest) ([]releaseinstall.ReleaseFile, error) {
	payloads, err := manifest.NativeUpdatePayloads()
	if err != nil {
		return nil, err
	}
	files := []releaseinstall.ReleaseFile{{Name: "helox-release-manifest.json", Role: "release-metadata", SHA256: manifest.SHA256()}}
	files = append(files, payloads...)

	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}

func verifyMaterials(ctx context.Context, manifest releaseinstall.ReleaseManifest, proofs [][]byte) ([]byte, error) {
	if len(proofs) == 0 || len(proofs) > 16 {
		return nil, errors.New("bounded provenance is required")
	}
	files, err := bootstrapFiles(manifest)
	if err != nil {
		return nil, err
	}
	subjects := make([]releaseprovenance.Subject, 0, len(files))
	for _, file := range files {
		subjects = append(subjects, releaseprovenance.Subject{Name: file.Name, SHA256: file.SHA256})
	}
	expected := releaseprovenance.Identity{Version: manifest.Version(), SourceCommit: manifest.SourceCommit(), WorkflowRun: manifest.WorkflowRun()}
	var firstErr error
	for _, proof := range proofs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(proof) == 0 || len(proof) > 1<<20 {
			return nil, errors.New("provenance body exceeds bound")
		}
		if _, err := releaseprovenance.Verify(ctx, proof, expected, subjects); err == nil {
			return append([]byte(nil), proof...), nil
		} else if firstErr == nil {
			firstErr = err
		}
	}
	return nil, errors.Join(errors.New("official provenance was not verified"), firstErr)
}

func prepare(ctx context.Context, version string, deps preparationDependencies) (result ports.BootstrapPreparationResult, resultErr error) {
	fail := func(code ports.BootstrapFailureCode, err error) (ports.BootstrapPreparationResult, error) {
		return result, ports.NewBootstrapPreparationFailure(code, err)
	}
	if ctx == nil {
		return fail(ports.BootstrapExportFailed, errors.New("bootstrap context is required"))
	}
	if err := ctx.Err(); err != nil {
		return fail(ports.BootstrapCancelled, err)
	}
	if runtime.GOARCH != "amd64" {
		return fail(ports.BootstrapUnsupported, errors.New("native preparation requires Linux amd64"))
	}
	if os.Geteuid() == 0 {
		return fail(ports.BootstrapRoot, errors.New("native acquisition runs without root"))
	}
	if err := artifact.ValidateHeliopauseVersion(version); err != nil {
		return fail(ports.BootstrapInvalid, err)
	}
	if deps.source == nil || deps.verify == nil {
		return fail(ports.BootstrapExportFailed, errors.New("bootstrap providers are required"))
	}
	result.Version = version
	output, err := newOwnedOutput()
	if err != nil {
		return fail(ports.BootstrapExportFailed, err)
	}
	result.Directory = output.path
	var source releaseSource
	defer func() {
		var cleanupErr error
		if source != nil {
			cleanupErr = source.Close()
		}
		cleanupErr = errors.Join(cleanupErr, output.removeIntake())
		if resultErr == nil && cleanupErr == nil {
			if err := output.verifyAll(ctx); err != nil {
				resultErr = ports.NewBootstrapPreparationFailure(ports.BootstrapExportFailed, err)
			}
		}
		if resultErr != nil || cleanupErr != nil {
			cleanupErr = errors.Join(cleanupErr, output.discardFiles())
		} else {
			cleanupErr = output.sync()
		}
		cleanupErr = errors.Join(cleanupErr, output.close())
		if cleanupErr != nil {
			failure := ports.NewBootstrapPreparationFailure(ports.BootstrapCleanupFailed, cleanupErr)
			resultErr = errors.Join(resultErr, failure)
		}
		if resultErr != nil {
			if err := output.removeDirectory(); err == nil {
				result.Directory = ""
			} else {
				resultErr = errors.Join(resultErr, ports.NewBootstrapPreparationFailure(ports.BootstrapCleanupFailed, err))
			}
		}
	}()
	ports.ReportProgress(ctx, ports.Progress{Stage: "resolve"})
	source, err = deps.source(ctx, filepath.Join(output.path, "intake"), version)
	if err != nil {
		return fail(ports.BootstrapAcquisitionFailed, err)
	}
	if source == nil {
		return fail(ports.BootstrapAcquisitionFailed, errors.New("bootstrap source is missing"))
	}
	manifest, proofs, err := source.BootstrapMaterials()
	if err != nil {
		return fail(ports.BootstrapAcquisitionFailed, err)
	}
	if manifest.Version() != version || manifest.RuntimeLockSHA256() != runtimeidentity.RuntimeLockSHA256 || manifest.GVisorRelease() != runtimeidentity.GVisorRelease || manifest.GVisorCommit() != runtimeidentity.GVisorCommit {
		return fail(ports.BootstrapProvenanceFailed, errors.New("release differs from requested compiled runtime"))
	}
	files, err := bootstrapFiles(manifest)
	if err != nil {
		return fail(ports.BootstrapProvenanceFailed, err)
	}
	ports.ReportProgress(ctx, ports.Progress{Stage: "verify"})
	proof, err := deps.verify(ctx, manifest, proofs)
	if err != nil {
		return fail(ports.BootstrapProvenanceFailed, err)
	}
	if len(proof) == 0 || len(proof) > 1<<20 {
		return fail(ports.BootstrapProvenanceFailed, errors.New("verified provenance body is invalid"))
	}
	var total uint64
	for index, file := range files {
		reader, size, err := source.OpenBootstrapMember(ctx, file.Name)
		if err != nil {
			return fail(ports.BootstrapExportFailed, err)
		}
		if reader == nil {
			return fail(ports.BootstrapExportFailed, errors.New("bootstrap member reader is missing"))
		}
		limit := uint64(512 << 20)
		if file.Name == "helox-release-manifest.json" || file.Name == "helox-runtime-images.json" {
			limit = 1 << 20
		}
		if file.Name == "helox-release-sbom.cdx.json" {
			limit = 8 << 20
		}
		if size == 0 || size > limit || size > (1<<30)-total || file.Size > 0 && size != uint64(file.Size) {
			return fail(ports.BootstrapExportFailed, errors.Join(errors.New("bootstrap member exceeds declared size or budget"), reader.Close()))
		}
		ports.ReportProgress(ctx, ports.Progress{Stage: "stage", Current: int64(index), Total: int64(len(files))})
		err = output.copy(ctx, file.Name, reader, size, file.SHA256)
		err = errors.Join(err, reader.Close())
		if err != nil {
			return fail(ports.BootstrapExportFailed, err)
		}
		total += size
		result.Files = append(result.Files, file.Name)
	}
	sbom, err := output.read("helox-release-sbom.cdx.json", 8<<20)
	if err != nil {
		return fail(ports.BootstrapExportFailed, err)
	}
	if err := releaseinstall.ValidateReleaseSBOM(sbom, manifest); err != nil {
		return fail(ports.BootstrapProvenanceFailed, err)
	}
	images, err := output.read("helox-runtime-images.json", 1<<20)
	if err != nil {
		return fail(ports.BootstrapExportFailed, err)
	}
	if err := releaseinstall.ValidateRuntimeImagesForRelease(images, manifest); err != nil {
		return fail(ports.BootstrapProvenanceFailed, err)
	}
	notices, err := output.read("pnpm-NOTICE.txt", 64<<10)
	if err != nil {
		return fail(ports.BootstrapProvenanceFailed, err)
	}
	if err := releaseinstall.ValidatePNPMNotices(notices); err != nil {
		return fail(ports.BootstrapProvenanceFailed, err)
	}
	bunNotices, err := output.read("bun-NOTICE.txt", 64<<10)
	if err != nil {
		return fail(ports.BootstrapProvenanceFailed, err)
	}
	if err := releaseinstall.ValidateBunNotices(bunNotices); err != nil {
		return fail(ports.BootstrapProvenanceFailed, err)
	}
	if manifest.HasPipxRuntime() {
		pipxNotices, err := output.read("pipx-NOTICE.txt", 64<<10)
		if err != nil {
			return fail(ports.BootstrapProvenanceFailed, err)
		}
		if err := releaseinstall.ValidatePipxNotices(pipxNotices); err != nil {
			return fail(ports.BootstrapProvenanceFailed, err)
		}
	}

	sum := sha256.Sum256(proof)
	digest := hex.EncodeToString(sum[:])
	proofName := digest + ".json"
	if err := output.copy(ctx, proofName, bytes.NewReader(proof), uint64(len(proof)), digest); err != nil {
		return fail(ports.BootstrapExportFailed, err)
	}
	result.Proofs = []string{proofName}
	result.ManifestSHA256 = manifest.SHA256()
	if err := output.sync(); err != nil {
		return fail(ports.BootstrapExportFailed, err)
	}
	return result, nil
}
