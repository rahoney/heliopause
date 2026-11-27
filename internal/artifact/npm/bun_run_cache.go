package npm

import (
	"archive/tar"
	"context"
	"errors"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

var errBunRunCache = errors.New("bun one-off cache is incomplete or differs from its request")

// BunRunCache retains unapproved native DATA. Cache names and native controls
// neither authenticate archives nor approve a program or installation ownership.
type BunRunCache struct {
	layout   BunLayoutArchive
	root     string
	controls []domain.ProjectControlFile
	lock     BunLock
}

func (c BunRunCache) Layout() BunLayoutArchive { return c.layout }
func (c BunRunCache) Root() string             { return c.root }
func (c BunRunCache) Selection() BunLock       { return c.lock }
func (c BunRunCache) Controls() []domain.ProjectControlFile {
	return append([]domain.ProjectControlFile(nil), c.controls...)
}

func ValidateBunRunReference(reference domain.ArtifactReference) error {
	if !validNativeNodeRunReference(reference) {
		return errBunRunCache
	}
	return nil
}

// ParseBunRunCache reads only the fresh fixed-user bunx subtree. Registry-cache
// aliases outside that subtree are not inputs. The existing bounded tar and Bun
// lock parsers keep their owner, link, schema and whole-instance checks.
func ParseBunRunCache(ctx context.Context, body []byte, reference domain.ArtifactReference, clientVersion string) (BunRunCache, error) {
	if ctx == nil || ValidateBunRunReference(reference) != nil {
		return BunRunCache{}, errBunRunCache
	}
	const base = ".bunx-1000"
	layout, err := parsePNPMTreeArchive(ctx, body, base)
	if err != nil {
		return BunRunCache{}, errors.Join(errBunRunCache, err)
	}
	root := ""
	for _, entry := range layout.entries {
		parts := strings.Split(entry.Name, "/")
		if len(parts) == 3 && parts[2] == "package.json" {
			if root != "" {
				return BunRunCache{}, errBunRunCache
			}
			root = base + "/" + parts[1]
		}
	}
	if root == "" {
		return BunRunCache{}, errBunRunCache
	}
	controls := map[string]domain.ProjectControlFile{}
	lockPresent := false
	for _, entry := range layout.entries {
		if entry.Name == base && (entry.Kind != tar.TypeDir || entry.Mode != 0o700) {
			return BunRunCache{}, errBunRunCache
		}
		if entry.Name == root+".lock" {
			if entry.Kind != tar.TypeReg || entry.Size != 0 || entry.Mode != 0o600 {
				return BunRunCache{}, errBunRunCache
			}
			lockPresent = true
			continue
		}
		if entry.Name != base && entry.Name != root && !strings.HasPrefix(entry.Name, root+"/") {
			return BunRunCache{}, errBunRunCache
		}
		name := strings.TrimPrefix(entry.Name, root+"/")
		if name != "package.json" && name != "bun.lock" {
			continue
		}
		if entry.Kind != tar.TypeReg || entry.Size <= 0 || entry.Size > ProjectControlLimit || entry.Mode != 0o600 && entry.Mode != 0o644 {
			return BunRunCache{}, errBunRunCache
		}
		data, err := layout.ReadFile(entry.Name)
		if err != nil {
			return BunRunCache{}, errors.Join(errBunRunCache, err)
		}
		control, err := domain.NewProjectControlFile(name, data, true)
		if err != nil {
			return BunRunCache{}, errors.Join(errBunRunCache, err)
		}
		controls[name] = control
	}
	if !lockPresent {
		return BunRunCache{}, errBunRunCache
	}
	lock, err := ParseBunLock(controls["package.json"].Body(), controls["bun.lock"], clientVersion)
	if err != nil {
		return BunRunCache{}, errors.Join(errBunRunCache, err)
	}
	if _, err := lock.runPrimary(reference); err != nil {
		return BunRunCache{}, err
	}
	if ctx.Err() != nil {
		return BunRunCache{}, errors.Join(errBunRunCache, ctx.Err(), context.Cause(ctx))
	}
	return BunRunCache{BunLayoutArchive{layout}, root, []domain.ProjectControlFile{controls["package.json"], controls["bun.lock"]}, lock}, nil
}

func (lock BunLock) runPrimary(reference domain.ArtifactReference) (string, error) {
	if !lock.valid || len(lock.roots) != 1 || ValidateBunRunReference(reference) != nil {
		return "", errBunRunCache
	}
	parsed, _ := parseLocator(reference.Locator())
	root := lock.roots[0]
	if root.Group != "dependencies" || root.Name != parsed.name {
		return "", errBunRunCache
	}
	for _, pkg := range lock.packages {
		if pkg.Key == root.Instance && pkg.Name == parsed.name && (!parsed.exact || pkg.Version == parsed.selector) {
			return pkg.Key, nil
		}
	}
	return "", errBunRunCache
}
