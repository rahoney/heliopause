package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"

	artifactcargo "github.com/rahoney/heliopause/internal/artifact/cargo"
	"github.com/rahoney/heliopause/internal/core/domain"
)

type CargoRunner interface {
	RunCargo(context.Context, string, []string, ...string) ([]byte, error)
}

type CargoResolver struct{ runner CargoRunner }

func NewCargoResolver(runner CargoRunner) (*CargoResolver, error) {
	if runner == nil {
		return nil, errors.New("cargo resolver requires trusted Cargo runner")
	}
	return &CargoResolver{runner: runner}, nil
}

func CargoResolverEnvironment() []string {
	return []string{
		"CARGO_HOME=/tmp/heliopause-cargo-home",
		"CARGO_NET_OFFLINE=false",
		"CARGO_NET_GIT_FETCH_WITH_CLI=false",
		"CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse",
		"CARGO_REGISTRIES_CRATES_IO_INDEX=https://index.crates.io/",
	}
}

func (r *CargoResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (domain.DependencyResolution, error) {
	if r == nil || r.runner == nil || ctx == nil || reference.Source() != artifactcargo.Source() || !installContext.Valid() {
		return domain.DependencyResolution{}, errors.New("valid Cargo resolver request is required")
	}
	project := filepath.Clean(installContext.Target().String())
	if !filepath.IsAbs(project) || project == "/" {
		return domain.DependencyResolution{}, errors.New("cargo project path is invalid")
	}
	environment := CargoResolverEnvironment()
	body, err := r.runner.RunCargo(ctx, project, environment, "metadata", "--locked", "--format-version", "1")
	if err != nil {
		return domain.DependencyResolution{}, errors.New("cargo metadata resolution failed")
	}
	records, edges, err := artifactcargo.ParseMetadata(body)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	graph, err := artifactcargo.BuildLockedGraph(reference, records, edges)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	digestBytes := sha256.Sum256(body)
	digest, err := domain.NewSHA256Digest(hex.EncodeToString(digestBytes[:]))
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return domain.NewDependencyResolution(graph, "cargo:crates.io;sparse:index.crates.io;env:"+strings.Join(environment, ";"), digest)
}
