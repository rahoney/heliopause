package sandbox

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"strings"
)

type readonlyInputFile struct {
	size   int64
	digest string
}

func copyReadonlyInputTar(ctx context.Context, runner CommandRunner, container, root string, write func(io.Writer) error, failure error) error {
	if failure == nil {
		return errors.New("readonly input copy failure contract is unavailable")
	}
	input, ok := runner.(inputCommandRunner)
	if ctx == nil || !ok || !exactObservationContainerID(container) || write == nil || (root != pipxNativeWheels && root != sourceAnalysisRoot) {
		return failure
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		err := write(writer)
		_ = writer.CloseWithError(err)
		done <- err
	}()
	commandErr := input.RunInput(ctx, reader, "docker", "cp", "--archive", "-", container+":"+root)
	_ = reader.CloseWithError(commandErr)
	return errors.Join(commandErr, <-done, ctx.Err())
}

func readReadonlyInputTar(ctx context.Context, runner CommandRunner, container, root string, scan func(io.Reader) error, failure error) error {
	if failure == nil {
		return errors.New("readonly input readback failure contract is unavailable")
	}
	output, ok := runner.(streamingDockerOutput)
	if ctx == nil || !ok || !exactObservationContainerID(container) || scan == nil || (root != pipxNativeWheels && root != sourceAnalysisRoot) {
		return failure
	}
	if err := verifyObservationRuntimeAlive(ctx, runner, container); err != nil {
		return err
	}
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		err := output.RunOutput(ctx, writer, "docker", "cp", container+":"+root+"/.", "-")
		_ = writer.CloseWithError(err)
		done <- err
	}()
	err := scan(reader)
	_ = reader.CloseWithError(err)
	return errors.Join(err, <-done, ctx.Err())
}

// This is the existing whole-wheel readback contract, shared with source DATA.
// Each caller separately owns its bounded expected set and admission lifecycle.
func scanReadonlyInputTar(input io.Reader, expected map[string]readonlyInputFile, failure error) error {
	if input == nil || len(expected) == 0 || failure == nil {
		return errors.New("readonly input tar contract is unavailable")
	}
	var total int64
	overhead := int64(len(expected)+1)*2048 + 10240
	for _, file := range expected {
		if file.size < 0 || file.size > math.MaxInt64-overhead-total {
			return failure
		}
		total += file.size
	}
	bounded := &io.LimitedReader{R: input, N: total + overhead}
	archive := tar.NewReader(bounded)
	seen := make(map[string]bool, len(expected))
	root := false
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return failure
		}
		name := strings.TrimPrefix(header.Name, "./")
		if header.Typeflag == tar.TypeDir && (name == "" || name == ".") && !root && header.Uid == 0 && header.Gid == 0 && header.Mode == 0755 && header.Size == 0 {
			root = true
			continue
		}
		file, known := expected[name]
		if !known || seen[name] || header.Typeflag != tar.TypeReg || header.Linkname != "" || header.Uid != 0 || header.Gid != 0 || header.Mode != 0444 || header.Size != file.size {
			return failure
		}
		digest := sha256.New()
		if count, err := io.CopyN(digest, archive, header.Size); err != nil || count != header.Size || hex.EncodeToString(digest.Sum(nil)) != file.digest {
			return failure
		}
		seen[name] = true
	}
	if len(seen) != len(expected) || bounded.N == 0 {
		return failure
	}
	return drainUVTarPadding(bounded)
}
