package pypi

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// InstallationScheme identifies the deterministic target filesystem scheme for
// one wheel payload file.
type InstallationScheme string

const (
	SchemeSite     InstallationScheme = "site"      // purelib/platlib -> site-packages
	SchemeScripts  InstallationScheme = "scripts"   // .data/scripts -> bin
	SchemeHeaders  InstallationScheme = "headers"   // .data/headers -> include
	SchemeData     InstallationScheme = "data"      // .data/data -> share
	SchemeDistInfo InstallationScheme = "dist-info" // <dist>.dist-info -> site-packages/<dist>.dist-info
)

// RuntimeRole classifies one file according to its runtime behavior.
type RuntimeRole string

const (
	RuntimeRoleSiteStartupHook      RuntimeRole = "SITE_STARTUP_HOOK"     // root .pth in site-packages
	RuntimeRolePythonModule         RuntimeRole = "PYTHON_MODULE"         // .py module in site-packages
	RuntimeRolePythonPackage        RuntimeRole = "PYTHON_PACKAGE"        // package directory with __init__.py
	RuntimeRoleNamespaceContainer   RuntimeRole = "NAMESPACE_CONTAINER"   // directory without __init__.py containing packages
	RuntimeRoleExecutableSubpackage RuntimeRole = "EXECUTABLE_SUBPACKAGE" // subpackage under namespace container
	RuntimeRolePythonExtension      RuntimeRole = "PYTHON_EXTENSION"      // target CPython extension, .abi3, or bare .so with PyInit
	RuntimeRoleNativeLibrary        RuntimeRole = "NATIVE_LIBRARY"        // ordinary native library without PyInit
	RuntimeRoleScript               RuntimeRole = "SCRIPT"                // executable script in bin
	RuntimeRoleMetadata             RuntimeRole = "METADATA"              // canonical dist-info metadata
	RuntimeRoleInertData            RuntimeRole = "INERT_DATA"            // headers, licenses, docs, inert data
	RuntimeRoleUnresolved           RuntimeRole = "UNRESOLVED"            // unclassified/ambiguous -> fail closed
)

// InstalledFile is the deterministic installed-location and role mapping of one
// archive path.
type InstalledFile struct {
	ArchivePath string             `json:"archive_path"`
	Scheme      InstallationScheme `json:"scheme"`
	Destination string             `json:"destination"`
	Role        RuntimeRole        `json:"role"`
	Size        int64              `json:"size"`
	SHA256      string             `json:"sha256"`
}

// EntryPoint models one canonical distribution entry point.
type EntryPoint struct {
	Group  string `json:"group"`
	Name   string `json:"name"`
	Module string `json:"module"`
	Attr   string `json:"attr,omitempty"`
}

// SiteHookLine is one statically parsed, active site-level .pth line. A path
// line is declarative; an import line is executable and needs its own unit.
type SiteHookLine struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	Path      string `json:"path,omitempty"`
	Statement string `json:"statement,omitempty"`
}

type ObservationUnitKind string

const (
	DirectImportUnit     ObservationUnitKind = "DIRECT_IMPORT"
	ActivePTHHookUnit    ObservationUnitKind = "ACTIVE_PTH_HOOK"
	InstalledStartupUnit ObservationUnitKind = "INSTALLED_STARTUP_SCENARIO"
)

// PlannedObservationUnit is immutable controller input. Its identity is
// subsequently bound to the authenticated closure manifest by the executor.
type PlannedObservationUnit struct {
	Kind      ObservationUnitKind `json:"kind"`
	Candidate string              `json:"candidate,omitempty"`
	HookFile  string              `json:"hook_file,omitempty"`
	HookLine  int                 `json:"hook_line,omitempty"`
	Statement string              `json:"statement,omitempty"`
}

// ExecutableCoverageOutcome records the deterministic observation disposition of an executable surface.
type ExecutableCoverageOutcome string

const (
	CoverageObserved               ExecutableCoverageOutcome = "OBSERVED"
	CoverageCoveredByModule        ExecutableCoverageOutcome = "COVERED_BY_EXPLICIT_MODULE_OBSERVATION"
	CoverageNotApplicableWithProof ExecutableCoverageOutcome = "NOT_APPLICABLE_WITH_PROOF"
	CoverageManualReview           ExecutableCoverageOutcome = "MANUAL_REVIEW"
)

