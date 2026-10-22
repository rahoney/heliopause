package sandbox

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

const uvNativeLockLimit = 4 << 20

type uvNativeOutput struct {
	lock     []byte
	controls []artifactpypi.UVInstalledWheelControls
}

// readUVNativeOutput transports unapproved native data only. The caller must
// authenticate the separately acquired archives before returning any graph.
func readUVNativeOutput(ctx context.Context, runner CommandRunner, containerID string) (uvNativeOutput, error) {
	return readUVNativeTar(ctx, runner, containerID, pythonSitePath+"/.", scanUVNativeOutput)
}

func readUVNativeLock(ctx context.Context, runner CommandRunner, containerID string) ([]byte, error) {
	data, err := readUVNativeTar(ctx, runner, containerID, pythonSitePath+"/pylock.toml", scanUVNativeLock)
	return data.lock, err
}

func readUVNativeTar(ctx context.Context, runner CommandRunner, containerID, guestPath string, scan func(io.Reader) (uvNativeOutput, error)) (uvNativeOutput, error) {
	output, ok := runner.(streamingDockerOutput)
	if !ok || ctx == nil || scan == nil || !exactObservationContainerID(containerID) || (guestPath != pythonSitePath+"/." && guestPath != pythonSitePath+"/pylock.toml") {
		return uvNativeOutput{}, errors.New("uv native output transport is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return uvNativeOutput{}, err
	}
	reader, writer := io.Pipe()
	commandResult := make(chan error, 1)
	go func() {
		err := output.RunOutput(ctx, writer, "docker", "cp", containerID+":"+guestPath, "-")
		_ = writer.CloseWithError(err)
		commandResult <- err
	}()
	data, scanErr := scan(reader)
	_ = reader.CloseWithError(scanErr)
	commandErr := <-commandResult
	if err := errors.Join(scanErr, commandErr, ctx.Err()); err != nil {
		return uvNativeOutput{}, fmt.Errorf("uv native output collection failed: %w", err)
	}
	return data, nil
}

func scanUVNativeLock(input io.Reader) (uvNativeOutput, error) {
	if input == nil {
		return uvNativeOutput{}, errors.New("uv native lock stream is missing")
	}
	bounded := &io.LimitedReader{R: input, N: uvNativeLockLimit + (16 << 10)}
	archive := tar.NewReader(bounded)
	header, err := archive.Next()
	if err != nil || strings.TrimPrefix(header.Name, "./") != "pylock.toml" || (header.Typeflag != tar.TypeReg && header.Typeflag != 0) || header.Uid != 1000 || header.Gid != 1000 || header.Mode&^int64(0o777) != 0 || header.Mode&0o022 != 0 || header.Size < 1 || header.Size > uvNativeLockLimit {
		return uvNativeOutput{}, errors.New("uv native lock file identity, mode or size is invalid")
	}
	body, err := io.ReadAll(archive)
	if err != nil || int64(len(body)) != header.Size {
		return uvNativeOutput{}, errors.New("uv native lock is truncated")
	}
	if _, err := archive.Next(); !errors.Is(err, io.EOF) {
		return uvNativeOutput{}, errors.New("uv native lock tar has extra entries or is invalid")
	}
	if err := drainUVTarPadding(bounded); err != nil {
		return uvNativeOutput{}, err
	}
	if bounded.N == 0 {
		return uvNativeOutput{}, errors.New("uv native lock tar exceeds byte bound")
	}
	return uvNativeOutput{lock: body}, nil
}

func scanUVNativeOutput(input io.Reader) (uvNativeOutput, error) {
	if input == nil {
		return uvNativeOutput{}, errors.New("uv native output stream is missing")
	}
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	maxEntries := policy.MaxGraphFiles()*8 + int64(policy.MaxGraphArtifacts())*4 + 64
	// Account separately for file data, bounded tar headers and end padding.
	bounded := &io.LimitedReader{R: input, N: policy.MaxGraphUncompressed() + uvNativeLockLimit + maxEntries*1024 + 10240}
	archive := tar.NewReader(bounded)
	seen := make(map[string]bool)
	controls := make(map[string]*artifactpypi.UVInstalledWheelControls)
	var result uvNativeOutput
	var entries, expanded, metadataBytes int64
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return uvNativeOutput{}, errors.New("uv native tar is truncated or invalid")
		}
		entries++
		name := strings.TrimSuffix(strings.TrimPrefix(header.Name, "./"), "/")
		root := name == "." || name == ""
		// UV's empty auxiliary target lock is discarded. It carries neither
		// selection data nor an installation/completion proof.
		auxiliaryLock := name == "site/.lock" && header.Size == 0 && header.Mode == 0o666 &&
			(header.Typeflag == tar.TypeReg || header.Typeflag == 0)
		if entries > maxEntries || header.Uid != 1000 || header.Gid != 1000 || header.Mode&^int64(0o777) != 0 || (header.Mode&0o022 != 0 && !auxiliaryLock) ||
			seen[name] || (!root && !validClosureDestination(name)) || header.Size < 0 {
			return uvNativeOutput{}, fmt.Errorf("uv native path, ownership, mode or count is invalid: entries=%d uid=%d gid=%d mode=%o duplicate=%t root=%t lock=%t auxiliary_lock=%t path_valid=%t size=%d",
				entries, header.Uid, header.Gid, header.Mode, seen[name], root, name == "pylock.toml", name == "site/.lock", root || validClosureDestination(name), header.Size)
		}
		seen[name] = true
		if header.Typeflag == tar.TypeDir {
			if header.Size != 0 || (!root && name != "site" && !strings.HasPrefix(name, "site/")) {
				return uvNativeOutput{}, errors.New("uv native directory is unsupported")
			}
			continue
		}
		if root || (header.Typeflag != tar.TypeReg && header.Typeflag != 0) ||
			(name != "pylock.toml" && !strings.HasPrefix(name, "site/")) || header.Size > policy.MaxGraphUncompressed()-expanded {
			return uvNativeOutput{}, errors.New("uv native file type, location or size is unsupported")
		}
		expanded += header.Size
		parts := strings.Split(name, "/")
		isControl := len(parts) == 3 && strings.HasSuffix(parts[1], ".dist-info") && (parts[2] == "METADATA" || parts[2] == "WHEEL")
		if name == "pylock.toml" || isControl {
			limit := int64(uvNativeLockLimit)
			if isControl {
				limit = policy.WheelLimits().MaxMetadata
				if header.Size > uvNativeLockLimit-metadataBytes {
					return uvNativeOutput{}, errors.New("uv native control aggregate exceeds bounds")
				}
				metadataBytes += header.Size
			}
			if header.Size == 0 || header.Size > limit {
				return uvNativeOutput{}, errors.New("uv native control size is invalid")
			}
			body, err := io.ReadAll(io.LimitReader(archive, header.Size))
			if err != nil || int64(len(body)) != header.Size {
				return uvNativeOutput{}, errors.New("uv native control is truncated")
			}
			if !isControl {
				result.lock = body
				continue
			}
			control := controls[parts[1]]
			if control == nil {
				if len(controls) >= policy.MaxGraphArtifacts() {
					return uvNativeOutput{}, errors.New("uv native distribution count exceeds bounds")
				}
				control = &artifactpypi.UVInstalledWheelControls{DistInfo: parts[1]}
				controls[parts[1]] = control
			}
			if parts[2] == "METADATA" {
				control.Metadata = body
			} else {
				control.Wheel = body
			}
		} else if _, err := io.CopyN(io.Discard, archive, header.Size); err != nil {
			return uvNativeOutput{}, errors.New("uv native installed file is truncated")
		}
	}
	// Consume the producer to completion; only bounded zero tar padding may
	// follow the archive. A second archive is not part of this snapshot.
	if err := drainUVTarPadding(bounded); err != nil {
		return uvNativeOutput{}, err
	}
	if bounded.N == 0 || len(result.lock) == 0 || len(controls) == 0 {
		return uvNativeOutput{}, errors.New("uv native output is missing or exceeds bounds")
	}
	for _, control := range controls {
		if len(control.Metadata) == 0 || len(control.Wheel) == 0 {
			return uvNativeOutput{}, errors.New("uv native wheel controls are incomplete")
		}
		result.controls = append(result.controls, *control)
	}
	sort.Slice(result.controls, func(i, j int) bool { return result.controls[i].DistInfo < result.controls[j].DistInfo })
	return result, nil
}

func drainUVTarPadding(input io.Reader) error {
	var padding [4096]byte
	for {
		n, err := input.Read(padding[:])
		for _, value := range padding[:n] {
			if value != 0 {
				return errors.New("uv native tar has trailing data")
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
