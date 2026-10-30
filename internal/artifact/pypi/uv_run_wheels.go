package pypi

import (
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// BindUVRunWheels binds root extras only through authenticated wheel metadata.
// The complete source, constraints and pre-query startup gate remain shared
// with the existing single-root lane. The graph grants no execution admission.
func (l UVLock) BindUVRunWheels(request UVRunRequirement, inputs []UVWheelInput, pages []SimpleProject, target WheelTarget, limits WheelLimits) (domain.LockedDependencyGraph, error) {
	if !request.Valid() || request.Reference() != l.reference {
		return domain.LockedDependencyGraph{}, errors.New("UV run request differs from the locked primary")
	}
	if len(request.extras) == 0 {
		return l.BindPipxQueryWheels(inputs, pages, target, limits)
	}
	checked, err := bindUVSelectedWheelsWithSurface(l.packages, l.python, request.base.Project(), request.extras, inputs, pages, target, limits, validatePipxMetadataSurface)
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	return BuildLockedGraph(l.reference, checked)
}

func candidateFromUVRunWheelMetadata(inspection WheelInspection, distributionURL, declaredSHA, python string, extras []string) (Candidate, error) {
	declared := map[string]bool{}
	for _, raw := range inspection.providesExtra {
		name, err := NormalizeProjectName(raw)
		if err != nil || declared[name] {
			return Candidate{}, errors.New("UV run wheel has ambiguous extra declarations")
		}
		declared[name] = true
	}
	for _, name := range extras {
		if !declared[name] {
			return Candidate{}, errors.New("UV run requested extra is absent from the primary wheel")
		}
	}
	return parseReportCandidateWithDependencies(wheelCandidateMetadata(inspection, distributionURL, declaredSHA, true), PublicPyPIProfile(), extras, func(value string) (string, bool, error) {
		return uvRunDependency(value, python, extras)
	})
}

func uvRunDependency(value, python string, extras []string) (string, bool, error) {
	if !strings.Contains(value, ";") {
		dependency, err := parseDeclaredDependency(value)
		return dependency, true, err
	}
	if strings.Count(value, ";") != 1 {
		return "", false, errors.New("unsupported UV run dependency marker")
	}
	parts := strings.SplitN(value, ";", 2)
	ast, err := parseAndValidateBoundedMarker(strings.TrimSpace(parts[1]), python)
	if err != nil {
		return "", false, err
	}
	if err := normalizeUVRunExtraMarkers(ast); err != nil {
		return "", false, err
	}
	// Extras add to the base dependency set. Evaluate the fully validated AST
	// in the base environment and independently for every requested extra.
	active, err := ast.evaluate(python, "")
	if err != nil {
		return "", false, err
	}
	for _, extra := range extras {
		truth, err := ast.evaluate(python, extra)
		if err != nil {
			return "", false, err
		}
		active = active || truth
	}
	if !active {
		return "", false, nil
	}
	dependency, err := parseDeclaredDependency(strings.TrimSpace(parts[0]))
	return dependency, true, err
}

func normalizeUVRunExtraMarkers(ast markerAST) error {
	switch node := ast.(type) {
	case *markerBinaryNode:
		if err := normalizeUVRunExtraMarkers(node.left); err != nil {
			return err
		}
		return normalizeUVRunExtraMarkers(node.right)
	case *markerAtomNode:
		if node.name == "extra" && node.literal != "" {
			name, err := NormalizeProjectName(node.literal)
			if err != nil {
				return err
			}
			node.literal = name
		}
		return nil
	default:
		return errors.New("unsupported UV run dependency marker")
	}
}
