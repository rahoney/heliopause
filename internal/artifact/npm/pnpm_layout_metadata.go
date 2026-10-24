package npm

import (
	"bytes"
	"encoding/json"
	"maps"
	"path"
	"slices"
	"strings"
	"time"
)

const pnpmWorkspaceCachePath = "node_modules/.pnpm-workspace-state-v1.json"

// The pinned native writer creates this derived freshness cache. Its claims
// are not consumed by HAA or by the fixed npm lifecycle. Do not reproduce its
// settings/freshness algorithm or expose the file as bound execution payload.
func discardPNPMWorkspaceCache(layout PNPMLayoutArchive) (PNPMLayoutArchive, error) {
	body, err := layout.ReadFile(pnpmWorkspaceCachePath)
	if err != nil || strictControlJSON(body) != nil {
		return PNPMLayoutArchive{}, errPNPMInspectionLayout
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(body, &object) != nil || object == nil {
		return PNPMLayoutArchive{}, errPNPMInspectionLayout
	}
	entries := make([]PNPMLayoutEntry, 0, len(layout.entries)-1)
	for _, entry := range layout.entries {
		if entry.Name != pnpmWorkspaceCachePath {
			entries = append(entries, entry)
		}
	}
	files := maps.Clone(layout.files)
	delete(files, pnpmWorkspaceCachePath)
	return PNPMLayoutArchive{layout.body, entries, files}, nil
}

// Native generated metadata is format data, never build/Policy authority.
type pnpmLayoutModules struct {
	HoistedDependencies map[string]map[string]string `json:"hoistedDependencies"`
	HoistPattern        []string                     `json:"hoistPattern"`
	Included            struct {
		Dependencies         bool `json:"dependencies"`
		DevDependencies      bool `json:"devDependencies"`
		OptionalDependencies bool `json:"optionalDependencies"`
	} `json:"included"`
	LayoutVersion            int                        `json:"layoutVersion"`
	NodeLinker               string                     `json:"nodeLinker"`
	PackageManager           string                     `json:"packageManager"`
	PendingBuilds            []string                   `json:"pendingBuilds"`
	PrunedAt                 string                     `json:"prunedAt"`
	PublicHoistPattern       []string                   `json:"publicHoistPattern"`
	Skipped                  []string                   `json:"skipped"`
	StoreDir                 string                     `json:"storeDir"`
	VirtualStoreDir          string                     `json:"virtualStoreDir"`
	VirtualStoreDirMaxLength int                        `json:"virtualStoreDirMaxLength"`
	AllowBuilds              map[string]json.RawMessage `json:"allowBuilds"`
}

func parsePNPMLayoutModules(input PNPMInspectionInput, layout PNPMLayoutArchive, storeRoot string) (pnpmLayoutModules, error) {
	var modules pnpmLayoutModules
	if storeRoot == "/" || !strings.HasPrefix(storeRoot, "/") || path.Clean(storeRoot) != storeRoot || !validInventoryPath(strings.TrimPrefix(storeRoot, "/")) {
		return modules, errPNPMInspectionLayout
	}
	body, err := layout.ReadFile("node_modules/.modules.yaml")
	if err != nil || strictControlJSON(body) != nil {
		return modules, errPNPMInspectionLayout
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return modules, errPNPMInspectionLayout
	}
	for _, name := range []string{"hoistedDependencies", "hoistPattern", "included", "layoutVersion", "nodeLinker", "packageManager", "pendingBuilds", "prunedAt", "publicHoistPattern", "skipped", "storeDir", "virtualStoreDir", "virtualStoreDirMaxLength", "allowBuilds"} {
		value, present := fields[name]
		if !present || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return modules, errPNPMInspectionLayout
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&modules) != nil || modules.LayoutVersion != 5 || modules.NodeLinker != "isolated" ||
		modules.PackageManager != "pnpm@"+input.context.selection.lock.version || modules.VirtualStoreDir != ".pnpm" ||
		modules.VirtualStoreDirMaxLength != PNPMVirtualStoreDirMaxLength || modules.StoreDir != storeRoot+"/v11" ||
		!modules.Included.Dependencies || !modules.Included.DevDependencies || !modules.Included.OptionalDependencies ||
		!slices.Equal(modules.HoistPattern, []string{"*"}) || len(modules.PublicHoistPattern) != 0 || len(modules.Skipped) != 0 || len(modules.AllowBuilds) != 0 || len(modules.PrunedAt) > 128 {
		return pnpmLayoutModules{}, errPNPMInspectionLayout
	}
	if _, err := time.Parse(time.RFC1123, modules.PrunedAt); err != nil {
		return pnpmLayoutModules{}, errPNPMInspectionLayout
	}
	instances := make(map[string]PNPMLockedSnapshot, len(input.context.selection.lock.snapshots))
	for _, instance := range input.context.selection.lock.snapshots {
		instances[instance.Key] = instance
	}
	if len(modules.PendingBuilds) > len(instances)+1 || len(modules.HoistedDependencies) > len(instances) {
		return pnpmLayoutModules{}, errPNPMInspectionLayout
	}
	seen := map[string]bool{}
	for _, key := range modules.PendingBuilds {
		_, present := instances[key]
		if key != "." && !present || seen[key] {
			return pnpmLayoutModules{}, errPNPMInspectionLayout
		}
		seen[key] = true
	}
	packages := map[string]PNPMLockedPackage{}
	for _, pkg := range input.context.selection.lock.packages {
		packages[pkg.Key] = pkg
	}
	aliases := map[string]bool{}
	for key, names := range modules.HoistedDependencies {
		instance, present := instances[key]
		if !present || len(names) != 1 {
			return pnpmLayoutModules{}, errPNPMInspectionLayout
		}
		for name, kind := range names {
			if kind != "private" || name != packages[instance.Package].Name || aliases[name] {
				return pnpmLayoutModules{}, errPNPMInspectionLayout
			}
			aliases[name] = true
		}
	}
	return modules, nil
}

func checkPNPMVirtualLock(input PNPMInspectionInput, layout PNPMLayoutArchive) error {
	var manifest []byte
	for _, control := range input.context.controls {
		if control.Name() == "package.json" {
			manifest = control.Body()
		}
	}
	body, err := layout.ReadFile("node_modules/.pnpm/lock.yaml")
	if err != nil {
		return errPNPMInspectionLayout
	}
	lock, err := ParsePNPMLock(manifest, body, input.context.selection.lock.version)
	if err != nil {
		return errPNPMInspectionLayout
	}
	actual, err := json.Marshal(pnpmSnapshotLockState(lock))
	if err != nil {
		return errPNPMInspectionLayout
	}
	expected, err := json.Marshal(pnpmSnapshotLockState(input.context.selection.lock))
	if err != nil || !bytes.Equal(actual, expected) {
		return errPNPMInspectionLayout
	}
	return nil
}
