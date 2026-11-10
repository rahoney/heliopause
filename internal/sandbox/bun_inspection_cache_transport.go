package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func bunInspectionSeedManifest(input artifactnpm.BunInspectionInput, archives []pnpmInspectionArchive) ([]byte, error) {
	if !input.Valid() || len(archives) == 0 || len(archives) != len(input.Artifacts()) {
		return nil, errBunInspectionPreparation
	}
	ordered := append([]pnpmInspectionArchive(nil), archives...)
	slices.SortFunc(ordered, func(a, b pnpmInspectionArchive) int {
		x, y := a.subject.Identity(), b.subject.Identity()
		return strings.Compare(x.Name()+"@"+x.Version(), y.Name()+"@"+y.Version())
	})
	dependencies, overrides := map[string]string{}, map[string]string{}
	seen := map[domain.AcquiredArtifact]bool{}
	for _, archive := range ordered {
		if !slices.Contains(input.Artifacts(), archive.subject) || seen[archive.subject] || archive.name != "artifacts/"+archive.subject.Digest().String()+".tgz" {
			return nil, errBunInspectionPreparation
		}
		seen[archive.subject] = true
		name := "haa-cache-" + archive.subject.Digest().String()
		if _, duplicate := dependencies[name]; duplicate {
			return nil, errBunInspectionPreparation
		}
		file := "file:" + bunProjectRoot + "/" + archive.name
		dependencies[name] = file
		key := archive.subject.Identity().Name() + "@*"
		if _, present := overrides[key]; !present {
			overrides[key] = file
		}
	}
	body, err := json.Marshal(map[string]any{"name": "haa-bun-cache-seed", "private": true, "dependencies": dependencies, "overrides": overrides})
	if err != nil || len(body) > artifactnpm.ProjectControlLimit {
		return nil, errors.Join(errBunInspectionPreparation, err)
	}
	return body, nil
}

func copyBunInspectionSeed(ctx context.Context, runner CommandRunner, container string, input artifactnpm.BunInspectionInput, archives []pnpmInspectionArchive) error {
	writer, ok := runner.(inputCommandRunner)
	streams, streamOK := runner.(streamingDockerOutput)
	if ctx == nil || !ok || !streamOK || !exactObservationContainerID(container) {
		return errBunInspectionPreparation
	}
	manifest, err := bunInspectionSeedManifest(input, archives)
	if err != nil {
		return err
	}
	var output bytes.Buffer
	tarWriter := tar.NewWriter(&output)
	for _, name := range []string{"artifacts", "seed"} {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0o700, Uid: 1000, Gid: 1000}); err != nil {
			return errors.Join(errBunInspectionPreparation, err)
		}
	}
	emit := func(name string, body []byte) error {
		if err := ctx.Err(); err != nil {
			return errors.Join(errBunInspectionPreparation, err, context.Cause(ctx))
		}
		if len(body) > artifactnpm.BunInspectionWorkspaceLimit || output.Len() > artifactnpm.BunInspectionWorkspaceLimit-len(body) {
			return errBunInspectionPreparation
		}
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o600, Uid: 1000, Gid: 1000, Size: int64(len(body))}); err != nil {
			return err
		}
		_, err := tarWriter.Write(body)
		return err
	}
	if err := emit("seed/package.json", manifest); err != nil {
		return err
	}
	for _, archive := range archives {
		hash := sha256.Sum256(archive.body)
		if uint64(len(archive.body)) != archive.subject.SizeBytes() || hex.EncodeToString(hash[:]) != archive.subject.Digest().String() {
			return errBunInspectionPreparation
		}
		if err := emit(archive.name, archive.body); err != nil {
			return err
		}
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}
	if output.Len() > artifactnpm.BunInspectionWorkspaceLimit {
		return errBunInspectionPreparation
	}
	if err := writer.RunInput(ctx, &output, "docker", "cp", "--archive", "-", container+":"+bunProjectRoot); err != nil {
		return errors.Join(errBunInspectionPreparation, err)
	}
	for _, archive := range archives {
		if err := verifyNativeNPMGuestArchive(ctx, streams, container, nativeRuntimeBun, archive); err != nil {
			return errors.Join(errBunInspectionPreparation, err)
		}
	}
	return nil
}

func readBunInspectionCache(ctx context.Context, runner streamingDockerOutput, container string, input artifactnpm.BunInspectionInput, inventory artifactnpm.ArchiveInventory) (artifactnpm.BunNativeCacheEntry, error) {
	if ctx == nil || runner == nil || !exactObservationContainerID(container) {
		return artifactnpm.BunNativeCacheEntry{}, errBunInspectionPreparation
	}
	folder, err := input.NativeCacheFolder(inventory.Artifact())
	if err != nil {
		return artifactnpm.BunNativeCacheEntry{}, err
	}
	output := nativeToolArchiveOutput{limit: artifactnpm.PNPMLayoutArchiveLimit}
	if err := runner.RunOutput(ctx, &output, "docker", "cp", container+":"+bunProjectRoot+"/cache/"+folder, "-"); err != nil || output.failed {
		return artifactnpm.BunNativeCacheEntry{}, errors.Join(errBunInspectionPreparation, err)
	}
	return input.BindNativeCache(ctx, inventory, output.body)
}
