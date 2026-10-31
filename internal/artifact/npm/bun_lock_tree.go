package npm

import "strings"

func bindBunTree(selection *BunLock, packages map[string]BunLockedPackage) error {
	versions := map[string]map[string]bool{}
	for key, pkg := range packages {
		parts, _ := bunKeyParts(key)
		if len(parts) > 1 {
			if _, exists := packages[strings.Join(parts[:len(parts)-1], "/")]; !exists {
				return errBunLock
			}
		}
		if versions[pkg.Name] == nil {
			versions[pkg.Name] = map[string]bool{}
		}
		versions[pkg.Name][pkg.Version] = true
	}
	edges, peerComparisons := len(selection.roots), 0
	type peerRequirement struct{ name, spec string }
	peerCounts := map[peerRequirement]int{}
	for key, pkg := range packages {
		pkg.Targets = map[string]string{}
		for _, requirements := range []map[string]string{pkg.Dependencies, pkg.PeerDependencies} {
			for name, spec := range requirements {
				edges++
				if edges > bunLockEdgeLimit {
					return errBunLock
				}
				target, exists := resolveBunTreeDependency(key, name, packages)
				if !exists || target.Key == key || !bunConstraintMatches(spec, target.Version) {
					return errBunLock
				}
				pkg.Targets[name] = target.Key
			}
		}
		for name, spec := range pkg.PeerDependencies {
			requirement := peerRequirement{name, spec}
			count, cached := peerCounts[requirement]
			if !cached {
				for version := range versions[name] {
					peerComparisons++
					if peerComparisons > bunLockEdgeLimit {
						return errBunLock
					}
					if bunConstraintMatches(spec, version) {
						count++
					}
				}
				peerCounts[requirement] = count
			}
			// The upstream deferred-peer pass can select by version. Reject
			// multiple matching versions instead of guessing the native edge.
			if count != 1 {
				return errBunLock
			}
		}
		packages[key] = pkg
	}
	seen := map[string]bool{}
	queue := []string{}
	for _, root := range selection.roots {
		pkg, exists := packages[root.Instance]
		if !exists || pkg.Name != root.Name || !bunConstraintMatches(root.Specifier, pkg.Version) {
			return errBunLock
		}
		if !seen[pkg.Key] {
			seen[pkg.Key] = true
			queue = append(queue, pkg.Key)
		}
	}
	for i := 0; i < len(queue); i++ {
		for _, target := range packages[queue[i]].Targets {
			if !seen[target] {
				seen[target] = true
				queue = append(queue, target)
			}
		}
	}
	if len(seen) != len(packages) {
		return errBunLock
	}
	for _, pkg := range packages {
		selection.packages = append(selection.packages, pkg)
	}
	return nil
}

func resolveBunTreeDependency(from, name string, packages map[string]BunLockedPackage) (BunLockedPackage, bool) {
	parts, ok := bunKeyParts(from)
	if !ok || validatePackageName(name) != nil {
		return BunLockedPackage{}, false
	}
	for depth := len(parts); depth >= 0; depth-- {
		key := name
		if depth != 0 {
			key = strings.Join(parts[:depth], "/") + "/" + name
		}
		if pkg, exists := packages[key]; exists {
			return pkg, true
		}
	}
	return BunLockedPackage{}, false
}