// RuntimeSurface accounts for all categorized runtime surfaces in a wheel.
type RuntimeSurface struct {
	InstalledFiles        []InstalledFile   `json:"installed_files"`
	SiteStartupHooks      []string          `json:"site_startup_hooks,omitempty"`
	SiteHookLines         []SiteHookLine    `json:"site_hook_lines,omitempty"`
	PythonModules         []string          `json:"python_modules,omitempty"`
	PythonPackages        []string          `json:"python_packages,omitempty"`
	NamespaceContainers   []string          `json:"namespace_containers,omitempty"`
	ExecutableSubpackages []string          `json:"executable_subpackages,omitempty"`
	PythonExtensions      []string          `json:"python_extensions,omitempty"`
	NativeLibraries       []string          `json:"native_libraries,omitempty"`
	Scripts               []string          `json:"scripts,omitempty"`
	EntryPoints           []string          `json:"entry_points,omitempty"`
	EntryPointDetails     []EntryPoint      `json:"entry_point_details,omitempty"`
	EntryPointCoverage    map[string]string `json:"entry_point_coverage,omitempty"`
	ScriptCoverage        map[string]string `json:"script_coverage,omitempty"`
	InertDataFiles        []string          `json:"inert_data_files,omitempty"`
	UnresolvedFiles       []string          `json:"unresolved_files,omitempty"`
}

// ObservationPlan represents the validated, bounded execution plan agreed upon
// by the static planner and the dynamic sandbox backend.
type ObservationPlan struct {
	Project            string                   `json:"project"`
	Version            string                   `json:"version"`
	SiteStartupHooks   []string                 `json:"site_startup_hooks,omitempty"`
	SiteHookLines      []SiteHookLine           `json:"site_hook_lines,omitempty"`
	ImportCandidates   []string                 `json:"import_candidates,omitempty"`
	EntryPointCoverage map[string]string        `json:"entry_point_coverage,omitempty"`
	EntryPointDetails  []EntryPoint             `json:"entry_point_details,omitempty"`
	ScriptCoverage     map[string]string        `json:"script_coverage,omitempty"`
	NoImportSurface    bool                     `json:"no_import_surface"`
	MetadataOnly       bool                     `json:"metadata_only"`
	TotalImportCount   int                      `json:"total_import_count"`
	Units              []PlannedObservationUnit `json:"units,omitempty"`
}

