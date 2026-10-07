package sandbox

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type goBuildPreparedInputs struct {
	source, cache      *goBuildInputFile
	manifest           goBuildInputManifest
	cargoConfiguration bool
}

func prepareGoBuildInputs(ctx context.Context, intake string, snapshot domain.ProjectDependencySnapshot, source, cache domain.AcquiredArtifact) (_ *goBuildPreparedInputs, resultErr error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, errGoBuildInput
	}
	prepared := &goBuildPreparedInputs{manifest: goBuildInputManifest{
		members: map[string]goBuildInputMember{}, controls: map[string][]byte{}, storage: 16 << 20,
		identity: goBuildManifestIdentity(source, cache),
	}}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, prepared.close())
		}
	}()
	var err error
	prepared.source, err = openGoBuildInput(intake, source, "project")
	if err != nil {
		return nil, err
	}
	prepared.cache, err = openGoBuildInput(intake, cache, "cache")
	if err != nil {
		return nil, err
	}
	for _, input := range []*goBuildInputFile{prepared.source, prepared.cache} {
		if err := input.scan(ctx, &prepared.manifest, nil); err != nil {
			return nil, err
		}
	}
	if err := validateGoBuildInputBinding(snapshot, source, cache, prepared.manifest); err != nil {
		return nil, err
	}
	return prepared, nil
}

func (p *goBuildPreparedInputs) close() error {
	var err error
	if p.source != nil {
		err = errors.Join(err, p.source.close())
	}
	if p.cache != nil {
		err = errors.Join(err, p.cache.close())
	}
	return err
}

func (p *goBuildPreparedInputs) introduce(ctx context.Context, runner CommandRunner, container string) error {
	return p.introduceAt(ctx, runner, container, goBuildGuestInput)
}

func (p *goBuildPreparedInputs) introduceAt(ctx context.Context, runner CommandRunner, container, destination string) error {
	input, ok := runner.(inputCommandRunner)
	if p == nil || ctx == nil || !ok || !exactObservationContainerID(container) {
		return errGoBuildInput
	}
	reader, pipe := io.Pipe()
	result := make(chan error, 1)
	go func() {
		writer := tar.NewWriter(pipe)
		var err error
		for _, root := range []string{"project", "cache"} {
			if err = writer.WriteHeader(&tar.Header{Name: root, Typeflag: tar.TypeDir, Mode: 0o500, Uid: 1000, Gid: 1000}); err != nil {
				break
			}
		}
		if err == nil {
			err = p.source.scan(ctx, &p.manifest, writer)
		}
		if err == nil {
			err = p.cache.scan(ctx, &p.manifest, writer)
		}
		if err == nil && p.cargoConfiguration {
			err = writeCargoBuildConfiguration(writer)
		}
		err = errors.Join(err, writer.Close())
		_ = pipe.CloseWithError(err)
		result <- err
	}()
	// Only a newly-created, identity-checked preparation volume receives the
	// reconstructed stream. No artifact-controlled path enters this command.
	commandErr := input.RunInput(ctx, reader, "docker", "cp", "--archive", "-", container+":"+destination)
	_ = reader.CloseWithError(commandErr)
	copyErr := <-result
	if commandErr != nil || copyErr != nil || ctx.Err() != nil {
		return errors.Join(errGoBuildInput, copyErr, ctx.Err())
	}
	return nil
}

func verifyGoBuildMaterialization(ctx context.Context, runner CommandRunner, container string, manifest goBuildInputManifest) error {
	return verifyProjectBuildMaterialization(ctx, runner, container, goBuildGuestInput, manifest)
}

func verifyProjectBuildMaterialization(ctx context.Context, runner CommandRunner, container, destination string, manifest goBuildInputManifest) error {
	output, ok := runner.(streamingDockerOutput)
	if ctx == nil || !ok || !exactObservationContainerID(container) {
		return errGoBuildInput
	}
	reader, writer := io.Pipe()
	result := make(chan error, 1)
	go func() {
		err := output.RunOutput(ctx, writer, "docker", "cp", container+":"+destination+"/.", "-")
		_ = writer.CloseWithError(err)
		result <- err
	}()
	err := scanGoBuildMaterialization(ctx, reader, manifest)
	_ = reader.CloseWithError(err)
	commandErr := <-result
	if err != nil || commandErr != nil || ctx.Err() != nil {
		return errors.Join(errGoBuildInput, ctx.Err())
	}
	return nil
}

func scanGoBuildMaterialization(ctx context.Context, input io.Reader, manifest goBuildInputManifest) error {
	if ctx == nil || input == nil || len(manifest.members) == 0 {
		return errGoBuildInput
	}
	// Bound the entire transport, including extended headers and padding.
	bounded := &io.LimitedReader{R: input, N: goBuildInputCapacity + (4*(goBuildCacheFiles+10000)+16)*512}
	archive := tar.NewReader(bounded)
	seen := map[string]bool{}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || len(seen) > len(manifest.members)+2 {
			return errGoBuildInput
		}
		name := strings.TrimSuffix(strings.TrimPrefix(header.Name, "./"), "/")
		if name == "." || name == "" {
			if header.Typeflag != tar.TypeDir || header.Size != 0 || header.Uid != 1000 || header.Gid != 1000 || header.Mode != 0o700 || seen["."] {
				return errGoBuildInput
			}
			seen["."] = true
			continue
		}
		if !validClosureDestination(name) || seen[name] || header.Uid != 1000 || header.Gid != 1000 || header.Linkname != "" {
			return errGoBuildInput
		}
		seen[name] = true
		if name == "project" || name == "cache" {
			if header.Typeflag != tar.TypeDir || header.Size != 0 || header.Mode != 0o500 {
				return errGoBuildInput
			}
			continue
		}
		expected, ok := manifest.members[name]
		if !ok || header.Size != expected.size || header.Typeflag != expected.kind {
			return errGoBuildInput
		}
		if expected.kind == tar.TypeDir {
			if header.Mode != 0o500 {
				return errGoBuildInput
			}
			continue
		}
		if header.Mode != 0o400 || expected.kind != tar.TypeReg {
			return errGoBuildInput
		}
		hash := sha256.New()
		if _, err := io.CopyN(hash, archive, expected.size); err != nil || hex.EncodeToString(hash.Sum(nil)) != expected.sha256 {
			return errGoBuildInput
		}
	}
	if !seen["project"] || !seen["cache"] {
		return errGoBuildInput
	}
	for name := range manifest.members {
		if !seen[name] {
			return errGoBuildInput
		}
	}
	if _, err := io.Copy(io.Discard, bounded); err != nil || bounded.N == 0 {
		return errGoBuildInput
	}
	return nil
}
