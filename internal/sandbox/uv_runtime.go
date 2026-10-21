package sandbox

const uvResolverProfile = "uv-pip-resolver"

func uvCreateArguments(network string, hostArguments []string, volume uvRuntimeVolume) ([]string, error) {
	mount, err := volume.mountArgument()
	if err != nil {
		return nil, err
	}
	return pythonResolverCreateArguments(network, hostArguments, []string{"--mount", mount}), nil
}
