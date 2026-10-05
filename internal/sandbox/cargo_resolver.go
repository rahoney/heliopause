package sandbox

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

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

func CargoResolverEnvironmentForHome(home string) ([]string, error) {
	if !filepath.IsAbs(home) || filepath.Clean(home) != home || home == "/" {
		return nil, errors.New("cargo resolver home is invalid")
	}
	environment := CargoResolverEnvironment()
	environment[0] = "CARGO_HOME=" + home
	return environment, nil
}

func ValidateCargoResolverEnvironment(environment []string, home string) error {
	expected, err := CargoResolverEnvironmentForHome(home)
	if err != nil || len(environment) != len(expected) {
		return errors.New("cargo resolver environment is not canonical")
	}
	for index := range expected {
		if environment[index] != expected[index] {
			return errors.New("cargo resolver environment is not canonical")
		}
	}
	return nil
}

func (r *CargoResolver) ResolveDependencies(ctx context.Context, reference domain.ArtifactReference, installContext domain.InstallContext) (domain.DependencyResolution, error) {
	if r == nil || r.runner == nil || ctx == nil || reference.Source() != artifactcargo.Source() || !installContext.Valid() {
		return domain.DependencyResolution{}, errors.New("valid Cargo resolver request is required")
	}
	if err := ctx.Err(); err != nil {
		return domain.DependencyResolution{}, err
	}
	project := filepath.Clean(installContext.Target().String())
	if !filepath.IsAbs(project) || project == "/" {
		return domain.DependencyResolution{}, errors.New("cargo project path is invalid")
	}
	selected, err := r.resolveLockedProject(ctx, installContext)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	records, edges, _, err := artifactcargo.ParseLockedMetadata(selected.metadata, selected.files["Cargo.lock"], selected.metadataRoot)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	graph, err := artifactcargo.BuildLockedGraph(reference, records, edges)
	if err != nil {
		return domain.DependencyResolution{}, err
	}
	return domain.NewDependencyResolution(graph, cargoResolutionDescriptor, selected.snapshot.GraphDigest())
}

func readCargoControlFiles(project string) ([]byte, []byte, error) {
	root, err := os.OpenRoot(project)
	if err != nil {
		return nil, nil, errors.New("cargo project controls are unavailable")
	}
	defer root.Close()
	manifest, err := readCargoControlFile(root, "Cargo.toml")
	if err != nil {
		return nil, nil, err
	}
	lock, err := readCargoControlFile(root, "Cargo.lock")
	if err != nil {
		return nil, nil, err
	}
	return manifest, lock, nil
}

func readCargoControlFile(root *os.Root, name string) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > artifactcargo.MaxProjectControlBytes {
		return nil, errors.New("cargo control is not bounded regular content")
	}
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errors.New("open Cargo control file")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return nil, errors.New("cargo control identity changed")
	}
	body, err := io.ReadAll(io.LimitReader(file, artifactcargo.MaxProjectControlBytes+1))
	after, statErr := file.Stat()
	current, currentErr := root.Lstat(name)
	if err != nil || statErr != nil || currentErr != nil || !os.SameFile(info, current) || len(body) != int(info.Size()) || after.Size() != info.Size() || after.ModTime() != info.ModTime() {
		return nil, errors.New("cargo control changed during bounded read")
	}
	return body, nil
}
