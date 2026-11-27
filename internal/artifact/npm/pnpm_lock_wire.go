package npm

type pnpmLockWire struct {
	Version   string                      `yaml:"lockfileVersion"`
	Settings  *pnpmLockSettings           `yaml:"settings"`
	Importers map[string]pnpmImporterWire `yaml:"importers"`
	Packages  map[string]pnpmPackageWire  `yaml:"packages"`
	Snapshots map[string]pnpmSnapshotWire `yaml:"snapshots"`
}

type pnpmLockSettings struct {
	AutoInstallPeers *bool  `yaml:"autoInstallPeers"`
	ExcludeLinks     *bool  `yaml:"excludeLinksFromLockfile"`
	IntegrityTarget  string `yaml:"integrityTarget"`
}

type pnpmImporterWire struct {
	Dependencies map[string]pnpmRootWire `yaml:"dependencies,omitempty"`
	Dev          map[string]pnpmRootWire `yaml:"devDependencies,omitempty"`
	Optional     map[string]pnpmRootWire `yaml:"optionalDependencies,omitempty"`
}

type pnpmRootWire struct {
	Specifier string `yaml:"specifier"`
	Version   string `yaml:"version"`
}

type pnpmPackageWire struct {
	Resolution struct {
		Integrity string `yaml:"integrity"`
		Tarball   string `yaml:"tarball,omitempty"`
	} `yaml:"resolution"`
	Engines    map[string]string `yaml:"engines,omitempty"`
	Peers      map[string]string `yaml:"peerDependencies,omitempty"`
	Deprecated string            `yaml:"deprecated,omitempty"`
	HasBin     bool              `yaml:"hasBin,omitempty"`
	Prepare    bool              `yaml:"prepare,omitempty"`
}

type pnpmSnapshotWire struct {
	Dependencies    map[string]string `yaml:"dependencies,omitempty"`
	Optional        map[string]string `yaml:"optionalDependencies,omitempty"`
	TransitivePeers []string          `yaml:"transitivePeerDependencies,omitempty"`
	ID              string            `yaml:"id,omitempty"`
	Patched         bool              `yaml:"patched,omitempty"`
	IsOptional      bool              `yaml:"optional,omitempty"`
}