// BuildObservationPlan validates the complete installed runtime surface.
// Units are the execution authority; each direct import is an independent experiment.
func BuildObservationPlan(inspection WheelInspection, policy ResourcePolicy) (ObservationPlan, error) {
	if len(inspection.Surface.UnresolvedFiles) > 0 {
		return ObservationPlan{}, fmt.Errorf("wheel contains %d unresolved surface(s): %v", len(inspection.Surface.UnresolvedFiles), inspection.Surface.UnresolvedFiles)
	}

	if inspection.NoImportSurface {
		if len(inspection.ImportNames) != 0 {
			return ObservationPlan{}, errors.New("no-import surface artifact must not have candidate imports")
		}
		if len(inspection.Surface.SiteStartupHooks) != 0 {
			return ObservationPlan{}, errors.New("no-import surface artifact must not have active site startup hooks")
		}
		if len(inspection.Surface.Scripts) != 0 {
			return ObservationPlan{}, errors.New("no-import surface artifact must not have installed scripts")
		}
		isMetadataOnly := len(inspection.Surface.InstalledFiles) > 0
		for _, f := range inspection.Surface.InstalledFiles {
			if f.Scheme != SchemeDistInfo {
				isMetadataOnly = false
				break
			}
		}
		if !provenNoImportSurfaceRoles(inspection.Surface.InstalledFiles) || len(inspection.Surface.EntryPoints) != 0 || len(inspection.Surface.UnresolvedFiles) != 0 {
			return ObservationPlan{}, errors.New("no-import surface is not proven by installed roles")
		}
		return ObservationPlan{
			Project:            inspection.Project,
			Version:            inspection.Version,
			SiteStartupHooks:   nil,
			ImportCandidates:   nil,
			EntryPointCoverage: nil,
			ScriptCoverage:     nil,
			NoImportSurface:    true,
			MetadataOnly:       isMetadataOnly,
			TotalImportCount:   0,
		}, nil
	}

	candidates := append([]string(nil), inspection.ImportNames...)
	for _, ep := range inspection.Surface.EntryPointDetails {
		if !containsCandidate(candidates, ep.Module) && installedModuleExists(inspection.Surface.InstalledFiles, ep.Module) {
			candidates = append(candidates, ep.Module)
		}
	}
	sort.Strings(candidates)
	if len(candidates) == 0 && len(inspection.Surface.Scripts) == 0 && len(inspection.Surface.EntryPoints) == 0 && len(inspection.Surface.SiteStartupHooks) == 0 {
		return ObservationPlan{}, errors.New("dynamic python artifact has no candidate imports")
	}

	limit := policy.MaxObservationImportsPerArtifact()
	if limit <= 0 {
		limit = 32
	}
	if len(candidates) > limit {
		return ObservationPlan{}, fmt.Errorf("import candidate count %d exceeds policy bound %d", len(candidates), limit)
	}

	hooks := make([]string, len(inspection.Surface.SiteStartupHooks))
	copy(hooks, inspection.Surface.SiteStartupHooks)
	sort.Strings(hooks)
	hookLines := append([]SiteHookLine(nil), inspection.Surface.SiteHookLines...)
	sort.Slice(hookLines, func(i, j int) bool {
		if hookLines[i].File != hookLines[j].File {
			return hookLines[i].File < hookLines[j].File
		}
		return hookLines[i].Line < hookLines[j].Line
	})
	units := make([]PlannedObservationUnit, 0, len(candidates)+len(inspection.Surface.SiteHookLines)+1)
	for _, candidate := range candidates {
		units = append(units, PlannedObservationUnit{Kind: DirectImportUnit, Candidate: candidate})
	}
	for _, line := range hookLines {
		if line.Statement != "" {
			units = append(units, PlannedObservationUnit{Kind: ActivePTHHookUnit, HookFile: line.File, HookLine: line.Line, Statement: line.Statement})
		}
	}
	if len(hooks) != 0 {
		units = append(units, PlannedObservationUnit{Kind: InstalledStartupUnit})
	}
	if len(units) > limit {
		return ObservationPlan{}, fmt.Errorf("observation unit count %d exceeds policy bound %d", len(units), limit)
	}

	epCoverage := make(map[string]string)
	for _, ep := range inspection.Surface.EntryPointDetails {
		key := ep.Group + ": " + ep.Name + " = " + ep.Module
		if ep.Attr != "" {
			key += ":" + ep.Attr
		}
		if containsCandidate(candidates, ep.Module) {
			epCoverage[key] = string(CoverageCoveredByModule)
		} else {
			epCoverage[key] = string(CoverageManualReview)
		}
	}
	for _, raw := range inspection.Surface.EntryPoints {
		if _, ok := epCoverage[raw]; !ok {
			epCoverage[raw] = string(CoverageManualReview)
		}
	}

	scriptCoverage := make(map[string]string)
	for _, sc := range inspection.Surface.Scripts {
		scriptCoverage[sc] = string(CoverageManualReview)
	}

	return ObservationPlan{
		Project:            inspection.Project,
		Version:            inspection.Version,
		SiteStartupHooks:   hooks,
		SiteHookLines:      hookLines,
		ImportCandidates:   candidates,
		EntryPointCoverage: epCoverage,
		EntryPointDetails:  append([]EntryPoint(nil), inspection.Surface.EntryPointDetails...),
		ScriptCoverage:     scriptCoverage,
		NoImportSurface:    false,
		MetadataOnly:       false,
		TotalImportCount:   len(candidates),
		Units:              units,
	}, nil
}

func containsCandidate(candidates []string, target string) bool {
	for _, candidate := range candidates {
		if candidate == target {
			return true
		}
	}
	return false
}

func installedModuleExists(files []InstalledFile, module string) bool {
	stem := strings.ReplaceAll(module, ".", "/")
	for _, file := range files {
		if file.Scheme != SchemeSite {
			continue
		}
		if file.Destination == stem+".py" || file.Destination == stem+"/__init__.py" {
			return file.Role == RuntimeRolePythonModule || file.Role == RuntimeRolePythonPackage || file.Role == RuntimeRoleExecutableSubpackage
		}
		if file.Role == RuntimeRolePythonExtension && (file.Destination == stem+".cpython-314-x86_64-linux-gnu.so" || file.Destination == stem+".abi3.so" || file.Destination == stem+".so") {
			return true
		}
	}
	return false
}

