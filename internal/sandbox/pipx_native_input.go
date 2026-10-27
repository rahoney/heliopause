package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path"
	"sort"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errPipxNativeInput = errors.New("pipx frozen whole-wheel input is unavailable or changed")

type pipxNativeInputFile struct {
	name, url, digest string
	body              []byte
}

// These owned bytes have passed whole original/source/graph and startup checks.
// They confer neither execution admission nor Policy or Host ownership.
type pipxNativeInput struct {
	files []pipxNativeInputFile
	graph domain.LockedDependencyGraph
	id    string
}

func freezePipxNativeInput(ctx context.Context, lock artifactpypi.UVLock, inputs []artifactpypi.UVWheelInput, pages []artifactpypi.SimpleProject) (*pipxNativeInput, error) {
	policy := artifactpypi.PublicPyPIProfile().ResourcePolicy()
	if ctx == nil || len(inputs) == 0 || len(inputs) > policy.MaxGraphArtifacts() || len(inputs) != len(lock.Packages()) {
		return nil, errPipxNativeInput
	}
	known := map[string]artifactpypi.UVLockedWheel{}
	for _, p := range lock.Packages() {
		for _, w := range p.Wheels {
			if _, exists := known[w.URL]; exists {
				return nil, errPipxNativeInput
			}
			known[w.URL] = w
		}
	}
	result := &pipxNativeInput{}
	seenURL, seenName := map[string]bool{}, map[string]bool{}
	var total int64
	for _, input := range inputs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		wheel, ok := known[input.URL]
		if !ok || seenURL[input.URL] || seenName[wheel.Filename] || input.Reader == nil || input.Size <= 0 || input.Size != wheel.Size || input.Size > policy.MaxGraphCompressed()-total || path.Base(wheel.Filename) != wheel.Filename {
			return nil, errPipxNativeInput
		}
		total += input.Size
		body := make([]byte, int(input.Size))
		if _, err := io.ReadFull(io.NewSectionReader(input.Reader, 0, input.Size), body); err != nil {
			return nil, errors.Join(errPipxNativeInput, err)
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != wheel.SHA256 {
			return nil, errPipxNativeInput
		}
		result.files = append(result.files, pipxNativeInputFile{name: wheel.Filename, url: input.URL, digest: wheel.SHA256, body: body})
		seenURL[input.URL], seenName[wheel.Filename] = true, true
	}
	sort.Slice(result.files, func(i, j int) bool { return result.files[i].name < result.files[j].name })
	frozen := make([]artifactpypi.UVWheelInput, 0, len(result.files))
	for _, file := range result.files {
		frozen = append(frozen, artifactpypi.UVWheelInput{URL: file.url, Reader: bytes.NewReader(file.body), Size: int64(len(file.body))})
	}
	runtime := PinnedPythonRuntime()
	graph, err := lock.BindPipxQueryWheels(frozen, pages, artifactpypi.WheelTarget{Python: runtime.InterpreterTag, ABI: runtime.ABITag, Platform: runtime.PlatformTag}, policy.WheelLimits())
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result.graph = graph
	result.id, err = result.manifestID()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *pipxNativeInput) manifestID() (string, error) {
	if p == nil || len(p.files) == 0 || len(p.files) > artifactpypi.PublicPyPIProfile().ResourcePolicy().MaxGraphArtifacts() {
		return "", errPipxNativeInput
	}
	type fact struct {
		Name, URL, SHA256 string
		Size              int64
	}
	facts := make([]fact, 0, len(p.files))
	var total int64
	previous := ""
	for _, file := range p.files {
		if file.name == "" || path.Base(file.name) != file.name || file.name <= previous || file.url == "" || len(file.body) == 0 || int64(len(file.body)) > artifactpypi.PublicPyPIProfile().ResourcePolicy().MaxGraphCompressed()-total {
			return "", errPipxNativeInput
		}
		if _, _, _, _, _, err := artifactpypi.ParseWheelFilename(file.name); err != nil {
			return "", errPipxNativeInput
		}
		sum := sha256.Sum256(file.body)
		if hex.EncodeToString(sum[:]) != file.digest {
			return "", errPipxNativeInput
		}
		total += int64(len(file.body))
		previous = file.name
		facts = append(facts, fact{file.name, file.url, file.digest, int64(len(file.body))})
	}
	body, err := json.Marshal(facts)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func (p *pipxNativeInput) valid() bool {
	id, err := p.manifestID()
	return err == nil && id == p.id && len(p.graph.Nodes()) == len(p.files)
}

func (p *pipxNativeInput) introduce(ctx context.Context, runner CommandRunner, container string) error {
	input, ok := runner.(inputCommandRunner)
	if ctx == nil || !ok || !exactObservationContainerID(container) || !p.valid() {
		return errPipxNativeInput
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		archive := tar.NewWriter(writer)
		var err error
		for _, file := range p.files {
			err = archive.WriteHeader(&tar.Header{Name: file.name, Typeflag: tar.TypeReg, Mode: 0444, Uid: 0, Gid: 0, Size: int64(len(file.body))})
			if err == nil {
				_, err = archive.Write(file.body)
			}
			if err != nil {
				break
			}
		}
		err = errors.Join(err, archive.Close())
		_ = writer.CloseWithError(err)
		done <- err
	}()
	commandErr := input.RunInput(ctx, reader, "docker", "cp", "--archive", "-", container+":"+pipxNativeWheels)
	_ = reader.CloseWithError(commandErr)
	return errors.Join(commandErr, <-done, ctx.Err())
}
