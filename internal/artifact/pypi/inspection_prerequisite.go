package pypi

import (
	"errors"
	"strings"
)

// ValidateInspectionPrerequisite supports an explicitly selected leaf wheel.
// It cannot expand extras/dependencies or introduce startup hooks. Existing
// installed roles and destinations remain the only applicability/path model.
func ValidateInspectionPrerequisite(prerequisite WheelInspection, originals []WheelInspection, policy ResourcePolicy) (ObservationPlan, error) {
	if len(prerequisite.RequiresDist) != 0 || len(prerequisite.Surface.SiteStartupHooks) != 0 {
		return ObservationPlan{}, errors.New("inspection prerequisite requires unsupported expansion or startup")
	}
	plan, err := BuildObservationPlan(prerequisite, policy)
	if err != nil || ValidateTypedObservationPlan(plan, policy) != nil || !plan.Admissible() {
		return ObservationPlan{}, errors.New("inspection prerequisite surface is nonqualifying")
	}
	for _, original := range originals {
		if original.Project == prerequisite.Project {
			return ObservationPlan{}, errors.New("inspection prerequisite replaces a selected project")
		}
		for _, added := range prerequisite.Surface.InstalledFiles {
			for _, present := range original.Surface.InstalledFiles {
				if added.Destination == present.Destination || strings.HasPrefix(added.Destination, present.Destination+"/") || strings.HasPrefix(present.Destination, added.Destination+"/") {
					return ObservationPlan{}, errors.New("inspection prerequisite destination collides")
				}
			}
		}
		roots := installedImportRoots(original.Surface)
		for root := range installedImportRoots(prerequisite.Surface) {
			if roots[root] {
				return ObservationPlan{}, errors.New("inspection prerequisite shadows selected imports")
			}
		}
	}
	return plan, nil
}

func installedImportRoots(surface RuntimeSurface) map[string]bool {
	roots := make(map[string]bool)
	for _, file := range surface.InstalledFiles {
		if file.Scheme != SchemeSite {
			continue
		}
		first := strings.SplitN(file.Destination, "/", 2)[0]
		if strings.Contains(file.Destination, "/") && validPythonImportComponent(first) {
			roots[first] = true
		} else if file.Role == RuntimeRolePythonModule {
			roots[strings.TrimSuffix(first, ".py")] = true
		} else if file.Role == RuntimeRolePythonExtension {
			if module, ok := pythonExtensionImportName(file.Destination); ok {
				roots[strings.SplitN(module, ".", 2)[0]] = true
			}
		}
	}
	return roots
}
