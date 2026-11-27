package npm

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// RenderRegistryCache returns only a bounded, controller-owned data envelope.
// Native bookkeeping and the private seed graph never enter the public cache.
func (i BunInspectionInput) RenderRegistryCache(ctx context.Context, entries []BunNativeCacheEntry) ([]byte, error) {
	if ctx == nil || !i.Valid() || len(entries) != len(i.artifacts) || len(entries) == 0 {
		return nil, errBunNativeCache
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errBunNativeCache, err, context.Cause(ctx))
	}
	bySubject := map[domain.AcquiredArtifact]BunNativeCacheEntry{}
	var inventories []ArchiveInventory
	for _, entry := range entries {
		if !entry.matches(i) {
			return nil, errBunNativeCache
		}
		if _, duplicate := bySubject[entry.inventory.artifact]; duplicate {
			return nil, errBunNativeCache
		}
		bySubject[entry.inventory.artifact] = entry
		inventories = append(inventories, entry.inventory)
	}
	if err := i.CheckNativeCacheInputs(inventories); err != nil {
		return nil, err
	}
	type cacheFile struct {
		entry    BunNativeCacheEntry
		original ArchiveEntry
	}
	files := map[string]cacheFile{}
	directories := map[string]bool{"cache": true}
	for _, subject := range i.artifacts {
		entry, present := bySubject[subject]
		if !present {
			return nil, errBunNativeCache
		}
		folder, err := bunRegistryCacheFolder(subject)
		if err != nil {
			return nil, err
		}
		base := "cache/" + folder
		for _, file := range entry.inventory.entries {
			name := base + "/" + file.Path
			if !validInventoryPath(name) {
				return nil, errBunNativeCache
			}
			if file.Directory {
				directories[name] = true
			} else {
				if _, duplicate := files[name]; duplicate {
					return nil, errBunNativeCache
				}
				files[name] = cacheFile{entry, file}
			}
			for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
				directories[parent] = true
				if len(files)+len(directories) > pnpmLayoutEntryLimit {
					return nil, errBunNativeCache
				}
			}
			if len(files)+len(directories) > pnpmLayoutEntryLimit {
				return nil, errBunNativeCache
			}
		}
	}
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	emit := func(header tar.Header, body []byte) error {
		if err := ctx.Err(); err != nil {
			return errors.Join(errBunNativeCache, err, context.Cause(ctx))
		}
		if len(body) > BunInspectionWorkspaceLimit || output.Len() > BunInspectionWorkspaceLimit-len(body) {
			return errBunNativeCache
		}
		if err := writer.WriteHeader(&header); err != nil {
			return err
		}
		if _, err := writer.Write(body); err != nil {
			return err
		}
		if output.Len() > BunInspectionWorkspaceLimit {
			return errBunNativeCache
		}
		return nil
	}
	names := make([]string, 0, len(directories))
	for name := range directories {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if _, collision := files[name]; collision {
			return nil, errBunNativeCache
		}
		mode := int64(0o755)
		if name == "cache" {
			mode = 0o700
		}
		if err := emit(tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: mode, Uid: 1000, Gid: 1000}, nil); err != nil {
			return nil, err
		}
	}
	names = names[:0]
	for name := range files {
		names = append(names, name)
	}
	slices.SortFunc(names, strings.Compare)
	for _, name := range names {
		file := files[name]
		body, err := file.entry.native.ReadFile(file.entry.root + "/" + file.original.Path)
		if err != nil {
			return nil, err
		}
		if err := emit(tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: file.original.Mode &^ 0o022, Uid: 1000, Gid: 1000, Size: file.original.Size}, body); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	if output.Len() > BunInspectionWorkspaceLimit {
		return nil, errBunNativeCache
	}
	return output.Bytes(), nil
}
