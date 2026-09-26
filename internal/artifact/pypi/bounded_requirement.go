package pypi

import (
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	boundedRequirementHead = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)(?:\[([^]]*)\])?\s*(.*)$`)
	numericRelease         = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)*$`)
	wildcardRelease        = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+){2,}\.\*$`)
)

var allowedCUDAToolkitExtras = map[string]bool{
	"cublas": true, "cudart": true, "cufft": true, "cufile": true, "cupti": true,
	"curand": true, "cusolver": true, "cusparse": true, "nvjitlink": true, "nvrtc": true, "nvtx": true,
}

// BoundedRequirement is the deliberately small requirement language accepted
// by the modern PyTorch graph resolver. It is not a general PEP 440 parser.
type BoundedRequirement struct {
	project     string
	extras      []string
	constraints []boundedConstraint
}

type boundedConstraint struct {
	operator, version string
	prefix            bool
}

func (r BoundedRequirement) Project() string { return r.project }

// Extras returns the canonical, bounded extra set requested for this
// requirement. Extras are requirement semantics: they select dependency edges
// and must survive project-level aggregation.
func (r BoundedRequirement) Extras() []string { return append([]string(nil), r.extras...) }

// ParseBoundedRequirement accepts only bare names, numeric exact/compatible
// releases, release prefixes, and comparison conjunctions. cuda-toolkit extras require
// the one active Linux marker emitted by the supported CUDA metadata.
func ParseBoundedRequirement(value string) (BoundedRequirement, error) {
	if value == "" || value != strings.TrimSpace(value) || strings.Contains(value, "@") || strings.Count(value, ";") > 1 {
		return BoundedRequirement{}, errors.New("bounded requirement is invalid")
	}
	base := value
	marker := ""
	if strings.Contains(value, ";") {
		parts := strings.SplitN(value, ";", 2)
		base, marker = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if marker != `platform_system == "Linux"` && marker != "platform_system == 'Linux'" {
			return BoundedRequirement{}, errors.New("bounded requirement marker is unsupported")
		}
	}
	matches := boundedRequirementHead.FindStringSubmatch(base)
	if matches == nil || matches[1] == "" {
		return BoundedRequirement{}, errors.New("bounded requirement project is invalid")
	}
	project, err := NormalizeProjectName(matches[1])
	if err != nil {
		return BoundedRequirement{}, errors.New("bounded requirement project is invalid")
	}
	extras, specifier := matches[2], strings.TrimSpace(matches[3])
	if extras != "" {
		if project != "cuda-toolkit" || marker == "" {
			return BoundedRequirement{}, errors.New("bounded requirement extras are unsupported")
		}
		seen := map[string]bool{}
		for _, extra := range strings.Split(extras, ",") {
			extra = strings.TrimSpace(extra)
			if extra == "" || !allowedCUDAToolkitExtras[extra] || seen[extra] {
				return BoundedRequirement{}, errors.New("bounded requirement extras are invalid")
			}
			seen[extra] = true
		}
	} else if marker != "" {
		return BoundedRequirement{}, errors.New("bounded requirement marker is unsupported")
	}
	result := BoundedRequirement{project: project}
	if extras != "" {
		for _, extra := range strings.Split(extras, ",") {
			result.extras = append(result.extras, strings.TrimSpace(extra))
		}
		sort.Strings(result.extras)
	}
	if specifier == "" {
		return result, nil
	}
	for _, part := range strings.Split(specifier, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return BoundedRequirement{}, errors.New("bounded requirement constraint is invalid")
		}
		operator := ""
		for _, candidate := range []string{"==", "~=", ">=", "<=", ">", "<"} {
			if strings.HasPrefix(part, candidate) {
				operator = candidate
				break
			}
		}
		if operator == "" {
			return BoundedRequirement{}, errors.New("bounded requirement operator is unsupported")
		}
		version := strings.TrimSpace(strings.TrimPrefix(part, operator))
		prefix := wildcardRelease.MatchString(version)
		if version == "" || strings.ContainsAny(version, " <>!=~+@") || (!numericRelease.MatchString(version) && !(operator == "==" && prefix)) || (operator == "~=" && !strings.Contains(version, ".")) {
			return BoundedRequirement{}, errors.New("bounded requirement version is invalid")
		}
		result.constraints = append(result.constraints, boundedConstraint{operator: operator, version: version, prefix: prefix})
	}
	return result, nil
}

// Satisfies reports whether a final numeric candidate version meets every
// bounded constraint. Unsupported candidate versions fail closed.
func (r BoundedRequirement) Satisfies(version string) bool {
	candidate, ok := parseNumericRelease(version)
	if !ok {
		return false
	}
	for _, constraint := range r.constraints {
		wantText := strings.TrimSuffix(constraint.version, ".*")
		want, ok := parseNumericRelease(wantText)
		if !ok {
			return false
		}
		comparison := compareNumericRelease(candidate, want)
		valid := false
		switch constraint.operator {
		case "==":
			valid = comparison == 0
			if constraint.prefix {
				valid = len(candidate) >= len(want) && comparisonPrefix(candidate, want)
			}
		case "~=":
			// PEP 440: ~=X.Y.Z is >=X.Y.Z together with ==X.Y.*.
			// Compare a zero-padded prefix without incrementing an integer
			// upper bound, so large components cannot overflow into acceptance.
			valid = comparison >= 0 && len(want) >= 2 && compatibleReleasePrefix(candidate, want[:len(want)-1])
		case ">=":
			valid = comparison >= 0
		case ">":
			valid = comparison > 0
		case "<=":
			valid = comparison <= 0
		case "<":
			valid = comparison < 0
		}
		if !valid {
			return false
		}
	}
	return true
}

func compatibleReleasePrefix(candidate, prefix []uint64) bool {
	for i, value := range prefix {
		var actual uint64
		if i < len(candidate) {
			actual = candidate[i]
		}
		if actual != value {
			return false
		}
	}
	return true
}

// CandidateSatisfiesBoundedRequirements independently checks every incoming
// bounded requirement after pip selected a single source-pinned candidate.
func CandidateSatisfiesBoundedRequirements(version string, requirements []BoundedRequirement) bool {
	if len(requirements) == 0 {
		return false
	}
	for _, requirement := range requirements {
		if !requirement.Satisfies(version) {
			return false
		}
	}
	return true
}

func parseNumericRelease(value string) ([]uint64, bool) {
	if !numericRelease.MatchString(value) {
		return nil, false
	}
	parts := strings.Split(value, ".")
	result := make([]uint64, len(parts))
	for i, part := range parts {
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return nil, false
		}
		result[i] = n
	}
	return result, true
}
func compareNumericRelease(a, b []uint64) int {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		var x, y uint64
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}
func comparisonPrefix(candidate, prefix []uint64) bool {
	for i := range prefix {
		if candidate[i] != prefix[i] {
			return false
		}
	}
	return true
}

// AggregatedBoundedRequirement is the one deterministic pip request produced
// for a same-source project. It retains the union of every accepted requested
// extra so resolution cannot silently drop extra-activated dependencies.
type AggregatedBoundedRequirement struct {
	request string
	extras  []string
}

func (r AggregatedBoundedRequirement) Request() string  { return r.request }
func (r AggregatedBoundedRequirement) Extras() []string { return append([]string(nil), r.extras...) }

// AggregateBoundedRequirements produces the sole pip request used for a
// same-source project, retaining constraints and the canonical union of its
// bounded extras.
func AggregateBoundedRequirements(project string, requirements []BoundedRequirement) (AggregatedBoundedRequirement, error) {
	if len(requirements) == 0 {
		return AggregatedBoundedRequirement{}, errors.New("bounded requirement set is empty")
	}
	parts := make([]string, 0)
	extraSet := map[string]bool{}
	for _, requirement := range requirements {
		if requirement.project != project {
			return AggregatedBoundedRequirement{}, errors.New("bounded requirement project mismatch")
		}
		for _, extra := range requirement.extras {
			extraSet[extra] = true
		}
		for _, constraint := range requirement.constraints {
			parts = append(parts, constraint.operator+constraint.version)
		}
	}
	extras := make([]string, 0, len(extraSet))
	for extra := range extraSet {
		extras = append(extras, extra)
	}
	sort.Strings(extras)
	if len(extras) > 0 && project != "cuda-toolkit" {
		return AggregatedBoundedRequirement{}, errors.New("bounded requirement extras are invalid")
	}
	name := project
	if len(extras) > 0 {
		name += "[" + strings.Join(extras, ",") + "]"
	}
	if len(parts) == 0 {
		if len(extras) == 0 {
			return AggregatedBoundedRequirement{request: name}, nil
		}
		return AggregatedBoundedRequirement{request: name + `; platform_system == "Linux"`, extras: extras}, nil
	}
	request := name + strings.Join(parts, ",")
	if len(extras) > 0 {
		request += `; platform_system == "Linux"`
	}
	return AggregatedBoundedRequirement{request: request, extras: extras}, nil
}

// AggregateBoundedRequirement is retained as the string-only compatibility
// view. Resolver code must use AggregateBoundedRequirements to preserve
// extra semantics.
func AggregateBoundedRequirement(project string, requirements []BoundedRequirement) (string, error) {
	aggregated, err := AggregateBoundedRequirements(project, requirements)
	if err != nil {
		return "", err
	}
	return aggregated.Request(), nil
}
