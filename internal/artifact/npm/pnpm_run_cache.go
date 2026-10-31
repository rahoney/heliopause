package npm

import (
	"archive/tar"
	"context"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errPNPMRunCache = errors.New("pnpm one-off cache is incomplete or differs from its request")

// PNPMRunCache is unapproved native DATA. Neither cache links nor native lock
// metadata authenticate archives, approve a program or establish installation ownership.
type PNPMRunCache struct {
	layout   PNPMLayoutArchive
	root     string
	controls []domain.ProjectControlFile
	lock     PNPMLock
}

func (c PNPMRunCache) Layout() PNPMLayoutArchive { return c.layout }
func (c PNPMRunCache) Root() string              { return c.root }
func (c PNPMRunCache) Selection() PNPMLock       { return c.lock }
func (c PNPMRunCache) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), c.controls...)
}

// ValidatePNPMRunReference excludes provisioning and non-package inputs before
// the native resolver touches tools, DNS or resources.
func ValidatePNPMRunReference(reference domain.ArtifactReference) error {
	if !validNativeNodeRunReference(reference) {
		return errPNPMRunCache
	}
	return nil
}

func validNativeNodeRunReference(reference domain.ArtifactReference) bool {
	parsed, err := parseLocator(reference.Locator())
	if err != nil || reference.Source().String() != "npm" || !parsed.exact && parsed.locator != parsed.name {
		return false
	}
	switch parsed.name {
	case "node", "deno", "bun", "pnpm", "npm", "yarn", "rust":
		return false
	}
	return true
}

// ParsePNPMRunCache reuses the bounded tar reader without following Host links.
func ParsePNPMRunCache(ctx context.Context, body []byte, reference domain.ArtifactReference, clientVersion string) (PNPMRunCache, error) {
	if ctx == nil || ValidatePNPMRunReference(reference) != nil {
		return PNPMRunCache{}, errPNPMRunCache
	}
	layout, err := parsePNPMTreeArchive(ctx, body, "dlx")
	if err != nil {
		return PNPMRunCache{}, err
	}
	key, root := "", ""
	for _, entry := range layout.entries {
		parts := strings.Split(entry.Name, "/")
		if len(parts) != 3 || parts[2] != "pkg" {
			continue
		}
		if root != "" || entry.Kind != tar.TypeSymlink || entry.Link == "pkg" || strings.Contains(entry.Link, "/") {
			return PNPMRunCache{}, errPNPMRunCache
		}
		key, root = parts[1], "dlx/"+parts[1]+"/"+entry.Link
	}
	if root == "" {
		return PNPMRunCache{}, errPNPMRunCache
	}
	controls := map[string]domain.ProjectControlFile{}
	for _, entry := range layout.entries {
		if entry.Name != "dlx" && entry.Name != "dlx/"+key && !strings.HasPrefix(entry.Name, "dlx/"+key+"/") {
			return PNPMRunCache{}, errPNPMRunCache
		}
		name := strings.TrimPrefix(entry.Name, root+"/")
		if name != "package.json" && name != "pnpm-lock.yaml" {
			continue
		}
		if entry.Kind != tar.TypeReg || entry.Size <= 0 || entry.Size > ProjectControlLimit || entry.Mode != 0o600 && entry.Mode != 0o644 {
			return PNPMRunCache{}, errPNPMRunCache
		}
		data, readErr := layout.ReadFile(entry.Name)
		if readErr != nil {
			return PNPMRunCache{}, readErr
		}
		control, controlErr := domain.NewProjectControlFile(name, data, true)
		if controlErr != nil {
			return PNPMRunCache{}, controlErr
		}
		controls[name] = control
	}
	lock, err := ParsePNPMLock(controls["package.json"].Body(), controls["pnpm-lock.yaml"].Body(), clientVersion)
	if err != nil {
		return PNPMRunCache{}, errPNPMRunCache
	}
	if _, err := lock.runPrimary(reference); err != nil {
		return PNPMRunCache{}, err
	}
	if ctx.Err() != nil {
		return PNPMRunCache{}, errors.Join(errPNPMRunCache, ctx.Err(), context.Cause(ctx))
	}
	return PNPMRunCache{layout, root, []domain.ProjectControlFile{controls["package.json"], controls["pnpm-lock.yaml"]}, lock}, nil
}

// Both native cache intake and later independent source binding use the same
// exact request/root check; neither consumer may infer a different primary.
func (lock PNPMLock) runPrimary(reference domain.ArtifactReference) (string, error) {
	if !lock.valid || len(lock.roots) != 1 || ValidatePNPMRunReference(reference) != nil {
		return "", errPNPMRunCache
	}
	parsed, _ := parseLocator(reference.Locator())
	requested := lock.roots[0]
	if requested.Group != "dependencies" || requested.Name != parsed.name {
		return "", errPNPMRunCache
	}
	for _, snapshot := range lock.snapshots {
		if snapshot.Key != requested.Snapshot {
			continue
		}
		for _, pkg := range lock.packages {
			if pkg.Key == snapshot.Package && pkg.Name == parsed.name && (!parsed.exact || pkg.Version == parsed.selector) {
				return requested.Snapshot, nil
			}
		}
	}
	return "", errPNPMRunCache
}
