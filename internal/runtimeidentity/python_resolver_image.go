package runtimeidentity

// PythonResolverImage identifies the optional local Linux/amd64 image made by
// appending one precompiled bytecode layer to the pinned official Python base.
type PythonResolverImage struct {
	BaseConfig  string `json:"base_config"`
	LayerDiffID string `json:"layer_diff_id"`
	Config      string `json:"config"`
}

func PinnedPythonResolverImage() PythonResolverImage {
	return PythonResolverImage{BaseConfig: PythonResolverBaseConfig, LayerDiffID: PythonResolverLayerDiffID, Config: PythonResolverImageConfig}
}
