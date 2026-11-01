package sandbox

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// Native npm owns the root's full install lifecycle, including implicit gyp and
// prepare phases. It receives only measured local sources and offline archives.
const npmProjectRootLifecycleCommand = `cd /tmp/haa-node-execution/project; mkdir -p /tmp/.npm; HOME=/tmp npm_config_cache=/tmp/.npm npm_config_script_shell=/haa-runtime/haa-boundary npm ci --ignore-scripts=false --bin-links=true --foreground-scripts --offline --no-update-notifier --workspaces=false --no-audit --no-fund --dangerously-allow-all-scripts`

func npmProjectRootLifecycleExecArguments(container string) ([]string, error) {
	if !exactObservationContainerID(container) {
		return nil, errors.New("npm root lifecycle requires an exact container")
	}
	return boundaryExecArguments(container, boundaryLaunchMode, "/bin/sh", "-ceu", npmProjectRootLifecycleCommand, "helox-npm-project-root", nodeExecutionProject), nil
}

func (b *Backend) ExecuteNPMProjectRoot(ctx context.Context, input artifactnpm.NPMProjectRootInspectionInput) (domain.SandboxResult, error) {
	if b == nil || ctx == nil || b.runner == nil || b.probe == nil || b.newSessionID == nil || !input.Valid() || (input.Dependencies().Context().RuntimeIdentity() != NPMProjectRuntimeIdentity() || input.Dependencies().Context().ClientVersion() != resolverNPMVersion) {
		return domain.SandboxResult{}, errors.New("npm root lifecycle lacks current complete source context")
	}
	observer, ok := b.observer.(interface {
		StartNPMProjectTargetLifecycleProfile(context.Context, string) (TraceReader, error)
	})
	intake, measured := b.introducer.(*DockerArtifactIntroducer)
	if !ok || !measured {
		return domain.SandboxResult{}, errors.New("npm root lifecycle requires typed observation and measured intake")
	}
	reader, err := NewNPMProjectSourceReader(intake.intakeRoot)
	if err != nil {
		return domain.SandboxResult{}, err
	}
	source, err := reader.ReadSource(ctx, input.Source())
	if err != nil {
		return domain.SandboxResult{}, err
	}
	payload, err := b.prepareNPMRootInspectionPayload(ctx, input, source)
	if err != nil {
		return domain.SandboxResult{}, err
	}
	return b.executeNPMInstance(ctx, nativeRuntimeNPMProject, nodeExecutionProject, payload, observer.StartNPMProjectTargetLifecycleProfile)
}

func (b *Backend) prepareNPMRootInspectionPayload(ctx context.Context, input artifactnpm.NPMProjectRootInspectionInput, source artifactnpm.PNPMLayoutArchive) (pnpmInspectionPayload, error) {
	invalid := errors.New("npm root payload differs from its current measured sources or workspace budget")
	if b == nil || ctx == nil || !input.Valid() {
		return pnpmInspectionPayload{}, invalid
	}
	intake, ok := b.introducer.(*DockerArtifactIntroducer)
	if !ok {
		return pnpmInspectionPayload{}, invalid
	}
	controls, err := input.Dependencies().OfflineControls()
	if err != nil {
		return pnpmInspectionPayload{}, err
	}
	manifest, err := source.ReadFile("project/package.json")
	if err != nil || string(manifest) != string(controls[0].Body()) {
		return pnpmInspectionPayload{}, errors.Join(invalid, err)
	}
	var archives []pnpmInspectionArchive
	if len(input.Dependencies().Artifacts()) != 0 {
		archives, err = intake.freezeNPMProjectArchives(ctx, input.Dependencies().Artifacts())
		if err != nil {
			return pnpmInspectionPayload{}, err
		}
	}
	// Charge the resulting dependency tree as well as source and compressed
	// seeds before any runtime is created. The existing workspace cap is fixed.
	var storage int64
	charge := func(size int64) bool {
		if size < 0 || size > int64(artifactnpm.NPMProjectInspectionWorkspaceLimit)-4096 {
			return false
		}
		cost := int64(4096) + (size+4095)/4096*4096
		if storage > int64(artifactnpm.NPMProjectInspectionWorkspaceLimit)-cost {
			return false
		}
		storage += cost
		return true
	}
	for _, entries := range [][]artifactnpm.PNPMLayoutEntry{input.DependencyLayout().Entries(), source.Entries()} {
		for _, entry := range entries {
			if !charge(entry.Size) {
				return pnpmInspectionPayload{}, invalid
			}
		}
	}
	// The private archive seed adds two directories. Charge both separately.
	for range 2 {
		if !charge(0) {
			return pnpmInspectionPayload{}, invalid
		}
	}
	for _, archive := range archives {
		if !charge(int64(len(archive.body))) {
			return pnpmInspectionPayload{}, invalid
		}
	}
	for _, control := range controls {
		if !charge(int64(len(control.Body()))) {
			return pnpmInspectionPayload{}, invalid
		}
	}
	baseBody, err := nodeMaterializationPayload(ctx, archives, controls)
	if err != nil {
		return pnpmInspectionPayload{}, err
	}
	base, err := artifactnpm.ParsePNPMInspectionPayloadArchive(ctx, baseBody)
	if err != nil {
		return pnpmInspectionPayload{}, err
	}
	byName := map[string]artifactnpm.PNPMLayoutEntry{}
	fromSource := map[string]bool{}
	for _, entry := range base.Entries() {
		if entry.Name == "project" || entry.Name == "project/package.json" {
			continue
		}
		name, present := strings.CutPrefix(entry.Name, "project/")
		if !present {
			return pnpmInspectionPayload{}, invalid
		}
		entry.Name = name
		byName[name] = entry
	}
	for _, entry := range source.Entries() {
		if entry.Name == "project" || entry.Name == "project/package.json" || entry.Name == "project/package-lock.json" {
			continue
		}
		name, present := strings.CutPrefix(entry.Name, "project/")
		if !present || name == "node_modules" || strings.HasPrefix(name, "node_modules/") || byName[name].Name != "" || (entry.Kind != tar.TypeReg && entry.Kind != tar.TypeDir) {
			return pnpmInspectionPayload{}, invalid
		}
		entry.Name = name
		entry.Mode |= 0o200 // the local project's own files are writable inside isolation
		byName[name] = entry
		fromSource[name] = true
	}
	entries := make([]artifactnpm.PNPMLayoutEntry, 0, len(byName))
	for _, entry := range byName {
		entries = append(entries, entry)
	}
	binding, err := json.Marshal(struct{ Schema, Graph, Runtime, Source, Layout string }{"npm-project-root-input/v1", input.Dependencies().Context().Snapshot().GraphDigest().String(), NPMProjectRuntimeIdentity(), input.Source().Digest().String(), input.DependencyLayout().Digest().String()})
	if err != nil {
		return pnpmInspectionPayload{}, err
	}
	digest, err := pnpmPayloadSHA256(binding)
	if err != nil {
		return pnpmInspectionPayload{}, err
	}
	return renderNPMInspectionPayload(ctx, digest, manifest, entries, func(name string) ([]byte, error) {
		if fromSource[name] {
			return source.ReadFile("project/" + name)
		}
		return base.ReadFile("project/" + name)
	}, nativeRuntimeNPMProject)
}
