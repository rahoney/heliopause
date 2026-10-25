package pypi

import (
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// BindWheels binds native tool claims to the same complete archive/source and
// dependency validation as UV installation. The graph remains unapproved;
// runtime admission, current inspection/Policy and held ownership are separate.
func (r ToolReceipt) BindWheels(lock UVLock, inputs []UVWheelInput, pages []SimpleProject, target WheelTarget, limits WheelLimits) (domain.LockedDependencyGraph, error) {
	root, err := RequestedProject(lock.reference)
	if !r.Valid() || err != nil || root != r.requirement.Project() || len(lock.packages) == 0 {
		return domain.LockedDependencyGraph{}, errors.New("tool receipt differs from the frozen native root")
	}
	rootSeen := false
	candidates, err := inspectUVArchiveCandidates(lock.packages, lock.python, root, inputs, target, limits, func(wheel WheelInspection) error {
		if r.client == "pipx" {
			if err := validatePipxMetadataSurface(wheel); err != nil {
				return err
			}
		}
		if wheel.Project != root {
			return nil
		}
		if rootSeen || !r.requirement.AcceptsVersion(wheel.Version) || r.version != "" && r.version != wheel.Version {
			return errors.New("native tool version differs from its authenticated main wheel")
		}
		rootSeen = true
		if len(wheel.Surface.Scripts) != 0 {
			return errors.New("native tool data scripts are outside the reviewed entrypoint subset")
		}
		names := map[string]bool{}
		for _, ep := range wheel.Surface.EntryPointDetails {
			if ep.Group == "gui_scripts" {
				return errors.New("GUI tool exposure is outside the reviewed entrypoint subset")
			}
			if ep.Group != "console_scripts" {
				continue
			}
			if !validToolCommand(ep.Name) || names[strings.ToLower(ep.Name)] || len(ep.Extras) != 0 {
				return errors.New("tool wheel command identity is ambiguous or requires extras")
			}
			names[strings.ToLower(ep.Name)] = true
		}
		if len(names) != len(r.commands) {
			return errors.New("native tool command coverage differs from its main wheel")
		}
		for _, command := range r.commands {
			// Require exact case, in addition to collision detection above.
			found := false
			for _, ep := range wheel.Surface.EntryPointDetails {
				if ep.Group == "console_scripts" && ep.Name == command.Name {
					found = true
				}
			}
			if !found {
				return errors.New("native tool command is absent from its main wheel")
			}
		}
		return nil
	})
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	if !rootSeen || len(pages) != len(lock.packages) {
		return domain.LockedDependencyGraph{}, errors.New("tool wheel or independent source coverage is incomplete")
	}
	checked, err := CrossCheckReport(InstallationReport{candidates: candidates}, pages)
	if err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	if err := validateUVCandidateConstraints(checked, lock.python); err != nil {
		return domain.LockedDependencyGraph{}, err
	}
	return BuildLockedGraph(lock.reference, checked)
}

// pipx's native metadata query starts the installed interpreter without -S.
// This initial lane rejects startup surfaces across the complete closure before
// that query. It does not certify safety or waive Artifact runtime admission.
func validatePipxMetadataSurface(wheel WheelInspection) error {
	if len(wheel.Surface.SiteStartupHooks) != 0 {
		return errors.New("pipx metadata requires a separately admitted startup-hook execution lane")
	}
	for _, file := range wheel.Surface.InstalledFiles {
		if file.Scheme != SchemeSite {
			continue
		}
		head := strings.SplitN(file.Destination, "/", 2)[0]
		stem := strings.SplitN(head, ".", 2)[0]
		if stem == "sitecustomize" || stem == "usercustomize" {
			return errors.New("pipx metadata would start an installed customization module")
		}
	}
	return nil
}
