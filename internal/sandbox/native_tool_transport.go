package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type nativeRuntimeToolKind uint8

const (
	nativeRuntimeUV nativeRuntimeToolKind = iota
	nativeRuntimePNPM
	nativeRuntimeBun
	nativeRuntimePipx
)

type nativeRuntimeToolSpec struct {
	name, guestPath, sha256              string
	size                                 int64
	volumePrefix, volumeLabel, hashLabel string
	installedName, volumeOptions         string
	maxBytes                             int64
}

func (s nativeRuntimeToolSpec) fileMode() int64 {
	if s.installedName == runtimeidentity.PipxRuntimeAssetName {
		return 0444
	}
	return 0555
}

func nativeRuntimeToolIdentity(kind nativeRuntimeToolKind) (nativeRuntimeToolSpec, bool) {
	switch kind {
	case nativeRuntimeUV:
		return nativeRuntimeToolSpec{"uv", uvToolGuestPath, runtimeidentity.UVToolSHA256, runtimeidentity.UVToolSize, "haa-uv-runtime-", uvRuntimeVolumeLabel, uvRuntimeToolLabel, "", uvRuntimeVolumeOptions, 64 << 20}, true
	case nativeRuntimePNPM:
		return nativeRuntimeToolSpec{"pnpm", pnpmToolGuestPath, runtimeidentity.PNPMToolSHA256, runtimeidentity.PNPMToolSize, "haa-pnpm-runtime-", "io.heliopause.pnpm-runtime", "io.heliopause.pnpm-tool-sha256", runtimeidentity.PNPMNativeAssetName, uvRuntimeVolumeOptions, 64 << 20}, true
	case nativeRuntimeBun:
		return nativeRuntimeToolSpec{"bun", bunToolGuestPath, runtimeidentity.BunToolSHA256, runtimeidentity.BunToolSize, "haa-bun-runtime-", "io.heliopause.bun-runtime", "io.heliopause.bun-tool-sha256", runtimeidentity.BunNativeAssetName, "size=134217728,nosuid,nodev,uid=0,gid=0,mode=0755", 128 << 20}, true
	case nativeRuntimePipx:
		// DATA transport only. The existing single-writer native volume is
		// unavailable for this kind; readonly lifecycle is a separate owner.
		return nativeRuntimeToolSpec{name: "pipx-runtime.zip", guestPath: pipxToolGuestPath, sha256: runtimeidentity.PipxToolSHA256, size: runtimeidentity.PipxToolSize, installedName: runtimeidentity.PipxRuntimeAssetName, maxBytes: 8 << 20}, true
	default:
		return nativeRuntimeToolSpec{}, false
	}
}

func validPipxTransportIdentity(identity nativeToolFileIdentity, kind nativeRuntimeToolKind) bool {
	return kind != nativeRuntimePipx || identity == (nativeToolFileIdentity{runtimeidentity.PipxToolSize, runtimeidentity.PipxToolSHA256, 0444})
}

var errNativeToolTransport = errors.New("immutable native tool transport is unavailable or changed")

type nativeToolFileIdentity struct {
	size   int64
	sha256 string
	mode   int64
}

// Exposing only Write prevents io.Copy/ReaderFrom from bypassing the complete
// Docker tar transport bound, including extended headers and padding.
type nativeToolArchiveOutput struct {
	body   []byte
	limit  int64
	failed bool
}

func (w *nativeToolArchiveOutput) Write(body []byte) (int, error) {
	if w.failed || int64(len(body)) > w.limit-int64(len(w.body)) {
		w.failed = true
		return 0, errNativeToolTransport
	}
	w.body = append(w.body, body...)
	return len(body), nil
}

func readNativeToolFromContainer(ctx context.Context, runner streamingDockerOutput, container, path string, identity nativeToolFileIdentity, kind nativeRuntimeToolKind) ([]byte, error) {
	spec, known := nativeRuntimeToolIdentity(kind)
	validPath := path == spec.guestPath || kind == nativeRuntimeUV && path == runtimeidentity.UVToolPath
	if ctx == nil || runner == nil || !known || !validPath || !validPipxTransportIdentity(identity, kind) || !exactObservationContainerID(container) || identity.size <= 0 || identity.size > spec.maxBytes {
		return nil, errNativeToolTransport
	}
	output := nativeToolArchiveOutput{limit: identity.size + 64<<10}
	if err := runner.RunOutput(ctx, &output, "docker", "cp", container+":"+path, "-"); err != nil || output.failed || ctx.Err() != nil {
		return nil, errors.Join(errNativeToolTransport, err, ctx.Err())
	}
	return scanNativeToolTar(output.body, identity, kind)
}

func scanNativeToolTar(body []byte, identity nativeToolFileIdentity, kind nativeRuntimeToolKind) ([]byte, error) {
	spec, known := nativeRuntimeToolIdentity(kind)
	if !known || !validPipxTransportIdentity(identity, kind) || identity.size <= 0 || identity.size > spec.maxBytes || int64(len(body)) > identity.size+64<<10 {
		return nil, errNativeToolTransport
	}
	input := bytes.NewReader(body)
	archive := tar.NewReader(input)
	header, err := archive.Next()
	if err != nil || header.Name != spec.name && header.Name != "./"+spec.name || header.Typeflag != tar.TypeReg || header.Linkname != "" || header.Uid != 0 || header.Gid != 0 || header.Mode != identity.mode || header.Size != identity.size {
		return nil, errNativeToolTransport
	}
	native, err := io.ReadAll(archive)
	if err != nil || int64(len(native)) != identity.size {
		return nil, errNativeToolTransport
	}
	digest := sha256.Sum256(native)
	if hex.EncodeToString(digest[:]) != identity.sha256 {
		return nil, errNativeToolTransport
	}
	if _, err := archive.Next(); err != io.EOF {
		return nil, errNativeToolTransport
	}
	for _, value := range body[len(body)-input.Len():] {
		if value != 0 {
			return nil, errNativeToolTransport
		}
	}
	return native, nil
}

func introduceNativeTool(ctx context.Context, runner CommandRunner, container string, kind nativeRuntimeToolKind, body []byte) error {
	spec, known := nativeRuntimeToolIdentity(kind)
	input, acceptsInput := runner.(inputCommandRunner)
	output, streamsOutput := runner.(streamingDockerOutput)
	if ctx == nil || !known || !acceptsInput || !streamsOutput || !exactObservationContainerID(container) || int64(len(body)) != spec.size {
		return errNativeToolTransport
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(errNativeToolTransport, err)
	}
	digest := sha256.Sum256(body)
	if hex.EncodeToString(digest[:]) != spec.sha256 {
		return errNativeToolTransport
	}
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		archive := tar.NewWriter(writer)
		err := archive.WriteHeader(&tar.Header{Name: spec.name, Typeflag: tar.TypeReg, Mode: spec.fileMode(), Uid: 0, Gid: 0, Size: int64(len(body))})
		if err == nil {
			_, err = archive.Write(body)
		}
		err = errors.Join(err, archive.Close())
		_ = writer.CloseWithError(err)
		done <- err
	}()
	commandErr := input.RunInput(ctx, reader, "docker", "cp", "--archive", "-", container+":"+path.Dir(spec.guestPath))
	_ = reader.CloseWithError(commandErr)
	copyErr := <-done
	if err := errors.Join(commandErr, copyErr, ctx.Err()); err != nil {
		return errors.Join(errNativeToolTransport, err)
	}
	_, err := readNativeToolFromContainer(ctx, output, container, spec.guestPath, nativeToolFileIdentity{spec.size, spec.sha256, spec.fileMode()}, kind)
	return err
}
