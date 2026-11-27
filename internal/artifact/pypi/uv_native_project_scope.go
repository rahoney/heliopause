package pypi

import (
	"errors"
	"fmt"
	"strings"
)

func uvNativeWireHasScopes(wire uvNativeProjectLockWire) bool {
	if len(wire.ResolutionMarkers) > 0 {
		return true
	}
	for _, item := range wire.Packages {
		if len(item.Optional)+len(item.Groups)+len(item.DefaultGroups)+len(item.ResolutionMarkers) > 0 {
			return true
		}
		for _, edge := range item.Dependencies {
			if edge.Marker != "" || edge.Version != "" || edge.Source != nil || len(edge.Extra) > 0 {
				return true
			}
		}
		if item.Metadata != nil {
			if len(item.Metadata.ProvidesExtras)+len(item.Metadata.RequiresDev) > 0 {
				return true
			}
			for _, r := range item.Metadata.RequiresDist {
				if r.Marker != "" || len(r.Extras) > 0 {
					return true
				}
			}
		}
	}
	return false
}

// Validate the universal graph's identities and public references. A later
// target projection and authenticated metadata validation remain mandatory.
func validateUVNativeVariantGraph(root string, variants map[string]uvNativePackage) error {
	invalid := errors.New("UV universal dependency graph is invalid")
	byName := map[string][]string{}
	for key, item := range variants {
		byName[item.Name] = append(byName[item.Name], key)
	}
	edges := map[string][]string{}
	count := 0
	for key, item := range variants {
		all := append([]uvNativeDependency(nil), item.Dependencies...)
		for _, groups := range []map[string][]uvNativeDependency{item.Optional, item.Groups} {
			seen := map[string]bool{}
			for name, members := range groups {
				normalized, err := NormalizeProjectName(name)
				if err != nil || name != normalized || seen[normalized] {
					return invalid
				}
				seen[normalized] = true
				all = append(all, members...)
			}
		}
		for _, name := range item.DefaultGroups {
			if _, exists := item.Groups[name]; !exists {
				if item.Metadata == nil {
					return fmt.Errorf("UV native default group is absent from locked scope: %s", name)
				}
				if requirements, present := item.Metadata.RequiresDev[name]; !present || len(requirements) != 0 {
					return fmt.Errorf("UV native default group is absent from locked scope: %s", name)
				}
			}
		}
		for _, marker := range item.ResolutionMarkers {
			if !validUVProjectText([]byte(marker), false) {
				return invalid
			}
		}
		if item.Metadata != nil {
			for _, r := range item.Metadata.RequiresDist {
				if !validUVNativeRequirement(r) {
					return invalid
				}
			}
			for name, rs := range item.Metadata.RequiresDev {
				normalized, err := NormalizeProjectName(name)
				if err != nil || name != normalized {
					return invalid
				}
				if _, exists := item.Groups[name]; !exists && len(rs) != 0 {
					return fmt.Errorf("UV native group metadata is absent from locked scope: %s", name)
				}
				for _, r := range rs {
					if !validUVNativeRequirement(r) {
						return invalid
					}
				}
			}
			seen := map[string]bool{}
			for _, name := range item.Metadata.ProvidesExtras {
				normalized, err := NormalizeProjectName(name)
				if err != nil || name != normalized || seen[name] {
					return invalid
				}
				seen[name] = true
				if _, exists := item.Optional[name]; !exists {
					return fmt.Errorf("UV native extra metadata is absent from locked scope: %s", name)
				}
			}
		}
		for _, edge := range all {
			count++
			if count > maxPyPIReportEntries {
				return invalid
			}
			normalized, err := NormalizeProjectName(edge.Name)
			if err != nil || normalized != edge.Name {
				return invalid
			}
			if edge.Source != nil && !uvPublicProjectIndex(edge.Source.Registry) {
				return invalid
			}
			target := ""
			if edge.Version != "" {
				target = edge.Name + "@" + edge.Version
			} else if len(byName[edge.Name]) == 1 {
				target = byName[edge.Name][0]
			} else {
				return invalid
			}
			selected, exists := variants[target]
			if !exists || target == root {
				return invalid
			}
			if edge.Marker != "" && !validUVProjectText([]byte(edge.Marker), false) {
				return invalid
			}
			extras := map[string]bool{}
			for _, name := range edge.Extra {
				canonical, err := NormalizeProjectName(name)
				if err != nil || name != canonical || extras[name] {
					return invalid
				}
				extras[name] = true
				if _, exists := selected.Optional[name]; !exists {
					return invalid
				}
			}
			edges[key] = append(edges[key], target)
		}
	}
	seen := map[string]bool{}
	pending := []string{root}
	for len(pending) > 0 {
		key := pending[0]
		pending = pending[1:]
		if seen[key] {
			continue
		}
		if _, exists := variants[key]; !exists {
			return invalid
		}
		seen[key] = true
		pending = append(pending, edges[key]...)
	}
	if len(seen) != len(variants) {
		return errors.New("UV native universal graph contains an unreachable variant")
	}
	return nil
}

func validUVNativeRequirement(r uvNativeRequirement) bool {
	name, err := NormalizeProjectName(r.Name)
	if err != nil || name != r.Name {
		return false
	}
	text := r.Name
	if len(r.Extras) > 0 {
		seen := map[string]bool{}
		for _, name := range r.Extras {
			canonical, err := NormalizeProjectName(name)
			if err != nil || name != canonical || seen[name] {
				return false
			}
			seen[name] = true
		}
		text += "[" + strings.Join(r.Extras, ",") + "]"
	}
	text += r.Specifier
	if r.Marker != "" {
		text += "; " + r.Marker
	}
	_, err = uvProjectPublicRequirementName(text)
	return err == nil
}

// Project requires-python is already restricted to this bounded language.
// Compare its conjunction after the same parser, including native whitespace.
func uvProjectPythonMatches(left, right string) bool {
	a, err := ParseUpgradeRequirement("python" + left)
	if err != nil {
		return false
	}
	b, err := ParseUpgradeRequirement("python" + right)
	return err == nil && a.SameConstraints(b)
}
