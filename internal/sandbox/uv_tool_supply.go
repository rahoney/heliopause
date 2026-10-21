package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const uvToolGuestPath = "/haa-runtime/uv"
const uvSupplyOwnerLabel = "io.heliopause.uv-supply"

var errUVToolSupply = errors.New("uv immutable tool supply is unavailable or changed")

type uvPreparedTool struct{ body []byte }
type uvToolFileIdentity struct {
	size   int64
	sha256 string
	mode   int64
}

// prepareUVTool reads a never-started upstream container. It cannot execute UV
// on the Host or grant a target container observation/admission authority.
func prepareUVTool(ctx context.Context, runner CommandRunner) (prepared *uvPreparedTool, resultErr error) {
	if ctx == nil || runner == nil {
		return nil, errUVToolSupply
	}
	output, ok := runner.(streamingDockerOutput)
	if !ok {
		return nil, errUVToolSupply
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	image, err := runner.Output(ctx, "docker", "image", "inspect", runtimeidentity.UVImageReference, "--format", "{{.Id}} {{.Architecture}}")
	if err != nil || strings.TrimSpace(string(image)) != "sha256:"+runtimeidentity.UVConfigSHA256+" "+runtimeidentity.UVArchitecture {
		return nil, errUVToolSupply
	}
	session, err := domain.NewSandboxSessionID()
	if err != nil {
		return nil, errUVToolSupply
	}
	owner := session.String()
	name := "haa-uv-supply-" + owner
	var createdID string
	defer func() {
		cleanupCtx, cleanupCancel := resolverCleanupContext()
		defer cleanupCancel()
		id, err := inspectOwnedUVSupply(cleanupCtx, runner, name, owner, createdID)
		if err == nil {
			_, err = runner.Output(cleanupCtx, "docker", "rm", id)
		}
		if err != nil {
			prepared = nil
			resultErr = errors.Join(resultErr, fmt.Errorf("uv supply cleanup unproven container=%s: %w", name, err))
		}
	}()
	created, err := runner.Output(ctx, "docker", "create", "--pull", "never", "--name", name, "--label", uvSupplyOwnerLabel+"="+owner, "--network", "none", "--read-only", "--user", "0:0", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", runtimeidentity.UVImageReference)
	if err != nil || !exactObservationContainerID(strings.TrimSpace(string(created))) {
		return nil, errUVToolSupply
	}
	createdID = strings.TrimSpace(string(created))
	if _, err := inspectOwnedUVSupply(ctx, runner, name, owner, createdID); err != nil {
		return nil, err
	}
	body, err := readUVToolFromContainer(ctx, output, createdID, runtimeidentity.UVToolPath, uvToolFileIdentity{runtimeidentity.UVToolSize, runtimeidentity.UVToolSHA256, 0755})
	if err != nil {
		return nil, err
	}
	return &uvPreparedTool{body: body}, nil
}

func inspectOwnedUVSupply(ctx context.Context, runner CommandRunner, name, owner, expectedID string) (string, error) {
	body, err := runner.Output(ctx, "docker", "container", "inspect", name, "--format", "{{json .}}")
	if err != nil || len(body) > 64<<10 {
		return "", errUVToolSupply
	}
	var value struct {
		ID          string `json:"Id"`
		Name, Image string
		Config      struct {
			Image  string
			Labels map[string]string
		}
		State struct {
			Status, StartedAt string
			Running           bool
			PID               int `json:"Pid"`
		}
	}
	d := json.NewDecoder(bytes.NewReader(body))
	if d.Decode(&value) != nil || d.Decode(new(any)) != io.EOF || !exactObservationContainerID(value.ID) || expectedID != "" && value.ID != expectedID || value.Name != "/"+name || value.Image != "sha256:"+runtimeidentity.UVConfigSHA256 || value.Config.Image != runtimeidentity.UVImageReference || value.Config.Labels[uvSupplyOwnerLabel] != owner || value.State.Status != "created" || value.State.Running || value.State.PID != 0 || value.State.StartedAt != "0001-01-01T00:00:00Z" {
		return "", errUVToolSupply
	}
	return value.ID, nil
}

// Only Write is exposed. io.Copy/StringWriter promotion cannot bypass the
// complete tar transport bound, including extended headers and padding.
type uvToolArchiveOutput struct {
	body   []byte
	limit  int64
	failed bool
}

func (w *uvToolArchiveOutput) Write(body []byte) (int, error) {
	if w.failed || int64(len(body)) > w.limit-int64(len(w.body)) {
		w.failed = true
		return 0, errUVToolSupply
	}
	w.body = append(w.body, body...)
	return len(body), nil
}

func readUVToolFromContainer(ctx context.Context, runner streamingDockerOutput, container, path string, identity uvToolFileIdentity) ([]byte, error) {
	if ctx == nil || runner == nil || !exactObservationContainerID(container) || path != runtimeidentity.UVToolPath && path != uvToolGuestPath || identity.size <= 0 || identity.size > 64<<20 {
		return nil, errUVToolSupply
	}
	output := uvToolArchiveOutput{limit: identity.size + 64<<10}
	if err := runner.RunOutput(ctx, &output, "docker", "cp", container+":"+path, "-"); err != nil || output.failed || ctx.Err() != nil {
		return nil, errors.Join(errUVToolSupply, err, ctx.Err())
	}
	return scanUVToolTar(output.body, identity)
}

func scanUVToolTar(body []byte, identity uvToolFileIdentity) ([]byte, error) {
	if identity.size <= 0 || identity.size > 64<<20 || int64(len(body)) > identity.size+64<<10 {
		return nil, errUVToolSupply
	}
	input := bytes.NewReader(body)
	archive := tar.NewReader(input)
	header, err := archive.Next()
	if err != nil || header.Name != "uv" && header.Name != "./uv" || header.Typeflag != tar.TypeReg || header.Linkname != "" || header.Uid != 0 || header.Gid != 0 || header.Mode != identity.mode || header.Size != identity.size {
		return nil, errUVToolSupply
	}
	native, err := io.ReadAll(archive)
	if err != nil || int64(len(native)) != identity.size {
		return nil, errUVToolSupply
	}
	digest := sha256.Sum256(native)
	if hex.EncodeToString(digest[:]) != identity.sha256 {
		return nil, errUVToolSupply
	}
	if _, err := archive.Next(); err != io.EOF {
		return nil, errUVToolSupply
	}
	// tar EOF cannot hide a concatenated archive or nonzero trailing data.
	for _, value := range body[len(body)-input.Len():] {
		if value != 0 {
			return nil, errUVToolSupply
		}
	}
	return native, nil
}

func (p *uvPreparedTool) introduce(ctx context.Context, runner CommandRunner, container string) error {
	input, ok := runner.(inputCommandRunner)
	output, streaming := runner.(streamingDockerOutput)
	if p == nil || ctx == nil || !ok || !streaming || !exactObservationContainerID(container) || int64(len(p.body)) != runtimeidentity.UVToolSize {
		return errUVToolSupply
	}
	digest := sha256.Sum256(p.body)
	if hex.EncodeToString(digest[:]) != runtimeidentity.UVToolSHA256 {
		return errUVToolSupply
	}
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		archive := tar.NewWriter(writer)
		err := archive.WriteHeader(&tar.Header{Name: "uv", Typeflag: tar.TypeReg, Mode: 0555, Uid: 0, Gid: 0, Size: int64(len(p.body))})
		if err == nil {
			_, err = archive.Write(p.body)
		}
		err = errors.Join(err, archive.Close())
		_ = writer.CloseWithError(err)
		done <- err
	}()
	commandErr := input.RunInput(ctx, reader, "docker", "cp", "--archive", "-", container+":/haa-runtime")
	_ = reader.CloseWithError(commandErr)
	copyErr := <-done
	if err := errors.Join(commandErr, copyErr, ctx.Err()); err != nil {
		return errors.Join(errUVToolSupply, err)
	}
	_, err := readUVToolFromContainer(ctx, output, container, uvToolGuestPath, uvToolFileIdentity{runtimeidentity.UVToolSize, runtimeidentity.UVToolSHA256, 0555})
	return err
}
