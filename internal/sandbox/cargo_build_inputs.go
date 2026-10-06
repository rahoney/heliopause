package sandbox

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func cargoBuildInputLimits(prefix string) projectBuildInputLimits {
	return projectBuildInputLimits{variant: map[string]string{"project": "cargo-source", "cache": "cargo-cache"}[prefix], maxPath: 4096, files: artifactcargo.MaxCrateFiles, payload: artifactcargo.MaxCrateExpandedBytes, file: artifactcargo.MaxCrateFileBytes, control: artifactcargo.MaxProjectControlBytes, controls: [2]string{"Cargo.toml", "Cargo.lock"}}
}

type CargoBuildSourceReader struct{ intake string }

func NewCargoBuildSourceReader(intake string) (*CargoBuildSourceReader, error) {
	if !filepath.IsAbs(intake) || filepath.Clean(intake) != intake || intake == "/" {
		return nil, errors.New("cargo build intake is invalid")
	}
	return &CargoBuildSourceReader{intake}, nil
}
func (r *CargoBuildSourceReader) VerifyBuildSource(ctx context.Context, source domain.AcquiredArtifact) (resultErr error) {
	if r == nil || ctx == nil {
		return errors.New("cargo source reader is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	input, err := openProjectBuildInput(r.intake, source, "project", cargoBuildInputLimits("project"))
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, input.close()) }()
	manifest := &goBuildInputManifest{members: map[string]goBuildInputMember{}, controls: map[string][]byte{}, storage: 16 << 20}
	if err := input.scan(ctx, manifest, nil); err != nil {
		return err
	}
	if len(manifest.controls) != 2 || artifactcargo.ValidateProjectManifest(manifest.controls["Cargo.toml"], "Cargo.toml") != nil || artifactcargo.ValidateProjectLock(manifest.controls["Cargo.lock"]) != nil {
		return errors.New("cargo build controls are invalid")
	}
	return nil
}

func prepareCargoBuildInputs(ctx context.Context, intake string, inputs domain.ProjectBuildInputs) (_ *goBuildPreparedInputs, resultErr error) {
	if ctx == nil || ctx.Err() != nil || !inputs.Valid() || inputs.Kind() != "cargo" || inputs.Selector() != "default" || inputs.Snapshot().Source() != artifactcargo.Source() {
		return nil, errors.New("cargo build inputs are invalid")
	}
	prepared := &goBuildPreparedInputs{manifest: goBuildInputManifest{members: map[string]goBuildInputMember{}, controls: map[string][]byte{}, storage: 16 << 20, identity: goBuildManifestIdentity(inputs.Source(), inputs.Cache())}}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, prepared.close())
		}
	}()
	var err error
	prepared.source, err = openProjectBuildInput(intake, inputs.Source(), "project", cargoBuildInputLimits("project"))
	if err != nil {
		return nil, err
	}
	prepared.cache, err = openProjectBuildInput(intake, inputs.Cache(), "cache", cargoBuildInputLimits("cache"))
	if err != nil {
		return nil, err
	}
	for _, input := range []*goBuildInputFile{prepared.source, prepared.cache} {
		if err := input.scan(ctx, &prepared.manifest, nil); err != nil {
			return nil, err
		}
	}
	project := sha256.Sum256([]byte(inputs.Snapshot().Context().Target().String()))
	key := hex.EncodeToString(project[:])
	if inputs.Source().Identity().Name() != "project-"+key || inputs.Source().Identity().Version() != "snapshot" || inputs.Cache().Identity().Name() != "cache-"+key || len(inputs.Snapshot().ControlDigests()) != 2 || len(prepared.manifest.controls) != 2 {
		return nil, errors.New("cargo build project binding differs")
	}
	for _, control := range inputs.Snapshot().ControlDigests() {
		body, ok := prepared.manifest.controls[control.Name()]
		hash := sha256.Sum256(body)
		if !ok || hex.EncodeToString(hash[:]) != control.Digest().String() {
			return nil, errors.New("cargo build control binding differs")
		}
	}
	if artifactcargo.ValidateProjectManifest(prepared.manifest.controls["Cargo.toml"], "Cargo.toml") != nil || artifactcargo.ValidateProjectLock(prepared.manifest.controls["Cargo.lock"]) != nil {
		return nil, errors.New("cargo build controls are unsupported")
	}
	// Fixed controller configuration joins the fully rehashed read-only input
	// inventory. Build code cannot replace it through rename/unlink aliases.
	prepared.cargoConfiguration = true
	prepared.manifest.storage += 3 * 4096
	if prepared.manifest.storage > goBuildInputCapacity {
		return nil, errors.New("cargo build configuration exceeds input capacity")
	}
	prepared.manifest.members["cargo-home"] = goBuildInputMember{kind: tar.TypeDir}
	hash := sha256.Sum256([]byte(cargoBuildSourceConfiguration))
	prepared.manifest.members["cargo-home/config.toml"] = goBuildInputMember{kind: tar.TypeReg, size: int64(len(cargoBuildSourceConfiguration)), sha256: hex.EncodeToString(hash[:])}
	return prepared, nil
}

func writeCargoBuildConfiguration(writer *tar.Writer) error {
	for _, header := range []*tar.Header{
		{Name: "cargo-home", Typeflag: tar.TypeDir, Mode: 0o500, Uid: 1000, Gid: 1000},
		{Name: "cargo-home/config.toml", Typeflag: tar.TypeReg, Mode: 0o400, Uid: 1000, Gid: 1000, Size: int64(len(cargoBuildSourceConfiguration))},
	} {
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
	}
	_, err := writer.Write([]byte(cargoBuildSourceConfiguration))
	return err
}
