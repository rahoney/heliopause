package sandbox

import "errors"

const bunResolverProfile = "bun-resolver"

// Query startup uses the pinned Node image only as its immutable GNU ABI.
// It has no registry network, project mount, or Host Bun execution.
func bunQueryCreateArguments(volume nativeRuntimeVolume) ([]string, error) {
	if volume.kind != nativeRuntimeBun {
		return nil, errors.New("bun requires its own attested runtime volume")
	}
	mount, err := volume.mountArgument()
	if err != nil {
		return nil, err
	}
	return nodeResolverCreateArguments("none", nil, []string{"--mount", mount}), nil
}
