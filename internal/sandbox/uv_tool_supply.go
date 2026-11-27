package sandbox

import (
	"bytes"
	"context"
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
type uvToolFileIdentity = nativeToolFileIdentity

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

// Preserve the existing UV entry points and upstream/guest identities while
// both native runtimes consume the same bounded transport implementation.
type uvToolArchiveOutput = nativeToolArchiveOutput

func readUVToolFromContainer(ctx context.Context, runner streamingDockerOutput, container, path string, identity uvToolFileIdentity) ([]byte, error) {
	body, err := readNativeToolFromContainer(ctx, runner, container, path, identity, nativeRuntimeUV)
	if err != nil {
		return nil, errors.Join(errUVToolSupply, err)
	}
	return body, nil
}

func scanUVToolTar(body []byte, identity uvToolFileIdentity) ([]byte, error) {
	native, err := scanNativeToolTar(body, identity, nativeRuntimeUV)
	if err != nil {
		return nil, errors.Join(errUVToolSupply, err)
	}
	return native, nil
}

func (p *uvPreparedTool) introduce(ctx context.Context, runner CommandRunner, container string) error {
	if p == nil {
		return errUVToolSupply
	}
	if err := introduceNativeTool(ctx, runner, container, nativeRuntimeUV, p.body); err != nil {
		return errors.Join(errUVToolSupply, err)
	}
	return nil
}
