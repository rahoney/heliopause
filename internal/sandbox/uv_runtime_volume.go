package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const uvRuntimeVolumeLabel = "io.heliopause.uv-runtime"
const uvRuntimeToolLabel = "io.heliopause.uv-tool-sha256"
const uvRuntimeVolumeOptions = "size=67108864,nosuid,nodev,uid=0,gid=0,mode=0755"

// This volume holds only the fixed helper and authenticated native UV tool.
// Docker knows its tmpfs mount, allowing data copy into the running gVisor
// target. Artifact-owned Python closure volumes retain their separate owner.
type uvRuntimeVolume struct {
	name, owner, createdAt, mountpoint string
}

var errUVRuntimeVolume = errors.New("uv private runtime volume is unavailable or changed")

func createUVRuntimeVolume(ctx context.Context, runner CommandRunner) (uvRuntimeVolume, error) {
	if ctx == nil || runner == nil {
		return uvRuntimeVolume{}, errUVRuntimeVolume
	}
	owner, err := domain.NewSandboxSessionID()
	if err != nil {
		return uvRuntimeVolume{}, errUVRuntimeVolume
	}
	v := uvRuntimeVolume{name: "haa-uv-runtime-" + owner.String(), owner: owner.String()}
	listed, err := runner.Output(ctx, "docker", "volume", "ls", "--quiet", "--filter", "name=^"+v.name+"$")
	if err != nil || len(listed) > 1024 || strings.TrimSpace(string(listed)) != "" {
		return uvRuntimeVolume{}, errUVRuntimeVolume
	}
	created, createErr := runner.Output(ctx, "docker", "volume", "create", "--driver", "local",
		"--opt", "type=tmpfs", "--opt", "device=tmpfs", "--opt", "o="+uvRuntimeVolumeOptions,
		"--label", uvRuntimeVolumeLabel+"="+v.owner, "--label", uvRuntimeToolLabel+"="+runtimeidentity.UVToolSHA256, v.name)
	verifyErr := v.verify(ctx, runner)
	if createErr != nil || strings.TrimSpace(string(created)) != v.name || verifyErr != nil {
		// A failed CREATE may have succeeded remotely. Remove only after the
		// exact ownership/options can be attested; unknown identity is retained.
		cleanup, done := resolverCleanupContext()
		defer done()
		cleanupErr := v.close(cleanup, runner)
		return uvRuntimeVolume{}, fmt.Errorf("uv runtime creation failed name=%s: %w", v.name, errors.Join(errUVRuntimeVolume, createErr, verifyErr, cleanupErr))
	}
	return v, nil
}

func (v *uvRuntimeVolume) verify(ctx context.Context, runner CommandRunner) error {
	if v == nil || ctx == nil || runner == nil || v.owner == "" || v.name != "haa-uv-runtime-"+v.owner {
		return errUVRuntimeVolume
	}
	body, err := runner.Output(ctx, "docker", "volume", "inspect", "--format", "{{json .}}", v.name)
	var actual dockerVolumeInspection
	if err != nil || len(body) == 0 || len(body) > 16<<10 || json.Unmarshal(body, &actual) != nil ||
		actual.Name != v.name || actual.Driver != "local" || actual.Scope != "local" ||
		actual.Mountpoint == "" || actual.CreatedAt == "" || len(actual.Options) != 3 ||
		actual.Options["type"] != "tmpfs" || actual.Options["device"] != "tmpfs" || actual.Options["o"] != uvRuntimeVolumeOptions ||
		actual.Labels[uvRuntimeVolumeLabel] != v.owner || actual.Labels[uvRuntimeToolLabel] != runtimeidentity.UVToolSHA256 ||
		v.createdAt != "" && (v.createdAt != actual.CreatedAt || v.mountpoint != actual.Mountpoint) {
		return errUVRuntimeVolume
	}
	v.createdAt, v.mountpoint = actual.CreatedAt, actual.Mountpoint
	return nil
}

func (v uvRuntimeVolume) mountArgument() (string, error) {
	if v.owner == "" || v.name != "haa-uv-runtime-"+v.owner || v.createdAt == "" || v.mountpoint == "" {
		return "", errUVRuntimeVolume
	}
	return "type=volume,source=" + v.name + ",target=/haa-runtime,volume-nocopy", nil
}

func (v *uvRuntimeVolume) verifyTarget(ctx context.Context, runner CommandRunner, id string) error {
	if !exactObservationContainerID(id) || v.verify(ctx, runner) != nil {
		return errUVRuntimeVolume
	}
	// An exact successful inventory is required. A competing attachment cannot
	// become a second writer to this root-owned tool mount.
	listed, err := runner.Output(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "volume="+v.name, "--format", "{{.ID}}")
	if err != nil || strings.TrimSpace(string(listed)) != id {
		return errUVRuntimeVolume
	}
	body, err := runner.Output(ctx, "docker", "inspect", "--format", containerMountInspectTemplate, id)
	var actual dockerContainerMountInspection
	if err != nil || len(body) == 0 || len(body) > 16<<10 || json.Unmarshal(body, &actual) != nil || actual.ID != id || len(actual.Binds) != 0 {
		return errUVRuntimeVolume
	}
	matched, configured := 0, 0
	for _, m := range actual.Mounts {
		if m.Type == "bind" || m.Destination == "/" || strings.HasPrefix(m.Destination, "/haa-runtime/") {
			return errUVRuntimeVolume
		}
		if m.Destination != "/haa-runtime" {
			continue
		}
		matched++
		if m.Type != "volume" || m.Name != v.name || m.Source != v.mountpoint || m.Driver != "local" || !m.RW {
			return errUVRuntimeVolume
		}
	}
	for _, m := range actual.HostMounts {
		if m.Type == "bind" || m.Target == "/" || strings.HasPrefix(m.Target, "/haa-runtime/") {
			return errUVRuntimeVolume
		}
		if m.Target != "/haa-runtime" {
			continue
		}
		configured++
		if m.Type != "volume" || m.Source != v.name || m.ReadOnly || !m.VolumeOptions.NoCopy {
			return errUVRuntimeVolume
		}
	}
	if matched != 1 || configured != 1 {
		return errUVRuntimeVolume
	}
	return nil
}

func (v *uvRuntimeVolume) close(ctx context.Context, runner CommandRunner) error {
	if v.verify(ctx, runner) != nil {
		return errUVRuntimeVolume
	}
	listed, err := runner.Output(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "volume="+v.name, "--format", "{{.ID}}")
	if err != nil || len(listed) > 1024 || strings.TrimSpace(string(listed)) != "" {
		return errUVRuntimeVolume
	}
	_, err = runner.Output(ctx, "docker", "volume", "rm", v.name)
	return err
}
