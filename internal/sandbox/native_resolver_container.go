package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const pnpmResolverOwnerLabel = "io.heliopause.pnpm-resolver"

type nativeResolverOwner struct{ name, prefix, label, image string }

func nativeResolverOwnership(kind nativeRuntimeToolKind) (nativeResolverOwner, bool) {
	switch kind {
	case nativeRuntimeUV:
		return nativeResolverOwner{"uv", "haa-uv-resolver-", uvResolverOwnerLabel, pythonImageReference}, true
	case nativeRuntimePNPM:
		return nativeResolverOwner{"pnpm", "haa-pnpm-resolver-", pnpmResolverOwnerLabel, nodeImageReference}, true
	default:
		return nativeResolverOwner{}, false
	}
}

// Only the two compiled native resolver kinds share this owner protocol.
// Unknown CREATE outcomes require a fresh successful inventory and attestation.
func removeOwnedNativeResolverContainer(ctx context.Context, runner CommandRunner, name, owner, expectedID string, kind nativeRuntimeToolKind) error {
	spec, ok := nativeResolverOwnership(kind)
	if !ok {
		return errors.New("native resolver kind unavailable")
	}
	if name != spec.prefix+owner || owner == "" {
		return fmt.Errorf("%s resolver cleanup owner unavailable", spec.name)
	}
	listed, err := runner.Output(ctx, "docker", "ps", "--all", "--quiet", "--no-trunc", "--filter", "name=^/"+name+"$")
	if err != nil || len(listed) > 1024 {
		return fmt.Errorf("%s resolver cleanup inventory unavailable", spec.name)
	}
	ids := strings.Fields(string(listed))
	if len(ids) == 0 && expectedID == "" {
		return nil
	}
	if len(ids) != 1 || !exactObservationContainerID(ids[0]) || expectedID != "" && ids[0] != expectedID {
		return fmt.Errorf("%s resolver cleanup identity differs", spec.name)
	}
	if err := verifyOwnedNativeResolverContainer(ctx, runner, name, owner, ids[0], kind); err != nil {
		return err
	}
	_, err = runner.Output(ctx, "docker", "rm", "--force", ids[0])
	return err
}

func verifyOwnedNativeResolverContainer(ctx context.Context, runner CommandRunner, name, owner, id string, kind nativeRuntimeToolKind) error {
	spec, ok := nativeResolverOwnership(kind)
	if !ok {
		return errors.New("native resolver kind unavailable")
	}
	if owner == "" || name != spec.prefix+owner || !exactObservationContainerID(id) {
		return fmt.Errorf("%s resolver container identity unavailable", spec.name)
	}
	body, err := runner.Output(ctx, "docker", "inspect", "--format", "{{json .}}", id)
	var actual struct {
		ID     string `json:"Id"`
		Name   string
		Config struct {
			Image  string
			Labels map[string]string
		}
	}
	if err != nil || len(body) == 0 || len(body) > 32<<10 || json.Unmarshal(body, &actual) != nil || actual.ID != id || actual.Name != "/"+name || actual.Config.Image != spec.image || actual.Config.Labels[spec.label] != owner {
		return fmt.Errorf("%s resolver container ownership unproven", spec.name)
	}
	return nil
}