// Admissible rejects every unsupported executable disposition before any
// wheel code is introduced to a runtime.
func (p ObservationPlan) Admissible() bool {
	for _, outcome := range p.EntryPointCoverage {
		if outcome != string(CoverageCoveredByModule) {
			return false
		}
	}
	for _, outcome := range p.ScriptCoverage {
		if outcome != string(CoverageCoveredByModule) {
			return false
		}
	}
	return !strings.Contains(p.Project, "\x00")
}

// ValidateTypedObservationPlan reconciles all executable facts before the
// controller creates a transaction. The executor consumes Units directly.
func ValidateTypedObservationPlan(p ObservationPlan, policy ResourcePolicy) error {
	limit := policy.MaxObservationImportsPerArtifact()
	if p.Project == "" || p.Version == "" || len(p.ImportCandidates) > limit || len(p.Units) > limit ||
		p.TotalImportCount != len(p.ImportCandidates) || len(p.SiteStartupHooks) > limit {
		return errors.New("observation plan exceeds policy or lacks identity")
	}
	if p.NoImportSurface {
		if len(p.ImportCandidates) != 0 || len(p.SiteStartupHooks) != 0 || len(p.SiteHookLines) != 0 || len(p.Units) != 0 || len(p.EntryPointCoverage) != 0 || len(p.ScriptCoverage) != 0 {
			return errors.New("no-import plan contains executable surface")
		}
		return nil
	}
	for i, candidate := range p.ImportCandidates {
		if !validPythonImportName(candidate) || i > 0 && p.ImportCandidates[i-1] >= candidate {
			return errors.New("observation import identity is invalid")
		}
	}
	if len(p.ImportCandidates) == 0 && len(p.SiteStartupHooks) == 0 && len(p.ScriptCoverage) == 0 && len(p.EntryPointCoverage) == 0 {
		return errors.New("executable plan has no surface")
	}
	for i, hook := range p.SiteStartupHooks {
		if hook == "" || strings.Contains(hook, "/") || !strings.HasSuffix(hook, ".pth") || i > 0 && p.SiteStartupHooks[i-1] >= hook {
			return errors.New("active site hook identity is invalid")
		}
	}
	for i, line := range p.SiteHookLines {
		if line.Line < 1 || !containsString(p.SiteStartupHooks, line.File) || (line.Path == "") == (line.Statement == "") ||
			i > 0 && (p.SiteHookLines[i-1].File > line.File || p.SiteHookLines[i-1].File == line.File && p.SiteHookLines[i-1].Line >= line.Line) {
			return errors.New("active site hook line is invalid")
		}
	}
	expected := make([]PlannedObservationUnit, 0, len(p.ImportCandidates)+len(p.SiteHookLines)+1)
	for _, candidate := range p.ImportCandidates {
		expected = append(expected, PlannedObservationUnit{Kind: DirectImportUnit, Candidate: candidate})
	}
	for _, line := range p.SiteHookLines {
		if line.Statement != "" {
			expected = append(expected, PlannedObservationUnit{Kind: ActivePTHHookUnit, HookFile: line.File, HookLine: line.Line, Statement: line.Statement})
		}
	}
	if len(p.SiteStartupHooks) != 0 {
		expected = append(expected, PlannedObservationUnit{Kind: InstalledStartupUnit})
	}
	if len(expected) != len(p.Units) {
		return errors.New("observation unit set is incomplete")
	}
	for i := range expected {
		if expected[i] != p.Units[i] {
			return errors.New("observation unit is substituted or duplicated")
		}
	}
	if len(p.EntryPointCoverage) < len(p.EntryPointDetails) {
		return errors.New("entry point coverage is incomplete")
	}
	knownEntryPoints := make(map[string]bool, len(p.EntryPointDetails))
	for _, ep := range p.EntryPointDetails {
		key := ep.Group + ": " + ep.Name + " = " + ep.Module
		if ep.Attr != "" {
			key += ":" + ep.Attr
		}
		want := string(CoverageManualReview)
		if containsCandidate(p.ImportCandidates, ep.Module) {
			want = string(CoverageCoveredByModule)
		}
		if p.EntryPointCoverage[key] != want {
			return errors.New("entry point target coverage is inexact")
		}
		knownEntryPoints[key] = true
	}
	for key, outcome := range p.EntryPointCoverage {
		if !knownEntryPoints[key] && outcome != string(CoverageManualReview) {
			return errors.New("undeclared entry point coverage is unsafe")
		}
	}
	for _, outcome := range p.ScriptCoverage {
		if outcome != string(CoverageManualReview) {
			return errors.New("installed script disposition is unsupported")
		}
	}
	return nil
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
