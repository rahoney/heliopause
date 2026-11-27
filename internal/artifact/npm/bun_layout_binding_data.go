package npm

import "encoding/json"

func bunInspectionManifest(lock BunLock) ([]byte, error) {
	packages := map[string]BunLockedPackage{}
	for _, pkg := range lock.packages {
		packages[pkg.Key] = pkg
	}
	manifest := map[string]any{"private": true}
	groups := map[string]map[string]string{}
	for _, root := range lock.roots {
		pkg, present := packages[root.Instance]
		if !present || pkg.Name != root.Name || root.Group != "dependencies" && root.Group != "devDependencies" {
			return nil, errBunInspectionLayout
		}
		if groups[root.Group] == nil {
			groups[root.Group] = map[string]string{}
		}
		if previous, present := groups[root.Group][root.Name]; present && previous != pkg.Version {
			return nil, errBunInspectionLayout
		}
		groups[root.Group][root.Name] = pkg.Version
	}
	for group, dependencies := range groups {
		manifest[group] = dependencies
	}
	return json.Marshal(manifest)
}
